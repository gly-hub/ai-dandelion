package logic

import (
	"context"
	"errors"
	"testing"

	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
)

func TestFunctionLogicDefersAiAgentConnectionUntilRPC(t *testing.T) {
	calls := 0
	logic := NewFunctionLogic(nil, nil, nil, nil, nil, func(context.Context) (aiagent.AiAgentServiceClient, error) {
		calls++
		return nil, errors.New("ai-agent is unavailable")
	}, nil, nil, nil)

	if calls != 0 {
		t.Fatalf("provider calls during construction = %d, want 0", calls)
	}
	if _, _, err := logic.ensureAiAgentSession(context.Background(), "session-1", "title"); err == nil {
		t.Fatal("ensure session should return the deferred connection error")
	}
	if calls != 1 {
		t.Fatalf("provider calls after RPC = %d, want 1", calls)
	}
}
