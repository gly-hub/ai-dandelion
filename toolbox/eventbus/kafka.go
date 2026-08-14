package eventbus

import (
	"context"
	"errors"
)

// KafkaClient is the small transport contract needed by KafkaBus. Keeping the
// SDK-specific producer/consumer code behind this interface lets services use
// Bus without importing a Kafka library or changing when Redis is replaced.
type KafkaClient interface {
	Publish(context.Context, string, []byte, string) error
	Subscribe(context.Context, string, string, string, func(context.Context, []byte) error) error
	Close() error
}

type KafkaBus struct{ client KafkaClient }

func NewKafkaBus(client KafkaClient) (*KafkaBus, error) {
	if client == nil {
		return nil, errors.New("kafka client is required")
	}
	return &KafkaBus{client: client}, nil
}

func (b *KafkaBus) Publish(ctx context.Context, event Event) error {
	if b == nil || b.client == nil {
		return errors.New("kafka bus is not configured")
	}
	if event.Topic == "" {
		return errors.New("event topic is required")
	}
	payload, err := marshalEvent(event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, event.Topic, payload, event.Key)
}

func (b *KafkaBus) Subscribe(ctx context.Context, sub Subscription) error {
	if b == nil || b.client == nil {
		return errors.New("kafka bus is not configured")
	}
	if sub.Topic == "" || sub.Group == "" || sub.Consumer == "" || sub.Handler == nil {
		return errors.New("topic, group, consumer and handler are required")
	}
	return b.client.Subscribe(ctx, sub.Topic, sub.Group, sub.Consumer, func(handlerCtx context.Context, payload []byte) error {
		var event Event
		if err := unmarshalEvent(payload, &event); err != nil {
			return err
		}
		return sub.Handler(handlerCtx, event)
	})
}

func (b *KafkaBus) Close() error {
	if b == nil || b.client == nil {
		return nil
	}
	return b.client.Close()
}
