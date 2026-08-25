package agent

import (
	"context"
	"testing"

	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

func TestStreamStateBuildsToolLifecycleEvents(t *testing.T) {
	state := newStreamState()

	startEvents := state.eventsFromMessage(&claudeagentsdk.StreamEvent{
		Event: map[string]any{
			"type":  "content_block_start",
			"index": 0,
			"content_block": map[string]any{
				"type": "tool_use",
				"id":   "tool-1",
				"name": "search",
			},
		},
	})
	if len(startEvents) != 1 || startEvents[0].Type != "tool_start" {
		t.Fatalf("unexpected start events: %#v", startEvents)
	}

	deltaEvents := state.eventsFromMessage(&claudeagentsdk.StreamEvent{
		Event: map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{
				"type":         "input_json_delta",
				"partial_json": "{\"q\":\"golang\"}",
			},
		},
	})
	if len(deltaEvents) != 1 || deltaEvents[0].Type != "tool_delta" {
		t.Fatalf("unexpected delta events: %#v", deltaEvents)
	}

	stopEvents := state.eventsFromMessage(&claudeagentsdk.StreamEvent{
		Event: map[string]any{
			"type":  "content_block_stop",
			"index": 0,
		},
	})
	if len(stopEvents) != 1 || stopEvents[0].Type != "tool_stop" {
		t.Fatalf("unexpected stop events: %#v", stopEvents)
	}
	if stopEvents[0].ToolInput == "" {
		t.Fatalf("expected formatted tool input in stop event")
	}
}

func TestStreamStateAssistantMessageEmitsDeltaOnly(t *testing.T) {
	state := newStreamState()

	first := state.eventsFromMessage(&claudeagentsdk.AssistantMessage{
		Content: []claudeagentsdk.ContentBlock{
			claudeagentsdk.TextBlock{Text: "hello"},
		},
	})
	if len(first) != 1 || first[0].Text != "hello" {
		t.Fatalf("unexpected first assistant events: %#v", first)
	}

	second := state.eventsFromMessage(&claudeagentsdk.AssistantMessage{
		Content: []claudeagentsdk.ContentBlock{
			claudeagentsdk.TextBlock{Text: "hello world"},
		},
	})
	if len(second) != 1 || second[0].Text != " world" {
		t.Fatalf("unexpected second assistant events: %#v", second)
	}
}

func TestEventsFromUserIncludesToolResult(t *testing.T) {
	state := newStreamState()
	state.toolNames["tool-1"] = "mcp__navigation__navigate_to_target"
	events := state.eventsFromUser(&claudeagentsdk.UserMessage{
		Content: []claudeagentsdk.ContentBlock{
			claudeagentsdk.ToolResultBlock{
				ToolUseID: "tool-1",
				IsError:   true,
				Content:   map[string]any{"message": "failed"},
			},
		},
	})
	if len(events) != 1 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	if events[0].Type != "tool_result" || !events[0].IsError {
		t.Fatalf("unexpected tool result event: %#v", events[0])
	}
	if events[0].ToolName != "mcp__navigation__navigate_to_target" {
		t.Fatalf("expected originating tool name, got %#v", events[0])
	}
}

func TestClaudeRunnerOptionsAlignWithReferenceBehavior(t *testing.T) {
	runner := NewClaudeRunner(Config{
		CWD:     "/tmp/work",
		CLIPath: "/tmp/claude",
		Model:   "test-model",
	})

	firstOptions := runner.options(context.Background(), "local-session", false, StreamOptions{}, nil)
	if firstOptions.SessionID != "local-session" || firstOptions.Resume != "" {
		t.Fatalf("unexpected new-session options: %#v", firstOptions)
	}
	if firstOptions.CLIPath != "/tmp/claude" {
		t.Fatalf("CLIPath was not propagated: %q", firstOptions.CLIPath)
	}
	if !firstOptions.IncludePartialMessages || !firstOptions.IncludeHookEvents {
		t.Fatalf("expected stream and hook events to be included")
	}

	resumeOptions := runner.options(context.Background(), "agent-session", true, StreamOptions{}, nil)
	if resumeOptions.Resume != "agent-session" || resumeOptions.SessionID != "" {
		t.Fatalf("unexpected resume options: %#v", resumeOptions)
	}
}

func TestClaudeRunnerOptionsIncludeSDKMCPServer(t *testing.T) {
	runner := NewClaudeRunner(Config{})
	sdkServer := claudeagentsdk.CreateSDKMCPServer("function-skills", "1.0.0", nil)
	options := runner.options(context.Background(), "session", false, StreamOptions{
		SDKMCPServers: map[string]claudeagentsdk.MCPServerConfig{"function-skills": sdkServer},
	}, nil)
	if _, ok := options.MCPServers["function-skills"].(claudeagentsdk.SDKMCPServerConfig); !ok {
		t.Fatalf("SDK MCP server was not propagated: %#v", options.MCPServers)
	}
}

func TestClaudeRunnerCanUseToolWaitsOnlyForAskUserQuestion(t *testing.T) {
	expectedInput := map[string]any{"questions": []any{"original"}, "answers": map[string]any{"original": "answer"}}
	runner := NewClaudeRunner(Config{})
	options := runner.options(context.Background(), "session", false, StreamOptions{
		AskUserQuestion: func(_ context.Context, req AskUserQuestionRequest, _ func(Event) bool) (map[string]any, error) {
			if req.ToolID != "tool-1" || req.ToolName != "AskUserQuestion" {
				t.Fatalf("unexpected request: %#v", req)
			}
			return expectedInput, nil
		},
	}, nil)
	if options.CanUseTool == nil {
		t.Fatal("expected CanUseTool callback")
	}

	decision, err := options.CanUseTool(claudeagentsdk.ToolPermissionRequest{
		ToolName:  "AskUserQuestion",
		ToolUseID: "tool-1",
		Input:     map[string]any{"questions": []any{"original"}},
	})
	if err != nil || decision.Behavior != string(claudeagentsdk.PermissionBehaviorAllow) {
		t.Fatalf("unexpected AskUserQuestion decision: %#v, %v", decision, err)
	}
	if decision.UpdatedInput["answers"].(map[string]any)["original"] != "answer" {
		t.Fatalf("updated input was not returned: %#v", decision.UpdatedInput)
	}

	decision, err = options.CanUseTool(claudeagentsdk.ToolPermissionRequest{ToolName: "Bash"})
	if err != nil || decision.Behavior != string(claudeagentsdk.PermissionBehaviorDeny) {
		t.Fatalf("unexpected non-interactive decision: %#v, %v", decision, err)
	}
}

func TestClaudeRunnerBypassPermissionsUsesCanUseToolForAskUserQuestion(t *testing.T) {
	expectedInput := map[string]any{"questions": []any{"original"}, "answers": map[string]any{"original": "answer"}}
	runner := NewClaudeRunner(Config{PermissionMode: "bypassPermissions"})
	options := runner.options(context.Background(), "session", false, StreamOptions{
		AskUserQuestion: func(_ context.Context, req AskUserQuestionRequest, _ func(Event) bool) (map[string]any, error) {
			if req.ToolID != "tool-1" || req.ToolName != "AskUserQuestion" {
				t.Fatalf("unexpected request: %#v", req)
			}
			return expectedInput, nil
		},
	}, nil)
	if options.PermissionMode != claudeagentsdk.PermissionModeDefault {
		t.Fatalf("expected default transport permission mode, got %q", options.PermissionMode)
	}
	if options.CanUseTool == nil {
		t.Fatal("expected CanUseTool callback")
	}

	decision, err := options.CanUseTool(claudeagentsdk.ToolPermissionRequest{
		ToolName:  "AskUserQuestion",
		ToolUseID: "tool-1",
		Input:     map[string]any{"questions": []any{"original"}},
	})
	if err != nil {
		t.Fatalf("AskUserQuestion callback failed: %v", err)
	}
	if decision.Behavior != string(claudeagentsdk.PermissionBehaviorAllow) || decision.UpdatedInput["answers"].(map[string]any)["original"] != "answer" {
		t.Fatalf("unexpected AskUserQuestion decision: %#v", decision)
	}

	decision, err = options.CanUseTool(claudeagentsdk.ToolPermissionRequest{ToolName: "Bash", Input: map[string]any{"command": "pwd"}})
	if err != nil || decision.Behavior != string(claudeagentsdk.PermissionBehaviorAllow) || decision.UpdatedInput["command"] != "pwd" {
		t.Fatalf("bypass behavior for other tools changed: %#v, %v", decision, err)
	}
}
