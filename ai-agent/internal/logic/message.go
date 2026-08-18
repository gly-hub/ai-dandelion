package logic

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/dao"
	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/ai-dandelion/toolbox/agent"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type runnerFactory interface {
	DefaultRunner() agent.Runner
	RunnerFor(ctx context.Context, record *model.AgentModel) agent.Runner
	RunnerForConfig(ctx context.Context, record *model.AgentModel, override AgentRuntimeOverride) agent.Runner
}

type MessageLogic struct {
	sessionDao            *dao.Session
	messageDao            *dao.Message
	sessionReferenceDao   *dao.SessionReference
	runnerFactory         runnerFactory
	agentModelLogic       *AgentModelLogic
	agentEngine           *AgentEngine
	agentSessionConfigDao *dao.AgentSessionConfig
	skillLogic            *SkillLogic
	mcpLogic              *MCPLogic
	functionSkillRuntime  *FunctionSkillRuntime
	askUserQuestionBroker *AskUserQuestionBroker
	toolPermissionBroker  *ToolPermissionBroker
}

var (
	errAgentRunnerNotConfigured = errors.New("agent runner is not configured")
	errContentRequired          = errors.New("content is required")
)

var defaultFunctionSessionSkills = map[string][]string{
	"func_product":    {"product-doc-builder"},
	"func_technical":  {"technical-doc-builder", "generated-app-builder"},
	"func_generation": {"generated-app-builder"},
}

func NewMessageLogic(
	sessionDao *dao.Session,
	messageDao *dao.Message,
	sessionReferenceDao *dao.SessionReference,
	runnerFactory runnerFactory,
	agentModelLogic *AgentModelLogic,
	agentSessionConfigDao *dao.AgentSessionConfig,
	skillLogic *SkillLogic,
	mcpLogic *MCPLogic,
	functionSkillRuntimes ...*FunctionSkillRuntime,
) *MessageLogic {
	var functionSkillRuntime *FunctionSkillRuntime
	if len(functionSkillRuntimes) > 0 {
		functionSkillRuntime = functionSkillRuntimes[0]
	}
	return &MessageLogic{
		sessionDao:            sessionDao,
		messageDao:            messageDao,
		sessionReferenceDao:   sessionReferenceDao,
		runnerFactory:         runnerFactory,
		agentModelLogic:       agentModelLogic,
		agentEngine:           NewAgentEngine(runnerFactory, agentModelLogic),
		agentSessionConfigDao: agentSessionConfigDao,
		skillLogic:            skillLogic,
		mcpLogic:              mcpLogic,
		functionSkillRuntime:  functionSkillRuntime,
		askUserQuestionBroker: NewAskUserQuestionBroker(),
		toolPermissionBroker:  NewToolPermissionBroker(),
	}
}

func (m *MessageLogic) ListMessages(ctx context.Context, req *aiagent.GetMessageReq) (
	[]*aiagent.Message, bool, string, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, false, "", err
	}
	sessionID, err := requireSessionID(req.GetSessionId())
	if err != nil {
		return nil, false, "", err
	}
	if err := m.sessionDao.Exists(ctx, userID, sessionID); err != nil {
		return nil, false, "", err
	}

	page, err := m.messageDao.List(ctx, sessionID, dao.MessagePageOptions{
		Limit:  int(req.GetLimit()),
		Before: strings.TrimSpace(req.GetBefore()),
	})
	if err != nil {
		return nil, false, "", err
	}

	messages := make([]*aiagent.Message, 0, len(page.Items))
	for i := range page.Items {
		messages = append(messages, modelMessageToProto(&page.Items[i]))
	}
	return messages, page.HasMore, page.NextBefore, nil
}

func (m *MessageLogic) StreamMessage(
	ctx context.Context,
	req *aiagent.StreamMessageReq,
	send func(*aiagent.StreamMessageResp) error,
) error {
	if m.agentEngine == nil {
		return errAgentRunnerNotConfigured
	}

	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return err
	}
	sessionID, err := requireSessionID(req.GetSessionId())
	if err != nil {
		return err
	}
	content := strings.TrimSpace(req.GetContent())
	userParts := userMessageParts(req)
	if content == "" && len(userParts) == 0 {
		return errContentRequired
	}
	prompt := promptFromParts(userParts)
	if strings.TrimSpace(prompt) == "" {
		prompt = content
	}
	session, err := m.sessionDao.Get(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	userContent, err := m.buildAgentUserContent(ctx, userParts, prompt)
	if err != nil {
		return err
	}
	if _, err := m.addMessage(ctx, sessionID, model.RoleUser, content, userParts); err != nil {
		return err
	}
	sessionRefs, err := m.recordAndListSessionReferences(ctx, sessionID, userParts, req.GetExtra())
	if err != nil {
		return err
	}

	agentSessionID := strings.TrimSpace(session.AgentSessionId)
	resume := agentSessionID != ""
	if agentSessionID == "" {
		agentSessionID = uuid.New().String()
	}

	engineConfig, err := m.resolveStreamEngineConfig(ctx, sessionID, req, userParts, sessionRefs)
	if err != nil {
		return err
	}
	if engineConfig.Cleanup != nil {
		defer engineConfig.Cleanup()
	}
	engineConfig.UserContent = userContent
	events, errs, err := m.agentEngine.Stream(ctx, agentSessionID, prompt, resume, engineConfig)
	if err != nil {
		return err
	}
	var answer strings.Builder
	parts := newResponseParts()
	partialSaved := false
	savePartial := func() {
		if partialSaved || (answer.Len() == 0 && len(parts.parts()) == 0) {
			return
		}
		// The request context is canceled by the gateway. Persisting the output
		// uses a short independent context so refresh/reconnect can recover it.
		persistCtx, cancelPersist := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelPersist()
		message, saveErr := m.addMessage(persistCtx, sessionID, model.RoleAssistant, answer.String(), parts.parts())
		if saveErr == nil {
			if updateErr := m.sessionDao.UpdateAgentSession(persistCtx, userID, sessionID, agentSessionID, message.CreatedAt); updateErr == nil {
				partialSaved = true
			}
		}
	}

	for events != nil || errs != nil {
		select {
		case event, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if event.Type == "text_delta" && event.Text != "" {
				answer.WriteString(event.Text)
			}
			parts.apply(event)
			if event.Done {
				message, err := m.addMessage(ctx, sessionID, model.RoleAssistant, answer.String(), parts.parts())
				if err != nil {
					return err
				}
				if event.AgentSessionID != "" {
					agentSessionID = event.AgentSessionID
				}
				if err := m.sessionDao.UpdateAgentSession(ctx, userID, sessionID, agentSessionID, message.CreatedAt); err != nil {
					return err
				}
				return send(streamRespFromEvent(agent.Event{Type: "done", AgentSessionID: agentSessionID, Done: true}, modelMessageToProto(message)))
			}
			if event.Type != "" {
				if err := send(streamRespFromEvent(event, nil)); err != nil {
					return err
				}
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
				savePartial()
				return context.Canceled
			}
			return err
		case <-ctx.Done():
			savePartial()
			return ctx.Err()
		}
	}
	return nil
}

func (m *MessageLogic) skillDirs(ctx context.Context, userID string, skillIDs []string) ([]string, error) {
	if m.skillLogic == nil {
		return nil, nil
	}
	resolvedUserID := userIDForSkills(ctx, userID)
	if strings.TrimSpace(resolvedUserID) == "" {
		return nil, nil
	}
	return m.skillLogic.ResolveSkillDirs(resolvedUserID, skillIDs)
}

func userIDForSkills(ctx context.Context, fallback string) string {
	if userID, err := authctx.RequireUserID(ctx); err == nil {
		return userID
	}
	return strings.TrimSpace(fallback)
}

func (m *MessageLogic) resolveStreamEngineConfig(
	ctx context.Context,
	sessionID string,
	req *aiagent.StreamMessageReq,
	userParts []*aiagent.MessagePart,
	sessionRefs []model.SessionReference,
) (AgentEngineRunConfig, error) {
	modelID, override, err := m.resolveRuntimeConfig(ctx, req)
	if err != nil {
		return AgentEngineRunConfig{}, err
	}
	sessionConfigType := strings.TrimSpace(req.GetAgentSessionConfigType())
	skillIDs := extractSessionReferenceIDs(sessionRefs, model.SessionReferenceTypeSkill)
	if len(skillIDs) == 0 {
		skillIDs = extractSkillIDsFromParts(userParts, req.GetExtra())
	}
	skillIDs = uniqueNormalizedSkillIDs(append(skillIDs, defaultFunctionSessionSkills[sessionConfigType]...))
	addDirs, err := m.skillDirs(ctx, req.GetUserId(), skillIDs)
	if err != nil {
		return AgentEngineRunConfig{}, err
	}
	mcpServers, err := m.resolveMCPServers(ctx, req, extractSessionReferenceIDs(sessionRefs, model.SessionReferenceTypeMCP))
	if err != nil {
		return AgentEngineRunConfig{}, err
	}
	engineConfig := AgentEngineRunConfig{
		ModelID:         modelID,
		SystemPrompt:    override.SystemPrompt,
		PermissionMode:  override.PermissionMode,
		MaxTurns:        override.MaxTurns,
		Skills:          skillIDs,
		AddDirs:         addDirs,
		MCPServers:      mcpServers,
		AskUserQuestion: m.askUserQuestionHandler(sessionID),
		ToolPermission:  m.toolPermissionHandler(sessionID),
	}
	functionSkillIDs := extractSessionReferenceIDs(sessionRefs, model.SessionReferenceTypeFunctionSkill)
	if len(functionSkillIDs) == 0 {
		functionSkillIDs = extractFunctionSkillIDsFromParts(userParts, req.GetExtra())
	}
	if len(functionSkillIDs) == 0 || m.functionSkillRuntime == nil {
		return engineConfig, nil
	}
	setup, err := m.functionSkillRuntime.Prepare(ctx, userIDForSkills(ctx, req.GetUserId()), sessionID, functionSkillIDs)
	if err != nil {
		return AgentEngineRunConfig{}, err
	}
	engineConfig.Skills = append(engineConfig.Skills, setup.SkillNames...)
	engineConfig.AddDirs = append(engineConfig.AddDirs, setup.AddDirs...)
	if engineConfig.SDKMCPServers == nil {
		engineConfig.SDKMCPServers = make(map[string]claudeagentsdk.MCPServerConfig, len(setup.SDKMCPServers))
	}
	for id, server := range setup.SDKMCPServers {
		engineConfig.SDKMCPServers[id] = server
	}
	engineConfig.Cleanup = setup.Cleanup
	basePermission := engineConfig.ToolPermission
	engineConfig.ToolPermission = m.functionSkillToolPermissionHandler(sessionID, setup, basePermission)
	engineConfig.ForceToolPermission = setup.IsFunctionTool
	return engineConfig, nil
}

func (m *MessageLogic) functionSkillToolPermissionHandler(_ string, setup *FunctionSkillSetup, fallback agent.ToolPermissionHandler) agent.ToolPermissionHandler {
	return func(ctx context.Context, req agent.ToolPermissionRequest, emit func(agent.Event) bool) (agent.ToolPermissionDecision, error) {
		if setup == nil || !setup.IsFunctionTool(req.ToolName) {
			return fallback(ctx, req, emit)
		}
		updated := make(map[string]any, len(req.Input)+1)
		for key, value := range req.Input {
			updated[key] = value
		}
		updated[functionSkillToolUseIDInputKey] = req.ToolID
		if setup.IsAutoTool(req.ToolName) {
			return agent.ToolPermissionDecision{Allow: true, UpdatedInput: updated}, nil
		}
		decision, err := fallback(ctx, req, emit)
		if err != nil || !decision.Allow {
			return decision, err
		}
		approval, err := m.functionSkillRuntime.CreateApproval(ctx, setup, setup.CanonicalToolName(req.ToolName), req.ToolID, req.Input)
		if err != nil {
			return agent.ToolPermissionDecision{}, err
		}
		updated[functionSkillApprovalInputKey] = approval
		decision.UpdatedInput = updated
		return decision, nil
	}
}

func (m *MessageLogic) askUserQuestionHandler(sessionID string) agent.AskUserQuestionHandler {
	return func(
		ctx context.Context,
		req agent.AskUserQuestionRequest,
		emit func(agent.Event) bool,
	) (map[string]any, error) {
		return m.askUserQuestionBroker.Wait(ctx, sessionID, req, emit)
	}
}

func (m *MessageLogic) toolPermissionHandler(sessionID string) agent.ToolPermissionHandler {
	return func(
		ctx context.Context,
		req agent.ToolPermissionRequest,
		emit func(agent.Event) bool,
	) (agent.ToolPermissionDecision, error) {
		return m.toolPermissionBroker.Wait(ctx, sessionID, req, emit)
	}
}

func (m *MessageLogic) SubmitAskUserQuestion(ctx context.Context, req *aiagent.SubmitAskUserQuestionReq) error {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return err
	}
	sessionID, err := requireSessionID(req.GetSessionId())
	if err != nil {
		return err
	}
	if err := m.sessionDao.Exists(ctx, userID, sessionID); err != nil {
		return err
	}
	return m.askUserQuestionBroker.Submit(sessionID, strings.TrimSpace(req.GetToolId()), req.GetAnswersJson(), req.GetResponse())
}

func (m *MessageLogic) SubmitToolPermission(ctx context.Context, req *aiagent.SubmitToolPermissionReq) error {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return err
	}
	sessionID, err := requireSessionID(req.GetSessionId())
	if err != nil {
		return err
	}
	if err := m.sessionDao.Exists(ctx, userID, sessionID); err != nil {
		return err
	}
	return m.toolPermissionBroker.Submit(
		sessionID,
		strings.TrimSpace(req.GetToolId()),
		req.GetAllow(),
		req.GetMessage(),
	)
}

func (m *MessageLogic) resolveMCPServers(ctx context.Context, req *aiagent.StreamMessageReq, ids []string) (map[string]agent.MCPServerConfig, error) {
	if m.mcpLogic == nil {
		return nil, nil
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return m.mcpLogic.ResolveMCPServers(ctx, userIDForSkills(ctx, req.GetUserId()), ids)
}

func (m *MessageLogic) recordAndListSessionReferences(
	ctx context.Context,
	sessionID string,
	parts []*aiagent.MessagePart,
	fallbackExtra []*aiagent.MessageExtra,
) ([]model.SessionReference, error) {
	if m.sessionReferenceDao == nil {
		return referencesFromMessageParts(sessionID, parts, fallbackExtra), nil
	}
	refs := referencesFromMessageParts(sessionID, parts, fallbackExtra)
	if err := m.sessionReferenceDao.UpsertMany(ctx, refs); err != nil {
		return nil, err
	}
	allRefs, err := m.sessionReferenceDao.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return allRefs, nil
}

func (m *MessageLogic) resolveRuntimeConfig(ctx context.Context, req *aiagent.StreamMessageReq) (string, AgentRuntimeOverride, error) {
	modelID := strings.TrimSpace(req.GetModelId())
	override := AgentRuntimeOverride{
		SystemPrompt:   strings.TrimSpace(req.GetSystemPrompt()),
		PermissionMode: strings.TrimSpace(req.GetPermissionMode()),
		MaxTurns:       int(req.GetMaxTurns()),
		Skills:         extractSkillIDs(req.GetExtra()),
	}
	sessionConfigType := strings.TrimSpace(req.GetAgentSessionConfigType())
	if sessionConfigType == "" || m.agentSessionConfigDao == nil {
		return modelID, override, nil
	}
	config, err := m.agentSessionConfigDao.GetEnabled(ctx, sessionConfigType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return modelID, override, nil
		}
		return "", override, err
	}
	if modelID == "" {
		modelID = strings.TrimSpace(config.ModelID)
	}
	if override.SystemPrompt == "" {
		override.SystemPrompt = strings.TrimSpace(config.SystemPrompt)
	}
	if override.PermissionMode == "" {
		override.PermissionMode = strings.TrimSpace(config.PermissionMode)
	}
	if override.MaxTurns <= 0 {
		override.MaxTurns = config.MaxTurns
	}
	return modelID, override, nil
}

func (m *MessageLogic) addMessage(ctx context.Context, sessionID string, role string, content string, parts []*aiagent.MessagePart) (*model.Message, error) {
	now := nowUnixMicro()
	partsJSON, err := encodeParts(parts)
	if err != nil {
		return nil, err
	}
	message := &model.Message{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		PartsJSON: partsJSON,
		CreatedAt: now,
	}
	if err := m.messageDao.Add(ctx, message, summarizeTitle(content)); err != nil {
		return nil, err
	}
	return message, nil
}

func requireSessionID(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("session id is required")
	}
	return sessionID, nil
}

func modelMessageToProto(message *model.Message) *aiagent.Message {
	if message == nil {
		return nil
	}
	parts := decodeParts(message.PartsJSON, message.Content)
	return &aiagent.Message{
		Id:        message.ID,
		SessionId: message.SessionID,
		Role:      message.Role,
		Content:   message.Content,
		Parts:     parts,
		CreatedAt: message.CreatedAt,
		Extra:     extraFromParts(parts),
	}
}

func defaultParts(content string) []*aiagent.MessagePart {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return []*aiagent.MessagePart{{Type: "text", Text: content}}
}

func userMessageParts(req *aiagent.StreamMessageReq) []*aiagent.MessagePart {
	if parts := normalizeUserMessageParts(req.GetMessageParts()); len(parts) > 0 {
		return parts
	}
	content := strings.TrimSpace(req.GetContent())
	parts := make([]*aiagent.MessagePart, 0, len(req.GetExtra())+1)
	for _, item := range normalizeExtra(req.GetExtra()) {
		switch item.GetType() {
		case "skill":
			parts = append(parts, &aiagent.MessagePart{
				Type:    "skill",
				SkillId: item.GetId(),
				Label:   item.GetName(),
			})
		case "mcp":
			parts = append(parts, &aiagent.MessagePart{
				Type:  "mcp",
				McpId: item.GetId(),
				Label: item.GetName(),
			})
		case "function_skill":
			parts = append(parts, &aiagent.MessagePart{Type: "function_skill", SkillId: item.GetId(), Label: item.GetName()})
		}
	}
	if content != "" {
		parts = append(parts, &aiagent.MessagePart{Type: "text", Text: content})
	}
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func normalizeUserMessageParts(parts []*aiagent.MessagePart) []*aiagent.MessagePart {
	next := make([]*aiagent.MessagePart, 0, len(parts))
	for _, part := range parts {
		if part == nil {
			continue
		}
		switch part.GetType() {
		case "text":
			text := strings.TrimSpace(part.GetText())
			if text != "" {
				next = append(next, &aiagent.MessagePart{Type: "text", Text: text})
			}
		case "skill":
			id := strings.TrimSpace(part.GetSkillId())
			if id == "" {
				continue
			}
			label := strings.TrimSpace(part.GetLabel())
			if label == "" {
				label = id
			}
			next = append(next, &aiagent.MessagePart{Type: "skill", SkillId: id, Label: label})
		case "function_skill":
			id := strings.TrimSpace(part.GetSkillId())
			if id == "" {
				continue
			}
			label := strings.TrimSpace(part.GetLabel())
			if label == "" {
				label = id
			}
			next = append(next, &aiagent.MessagePart{Type: "function_skill", SkillId: id, Label: label})
		case "mcp":
			id := strings.TrimSpace(part.GetMcpId())
			if id == "" {
				continue
			}
			label := strings.TrimSpace(part.GetLabel())
			if label == "" {
				label = id
			}
			next = append(next, &aiagent.MessagePart{Type: "mcp", McpId: id, Label: label})
		case "file", "image", "document":
			uuid := strings.TrimSpace(part.GetFileUuid())
			if uuid == "" {
				continue
			}
			url := strings.TrimSpace(part.GetFileUrl())
			if url == "" {
				continue
			}
			next = append(next, &aiagent.MessagePart{Type: part.GetType(), FileUuid: uuid, FileName: strings.TrimSpace(part.GetFileName()), ContentType: strings.TrimSpace(part.GetContentType()), FileSize: part.GetFileSize(), Md5: strings.TrimSpace(part.GetMd5()), FileUrl: url})
		}
	}
	return next
}

func promptFromParts(parts []*aiagent.MessagePart) string {
	if len(parts) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, part := range parts {
		if part == nil {
			continue
		}
		switch part.GetType() {
		case "text":
			builder.WriteString(part.GetText())
		case "skill":
			builder.WriteString(formatPromptMarker("技能", part.GetLabel(), part.GetSkillId()))
		case "function_skill":
			builder.WriteString(formatPromptMarker("功能技能", part.GetLabel(), part.GetSkillId()))
		case "mcp":
			builder.WriteString(formatPromptMarker("MCP", part.GetLabel(), part.GetMcpId()))
		case "file", "image", "document":
			name := strings.TrimSpace(part.GetFileName())
			if name == "" {
				name = part.GetFileUuid()
			}
			builder.WriteString(formatPromptMarker("附件", name, part.GetFileUuid()))
		}
	}
	return strings.TrimSpace(builder.String())
}

func (m *MessageLogic) buildAgentUserContent(ctx context.Context, parts []*aiagent.MessagePart, prompt string) (any, error) {
	content := make([]map[string]any, 0, len(parts)+1)
	if strings.TrimSpace(prompt) != "" {
		content = append(content, map[string]any{"type": "text", "text": prompt})
	}
	for _, part := range parts {
		if part == nil || (part.GetType() != "image" && part.GetType() != "file" && part.GetType() != "document") {
			continue
		}
		partContent, err := m.resolveAttachmentContent(ctx, part)
		if err != nil {
			return nil, err
		}
		content = append(content, partContent)
	}
	if len(content) == 0 {
		return nil, nil
	}
	return content, nil
}

func (m *MessageLogic) resolveAttachmentContent(ctx context.Context, part *aiagent.MessagePart) (map[string]any, error) {
	name := strings.TrimSpace(part.GetFileName())
	if name == "" {
		name = part.GetFileUuid()
	}
	contentType := strings.ToLower(strings.TrimSpace(part.GetContentType()))
	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(name))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	data, err := downloadAttachment(ctx, part.GetFileUrl(), 16*1024*1024)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	if strings.HasPrefix(contentType, "image/") {
		return map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": contentType, "data": encoded}}, nil
	}
	if contentType == "application/pdf" {
		return map[string]any{"type": "document", "source": map[string]any{"type": "base64", "media_type": contentType, "data": encoded}}, nil
	}
	if strings.HasPrefix(contentType, "text/") {
		return map[string]any{"type": "text", "text": fmt.Sprintf("附件 %s 内容：\n%s", name, string(data))}, nil
	}
	return map[string]any{"type": "text", "text": fmt.Sprintf("附件 %s（%s）已上传，但当前不支持直接解析该文件类型。", name, contentType)}, nil
}

func downloadAttachment(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("attachment url is invalid: %w", err)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("download attachment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download attachment: %s", resp.Status)
	}
	if resp.ContentLength > limit {
		return nil, errors.New("attachment exceeds analysis limit")
	}
	buffer := bytes.NewBuffer(make([]byte, 0, minInt64(resp.ContentLength, limit)))
	if _, err := buffer.ReadFrom(io.LimitReader(resp.Body, limit+1)); err != nil {
		return nil, err
	}
	if int64(buffer.Len()) > limit {
		return nil, errors.New("attachment exceeds analysis limit")
	}
	return buffer.Bytes(), nil
}

func minInt64(left, right int64) int {
	if left < 0 || left > right {
		return int(right)
	}
	return int(left)
}

func formatPromptMarker(kind string, name string, id string) string {
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	if name == "" {
		name = id
	}
	if id == "" {
		return fmt.Sprintf("[%s:%s]", kind, name)
	}
	return fmt.Sprintf("[%s:%s](%s)", kind, name, id)
}

func extraFromParts(parts []*aiagent.MessagePart) []*aiagent.MessageExtra {
	extra := make([]*aiagent.MessageExtra, 0)
	for index, part := range parts {
		if part == nil {
			continue
		}
		switch part.GetType() {
		case "skill":
			id := strings.TrimSpace(part.GetSkillId())
			if id == "" {
				continue
			}
			name := strings.TrimSpace(part.GetLabel())
			if name == "" {
				name = id
			}
			extra = append(extra, &aiagent.MessageExtra{Type: "skill", Id: id, Name: name, Index: int32(index)})
		case "function_skill":
			id := strings.TrimSpace(part.GetSkillId())
			if id == "" {
				continue
			}
			name := strings.TrimSpace(part.GetLabel())
			if name == "" {
				name = id
			}
			extra = append(extra, &aiagent.MessageExtra{Type: "function_skill", Id: id, Name: name, Index: int32(index)})
		case "mcp":
			id := strings.TrimSpace(part.GetMcpId())
			if id == "" {
				continue
			}
			name := strings.TrimSpace(part.GetLabel())
			if name == "" {
				name = id
			}
			extra = append(extra, &aiagent.MessageExtra{Type: "mcp", Id: id, Name: name, Index: int32(index)})
		}
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

func extractSkillIDs(extra []*aiagent.MessageExtra) []string {
	items := normalizeExtra(extra)
	skills := make([]string, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		if item.GetType() == "skill" {
			if _, ok := seen[item.GetId()]; ok {
				continue
			}
			seen[item.GetId()] = struct{}{}
			skills = append(skills, item.GetId())
		}
	}
	return skills
}

func extractSkillIDsFromParts(parts []*aiagent.MessagePart, fallbackExtra []*aiagent.MessageExtra) []string {
	skills := make([]string, 0)
	seen := make(map[string]struct{})
	for _, part := range parts {
		if part == nil || part.GetType() != "skill" {
			continue
		}
		id := strings.TrimSpace(part.GetSkillId())
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		skills = append(skills, id)
	}
	if len(skills) > 0 {
		return skills
	}
	return extractSkillIDs(fallbackExtra)
}

func extractFunctionSkillIDsFromParts(parts []*aiagent.MessagePart, fallbackExtra []*aiagent.MessageExtra) []string {
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	for _, part := range parts {
		if part == nil || part.GetType() != "function_skill" {
			continue
		}
		id := strings.TrimSpace(part.GetSkillId())
		if id == "" {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		return ids
	}
	for _, item := range normalizeExtra(fallbackExtra) {
		if item.GetType() != "function_skill" {
			continue
		}
		if _, ok := seen[item.GetId()]; !ok {
			seen[item.GetId()] = struct{}{}
			ids = append(ids, item.GetId())
		}
	}
	return ids
}

func extractMCPIDs(extra []*aiagent.MessageExtra) []string {
	items := normalizeExtra(extra)
	mcps := make([]string, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		if item.GetType() != "mcp" {
			continue
		}
		id := normalizeMCPID(item.GetId())
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		mcps = append(mcps, id)
	}
	return mcps
}

func extractMCPIDsFromParts(parts []*aiagent.MessagePart, fallbackExtra []*aiagent.MessageExtra) []string {
	mcps := make([]string, 0)
	seen := make(map[string]struct{})
	for _, part := range parts {
		if part == nil || part.GetType() != "mcp" {
			continue
		}
		id := normalizeMCPID(part.GetMcpId())
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		mcps = append(mcps, id)
	}
	if len(mcps) > 0 {
		return mcps
	}
	return extractMCPIDs(fallbackExtra)
}

func referencesFromMessageParts(sessionID string, parts []*aiagent.MessagePart, fallbackExtra []*aiagent.MessageExtra) []model.SessionReference {
	now := nowUnixMicro()
	refs := make([]model.SessionReference, 0)
	seen := make(map[string]struct{})
	addRef := func(refType string, id string, name string) {
		id = strings.TrimSpace(id)
		if refType == model.SessionReferenceTypeMCP {
			id = normalizeMCPID(id)
		}
		if id == "" {
			return
		}
		key := refType + ":" + id
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		name = strings.TrimSpace(name)
		if name == "" {
			name = id
		}
		refs = append(refs, model.SessionReference{
			ID:        uuid.NewString(),
			SessionID: sessionID,
			RefType:   refType,
			RefID:     id,
			Name:      name,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	for _, part := range parts {
		if part == nil {
			continue
		}
		switch part.GetType() {
		case "skill":
			addRef(model.SessionReferenceTypeSkill, part.GetSkillId(), part.GetLabel())
		case "mcp":
			addRef(model.SessionReferenceTypeMCP, part.GetMcpId(), part.GetLabel())
		case "function_skill":
			addRef(model.SessionReferenceTypeFunctionSkill, part.GetSkillId(), part.GetLabel())
		}
	}
	if len(refs) > 0 {
		return refs
	}
	for _, item := range normalizeExtra(fallbackExtra) {
		switch item.GetType() {
		case "skill":
			addRef(model.SessionReferenceTypeSkill, item.GetId(), item.GetName())
		case "mcp":
			addRef(model.SessionReferenceTypeMCP, item.GetId(), item.GetName())
		case "function_skill":
			addRef(model.SessionReferenceTypeFunctionSkill, item.GetId(), item.GetName())
		}
	}
	return refs
}

func extractSessionReferenceIDs(refs []model.SessionReference, refType string) []string {
	ids := make([]string, 0, len(refs))
	seen := make(map[string]struct{})
	for _, ref := range refs {
		if ref.RefType != refType {
			continue
		}
		id := strings.TrimSpace(ref.RefID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func normalizeExtra(extra []*aiagent.MessageExtra) []*aiagent.MessageExtra {
	items := make([]*aiagent.MessageExtra, 0, len(extra))
	for _, item := range extra {
		if item == nil {
			continue
		}
		extraType := strings.TrimSpace(item.GetType())
		id := strings.TrimSpace(item.GetId())
		if id == "" || (extraType != "skill" && extraType != "mcp" && extraType != "function_skill") {
			continue
		}
		name := strings.TrimSpace(item.GetName())
		if name == "" {
			name = id
		}
		items = append(items, &aiagent.MessageExtra{
			Type:  extraType,
			Id:    id,
			Name:  name,
			Index: item.GetIndex(),
		})
	}
	return items
}

func encodeParts(parts []*aiagent.MessagePart) (string, error) {
	if len(parts) == 0 {
		return "", nil
	}
	data, err := json.Marshal(parts)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeParts(partsJSON string, content string) []*aiagent.MessagePart {
	if strings.TrimSpace(partsJSON) == "" {
		return defaultParts(content)
	}
	var parts []*aiagent.MessagePart
	if err := json.Unmarshal([]byte(partsJSON), &parts); err != nil {
		return defaultParts(content)
	}
	return parts
}

func summarizeTitle(content string) string {
	const max = 28
	content = strings.TrimSpace(content)
	runes := []rune(content)
	if len(runes) <= max {
		return content
	}
	return string(runes[:max]) + "..."
}

func streamRespFromEvent(event agent.Event, message *aiagent.Message) *aiagent.StreamMessageResp {
	resp := &aiagent.StreamMessageResp{
		Type:            event.Type,
		Text:            event.Text,
		ToolId:          event.ToolID,
		ToolName:        event.ToolName,
		ToolTitle:       event.ToolTitle,
		ToolDescription: event.ToolDescription,
		ToolInput:       event.ToolInput,
		ResultText:      event.ResultText,
		AgentSessionId:  event.AgentSessionID,
		IsError:         event.IsError,
		Done:            event.Done,
		Message:         message,
	}
	resp.Content = mustMarshalStreamContent(resp)
	return resp
}

func mustMarshalStreamContent(resp *aiagent.StreamMessageResp) string {
	payload := struct {
		Type            string           `json:"type"`
		Text            string           `json:"text,omitempty"`
		ToolID          string           `json:"toolId,omitempty"`
		ToolName        string           `json:"toolName,omitempty"`
		ToolTitle       string           `json:"toolTitle,omitempty"`
		ToolDescription string           `json:"toolDescription,omitempty"`
		ToolInput       string           `json:"toolInput,omitempty"`
		ResultText      string           `json:"resultText,omitempty"`
		SessionID       string           `json:"agentSessionId,omitempty"`
		IsError         bool             `json:"isError,omitempty"`
		Done            bool             `json:"done,omitempty"`
		Message         *aiagent.Message `json:"message,omitempty"`
	}{
		Type:            resp.Type,
		Text:            resp.Text,
		ToolID:          resp.ToolId,
		ToolName:        resp.ToolName,
		ToolTitle:       resp.ToolTitle,
		ToolDescription: resp.ToolDescription,
		ToolInput:       resp.ToolInput,
		ResultText:      resp.ResultText,
		SessionID:       resp.AgentSessionId,
		IsError:         resp.IsError,
		Done:            resp.Done,
		Message:         resp.Message,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return `{"type":"error","text":"encode stream event failed"}`
	}
	return string(data)
}

type responseParts struct {
	items []*aiagent.MessagePart
}

func newResponseParts() *responseParts {
	return &responseParts{items: make([]*aiagent.MessagePart, 0)}
}

func (p *responseParts) apply(event agent.Event) {
	switch event.Type {
	case "text_delta":
		p.appendText(event.Text)
	case "thinking_start":
		p.startThinking()
	case "thinking_delta":
		p.appendThinking(event.Text)
	case "thinking_stop":
		p.finishThinking()
	case "tool_start":
		p.upsertTool(event.ToolID, &aiagent.MessagePart{
			Type:     "tool",
			ToolId:   event.ToolID,
			ToolName: event.ToolName,
			Input:    event.ToolInput,
			Status:   "running",
		})
	case "tool_delta":
		if tool := p.findTool(event.ToolID); tool != nil {
			tool.Input += event.ToolInput
		} else {
			p.upsertTool(event.ToolID, &aiagent.MessagePart{
				Type:   "tool",
				ToolId: event.ToolID,
				Input:  event.ToolInput,
				Status: "running",
			})
		}
	case "tool_stop":
		p.upsertTool(event.ToolID, &aiagent.MessagePart{
			Type:   "tool",
			ToolId: event.ToolID,
			Input:  event.ToolInput,
			Status: "finished",
		})
	case "tool_result":
		status := "finished"
		if event.IsError {
			status = "error"
		}
		p.upsertTool(event.ToolID, &aiagent.MessagePart{
			Type:    "tool",
			ToolId:  event.ToolID,
			Result:  event.ResultText,
			Status:  status,
			IsError: event.IsError,
		})
	case "ask_user_question":
		p.upsertTool(event.ToolID, &aiagent.MessagePart{
			Type:     "tool",
			ToolId:   event.ToolID,
			ToolName: event.ToolName,
			Input:    event.ToolInput,
			Status:   "waiting",
		})
	case "tool_permission_request":
		p.upsertTool(event.ToolID, &aiagent.MessagePart{
			Type:            "tool",
			ToolId:          event.ToolID,
			ToolName:        event.ToolName,
			ToolTitle:       event.ToolTitle,
			ToolDescription: event.ToolDescription,
			Input:           event.ToolInput,
			Status:          "waiting_permission",
		})
	}
}

func (p *responseParts) parts() []*aiagent.MessagePart {
	if len(p.items) == 0 {
		return nil
	}
	return p.items
}

func (p *responseParts) appendText(text string) {
	if text == "" {
		return
	}
	last := len(p.items) - 1
	if last >= 0 && p.items[last].Type == "text" {
		p.items[last].Text += text
		return
	}
	p.items = append(p.items, &aiagent.MessagePart{Type: "text", Text: text})
}

func (p *responseParts) findTool(toolID string) *aiagent.MessagePart {
	for _, item := range p.items {
		if item.Type == "tool" && item.ToolId == toolID {
			return item
		}
	}
	return nil
}

func (p *responseParts) findThinking() *aiagent.MessagePart {
	for i := len(p.items) - 1; i >= 0; i-- {
		if p.items[i].Type == "thinking" {
			return p.items[i]
		}
	}
	return nil
}

func (p *responseParts) startThinking() {
	thinking := p.findThinking()
	if thinking != nil && thinking.Status == "running" {
		return
	}
	p.items = append(p.items, &aiagent.MessagePart{
		Type:   "thinking",
		Status: "running",
	})
}

func (p *responseParts) appendThinking(text string) {
	if text == "" {
		return
	}
	thinking := p.findThinking()
	if thinking == nil {
		p.startThinking()
		thinking = p.findThinking()
	}
	thinking.Text += text
}

func (p *responseParts) finishThinking() {
	thinking := p.findThinking()
	if thinking == nil {
		return
	}
	thinking.Status = "finished"
}

func (p *responseParts) upsertTool(toolID string, patch *aiagent.MessagePart) {
	if toolID == "" {
		return
	}
	tool := p.findTool(toolID)
	if tool == nil {
		if patch.Type == "" {
			patch.Type = "tool"
		}
		if patch.Status == "" {
			patch.Status = "running"
		}
		p.items = append(p.items, patch)
		return
	}
	if patch.ToolName != "" {
		tool.ToolName = patch.ToolName
	}
	if patch.ToolTitle != "" {
		tool.ToolTitle = patch.ToolTitle
	}
	if patch.ToolDescription != "" {
		tool.ToolDescription = patch.ToolDescription
	}
	if patch.Input != "" {
		tool.Input = patch.Input
	}
	if patch.Result != "" {
		tool.Result = patch.Result
	}
	if patch.Status != "" {
		tool.Status = patch.Status
	}
	tool.IsError = patch.IsError
}
