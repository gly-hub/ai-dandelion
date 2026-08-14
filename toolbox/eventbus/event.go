package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Event is the transport-neutral envelope shared by business services.
// Payload is intentionally opaque so Redis Streams and Kafka can share it.
type Event struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	SchemaVersion int               `json:"schemaVersion"`
	Topic         string            `json:"topic"`
	Key           string            `json:"key"`
	Producer      string            `json:"producer"`
	OccurredAt    int64             `json:"occurredAt"`
	CorrelationID string            `json:"correlationId,omitempty"`
	CausationID   string            `json:"causationId,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Payload       []byte            `json:"payload"`
}

func NewEvent(typ, topic, key, producer string, payload any) (Event, error) {
	if strings.TrimSpace(typ) == "" || strings.TrimSpace(topic) == "" || strings.TrimSpace(producer) == "" {
		return Event{}, errors.New("event type, topic and producer are required")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{ID: newID(), Type: typ, SchemaVersion: 1, Topic: topic, Key: key, Producer: producer, OccurredAt: time.Now().UnixMilli(), Payload: data}, nil
}

type Handler func(context.Context, Event) error

type Subscription struct {
	Topic    string
	Group    string
	Consumer string
	Handler  Handler
}

type Bus interface {
	Publish(context.Context, Event) error
	Subscribe(context.Context, Subscription) error
	Close() error
}

func marshalEvent(event Event) ([]byte, error) { return json.Marshal(event) }

func unmarshalEvent(data []byte, event *Event) error { return json.Unmarshal(data, event) }

func newID() string {
	return uuid.NewString()
}
