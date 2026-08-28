package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

type ClaudeRunner struct {
	config Config
}

func NewClaudeRunner(config Config) *ClaudeRunner {
	return &ClaudeRunner{config: config}
}

func (r *ClaudeRunner) Stream(ctx context.Context, sessionID string, prompt string, resume bool, streamOptions StreamOptions) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		client, err := r.newClient(ctx, sessionID, resume, streamOptions, func(event Event) bool {
			return sendEvent(ctx, events, errs, event)
		})
		if err != nil {
			errs <- err
			return
		}
		defer client.Close()

		if err := sendUserContent(ctx, client, sessionID, prompt, streamOptions.UserContent); err != nil {
			errs <- fmt.Errorf("send user message: %w", err)
			return
		}
		// The SDK sends permission decisions through its stdin control channel.
		// Client.Close closes the input after the terminal result; closing it here
		// causes AskUserQuestion control responses to fail with "Stream closed".

		state := newStreamState()
		for {
			msg, err := client.Next(ctx)
			if err == io.EOF {
				errs <- fmt.Errorf("claude client closed")
				return
			}
			if err != nil {
				errs <- fmt.Errorf("read claude stream: %w", err)
				return
			}

			for _, event := range state.eventsFromMessage(msg) {
				if !sendEvent(ctx, events, errs, event) {
					return
				}
			}
			if result, ok := msg.(*claudeagentsdk.ResultMessage); ok {
				if !sendEvent(ctx, events, errs, Event{Type: "done", AgentSessionID: result.SessionID, Done: true, TerminalStatus: terminalStatusFromResult(result), TerminalReason: terminalReasonFromResult(result)}) {
					return
				}
				return
			}
		}
	}()

	return events, errs
}

func terminalStatusFromResult(result *claudeagentsdk.ResultMessage) string {
	if result == nil {
		return "error"
	}
	combined := strings.ToLower(strings.Join([]string{result.TerminalReason, result.StopReason, result.Subtype}, " "))
	if strings.Contains(combined, "max_turn") || strings.Contains(combined, "turn_limit") || strings.Contains(combined, "turn limit") {
		return "max_turns"
	}
	if result.IsError {
		return "error"
	}
	return "normal"
}

func terminalReasonFromResult(result *claudeagentsdk.ResultMessage) string {
	if result == nil {
		return "agent returned no terminal result"
	}
	for _, value := range append([]string{result.TerminalReason, result.StopReason}, result.Errors...) {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func sendUserContent(ctx context.Context, client *claudeagentsdk.Client, sessionID, prompt string, content any) error {
	if content == nil {
		return client.SendUser(ctx, prompt, sessionID)
	}
	if sessionID == "" {
		sessionID = "default"
	}
	return client.Send(ctx, map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": content,
		},
		"parent_tool_use_id": nil,
		"session_id":         sessionID,
	})
}

func (r *ClaudeRunner) DeleteSession(_ context.Context, sessionID string) error {
	err := claudeagentsdk.DeleteSession(sessionID, r.config.CWD)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return nil
	}
	return fmt.Errorf("delete claude session: %w", err)
}

func (r *ClaudeRunner) newClient(
	ctx context.Context,
	sessionID string,
	resume bool,
	streamOptions StreamOptions,
	emit func(Event) bool,
) (*claudeagentsdk.Client, error) {
	client := claudeagentsdk.NewClient(r.options(ctx, sessionID, resume, streamOptions, emit))
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect claude client: %w", err)
	}
	return client, nil
}

func (r *ClaudeRunner) options(
	ctx context.Context,
	sessionID string,
	resume bool,
	streamOptions StreamOptions,
	emit func(Event) bool,
) claudeagentsdk.Options {
	options := claudeagentsdk.Options{
		CWD:                    r.config.CWD,
		SystemPrompt:           r.config.SystemPrompt,
		Model:                  r.config.Model,
		CLIPath:                r.config.CLIPath,
		MaxTurns:               r.config.MaxTurns,
		PermissionMode:         effectivePermissionMode(r.config.PermissionMode, streamOptions.AskUserQuestion != nil),
		ToolsPreset:            claudeagentsdk.ToolsPreset(r.config.ToolsPreset),
		AllowedTools:           r.config.AllowedTools,
		DisallowedTools:        r.config.DisallowedTools,
		Env:                    r.config.Env,
		SettingSources:         []string{"project"},
		IncludePartialMessages: true,
		IncludeHookEvents:      true,
		Thinking:               sdkThinkingConfig(r.config.Thinking),
		MaxThinkingTokens:      r.config.MaxThinkingTokens,
		Stderr: func(line string) {
			if r.config.Logger != nil {
				r.config.Logger.Warn("claude stderr", "session_id", sessionID, "line", line)
			}
		},
	}
	if resume {
		options.Resume = sessionID
	} else {
		options.SessionID = sessionID
	}
	if len(streamOptions.Skills) > 0 {
		options.Skills = streamOptions.Skills
	}
	if len(streamOptions.AddDirs) > 0 {
		options.AddDirs = append(options.AddDirs, streamOptions.AddDirs...)
	}
	if len(streamOptions.MCPServers) > 0 || len(streamOptions.SDKMCPServers) > 0 {
		options.MCPServers = sdkMCPServers(streamOptions.MCPServers)
		for id, server := range streamOptions.SDKMCPServers {
			options.MCPServers[id] = server
		}
	}
	if streamOptions.AskUserQuestion != nil || streamOptions.ToolPermission != nil {
		options.CanUseTool = func(req claudeagentsdk.ToolPermissionRequest) (claudeagentsdk.PermissionDecision, error) {
			if req.ToolName == "AskUserQuestion" {
				if streamOptions.AskUserQuestion == nil {
					return claudeagentsdk.PermissionDecision{
						Behavior: string(claudeagentsdk.PermissionBehaviorDeny),
						Message:  "AskUserQuestion is not configured",
					}, nil
				}
				updatedInput, err := runAskUserQuestion(ctx, streamOptions.AskUserQuestion, req.ToolUseID, req.ToolName, req.Input, emit)
				if err != nil {
					return claudeagentsdk.PermissionDecision{}, err
				}
				return claudeagentsdk.PermissionDecision{
					Behavior:     string(claudeagentsdk.PermissionBehaviorAllow),
					UpdatedInput: updatedInput,
				}, nil
			}

			if usesBypassPermissions(r.config.PermissionMode) && (streamOptions.ForceToolPermission == nil || !streamOptions.ForceToolPermission(req.ToolName)) {
				return claudeagentsdk.PermissionDecision{
					Behavior:     string(claudeagentsdk.PermissionBehaviorAllow),
					UpdatedInput: req.Input,
				}, nil
			}
			if streamOptions.ToolPermission == nil {
				return claudeagentsdk.PermissionDecision{
					Behavior: string(claudeagentsdk.PermissionBehaviorDeny),
					Message:  "interactive tool approval is not configured",
				}, nil
			}
			title := req.Title
			if title == "" {
				title = req.DisplayName
			}
			decision, err := streamOptions.ToolPermission(ctx, ToolPermissionRequest{
				ToolID:      req.ToolUseID,
				ToolName:    req.ToolName,
				Title:       title,
				Description: req.Description,
				Input:       req.Input,
			}, emit)
			if err != nil {
				return claudeagentsdk.PermissionDecision{}, err
			}
			if decision.Allow {
				updatedInput := req.Input
				if decision.UpdatedInput != nil {
					updatedInput = decision.UpdatedInput
				}
				return claudeagentsdk.PermissionDecision{
					Behavior:     string(claudeagentsdk.PermissionBehaviorAllow),
					UpdatedInput: updatedInput,
				}, nil
			}
			message := strings.TrimSpace(decision.Message)
			if message == "" {
				message = "user denied this tool call"
			}
			return claudeagentsdk.PermissionDecision{
				Behavior: string(claudeagentsdk.PermissionBehaviorDeny),
				Message:  message,
			}, nil
		}
	}
	return options
}

func usesBypassPermissions(permissionMode string) bool {
	return strings.EqualFold(strings.TrimSpace(permissionMode), string(claudeagentsdk.PermissionModeBypassPermissions))
}

func effectivePermissionMode(permissionMode string, needsAskUserQuestion bool) claudeagentsdk.PermissionMode {
	if needsAskUserQuestion && usesBypassPermissions(permissionMode) {
		// The Claude CLI bypasses CanUseTool in bypassPermissions mode. Switching
		// the transport to default lets this one callback pause AskUserQuestion;
		// the callback immediately allows every other tool, preserving bypass's
		// existing behavior for this application.
		return claudeagentsdk.PermissionModeDefault
	}
	return claudeagentsdk.PermissionMode(permissionMode)
}

func runAskUserQuestion(
	ctx context.Context,
	handler AskUserQuestionHandler,
	toolID string,
	toolName string,
	input map[string]any,
	emit func(Event) bool,
) (map[string]any, error) {
	return handler(ctx, AskUserQuestionRequest{ToolID: toolID, ToolName: toolName, Input: input}, emit)
}

func sdkThinkingConfig(config *ThinkingConfig) *claudeagentsdk.ThinkingConfig {
	if config == nil {
		return nil
	}
	mode := claudeagentsdk.ThinkingMode(config.Mode)
	switch mode {
	case claudeagentsdk.ThinkingAdaptive, claudeagentsdk.ThinkingEnabled, claudeagentsdk.ThinkingDisabled:
	default:
		mode = claudeagentsdk.ThinkingAdaptive
	}

	display := claudeagentsdk.ThinkingDisplay(config.Display)
	switch display {
	case claudeagentsdk.ThinkingDisplaySummarized, claudeagentsdk.ThinkingDisplayOmitted:
	default:
		display = claudeagentsdk.ThinkingDisplaySummarized
	}

	return &claudeagentsdk.ThinkingConfig{
		Mode:         mode,
		BudgetTokens: config.BudgetTokens,
		Display:      display,
	}
}

func sdkMCPServers(servers map[string]MCPServerConfig) map[string]claudeagentsdk.MCPServerConfig {
	resolved := make(map[string]claudeagentsdk.MCPServerConfig, len(servers))
	for id, server := range servers {
		switch server.Type {
		case "stdio":
			resolved[id] = claudeagentsdk.MCPStdioServerConfig{
				Type:    "stdio",
				Command: server.Command,
				Args:    append([]string(nil), server.Args...),
				Env:     copyStringMap(server.Env),
			}
		case "http":
			resolved[id] = claudeagentsdk.MCPHTTPServerConfig{
				Type:    "http",
				URL:     server.URL,
				Headers: copyStringMap(server.Headers),
			}
		case "sse":
			resolved[id] = claudeagentsdk.MCPSSEServerConfig{
				Type:    "sse",
				URL:     server.URL,
				Headers: copyStringMap(server.Headers),
			}
		}
	}
	return resolved
}

func copyStringMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	copied := make(map[string]string, len(items))
	for key, value := range items {
		copied[key] = value
	}
	return copied
}

func sendEvent(ctx context.Context, events chan<- Event, errs chan<- error, event Event) bool {
	select {
	case events <- event:
		return true
	case <-ctx.Done():
		errs <- ctx.Err()
		return false
	}
}
