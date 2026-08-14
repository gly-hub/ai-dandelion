package model

import "github.com/gly-hub/ai-dandelion/func-operation/boot"

func init() {
	boot.Register(&GeneratedApp{}, &AppRecord{})
}

type GeneratedApp struct {
	ID            uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID          string `gorm:"column:uuid;type:varchar(120);not null;uniqueIndex;comment:应用UUID"`
	Name          string `gorm:"column:name;type:varchar(120);not null;default:'';comment:应用名称"`
	Version       string `gorm:"column:version;type:varchar(64);not null;default:'';comment:版本"`
	Description   string `gorm:"column:description;type:varchar(500);not null;default:'';comment:描述"`
	Export        string `gorm:"column:export;type:varchar(120);not null;default:'handle';comment:WASM导出函数"`
	FrontendEntry string `gorm:"column:frontend_entry;type:varchar(500);not null;default:'';comment:前端入口"`
	BackendSource string `gorm:"column:backend_source;type:varchar(500);not null;default:'';comment:后端源码"`
	BackendModule string `gorm:"column:backend_module;type:varchar(500);not null;default:'';comment:WASM模块"`
	CreatedAt     int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt     int64  `gorm:"column:updated_at;type:bigint;not null;default:0;index;comment:更新时间"`
}

func (GeneratedApp) TableName() string {
	return "func_operation_generated_apps"
}

func (GeneratedApp) TableComment() string {
	return "生成应用注册表"
}

type AppRecord struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID      string `gorm:"column:uuid;type:varchar(120);not null;uniqueIndex;comment:记录UUID"`
	AppUUID   string `gorm:"column:app_uuid;type:varchar(120);not null;index;comment:应用UUID"`
	DataJSON  string `gorm:"column:data_json;type:longtext;not null;comment:记录JSON"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint;not null;default:0;index;comment:更新时间"`
}

func (AppRecord) TableName() string {
	return "func_operation_app_records"
}

func (AppRecord) TableComment() string {
	return "生成应用记录表"
}
