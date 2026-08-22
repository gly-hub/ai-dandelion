package logic

import (
	"context"
	"errors"
	"strings"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

const (
	legacyFuncGenerationSystemPrompt = "你是功能页面生成助手。请根据已确认的产品方案和技术方案生成可运行页面，优先保证业务动作完整、布局稳定、交互清晰。"
	funcGenerationSystemPrompt       = "你是功能页面生成助手。请根据已确认的产品方案和技术方案生成可运行、可操作的业务页面，优先保证业务动作完整、布局稳定、交互清晰。页面必须同时适配 390x844 移动端、768x1024 平板和桌面宽度：使用 width: 100%、max-width、minmax(0, 1fr) 和合理的媒体查询，禁止整个页面依赖固定宽度、固定最小宽度或产生横向溢出；移动端表单应单列排列，按钮触控高度至少 40px，文字不得重叠或裁切；表格在移动端必须可横向滚动且操作列可达，或转换为列表/卡片；弹层和抽屉必须限制最大宽高并支持内部滚动。页面根容器需要在 iframe 内处理纵向滚动，不能依赖父页面。业务请求只能使用 context.invokeData 或 context.invoke，不能直接使用 fetch。输出应是可操作业务页面，不是产品或技术方案说明。"
)

type AgentSessionConfigLogic struct {
	agentSessionConfigDao *dao.AgentSessionConfig
}

func NewAgentSessionConfigLogic(agentSessionConfigDao *dao.AgentSessionConfig) *AgentSessionConfigLogic {
	return &AgentSessionConfigLogic{agentSessionConfigDao: agentSessionConfigDao}
}

func (a *AgentSessionConfigLogic) ListAgentSessionConfigs(ctx context.Context, _ *systemproto.ListAgentSessionConfigsReq) ([]*systemproto.AgentSessionConfig, error) {
	items, err := a.agentSessionConfigDao.List(ctx)
	if err != nil {
		return nil, err
	}
	itemMap := make(map[string]*model.AgentSessionConfig, len(items))
	for i := range items {
		itemMap[items[i].SessionType] = &items[i]
	}
	out := make([]*systemproto.AgentSessionConfig, 0, len(defaultAgentSessionConfigs()))
	for _, defaults := range defaultAgentSessionConfigs() {
		item := defaults
		if stored := itemMap[defaults.SessionType]; stored != nil {
			item = *stored
		}
		out = append(out, modelAgentSessionConfigToProto(&item))
	}
	return out, nil
}

func (a *AgentSessionConfigLogic) UpdateAgentSessionConfig(ctx context.Context, req *systemproto.UpdateAgentSessionConfigReq) (*systemproto.AgentSessionConfig, error) {
	defaults, ok := defaultAgentSessionConfig(req.GetSessionType())
	if !ok {
		return nil, errors.New("unsupported agent session config type")
	}
	now := nowUnixMicro()
	item, err := a.agentSessionConfigDao.Get(ctx, defaults.SessionType)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		item = &model.AgentSessionConfig{
			SessionType: defaults.SessionType,
			Name:        defaults.Name,
			Description: defaults.Description,
			CreatedAt:   now,
		}
	}
	item.Name = defaults.Name
	item.Description = defaults.Description
	item.SystemPrompt = strings.TrimSpace(req.GetSystemPrompt())
	if item.SystemPrompt == "" {
		item.SystemPrompt = defaults.SystemPrompt
	}
	item.PermissionMode = strings.TrimSpace(req.GetPermissionMode())
	if item.PermissionMode == "" {
		item.PermissionMode = defaults.PermissionMode
	}
	item.MaxTurns = int(req.GetMaxTurns())
	if item.MaxTurns <= 0 {
		item.MaxTurns = defaults.MaxTurns
	}
	item.ModelID = strings.TrimSpace(req.GetModelId())
	item.Enabled = req.GetEnabled()
	item.UpdatedAt = now
	if err := a.agentSessionConfigDao.Save(ctx, item); err != nil {
		return nil, err
	}
	return modelAgentSessionConfigToProto(item), nil
}

func (a *AgentSessionConfigLogic) EnsureSeedAgentSessionConfigs(ctx context.Context) error {
	now := nowUnixMicro()
	for _, defaults := range defaultAgentSessionConfigs() {
		item, err := a.agentSessionConfigDao.Get(ctx, defaults.SessionType)
		if err == nil {
			if defaults.SessionType == model.AgentSessionConfigTypeFuncGeneration && item.SystemPrompt == legacyFuncGenerationSystemPrompt {
				item.SystemPrompt = defaults.SystemPrompt
				item.UpdatedAt = now
				if err := a.agentSessionConfigDao.Save(ctx, item); err != nil {
					return err
				}
			}
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		defaults.CreatedAt = now
		defaults.UpdatedAt = now
		if err := a.agentSessionConfigDao.Save(ctx, &defaults); err != nil {
			return err
		}
	}
	return nil
}

func defaultAgentSessionConfig(sessionType string) (model.AgentSessionConfig, bool) {
	for _, item := range defaultAgentSessionConfigs() {
		if item.SessionType == strings.TrimSpace(sessionType) {
			return item, true
		}
	}
	return model.AgentSessionConfig{}, false
}

func defaultAgentSessionConfigs() []model.AgentSessionConfig {
	return []model.AgentSessionConfig{
		{
			SessionType:    model.AgentSessionConfigTypeFuncProduct,
			Name:           "产品方案",
			Description:    "用于功能搭建的产品目标、页面范围与业务流程梳理。",
			SystemPrompt:   "你是产品方案生成助手。请围绕用户要搭建的功能，输出清晰的目标、页面范围、字段与业务流程，并保持可落地。",
			PermissionMode: "bypassPermissions",
			MaxTurns:       20,
			Enabled:        true,
		},
		{
			SessionType:    model.AgentSessionConfigTypeFuncTechnical,
			Name:           "技术方案",
			Description:    "用于基于产品方案设计数据模型、接口契约和实现计划。",
			SystemPrompt:   "你是技术方案生成助手。请基于已确认的产品方案，输出数据模型、接口设计、权限动作和实现步骤，避免引入不必要复杂度。",
			PermissionMode: "bypassPermissions",
			MaxTurns:       20,
			Enabled:        true,
		},
		{
			SessionType:    model.AgentSessionConfigTypeFuncGeneration,
			Name:           "页面生成",
			Description:    "用于根据产品与技术方案生成、修复和刷新可操作页面。",
			SystemPrompt:   funcGenerationSystemPrompt,
			PermissionMode: "bypassPermissions",
			MaxTurns:       30,
			Enabled:        true,
		},
	}
}

func modelAgentSessionConfigToProto(item *model.AgentSessionConfig) *systemproto.AgentSessionConfig {
	if item == nil {
		return &systemproto.AgentSessionConfig{}
	}
	return &systemproto.AgentSessionConfig{
		SessionType:    item.SessionType,
		Name:           item.Name,
		Description:    item.Description,
		SystemPrompt:   item.SystemPrompt,
		PermissionMode: item.PermissionMode,
		MaxTurns:       int32(item.MaxTurns),
		ModelId:        item.ModelID,
		Enabled:        item.Enabled,
		UpdatedAt:      item.UpdatedAt,
	}
}
