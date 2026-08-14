package dao

import (
	"context"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	"gorm.io/gorm"
)

type AgentModel struct {
	db *gorm.DB
}

func NewAgentModel(db *gorm.DB) *AgentModel {
	return &AgentModel{db: db}
}

func (d *AgentModel) ListEnabled(ctx context.Context) ([]model.AgentModel, error) {
	var items []model.AgentModel
	err := d.db.WithContext(ctx).
		Where("status = ?", model.AgentModelStatusEnabled).
		Order("sort ASC, created_at ASC").
		Find(&items).Error
	return items, err
}

func (d *AgentModel) GetEnabled(ctx context.Context, id string) (*model.AgentModel, error) {
	var item model.AgentModel
	err := d.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, model.AgentModelStatusEnabled).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentModel) GetDefaultEnabled(ctx context.Context) (*model.AgentModel, error) {
	var item model.AgentModel
	err := d.db.WithContext(ctx).
		Where("status = ? AND is_default = ?", model.AgentModelStatusEnabled, true).
		Order("sort ASC, created_at ASC").
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentModel) Count(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.AgentModel{}).Count(&count).Error
	return count, err
}

func (d *AgentModel) Create(ctx context.Context, item *model.AgentModel) error {
	return d.db.WithContext(ctx).Create(item).Error
}
