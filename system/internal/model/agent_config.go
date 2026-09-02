package model

import "github.com/gly-hub/ai-dandelion/system/boot"

func init() {
	boot.Register(&AgentSystemConfig{})
}

const AgentSystemConfigID = "default"

type AgentSystemConfig struct {
	ID               string `gorm:"column:id;type:varchar(36);primaryKey"`
	SystemPrompt     string `gorm:"column:system_prompt;type:text;comment:系统提示词"`
	PermissionMode   string `gorm:"column:permission_mode;type:varchar(64);not null;default:'';comment:权限模式"`
	MaxTurns         int    `gorm:"column:max_turns;type:int;not null;default:0;comment:最大轮次"`
	ImageToolEnabled bool   `gorm:"column:image_tool_enabled;type:tinyint(1);not null;default:0;comment:启用图片生成工具"`
	ImageModelID     string `gorm:"column:image_model_id;type:varchar(36);not null;default:'';comment:图片模型ID"`
	CreatedAt        int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt        int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (AgentSystemConfig) TableName() string {
	return "sys_agent_config"
}

func (AgentSystemConfig) TableComment() string {
	return "AI Agent 系统配置表"
}
