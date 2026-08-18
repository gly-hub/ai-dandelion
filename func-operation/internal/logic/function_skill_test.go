package logic

import (
	"encoding/json"
	"testing"
)

func TestFunctionSkillInvokePayloadUsesGeneratedAppDataEnvelope(t *testing.T) {
	raw, err := functionSkillInvokePayload("book_create", map[string]any{"title": "百年孤独", "stock": 5})
	if err != nil {
		t.Fatalf("functionSkillInvokePayload() error = %v", err)
	}
	var payload struct {
		Action string         `json:"action"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Action != "book_create" || payload.Data["title"] != "百年孤独" || payload.Data["stock"] != float64(5) {
		t.Fatalf("unexpected generated app payload: %s", raw)
	}
}
