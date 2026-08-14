package logic

import (
	"encoding/json"
	"testing"
)

func TestExtractInvokeActionKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "empty payload", payload: `{}`, want: ""},
		{name: "action payload", payload: `{"action":"create"}`, want: "create"},
		{name: "trim action", payload: `{"action":" publish "}`, want: "publish"},
		{name: "invalid payload", payload: `{`, want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractInvokeActionKey(json.RawMessage(tt.payload)); got != tt.want {
				t.Fatalf("extractInvokeActionKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestActionDeclaredInManifest(t *testing.T) {
	t.Parallel()

	if actionDeclaredInManifest([]string{"book_create", "book_delete"}, "book_list") {
		t.Fatalf("undeclared action must not be authorized")
	}
	if !actionDeclaredInManifest([]string{"book_create", "book_delete"}, "book_delete") {
		t.Fatalf("book_delete should require strict action permission when declared")
	}
}
