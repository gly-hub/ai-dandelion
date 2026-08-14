package model

import "github.com/team-dandelion/ai-dandelion/system/boot"

func init() {
	boot.Register(&AgentModel{})
}

const (
	AgentModelStatusEnabled  = 1
	AgentModelStatusDisabled = 2
)

type AgentModel struct {
	ID                string `gorm:"column:id;type:varchar(36);primaryKey"`
	Name              string `gorm:"column:name;type:varchar(120);not null;comment:展示名称"`
	Model             string `gorm:"column:model;type:varchar(120);not null;comment:SDK 模型标识"`
	BaseURL           string `gorm:"column:base_url;type:varchar(255);not null;default:'';comment:API Base URL"`
	AuthToken         string `gorm:"column:auth_token;type:varchar(255);not null;default:'';comment:鉴权 Token"`
	SystemPrompt      string `gorm:"column:system_prompt;type:text;comment:系统提示词"`
	PermissionMode    string `gorm:"column:permission_mode;type:varchar(64);not null;default:'';comment:权限模式"`
	MaxTurns          int    `gorm:"column:max_turns;type:int;not null;default:0;comment:最大轮次"`
	ThinkMode         string `gorm:"column:think_mode;type:varchar(32);not null;default:'';comment:思考模式"`
	ThinkBudgetTokens int    `gorm:"column:think_budget_tokens;type:int;not null;default:0;comment:思考预算"`
	ThinkDisplay      string `gorm:"column:think_display;type:varchar(32);not null;default:'';comment:思考展示"`
	MaxThinkingTokens int    `gorm:"column:max_thinking_tokens;type:int;not null;default:0;comment:最大思考 Token"`
	Status            int    `gorm:"column:status;type:tinyint(1);not null;default:1;index;comment:状态 1启用 2禁用"`
	IsDefault         bool   `gorm:"column:is_default;type:tinyint(1);not null;default:0;index;comment:是否默认"`
	Sort              int    `gorm:"column:sort;type:int;not null;default:0;comment:排序"`
	Remark            string `gorm:"column:remark;type:varchar(255);not null;default:'';comment:备注"`
	CreatedAt         int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt         int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (AgentModel) TableName() string {
	return "sys_agent_models"
}

func (AgentModel) TableComment() string {
	return "AI Agent 模型配置表"
}
