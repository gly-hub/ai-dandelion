package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

const functionSkillMCPServerID = "function-skills"
const functionSkillApprovalInputKey = "__function_skill_approval"
const functionSkillToolUseIDInputKey = "__function_skill_tool_use_id"

type FunctionSkillClientProvider func(context.Context) (funcoperation.FuncOperationServiceClient, error)

type FunctionSkillRuntime struct {
	provider FunctionSkillClientProvider
}

type FunctionSkillSetup struct {
	SkillNames     []string
	AddDirs        []string
	SDKMCPServers  map[string]claudeagentsdk.MCPServerConfig
	GrantToken     string
	autoTools      map[string]bool
	protectedTools map[string]bool
	cleanup        func()
}

func NewFunctionSkillRuntime(provider FunctionSkillClientProvider) *FunctionSkillRuntime {
	return &FunctionSkillRuntime{provider: provider}
}

func (r *FunctionSkillRuntime) List(ctx context.Context, req *aiagent.ListFunctionSkillsReq) ([]*aiagent.AgentFunctionSkill, error) {
	client, err := r.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListFunctionSkills(authctx.ForwardUserContext(ctx), &funcoperation.ListFunctionSkillsReq{UserId: req.GetUserId()})
	if err != nil {
		return nil, err
	}
	out := make([]*aiagent.AgentFunctionSkill, 0, len(resp.GetSkills()))
	for _, item := range resp.GetSkills() {
		out = append(out, &aiagent.AgentFunctionSkill{Id: item.GetId(), FunctionId: item.GetFunctionId(), Name: item.GetName(), Description: item.GetDescription(), ToolPrefix: item.GetToolPrefix(), UpdatedAt: item.GetUpdatedAt()})
	}
	return out, nil
}

func (r *FunctionSkillRuntime) Prepare(ctx context.Context, userID, sessionID string, ids []string) (*FunctionSkillSetup, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if r == nil || r.provider == nil {
		return nil, errors.New("function skill runtime is not configured")
	}
	client, err := r.client(ctx)
	if err != nil {
		return nil, err
	}
	list, err := client.ListFunctionSkills(authctx.ForwardUserContext(ctx), &funcoperation.ListFunctionSkillsReq{UserId: userID})
	if err != nil {
		return nil, err
	}
	selected := make(map[string]*funcoperation.FunctionSkill, len(list.GetSkills()))
	for _, item := range list.GetSkills() {
		selected[item.GetId()] = item
	}
	ids = uniqueFunctionSkillIDs(ids)
	ids, items := selectAvailableFunctionSkills(ids, selected)
	if len(ids) == 0 {
		return nil, nil
	}
	grant, err := client.IssueFunctionSkillGrant(authctx.ForwardUserContext(ctx), &funcoperation.IssueFunctionSkillGrantReq{SessionId: sessionID, SkillIds: ids})
	if err != nil {
		return nil, err
	}
	toolResp, err := client.GetFunctionSkillTools(authctx.ForwardUserContext(ctx), &funcoperation.GetFunctionSkillToolsReq{GrantToken: grant.GetGrantToken()})
	if err != nil {
		return nil, fmt.Errorf("load function skill tools: %w", err)
	}
	if len(toolResp.GetTools()) == 0 {
		return nil, errors.New("no function skill tools are available for the current user; check the function menu and action permissions")
	}
	sdkServer, err := r.newSDKMCPServer(ctx, client, grant.GetGrantToken(), toolResp.GetTools())
	if err != nil {
		return nil, err
	}
	toolsByPrefix := make(map[string][]*funcoperation.FunctionSkillTool, len(items))
	for _, tool := range toolResp.GetTools() {
		prefix, _, ok := strings.Cut(tool.GetName(), "__")
		if !ok || prefix == "" {
			continue
		}
		toolsByPrefix[prefix] = append(toolsByPrefix[prefix], tool)
	}
	root, err := os.MkdirTemp("", "ai-dandelion-function-skills-")
	if err != nil {
		return nil, err
	}
	setup := &FunctionSkillSetup{GrantToken: grant.GetGrantToken(), autoTools: make(map[string]bool), protectedTools: make(map[string]bool), cleanup: func() { _ = os.RemoveAll(root) }}
	for _, item := range items {
		name := "function-" + safeFunctionSkillName(item.GetId()) + "-" + safeFunctionSkillName(item.GetReleaseId())
		skillDir := filepath.Join(root, ".claude", "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			setup.cleanup()
			return nil, err
		}
		markdown := functionSkillMarkdown(item, toolsByPrefix[item.GetToolPrefix()])
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(markdown), 0o644); err != nil {
			setup.cleanup()
			return nil, err
		}
		setup.SkillNames = append(setup.SkillNames, name)
		for _, operation := range item.GetOperations() {
			toolName := item.GetToolPrefix() + "__" + operation.GetKey()
			if operation.GetEffect() == "read" || (operation.GetEffect() == "create" && operation.GetAutoExecute()) {
				setup.autoTools[toolName] = true
			} else {
				setup.protectedTools[toolName] = true
			}
		}
	}
	sort.Strings(setup.SkillNames)
	setup.AddDirs = []string{root}
	setup.SDKMCPServers = map[string]claudeagentsdk.MCPServerConfig{functionSkillMCPServerID: sdkServer}
	return setup, nil
}

func (r *FunctionSkillRuntime) newSDKMCPServer(ctx context.Context, client funcoperation.FuncOperationServiceClient, grantToken string, source []*funcoperation.FunctionSkillTool) (claudeagentsdk.MCPServerConfig, error) {
	tools := make([]claudeagentsdk.MCPTool, 0, len(source))
	callCtx := authctx.ForwardUserContext(context.WithoutCancel(ctx))
	for _, sourceTool := range source {
		var schema any
		if err := json.Unmarshal([]byte(sourceTool.GetInputSchemaJson()), &schema); err != nil {
			return nil, fmt.Errorf("parse function skill schema for %q: %w", sourceTool.GetName(), err)
		}
		toolName := sourceTool.GetName()
		description := sourceTool.GetDescription()
		tools = append(tools, claudeagentsdk.NewMCPTool(toolName, description, schema, func(input map[string]any) (claudeagentsdk.MCPToolResult, error) {
			return executeFunctionSkillSDKTool(callCtx, client, grantToken, toolName, input)
		}))
	}
	return claudeagentsdk.CreateSDKMCPServer(functionSkillMCPServerID, "1.0.0", tools), nil
}

func executeFunctionSkillSDKTool(ctx context.Context, client funcoperation.FuncOperationServiceClient, grantToken, toolName string, input map[string]any) (claudeagentsdk.MCPToolResult, error) {
	input = copyFunctionSkillInput(input)
	toolUseID, _ := input[functionSkillToolUseIDInputKey].(string)
	approvalToken, _ := input[functionSkillApprovalInputKey].(string)
	delete(input, functionSkillToolUseIDInputKey)
	delete(input, functionSkillApprovalInputKey)
	if strings.TrimSpace(toolUseID) == "" {
		return functionSkillSDKToolError("function skill tool use id is required"), nil
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return functionSkillSDKToolError("function skill input is invalid"), nil
	}
	resp, err := client.ExecuteFunctionSkill(ctx, &funcoperation.ExecuteFunctionSkillReq{GrantToken: grantToken, ToolName: toolName, ToolUseId: toolUseID, InputJson: string(raw), ApprovalToken: approvalToken})
	if err != nil {
		return functionSkillSDKToolError(err.Error()), nil
	}
	if resp.GetIsError() {
		return functionSkillSDKToolError(firstNonEmpty(resp.GetErrorMessage(), "function skill execution failed")), nil
	}
	return claudeagentsdk.MCPToolResult{Content: []claudeagentsdk.MCPToolContent{{Type: "text", Text: firstNonEmpty(resp.GetResultJson(), "{}")}}}, nil
}

func functionSkillSDKToolError(message string) claudeagentsdk.MCPToolResult {
	return claudeagentsdk.MCPToolResult{Content: []claudeagentsdk.MCPToolContent{{Type: "text", Text: message}}, IsError: true}
}

func copyFunctionSkillInput(input map[string]any) map[string]any {
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func (r *FunctionSkillRuntime) CreateApproval(ctx context.Context, setup *FunctionSkillSetup, toolName, toolUseID string, input map[string]any) (string, error) {
	if r == nil || setup == nil {
		return "", errors.New("function skill setup is required")
	}
	canonical := setup.CanonicalToolName(toolName)
	if canonical == "" || !setup.protectedTools[canonical] {
		return "", errors.New("function skill approval is not available for this tool")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	client, err := r.client(ctx)
	if err != nil {
		return "", err
	}
	resp, err := client.CreateFunctionSkillApproval(authctx.ForwardUserContext(ctx), &funcoperation.CreateFunctionSkillApprovalReq{GrantToken: setup.GrantToken, ToolName: canonical, ToolUseId: toolUseID, InputJson: string(raw)})
	if err != nil {
		return "", err
	}
	return resp.GetApprovalToken(), nil
}

func (r *FunctionSkillRuntime) client(ctx context.Context) (funcoperation.FuncOperationServiceClient, error) {
	if r == nil || r.provider == nil {
		return nil, errors.New("function skill client is not configured")
	}
	return r.provider(ctx)
}

func (s *FunctionSkillSetup) CanonicalToolName(value string) string {
	value = strings.TrimSpace(value)
	if s == nil {
		return ""
	}
	if s.autoTools[value] || s.protectedTools[value] {
		return value
	}
	for tool := range s.autoTools {
		if strings.HasSuffix(value, "__"+tool) {
			return tool
		}
	}
	for tool := range s.protectedTools {
		if strings.HasSuffix(value, "__"+tool) {
			return tool
		}
	}
	return ""
}
func (s *FunctionSkillSetup) IsAutoTool(value string) bool {
	return s != nil && s.autoTools[s.CanonicalToolName(value)]
}
func (s *FunctionSkillSetup) IsProtectedTool(value string) bool {
	return s != nil && s.protectedTools[s.CanonicalToolName(value)]
}
func (s *FunctionSkillSetup) IsFunctionTool(value string) bool {
	return s != nil && s.CanonicalToolName(value) != ""
}
func (s *FunctionSkillSetup) Cleanup() {
	if s != nil && s.cleanup != nil {
		s.cleanup()
	}
}

func functionSkillMarkdown(item *funcoperation.FunctionSkill, availableTools []*funcoperation.FunctionSkillTool) string {
	var builder strings.Builder
	builder.WriteString("---\nname: function-")
	builder.WriteString(safeFunctionSkillName(item.GetId()))
	builder.WriteString("\ndescription: ")
	builder.WriteString(strings.TrimSpace(item.GetDescription()))
	builder.WriteString("\n---\n\n# ")
	builder.WriteString(item.GetName())
	builder.WriteString("\n\nThe `function-skills` SDK MCP server is attached locally to this conversation for the current turn. Claude displays each tool with the `mcp__function-skills__` prefix shown below. Do not inspect `.mcp.json` or run `claude mcp list`: those only show persistent developer configuration and do not include this session-scoped server.\n\nUse only the listed tools. Ask a focused follow-up when required fields are unavailable. Never invent fields or call a tool outside this skill.\n")
	for _, tool := range availableTools {
		builder.WriteString("\n- `")
		builder.WriteString("mcp__")
		builder.WriteString(functionSkillMCPServerID)
		builder.WriteString("__")
		builder.WriteString(tool.GetName())
		builder.WriteString("`: ")
		builder.WriteString(strings.TrimSpace(tool.GetDescription()))
		builder.WriteString("\n  Input schema: `")
		builder.WriteString(strings.TrimSpace(tool.GetInputSchemaJson()))
		builder.WriteString("`")
		if tool.GetAutoExecute() {
			builder.WriteString(" This create operation may execute immediately once its required data is complete.")
		}
	}
	return builder.String()
}
func safeFunctionSkillName(id string) string {
	return strings.NewReplacer("-", "", "_", "").Replace(strings.TrimSpace(id))
}
func uniqueFunctionSkillIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func selectAvailableFunctionSkills(ids []string, selected map[string]*funcoperation.FunctionSkill) ([]string, []*funcoperation.FunctionSkill) {
	availableIDs := make([]string, 0, len(ids))
	items := make([]*funcoperation.FunctionSkill, 0, len(ids))
	for _, id := range ids {
		item := selected[id]
		if item == nil {
			continue
		}
		availableIDs = append(availableIDs, id)
		items = append(items, item)
	}
	return availableIDs, items
}
