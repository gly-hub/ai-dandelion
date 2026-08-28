package agent

import (
	"testing"

	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

func TestTerminalStatusFromResult(t *testing.T) {
	tests := []struct {
		name   string
		result *claudeagentsdk.ResultMessage
		want   string
	}{
		{name: "normal", result: &claudeagentsdk.ResultMessage{}, want: "normal"},
		{name: "max turns", result: &claudeagentsdk.ResultMessage{TerminalReason: "max_turns"}, want: "max_turns"},
		{name: "turn limit", result: &claudeagentsdk.ResultMessage{StopReason: "turn limit reached"}, want: "max_turns"},
		{name: "error", result: &claudeagentsdk.ResultMessage{IsError: true}, want: "error"},
	}
	for _, test := range tests {
		if actual := terminalStatusFromResult(test.result); actual != test.want {
			t.Fatalf("%s: got %q, want %q", test.name, actual, test.want)
		}
	}
}
