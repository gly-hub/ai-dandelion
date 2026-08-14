package dao

import (
	"context"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type NotificationListFilter struct {
	UserID         string
	Page, PageSize int
	UnreadOnly     bool
}

type Notification struct{ db *gorm.DB }

func NewNotification(db *gorm.DB) *Notification { return &Notification{db: db} }
func (d *Notification) Create(ctx context.Context, item *model.Notification) error {
	return d.db.WithContext(ctx).Create(item).Error
}
func (d *Notification) List(ctx context.Context, f NotificationListFilter) ([]model.Notification, int64, error) {
	q := d.db.WithContext(ctx).Model(&model.Notification{}).Where("target_user_id = ?", f.UserID)
	if f.UnreadOnly {
		q = q.Where("read = ?", false)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Notification
	err := q.Order("created_at DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&items).Error
	return items, total, err
}
func (d *Notification) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Notification{}).Where("target_user_id = ? AND read = ?", userID, false).Count(&count).Error
	return count, err
}
func (d *Notification) MarkRead(ctx context.Context, userID, id string) error {
	result := d.db.WithContext(ctx).Model(&model.Notification{}).Where("id = ? AND target_user_id = ?", id, userID).Updates(map[string]any{"read": true})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
