package eventbus

import "testing"

func TestNewEventValidationAndPayload(t *testing.T) {
	if _, err := NewEvent("", "topic", "", "service", nil); err == nil {
		t.Fatal("expected type validation")
	}
	event, err := NewEvent("system.notice", "realtime.events", "u-1", "system", map[string]string{"ok": "yes"})
	if err != nil || event.ID == "" || event.SchemaVersion != 1 || len(event.Payload) == 0 {
		t.Fatalf("event = %+v, err = %v", event, err)
	}
}
