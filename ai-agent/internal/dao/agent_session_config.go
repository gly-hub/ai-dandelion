package dao

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	"gorm.io/gorm"
)

type AgentSessionConfig struct {
	db *gorm.DB
}

func NewAgentSessionConfig(db *gorm.DB) *AgentSessionConfig {
	return &AgentSessionConfig{db: db}
}

func (d *AgentSessionConfig) GetEnabled(ctx context.Context, sessionType string) (*model.AgentSessionConfig, error) {
	var item model.AgentSessionConfig
	err := d.db.WithContext(ctx).
		Where("session_type = ? AND enabled = ?", sessionType, true).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}
