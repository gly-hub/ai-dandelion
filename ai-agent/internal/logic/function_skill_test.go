package logic

import (
	"context"
	"strings"
	"testing"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/agent"
	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

func TestFunctionSkillSetupToolPolicy(t *testing.T) {
	setup := &FunctionSkillSetup{
		autoTools:      map[string]bool{"books__create_book": true, "books__list_books": true},
		protectedTools: map[string]bool{"books__delete_book": true},
	}
	if !setup.IsAutoTool("mcp__function-skills__books__create_book") {
		t.Fatal("auto tool should be recognized through MCP namespace")
	}
	if !setup.IsProtectedTool("books__delete_book") {
		t.Fatal("delete tool should require confirmation")
	}
	if setup.IsAutoTool("books__delete_book") {
		t.Fatal("protected tool must not auto execute")
	}
}

func TestFunctionSkillMarkdownUsesPublishedToolNames(t *testing.T) {
	markdown := functionSkillMarkdown(&funcoperation.FunctionSkill{
		Id: "skill-1", Name: "图书管理", ToolPrefix: "books",
		Operations: []*funcoperation.FunctionSkillOperation{{Key: "create_book", Effect: "create", AutoExecute: true}},
	}, []*funcoperation.FunctionSkillTool{{
		Name: "books__create_book", Description: "新增图书", InputSchemaJson: `{"type":"object","required":["title"]}`,
		AutoExecute: true,
	}})
	if markdown == "" || !contains(markdown, "mcp__function-skills__books__create_book") {
		t.Fatalf("tool markdown does not describe published tool: %q", markdown)
	}
	if !contains(markdown, "MCP server is attached") || !contains(markdown, `"title"`) {
		t.Fatalf("tool markdown must describe the dynamic MCP server and schema: %q", markdown)
	}
}

func TestFunctionSkillRuntimeCreatesSDKMCPTool(t *testing.T) {
	runtime := &FunctionSkillRuntime{}
	server, err := runtime.newSDKMCPServer(context.Background(), nil, "grant", []*funcoperation.FunctionSkillTool{{
		Name: "books__create_book", Description: "新增图书", InputSchemaJson: `{"type":"object","required":["title"]}`,
	}})
	if err != nil {
		t.Fatalf("newSDKMCPServer() error = %v", err)
	}
	config, ok := server.(claudeagentsdk.SDKMCPServerConfig)
	if !ok || config.Instance == nil {
		t.Fatalf("server = %#v, want SDK MCP server", server)
	}
	config.Instance.HandleMessage(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	response := config.Instance.HandleMessage(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}})
	result, _ := response["result"].(map[string]any)
	tools, _ := result["tools"].([]map[string]any)
	if len(tools) != 1 || tools[0]["name"] != "books__create_book" {
		t.Fatalf("tools/list = %#v, want books__create_book", response)
	}
}

func TestSelectAvailableFunctionSkillsSkipsStaleReferences(t *testing.T) {
	available := &funcoperation.FunctionSkill{Id: "available", Name: "可用功能"}
	ids, items := selectAvailableFunctionSkills(
		[]string{"stale", "available"},
		map[string]*funcoperation.FunctionSkill{"available": available},
	)
	if len(ids) != 1 || ids[0] != "available" || len(items) != 1 || items[0] != available {
		t.Fatalf("filtered skills = %#v, %#v; want only available skill", ids, items)
	}
}

func TestFunctionSkillAutoToolInjectsToolUseID(t *testing.T) {
	setup := &FunctionSkillSetup{autoTools: map[string]bool{"books__create_book": true}, protectedTools: map[string]bool{}}
	handler := (&MessageLogic{}).functionSkillToolPermissionHandler("session", setup, nil)
	decision, err := handler(context.Background(), agent.ToolPermissionRequest{ToolID: "tool-use-1", ToolName: "mcp__function-skills__books__create_book", Input: map[string]any{"title": "百年孤独"}}, nil)
	if err != nil || !decision.Allow {
		t.Fatalf("decision = %#v, err = %v", decision, err)
	}
	if decision.UpdatedInput[functionSkillToolUseIDInputKey] != "tool-use-1" {
		t.Fatalf("updated input = %#v, missing tool use id", decision.UpdatedInput)
	}
}

func contains(value, needle string) bool {
	return strings.Contains(value, needle)
}
