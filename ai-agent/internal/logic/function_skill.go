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

	"github.com/gly-hub/ai-dandelion/ai-agent/config"
	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/agent"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
)

const functionSkillMCPServerID = "function-skills"

type FunctionSkillClientProvider func(context.Context) (funcoperation.FuncOperationServiceClient, error)

type FunctionSkillRuntime struct {
	provider FunctionSkillClientProvider
	mcpURL   string
}

type FunctionSkillSetup struct {
	SkillNames     []string
	AddDirs        []string
	MCPServers     map[string]agent.MCPServerConfig
	GrantToken     string
	autoTools      map[string]bool
	protectedTools map[string]bool
	cleanup        func()
}

func NewFunctionSkillRuntime(provider FunctionSkillClientProvider, cfg config.AgentConfig) *FunctionSkillRuntime {
	return &FunctionSkillRuntime{provider: provider, mcpURL: strings.TrimSpace(cfg.FunctionSkillMCPURL)}
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
	if r.mcpURL == "" {
		return nil, errors.New("function skill MCP URL is not configured")
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
	items := make([]*funcoperation.FunctionSkill, 0, len(ids))
	for _, id := range ids {
		item := selected[id]
		if item == nil {
			return nil, fmt.Errorf("function skill %q is unavailable", id)
		}
		items = append(items, item)
	}
	grant, err := client.IssueFunctionSkillGrant(authctx.ForwardUserContext(ctx), &funcoperation.IssueFunctionSkillGrantReq{SessionId: sessionID, SkillIds: ids})
	if err != nil {
		return nil, err
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
		markdown := functionSkillMarkdown(item)
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
	setup.MCPServers = map[string]agent.MCPServerConfig{functionSkillMCPServerID: {Type: "http", URL: r.mcpURL, Headers: map[string]string{"Authorization": "Bearer " + setup.GrantToken}}}
	return setup, nil
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
func (s *FunctionSkillSetup) Cleanup() {
	if s != nil && s.cleanup != nil {
		s.cleanup()
	}
}

func functionSkillMarkdown(item *funcoperation.FunctionSkill) string {
	var builder strings.Builder
	builder.WriteString("---\nname: function-")
	builder.WriteString(safeFunctionSkillName(item.GetId()))
	builder.WriteString("\ndescription: ")
	builder.WriteString(strings.TrimSpace(item.GetDescription()))
	builder.WriteString("\n---\n\n# ")
	builder.WriteString(item.GetName())
	builder.WriteString("\n\nUse only the MCP tools declared below. Ask a focused follow-up when required fields are unavailable. Never invent fields or call a tool outside this skill.\n")
	for _, operation := range item.GetOperations() {
		builder.WriteString("\n- `")
		builder.WriteString(item.GetToolPrefix() + "__" + operation.GetKey())
		builder.WriteString("`: ")
		builder.WriteString(firstNonEmpty(operation.GetDescription(), operation.GetEffect()))
		if operation.GetAutoExecute() {
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
