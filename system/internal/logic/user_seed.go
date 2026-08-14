package logic

import (
	"context"
	"errors"

	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (u *UserLogic) EnsureSeedAdminUser(ctx context.Context) error {
	return seedAdminUser(ctx, u.userDao, u.roleDao)
}

func seedAdminUser(ctx context.Context, userDao *dao.User, roleDao *dao.Role) error {
	_, err := userDao.GetByUsername(ctx, "admin")
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	adminRole, err := roleDao.GetByCode(ctx, model.RoleCodeAdmin)
	if err != nil {
		return err
	}

	now := nowUnixMicro()
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		ID:           "00000000-0000-4000-8000-000000000001",
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hash),
		Status:       model.UserStatusEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := userDao.Create(ctx, user); err != nil {
		return err
	}
	return roleDao.SetUserRoles(ctx, user.ID, []string{adminRole.ID}, now)
}
