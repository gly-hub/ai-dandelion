package model

import "github.com/gly-hub/ai-dandelion/ai-agent/boot"

func init() {
	boot.Register(&AgentModel{})
}

const (
	AgentModelStatusEnabled  = 1
	AgentModelStatusDisabled = 2
)

type AgentModel struct {
	ID                string `gorm:"column:id;type:varchar(36);primaryKey"`
	Name              string `gorm:"column:name;type:varchar(120);not null"`
	Model             string `gorm:"column:model;type:varchar(120);not null"`
	Type              string `gorm:"column:type;type:varchar(32);not null;default:'chat';index"`
	BaseURL           string `gorm:"column:base_url;type:varchar(255);not null;default:''"`
	AuthToken         string `gorm:"column:auth_token;type:varchar(255);not null;default:''"`
	SystemPrompt      string `gorm:"column:system_prompt;type:text"`
	PermissionMode    string `gorm:"column:permission_mode;type:varchar(64);not null;default:''"`
	MaxTurns          int    `gorm:"column:max_turns;type:int;not null;default:0"`
	ThinkMode         string `gorm:"column:think_mode;type:varchar(32);not null;default:''"`
	ThinkBudgetTokens int    `gorm:"column:think_budget_tokens;type:int;not null;default:0"`
	ThinkDisplay      string `gorm:"column:think_display;type:varchar(32);not null;default:''"`
	MaxThinkingTokens int    `gorm:"column:max_thinking_tokens;type:int;not null;default:0"`
	Status            int    `gorm:"column:status;type:tinyint(1);not null;default:1;index"`
	IsDefault         bool   `gorm:"column:is_default;type:tinyint(1);not null;default:0;index"`
	Sort              int    `gorm:"column:sort;type:int;not null;default:0"`
	Remark            string `gorm:"column:remark;type:varchar(255);not null;default:''"`
	CreatedAt         int64  `gorm:"column:created_at;type:bigint;not null;default:0"`
	UpdatedAt         int64  `gorm:"column:updated_at;type:bigint;not null;default:0"`
}

func (AgentModel) TableName() string {
	return "sys_agent_models"
}

func (AgentModel) TableComment() string {
	return "AI Agent 模型配置表"
}
