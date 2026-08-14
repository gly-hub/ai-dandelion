package logic

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
	funcoperation "github.com/team-dandelion/ai-dandelion/proto/func-operation"
)

type OutboxManagementLogic struct {
	outbox     *dao.FunctionOutbox
	processor  *OutboxProcessor
	authorizer *FunctionAuthorizer
}

func NewOutboxManagementLogic(outbox *dao.FunctionOutbox, processor *OutboxProcessor, authorizer *FunctionAuthorizer) *OutboxManagementLogic {
	return &OutboxManagementLogic{outbox: outbox, processor: processor, authorizer: authorizer}
}
func (l *OutboxManagementLogic) List(ctx context.Context, limit int32) ([]*funcoperation.OutboxEvent, error) {
	if err := l.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	events, err := l.outbox.List(ctx, int(limit))
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.OutboxEvent, 0, len(events))
	for _, e := range events {
		out = append(out, &funcoperation.OutboxEvent{Id: e.UUID, FunctionId: e.FunctionID, EventType: e.EventType, Status: e.Status, Attempts: e.Attempts, NextAttemptAt: e.NextAttemptAt, LastError: e.LastError, CreatedAt: e.CreatedAt})
	}
	return out, nil
}
func (l *OutboxManagementLogic) Replay(ctx context.Context, limit int32) (int32, error) {
	if err := l.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return 0, err
	}
	if err := l.outbox.Requeue(ctx, nowUnixMicro()); err != nil {
		return 0, err
	}
	count, err := l.processor.ProcessReady(ctx, int(limit))
	return int32(count), err
}
