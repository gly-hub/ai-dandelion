package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
	"github.com/team-dandelion/ai-dandelion/toolbox/eventbus"
	"gorm.io/gorm"
)

type OutboxProcessor struct {
	outbox    *dao.FunctionOutbox
	functions *dao.Function
	menus     *FunctionMenuSync
	bus       eventbus.Bus
}

func NewOutboxProcessor(outbox *dao.FunctionOutbox, functions *dao.Function, menus *FunctionMenuSync, buses ...eventbus.Bus) *OutboxProcessor {
	var bus eventbus.Bus
	if len(buses) > 0 {
		bus = buses[0]
	}
	return &OutboxProcessor{outbox: outbox, functions: functions, menus: menus, bus: bus}
}

func (p *OutboxProcessor) ProcessReady(ctx context.Context, limit int) (int, error) {
	if p == nil || p.outbox == nil {
		return 0, errors.New("outbox processor is not configured")
	}
	now := nowUnixMicro()
	events, err := p.outbox.ClaimReady(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	for _, event := range events {
		if err := p.deliver(ctx, event); err != nil {
			backoff := int64(event.Attempts) * 5 * 1000000
			_ = p.outbox.MarkRetry(ctx, event.ID, now+backoff, err, now)
			continue
		}
		if err := p.outbox.MarkDone(ctx, event.ID, now); err != nil {
			return 0, err
		}
	}
	return len(events), nil
}

func (p *OutboxProcessor) deliver(ctx context.Context, event model.FunctionOutboxEvent) error {
	if handled, err := publishFunctionRealtimeEvent(ctx, p.bus, event); handled {
		return err
	}
	switch event.EventType {
	case "release.published":
		// This event type was recorded before release events were delivered to
		// the realtime bus. Existing rows are safe to acknowledge on upgrade.
		return nil
	case "menu.delete":
		return p.menus.Delete(ctx, event.FunctionID)
	case "menu.sync":
		function, err := p.functions.Get(ctx, event.FunctionID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var payload struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal([]byte(event.PayloadJSON), &payload)
		if strings.TrimSpace(payload.Status) == model.FunctionStatusPublished {
			_, err = p.menus.SyncPublished(ctx, function.UUID, function.Name, function.MenuParentID, nil)
			return err
		}
		return p.menus.Unpublish(ctx, function.UUID)
	default:
		return errors.New("unsupported outbox event type")
	}
}

func publishFunctionRealtimeEvent(ctx context.Context, bus eventbus.Bus, event model.FunctionOutboxEvent) (bool, error) {
	switch event.EventType {
	case FunctionVersionPublishedEventType:
		if bus == nil {
			return true, errors.New("function release realtime event bus is unavailable")
		}
		var payload functionVersionPublishedPayload
		if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
			return true, fmt.Errorf("decode function release event: %w", err)
		}
		payload.FunctionID = strings.TrimSpace(payload.FunctionID)
		if payload.FunctionID == "" {
			return true, errors.New("function release event functionId is required")
		}
		realtimeEvent, err := eventbus.NewEvent(FunctionVersionPublishedEventType, "realtime.events", payload.FunctionID, "func-operation", payload)
		if err != nil {
			return true, err
		}
		return true, bus.Publish(ctx, realtimeEvent)
	case FunctionUnpublishedEventType:
		if bus == nil {
			return true, errors.New("function release realtime event bus is unavailable")
		}
		var payload functionUnpublishedPayload
		if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
			return true, fmt.Errorf("decode function unpublish event: %w", err)
		}
		payload.FunctionID = strings.TrimSpace(payload.FunctionID)
		if payload.FunctionID == "" {
			return true, errors.New("function unpublish event functionId is required")
		}
		realtimeEvent, err := eventbus.NewEvent(FunctionUnpublishedEventType, "realtime.events", payload.FunctionID, "func-operation", payload)
		if err != nil {
			return true, err
		}
		return true, bus.Publish(ctx, realtimeEvent)
	default:
		return false, nil
	}
}
