package model

import "github.com/gly-hub/ai-dandelion/func-operation/boot"

func init() {
	boot.Register(&FunctionRelease{}, &FunctionOutboxEvent{})
}

const (
	FunctionReleaseStatusStaged    = "staged"
	FunctionReleaseStatusPublished = "published"
	FunctionReleaseStatusRevoked   = "revoked"
)

// FunctionRelease binds one immutable artifact hash to a function version.
// Runtime code is never trusted merely because it exists on the filesystem.
type FunctionRelease struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID           string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:发布UUID"`
	FunctionID     string `gorm:"column:function_id;type:varchar(36);not null;index;comment:功能UUID"`
	AppID          string `gorm:"column:app_id;type:varchar(120);not null;index;comment:应用UUID"`
	Version        int64  `gorm:"column:version;type:bigint;not null;comment:功能发布版本"`
	ArtifactSHA256 string `gorm:"column:artifact_sha256;type:char(64);not null;index;comment:制品SHA256"`
	ManifestJSON   string `gorm:"column:manifest_json;type:longtext;not null;comment:校验后的manifest快照"`
	Status         string `gorm:"column:status;type:varchar(32);not null;index;comment:staged/published/revoked"`
	CreatedBy      string `gorm:"column:created_by;type:varchar(36);not null;default:'';index;comment:创建用户"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;comment:创建时间"`
	PublishedAt    int64  `gorm:"column:published_at;type:bigint;not null;default:0;comment:发布时间"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;comment:更新时间"`
}

func (FunctionRelease) TableName() string    { return "func_operation_function_releases" }
func (FunctionRelease) TableComment() string { return "功能不可变发布版本表" }

// FunctionOutboxEvent records cross-service effects for retry and audit.
type FunctionOutboxEvent struct {
	ID            uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID          string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:事件UUID"`
	FunctionID    string `gorm:"column:function_id;type:varchar(36);not null;index;comment:功能UUID"`
	ReleaseID     string `gorm:"column:release_id;type:varchar(36);not null;default:'';index;comment:发布UUID"`
	EventType     string `gorm:"column:event_type;type:varchar(64);not null;index;comment:事件类型"`
	PayloadJSON   string `gorm:"column:payload_json;type:longtext;not null;comment:事件载荷"`
	Status        string `gorm:"column:status;type:varchar(32);not null;default:'pending';index;comment:pending/done/failed"`
	Attempts      int32  `gorm:"column:attempts;type:int;not null;default:0;comment:尝试次数"`
	NextAttemptAt int64  `gorm:"column:next_attempt_at;type:bigint;not null;default:0;index;comment:下次重试时间"`
	LastError     string `gorm:"column:last_error;type:varchar(1000);not null;default:'';comment:最近失败原因"`
	CreatedAt     int64  `gorm:"column:created_at;type:bigint;not null;comment:创建时间"`
	UpdatedAt     int64  `gorm:"column:updated_at;type:bigint;not null;comment:更新时间"`
}

func (FunctionOutboxEvent) TableName() string    { return "func_operation_outbox_events" }
func (FunctionOutboxEvent) TableComment() string { return "功能跨服务副作用事件表" }
