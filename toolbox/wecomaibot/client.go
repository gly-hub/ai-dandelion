package wecomaibot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

const (
	DefaultWSURL             = "wss://openws.work.weixin.qq.com"
	DefaultHeartbeatInterval = 30 * time.Second
	CmdSubscribe             = "aibot_subscribe"
	CmdHeartbeat             = "ping"
	CmdResponse              = "aibot_respond_msg"
	CmdResponseWelcome       = "aibot_respond_welcome_msg"
	CmdCallback              = "aibot_msg_callback"
	CmdEventCallback         = "aibot_event_callback"
)

var mentionPrefixPattern = regexp.MustCompile(`^@\S+\s*`)

type Options struct {
	BotID                string
	Secret               string
	WSURL                string
	HeartbeatInterval    time.Duration
	MaxReconnectAttempts int
}

type Event struct {
	Type  string
	Frame map[string]any
}

type Client struct {
	options Options
	handler Handler

	mu     sync.Mutex
	conn   *websocket.Conn
	cancel context.CancelFunc
}

type Handler interface {
	HandleEvent(ctx context.Context, client *Client, event Event)
	HandleError(ctx context.Context, err error)
}

type HandlerFunc func(ctx context.Context, client *Client, event Event)

func (f HandlerFunc) HandleEvent(ctx context.Context, client *Client, event Event) {
	f(ctx, client, event)
}

func (f HandlerFunc) HandleError(context.Context, error) {}

func NewClient(options Options, handler Handler) *Client {
	options.BotID = strings.TrimSpace(options.BotID)
	options.Secret = strings.TrimSpace(options.Secret)
	options.WSURL = strings.TrimSpace(options.WSURL)
	if options.WSURL == "" {
		options.WSURL = DefaultWSURL
	}
	if options.HeartbeatInterval <= 0 {
		options.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if options.MaxReconnectAttempts == 0 {
		options.MaxReconnectAttempts = -1
	}
	return &Client{options: options, handler: handler}
}

func (c *Client) Start(ctx context.Context) error {
	if c.options.BotID == "" {
		return errors.New("wecom bot id is required")
	}
	if c.options.Secret == "" {
		return errors.New("wecom bot secret is required")
	}
	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	if c.cancel != nil {
		c.mu.Unlock()
		cancel()
		return errors.New("wecom client already started")
	}
	c.cancel = cancel
	c.mu.Unlock()

	go c.run(runCtx)
	return nil
}

func (c *Client) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	conn := c.conn
	c.cancel = nil
	c.conn = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if conn != nil {
		_ = conn.Close()
	}
}

func (c *Client) SendJSON(payload any) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errors.New("wecom websocket is not connected")
	}
	return websocket.JSON.Send(conn, payload)
}

func (c *Client) ReplyWelcome(frame map[string]any, content string) error {
	return c.reply(frame, CmdResponseWelcome, map[string]any{
		"msgtype": "text",
		"text": map[string]any{
			"content": content,
		},
	})
}

func (c *Client) ReplyStream(frame map[string]any, streamID string, content string, finish bool) error {
	return c.reply(frame, CmdResponse, map[string]any{
		"msgtype": "stream",
		"stream": map[string]any{
			"id":      streamID,
			"finish":  finish,
			"content": content,
		},
	})
}

func (c *Client) reply(frame map[string]any, cmd string, body map[string]any) error {
	reqID := HeaderReqID(frame)
	if reqID == "" {
		return errors.New("wecom callback req_id is required")
	}
	return c.SendJSON(map[string]any{
		"cmd": cmd,
		"headers": map[string]any{
			"req_id": reqID,
		},
		"body": body,
	})
}

func (c *Client) run(ctx context.Context) {
	attempt := 0
	for {
		if ctx.Err() != nil {
			return
		}
		if c.options.MaxReconnectAttempts > 0 && attempt >= c.options.MaxReconnectAttempts {
			c.handleError(ctx, fmt.Errorf("wecom reconnect attempts exhausted: %d", attempt))
			return
		}
		attempt++
		if err := c.connectAndServe(ctx); err != nil && ctx.Err() == nil {
			c.handleError(ctx, err)
			time.Sleep(reconnectDelay(attempt))
			continue
		}
		return
	}
}

func (c *Client) connectAndServe(ctx context.Context) error {
	config, err := websocket.NewConfig(c.options.WSURL, "http://localhost/")
	if err != nil {
		return err
	}

	conn, err := websocket.DialConfig(config)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		_ = conn.Close()
	}()

	done := make(chan error, 1)
	go func() {
		for {
			var frame map[string]any
			if err := websocket.JSON.Receive(conn, &frame); err != nil {
				done <- err
				return
			}
			c.handleFrame(ctx, frame)
		}
	}()

	if err := c.sendAuth(conn); err != nil {
		return err
	}

	ticker := time.NewTicker(c.options.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		case <-ticker.C:
			if err := websocket.JSON.Send(conn, map[string]any{
				"cmd": CmdHeartbeat,
				"headers": map[string]any{
					"req_id": ReqID(CmdHeartbeat),
				},
			}); err != nil {
				return err
			}
		}
	}
}

func (c *Client) sendAuth(conn *websocket.Conn) error {
	return websocket.JSON.Send(conn, map[string]any{
		"cmd": CmdSubscribe,
		"headers": map[string]any{
			"req_id": ReqID(CmdSubscribe),
		},
		"body": map[string]any{
			"bot_id": c.options.BotID,
			"secret": c.options.Secret,
		},
	})
}

func (c *Client) handleFrame(ctx context.Context, frame map[string]any) {
	cmd := stringField(frame, "cmd")
	if cmd != CmdCallback && cmd != CmdEventCallback {
		return
	}
	if c.handler != nil {
		go c.handler.HandleEvent(ctx, c, Event{Type: EventType(frame), Frame: frame})
	}
}

func (c *Client) handleError(ctx context.Context, err error) {
	if c.handler != nil && err != nil && !errors.Is(err, context.Canceled) {
		c.handler.HandleError(ctx, err)
	}
}

func reconnectDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return time.Second
	}
	delay := time.Duration(attempt) * time.Second
	if delay > 15*time.Second {
		return 15 * time.Second
	}
	return delay
}

func ReqID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "req"
	}
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixMilli(), randomHex(8))
}

func EventType(frame map[string]any) string {
	cmd := stringField(frame, "cmd")
	body := recordField(frame, "body")
	if cmd == CmdEventCallback {
		event := recordField(body, "event")
		if eventType := stringField(event, "eventtype"); eventType != "" {
			return "event." + eventType
		}
		return "event"
	}
	if cmd == CmdCallback {
		if msgType := stringField(body, "msgtype"); msgType != "" {
			return "message." + msgType
		}
		return "message"
	}
	return ""
}

func TextContent(frame map[string]any) string {
	body := recordField(frame, "body")
	for _, source := range []map[string]any{recordField(body, "text"), recordField(frame, "text")} {
		if content := stringField(source, "content"); content != "" {
			return cleanTextContent(content)
		}
	}
	return cleanTextContent(stringField(frame, "content"))
}

func SenderID(frame map[string]any) string {
	body := recordField(frame, "body")
	if value := stringField(recordField(body, "from"), "userid"); value != "" {
		return value
	}
	for _, key := range []string{"from_userid", "userid", "sender", "user_id"} {
		if value := stringField(body, key); value != "" {
			return value
		}
		if value := stringField(frame, key); value != "" {
			return value
		}
	}
	return "wecom-user"
}

func ConversationKey(frame map[string]any) string {
	body := recordField(frame, "body")
	if stringField(body, "chattype") == "group" {
		if value := stringField(body, "chatid"); value != "" {
			return "group:" + value
		}
	}
	if userID := stringField(recordField(body, "from"), "userid"); userID != "" {
		return "single:" + userID
	}
	for _, key := range []string{"chatid", "conversation_id", "chat_id", "roomid"} {
		if value := stringField(body, key); value != "" {
			return value
		}
		if value := stringField(frame, key); value != "" {
			return value
		}
	}
	return SenderID(frame)
}

func cleanTextContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	stripped := strings.TrimSpace(mentionPrefixPattern.ReplaceAllString(content, ""))
	if stripped != "" {
		return stripped
	}
	return content
}

func HeaderReqID(frame map[string]any) string {
	return stringField(recordField(frame, "headers"), "req_id")
}

func ConfigInt(config map[string]any, key string, fallback int) int {
	value, ok := config[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(typed, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func DecodeConfig(value string) map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func recordField(data map[string]any, key string) map[string]any {
	value, ok := data[key]
	if !ok || value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func stringField(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func randomHex(length int) string {
	const alphabet = "0123456789abcdef"
	if length <= 0 {
		return ""
	}
	now := time.Now().UnixNano()
	out := make([]byte, length)
	for i := range out {
		shift := uint((i % 8) * 4)
		out[i] = alphabet[(now>>shift)&0x0f]
	}
	return string(out)
}
