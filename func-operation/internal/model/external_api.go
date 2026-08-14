package model

import "github.com/gly-hub/ai-dandelion/func-operation/boot"

func init() { boot.Register(&ExternalAPIClient{}, &ExternalAPIGroup{}, &ExternalAPI{}) }

type ExternalAPIClient struct {
	ID                   uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	UUID                 string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex"`
	ClientKey            string `gorm:"column:client_key;type:varchar(120);not null;uniqueIndex"`
	Name                 string `gorm:"column:name;type:varchar(120);not null"`
	BaseURL              string `gorm:"column:base_url;type:varchar(500);not null"`
	DefaultHeadersJSON   string `gorm:"column:default_headers_json;type:longtext;not null"`
	Description          string `gorm:"column:description;type:varchar(1000);not null;default:''"`
	Status               string `gorm:"column:status;type:varchar(32);not null;index"`
	CreatedBy            string `gorm:"column:created_by;type:varchar(36);not null;default:''"`
	UpdatedBy            string `gorm:"column:updated_by;type:varchar(36);not null;default:''"`
	CreatedAt            int64  `gorm:"column:created_at;type:bigint;not null;default:0"`
	UpdatedAt            int64  `gorm:"column:updated_at;type:bigint;not null;default:0"`
	DeletedAt            int64  `gorm:"column:deleted_at;type:bigint;not null;default:0;index"`
	SwaggerImportKeyHash string `gorm:"column:swagger_import_key_hash;type:char(64);not null;default:''"`
	PreRequestScript     string `gorm:"column:pre_request_script;type:longtext;not null;default:''"`
	PostResponseScript   string `gorm:"column:post_response_script;type:longtext;not null;default:''"`
}

func (ExternalAPIClient) TableName() string    { return "func_operation_external_api_clients" }
func (ExternalAPIClient) TableComment() string { return "功能运营外部接口客户端表" }

type ExternalAPI struct {
	ID                 uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	UUID               string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex"`
	ClientKey          string `gorm:"column:client_key;type:varchar(120);not null;index"`
	GroupID            string `gorm:"column:group_id;type:varchar(36);not null;default:'';index"`
	APIKey             string `gorm:"column:api_key;type:varchar(160);not null;uniqueIndex"`
	Name               string `gorm:"column:name;type:varchar(120);not null"`
	Method             string `gorm:"column:method;type:varchar(12);not null"`
	Path               string `gorm:"column:path;type:varchar(500);not null"`
	HeadersJSON        string `gorm:"column:headers_json;type:longtext;not null"`
	RequestSchemaJSON  string `gorm:"column:request_schema_json;type:longtext;not null"`
	ResponseSchemaJSON string `gorm:"column:response_schema_json;type:longtext;not null"`
	Description        string `gorm:"column:description;type:varchar(1000);not null;default:''"`
	Status             string `gorm:"column:status;type:varchar(32);not null;index"`
	CreatedBy          string `gorm:"column:created_by;type:varchar(36);not null;default:''"`
	UpdatedBy          string `gorm:"column:updated_by;type:varchar(36);not null;default:''"`
	CreatedAt          int64  `gorm:"column:created_at;type:bigint;not null;default:0"`
	UpdatedAt          int64  `gorm:"column:updated_at;type:bigint;not null;default:0"`
	DeletedAt          int64  `gorm:"column:deleted_at;type:bigint;not null;default:0;index"`
}

func (ExternalAPI) TableName() string    { return "func_operation_external_apis" }
func (ExternalAPI) TableComment() string { return "功能运营外部接口定义表" }

// ExternalAPIGroup is the YAPI-style directory that owns endpoint documents.
// ParentID is reserved for nested directories without changing endpoint ownership.
type ExternalAPIGroup struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	UUID        string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex"`
	ClientKey   string `gorm:"column:client_key;type:varchar(120);not null;index;uniqueIndex:idx_external_api_group_name"`
	ParentID    string `gorm:"column:parent_id;type:varchar(36);not null;default:'';index;uniqueIndex:idx_external_api_group_name"`
	Name        string `gorm:"column:name;type:varchar(120);not null;uniqueIndex:idx_external_api_group_name"`
	Description string `gorm:"column:description;type:varchar(1000);not null;default:''"`
	Sort        int32  `gorm:"column:sort;type:int;not null;default:0"`
	CreatedBy   string `gorm:"column:created_by;type:varchar(36);not null;default:''"`
	UpdatedBy   string `gorm:"column:updated_by;type:varchar(36);not null;default:''"`
	CreatedAt   int64  `gorm:"column:created_at;type:bigint;not null;default:0"`
	UpdatedAt   int64  `gorm:"column:updated_at;type:bigint;not null;default:0"`
	DeletedAt   int64  `gorm:"column:deleted_at;type:bigint;not null;default:0;index"`
}

func (ExternalAPIGroup) TableName() string    { return "func_operation_external_api_groups" }
func (ExternalAPIGroup) TableComment() string { return "功能运营外部接口分组表" }
