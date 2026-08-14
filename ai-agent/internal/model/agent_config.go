package model

import "github.com/team-dandelion/ai-dandelion/ai-agent/boot"

func init() {
	boot.Register(&AgentSystemConfig{})
}

const AgentSystemConfigID = "default"

type AgentSystemConfig struct {
	ID             string `gorm:"column:id;type:varchar(36);primaryKey"`
	SystemPrompt   string `gorm:"column:system_prompt;type:text"`
	PermissionMode string `gorm:"column:permission_mode;type:varchar(64);not null;default:''"`
	MaxTurns       int    `gorm:"column:max_turns;type:int;not null;default:0"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0"`
}

func (AgentSystemConfig) TableName() string {
	return "sys_agent_config"
}

func (AgentSystemConfig) TableComment() string {
	return "AI Agent 系统配置表"
}
