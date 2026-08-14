package logic

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
)

type seedMenu struct {
	ParentCode string
	Module     string
	Placement  string
	Name       string
	Code       string
	ViewKey    string
	Icon       string
	MenuType   int
	Sort       int
	IsDefault  bool
	Remark     string
}

func seedMenus(ctx context.Context, menuDao *dao.Menu) error {
	seeds := []seedMenu{
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "功能库", Code: "func-operation.use", ViewKey: "published", Icon: "RocketOutlined", MenuType: model.MenuTypeDirectory, Sort: 10, IsDefault: true, Remark: "使用已发布功能"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, Name: "后台管理", Code: "console.manager", ViewKey: "console.manager", Icon: "SettingOutlined", MenuType: model.MenuTypeDirectory, Sort: 30, Remark: "统一管理功能与系统配置"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "console.manager", Name: "功能管理", Code: "console.func.manager", ViewKey: "console.func.manager", Icon: "ToolOutlined", MenuType: model.MenuTypeDirectory, Sort: 10, Remark: "管理 AI 生成的功能"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "console.func.manager", Name: "功能列表", Code: "func-operation.functions", ViewKey: "admin-functions", Icon: "SettingOutlined", MenuType: model.MenuTypeMenu, Sort: 10, Remark: "管理 AI 生成的功能"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "console.func.manager", Name: "公共配置", Code: "func-operation.configs", ViewKey: "public-configs", Icon: "DatabaseOutlined", MenuType: model.MenuTypeMenu, Sort: 20, Remark: "维护功能复用的动态选项配置"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "console.func.manager", Name: "接口管理", Code: "func-operation.external-apis", ViewKey: "external-api-clients", Icon: "ApiOutlined", MenuType: model.MenuTypeMenu, Sort: 30, Remark: "维护外部接口客户端及其 API 定义"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "console.func.manager", Name: "上传密钥", Code: "func-operation.upload-keys", ViewKey: "upload-keys", Icon: "KeyOutlined", MenuType: model.MenuTypeMenu, Sort: 40, Remark: "管理 Swagger 文档与公共配置上传密钥"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "console.manager", Name: "系统管理", Code: "system.manager", ViewKey: "system.manager", Icon: "SettingOutlined", MenuType: model.MenuTypeDirectory, Sort: 20, Remark: "管理系统账号、权限与 Agent 配置"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.manager", Name: "用户管理", Code: "system.users", ViewKey: "users", Icon: "TeamOutlined", MenuType: model.MenuTypeMenu, Sort: 10, IsDefault: true, Remark: "维护系统用户账号"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.manager", Name: "菜单管理", Code: "system.menus", ViewKey: "menus", Icon: "MenuOutlined", MenuType: model.MenuTypeMenu, Sort: 20, Remark: "配置全平台导航菜单"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.manager", Name: "角色管理", Code: "system.roles", ViewKey: "roles", Icon: "SafetyOutlined", MenuType: model.MenuTypeMenu, Sort: 30, Remark: "配置角色与菜单权限"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.manager", Name: "操作记录", Code: "system.operation-logs", ViewKey: "operation-logs", Icon: "HistoryOutlined", MenuType: model.MenuTypeMenu, Sort: 40, Remark: "查询平台操作与资源变更记录"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.manager", Name: "通知管理", Code: "system.notifications", ViewKey: "notifications", Icon: "BellOutlined", MenuType: model.MenuTypeMenu, Sort: 50, Remark: "发送弹窗和气泡通知"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "console.manager", Name: "AI Agent 管理", Code: "system.agent-admin", ViewKey: "", Icon: "RobotOutlined", MenuType: model.MenuTypeDirectory, Sort: 30, Remark: "管理 Agent 系统配置与模型"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.agent-admin", Name: "系统配置", Code: "system.agent-config", ViewKey: "agent-config", Icon: "SettingOutlined", MenuType: model.MenuTypeMenu, Sort: 10, Remark: "配置 Agent 系统提示词与运行参数"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.agent-admin", Name: "模型配置", Code: "system.agent-models", ViewKey: "agent-models", Icon: "ApiOutlined", MenuType: model.MenuTypeMenu, Sort: 20, Remark: "管理 Agent 可用模型与思考参数"},
		{Module: model.MenuModuleFuncOperation, Placement: model.MenuPlacementModuleNav, ParentCode: "system.agent-admin", Name: "搭建会话配置", Code: "system.agent-session-configs", ViewKey: "agent-session-configs", Icon: "MessageOutlined", MenuType: model.MenuTypeMenu, Sort: 30, Remark: "配置功能搭建不同会话的模型与提示词"},
	}

	codeToID := make(map[string]string, len(seeds))
	now := nowUnixMicro()
	for _, seed := range seeds {
		if existing, err := menuDao.GetByCode(ctx, seed.Code); err == nil {
			expectedID := newMenuID(seed.Code)
			if existing.ID == "" && expectedID != "" {
				if err := menuDao.ReassignID(ctx, existing.ID, expectedID); err != nil {
					return err
				}
				existing.ID = expectedID
			}
			codeToID[seed.Code] = existing.ID
			parentID := ""
			if seed.ParentCode != "" {
				parentID = codeToID[seed.ParentCode]
				if parentID == "" {
					if parent, parentErr := menuDao.GetByCode(ctx, seed.ParentCode); parentErr == nil {
						parentID = parent.ID
						codeToID[seed.ParentCode] = parent.ID
					}
				}
			}
			if seed.ParentCode != "" && parentID == "" {
				continue
			}
			existing.ParentID = parentID
			existing.Module = seed.Module
			existing.Placement = seed.Placement
			existing.Name = seed.Name
			existing.ViewKey = seed.ViewKey
			existing.Icon = seed.Icon
			existing.MenuType = seed.MenuType
			existing.Sort = seed.Sort
			existing.IsDefault = seed.IsDefault
			existing.Remark = seed.Remark
			existing.UpdatedAt = now
			if err := menuDao.Update(ctx, existing); err != nil {
				return err
			}
			continue
		}
		parentID := ""
		if seed.ParentCode != "" {
			parentID = codeToID[seed.ParentCode]
			if parentID == "" {
				if parent, err := menuDao.GetByCode(ctx, seed.ParentCode); err == nil {
					parentID = parent.ID
					codeToID[seed.ParentCode] = parent.ID
				}
			}
		}
		if seed.ParentCode != "" && parentID == "" {
			continue
		}
		id := codeToID[seed.Code]
		if id == "" {
			id = newMenuID(seed.Code)
		}
		menu := &model.Menu{
			ID:        id,
			ParentID:  parentID,
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
		codeToID[seed.Code] = menu.ID
	}
	return nil
}

func newMenuID(code string) string {
	// Stable ids for seed rows so re-seed attempts remain predictable in dev.
	switch code {
	case "platform.system":
		return "00000000-0000-4000-8000-000000000001"
	case "platform.func-operation":
		return "00000000-0000-4000-8000-000000000002"
	case "platform.ai-agent":
		return "00000000-0000-4000-8000-000000000003"
	case "system.users":
		return "00000000-0000-4000-8000-000000000011"
	case "system.menus":
		return "00000000-0000-4000-8000-000000000012"
	case "system.roles":
		return "00000000-0000-4000-8000-000000000013"
	case "system.agent-admin":
		return "00000000-0000-4000-8000-000000000015"
	case "system.agent-config":
		return "00000000-0000-4000-8000-000000000016"
	case "system.agent-models":
		return "00000000-0000-4000-8000-000000000014"
	case "system.agent-session-configs":
		return "00000000-0000-4000-8000-000000000017"
	case "system.operation-logs":
		return "00000000-0000-4000-8000-000000000018"
	case "system.notifications":
		return "00000000-0000-4000-8000-000000000019"
	case "ai-agent.chat":
		return "00000000-0000-4000-8000-000000000021"
	case "ai-agent.toolbox":
		return "00000000-0000-4000-8000-000000000022"
	case "ai-agent.my-space":
		return "00000000-0000-4000-8000-000000000023"
	case "func-operation.use":
		return "00000000-0000-4000-8000-000000000031"
	case "func-operation.functions":
		return "00000000-0000-4000-8000-000000000032"
	case "func-operation.configs":
		return "00000000-0000-4000-8000-000000000036"
	case "func-operation.external-apis":
		return "00000000-0000-4000-8000-000000000037"
	case "func-operation.upload-keys":
		return "00000000-0000-4000-8000-000000000038"
	case "console.manager":
		return "00000000-0000-4000-8000-000000000033"
	case "console.func.manager":
		return "00000000-0000-4000-8000-000000000034"
	case "system.manager":
		return "00000000-0000-4000-8000-000000000035"
	default:
		return ""
	}
}
