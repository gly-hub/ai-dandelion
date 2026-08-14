package dao

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type AgentSessionConfig struct {
	db *gorm.DB
}

func NewAgentSessionConfig(db *gorm.DB) *AgentSessionConfig {
	return &AgentSessionConfig{db: db}
}

func (d *AgentSessionConfig) List(ctx context.Context) ([]model.AgentSessionConfig, error) {
	var items []model.AgentSessionConfig
	err := d.db.WithContext(ctx).Order("created_at ASC").Find(&items).Error
	return items, err
}

func (d *AgentSessionConfig) Get(ctx context.Context, sessionType string) (*model.AgentSessionConfig, error) {
	var item model.AgentSessionConfig
	err := d.db.WithContext(ctx).Where("session_type = ?", sessionType).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentSessionConfig) Save(ctx context.Context, item *model.AgentSessionConfig) error {
	return d.db.WithContext(ctx).Save(item).Error
}
