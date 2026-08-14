package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/team-dandelion/ai-dandelion/toolbox/agent"
)

func TestToolPermissionBrokerWaitAndSubmit(t *testing.T) {
	broker := NewToolPermissionBroker()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	emitted := make(chan agent.Event, 1)
	result := make(chan agent.ToolPermissionDecision, 1)
	failure := make(chan error, 1)
	go func() {
		decision, err := broker.Wait(ctx, "session-1", agent.ToolPermissionRequest{
			ToolID:      "tool-1",
			ToolName:    "Bash",
			Title:       "Run command",
			Description: "Runs a local command.",
			Input:       map[string]any{"command": "pwd"},
		}, func(event agent.Event) bool {
			emitted <- event
			return true
		})
		if err != nil {
			failure <- err
			return
		}
		result <- decision
	}()

	event := <-emitted
	if event.Type != "tool_permission_request" || event.ToolID != "tool-1" || event.ToolInput == "" {
		t.Fatalf("unexpected permission event: %#v", event)
	}
	if err := broker.Submit("other-session", "tool-1", true, ""); !errors.Is(err, errToolPermissionNotPending) {
		t.Fatalf("mismatched session error = %v, want %v", err, errToolPermissionNotPending)
	}
	if err := broker.Submit("session-1", "tool-1", true, ""); err != nil {
		t.Fatalf("submit permission: %v", err)
	}

	select {
	case err := <-failure:
		t.Fatalf("wait permission: %v", err)
	case decision := <-result:
		if !decision.Allow {
			t.Fatal("decision should allow")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for permission decision")
	}
}
