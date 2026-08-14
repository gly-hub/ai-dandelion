package dao

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type Upload struct{ db *gorm.DB }

func NewUpload(db *gorm.DB) *Upload { return &Upload{db: db} }

func (d *Upload) FindReusable(ctx context.Context, md5 string) (*model.Upload, error) {
	var item model.Upload
	err := d.db.WithContext(ctx).Where("md5 = ?", md5).Order("updated_at DESC").First(&item).Error
	return &item, err
}
func (d *Upload) Create(ctx context.Context, item *model.Upload) error {
	return d.db.WithContext(ctx).Create(item).Error
}
func (d *Upload) Get(ctx context.Context, uuid string) (*model.Upload, error) {
	var item model.Upload
	err := d.db.WithContext(ctx).Where("uuid = ?", uuid).First(&item).Error
	return &item, err
}
func (d *Upload) MarkCompleted(ctx context.Context, uuid string, completedAt int64) error {
	return d.db.WithContext(ctx).Model(&model.Upload{}).Where("uuid = ?", uuid).Updates(map[string]interface{}{
		"status": model.UploadStatusCompleted, "completed_at": completedAt, "updated_at": completedAt,
	}).Error
}
