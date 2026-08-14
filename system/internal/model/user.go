package model

import "github.com/gly-hub/ai-dandelion/system/boot"

func init() {
	boot.Register(&User{})
}

const (
	UserStatusEnabled  = 1
	UserStatusDisabled = 2
)

type User struct {
	ID           string `gorm:"column:id;type:varchar(36);primaryKey"`
	Username     string `gorm:"column:username;type:varchar(64);not null;uniqueIndex;comment:用户名"`
	Email        string `gorm:"column:email;type:varchar(120);not null;uniqueIndex;comment:邮箱"`
	Phone        string `gorm:"column:phone;type:varchar(20);not null;default:'';comment:手机号"`
	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null;comment:密码哈希"`
	Status       int    `gorm:"column:status;type:tinyint(1);not null;default:1;index;comment:状态 1启用 2禁用"`
	CreatedAt    int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt    int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (User) TableName() string {
	return "sys_users"
}

func (User) TableComment() string {
	return "系统用户表"
}
