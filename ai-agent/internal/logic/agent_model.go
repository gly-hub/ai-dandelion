package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/team-dandelion/ai-dandelion/ai-agent/config"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/dao"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"github.com/team-dandelion/ai-dandelion/toolbox/agent"
	"gorm.io/gorm"
)

type AgentModelLogic struct {
	agentModelDao *dao.AgentModel
}

func NewAgentModelLogic(agentModelDao *dao.AgentModel) *AgentModelLogic {
	return &AgentModelLogic{agentModelDao: agentModelDao}
}

func (a *AgentModelLogic) ListAgentModels(ctx context.Context, _ *aiagent.ListAgentModelsReq) ([]*aiagent.AgentModelOption, error) {
	items, err := a.agentModelDao.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*aiagent.AgentModelOption, 0, len(items))
	for i := range items {
		out = append(out, modelAgentModelOptionToProto(&items[i]))
	}
	return out, nil
}

func (a *AgentModelLogic) ResolveRunner(
	ctx context.Context,
	factory runnerFactory,
	modelID string,
	override AgentRuntimeOverride,
) (agent.Runner, error) {
	if factory == nil || factory.DefaultRunner() == nil {
		return nil, errAgentRunnerNotConfigured
	}
	modelID = strings.TrimSpace(modelID)
	if modelID != "" {
		record, err := a.agentModelDao.GetEnabled(ctx, modelID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("agent model not found or disabled")
			}
			return nil, err
		}
		return factory.RunnerForConfig(ctx, record, override), nil
	}
	record, err := a.agentModelDao.GetDefaultEnabled(ctx)
	if err == nil && record != nil {
		return factory.RunnerForConfig(ctx, record, override), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		return nil, err
	}
	return factory.RunnerForConfig(ctx, nil, override), nil
}

func EnsureSeedAgentModels(ctx context.Context, agentModelDao *dao.AgentModel, cfg config.AgentConfig) error {
	count, err := agentModelDao.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil
	}
	now := nowUnixMicro()
	item := &model.AgentModel{
		ID:                uuid.New().String(),
		Name:              cfg.Model,
		Model:             cfg.Model,
		BaseURL:           strings.TrimSpace(cfg.BaseURL),
		AuthToken:         strings.TrimSpace(cfg.AuthToken),
		ThinkMode:         strings.TrimSpace(cfg.ThinkConfig.Mode),
		ThinkBudgetTokens: cfg.ThinkConfig.BudgetTokens,
		ThinkDisplay:      strings.TrimSpace(cfg.ThinkConfig.Display),
		MaxThinkingTokens: cfg.ThinkConfig.MaxThinkingTokens,
		Status:            model.AgentModelStatusEnabled,
		IsDefault:         true,
		Sort:              10,
		Remark:            "从本地配置文件初始化",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return agentModelDao.Create(ctx, item)
}

func modelAgentModelOptionToProto(item *model.AgentModel) *aiagent.AgentModelOption {
	return &aiagent.AgentModelOption{
		Id:        item.ID,
		Name:      item.Name,
		Model:     item.Model,
		IsDefault: item.IsDefault,
	}
}
