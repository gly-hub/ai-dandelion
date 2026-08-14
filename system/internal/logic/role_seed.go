package logic

import (
	"context"
	"errors"

	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

func (m *MenuLogic) EnsureSeedRoles(ctx context.Context) error {
	return seedRoles(ctx, m.menuDao, m.roleDao)
}

func seedRoles(ctx context.Context, menuDao *dao.Menu, roleDao *dao.Role) error {
	now := nowUnixMicro()
	adminRole, err := roleDao.GetByCode(ctx, model.RoleCodeAdmin)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		adminRole = &model.Role{
			ID:        "00000000-0000-4000-8000-000000000101",
			Name:      "超级管理员",
			Code:      model.RoleCodeAdmin,
			Status:    model.RoleStatusEnabled,
			Remark:    "拥有全部菜单与按钮权限",
			Sort:      10,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := roleDao.Create(ctx, adminRole); err != nil {
			return err
		}
	}

	if _, err := menuDao.GetByCode(ctx, "system.roles"); errors.Is(err, gorm.ErrRecordNotFound) {
		rolesMenu := &model.Menu{
			ID:        "00000000-0000-4000-8000-000000000013",
			ParentID:  "00000000-0000-4000-8000-000000000035",
			Module:    model.MenuModuleFuncOperation,
			Placement: model.MenuPlacementModuleNav,
			Name:      "角色管理",
			Code:      "system.roles",
			ViewKey:   "roles",
			Icon:      "SafetyOutlined",
			MenuType:  model.MenuTypeMenu,
			Sort:      30,
			Status:    model.MenuStatusEnabled,
			Visible:   model.MenuVisibleYes,
			Remark:    "配置角色与菜单权限",
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := menuDao.Create(ctx, rolesMenu); err != nil {
			return err
		}
	}

	if err := ensureAgentAdminMenus(ctx, menuDao); err != nil {
		return err
	}

	menus, err := menuDao.List(ctx, dao.MenuListFilter{})
	if err != nil {
		return err
	}
	menuIDs := make([]string, 0, len(menus))
	for i := range menus {
		menuIDs = append(menuIDs, menus[i].ID)
	}
	return roleDao.SetRoleMenus(ctx, adminRole.ID, menuIDs, now)
}
