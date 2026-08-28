package model

import "github.com/gly-hub/ai-dandelion/func-operation/boot"

func init() {
	boot.Register(&FunctionConversationOperation{}, &FunctionConversationProgressExecution{})
}

const (
	ConversationOperationStateRunning       = "running"
	ConversationOperationStateAwaitingUser  = "awaiting_user"
	ConversationOperationStateNeedsContinue = "needs_continue"
	ConversationOperationStateCompleted     = "completed"
	ConversationOperationStateBlocked       = "blocked"
	ConversationOperationStateCancelled     = "cancelled"
	ConversationOperationStateSuperseded    = "superseded"

	ConversationTerminalNormal    = "normal"
	ConversationTerminalMaxTurns  = "max_turns"
	ConversationTerminalCancelled = "cancelled"
	ConversationTerminalError     = "error"
	ConversationTerminalSubmitted = "submitted"
)

// FunctionConversationOperation is the persisted boundary of one business
// request. It deliberately does not store Agent tasks, because those are the
// native TaskCreate/TaskUpdate parts persisted with each message.
type FunctionConversationOperation struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID           string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:操作UUID"`
	FunctionID     string `gorm:"column:function_id;type:varchar(36);not null;index:idx_conversation_operation_scope,priority:1;comment:功能UUID"`
	SessionID      string `gorm:"column:session_id;type:varchar(36);not null;index:idx_conversation_operation_scope,priority:2;comment:Agent会话ID"`
	Conversation   string `gorm:"column:conversation;type:varchar(32);not null;index:idx_conversation_operation_scope,priority:3;comment:product/technical/generation"`
	UserID         string `gorm:"column:user_id;type:varchar(36);not null;index;comment:发起用户ID"`
	State          string `gorm:"column:state;type:varchar(32);not null;index;comment:操作状态"`
	BaselineJSON   string `gorm:"column:baseline_json;type:longtext;not null;default:'';comment:开始时版本快照"`
	TerminalStatus string `gorm:"column:terminal_status;type:varchar(32);not null;default:'';comment:流式终态"`
	TerminalReason string `gorm:"column:terminal_reason;type:varchar(1000);not null;default:'';comment:流式终态原因"`
	Outcome        string `gorm:"column:outcome;type:varchar(2000);not null;default:'';comment:业务完成摘要"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0;index;comment:创建时间"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0;index;comment:更新时间"`
	FinishedAt     int64  `gorm:"column:finished_at;type:bigint;not null;default:0;comment:本轮结束时间"`
}

func (FunctionConversationOperation) TableName() string {
	return "func_operation_conversation_operations"
}

func (FunctionConversationOperation) TableComment() string {
	return "功能会话业务操作表"
}

// FunctionConversationProgressExecution makes completion tool calls safe to
// retry when the Agent SDK redelivers the same tool_use_id.
type FunctionConversationProgressExecution struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID         string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:执行UUID"`
	OperationID  string `gorm:"column:operation_id;type:varchar(36);not null;index;comment:业务操作UUID"`
	FunctionID   string `gorm:"column:function_id;type:varchar(36);not null;index;comment:功能UUID"`
	SessionID    string `gorm:"column:session_id;type:varchar(36);not null;index;comment:Agent会话ID"`
	Conversation string `gorm:"column:conversation;type:varchar(32);not null;comment:会话类型"`
	ToolName     string `gorm:"column:tool_name;type:varchar(120);not null;comment:完成工具名"`
	ToolUseID    string `gorm:"column:tool_use_id;type:varchar(180);not null;uniqueIndex:uk_conversation_progress_execution;comment:Agent工具调用ID"`
	UserID       string `gorm:"column:user_id;type:varchar(36);not null;index;comment:用户ID"`
	Status       string `gorm:"column:status;type:varchar(32);not null;comment:succeeded/failed"`
	Outcome      string `gorm:"column:outcome;type:varchar(2000);not null;default:'';comment:提交摘要"`
	ResultJSON   string `gorm:"column:result_json;type:longtext;not null;default:'';comment:结果快照"`
	ErrorMessage string `gorm:"column:error_message;type:varchar(1000);not null;default:'';comment:错误信息"`
	CreatedAt    int64  `gorm:"column:created_at;type:bigint;not null;default:0;index;comment:创建时间"`
	UpdatedAt    int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (FunctionConversationProgressExecution) TableName() string {
	return "func_operation_function_progress_executions"
}

func (FunctionConversationProgressExecution) TableComment() string {
	return "功能会话完成工具执行表"
}
