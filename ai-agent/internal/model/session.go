package model

import "github.com/team-dandelion/ai-dandelion/ai-agent/boot"

func init() {
	boot.Register(&Session{})
}

const (
	DefaultSessionTitle = "New chat"
	SessionTypeNormal   = 1
	SessionTypeFunction = 2
	SessionTypeChannel  = 3
)

// Session 对话会话表（与 session.proto 字段对应）。
type Session struct {
	ID             string `gorm:"column:id;type:varchar(36);primaryKey"`
	Title          string `gorm:"column:title;type:varchar(200);not null;default:'';comment:会话标题"`
	SessionType    int    `gorm:"column:session_type;type:tinyint(1);not null;default:1;comment:回话类型"` // 1-普通会话
	AgentSessionId string `gorm:"column:agent_session_id;type:varchar(200);not null;default:'';comment:sdk会话id"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (Session) TableName() string {
	return "sessions"
}

func (Session) TableComment() string {
	return "会话表"
}
