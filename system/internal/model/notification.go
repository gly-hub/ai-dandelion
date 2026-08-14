package model

import "github.com/gly-hub/ai-dandelion/system/boot"

func init() { boot.Register(&Notification{}) }

const (
	NotificationDisplayModal = "modal"
	NotificationDisplayToast = "toast"
	NotificationLevelInfo    = "info"
	NotificationLevelSuccess = "success"
	NotificationLevelWarning = "warning"
	NotificationLevelError   = "error"
)

type Notification struct {
	ID             string `gorm:"column:id;type:varchar(36);primaryKey"`
	Title          string `gorm:"column:title;type:varchar(200);not null;comment:通知标题"`
	Content        string `gorm:"column:content;type:text;not null;comment:通知内容"`
	DisplayType    string `gorm:"column:display_type;type:varchar(20);not null;index;comment:展示类型 modal/toast"`
	Level          string `gorm:"column:level;type:varchar(20);not null;default:'info';comment:通知级别"`
	TargetUserID   string `gorm:"column:target_user_id;type:varchar(36);not null;index;comment:目标用户ID"`
	TargetUserName string `gorm:"column:target_user_name;type:varchar(120);not null;default:'';comment:目标用户名称"`
	Read           bool   `gorm:"column:read;type:tinyint(1);not null;default:0;index;comment:是否已读"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;index;comment:创建时间"`
}

func (Notification) TableName() string    { return "sys_notifications" }
func (Notification) TableComment() string { return "系统通知表" }
