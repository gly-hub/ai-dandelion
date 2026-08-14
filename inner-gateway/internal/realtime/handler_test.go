package realtime

import (
	"encoding/json"
	"testing"

	"github.com/team-dandelion/ai-dandelion/toolbox/eventbus"
)

func TestPublishEventTargetsUserConnections(t *testing.T) {
	h := NewHandler(nil, nil)
	var gotA, gotB int
	h.registerClient("u-a", "a", func(Envelope) error { gotA++; return nil })
	h.registerClient("u-b", "b", func(Envelope) error { gotB++; return nil })
	payload, _ := json.Marshal(map[string]string{"value": "ok"})
	event := eventbus.Event{ID: "evt-1", Type: "system.notice", OccurredAt: 1, Payload: payload, Headers: map[string]string{"userId": "u-a"}}
	if err := h.publishEvent(event); err != nil {
		t.Fatal(err)
	}
	if gotA != 1 || gotB != 0 {
		t.Fatalf("targeted delivery = %d/%d", gotA, gotB)
	}

	if err := h.publishEvent(eventbus.Event{ID: "evt-2", Type: "system.broadcast", OccurredAt: 1, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	if gotA != 2 || gotB != 1 {
		t.Fatalf("broadcast delivery = %d/%d", gotA, gotB)
	}
	h.unregisterClient("u-a", "a")
	if len(h.clients) != 1 {
		t.Fatalf("clients after unregister = %d", len(h.clients))
	}
}

func TestPublishEventKeepsTargetedHistoryPrivate(t *testing.T) {
	h := NewHandler(nil, nil)
	event := eventbus.Event{ID: "evt-private", Type: "system.notice", OccurredAt: 1, Headers: map[string]string{"userId": "u-a"}}
	if err := h.publishEvent(event); err != nil {
		t.Fatal(err)
	}
	if len(h.history) != 1 || h.history[0].target != "u-a" {
		t.Fatalf("history = %+v", h.history)
	}
}
