package agent

import (
	"context"
	"log/slog"
)

type Config struct {
	CWD               string
	SystemPrompt      string
	Model             string
	CLIPath           string
	PermissionMode    string
	ToolsPreset       string
	AllowedTools      []string
	DisallowedTools   []string
	Env               map[string]string
	MaxTurns          int
	Thinking          *ThinkingConfig
	MaxThinkingTokens int
	Logger            *slog.Logger
}

type ThinkingConfig struct {
	Mode         string
	BudgetTokens int
	Display      string
}

type Event struct {
	Type            string `json:"type"`
	Text            string `json:"text,omitempty"`
	ToolID          string `json:"toolId,omitempty"`
	ToolName        string `json:"toolName,omitempty"`
	ToolTitle       string `json:"toolTitle,omitempty"`
	ToolDescription string `json:"toolDescription,omitempty"`
	ToolInput       string `json:"toolInput,omitempty"`
	IsError         bool   `json:"isError,omitempty"`
	ResultText      string `json:"resultText,omitempty"`
	AgentSessionID  string `json:"agentSessionId,omitempty"`
	Done            bool   `json:"done,omitempty"`
}

// AskUserQuestionRequest is the native Claude AskUserQuestion tool request.
// Input is the original tool input and must be returned with the user's answers.
type AskUserQuestionRequest struct {
	ToolID   string
	ToolName string
	Input    map[string]any
}

// AskUserQuestionHandler blocks until the application has collected a response.
// emit publishes the waiting state to the active client stream after registration.
type AskUserQuestionHandler func(
	ctx context.Context,
	req AskUserQuestionRequest,
	emit func(Event) bool,
) (map[string]any, error)

// ToolPermissionRequest describes a tool call that needs the user's approval.
type ToolPermissionRequest struct {
	ToolID      string
	ToolName    string
	Title       string
	Description string
	Input       map[string]any
}

// ToolPermissionDecision is returned after the user approves or declines a tool call.
type ToolPermissionDecision struct {
	Allow        bool
	Message      string
	UpdatedInput map[string]any
}

// ToolPermissionHandler blocks until the application receives an approval decision.
type ToolPermissionHandler func(
	ctx context.Context,
	req ToolPermissionRequest,
	emit func(Event) bool,
) (ToolPermissionDecision, error)

type StreamOptions struct {
	Skills          []string
	AddDirs         []string
	MCPServers      map[string]MCPServerConfig
	AskUserQuestion AskUserQuestionHandler
	ToolPermission  ToolPermissionHandler
	ForceToolPermission func(string) bool
	UserContent     any
}

type MCPServerConfig struct {
	Type    string
	Command string
	Args    []string
	Env     map[string]string
	URL     string
	Headers map[string]string
}

type Runner interface {
	Stream(ctx context.Context, sessionID string, prompt string, resume bool, options StreamOptions) (<-chan Event, <-chan error)
	DeleteSession(ctx context.Context, sessionID string) error
}
