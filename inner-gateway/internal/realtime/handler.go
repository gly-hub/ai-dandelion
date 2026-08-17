package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/gly-hub/ai-dandelion/toolbox/eventbus"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClientManager interface {
	GetConn(context.Context, string) (*grpc.ClientConn, error)
}
type Writer func(Envelope) error
type CommandHandler func(context.Context, Envelope, Writer)

type agentRun struct {
	userID string
	cancel context.CancelFunc
}

type historyEvent struct {
	envelope Envelope
	target   string
}

type Handler struct {
	clientMgr ClientManager
	tickets   *TicketManager
	runsMu    sync.Mutex
	runs      map[string]agentRun
	routesMu  sync.RWMutex
	routes    map[string]CommandHandler
	clientsMu sync.RWMutex
	clients   map[string]map[string]Writer
	historyMu sync.Mutex
	history   []historyEvent
	seqMu     sync.Mutex
	seq       uint64
	originsMu sync.RWMutex
	origins   map[string]struct{}
}

func NewHandler(clientMgr ClientManager, tickets *TicketManager) *Handler {
	return &Handler{clientMgr: clientMgr, tickets: tickets, runs: make(map[string]agentRun), routes: make(map[string]CommandHandler), clients: make(map[string]map[string]Writer), origins: map[string]struct{}{"http://localhost:5173": {}, "http://127.0.0.1:5173": {}, "http://localhost:5174": {}, "http://127.0.0.1:5174": {}}}
}

func (h *Handler) SetAllowedOrigins(origins []string) {
	values := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(strings.TrimRight(origin, "/"))
		if origin != "" {
			values[origin] = struct{}{}
		}
	}
	if len(values) == 0 {
		return
	}
	h.originsMu.Lock()
	h.origins = values
	h.originsMu.Unlock()
}

// SubscribeEvents connects a transport-neutral event bus to local WebSocket
// connections. Events target a user through headers["userId"], or all users
// when that header is omitted.
func (h *Handler) SubscribeEvents(ctx context.Context, bus eventbus.Bus, subscription eventbus.Subscription) {
	if bus == nil || subscription.Handler != nil {
		return
	}
	subscription.Handler = func(eventCtx context.Context, event eventbus.Event) error {
		return h.publishEvent(event)
	}
	go func() { _ = bus.Subscribe(ctx, subscription) }()
}

func (h *Handler) publishEvent(event eventbus.Event) error {
	target := ""
	if event.Headers != nil {
		target = strings.TrimSpace(event.Headers["userId"])
	}
	envelope := Envelope{ProtocolVersion: 1, Type: event.Type, EventID: event.ID, Timestamp: event.OccurredAt, Payload: event.Payload}
	if event.CorrelationID != "" {
		envelope.RequestID = event.CorrelationID
	}
	h.historyMu.Lock()
	h.history = append(h.history, historyEvent{envelope: envelope, target: target})
	if len(h.history) > 256 {
		h.history = h.history[len(h.history)-256:]
	}
	h.historyMu.Unlock()
	h.clientsMu.RLock()
	var writers []Writer
	if target != "" {
		for _, writer := range h.clients[target] {
			writers = append(writers, writer)
		}
	} else {
		for _, userClients := range h.clients {
			for _, writer := range userClients {
				writers = append(writers, writer)
			}
		}
	}
	h.clientsMu.RUnlock()
	for _, writer := range writers {
		// A connection may close between the snapshot and the write. The event
		// remains durable in the stream; stale local delivery must not prevent ack.
		_ = writer(envelope)
	}
	return nil
}

// RegisterNamespace lets business modules add commands without coupling to the
// connection lifecycle. The namespace is matched by the prefix before the dot.
func (h *Handler) RegisterNamespace(namespace string, handler CommandHandler) {
	if namespace == "" || handler == nil {
		return
	}
	h.routesMu.Lock()
	h.routes[namespace] = handler
	h.routesMu.Unlock()
}
func (h *Handler) UpgradeCheck(c *fiber.Ctx) error {
	if !fiberws.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	origin := c.Get(fiber.HeaderOrigin)
	if origin != "" {
		parsed, err := url.Parse(origin)
		h.originsMu.RLock()
		_, configured := h.origins[strings.TrimRight(origin, "/")]
		h.originsMu.RUnlock()
		if err != nil || parsed.Host == "" || (!configured && !allowedOrigin(parsed, c.Get("Host"), c.Hostname())) {
			return fiber.ErrForbidden
		}
	}
	return c.Next()
}

func allowedOrigin(origin *url.URL, requestHost, _ string) bool {
	if origin == nil || origin.Host == "" {
		return false
	}
	if origin.Host == requestHost {
		return true
	}
	return origin.Host == "localhost:5173" || origin.Host == "127.0.0.1:5173" ||
		origin.Host == "localhost:5174" || origin.Host == "127.0.0.1:5174"
}

func (h *Handler) Serve(c *fiberws.Conn) {
	user, err := h.tickets.Consume(c.Query("ticket"))
	if err != nil {
		_ = c.WriteJSON(errorEnvelope("", "invalid token"))
		_ = c.Close()
		return
	}
	ctx, cancel := context.WithCancel(authctx.ContextWithUser(context.Background(), user))
	defer cancel()
	c.SetReadLimit(1 << 20)
	_ = c.SetReadDeadline(time.Now().Add(70 * time.Second))
	c.SetPongHandler(func(string) error { return c.SetReadDeadline(time.Now().Add(70 * time.Second)) })
	var writeMu sync.Mutex
	write := func(event Envelope) error { writeMu.Lock(); defer writeMu.Unlock(); return c.WriteJSON(event) }
	connectionID := uuid.NewString()
	h.clientsMu.RLock()
	userConnections := len(h.clients[user.ID])
	globalConnections := 0
	for _, clients := range h.clients {
		globalConnections += len(clients)
	}
	h.clientsMu.RUnlock()
	if userConnections >= 8 || globalConnections >= 2000 {
		_ = write(errorEnvelope("", "realtime connection limit exceeded"))
		_ = c.Close()
		return
	}
	h.registerClient(user.ID, connectionID, write)
	defer h.unregisterClient(user.ID, connectionID)
	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				writeMu.Lock()
				_ = c.WriteMessage(fiberws.PingMessage, nil)
				writeMu.Unlock()
			case <-stopHeartbeat:
				return
			}
		}
	}()
	ready, _ := json.Marshal(map[string]string{"userId": user.ID, "namespaces": "ai-agent,system,func-operation"})
	_ = write(Envelope{ProtocolVersion: 1, Type: "connection.ready", Timestamp: time.Now().UnixMilli(), Payload: ready})
	if cursor := strings.TrimSpace(c.Query("lastEventId")); cursor != "" {
		h.historyMu.Lock()
		seen := false
		for _, event := range h.history {
			if event.envelope.EventID == cursor {
				seen = true
				continue
			}
			if (seen || cursor == "") && (event.target == "" || event.target == user.ID) {
				_ = write(event.envelope)
			}
		}
		h.historyMu.Unlock()
	}

	windowStart, commandCount := time.Now(), 0
	for {
		var in Envelope
		if err := c.ReadJSON(&in); err != nil {
			return
		}
		if in.ProtocolVersion == 0 {
			in.ProtocolVersion = 1
		}
		if err := in.ValidateCommand(); err != nil {
			_ = write(errorEnvelope(in.RequestID, err.Error()))
			continue
		}
		if time.Since(windowStart) >= time.Minute {
			windowStart, commandCount = time.Now(), 0
		}
		commandCount++
		if commandCount > 120 {
			_ = write(errorEnvelope(in.RequestID, "realtime command rate limit exceeded"))
			continue
		}
		if namespace, _, ok := strings.Cut(in.Type, "."); ok {
			h.routesMu.RLock()
			route := h.routes[namespace]
			h.routesMu.RUnlock()
			if route != nil {
				route(ctx, in, write)
				continue
			}
		}
		switch in.Type {
		case "ping":
			_ = write(Envelope{ProtocolVersion: 1, Type: "pong", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli()})
		case "ai-agent.stream.start":
			h.startAgentStream(ctx, in, write)
		case "ai-agent.ask-user.answer":
			h.answerAskUser(ctx, in, write)
		case "ai-agent.tool-permission.answer":
			h.answerPermission(ctx, in, write)
		case "ai-agent.stream.cancel":
			h.cancelRun(ctx, in.RequestID, write)
		default:
			_ = write(errorEnvelope(in.RequestID, "unsupported realtime command: "+in.Type))
		}
	}
}

func (h *Handler) aiClient(ctx context.Context) (aiagent.AiAgentServiceClient, error) {
	conn, err := h.clientMgr.GetConn(ctx, "ai-agent")
	if err != nil {
		return nil, err
	}
	return aiagent.NewAiAgentServiceClient(conn), nil
}
func (h *Handler) startAgentStream(ctx context.Context, in Envelope, write Writer) {
	if strings.TrimSpace(in.RequestID) == "" {
		_ = write(errorEnvelope("", "requestId is required"))
		return
	}
	var p AgentStreamPayload
	if err := json.Unmarshal(in.Payload, &p); err != nil || strings.TrimSpace(p.SessionID) == "" {
		_ = write(errorEnvelope(in.RequestID, "sessionId and valid payload are required"))
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	h.runsMu.Lock()
	if _, exists := h.runs[in.RequestID]; exists {
		h.runsMu.Unlock()
		cancel()
		_ = write(errorEnvelope(in.RequestID, "requestId is already running"))
		return
	}
	user, _ := authctx.CurrentUser(ctx)
	h.runs[in.RequestID] = agentRun{userID: user.ID, cancel: cancel}
	h.runsMu.Unlock()
	client, err := h.aiClient(runCtx)
	if err != nil {
		h.removeRun(in.RequestID)
		cancel()
		_ = write(errorEnvelope(in.RequestID, err.Error()))
		return
	}
	stream, err := client.StreamMessage(runCtx, &aiagent.StreamMessageReq{
		SessionId: p.SessionID, Content: p.Content, ModelId: p.ModelID,
		AgentSessionConfigType: p.AgentSessionConfigType, SystemPrompt: p.SystemPrompt,
		PermissionMode: p.PermissionMode, MaxTurns: p.MaxTurns, Extra: p.Extra,
		UserId: p.UserID, MessageParts: p.MessageParts,
	})
	if err != nil {
		h.removeRun(in.RequestID)
		cancel()
		_ = write(errorEnvelope(in.RequestID, err.Error()))
		return
	}
	_ = write(Envelope{ProtocolVersion: 1, Type: "ai-agent.stream.accepted", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli()})
	go func() {
		defer func() { h.removeRun(in.RequestID); cancel() }()
		for {
			item, recvErr := stream.Recv()
			if recvErr != nil {
				if errors.Is(recvErr, context.Canceled) || status.Code(recvErr) == codes.Canceled {
					_ = write(Envelope{ProtocolVersion: 1, Type: "ai-agent.stream.canceled", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli()})
					return
				}
				_ = write(errorEnvelope(in.RequestID, recvErr.Error()))
				return
			}
			payload, _ := json.Marshal(item)
			eventType := "ai-agent.stream.delta"
			if item.GetDone() {
				eventType = "ai-agent.stream.done"
			} else if item.GetType() != "" {
				eventType = "ai-agent.stream." + item.GetType()
			}
			h.seqMu.Lock()
			h.seq++
			sequence := h.seq
			h.seqMu.Unlock()
			if err := write(Envelope{ProtocolVersion: 1, Type: eventType, RequestID: in.RequestID, EventID: fmt.Sprintf("%s:%d", in.RequestID, sequence), Timestamp: time.Now().UnixMilli(), Payload: payload}); err != nil {
				return
			}
			if item.GetDone() {
				return
			}
		}
	}()
}

func (h *Handler) removeRun(requestID string) {
	h.runsMu.Lock()
	delete(h.runs, requestID)
	h.runsMu.Unlock()
}

func (h *Handler) registerClient(userID, connectionID string, writer Writer) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[string]Writer)
	}
	h.clients[userID][connectionID] = writer
}

func (h *Handler) unregisterClient(userID, connectionID string) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	if clients := h.clients[userID]; clients != nil {
		delete(clients, connectionID)
		if len(clients) == 0 {
			delete(h.clients, userID)
		}
	}
}

func (h *Handler) cancelRun(ctx context.Context, requestID string, write Writer) {
	h.runsMu.Lock()
	run, ok := h.runs[requestID]
	h.runsMu.Unlock()
	user, _ := authctx.CurrentUser(ctx)
	if !ok || run.userID != user.ID {
		_ = write(errorEnvelope(requestID, "run not found"))
		return
	}
	run.cancel()
	_ = write(Envelope{ProtocolVersion: 1, Type: "ai-agent.stream.cancel.ack", RequestID: requestID, Timestamp: time.Now().UnixMilli()})
}
func (h *Handler) answerAskUser(ctx context.Context, in Envelope, write func(Envelope) error) {
	var p AskUserPayload
	if json.Unmarshal(in.Payload, &p) != nil {
		_ = write(errorEnvelope(in.RequestID, "invalid payload"))
		return
	}
	client, err := h.aiClient(ctx)
	if err == nil {
		_, err = client.SubmitAskUserQuestion(ctx, &aiagent.SubmitAskUserQuestionReq{SessionId: p.SessionID, ToolId: p.ToolID, AnswersJson: p.AnswersJSON, Response: p.Response})
	}
	if err != nil {
		_ = write(errorEnvelope(in.RequestID, err.Error()))
		return
	}
	_ = write(Envelope{ProtocolVersion: 1, Type: "ai-agent.ask-user.ack", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli()})
}
func (h *Handler) answerPermission(ctx context.Context, in Envelope, write func(Envelope) error) {
	var p ToolPermissionPayload
	if json.Unmarshal(in.Payload, &p) != nil {
		_ = write(errorEnvelope(in.RequestID, "invalid payload"))
		return
	}
	client, err := h.aiClient(ctx)
	if err == nil {
		_, err = client.SubmitToolPermission(ctx, &aiagent.SubmitToolPermissionReq{SessionId: p.SessionID, ToolId: p.ToolID, Allow: p.Allow, Message: p.Message})
	}
	if err != nil {
		_ = write(errorEnvelope(in.RequestID, err.Error()))
		return
	}
	_ = write(Envelope{ProtocolVersion: 1, Type: "ai-agent.tool-permission.ack", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli()})
}
func errorEnvelope(requestID, message string) Envelope {
	payload, _ := json.Marshal(map[string]string{"message": message})
	return Envelope{ProtocolVersion: 1, Type: "error", RequestID: requestID, Timestamp: time.Now().UnixMilli(), Payload: payload}
}

// RealtimeError lets namespace adapters return the same protocol error shape.
func RealtimeError(requestID string, err error) Envelope {
	if err == nil {
		err = errors.New("realtime command failed")
	}
	return errorEnvelope(requestID, err.Error())
}
