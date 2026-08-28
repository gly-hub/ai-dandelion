package logic

import (
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
)

func TestConversationStateForTerminal(t *testing.T) {
	tests := []struct {
		terminal string
		state    string
	}{
		{model.ConversationTerminalNormal, model.ConversationOperationStateAwaitingUser},
		{model.ConversationTerminalMaxTurns, model.ConversationOperationStateNeedsContinue},
		{model.ConversationTerminalCancelled, model.ConversationOperationStateCancelled},
		{model.ConversationTerminalError, model.ConversationOperationStateBlocked},
	}
	for _, test := range tests {
		if actual := conversationStateForTerminal(test.terminal); actual != test.state {
			t.Fatalf("terminal %q: got %q, want %q", test.terminal, actual, test.state)
		}
	}
}

func TestProgressToolMatchesConversation(t *testing.T) {
	tests := []struct {
		tool         string
		conversation string
		matches      bool
	}{
		{conversationProgressProductTool, functionConversationProduct, true},
		{conversationProgressTechnicalTool, functionConversationTechnical, true},
		{conversationProgressGenerationTool, functionConversationGeneration, true},
		{conversationProgressGenerationTool, functionConversationProduct, false},
		{conversationProgressProductTool, functionConversationTechnical, false},
	}
	for _, test := range tests {
		if actual := progressToolMatchesConversation(test.tool, test.conversation); actual != test.matches {
			t.Fatalf("tool %q for %q: got %v, want %v", test.tool, test.conversation, actual, test.matches)
		}
	}
}
