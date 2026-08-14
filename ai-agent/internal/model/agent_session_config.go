package model

import "github.com/team-dandelion/ai-dandelion/ai-agent/boot"

func init() {
	boot.Register(&AgentSessionConfig{})
}

type AgentSessionConfig struct {
	SessionType    string `gorm:"column:session_type;type:varchar(64);primaryKey"`
	Name           string `gorm:"column:name;type:varchar(120);not null;default:''"`
	Description    string `gorm:"column:description;type:varchar(255);not null;default:''"`
	SystemPrompt   string `gorm:"column:system_prompt;type:text"`
	PermissionMode string `gorm:"column:permission_mode;type:varchar(64);not null;default:''"`
	MaxTurns       int    `gorm:"column:max_turns;type:int;not null;default:0"`
	ModelID        string `gorm:"column:model_id;type:varchar(36);not null;default:''"`
	Enabled        bool   `gorm:"column:enabled;type:tinyint(1);not null;default:1"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0"`
}

func (AgentSessionConfig) TableName() string {
	return "sys_agent_session_configs"
}

func (AgentSessionConfig) TableComment() string {
	return "AI Agent 会话配置表"
}
