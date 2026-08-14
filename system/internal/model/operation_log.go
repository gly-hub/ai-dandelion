package model

import "github.com/team-dandelion/ai-dandelion/system/boot"

func init() {
	boot.Register(&OperationLog{})
}

type OperationLog struct {
	ID           string `gorm:"column:id;type:varchar(36);primaryKey"`
	Module       string `gorm:"column:module;type:varchar(64);not null;index;comment:业务模块"`
	Action       string `gorm:"column:action;type:varchar(64);not null;index;comment:操作编码"`
	ActionLabel  string `gorm:"column:action_label;type:varchar(120);not null;default:'';comment:操作名称"`
	ResourceType string `gorm:"column:resource_type;type:varchar(64);not null;index:idx_sys_operation_logs_resource;comment:资源类型"`
	ResourceID   string `gorm:"column:resource_id;type:varchar(120);not null;index:idx_sys_operation_logs_resource;comment:资源ID"`
	ResourceName string `gorm:"column:resource_name;type:varchar(200);not null;default:'';index;comment:资源名称"`
	OperatorID   string `gorm:"column:operator_id;type:varchar(36);not null;default:'';index;comment:操作人ID"`
	OperatorName string `gorm:"column:operator_name;type:varchar(120);not null;default:'';comment:操作人名称"`
	Summary      string `gorm:"column:summary;type:varchar(500);not null;default:'';comment:操作摘要"`
	BeforeData   string `gorm:"column:before_data;type:mediumtext;not null;comment:变更前快照"`
	AfterData    string `gorm:"column:after_data;type:mediumtext;not null;comment:变更后快照"`
	CreatedAt    int64  `gorm:"column:created_at;type:bigint;not null;index;comment:操作时间"`
}

func (OperationLog) TableName() string {
	return "sys_operation_logs"
}

func (OperationLog) TableComment() string {
	return "平台操作记录表"
}
