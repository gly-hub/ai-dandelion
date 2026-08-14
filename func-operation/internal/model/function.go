package model

import "github.com/team-dandelion/ai-dandelion/func-operation/boot"

func init() {
	boot.Register(&Function{})
}

const (
	FunctionStatusDraft     = "draft"
	FunctionStatusPublished = "published"

	FunctionWorkflowStageProductDoc     = "product_doc"
	FunctionWorkflowStageTechnicalDoc   = "technical_doc"
	FunctionWorkflowStageCodeGeneration = "code_generation"
	FunctionWorkflowStageCodeGenerated  = "code_generated"
)

type Function struct {
	ID                    uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID                  string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:功能UUID"`
	Name                  string `gorm:"column:name;type:varchar(120);not null;default:'';comment:功能名称"`
	Description           string `gorm:"column:description;type:varchar(500);not null;default:'';comment:功能描述"`
	Status                string `gorm:"column:status;type:varchar(32);not null;default:'draft';index;comment:功能发布状态"`
	WorkflowStage         string `gorm:"column:workflow_stage;type:varchar(32);not null;default:'product_doc';index;comment:功能生成阶段"`
	ProductDoc            string `gorm:"column:product_doc;type:mediumtext;not null;comment:产品文档"`
	TechnicalDoc          string `gorm:"column:technical_doc;type:mediumtext;not null;comment:研发文档"`
	ProductDocPath        string `gorm:"column:product_doc_path;type:varchar(500);not null;default:'';comment:产品文档正式文件地址"`
	TechnicalDocPath      string `gorm:"column:technical_doc_path;type:varchar(500);not null;default:'';comment:研发文档正式文件地址"`
	Entry                 string `gorm:"column:entry;type:varchar(500);not null;default:'';comment:功能入口"`
	ProductSessionID      string `gorm:"column:product_session_id;type:varchar(36);not null;default:'';index;comment:关联产品文档会话ID"`
	TechnicalSessionID    string `gorm:"column:technical_session_id;type:varchar(36);not null;default:'';index;comment:关联研发文档会话ID"`
	GenerationSessionID   string `gorm:"column:generation_session_id;type:varchar(36);not null;default:'';index;comment:关联代码生成会话ID"`
	GeneratedAppID        string `gorm:"column:generated_app_id;type:varchar(120);not null;default:'';index;comment:关联生成应用UUID"`
	ActiveReleaseID       string `gorm:"column:active_release_id;type:varchar(36);not null;default:'';index;comment:当前已发布版本UUID"`
	FunctionVersion       int64  `gorm:"column:function_version;type:bigint;not null;default:0;comment:功能全局版本号"`
	ProductDocVersion     int64  `gorm:"column:product_doc_version;type:bigint;not null;default:0;comment:产品文档已应用版本号"`
	ProductDraftVersion   int64  `gorm:"column:product_draft_version;type:bigint;not null;default:0;comment:产品文档草稿版本号"`
	TechnicalDocVersion   int64  `gorm:"column:technical_doc_version;type:bigint;not null;default:0;comment:研发文档已应用版本号"`
	TechnicalDraftVersion int64  `gorm:"column:technical_draft_version;type:bigint;not null;default:0;comment:研发文档草稿版本号"`
	CodeVersion           int64  `gorm:"column:code_version;type:bigint;not null;default:0;comment:代码已生成版本号"`
	CodeDraftVersion      int64  `gorm:"column:code_draft_version;type:bigint;not null;default:0;comment:代码草稿版本号"`
	MenuParentID          string `gorm:"column:menu_parent_id;type:varchar(36);not null;default:'';index;comment:所属目录菜单ID"`
	MenuID                string `gorm:"column:menu_id;type:varchar(36);not null;default:'';index;comment:同步生成的菜单ID"`
	CreatedAt             int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt             int64  `gorm:"column:updated_at;type:bigint;not null;default:0;index;comment:更新时间"`

	// Populated from generated_apps/<id>/documents/* at read time; not persisted.
	DocTechnicalStale bool `gorm:"-"`
}

func (Function) TableName() string {
	return "func_operation_functions"
}

func (Function) TableComment() string {
	return "功能运营功能表"
}
