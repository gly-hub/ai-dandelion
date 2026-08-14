package model

import "github.com/team-dandelion/ai-dandelion/ai-agent/boot"

func init() {
	boot.Register(&SessionReference{})
}

const (
	SessionReferenceTypeSkill = "skill"
	SessionReferenceTypeMCP   = "mcp"
)

// SessionReference 会话引用表，记录会话使用过的技能和 MCP。
type SessionReference struct {
	ID        string `gorm:"column:id;type:varchar(36);primaryKey"`
	SessionID string `gorm:"column:session_id;type:varchar(36);not null;default:'';uniqueIndex:uk_session_ref;index;comment:会话ID"`
	RefType   string `gorm:"column:ref_type;type:varchar(20);not null;default:'';uniqueIndex:uk_session_ref;comment:引用类型"`
	RefID     string `gorm:"column:ref_id;type:varchar(200);not null;default:'';uniqueIndex:uk_session_ref;comment:引用ID"`
	Name      string `gorm:"column:name;type:varchar(200);not null;default:'';comment:引用名称"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (SessionReference) TableName() string {
	return "session_references"
}

func (SessionReference) TableComment() string {
	return "会话引用表"
}
