package logic

import (
	"testing"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/toolbox/agent"
)

func TestFlattenNavigationTargetsOnlyIncludesPageMenus(t *testing.T) {
	targets := flattenNavigationTargets([]*systemproto.Menu{
		{Id: "directory", MenuType: 1, Children: []*systemproto.Menu{
			{Id: "page", MenuType: 2, Name: "图书管理", Module: "func-operation", ViewKey: "books"},
			{Id: "button", MenuType: 3, Name: "删除"},
		}},
	})
	if len(targets) != 1 || targets[0]["id"] != "page" {
		t.Fatalf("unexpected navigation targets: %#v", targets)
	}
	if !navigationTargetMatches(targets[0], "图书") {
		t.Fatalf("expected target to match the page label")
	}
}

func TestNavigationToolResultProducesUIAction(t *testing.T) {
	action := uiActionFromToolResult(agent.Event{
		Type:       "tool_result",
		ToolName:   "mcp__navigation__navigate_to_target",
		ResultText: `{"status":"accepted","uiAction":{"action":"navigate","target":{"targetId":"page"}}}`,
	})
	if action == "" || action == "{}" {
		t.Fatalf("expected ui action, got %q", action)
	}
	if ignored := uiActionFromToolResult(agent.Event{Type: "tool_result", ToolName: "mcp__navigation__list_navigation_targets", ResultText: `{"targets":[]}`}); ignored != "" {
		t.Fatalf("list tool must not emit a ui action: %q", ignored)
	}
}

func TestNavigationToolResultUnwrapsMCPContentText(t *testing.T) {
	action := uiActionFromToolResult(agent.Event{
		Type:     "tool_result",
		ToolName: "mcp__navigation__navigate_to_target",
		ResultText: `[{
			"type":"text",
			"text":"{\"status\":\"accepted\",\"uiAction\":{\"action\":\"navigate\",\"target\":{\"targetId\":\"page\"}}}"
		}]`,
	})
	if action != `{"action":"navigate","target":{"targetId":"page"}}` {
		t.Fatalf("expected unwrapped ui action, got %q", action)
	}
}
