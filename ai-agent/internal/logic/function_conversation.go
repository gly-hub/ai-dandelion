package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

const (
	functionConversationMCPServerID       = "function-conversation-progress"
	functionConversationToolUseIDInputKey = "__function_conversation_tool_use_id"
)

type FunctionConversationRuntime struct {
	provider FunctionSkillClientProvider
}

type FunctionConversationSetup struct {
	OperationID   string
	FunctionID    string
	SessionID     string
	Conversation  string
	ToolName      string
	SDKMCPServers map[string]claudeagentsdk.MCPServerConfig
}

func NewFunctionConversationRuntime(provider FunctionSkillClientProvider) *FunctionConversationRuntime {
	return &FunctionConversationRuntime{provider: provider}
}

func (r *FunctionConversationRuntime) Prepare(ctx context.Context, userID, operationID, functionID, sessionID, conversation string) (*FunctionConversationSetup, error) {
	if r == nil || r.provider == nil {
		return nil, errors.New("function conversation runtime is not configured")
	}
	operationID, functionID, sessionID, conversation = strings.TrimSpace(operationID), strings.TrimSpace(functionID), strings.TrimSpace(sessionID), strings.TrimSpace(conversation)
	if operationID == "" || functionID == "" || sessionID == "" || conversation == "" {
		return nil, errors.New("function conversation operation context is incomplete")
	}
	client, err := r.provider(ctx)
	if err != nil {
		return nil, err
	}
	latest, err := client.GetLatestFunctionConversationOperation(authctx.ForwardUserContext(ctx), &funcoperation.GetLatestFunctionConversationOperationReq{Id: functionID, Conversation: conversation})
	if err != nil {
		return nil, fmt.Errorf("verify function conversation operation: %w", err)
	}
	operation := latest.GetOperation()
	if operation == nil || operation.GetId() != operationID || operation.GetFunctionId() != functionID || operation.GetSessionId() != sessionID || operation.GetConversation() != conversation {
		return nil, errors.New("function conversation operation is no longer active")
	}
	if operation.GetState() != "running" {
		return nil, fmt.Errorf("function conversation operation is %s and cannot receive a message", operation.GetState())
	}
	toolName, err := completionToolForConversation(conversation)
	if err != nil {
		return nil, err
	}
	callCtx := authctx.ForwardUserContext(context.WithoutCancel(ctx))
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{"type": "string", "description": "A concise description of the completed draft or generated application."},
		},
		"required": []string{"summary"},
	}
	tool := claudeagentsdk.NewMCPTool(toolName, completionToolDescription(conversation), schema, func(input map[string]any) (claudeagentsdk.MCPToolResult, error) {
		return r.submit(callCtx, client, operationID, functionID, sessionID, conversation, toolName, input)
	})
	server := claudeagentsdk.CreateSDKMCPServer(functionConversationMCPServerID, "1.0.0", []claudeagentsdk.MCPTool{tool})
	return &FunctionConversationSetup{OperationID: operationID, FunctionID: functionID, SessionID: sessionID, Conversation: conversation, ToolName: toolName, SDKMCPServers: map[string]claudeagentsdk.MCPServerConfig{functionConversationMCPServerID: server}}, nil
}

func (r *FunctionConversationRuntime) submit(ctx context.Context, client funcoperation.FuncOperationServiceClient, operationID, functionID, sessionID, conversation, toolName string, input map[string]any) (claudeagentsdk.MCPToolResult, error) {
	toolUseID, _ := input[functionConversationToolUseIDInputKey].(string)
	if strings.TrimSpace(toolUseID) == "" {
		return conversationToolError("completion tool use id is required"), nil
	}
	summary, _ := input["summary"].(string)
	resp, err := client.SubmitFunctionConversationProgress(ctx, &funcoperation.SubmitFunctionConversationProgressReq{OperationId: operationID, FunctionId: functionID, SessionId: sessionID, Conversation: conversation, ToolName: toolName, ToolUseId: toolUseID, Summary: summary})
	if err != nil {
		return conversationToolError(err.Error()), nil
	}
	result, _ := json.Marshal(map[string]any{"completed": true, "operationId": operationID, "alreadySubmitted": resp.GetAlreadySubmitted()})
	return claudeagentsdk.MCPToolResult{Content: []claudeagentsdk.MCPToolContent{{Type: "text", Text: string(result)}}}, nil
}

func (r *FunctionConversationRuntime) Finish(ctx context.Context, operationID, terminalStatus, terminalReason string) error {
	if r == nil || r.provider == nil || strings.TrimSpace(operationID) == "" {
		return nil
	}
	client, err := r.provider(ctx)
	if err != nil {
		return err
	}
	_, err = client.FinishFunctionConversationOperation(authctx.ForwardUserContext(ctx), &funcoperation.FinishFunctionConversationOperationReq{OperationId: operationID, TerminalStatus: terminalStatus, TerminalReason: terminalReason})
	return err
}

func (s *FunctionConversationSetup) IsCompletionTool(value string) bool {
	if s == nil {
		return false
	}
	value = strings.TrimSpace(value)
	return value == s.ToolName || strings.HasSuffix(value, "__"+s.ToolName)
}

func completionToolForConversation(conversation string) (string, error) {
	switch conversation {
	case "product":
		return "submit_product_document_draft", nil
	case "technical":
		return "submit_technical_document_draft", nil
	case "generation":
		return "submit_generated_app", nil
	default:
		return "", errors.New("unsupported function conversation")
	}
}

func completionToolDescription(conversation string) string {
	switch conversation {
	case "product":
		return "Call only after the complete product document draft has been written to the exact requested draft path. This marks the current business operation complete."
	case "technical":
		return "Call only after the complete technical document draft has been written to the exact requested draft path. This marks the current business operation complete."
	default:
		return "Call only after the generated application is complete, built, and verified in the requested app directory. This marks the current business operation complete."
	}
}

func conversationToolError(message string) claudeagentsdk.MCPToolResult {
	return claudeagentsdk.MCPToolResult{Content: []claudeagentsdk.MCPToolContent{{Type: "text", Text: message}}, IsError: true}
}
