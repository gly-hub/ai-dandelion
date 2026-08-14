package model

import "github.com/gly-hub/ai-dandelion/system/boot"

func init() {
	boot.Register(&AgentSessionConfig{})
}

const (
	AgentSessionConfigTypeFuncProduct    = "func_product"
	AgentSessionConfigTypeFuncTechnical  = "func_technical"
	AgentSessionConfigTypeFuncGeneration = "func_generation"
)

type AgentSessionConfig struct {
	SessionType    string `gorm:"column:session_type;type:varchar(64);primaryKey;comment:会话类型"`
	Name           string `gorm:"column:name;type:varchar(120);not null;default:'';comment:展示名称"`
	Description    string `gorm:"column:description;type:varchar(255);not null;default:'';comment:说明"`
	SystemPrompt   string `gorm:"column:system_prompt;type:text;comment:系统提示词"`
	PermissionMode string `gorm:"column:permission_mode;type:varchar(64);not null;default:'';comment:权限模式"`
	MaxTurns       int    `gorm:"column:max_turns;type:int;not null;default:0;comment:最大轮次"`
	ModelID        string `gorm:"column:model_id;type:varchar(36);not null;default:'';comment:模型配置 ID"`
	Enabled        bool   `gorm:"column:enabled;type:tinyint(1);not null;default:1;comment:是否启用"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (AgentSessionConfig) TableName() string {
	return "sys_agent_session_configs"
}

func (AgentSessionConfig) TableComment() string {
	return "AI Agent 会话配置表"
}
