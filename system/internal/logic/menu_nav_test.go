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

func TestBuildUnifiedNavTree(t *testing.T) {
	items := []*systemproto.Menu{
		{Id: "1", Module: model.MenuModuleSystem, Placement: model.MenuPlacementPlatform, Name: "系统", ViewKey: "system", Sort: 10},
		{Id: "2", Module: model.MenuModuleAIAgent, Placement: model.MenuPlacementPlatform, Name: "Agent", ViewKey: "ai-agent", Sort: 30},
		{Id: "3", Module: model.MenuModuleSystem, Placement: model.MenuPlacementModuleNav, Name: "用户管理", ViewKey: "users", Sort: 10},
		{Id: "4", Module: model.MenuModuleAIAgent, Placement: model.MenuPlacementModuleNav, Name: "对话", ViewKey: "chat", Sort: 10},
		{Id: "5", Module: model.MenuModuleAIAgent, Placement: model.MenuPlacementModuleNav, Name: "工具集", ViewKey: "toolbox", Sort: 20},
	}

	tree := buildUnifiedNavTree(items)
	if len(tree) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree))
	}
	if tree[0].ViewKey != "system" {
		t.Fatalf("expected first root system, got %s", tree[0].ViewKey)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].ViewKey != "users" {
		t.Fatalf("unexpected system children: %+v", tree[0].Children)
	}
	if tree[1].ViewKey != "ai-agent" {
		t.Fatalf("expected second root ai-agent, got %s", tree[1].ViewKey)
	}
	if len(tree[1].Children) != 2 {
		t.Fatalf("expected 2 ai-agent children, got %d", len(tree[1].Children))
	}
}

func TestFilterNavTreeByModule(t *testing.T) {
	tree := []*systemproto.Menu{
		{
			Id:       "1",
			Module:   model.MenuModuleAIAgent,
			ViewKey:  "ai-agent",
			Name:     "Agent",
			Children: []*systemproto.Menu{{Id: "2", ViewKey: "chat", Name: "对话"}},
		},
	}

	filtered := filterNavTreeByModule(tree, model.MenuModuleAIAgent)
	if len(filtered) != 1 || len(filtered[0].Children) != 1 {
		t.Fatalf("unexpected filtered tree: %+v", filtered)
	}
}

func TestGetNavMenusReturnsOnlyMenusAssignedToUser(t *testing.T) {
	logic, db := newMenuNavigationLogic(t)
	role := &model.Role{ID: "role-1", Name: "测试", Code: "tester", Status: model.RoleStatusEnabled}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: "user-1", RoleID: role.ID}).Error; err != nil {
		t.Fatalf("assign role: %v", err)
	}
	platform := &model.Menu{ID: "platform", Module: model.MenuModuleSystem, Placement: model.MenuPlacementPlatform, Name: "后台管理", Code: "platform.system", ViewKey: "system", MenuType: model.MenuTypeMenu, Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes}
	allowedMenu := &model.Menu{ID: "allowed", ParentID: "root", Module: model.MenuModuleSystem, Placement: model.MenuPlacementModuleNav, Name: "用户管理", Code: "system.users", ViewKey: "users", MenuType: model.MenuTypeMenu, Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes}
	deniedMenu := &model.Menu{ID: "denied", ParentID: "root", Module: model.MenuModuleSystem, Placement: model.MenuPlacementModuleNav, Name: "通知管理", Code: "system.notifications", ViewKey: "notifications", MenuType: model.MenuTypeMenu, Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes}
	directory := &model.Menu{ID: "root", Module: model.MenuModuleSystem, Placement: model.MenuPlacementModuleNav, Name: "系统管理", Code: "system.manager", MenuType: model.MenuTypeDirectory, Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes}
	for _, menu := range []*model.Menu{platform, directory, allowedMenu, deniedMenu} {
		if err := db.Create(menu).Error; err != nil {
			t.Fatalf("create %s: %v", menu.ID, err)
		}
	}
	for _, menuID := range []string{directory.ID, allowedMenu.ID} {
		if err := db.Create(&model.RoleMenu{RoleID: role.ID, MenuID: menuID}).Error; err != nil {
			t.Fatalf("grant %s: %v", menuID, err)
		}
	}

	menus, err := logic.GetNavMenus(context.Background(), &systemproto.GetNavMenusReq{UserId: "user-1"})
	if err != nil {
		t.Fatalf("get nav menus: %v", err)
	}
	if len(menus) != 1 || len(menus[0].Children) != 1 || len(menus[0].Children[0].Children) != 1 {
		t.Fatalf("unexpected navigation tree: %#v", menus)
	}
	if got := menus[0].Children[0].Children[0].GetCode(); got != allowedMenu.Code {
		t.Fatalf("visible menu = %q, want %q", got, allowedMenu.Code)
	}
}

func TestGetNavMenusWithoutUserReturnsNoMenus(t *testing.T) {
	logic, db := newMenuNavigationLogic(t)
	if err := db.Create(&model.Menu{ID: "platform", Module: model.MenuModuleSystem, Placement: model.MenuPlacementPlatform, Name: "后台管理", Code: "platform.system", ViewKey: "system", MenuType: model.MenuTypeMenu, Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes}).Error; err != nil {
		t.Fatalf("create menu: %v", err)
	}

	menus, err := logic.GetNavMenus(context.Background(), &systemproto.GetNavMenusReq{})
	if err != nil {
		t.Fatalf("get nav menus: %v", err)
	}
	if len(menus) != 0 {
		t.Fatalf("menus without a user = %#v, want none", menus)
	}
}

func newMenuNavigationLogic(t *testing.T) (*MenuLogic, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Menu{}, &model.Role{}, &model.UserRole{}, &model.RoleMenu{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return NewMenuLogic(dao.NewMenu(db), dao.NewRole(db)), db
}
