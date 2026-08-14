package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStreams struct{ client redis.UniversalClient }

func NewRedisStreams(client redis.UniversalClient) (*RedisStreams, error) {
	if client == nil {
		return nil, errors.New("redis client is required")
	}
	return &RedisStreams{client: client}, nil
}

func (b *RedisStreams) Publish(ctx context.Context, event Event) error {
	if strings.TrimSpace(event.Topic) == "" {
		return errors.New("event topic is required")
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.client.XAdd(ctx, &redis.XAddArgs{Stream: event.Topic, Values: map[string]any{"event": string(data)}}).Err()
}

func (b *RedisStreams) Subscribe(ctx context.Context, sub Subscription) error {
	if strings.TrimSpace(sub.Topic) == "" || strings.TrimSpace(sub.Group) == "" || strings.TrimSpace(sub.Consumer) == "" || sub.Handler == nil {
		return errors.New("topic, group, consumer and handler are required")
	}
	// Each gateway instance uses its own group so broadcasts reach every
	// instance. Start at '$' to avoid replaying the full stream after restart.
	if err := b.client.XGroupCreateMkStream(ctx, sub.Topic, sub.Group, "$").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, cursor := range []string{"0", ">"} {
			block := time.Duration(0)
			if cursor == ">" {
				block = time.Second
			}
			streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: sub.Group, Consumer: sub.Consumer, Streams: []string{sub.Topic, cursor}, Count: 32, Block: block}).Result()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				continue
			}
			for _, stream := range streams {
				for _, item := range stream.Messages {
					raw, ok := item.Values["event"].(string)
					if !ok {
						_ = b.client.XAck(ctx, sub.Topic, sub.Group, item.ID).Err()
						continue
					}
					var event Event
					if err := json.Unmarshal([]byte(raw), &event); err != nil {
						_ = b.client.XAck(ctx, sub.Topic, sub.Group, item.ID).Err()
						continue
					}
					if err := sub.Handler(ctx, event); err == nil {
						_ = b.client.XAck(ctx, sub.Topic, sub.Group, item.ID).Err()
					}
				}
			}
		}
	}
}

func (b *RedisStreams) Close() error { return nil }
