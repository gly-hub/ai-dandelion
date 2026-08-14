package dao

import (
	"context"
	"strings"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Role struct {
	db *gorm.DB
}

func NewRole(db *gorm.DB) *Role {
	return &Role{db: db}
}

func (r *Role) Transaction(ctx context.Context, fn func(*Role) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(NewRole(tx))
	})
}

func (r *Role) DB() *gorm.DB {
	return r.db
}

func (r *Role) List(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Order("sort ASC, created_at ASC").Find(&roles).Error
	return roles, err
}

func (r *Role) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *Role) Get(ctx context.Context, id string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Role) GetByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Role) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).
		Model(&model.Role{}).
		Where("id = ?", role.ID).
		Updates(map[string]any{
			"name":       role.Name,
			"code":       role.Code,
			"status":     role.Status,
			"remark":     role.Remark,
			"sort":       role.Sort,
			"updated_at": role.UpdatedAt,
		}).Error
}

func (r *Role) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&model.RoleMenu{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&model.Role{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *Role) UpdateStatus(ctx context.Context, id string, status int, updatedAt int64) error {
	result := r.db.WithContext(ctx).
		Model(&model.Role{}).
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

func (r *Role) ListMenuIDsByRole(ctx context.Context, roleID string) ([]string, error) {
	var menuIDs []string
	err := r.db.WithContext(ctx).
		Table("sys_role_menus rm").
		Joins("JOIN sys_menus m ON m.id = rm.menu_id").
		Where("rm.role_id = ?", roleID).
		Pluck("rm.menu_id", &menuIDs).Error
	return menuIDs, err
}

func (r *Role) ListMenusByRole(ctx context.Context, roleID string) ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.WithContext(ctx).
		Table("sys_menus m").
		Joins("JOIN sys_role_menus rm ON rm.menu_id = m.id").
		Where("rm.role_id = ?", roleID).
		Order("m.sort ASC, m.created_at ASC").
		Find(&menus).Error
	return menus, err
}

func (r *Role) SetRoleMenus(ctx context.Context, roleID string, menuIDs []string, createdAt int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleMenu{}).Error; err != nil {
			return err
		}
		if len(menuIDs) == 0 {
			return nil
		}
		rows := make([]model.RoleMenu, 0, len(menuIDs))
		for _, menuID := range menuIDs {
			rows = append(rows, model.RoleMenu{
				RoleID:    roleID,
				MenuID:    menuID,
				CreatedAt: createdAt,
			})
		}
		return tx.Create(&rows).Error
	})
}

// GrantRoleMenus adds menu permissions without replacing the role's existing
// assignments. It is used for menus created after the administrator role was
// initially seeded.
func (r *Role) GrantRoleMenus(ctx context.Context, roleID string, menuIDs []string, createdAt int64) error {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return nil
	}

	rows := make([]model.RoleMenu, 0, len(menuIDs))
	seen := make(map[string]struct{}, len(menuIDs))
	for _, menuID := range menuIDs {
		menuID = strings.TrimSpace(menuID)
		if menuID == "" {
			continue
		}
		if _, ok := seen[menuID]; ok {
			continue
		}
		seen[menuID] = struct{}{}
		rows = append(rows, model.RoleMenu{RoleID: roleID, MenuID: menuID, CreatedAt: createdAt})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (r *Role) ListMenuIDsByUser(ctx context.Context, userID string) ([]string, error) {
	var menuIDs []string
	err := r.db.WithContext(ctx).
		Table("sys_role_menus rm").
		Joins("JOIN sys_user_roles ur ON ur.role_id = rm.role_id").
		Joins("JOIN sys_roles r ON r.id = ur.role_id AND r.status = ?", model.RoleStatusEnabled).
		Where("ur.user_id = ?", userID).
		Distinct().
		Pluck("rm.menu_id", &menuIDs).Error
	return menuIDs, err
}

func (r *Role) ListRoleIDsByUser(ctx context.Context, userID string) ([]string, error) {
	var roleIDs []string
	err := r.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("user_id = ?", userID).
		Pluck("role_id", &roleIDs).Error
	return roleIDs, err
}

func (r *Role) ListRolesByUser(ctx context.Context, userID string) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).
		Table("sys_roles r").
		Joins("JOIN sys_user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Order("r.sort ASC, r.created_at ASC").
		Find(&roles).Error
	return roles, err
}

func (r *Role) SetUserRoles(ctx context.Context, userID string, roleIDs []string, createdAt int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		if len(roleIDs) == 0 {
			return nil
		}
		rows := make([]model.UserRole, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			rows = append(rows, model.UserRole{
				UserID:    userID,
				RoleID:    roleID,
				CreatedAt: createdAt,
			})
		}
		return tx.Create(&rows).Error
	})
}

func (r *Role) ListRoleIDsByUsers(ctx context.Context, userIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var rows []model.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], row.RoleID)
	}
	return result, nil
}
