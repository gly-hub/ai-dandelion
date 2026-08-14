package logic

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/team-dandelion/ai-dandelion/toolbox/agent"
)

func TestAskUserQuestionBrokerReturnsOfficialUpdatedInput(t *testing.T) {
	broker := NewAskUserQuestionBroker()
	input := map[string]any{
		"questions": []any{map[string]any{
			"question":    "Which format should I use?",
			"header":      "Format",
			"multiSelect": false,
			"options": []any{
				map[string]any{"label": "Summary", "description": "Brief"},
				map[string]any{"label": "Detailed", "description": "Full"},
			},
		}},
	}
	updatedCh := make(chan map[string]any, 1)
	errCh := make(chan error, 1)
	go func() {
		updated, err := broker.Wait(context.Background(), "session-1", agent.AskUserQuestionRequest{
			ToolID:   "tool-1",
			ToolName: "AskUserQuestion",
			Input:    input,
		}, func(event agent.Event) bool {
			if event.Type != "ask_user_question" || event.ToolID != "tool-1" {
				t.Errorf("unexpected event: %#v", event)
			}
			return true
		})
		if err != nil {
			errCh <- err
			return
		}
		updatedCh <- updated
	}()

	deadline := time.Now().Add(time.Second)
	for {
		broker.mu.Lock()
		_, pending := broker.pending["tool-1"]
		broker.mu.Unlock()
		if pending {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("question was not registered")
		}
		time.Sleep(time.Millisecond)
	}

	answers, _ := json.Marshal(map[string]any{"Which format should I use?": "Summary"})
	if err := broker.Submit("session-1", "tool-1", string(answers), ""); err != nil {
		t.Fatalf("submit answer: %v", err)
	}
	select {
	case err := <-errCh:
		t.Fatalf("wait: %v", err)
	case updated := <-updatedCh:
		if updated["questions"] == nil {
			t.Fatal("updated input must preserve questions")
		}
		answers, ok := updated["answers"].(map[string]any)
		if !ok || answers["Which format should I use?"] != "Summary" {
			t.Fatalf("unexpected updated answers: %#v", updated)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for answer")
	}
}
