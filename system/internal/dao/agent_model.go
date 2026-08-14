package dao

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type AgentModel struct {
	db *gorm.DB
}

func NewAgentModel(db *gorm.DB) *AgentModel {
	return &AgentModel{db: db}
}

func (d *AgentModel) List(ctx context.Context) ([]model.AgentModel, error) {
	var items []model.AgentModel
	err := d.db.WithContext(ctx).Order("sort ASC, created_at ASC").Find(&items).Error
	return items, err
}

func (d *AgentModel) Get(ctx context.Context, id string) (*model.AgentModel, error) {
	var item model.AgentModel
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentModel) Create(ctx context.Context, item *model.AgentModel) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *AgentModel) Save(ctx context.Context, item *model.AgentModel) error {
	return d.db.WithContext(ctx).Save(item).Error
}

func (d *AgentModel) Delete(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Where("id = ?", id).Delete(&model.AgentModel{}).Error
}

func (d *AgentModel) ClearDefault(ctx context.Context, exceptID string) error {
	query := d.db.WithContext(ctx).Model(&model.AgentModel{}).Where("is_default = ?", true)
	if exceptID != "" {
		query = query.Where("id <> ?", exceptID)
	}
	return query.Update("is_default", false).Error
}

func (d *AgentModel) Count(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.AgentModel{}).Count(&count).Error
	return count, err
}
