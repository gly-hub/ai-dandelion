package logic

import (
	"context"
	"strings"
	"testing"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRoleLogicRecordsRoleAndMenuPermissionChanges(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Role{}, &model.RoleMenu{}, &model.UserRole{}, &model.Menu{}, &model.OperationLog{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	menuDao := dao.NewMenu(db)
	roleDao := dao.NewRole(db)
	operationLogDao := dao.NewOperationLog(db)
	menu := &model.Menu{
		ID: "menu-1", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav,
		Name: "功能管理", Code: "func-operation.functions", MenuType: model.MenuTypeMenu,
		Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := menuDao.Create(context.Background(), menu); err != nil {
		t.Fatalf("create menu: %v", err)
	}
	operationLogLogic := NewOperationLogLogic(operationLogDao)
	roleLogic := NewRoleLogic(roleDao, menuDao, operationLogLogic)
	ctx := authctx.ContextWithUser(context.Background(), authctx.User{ID: "user-1", Username: "admin"})

	role, err := roleLogic.CreateRole(ctx, &systemproto.CreateRoleReq{Name: "运营", Code: "operator"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if _, err := roleLogic.SetRoleMenus(ctx, &systemproto.SetRoleMenusReq{Id: role.Id, MenuIds: []string{menu.ID}}); err != nil {
		t.Fatalf("set role menus: %v", err)
	}

	logs, total, err := operationLogDao.List(ctx, dao.OperationLogListFilter{
		ResourceType: OperationResourceRole,
		ResourceID:   role.Id,
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("log count = %d/%d, want 2", len(logs), total)
	}

	var permissionLog *model.OperationLog
	for i := range logs {
		if logs[i].Action == "role.menu.update" {
			permissionLog = &logs[i]
			break
		}
	}
	if permissionLog == nil {
		t.Fatal("expected role.menu.update record")
	}
	if permissionLog.OperatorID != "user-1" || permissionLog.OperatorName != "admin" {
		t.Fatalf("unexpected operator: %#v", permissionLog)
	}
	if strings.Contains(permissionLog.BeforeData, menu.Code) || !strings.Contains(permissionLog.AfterData, menu.Code) {
		t.Fatalf("unexpected permission snapshots: before=%s after=%s", permissionLog.BeforeData, permissionLog.AfterData)
	}
}

func TestGetRoleMenusSkipsDeletedMenuAssociations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Role{}, &model.RoleMenu{}, &model.Menu{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	roleDao := dao.NewRole(db)
	menuDao := dao.NewMenu(db)
	role := &model.Role{ID: "role-1", Name: "测试", Code: "tester", Status: model.RoleStatusEnabled}
	menu := &model.Menu{ID: "menu-1", Module: model.MenuModuleSystem, Placement: model.MenuPlacementPlatform, Name: "菜单", Code: "system.menu-1", MenuType: model.MenuTypeMenu}
	if err := roleDao.Create(context.Background(), role); err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := menuDao.Create(context.Background(), menu); err != nil {
		t.Fatalf("create menu: %v", err)
	}
	if err := db.Create(&model.RoleMenu{RoleID: role.ID, MenuID: menu.ID}).Error; err != nil {
		t.Fatalf("grant menu: %v", err)
	}
	if err := db.Create(&model.RoleMenu{RoleID: role.ID, MenuID: "deleted-menu"}).Error; err != nil {
		t.Fatalf("create stale grant: %v", err)
	}

	logic := NewRoleLogic(roleDao, menuDao, nil)
	menuIDs, err := logic.GetRoleMenus(context.Background(), &systemproto.GetRoleMenusReq{Id: role.ID})
	if err != nil {
		t.Fatalf("get role menus: %v", err)
	}
	if len(menuIDs) != 1 || menuIDs[0] != menu.ID {
		t.Fatalf("menu IDs = %v, want [%s]", menuIDs, menu.ID)
	}
}

func TestDeleteMenuRemovesRoleMenuAssociations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.RoleMenu{}, &model.Menu{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	menuDao := dao.NewMenu(db)
	menu := &model.Menu{ID: "menu-1", Module: model.MenuModuleSystem, Placement: model.MenuPlacementPlatform, Name: "菜单", Code: "system.menu-1", MenuType: model.MenuTypeMenu}
	if err := menuDao.Create(context.Background(), menu); err != nil {
		t.Fatalf("create menu: %v", err)
	}
	if err := db.Create(&model.RoleMenu{RoleID: "role-1", MenuID: menu.ID}).Error; err != nil {
		t.Fatalf("grant menu: %v", err)
	}
	if err := menuDao.Delete(context.Background(), menu.ID); err != nil {
		t.Fatalf("delete menu: %v", err)
	}
	var count int64
	if err := db.Model(&model.RoleMenu{}).Where("menu_id = ?", menu.ID).Count(&count).Error; err != nil {
		t.Fatalf("count menu grants: %v", err)
	}
	if count != 0 {
		t.Fatalf("remaining menu grants = %d, want 0", count)
	}
}
