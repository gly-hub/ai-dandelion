package model

import "github.com/gly-hub/ai-dandelion/ai-agent/boot"

func init() {
	boot.Register(&AgentBot{})
	boot.Register(&AgentBotChannel{})
	boot.Register(&AgentBotCapability{})
}

const (
	AgentBotStatusEnabled  = 1
	AgentBotStatusDisabled = 2

	AgentBotChannelStatusEnabled  = 1
	AgentBotChannelStatusDisabled = 2

	AgentBotCapabilityEnabled  = 1
	AgentBotCapabilityDisabled = 2

	AgentBotCapabilityTypeSkill = "skill"
	AgentBotCapabilityTypeMCP   = "mcp"
)

type AgentBot struct {
	ID             string `gorm:"column:id;type:varchar(36);primaryKey"`
	Name           string `gorm:"column:name;type:varchar(120);not null;default:'';comment:机器人名称"`
	Code           string `gorm:"column:code;type:varchar(80);not null;default:'';uniqueIndex;comment:机器人编码"`
	Status         int    `gorm:"column:status;type:tinyint(1);not null;default:1;index;comment:状态 1启用 2禁用"`
	Description    string `gorm:"column:description;type:varchar(255);not null;default:'';comment:说明"`
	BusinessScene  string `gorm:"column:business_scene;type:varchar(120);not null;default:'';comment:业务场景"`
	WelcomeMessage string `gorm:"column:welcome_message;type:varchar(500);not null;default:'';comment:默认欢迎语"`
	ModelID        string `gorm:"column:model_id;type:varchar(36);not null;default:'';comment:模型配置 ID"`
	SystemPrompt   string `gorm:"column:system_prompt;type:text;comment:系统提示词"`
	PermissionMode string `gorm:"column:permission_mode;type:varchar(64);not null;default:'';comment:权限模式"`
	MaxTurns       int    `gorm:"column:max_turns;type:int;not null;default:0;comment:最大轮次"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (AgentBot) TableName() string {
	return "sys_agent_bots"
}

func (AgentBot) TableComment() string {
	return "智能机器人配置表"
}

type AgentBotChannel struct {
	ID            string `gorm:"column:id;type:varchar(36);primaryKey"`
	BotID         string `gorm:"column:bot_id;type:varchar(36);not null;default:'';index;comment:机器人 ID"`
	Channel       string `gorm:"column:channel;type:varchar(32);not null;default:'';index;comment:渠道类型"`
	Name          string `gorm:"column:name;type:varchar(120);not null;default:'';comment:渠道名称"`
	Status        int    `gorm:"column:status;type:tinyint(1);not null;default:1;index;comment:状态 1启用 2禁用"`
	ExternalBotID string `gorm:"column:external_bot_id;type:varchar(200);not null;default:'';comment:外部平台机器人 ID"`
	Secret        string `gorm:"column:secret;type:varchar(500);not null;default:'';comment:渠道密钥"`
	EndpointURL   string `gorm:"column:endpoint_url;type:varchar(255);not null;default:'';comment:渠道连接地址"`
	ConfigJSON    string `gorm:"column:config_json;type:longtext;not null;default:'';comment:渠道扩展配置 JSON"`
	CreatedAt     int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt     int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (AgentBotChannel) TableName() string {
	return "sys_agent_bot_channels"
}

func (AgentBotChannel) TableComment() string {
	return "智能机器人渠道接入表"
}

type AgentBotCapability struct {
	ID             string `gorm:"column:id;type:varchar(36);primaryKey"`
	BotID          string `gorm:"column:bot_id;type:varchar(36);not null;default:'';uniqueIndex:uk_bot_capability;index;comment:机器人 ID"`
	CapabilityType string `gorm:"column:capability_type;type:varchar(32);not null;default:'';uniqueIndex:uk_bot_capability;comment:能力类型"`
	CapabilityID   string `gorm:"column:capability_id;type:varchar(200);not null;default:'';uniqueIndex:uk_bot_capability;comment:能力 ID"`
	Name           string `gorm:"column:name;type:varchar(200);not null;default:'';comment:能力名称"`
	Enabled        int    `gorm:"column:enabled;type:tinyint(1);not null;default:1;comment:是否启用"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (AgentBotCapability) TableName() string {
	return "sys_agent_bot_capabilities"
}

func (AgentBotCapability) TableComment() string {
	return "智能机器人能力范围表"
}
