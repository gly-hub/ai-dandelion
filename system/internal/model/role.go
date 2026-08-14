package model

import "github.com/team-dandelion/ai-dandelion/system/boot"

func init() {
	boot.Register(&Role{})
	boot.Register(&UserRole{})
	boot.Register(&RoleMenu{})
}

const (
	RoleStatusEnabled  = 1
	RoleStatusDisabled = 2

	RoleCodeAdmin = "admin"
)

type Role struct {
	ID        string `gorm:"column:id;type:varchar(36);primaryKey"`
	Name      string `gorm:"column:name;type:varchar(64);not null;comment:角色名称"`
	Code      string `gorm:"column:code;type:varchar(64);not null;uniqueIndex;comment:角色编码"`
	Status    int    `gorm:"column:status;type:tinyint(1);not null;default:1;index;comment:状态 1启用 2禁用"`
	Remark    string `gorm:"column:remark;type:varchar(200);not null;default:'';comment:备注"`
	Sort      int    `gorm:"column:sort;type:int;not null;default:0;index;comment:排序"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (Role) TableName() string {
	return "sys_roles"
}

func (Role) TableComment() string {
	return "系统角色表"
}

type UserRole struct {
	UserID    string `gorm:"column:user_id;type:varchar(36);primaryKey;comment:用户ID"`
	RoleID    string `gorm:"column:role_id;type:varchar(36);primaryKey;index;comment:角色ID"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
}

func (UserRole) TableName() string {
	return "sys_user_roles"
}

func (UserRole) TableComment() string {
	return "用户角色关联表"
}

type RoleMenu struct {
	RoleID    string `gorm:"column:role_id;type:varchar(36);primaryKey;comment:角色ID"`
	MenuID    string `gorm:"column:menu_id;type:varchar(36);primaryKey;index;comment:菜单ID"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
}

func (RoleMenu) TableName() string {
	return "sys_role_menus"
}

func (RoleMenu) TableComment() string {
	return "角色菜单关联表"
}
