package logic

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"github.com/team-dandelion/ai-dandelion/toolbox/agent"
)

type agentBotRunRequest struct {
	SessionID      string
	Content        string
	AgentSessionID string
	EngineConfig   AgentEngineRunConfig
}

type agentBotStreamEvent struct {
	Event agent.Event
	Done  bool
}

func (r *AgentBotRuntime) streamAgentBotMessage(
	ctx context.Context,
	req *agentBotRunRequest,
	send func(agentBotStreamEvent) error,
) error {
	if r == nil || r.messageLogic == nil || r.agentEngine == nil {
		return errAgentRunnerNotConfigured
	}
	sessionID, err := requireSessionID(req.SessionID)
	if err != nil {
		return err
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return errContentRequired
	}
	userParts := []*aiagent.MessagePart{{Type: "text", Text: content}}
	if _, err := r.messageLogic.addMessage(ctx, sessionID, model.RoleUser, content, userParts); err != nil {
		return err
	}

	agentSessionID := strings.TrimSpace(req.AgentSessionID)
	resume := agentSessionID != ""
	if agentSessionID == "" {
		agentSessionID = uuid.NewString()
	}

	events, errs, err := r.agentEngine.Stream(ctx, agentSessionID, content, resume, req.EngineConfig)
	if err != nil {
		return err
	}

	var answer strings.Builder
	parts := newResponseParts()
	for events != nil || errs != nil {
		select {
		case event, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if event.Type == "text_delta" && event.Text != "" {
				answer.WriteString(event.Text)
			}
			if shouldSendAgentBotEvent(event) {
				if err := send(agentBotStreamEvent{Event: event}); err != nil {
					return err
				}
			}
			parts.apply(event)
			if event.Done {
				message, err := r.messageLogic.addMessage(ctx, sessionID, model.RoleAssistant, answer.String(), parts.parts())
				if err != nil {
					return err
				}
				if event.AgentSessionID != "" {
					agentSessionID = event.AgentSessionID
				}
				if err := r.messageLogic.sessionDao.UpdateAgentSession(ctx, sessionID, agentSessionID, message.CreatedAt); err != nil {
					return err
				}
				return send(agentBotStreamEvent{Done: true})
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func shouldSendAgentBotEvent(event agent.Event) bool {
	switch event.Type {
	case "text_delta":
		return event.Text != ""
	case "thinking_start", "thinking_delta", "thinking_stop",
		"tool_start", "tool_delta", "tool_stop", "tool_result":
		return true
	default:
		return false
	}
}
