package dao

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type AgentConfig struct {
	db *gorm.DB
}

func NewAgentConfig(db *gorm.DB) *AgentConfig {
	return &AgentConfig{db: db}
}

func (d *AgentConfig) Get(ctx context.Context) (*model.AgentSystemConfig, error) {
	var item model.AgentSystemConfig
	err := d.db.WithContext(ctx).Where("id = ?", model.AgentSystemConfigID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentConfig) Save(ctx context.Context, item *model.AgentSystemConfig) error {
	return d.db.WithContext(ctx).Save(item).Error
}

func (d *AgentConfig) Count(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.AgentSystemConfig{}).Count(&count).Error
	return count, err
}
