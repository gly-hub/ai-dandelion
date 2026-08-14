package model

import "github.com/gly-hub/ai-dandelion/system/boot"

func init() {
	boot.Register(&Menu{})
}

const (
	MenuTypeDirectory = 1
	MenuTypeMenu      = 2
	MenuTypeButton    = 3

	MenuStatusEnabled  = 1
	MenuStatusDisabled = 2

	MenuVisibleYes = 1
	MenuVisibleNo  = 0

	MenuPlacementPlatform  = "platform"
	MenuPlacementModuleNav = "module_nav"

	MenuModuleSystem        = "system"
	MenuModuleFuncOperation = "func-operation"
	MenuModuleAIAgent       = "ai-agent"

	MenuSourceTypeStatic            = "static"
	MenuSourceTypeGeneratedFunction = "generated_function"

	FuncUseMenuCode = "func-operation.use"
)

type Menu struct {
	ID         string `gorm:"column:id;type:varchar(36);primaryKey"`
	ParentID   string `gorm:"column:parent_id;type:varchar(36);not null;default:'';index;comment:父菜单ID"`
	Module     string `gorm:"column:module;type:varchar(32);not null;index;comment:所属模块"`
	Placement  string `gorm:"column:placement;type:varchar(32);not null;index;comment:挂载位置"`
	Name       string `gorm:"column:name;type:varchar(64);not null;comment:菜单名称"`
	Code       string `gorm:"column:code;type:varchar(120);not null;uniqueIndex;comment:菜单编码"`
	ViewKey    string `gorm:"column:view_key;type:varchar(120);not null;default:'';comment:前端视图键"`
	Icon       string `gorm:"column:icon;type:varchar(64);not null;default:'';comment:图标"`
	MenuType   int    `gorm:"column:menu_type;type:tinyint(1);not null;default:2;comment:菜单类型 1目录 2菜单 3按钮"`
	Sort       int    `gorm:"column:sort;type:int;not null;default:0;index;comment:排序"`
	Status     int    `gorm:"column:status;type:tinyint(1);not null;default:1;index;comment:状态 1启用 2禁用"`
	Visible    int    `gorm:"column:visible;type:tinyint(1);not null;default:1;comment:是否显示 1显示 0隐藏"`
	IsDefault  bool   `gorm:"column:is_default;type:tinyint(1);not null;default:0;comment:是否默认页"`
	Remark     string `gorm:"column:remark;type:varchar(200);not null;default:'';comment:备注"`
	SourceType string `gorm:"column:source_type;type:varchar(32);not null;default:'static';index;comment:来源类型 static/generated_function"`
	SourceID   string `gorm:"column:source_id;type:varchar(120);not null;default:'';index;comment:来源业务ID"`
	CreatedAt  int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt  int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (Menu) TableName() string {
	return "sys_menus"
}

func (Menu) TableComment() string {
	return "系统菜单表"
}
