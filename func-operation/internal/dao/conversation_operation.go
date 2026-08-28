package dao

import (
	"context"
	"errors"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"gorm.io/gorm"
)

type FunctionConversationOperation struct{ db *gorm.DB }

func NewFunctionConversationOperation(db *gorm.DB) *FunctionConversationOperation {
	return &FunctionConversationOperation{db: db}
}

func (d *FunctionConversationOperation) Create(ctx context.Context, item *model.FunctionConversationOperation) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *FunctionConversationOperation) Get(ctx context.Context, operationID string) (*model.FunctionConversationOperation, error) {
	var item model.FunctionConversationOperation
	if err := d.db.WithContext(ctx).Where("uuid = ?", operationID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *FunctionConversationOperation) GetLatest(ctx context.Context, functionID, sessionID, conversation, userID string) (*model.FunctionConversationOperation, error) {
	var item model.FunctionConversationOperation
	err := d.db.WithContext(ctx).
		Where("function_id = ? AND session_id = ? AND conversation = ? AND user_id = ?", functionID, sessionID, conversation, userID).
		Order("created_at DESC").
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *FunctionConversationOperation) SupersedeActive(ctx context.Context, functionID, sessionID, conversation, userID string, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionConversationOperation{}).
		Where("function_id = ? AND session_id = ? AND conversation = ? AND user_id = ? AND state IN ?", functionID, sessionID, conversation, userID, []string{
			model.ConversationOperationStateRunning,
			model.ConversationOperationStateAwaitingUser,
			model.ConversationOperationStateNeedsContinue,
			model.ConversationOperationStateBlocked,
		}).
		Updates(map[string]any{"state": model.ConversationOperationStateSuperseded, "updated_at": now, "finished_at": now}).Error
}

func (d *FunctionConversationOperation) Resume(ctx context.Context, operationID string, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionConversationOperation{}).
		Where("uuid = ?", operationID).
		Updates(map[string]any{
			"state": model.ConversationOperationStateRunning, "terminal_status": "", "terminal_reason": "", "finished_at": int64(0), "updated_at": now,
		}).Error
}

func (d *FunctionConversationOperation) FinishIfActive(ctx context.Context, operationID, state, terminalStatus, terminalReason string, now int64) (*model.FunctionConversationOperation, error) {
	return d.finish(ctx, operationID, state, terminalStatus, terminalReason, "", now, false)
}

func (d *FunctionConversationOperation) Complete(ctx context.Context, operationID, outcome string, now int64) (*model.FunctionConversationOperation, error) {
	return d.finish(ctx, operationID, model.ConversationOperationStateCompleted, model.ConversationTerminalSubmitted, "", outcome, now, true)
}

func (d *FunctionConversationOperation) finish(ctx context.Context, operationID, state, terminalStatus, terminalReason, outcome string, now int64, setOutcome bool) (*model.FunctionConversationOperation, error) {
	updates := map[string]any{"state": state, "terminal_status": terminalStatus, "terminal_reason": terminalReason, "updated_at": now, "finished_at": now}
	if setOutcome {
		updates["outcome"] = outcome
	}
	result := d.db.WithContext(ctx).Model(&model.FunctionConversationOperation{}).
		Where("uuid = ? AND state IN ?", operationID, []string{model.ConversationOperationStateRunning, model.ConversationOperationStateAwaitingUser, model.ConversationOperationStateNeedsContinue, model.ConversationOperationStateBlocked}).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	item, err := d.Get(ctx, operationID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return item, err
}

type FunctionConversationProgressExecution struct{ db *gorm.DB }

func NewFunctionConversationProgressExecution(db *gorm.DB) *FunctionConversationProgressExecution {
	return &FunctionConversationProgressExecution{db: db}
}

func (d *FunctionConversationProgressExecution) GetByToolUseID(ctx context.Context, toolUseID string) (*model.FunctionConversationProgressExecution, error) {
	var item model.FunctionConversationProgressExecution
	if err := d.db.WithContext(ctx).Where("tool_use_id = ?", toolUseID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *FunctionConversationProgressExecution) Complete(ctx context.Context, execution *model.FunctionConversationProgressExecution, operation *model.FunctionConversationOperation) (*model.FunctionConversationProgressExecution, *model.FunctionConversationOperation, bool, error) {
	var cached *model.FunctionConversationProgressExecution
	var completed *model.FunctionConversationOperation
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.FunctionConversationProgressExecution
		if err := tx.Where("tool_use_id = ?", execution.ToolUseID).First(&existing).Error; err == nil {
			cached = &existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(execution).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.FunctionConversationOperation{}).
			Where("uuid = ?", operation.UUID).
			Updates(map[string]any{
				"state": model.ConversationOperationStateCompleted, "terminal_status": model.ConversationTerminalSubmitted,
				"terminal_reason": "", "outcome": execution.Outcome, "updated_at": execution.UpdatedAt, "finished_at": execution.UpdatedAt,
			}).Error; err != nil {
			return err
		}
		var item model.FunctionConversationOperation
		if err := tx.Where("uuid = ?", operation.UUID).First(&item).Error; err != nil {
			return err
		}
		completed = &item
		return nil
	})
	if err != nil {
		return nil, nil, false, err
	}
	if cached != nil {
		operationCopy, getErr := (&FunctionConversationOperation{db: d.db}).Get(ctx, cached.OperationID)
		return cached, operationCopy, true, getErr
	}
	return execution, completed, false, nil
}
