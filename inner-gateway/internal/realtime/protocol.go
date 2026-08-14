package realtime

import (
	"encoding/json"
	"errors"
	"strings"

	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
)

type Envelope struct {
	ProtocolVersion int             `json:"protocolVersion"`
	Type            string          `json:"type"`
	RequestID       string          `json:"requestId,omitempty"`
	EventID         string          `json:"eventId,omitempty"`
	Timestamp       int64           `json:"timestamp"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

type AgentStreamPayload struct {
	SessionID              string                  `json:"sessionId"`
	Content                string                  `json:"content"`
	ModelID                string                  `json:"modelId,omitempty"`
	AgentSessionConfigType string                  `json:"agentSessionConfigType,omitempty"`
	SystemPrompt           string                  `json:"systemPrompt,omitempty"`
	PermissionMode         string                  `json:"permissionMode,omitempty"`
	MaxTurns               int32                   `json:"maxTurns,omitempty"`
	Extra                  []*aiagent.MessageExtra `json:"extra,omitempty"`
	UserID                 string                  `json:"userId,omitempty"`
	MessageParts           []*aiagent.MessagePart  `json:"messageParts,omitempty"`
}

type AskUserPayload struct {
	SessionID   string `json:"sessionId"`
	ToolID      string `json:"toolId"`
	AnswersJSON string `json:"answersJson"`
	Response    string `json:"response,omitempty"`
}
type ToolPermissionPayload struct {
	SessionID string `json:"sessionId"`
	ToolID    string `json:"toolId"`
	Allow     bool   `json:"allow"`
	Message   string `json:"message,omitempty"`
}

func (e Envelope) ValidateCommand() error {
	if e.ProtocolVersion != 1 {
		return errors.New("unsupported realtime protocol version")
	}
	if strings.TrimSpace(e.Type) == "" {
		return errors.New("realtime command type is required")
	}
	if strings.ContainsAny(e.Type, " \t\r\n") {
		return errors.New("realtime command type is invalid")
	}
	if strings.Contains(e.Type, ".") && strings.TrimSpace(e.RequestID) == "" {
		return errors.New("requestId is required for namespaced commands")
	}
	return nil
}
