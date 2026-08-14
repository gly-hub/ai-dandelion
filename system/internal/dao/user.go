package dao

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type User struct {
	db *gorm.DB
}

func NewUser(db *gorm.DB) *User {
	return &User{db: db}
}

func (u *User) Transaction(ctx context.Context, fn func(*User) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(NewUser(tx))
	})
}

func (u *User) DB() *gorm.DB {
	return u.db
}

func (u *User) List(ctx context.Context) ([]model.User, error) {
	var users []model.User
	err := u.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&users).Error
	return users, err
}

func (u *User) Create(ctx context.Context, user *model.User) error {
	return u.db.WithContext(ctx).Create(user).Error
}

func (u *User) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := u.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *User) Get(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := u.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *User) Delete(ctx context.Context, id string) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&model.User{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (u *User) Update(ctx context.Context, user *model.User) error {
	return u.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"username":      user.Username,
			"email":         user.Email,
			"phone":         user.Phone,
			"password_hash": user.PasswordHash,
			"status":        user.Status,
			"updated_at":    user.UpdatedAt,
		}).Error
}

func (u *User) UpdateStatus(ctx context.Context, id string, status int, updatedAt int64) error {
	result := u.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"updated_at": updatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
