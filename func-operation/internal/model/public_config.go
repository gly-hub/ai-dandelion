package model

import "github.com/gly-hub/ai-dandelion/func-operation/boot"

func init() {
	boot.Register(&PublicConfig{}, &PublicConfigVersion{}, &PublicConfigImportKey{})
}

// PublicConfig is a globally-addressable runtime option list. ConfigKey is
// intentionally immutable so generated functions always have a stable handle.
type PublicConfig struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID        string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:公共配置UUID"`
	ConfigKey   string `gorm:"column:config_key;type:varchar(120);not null;uniqueIndex;comment:全局唯一配置键"`
	Name        string `gorm:"column:name;type:varchar(120);not null;default:'';comment:配置名称"`
	Description string `gorm:"column:description;type:varchar(500);not null;default:'';comment:配置说明"`
	ValueJSON   string `gorm:"column:value_json;type:longtext;not null;comment:当前配置JSON"`
	Version     int64  `gorm:"column:version;type:bigint;not null;default:1;comment:当前版本"`
	CreatedBy   string `gorm:"column:created_by;type:varchar(36);not null;default:'';index;comment:创建人"`
	UpdatedBy   string `gorm:"column:updated_by;type:varchar(36);not null;default:'';index;comment:最近更新人"`
	CreatedAt   int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt   int64  `gorm:"column:updated_at;type:bigint;not null;default:0;index;comment:更新时间"`
}

func (PublicConfig) TableName() string    { return "func_operation_public_configs" }
func (PublicConfig) TableComment() string { return "功能运营公共配置表" }

// PublicConfigVersion keeps immutable full snapshots so an upload can be
// reverted without reconstructing a value from incremental changes.
type PublicConfigVersion struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID       string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:配置版本UUID"`
	ConfigID   string `gorm:"column:config_id;type:varchar(36);not null;index;comment:公共配置UUID"`
	ConfigKey  string `gorm:"column:config_key;type:varchar(120);not null;index;comment:全局唯一配置键"`
	Version    int64  `gorm:"column:version;type:bigint;not null;comment:版本号"`
	ValueJSON  string `gorm:"column:value_json;type:longtext;not null;comment:版本配置JSON"`
	OperatorID string `gorm:"column:operator_id;type:varchar(36);not null;default:'';index;comment:操作人"`
	Source     string `gorm:"column:source;type:varchar(32);not null;default:'';comment:create/update/rollback"`
	CreatedAt  int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
}

func (PublicConfigVersion) TableName() string    { return "func_operation_public_config_versions" }
func (PublicConfigVersion) TableComment() string { return "功能运营公共配置版本表" }

// PublicConfigImportKey is the single credential scope for bulk public-config
// uploads. It intentionally has no config_key: one key may upload all keys.
type PublicConfigImportKey struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	KeyName   string `gorm:"column:key_name;type:varchar(64);not null;uniqueIndex"`
	KeyHash   string `gorm:"column:key_hash;type:char(64);not null"`
	UpdatedBy string `gorm:"column:updated_by;type:varchar(36);not null;default:''"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint;not null;default:0"`
}

func (PublicConfigImportKey) TableName() string { return "func_operation_public_config_import_keys" }
func (PublicConfigImportKey) TableComment() string {
	return "功能运营公共配置批量上传密钥表"
}
