package model

import "github.com/team-dandelion/ai-dandelion/system/boot"

const (
	UploadStatusPending   = "pending"
	UploadStatusCompleted = "completed"
)

func init() { boot.Register(&Upload{}) }

type Upload struct {
	UUID        string `gorm:"column:uuid;type:varchar(100);primaryKey"`
	UploadID    string `gorm:"column:upload_id;type:varchar(255);not null;default:'';index"`
	MD5         string `gorm:"column:md5;type:char(32);not null;uniqueIndex"`
	FileName    string `gorm:"column:file_name;type:varchar(255);not null;default:''"`
	ContentType string `gorm:"column:content_type;type:varchar(255);not null;default:''"`
	FileSize    int64  `gorm:"column:file_size;not null;default:0"`
	Mode        string `gorm:"column:mode;type:varchar(20);not null;default:''"`
	PartSize    int64  `gorm:"column:part_size;not null;default:0"`
	TotalParts  int    `gorm:"column:total_parts;not null;default:0"`
	Status      string `gorm:"column:status;type:varchar(20);not null;default:'pending';index"`
	CreatedAt   int64  `gorm:"column:created_at;not null;default:0"`
	UpdatedAt   int64  `gorm:"column:updated_at;not null;default:0"`
	CompletedAt int64  `gorm:"column:completed_at;not null;default:0"`
}

func (Upload) TableName() string    { return "uploads" }
func (Upload) TableComment() string { return "文件上传记录" }
