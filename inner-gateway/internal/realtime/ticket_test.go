package realtime

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
)

func TestTicketManagerIssueConsumeOnce(t *testing.T) {
	m := NewTicketManager(time.Minute)
	want := authctx.User{ID: "u-1", Username: "Ada", RoleIDs: []string{"admin"}}
	ticket, expires, err := m.Issue(want)
	if err != nil || ticket == "" || expires != 60 {
		t.Fatalf("issue = %q, %d, %v", ticket, expires, err)
	}
	got, err := m.Consume(ticket)
	if err != nil || got.ID != want.ID || got.Username != want.Username {
		t.Fatalf("consume = %+v, %v", got, err)
	}
	if _, err := m.Consume(ticket); err == nil {
		t.Fatal("expected replayed ticket to be rejected")
	}
}

func TestTicketManagerRejectsEmptyAndExpired(t *testing.T) {
	m := NewTicketManager(time.Millisecond)
	if _, err := m.Consume(""); err == nil {
		t.Fatal("expected empty ticket to be rejected")
	}
	ticket, _, err := m.Issue(authctx.User{ID: "u-1"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := m.Consume(ticket); err == nil {
		t.Fatal("expected expired ticket to be rejected")
	}
}

func TestAllowedOrigin(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "same host", url: "https://api.example.com", want: true},
		{name: "dev frontend", url: "http://localhost:5173", want: true},
		{name: "current dev frontend", url: "http://localhost:5174", want: true},
		{name: "unknown host", url: "https://evil.example", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origin, err := url.Parse(tt.url)
			if err != nil {
				t.Fatal(err)
			}
			if got := allowedOrigin(origin, "api.example.com", "api.example.com"); got != tt.want {
				t.Fatalf("allowedOrigin(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestEnvelopeValidateCommand(t *testing.T) {
	if err := (Envelope{ProtocolVersion: 1, Type: "ping"}).ValidateCommand(); err != nil {
		t.Fatal(err)
	}
	if err := (Envelope{ProtocolVersion: 1, Type: "ai-agent.stream.start"}).ValidateCommand(); err == nil {
		t.Fatal("expected request id validation")
	}
	if err := (Envelope{ProtocolVersion: 2, Type: "ping"}).ValidateCommand(); err == nil {
		t.Fatal("expected protocol version validation")
	}
}

func TestAgentStreamPayloadPreservesSelectedTools(t *testing.T) {
	payload := []byte(`{"sessionId":"s-1","content":"use it","extra":[{"type":"skill","id":"json-table","name":"JSON table"}],"messageParts":[{"type":"skill","skillId":"json-table","label":"JSON table"}]}`)
	var decoded AgentStreamPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Extra) != 1 || decoded.Extra[0].GetId() != "json-table" || len(decoded.MessageParts) != 1 || decoded.MessageParts[0].GetSkillId() != "json-table" {
		t.Fatalf("decoded payload = %+v", decoded)
	}
}
