package logic

import (
	"context"
	"testing"

	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const seededRolesMenuID = "00000000-0000-4000-8000-000000000013"

func TestSeedMenusMigratesLegacyEmptyRolesMenuID(t *testing.T) {
	ctx := context.Background()
	db := newMenuSeedDB(t)
	menuDao := dao.NewMenu(db)

	legacyRolesMenu := &model.Menu{
		ID:       "",
		Name:     "角色管理",
		Code:     "system.roles",
		MenuType: model.MenuTypeMenu,
	}
	if err := db.Create(legacyRolesMenu).Error; err != nil {
		t.Fatalf("create legacy roles menu: %v", err)
	}
	if err := db.Create(&model.RoleMenu{RoleID: "role-1", MenuID: ""}).Error; err != nil {
		t.Fatalf("create legacy role grant: %v", err)
	}

	if err := seedMenus(ctx, menuDao); err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	rolesMenu, err := menuDao.GetByCode(ctx, "system.roles")
	if err != nil {
		t.Fatalf("get roles menu: %v", err)
	}
	if rolesMenu.ID != seededRolesMenuID {
		t.Fatalf("roles menu ID = %q, want %q", rolesMenu.ID, seededRolesMenuID)
	}

	var grantCount int64
	if err := db.Model(&model.RoleMenu{}).
		Where("role_id = ? AND menu_id = ?", "role-1", seededRolesMenuID).
		Count(&grantCount).Error; err != nil {
		t.Fatalf("count migrated role grant: %v", err)
	}
	if grantCount != 1 {
		t.Fatalf("migrated role grant count = %d, want 1", grantCount)
	}
}

func TestSeedButtonMenusReparentsExistingRoleButtons(t *testing.T) {
	ctx := context.Background()
	db := newMenuSeedDB(t)
	menuDao := dao.NewMenu(db)
	rolesMenu := &model.Menu{
		ID:       "legacy-roles-menu-id",
		Name:     "角色管理",
		Code:     "system.roles",
		MenuType: model.MenuTypeMenu,
	}
	if err := db.Create(rolesMenu).Error; err != nil {
		t.Fatalf("create roles menu: %v", err)
	}
	if err := db.Create(&model.Menu{
		ID:       "roles-create",
		ParentID: "wrong-parent",
		Name:     "新建角色",
		Code:     "system.roles.create",
		MenuType: model.MenuTypeButton,
	}).Error; err != nil {
		t.Fatalf("create misplaced role button: %v", err)
	}

	if err := seedButtonMenus(ctx, menuDao); err != nil {
		t.Fatalf("seed button menus: %v", err)
	}

	for _, code := range []string{
		"system.roles.create",
		"system.roles.update",
		"system.roles.permissions",
		"system.roles.status",
		"system.roles.delete",
	} {
		button, err := menuDao.GetByCode(ctx, code)
		if err != nil {
			t.Fatalf("get %s: %v", code, err)
		}
		if button.ParentID != rolesMenu.ID {
			t.Errorf("%s parent ID = %q, want %q", code, button.ParentID, rolesMenu.ID)
		}
	}
}

func newMenuSeedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Menu{}, &model.RoleMenu{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}
