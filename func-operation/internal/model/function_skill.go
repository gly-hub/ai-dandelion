package model

import "github.com/gly-hub/ai-dandelion/func-operation/boot"

func init() {
	boot.Register(&FunctionSkill{}, &FunctionSkillRelease{}, &FunctionSkillGrant{}, &FunctionSkillApproval{}, &FunctionSkillExecution{})
}

const (
	FunctionSkillStatusEnabled  = "enabled"
	FunctionSkillStatusDisabled = "disabled"

	FunctionSkillReleaseStatusActive  = "active"
	FunctionSkillReleaseStatusRevoked = "revoked"
)

// FunctionSkill is the stable, user-visible capability attached to one function.
type FunctionSkill struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID        string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:技能UUID"`
	FunctionID  string `gorm:"column:function_id;type:varchar(36);not null;uniqueIndex;comment:功能UUID"`
	Name        string `gorm:"column:name;type:varchar(120);not null;default:'';comment:技能名称"`
	Description string `gorm:"column:description;type:varchar(500);not null;default:'';comment:技能描述"`
	ToolPrefix  string `gorm:"column:tool_prefix;type:varchar(120);not null;uniqueIndex;comment:MCP工具前缀"`
	Status      string `gorm:"column:status;type:varchar(32);not null;default:'enabled';index;comment:enabled/disabled"`
	CreatedAt   int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt   int64  `gorm:"column:updated_at;type:bigint;not null;default:0;index;comment:更新时间"`
}

func (FunctionSkill) TableName() string    { return "func_operation_function_skills" }
func (FunctionSkill) TableComment() string { return "功能对话技能表" }

// FunctionSkillRelease is an immutable skill contract tied to a trusted function release.
type FunctionSkillRelease struct {
	ID                uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID              string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:技能版本UUID"`
	SkillID           string `gorm:"column:skill_id;type:varchar(36);not null;index;comment:技能UUID"`
	FunctionID        string `gorm:"column:function_id;type:varchar(36);not null;index;comment:功能UUID"`
	FunctionReleaseID string `gorm:"column:function_release_id;type:varchar(36);not null;uniqueIndex;comment:功能发布版本UUID"`
	AppID             string `gorm:"column:app_id;type:varchar(120);not null;index;comment:应用UUID"`
	ArtifactSHA256    string `gorm:"column:artifact_sha256;type:char(64);not null;comment:制品SHA256"`
	ContractJSON      string `gorm:"column:contract_json;type:longtext;not null;comment:技能契约快照"`
	Status            string `gorm:"column:status;type:varchar(32);not null;default:'active';index;comment:active/revoked"`
	CreatedAt         int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
	UpdatedAt         int64  `gorm:"column:updated_at;type:bigint;not null;default:0;index;comment:更新时间"`
}

func (FunctionSkillRelease) TableName() string    { return "func_operation_function_skill_releases" }
func (FunctionSkillRelease) TableComment() string { return "功能对话技能发布版本表" }

type FunctionSkillGrant struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID      string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:授权UUID"`
	TokenHash string `gorm:"column:token_hash;type:char(64);not null;uniqueIndex;comment:授权令牌哈希"`
	UserID    string `gorm:"column:user_id;type:varchar(36);not null;index;comment:用户ID"`
	SessionID string `gorm:"column:session_id;type:varchar(36);not null;index;comment:Agent会话ID"`
	SkillIDs  string `gorm:"column:skill_ids;type:longtext;not null;comment:允许技能JSON"`
	ExpiresAt int64  `gorm:"column:expires_at;type:bigint;not null;index;comment:过期时间"`
	RevokedAt int64  `gorm:"column:revoked_at;type:bigint;not null;default:0;comment:撤销时间"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
}

func (FunctionSkillGrant) TableName() string    { return "func_operation_function_skill_grants" }
func (FunctionSkillGrant) TableComment() string { return "功能对话技能短期授权表" }

type FunctionSkillApproval struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID      string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:确认UUID"`
	TokenHash string `gorm:"column:token_hash;type:char(64);not null;uniqueIndex;comment:确认令牌哈希"`
	GrantID   string `gorm:"column:grant_id;type:varchar(36);not null;index;comment:授权UUID"`
	ToolName  string `gorm:"column:tool_name;type:varchar(180);not null;comment:工具名称"`
	ToolUseID string `gorm:"column:tool_use_id;type:varchar(180);not null;comment:工具调用ID"`
	InputHash string `gorm:"column:input_hash;type:char(64);not null;comment:输入哈希"`
	ExpiresAt int64  `gorm:"column:expires_at;type:bigint;not null;index;comment:过期时间"`
	UsedAt    int64  `gorm:"column:used_at;type:bigint;not null;default:0;comment:使用时间"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间"`
}

func (FunctionSkillApproval) TableName() string    { return "func_operation_function_skill_approvals" }
func (FunctionSkillApproval) TableComment() string { return "功能对话技能写操作确认表" }

type FunctionSkillExecution struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID"`
	UUID           string `gorm:"column:uuid;type:varchar(36);not null;uniqueIndex;comment:执行UUID"`
	FunctionID     string `gorm:"column:function_id;type:varchar(36);not null;index;comment:功能UUID"`
	SkillReleaseID string `gorm:"column:skill_release_id;type:varchar(36);not null;uniqueIndex:uk_skill_execution;index;comment:技能发布版本UUID"`
	UserID         string `gorm:"column:user_id;type:varchar(36);not null;index;comment:用户ID"`
	SessionID      string `gorm:"column:session_id;type:varchar(36);not null;index;comment:Agent会话ID"`
	ToolName       string `gorm:"column:tool_name;type:varchar(180);not null;comment:工具名称"`
	ToolUseID      string `gorm:"column:tool_use_id;type:varchar(180);not null;uniqueIndex:uk_skill_execution;comment:工具调用ID"`
	InputJSON      string `gorm:"column:input_json;type:longtext;not null;comment:脱敏输入"`
	ResultJSON     string `gorm:"column:result_json;type:longtext;not null;comment:脱敏结果"`
	Status         string `gorm:"column:status;type:varchar(32);not null;index;comment:succeeded/failed"`
	ErrorMessage   string `gorm:"column:error_message;type:varchar(1000);not null;default:'';comment:错误信息"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint;not null;default:0;index;comment:创建时间"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间"`
}

func (FunctionSkillExecution) TableName() string    { return "func_operation_function_skill_executions" }
func (FunctionSkillExecution) TableComment() string { return "功能对话技能执行审计表" }
