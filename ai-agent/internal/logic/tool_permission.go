package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/gly-hub/ai-dandelion/toolbox/agent"
)

var errToolPermissionNotPending = errors.New("tool permission request is no longer pending")

type toolPermissionPending struct {
	sessionID  string
	decisionCh chan agent.ToolPermissionDecision
}

// ToolPermissionBroker owns live tool permission callbacks for this process.
// The Claude execution remains paused until Submit delivers a decision.
type ToolPermissionBroker struct {
	mu      sync.Mutex
	pending map[string]*toolPermissionPending
}

func NewToolPermissionBroker() *ToolPermissionBroker {
	return &ToolPermissionBroker{pending: make(map[string]*toolPermissionPending)}
}

func (b *ToolPermissionBroker) Wait(
	ctx context.Context,
	sessionID string,
	req agent.ToolPermissionRequest,
	emit func(agent.Event) bool,
) (agent.ToolPermissionDecision, error) {
	if b == nil {
		return agent.ToolPermissionDecision{}, errors.New("tool permission broker is not configured")
	}
	if strings.TrimSpace(req.ToolID) == "" {
		return agent.ToolPermissionDecision{}, errors.New("tool permission id is required")
	}

	pending := &toolPermissionPending{
		sessionID:  sessionID,
		decisionCh: make(chan agent.ToolPermissionDecision, 1),
	}
	b.mu.Lock()
	if _, exists := b.pending[req.ToolID]; exists {
		b.mu.Unlock()
		return agent.ToolPermissionDecision{}, errors.New("tool permission request is already pending")
	}
	b.pending[req.ToolID] = pending
	b.mu.Unlock()
	defer b.remove(req.ToolID)

	input, err := json.Marshal(req.Input)
	if err != nil {
		return agent.ToolPermissionDecision{}, err
	}
	if !emit(agent.Event{
		Type:            "tool_permission_request",
		ToolID:          req.ToolID,
		ToolName:        req.ToolName,
		ToolTitle:       req.Title,
		ToolDescription: req.Description,
		ToolInput:       string(input),
	}) {
		return agent.ToolPermissionDecision{}, ctx.Err()
	}

	select {
	case decision := <-pending.decisionCh:
		return decision, nil
	case <-ctx.Done():
		return agent.ToolPermissionDecision{}, ctx.Err()
	}
}

func (b *ToolPermissionBroker) Submit(sessionID string, toolID string, allow bool, message string) error {
	if b == nil {
		return errors.New("tool permission broker is not configured")
	}
	b.mu.Lock()
	pending := b.pending[toolID]
	if pending == nil || pending.sessionID != sessionID {
		b.mu.Unlock()
		return errToolPermissionNotPending
	}
	delete(b.pending, toolID)
	b.mu.Unlock()

	pending.decisionCh <- agent.ToolPermissionDecision{
		Allow:   allow,
		Message: strings.TrimSpace(message),
	}
	return nil
}

func (b *ToolPermissionBroker) remove(toolID string) {
	b.mu.Lock()
	delete(b.pending, toolID)
	b.mu.Unlock()
}
