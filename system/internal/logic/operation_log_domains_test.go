package logic

import (
	"context"
	"testing"
	"time"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserAndMenuLogicRecordOperations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.UserRole{}, &model.Menu{}, &model.OperationLog{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	userDao := dao.NewUser(db)
	roleDao := dao.NewRole(db)
	menuDao := dao.NewMenu(db)
	operationLogDao := dao.NewOperationLog(db)
	operationLogs := NewOperationLogLogic(operationLogDao)
	userLogic := NewUserLogic(userDao, roleDao, "test-secret", time.Hour, operationLogs)
	menuLogic := NewMenuLogic(menuDao, roleDao, operationLogs)
	ctx := authctx.ContextWithUser(context.Background(), authctx.User{ID: "admin-1", Username: "admin"})

	user, err := userLogic.CreateUser(ctx, &systemproto.CreateUserReq{
		Username: "member", Email: "member@example.com", Password: "password", Status: model.UserStatusEnabled,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	menu, err := menuLogic.CreateMenu(ctx, &systemproto.CreateMenuReq{
		Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav,
		Name: "测试菜单", Code: "test.menu", ViewKey: "test-menu", MenuType: model.MenuTypeMenu,
		Status: model.MenuStatusEnabled, Visible: model.MenuVisibleYes,
	})
	if err != nil {
		t.Fatalf("create menu: %v", err)
	}

	for _, expected := range []struct {
		resourceType string
		resourceID   string
	}{
		{OperationResourceUser, user.Id},
		{OperationResourceMenu, menu.Id},
	} {
		logs, total, err := operationLogDao.List(ctx, dao.OperationLogListFilter{
			ResourceType: expected.resourceType, ResourceID: expected.resourceID, Page: 1, PageSize: 20,
		})
		if err != nil {
			t.Fatalf("list %s logs: %v", expected.resourceType, err)
		}
		if total != 1 || len(logs) != 1 {
			t.Fatalf("%s log count = %d/%d, want 1", expected.resourceType, len(logs), total)
		}
		if logs[0].OperatorName != "admin" || logs[0].AfterData == "{}" {
			t.Fatalf("unexpected %s log: %#v", expected.resourceType, logs[0])
		}
	}
}
