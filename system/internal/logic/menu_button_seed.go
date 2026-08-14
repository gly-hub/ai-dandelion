package logic

import (
	"context"
	"errors"

	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

func (m *MenuLogic) EnsureSeedButtonMenus(ctx context.Context) error {
	return seedButtonMenus(ctx, m.menuDao)
}

func seedButtonMenus(ctx context.Context, menuDao *dao.Menu) error {
	seeds := []seedMenu{
		{ParentCode: "func-operation.functions", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "新增功能", Code: "func-operation.functions.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 10, Remark: "功能管理页新增功能"},
		{ParentCode: "func-operation.functions", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "编辑功能", Code: "func-operation.functions.edit", ViewKey: "edit", MenuType: model.MenuTypeButton, Sort: 20, Remark: "进入功能设计台"},
		{ParentCode: "func-operation.functions", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "发布功能", Code: "func-operation.functions.publish", ViewKey: "publish", MenuType: model.MenuTypeButton, Sort: 30, Remark: "发布功能到使用列表"},
		{ParentCode: "func-operation.functions", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "下架功能", Code: "func-operation.functions.unpublish", ViewKey: "unpublish", MenuType: model.MenuTypeButton, Sort: 40, Remark: "将已发布功能下架"},
		{ParentCode: "func-operation.functions", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "删除功能", Code: "func-operation.functions.delete", ViewKey: "delete", MenuType: model.MenuTypeButton, Sort: 50, Remark: "删除功能记录"},
		{ParentCode: "func-operation.configs", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "查看配置", Code: "func-operation.configs.view", ViewKey: "view", MenuType: model.MenuTypeButton, Sort: 10, Remark: "查看公共配置"},
		{ParentCode: "func-operation.configs", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "新建配置", Code: "func-operation.configs.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 20, Remark: "新建公共配置"},
		{ParentCode: "func-operation.configs", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "更新配置", Code: "func-operation.configs.update", ViewKey: "update", MenuType: model.MenuTypeButton, Sort: 30, Remark: "上传和更新公共配置"},
		{ParentCode: "func-operation.configs", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "回滚配置", Code: "func-operation.configs.rollback", ViewKey: "rollback", MenuType: model.MenuTypeButton, Sort: 40, Remark: "回滚公共配置版本"},
		{ParentCode: "func-operation.external-apis", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "查看接口", Code: "func-operation.external-apis.view", ViewKey: "view", MenuType: model.MenuTypeButton, Sort: 10, Remark: "查看接口客户端与 API"},
		{ParentCode: "func-operation.external-apis", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "新建接口", Code: "func-operation.external-apis.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 20, Remark: "创建接口客户端和 API"},
		{ParentCode: "func-operation.external-apis", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "更新接口", Code: "func-operation.external-apis.update", ViewKey: "update", MenuType: model.MenuTypeButton, Sort: 30, Remark: "更新接口客户端和 API"},
		{ParentCode: "func-operation.upload-keys", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "查看密钥", Code: "func-operation.upload-keys.view", ViewKey: "view", MenuType: model.MenuTypeButton, Sort: 10, Remark: "查看上传密钥与使用说明"},
		{ParentCode: "func-operation.upload-keys", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "创建密钥", Code: "func-operation.upload-keys.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 20, Remark: "创建或轮换上传密钥"},
		{ParentCode: "func-operation.use", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "创建功能", Code: "func-operation.use.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 10, Remark: "功能使用页创建功能"},
		{ParentCode: "system.users", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "新建用户", Code: "system.users.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 10, Remark: "创建用户并分配角色"},
		{ParentCode: "system.users", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "编辑用户", Code: "system.users.update", ViewKey: "update", MenuType: model.MenuTypeButton, Sort: 20, Remark: "更新用户资料和角色"},
		{ParentCode: "system.users", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "启停用户", Code: "system.users.status", ViewKey: "status", MenuType: model.MenuTypeButton, Sort: 30, Remark: "启用或禁用用户"},
		{ParentCode: "system.users", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "删除用户", Code: "system.users.delete", ViewKey: "delete", MenuType: model.MenuTypeButton, Sort: 40, Remark: "删除用户"},
		{ParentCode: "system.menus", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "新建菜单", Code: "system.menus.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 10, Remark: "创建菜单或按钮权限"},
		{ParentCode: "system.menus", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "编辑菜单", Code: "system.menus.update", ViewKey: "update", MenuType: model.MenuTypeButton, Sort: 20, Remark: "更新菜单配置"},
		{ParentCode: "system.menus", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "启停菜单", Code: "system.menus.status", ViewKey: "status", MenuType: model.MenuTypeButton, Sort: 30, Remark: "启用或禁用菜单"},
		{ParentCode: "system.menus", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "删除菜单", Code: "system.menus.delete", ViewKey: "delete", MenuType: model.MenuTypeButton, Sort: 40, Remark: "删除菜单"},
		{ParentCode: "system.roles", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "新建角色", Code: "system.roles.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 10, Remark: "创建角色"},
		{ParentCode: "system.roles", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "编辑角色", Code: "system.roles.update", ViewKey: "update", MenuType: model.MenuTypeButton, Sort: 20, Remark: "更新角色资料"},
		{ParentCode: "system.roles", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "配置菜单权限", Code: "system.roles.permissions", ViewKey: "permissions", MenuType: model.MenuTypeButton, Sort: 30, Remark: "配置角色菜单和按钮权限"},
		{ParentCode: "system.roles", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "启停角色", Code: "system.roles.status", ViewKey: "status", MenuType: model.MenuTypeButton, Sort: 40, Remark: "启用或禁用角色"},
		{ParentCode: "system.roles", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "删除角色", Code: "system.roles.delete", ViewKey: "delete", MenuType: model.MenuTypeButton, Sort: 50, Remark: "删除角色"},
		{ParentCode: "system.agent-models", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "新建模型", Code: "system.agent-models.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 10, Remark: "创建模型配置"},
		{ParentCode: "system.agent-models", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "编辑模型", Code: "system.agent-models.update", ViewKey: "update", MenuType: model.MenuTypeButton, Sort: 20, Remark: "更新模型配置"},
		{ParentCode: "system.agent-models", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "启停模型", Code: "system.agent-models.status", ViewKey: "status", MenuType: model.MenuTypeButton, Sort: 30, Remark: "启用或禁用模型"},
		{ParentCode: "system.agent-models", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "删除模型", Code: "system.agent-models.delete", ViewKey: "delete", MenuType: model.MenuTypeButton, Sort: 40, Remark: "删除模型配置"},
		{ParentCode: "system.agent-config", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "保存配置", Code: "system.agent-config.save", ViewKey: "save", MenuType: model.MenuTypeButton, Sort: 10, Remark: "保存 Agent 系统配置"},
		{ParentCode: "system.agent-session-configs", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "刷新配置", Code: "system.agent-session-configs.refresh", ViewKey: "refresh", MenuType: model.MenuTypeButton, Sort: 10, Remark: "搭建会话配置页刷新列表"},
		{ParentCode: "system.agent-session-configs", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "保存配置", Code: "system.agent-session-configs.save", ViewKey: "save", MenuType: model.MenuTypeButton, Sort: 20, Remark: "搭建会话配置页保存配置"},
		{ParentCode: "system.notifications", Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "发送通知", Code: "system.notifications.create", ViewKey: "create", MenuType: model.MenuTypeButton, Sort: 10, Remark: "创建并发送通知"},
	}

	now := nowUnixMicro()
	for _, seed := range seeds {
		parent, err := menuDao.GetByCode(ctx, seed.ParentCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		if parent.ID == "" {
			continue
		}

		if existing, err := menuDao.GetByCode(ctx, seed.Code); err == nil {
			if existing.ParentID == parent.ID {
				continue
			}
			existing.ParentID = parent.ID
			existing.UpdatedAt = now
			if err := menuDao.Update(ctx, existing); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		id := buttonMenuID(seed.Code)
		menu := &model.Menu{
			ID:        id,
			ParentID:  parent.ID,
			Module:    seed.Module,
			Placement: seed.Placement,
			Name:      seed.Name,
			Code:      seed.Code,
			ViewKey:   seed.ViewKey,
			Icon:      seed.Icon,
			MenuType:  seed.MenuType,
			Sort:      seed.Sort,
			Status:    model.MenuStatusEnabled,
			Visible:   model.MenuVisibleYes,
			IsDefault: seed.IsDefault,
			Remark:    seed.Remark,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := menuDao.Create(ctx, menu); err != nil {
			return err
		}
	}
	return nil
}

func buttonMenuID(code string) string {
	switch code {
	case "func-operation.functions.create":
		return "00000000-0000-4000-8000-000000000041"
	case "func-operation.functions.edit":
		return "00000000-0000-4000-8000-000000000042"
	case "func-operation.functions.publish":
		return "00000000-0000-4000-8000-000000000043"
	case "func-operation.functions.unpublish":
		return "00000000-0000-4000-8000-000000000044"
	case "func-operation.functions.delete":
		return "00000000-0000-4000-8000-000000000045"
	case "func-operation.configs.view":
		return "00000000-0000-4000-8000-000000000049"
	case "func-operation.configs.create":
		return "00000000-0000-4000-8000-000000000050"
	case "func-operation.configs.update":
		return "00000000-0000-4000-8000-000000000051"
	case "func-operation.configs.rollback":
		return "00000000-0000-4000-8000-000000000052"
	case "func-operation.external-apis.view":
		return "00000000-0000-4000-8000-000000000053"
	case "func-operation.external-apis.create":
		return "00000000-0000-4000-8000-000000000054"
	case "func-operation.external-apis.update":
		return "00000000-0000-4000-8000-000000000055"
	case "func-operation.upload-keys.view":
		return "00000000-0000-4000-8000-000000000056"
	case "func-operation.upload-keys.create":
		return "00000000-0000-4000-8000-000000000057"
	case "func-operation.use.create":
		return "00000000-0000-4000-8000-000000000046"
	case "system.users.create":
		return "00000000-0000-4000-8000-000000000058"
	case "system.users.update":
		return "00000000-0000-4000-8000-000000000059"
	case "system.users.status":
		return "00000000-0000-4000-8000-000000000060"
	case "system.users.delete":
		return "00000000-0000-4000-8000-000000000061"
	case "system.menus.create":
		return "00000000-0000-4000-8000-000000000062"
	case "system.menus.update":
		return "00000000-0000-4000-8000-000000000063"
	case "system.menus.status":
		return "00000000-0000-4000-8000-000000000064"
	case "system.menus.delete":
		return "00000000-0000-4000-8000-000000000065"
	case "system.roles.create":
		return "00000000-0000-4000-8000-000000000066"
	case "system.roles.update":
		return "00000000-0000-4000-8000-000000000067"
	case "system.roles.permissions":
		return "00000000-0000-4000-8000-000000000068"
	case "system.roles.status":
		return "00000000-0000-4000-8000-000000000069"
	case "system.roles.delete":
		return "00000000-0000-4000-8000-000000000070"
	case "system.agent-models.create":
		return "00000000-0000-4000-8000-000000000071"
	case "system.agent-models.update":
		return "00000000-0000-4000-8000-000000000072"
	case "system.agent-models.status":
		return "00000000-0000-4000-8000-000000000073"
	case "system.agent-models.delete":
		return "00000000-0000-4000-8000-000000000074"
	case "system.agent-config.save":
		return "00000000-0000-4000-8000-000000000075"
	case "system.agent-session-configs.refresh":
		return "00000000-0000-4000-8000-000000000047"
	case "system.agent-session-configs.save":
		return "00000000-0000-4000-8000-000000000048"
	case "system.notifications.create":
		return "00000000-0000-4000-8000-000000000076"
	default:
		return ""
	}
}
