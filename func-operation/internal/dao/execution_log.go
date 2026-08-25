package dao

import (
	"context"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"gorm.io/gorm"
)

type FunctionExecutionLog struct{ db *gorm.DB }

type ExecutionLogFilter struct {
	Limit          int
	Page           int
	Query          string
	RequestID      string
	Status         string
	InvocationType string
	StartTime      int64
	EndTime        int64
}

func NewFunctionExecutionLog(db *gorm.DB) *FunctionExecutionLog {
	return &FunctionExecutionLog{db: db}
}

func (d *FunctionExecutionLog) Create(ctx context.Context, item *model.FunctionExecutionLog) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *FunctionExecutionLog) ListByFunctionID(ctx context.Context, functionID string, filter ExecutionLogFilter) ([]model.FunctionExecutionLog, int64, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	var items []model.FunctionExecutionLog
	db := d.db.WithContext(ctx).Where("function_id = ?", functionID)
	if value := strings.TrimSpace(filter.Status); value != "" {
		db = db.Where("status = ?", value)
	}
	if value := strings.TrimSpace(filter.InvocationType); value != "" {
		db = db.Where("invocation_type = ?", value)
	}
	if value := strings.TrimSpace(filter.RequestID); value != "" {
		db = db.Where("request_id = ?", value)
	}
	if filter.StartTime > 0 {
		db = db.Where("created_at >= ?", filter.StartTime)
	}
	if filter.EndTime > 0 {
		db = db.Where("created_at <= ?", filter.EndTime)
	}
	if value := strings.TrimSpace(filter.Query); value != "" {
		pattern := "%" + value + "%"
		// Query only diagnostic content, never request or response payloads.
		db = db.Where("logs_json LIKE ? OR error_message LIKE ? OR error_code LIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := db.Model(&model.FunctionExecutionLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("created_at DESC, id DESC").Offset((filter.Page - 1) * filter.Limit).Limit(filter.Limit).Find(&items).Error
	return items, total, err
}

func (d *FunctionExecutionLog) GetByFunctionID(ctx context.Context, functionID, id string) (*model.FunctionExecutionLog, error) {
	var item model.FunctionExecutionLog
	if err := d.db.WithContext(ctx).Where("function_id = ? AND uuid = ?", functionID, id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
