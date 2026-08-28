package model

import "github.com/gly-hub/ai-dandelion/ai-agent/boot"

func init() {
	boot.Register(&Message{})
}

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// Message 对话消息表。
type Message struct {
	ID             string `gorm:"column:id;type:varchar(36);primaryKey"`
	SessionID      string `gorm:"column:session_id;type:varchar(36);not null;index:idx_messages_session_created,priority:1;comment:会话ID"`
	OperationID    string `gorm:"column:operation_id;type:varchar(36);not null;default:'';index;comment:功能会话操作UUID"`
	Role           string `gorm:"column:role;type:varchar(32);not null;comment:消息角色"`
	Content        string `gorm:"column:content;type:longtext;not null;comment:文本内容"`
	PartsJSON      string `gorm:"column:parts_json;type:longtext;not null;default:'';comment:结构化消息分片JSON"`
	TerminalStatus string `gorm:"column:terminal_status;type:varchar(32);not null;default:'';comment:Agent流终态"`
	TerminalReason string `gorm:"column:terminal_reason;type:varchar(1000);not null;default:'';comment:Agent流终态原因"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0;index:idx_messages_session_created,priority:2;comment:创建时间"`
}

func (Message) TableName() string {
	return "messages"
}

func (Message) TableComment() string {
	return "消息表"
}
