package dao

import (
	"context"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
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
