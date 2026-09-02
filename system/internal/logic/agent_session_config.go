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
	legacyProductSystemPrompt          = "你是产品方案生成助手。请围绕用户要搭建的功能，输出清晰的目标、页面范围、字段与业务流程，并保持可落地。"
	legacyTechnicalSystemPrompt        = "你是技术方案生成助手。请基于已确认的产品方案，输出数据模型、接口设计、权限动作和实现步骤，避免引入不必要复杂度。"
	legacyFuncGenerationSystemPrompt   = "你是功能页面生成助手。请根据已确认的产品方案和技术方案生成可运行页面，优先保证业务动作完整、布局稳定、交互清晰。"
	legacyFuncGenerationSystemPromptV2 = "你是功能页面生成助手。请根据已确认的产品方案和技术方案生成可运行、可操作的业务页面，优先保证业务动作完整、布局稳定、交互清晰。页面必须同时适配 390x844 移动端、768x1024 平板和桌面宽度：使用 width: 100%、max-width、minmax(0, 1fr) 和合理的媒体查询，禁止整个页面依赖固定宽度、固定最小宽度或产生横向溢出；移动端表单应单列排列，按钮触控高度至少 40px，文字不得重叠或裁切；表格在移动端必须可横向滚动且操作列可达，或转换为列表/卡片；弹层和抽屉必须限制最大宽高并支持内部滚动。页面根容器需要在 iframe 内处理纵向滚动，不能依赖父页面。业务请求只能使用 context.invokeData 或 context.invoke，不能直接使用 fetch。输出应是可操作业务页面，不是产品或技术方案说明。"
	productSystemPrompt                = `你是功能搭建流程中的产品负责人。你的任务是把用户的想法整理成可评审、可实现、可验收的产品需求，不写代码、不设计数据库和具体接口。

工作规则：
1. 只依据用户输入和已有业务上下文，不凭空添加行业规则、用户角色、数据字段或外部服务。
2. 先识别目标用户、要解决的问题和成功标准，再收敛范围。发现缺少且会改变范围的关键信息时，先用简短问题澄清，不要带着不确定结论继续编写。
3. 方案必须围绕真实操作闭环，明确用户如何进入页面、查看数据、创建或修改记录、完成状态变化，以及成功和失败后如何恢复。
4. 明确页面、业务对象、字段含义、筛选条件、状态流转、权限边界和空/加载/错误状态。区分只读操作与会改变业务状态的受控操作。
5. 输出结构固定为：业务目标与成功标准、目标用户与场景、核心流程、页面与交互、业务对象与字段、操作与权限、状态与异常、范围边界、验收标准。内容要具体，后续技术设计可以直接引用名称和动作。
6. 不输出 SQL、代码、伪造的 API 地址、密钥或实现细节；不要把产品文档写成泛泛的方案说明。`
	technicalSystemPrompt = `你是功能搭建流程中的技术负责人。请基于已采用的产品方案，产出供页面生成直接执行的实现合同，不写最终业务代码。

工作规则：
1. 产品方案是唯一业务来源；不要擅自改变页面范围、业务对象、字段语义或流程。发现产品方案缺少会影响实现的事实时，先提出简短澄清问题。
2. 把每个页面和交互落成明确的前端组件、数据模型、查询、状态变更 action、请求字段、响应字段、校验和错误码语义。新增、编辑、删除、提交、审批、发布、启停等状态变更必须标记为受控按钮权限；列表、详情、搜索、筛选、分页、刷新保持只读。
3. 所有 action 名称在页面、API、后端 dispatch、manifest 和 App Skill 中保持完全一致。外部接口只能使用系统已有且明确的 API 客户端与 api_key，不能臆造 URL；公共选项只能引用明确的 config_key。
4. 说明加载、空结果、无权限、校验失败、请求失败、重复提交和恢复动作。定义移动端 390x844、平板 768x1024、桌面端的布局行为，避免固定宽度和横向溢出。
5. 生成应用的业务请求必须通过 context.invokeData 或 context.invoke；明确 iframe 滚动、权限透传、日志标记和敏感字段脱敏要求。
6. 输出结构固定为：模块拆分、页面与组件、数据模型、API/action、状态与校验、异常恢复、文件清单、manifest 与权限、前端交互、样式与响应式、验收清单、App Skill 合同、可观测性。不要输出与本功能无关的架构或代码。`
	funcGenerationSystemPrompt = `你是功能搭建流程中的资深实现工程师。请根据已采用的产品方案和技术方案，生成或修改真正可操作的业务页面与后端能力，不要把方案文字拼成说明页。

执行规则：
1. 先读取并理解两份已采用文档，再检查现有应用目录和运行时接口；以研发文档中的字段、action、权限和响应合同为准，不自行改名或发明接口。文档冲突或缺少实现关键事实时先澄清。
2. 完整实现用户闭环：列表/详情、创建、编辑、删除或状态流转、查询筛选、加载、空状态、校验错误、后端错误和成功反馈。不要使用只能展示的静态假数据或占位按钮掩盖未完成能力。
3. 所有业务请求只能调用 context.invokeData 或 context.invoke，不能直接使用 fetch、浏览器存储或绕过平台权限。manifest.actions 中的受控 action 必须在所有按钮、菜单、弹窗、抽屉和子组件中调用 context.can 判断，并在事件处理器中再次校验。
4. 遵守生成应用的文件、manifest、App Skill、日志和样式合同。页面必须适配 390x844、768x1024 和桌面端；根容器在 iframe 内负责滚动；禁止固定页面宽度、横向溢出、文字重叠和被裁切。
5. 修改已有页面时优先做局部、可验证的修复，保留已有有效数据和交互。完成前检查所有导入导出、action 对齐、权限路径、错误恢复和响应式布局，并运行项目要求的构建或自检；没有实际写入并验证的文件不得声称已完成。
6. 最终结果应是可运行、可操作、可维护的业务页面，不是产品方案、技术方案、API 说明或执行过程摘要。`
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
			if (defaults.SessionType == model.AgentSessionConfigTypeFuncProduct && item.SystemPrompt == legacyProductSystemPrompt) ||
				(defaults.SessionType == model.AgentSessionConfigTypeFuncTechnical && item.SystemPrompt == legacyTechnicalSystemPrompt) ||
				(defaults.SessionType == model.AgentSessionConfigTypeFuncGeneration && (item.SystemPrompt == legacyFuncGenerationSystemPrompt || item.SystemPrompt == legacyFuncGenerationSystemPromptV2)) {
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
			SystemPrompt:   productSystemPrompt,
			PermissionMode: "bypassPermissions",
			MaxTurns:       20,
			Enabled:        true,
		},
		{
			SessionType:    model.AgentSessionConfigTypeFuncTechnical,
			Name:           "技术方案",
			Description:    "用于基于产品方案设计数据模型、接口契约和实现计划。",
			SystemPrompt:   technicalSystemPrompt,
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
