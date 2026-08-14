package logic

import (
	"context"
	"testing"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLookupGeneratedFunctionMenuIgnoresButtonRows(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Menu{}); err != nil {
		t.Fatalf("migrate menu table: %v", err)
	}

	functionID := "func-123"
	parentID := "parent-1"
	button := &model.Menu{
		ID:         "btn-1",
		ParentID:   parentID,
		Module:     model.MenuModuleFuncOperation,
		Placement:  model.MenuPlacementModuleNav,
		Name:       "删除",
		Code:       "func-operation.app.func-123.book_delete",
		ViewKey:    "book_delete",
		MenuType:   model.MenuTypeButton,
		Status:     model.MenuStatusEnabled,
		Visible:    model.MenuVisibleYes,
		SourceType: model.MenuSourceTypeGeneratedFunction,
		SourceID:   functionID,
		CreatedAt:  1,
		UpdatedAt:  1,
	}
	menu := &model.Menu{
		ID:         "menu-1",
		ParentID:   parentID,
		Module:     model.MenuModuleFuncOperation,
		Placement:  model.MenuPlacementModuleNav,
		Name:       "图书管理",
		Code:       functionMenuCode(functionID),
		ViewKey:    functionMenuViewKey(functionID),
		MenuType:   model.MenuTypeMenu,
		Status:     model.MenuStatusEnabled,
		Visible:    model.MenuVisibleYes,
		SourceType: model.MenuSourceTypeGeneratedFunction,
		SourceID:   functionID,
		CreatedAt:  2,
		UpdatedAt:  2,
	}
	if err := db.Create(button).Error; err != nil {
		t.Fatalf("create button menu: %v", err)
	}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("create function menu: %v", err)
	}

	logic := &MenuLogic{
		menuDao: dao.NewMenu(db),
	}
	found, err := logic.lookupGeneratedFunctionMenu(context.Background(), functionID)
	if err != nil {
		t.Fatalf("lookupGeneratedFunctionMenu() error = %v", err)
	}
	if found == nil || found.ID != menu.ID {
		t.Fatalf("expected function menu %q, got %#v", menu.ID, found)
	}
	if found.MenuType != model.MenuTypeMenu {
		t.Fatalf("expected menu type %d, got %d", model.MenuTypeMenu, found.MenuType)
	}
}

func TestGeneratedFunctionMenuRemarkIncludesControlledActions(t *testing.T) {
	t.Parallel()

	got := generatedFunctionMenuRemark([]string{"book_delete", "book_create", "book_delete"})
	want := "功能发布自动创建; actions:book_create,book_delete"
	if got != want {
		t.Fatalf("generatedFunctionMenuRemark() = %q, want %q", got, want)
	}
}

func TestSyncGeneratedFunctionMenuGrantsAdminItsPageAndActionPermissions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Menu{}, &model.Role{}, &model.RoleMenu{}, &model.UserRole{}); err != nil {
		t.Fatalf("migrate system tables: %v", err)
	}

	menuDao := dao.NewMenu(db)
	roleDao := dao.NewRole(db)
	admin := &model.Role{ID: "admin", Name: "管理员", Code: model.RoleCodeAdmin, Status: model.RoleStatusEnabled, CreatedAt: 1, UpdatedAt: 1}
	if err := roleDao.Create(context.Background(), admin); err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	useMenu := &model.Menu{
		ID:     "func-use",
		Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav,
		Name: "功能使用", Code: model.FuncUseMenuCode, MenuType: model.MenuTypeDirectory,
		Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := menuDao.Create(context.Background(), useMenu); err != nil {
		t.Fatalf("create function use menu: %v", err)
	}

	logic := NewMenuLogic(menuDao, roleDao)
	if _, err := logic.SyncGeneratedFunctionMenu(context.Background(), &systemproto.SyncGeneratedFunctionMenuReq{
		FunctionId: "books", Name: "图书管理", ParentId: useMenu.ID, Action: generatedFunctionMenuActionPublish,
		ActionKeys: []string{"book_create", "book_delete"},
	}); err != nil {
		t.Fatalf("sync generated function menu: %v", err)
	}

	menuIDs, err := roleDao.ListMenuIDsByRole(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("list admin menu IDs: %v", err)
	}
	if len(menuIDs) != 3 {
		t.Fatalf("admin menu IDs = %v, want generated page and two actions", menuIDs)
	}
	if err := db.Create(&model.UserRole{UserID: "admin-user", RoleID: admin.ID, CreatedAt: 1}).Error; err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	allowed, err := logic.CheckFunctionMenuAccess(context.Background(), &systemproto.CheckFunctionMenuAccessReq{UserId: "admin-user", FunctionId: "books"})
	if err != nil || !allowed {
		t.Fatalf("admin page access = %v, %v; want true, nil", allowed, err)
	}
	allowed, err = logic.CheckFunctionMenuAccess(context.Background(), &systemproto.CheckFunctionMenuAccessReq{UserId: "admin-user", FunctionId: "books", ActionKey: "book_delete"})
	if err != nil || !allowed {
		t.Fatalf("admin action access = %v, %v; want true, nil", allowed, err)
	}
}
