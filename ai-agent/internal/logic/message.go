package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	sessionDao                  *dao.Session
	messageDao                  *dao.Message
	sessionReferenceDao         *dao.SessionReference
	runnerFactory               runnerFactory
	agentModelLogic             *AgentModelLogic
	agentEngine                 *AgentEngine
	agentSessionConfigDao       *dao.AgentSessionConfig
	skillLogic                  *SkillLogic
	mcpLogic                    *MCPLogic
	functionSkillRuntime        *FunctionSkillRuntime
	functionConversationRuntime *FunctionConversationRuntime
	attachmentResolver          *AttachmentResolver
	askUserQuestionBroker       *AskUserQuestionBroker
	toolPermissionBroker        *ToolPermissionBroker
	navigationRuntime           *NavigationRuntime
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

func (m *MessageLogic) SetAttachmentResolver(resolver *AttachmentResolver) {
	if m != nil {
		m.attachmentResolver = resolver
	}
}

func (m *MessageLogic) SetNavigationRuntime(runtime *NavigationRuntime) {
	if m != nil {
		m.navigationRuntime = runtime
	}
}

func (m *MessageLogic) SetFunctionConversationRuntime(runtime *FunctionConversationRuntime) {
	if m != nil {
		m.functionConversationRuntime = runtime
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
	operationID := strings.TrimSpace(req.GetFunctionOperationId())
	if operationID != "" && (strings.TrimSpace(req.GetFunctionId()) == "" || strings.TrimSpace(req.GetFunctionConversation()) == "") {
		return errors.New("function conversation operation context is incomplete")
	}
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
	preparedAttachments, err := m.prepareAttachments(ctx, sessionID, userParts)
	if err != nil {
		return err
	}
	prompt = attachmentPrompt(prompt, preparedAttachments)
	userContent, err := m.buildAgentUserContent(userParts, prompt, preparedAttachments)
	if err != nil {
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
	if preparedAttachments != nil && preparedAttachments.Dir != "" {
		engineConfig.AddDirs = append(engineConfig.AddDirs, preparedAttachments.Dir)
	}
	engineConfig.UserContent = userContent
	if _, err := m.addOperationMessage(ctx, sessionID, model.RoleUser, content, userParts, operationID, "", ""); err != nil {
		return err
	}
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
		message, saveErr := m.addOperationMessage(persistCtx, sessionID, model.RoleAssistant, answer.String(), parts.parts(), operationID, "cancelled", "stream interrupted")
		if saveErr == nil {
			if updateErr := m.sessionDao.UpdateAgentSession(persistCtx, userID, sessionID, agentSessionID, message.CreatedAt); updateErr == nil {
				if m.functionConversationRuntime != nil && operationID != "" {
					_ = m.functionConversationRuntime.Finish(persistCtx, operationID, "cancelled", "stream interrupted")
				}
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
				message, err := m.addOperationMessage(ctx, sessionID, model.RoleAssistant, answer.String(), parts.parts(), operationID, event.TerminalStatus, event.TerminalReason)
				if err != nil {
					return err
				}
				if event.AgentSessionID != "" {
					agentSessionID = event.AgentSessionID
				}
				if err := m.sessionDao.UpdateAgentSession(ctx, userID, sessionID, agentSessionID, message.CreatedAt); err != nil {
					return err
				}
				if m.functionConversationRuntime != nil && operationID != "" {
					_ = m.functionConversationRuntime.Finish(context.WithoutCancel(ctx), operationID, event.TerminalStatus, event.TerminalReason)
				}
				return send(streamRespFromEvent(agent.Event{Type: "done", AgentSessionID: agentSessionID, Done: true, TerminalStatus: event.TerminalStatus, TerminalReason: event.TerminalReason}, modelMessageToProto(message)))
			}
			if event.Type != "" {
				if err := send(streamRespFromEvent(event, nil)); err != nil {
					return err
				}
			}
			if actionJSON := uiActionFromToolResult(event); actionJSON != "" {
				if err := send(streamRespFromEvent(agent.Event{Type: "ui_action", UIActionJSON: actionJSON}, nil)); err != nil {
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
			if m.functionConversationRuntime != nil && operationID != "" {
				_ = m.functionConversationRuntime.Finish(context.WithoutCancel(ctx), operationID, "error", err.Error())
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
	if m.navigationRuntime != nil {
		navigationServer, navigationErr := m.navigationRuntime.Server(ctx)
		if navigationErr != nil {
			return AgentEngineRunConfig{}, navigationErr
		}
		if navigationServer.Instance != nil {
			if engineConfig.SDKMCPServers == nil {
				engineConfig.SDKMCPServers = make(map[string]claudeagentsdk.MCPServerConfig)
			}
			engineConfig.SDKMCPServers[navigationMCPServerID] = navigationServer
		}
	}
	var conversationSetup *FunctionConversationSetup
	if strings.TrimSpace(req.GetFunctionOperationId()) != "" {
		if m.functionConversationRuntime == nil {
			return AgentEngineRunConfig{}, errors.New("function conversation runtime is not configured")
		}
		conversationSetup, err = m.functionConversationRuntime.Prepare(ctx, userIDForSkills(ctx, req.GetUserId()), req.GetFunctionOperationId(), req.GetFunctionId(), sessionID, req.GetFunctionConversation())
		if err != nil {
			return AgentEngineRunConfig{}, err
		}
		if engineConfig.SDKMCPServers == nil {
			engineConfig.SDKMCPServers = make(map[string]claudeagentsdk.MCPServerConfig)
		}
		for id, server := range conversationSetup.SDKMCPServers {
			engineConfig.SDKMCPServers[id] = server
		}
		basePermission := engineConfig.ToolPermission
		engineConfig.ToolPermission = m.functionConversationToolPermissionHandler(conversationSetup, basePermission)
		engineConfig.ForceToolPermission = conversationSetup.IsCompletionTool
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
	baseForcePermission := engineConfig.ForceToolPermission
	engineConfig.ForceToolPermission = func(toolName string) bool {
		return setup.IsFunctionTool(toolName) || (baseForcePermission != nil && baseForcePermission(toolName))
	}
	return engineConfig, nil
}

func (m *MessageLogic) functionConversationToolPermissionHandler(setup *FunctionConversationSetup, fallback agent.ToolPermissionHandler) agent.ToolPermissionHandler {
	return func(ctx context.Context, req agent.ToolPermissionRequest, emit func(agent.Event) bool) (agent.ToolPermissionDecision, error) {
		if setup == nil || !setup.IsCompletionTool(req.ToolName) {
			return fallback(ctx, req, emit)
		}
		updated := make(map[string]any, len(req.Input)+1)
		for key, value := range req.Input {
			updated[key] = value
		}
		updated[functionConversationToolUseIDInputKey] = req.ToolID
		return agent.ToolPermissionDecision{Allow: true, UpdatedInput: updated}, nil
	}
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
	return m.addOperationMessage(ctx, sessionID, role, content, parts, "", "", "")
}

func (m *MessageLogic) addOperationMessage(ctx context.Context, sessionID string, role string, content string, parts []*aiagent.MessagePart, operationID, terminalStatus, terminalReason string) (*model.Message, error) {
	now := nowUnixMicro()
	partsJSON, err := encodeParts(parts)
	if err != nil {
		return nil, err
	}
	message := &model.Message{
		ID:             uuid.NewString(),
		SessionID:      sessionID,
		OperationID:    strings.TrimSpace(operationID),
		Role:           role,
		Content:        content,
		PartsJSON:      partsJSON,
		TerminalStatus: strings.TrimSpace(terminalStatus),
		TerminalReason: strings.TrimSpace(terminalReason),
		CreatedAt:      now,
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
		Id:             message.ID,
		SessionId:      message.SessionID,
		Role:           message.Role,
		Content:        message.Content,
		Parts:          parts,
		CreatedAt:      message.CreatedAt,
		Extra:          extraFromParts(parts),
		OperationId:    message.OperationID,
		TerminalStatus: message.TerminalStatus,
		TerminalReason: message.TerminalReason,
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
		case "function_operation_bootstrap":
			metadata := strings.TrimSpace(part.GetText())
			if metadata != "" {
				next = append(next, &aiagent.MessagePart{Type: "function_operation_bootstrap", Text: metadata})
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

func (m *MessageLogic) prepareAttachments(ctx context.Context, sessionID string, parts []*aiagent.MessagePart) (*PreparedAttachments, error) {
	if len(attachmentParts(parts)) == 0 {
		return &PreparedAttachments{}, nil
	}
	if m.attachmentResolver == nil {
		return nil, errors.New("attachment resolver is not configured")
	}
	return m.attachmentResolver.Prepare(ctx, sessionID, parts)
}

func (m *MessageLogic) buildAgentUserContent(parts []*aiagent.MessagePart, prompt string, prepared *PreparedAttachments) (any, error) {
	content := make([]map[string]any, 0, len(parts)+1)
	if strings.TrimSpace(prompt) != "" {
		content = append(content, map[string]any{"type": "text", "text": prompt})
	}
	for _, part := range parts {
		if part == nil || (part.GetType() != "image" && part.GetType() != "file" && part.GetType() != "document") {
			continue
		}
		if prepared == nil {
			continue
		}
		if item, ok := prepared.Items[part.GetFileUuid()]; ok {
			if partContent, ok := nativeAttachmentContent(item); ok {
				content = append(content, partContent)
			}
		}
	}
	if len(content) == 0 {
		return nil, nil
	}
	return content, nil
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
		UiActionJson:    event.UIActionJSON,
		TerminalStatus:  event.TerminalStatus,
		TerminalReason:  event.TerminalReason,
		IsError:         event.IsError,
		Done:            event.Done,
		Message:         message,
	}
	resp.Content = mustMarshalStreamContent(resp)
	return resp
}

func uiActionFromToolResult(event agent.Event) string {
	if event.Type != "tool_result" {
		return ""
	}
	toolName := strings.ToLower(event.ToolName)
	if !strings.Contains(toolName, "navigate_to_target") {
		return ""
	}
	var value any
	if json.Unmarshal([]byte(event.ResultText), &value) != nil {
		return ""
	}
	action, ok := findUIAction(value, 0)
	if !ok {
		return ""
	}
	data, err := json.Marshal(action)
	if err != nil {
		return ""
	}
	return string(data)
}

// MCP tool results are exposed by the SDK as a content-block array. The text
// block itself contains the JSON returned by the SDK MCP handler, so unwrap
// both layers while also accepting the direct-object shape used by tests and
// other runners.
func findUIAction(value any, depth int) (any, bool) {
	if depth > 4 || value == nil {
		return nil, false
	}
	switch item := value.(type) {
	case string:
		var nested any
		if json.Unmarshal([]byte(item), &nested) != nil {
			return nil, false
		}
		return findUIAction(nested, depth+1)
	case []any:
		for _, child := range item {
			if action, ok := findUIAction(child, depth+1); ok {
				return action, true
			}
		}
	case map[string]any:
		if item["action"] == "navigate" {
			if target, ok := item["target"].(map[string]any); ok && strings.TrimSpace(navStringValue(target["targetId"])) != "" {
				return item, true
			}
		}
		for _, key := range []string{"uiAction", "ui_action", "content", "text"} {
			if child, exists := item[key]; exists {
				if action, ok := findUIAction(child, depth+1); ok {
					return action, true
				}
			}
		}
	}
	return nil, false
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
		UIActionJSON    string           `json:"uiActionJson,omitempty"`
		TerminalStatus  string           `json:"terminalStatus,omitempty"`
		TerminalReason  string           `json:"terminalReason,omitempty"`
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
		UIActionJSON:    resp.UiActionJson,
		TerminalStatus:  resp.TerminalStatus,
		TerminalReason:  resp.TerminalReason,
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
