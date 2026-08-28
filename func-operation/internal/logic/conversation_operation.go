package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	conversationProgressProductTool    = "submit_product_document_draft"
	conversationProgressTechnicalTool  = "submit_technical_document_draft"
	conversationProgressGenerationTool = "submit_generated_app"
)

type ConversationOperationLogic struct {
	functions  *dao.Function
	operations *dao.FunctionConversationOperation
	executions *dao.FunctionConversationProgressExecution
	authorizer *FunctionAuthorizer
}

func NewConversationOperationLogic(functions *dao.Function, operations *dao.FunctionConversationOperation, executions *dao.FunctionConversationProgressExecution, authorizer *FunctionAuthorizer) *ConversationOperationLogic {
	return &ConversationOperationLogic{functions: functions, operations: operations, executions: executions, authorizer: authorizer}
}

func (l *ConversationOperationLogic) Start(ctx context.Context, req *funcoperation.StartFunctionConversationOperationReq) (*funcoperation.FunctionConversationOperation, error) {
	if err := l.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	function, conversation, sessionID, err := l.resolveScope(ctx, req.GetId(), req.GetConversation())
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	if operationID := strings.TrimSpace(req.GetOperationId()); operationID != "" {
		operation, getErr := l.operations.Get(ctx, operationID)
		if getErr != nil {
			return nil, getErr
		}
		if err := validateOperationScope(operation, userID, function.UUID, sessionID, conversation); err != nil {
			return nil, err
		}
		if !canResumeConversationOperation(operation.State) {
			return nil, errors.New("conversation operation is not resumable")
		}
		if err := l.operations.Resume(ctx, operation.UUID, now); err != nil {
			return nil, err
		}
		operation, err = l.operations.Get(ctx, operation.UUID)
		if err != nil {
			return nil, err
		}
		return conversationOperationToProto(operation), nil
	}
	if err := l.operations.SupersedeActive(ctx, function.UUID, sessionID, conversation, userID, now); err != nil {
		return nil, err
	}
	baseline, _ := json.Marshal(map[string]int64{
		"functionVersion": function.FunctionVersion, "productDocVersion": function.ProductDocVersion, "productDraftVersion": function.ProductDraftVersion,
		"technicalDocVersion": function.TechnicalDocVersion, "technicalDraftVersion": function.TechnicalDraftVersion,
		"codeVersion": function.CodeVersion, "codeDraftVersion": function.CodeDraftVersion,
	})
	operation := &model.FunctionConversationOperation{UUID: uuid.NewString(), FunctionID: function.UUID, SessionID: sessionID, Conversation: conversation, UserID: userID, State: model.ConversationOperationStateRunning, BaselineJSON: string(baseline), CreatedAt: now, UpdatedAt: now}
	if err := l.operations.Create(ctx, operation); err != nil {
		return nil, err
	}
	return conversationOperationToProto(operation), nil
}

func (l *ConversationOperationLogic) GetLatest(ctx context.Context, req *funcoperation.GetLatestFunctionConversationOperationReq) (*funcoperation.FunctionConversationOperation, error) {
	if err := l.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	function, conversation, sessionID, err := l.resolveScope(ctx, req.GetId(), req.GetConversation())
	if err != nil {
		return nil, err
	}
	operation, err := l.operations.GetLatest(ctx, function.UUID, sessionID, conversation, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return conversationOperationToProto(operation), nil
}

func (l *ConversationOperationLogic) Finish(ctx context.Context, req *funcoperation.FinishFunctionConversationOperationReq) (*funcoperation.FunctionConversationOperation, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	operation, err := l.operations.Get(ctx, strings.TrimSpace(req.GetOperationId()))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(operation.UserID) != userID {
		return nil, errors.New("conversation operation does not belong to the current user")
	}
	terminalStatus := normalizeConversationTerminalStatus(req.GetTerminalStatus())
	state := conversationStateForTerminal(terminalStatus)
	operation, err = l.operations.FinishIfActive(ctx, operation.UUID, state, terminalStatus, compactConversationText(req.GetTerminalReason(), 1000), nowUnixMicro())
	if err != nil {
		return nil, err
	}
	return conversationOperationToProto(operation), nil
}

func (l *ConversationOperationLogic) SubmitProgress(ctx context.Context, req *funcoperation.SubmitFunctionConversationProgressReq) (*funcoperation.FunctionConversationOperation, bool, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, false, err
	}
	operation, err := l.operations.Get(ctx, strings.TrimSpace(req.GetOperationId()))
	if err != nil {
		return nil, false, err
	}
	if err := validateOperationScope(operation, userID, strings.TrimSpace(req.GetFunctionId()), strings.TrimSpace(req.GetSessionId()), normalizeFunctionConversation(req.GetConversation())); err != nil {
		return nil, false, err
	}
	toolName := strings.TrimSpace(req.GetToolName())
	if !progressToolMatchesConversation(toolName, operation.Conversation) {
		return nil, false, fmt.Errorf("tool %q is not allowed for %s conversation", toolName, operation.Conversation)
	}
	toolUseID := strings.TrimSpace(req.GetToolUseId())
	if toolUseID == "" {
		return nil, false, errors.New("tool use id is required")
	}
	now := nowUnixMicro()
	outcome := compactConversationText(req.GetSummary(), 2000)
	if outcome == "" {
		return nil, false, errors.New("completion summary is required")
	}
	execution := &model.FunctionConversationProgressExecution{UUID: uuid.NewString(), OperationID: operation.UUID, FunctionID: operation.FunctionID, SessionID: operation.SessionID, Conversation: operation.Conversation, ToolName: toolName, ToolUseID: toolUseID, UserID: userID, Status: "succeeded", Outcome: outcome, ResultJSON: `{"completed":true}`, CreatedAt: now, UpdatedAt: now}
	_, completed, alreadySubmitted, err := l.executions.Complete(ctx, execution, operation)
	if err != nil {
		return nil, false, err
	}
	return conversationOperationToProto(completed), alreadySubmitted, nil
}

func (l *ConversationOperationLogic) resolveScope(ctx context.Context, functionID, conversationValue string) (*model.Function, string, string, error) {
	function, err := l.functions.Get(ctx, strings.TrimSpace(functionID))
	if err != nil {
		return nil, "", "", err
	}
	conversation := normalizeFunctionConversation(conversationValue)
	if conversation == "" {
		return nil, "", "", errors.New("unsupported conversation")
	}
	sessionID := sessionIDForConversation(function, conversation)
	if sessionID == "" {
		return nil, "", "", errors.New("function conversation session is not ready")
	}
	return function, conversation, sessionID, nil
}

func sessionIDForConversation(function *model.Function, conversation string) string {
	if function == nil {
		return ""
	}
	switch conversation {
	case functionConversationProduct:
		return strings.TrimSpace(function.ProductSessionID)
	case functionConversationTechnical:
		return strings.TrimSpace(function.TechnicalSessionID)
	case functionConversationGeneration:
		return strings.TrimSpace(function.GenerationSessionID)
	default:
		return ""
	}
}

func validateOperationScope(operation *model.FunctionConversationOperation, userID, functionID, sessionID, conversation string) error {
	if operation == nil {
		return errors.New("conversation operation is required")
	}
	if operation.UserID != userID || operation.FunctionID != functionID || operation.SessionID != sessionID || operation.Conversation != conversation {
		return errors.New("conversation operation scope does not match the current request")
	}
	return nil
}

func canResumeConversationOperation(state string) bool {
	return state == model.ConversationOperationStateAwaitingUser || state == model.ConversationOperationStateNeedsContinue
}

func normalizeConversationTerminalStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case model.ConversationTerminalMaxTurns:
		return model.ConversationTerminalMaxTurns
	case model.ConversationTerminalCancelled:
		return model.ConversationTerminalCancelled
	case model.ConversationTerminalError:
		return model.ConversationTerminalError
	default:
		return model.ConversationTerminalNormal
	}
}

func conversationStateForTerminal(terminalStatus string) string {
	switch terminalStatus {
	case model.ConversationTerminalMaxTurns:
		return model.ConversationOperationStateNeedsContinue
	case model.ConversationTerminalCancelled:
		return model.ConversationOperationStateCancelled
	case model.ConversationTerminalError:
		return model.ConversationOperationStateBlocked
	default:
		return model.ConversationOperationStateAwaitingUser
	}
}

func progressToolMatchesConversation(toolName, conversation string) bool {
	switch conversation {
	case functionConversationProduct:
		return toolName == conversationProgressProductTool
	case functionConversationTechnical:
		return toolName == conversationProgressTechnicalTool
	case functionConversationGeneration:
		return toolName == conversationProgressGenerationTool
	default:
		return false
	}
}

func compactConversationText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

func conversationOperationToProto(item *model.FunctionConversationOperation) *funcoperation.FunctionConversationOperation {
	if item == nil {
		return nil
	}
	return &funcoperation.FunctionConversationOperation{Id: item.UUID, FunctionId: item.FunctionID, SessionId: item.SessionID, Conversation: item.Conversation, State: item.State, TerminalStatus: item.TerminalStatus, TerminalReason: item.TerminalReason, Outcome: item.Outcome, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, FinishedAt: item.FinishedAt}
}
