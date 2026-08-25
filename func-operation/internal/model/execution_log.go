package model

import "github.com/gly-hub/ai-dandelion/func-operation/boot"

func init() {
	boot.Register(&FunctionExecutionLog{})
}

// FunctionExecutionLog is the bounded, queryable audit record for a generated
// function invocation. Input and output are retained for troubleshooting and
// therefore must only be exposed to users who can edit the function.
type FunctionExecutionLog struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID;index:idx_func_exec_logs_function_created_id,priority:3;index:idx_func_exec_logs_function_type_created_id,priority:4"`
	UUID           string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:执行日志UUID"`
	FunctionID     string `gorm:"column:function_id;type:varchar(36);not null;index;comment:功能UUID;index:idx_func_exec_logs_function_created_id,priority:1;index:idx_func_exec_logs_function_request_id,priority:1;index:idx_func_exec_logs_function_type_created_id,priority:1"`
	AppID          string `gorm:"column:app_id;type:varchar(120);not null;index;comment:应用UUID"`
	UserID         string `gorm:"column:user_id;type:varchar(36);not null;default:'';index;comment:调用用户UUID"`
	RequestID      string `gorm:"column:request_id;type:varchar(64);not null;default:'';index;comment:外层请求UUID;index:idx_func_exec_logs_function_request_id,priority:2"`
	InvocationType string `gorm:"column:invocation_type;type:varchar(24);not null;index;comment:preview/published;index:idx_func_exec_logs_function_type_created_id,priority:2"`
	Version        string `gorm:"column:version;type:varchar(64);not null;default:'';comment:应用版本"`
	Status         string `gorm:"column:status;type:varchar(24);not null;index;comment:succeeded/failed"`
	Stage          string `gorm:"column:stage;type:varchar(64);not null;default:'';comment:失败阶段"`
	ErrorCode      string `gorm:"column:error_code;type:varchar(64);not null;default:'';comment:错误代码"`
	ErrorMessage   string `gorm:"column:error_message;type:varchar(2000);not null;default:'';comment:错误信息"`
	InputJSON      string `gorm:"column:input_json;type:longtext;not null;comment:调用输入"`
	OutputJSON     string `gorm:"column:output_json;type:longtext;not null;comment:调用输出"`
	LogsJSON       string `gorm:"column:logs_json;type:longtext;not null;comment:结构化执行日志"`
	LogsTruncated  bool   `gorm:"column:logs_truncated;not null;default:false;comment:日志是否截断"`
	DurationMS     int64  `gorm:"column:duration_ms;type:bigint;not null;default:0;comment:执行耗时毫秒"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;index;comment:创建时间;index:idx_func_exec_logs_function_created_id,priority:2;index:idx_func_exec_logs_function_type_created_id,priority:3"`
}

func (FunctionExecutionLog) TableName() string    { return "func_operation_function_execution_logs" }
func (FunctionExecutionLog) TableComment() string { return "生成功能WASM执行日志" }
