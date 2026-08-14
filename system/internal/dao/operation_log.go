package dao

import (
	"context"
	"strings"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type OperationLogListFilter struct {
	Module       string
	Action       string
	ResourceType string
	ResourceID   string
	OperatorID   string
	Keyword      string
	Page         int
	PageSize     int
}

type OperationLog struct {
	db *gorm.DB
}

func NewOperationLog(db *gorm.DB) *OperationLog {
	return &OperationLog{db: db}
}

func (d *OperationLog) Create(ctx context.Context, item *model.OperationLog) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *OperationLog) List(ctx context.Context, filter OperationLogListFilter) ([]model.OperationLog, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.OperationLog{})
	if filter.Module != "" {
		query = query.Where("module = ?", filter.Module)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		query = query.Where("resource_id = ?", filter.ResourceID)
	}
	if filter.OperatorID != "" {
		query = query.Where("operator_id = ?", filter.OperatorID)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("resource_name LIKE ? OR operator_name LIKE ? OR summary LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]model.OperationLog, 0)
	err := query.Order("created_at DESC").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&items).Error
	return items, total, err
}
