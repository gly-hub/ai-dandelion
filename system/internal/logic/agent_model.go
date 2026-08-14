package logic

import (
	"context"
	"errors"
	"strings"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maskedAuthToken = "******"

type AgentModelLogic struct {
	agentModelDao *dao.AgentModel
}

func NewAgentModelLogic(agentModelDao *dao.AgentModel) *AgentModelLogic {
	return &AgentModelLogic{agentModelDao: agentModelDao}
}

func (a *AgentModelLogic) ListAgentModels(ctx context.Context, _ *systemproto.ListAgentModelsReq) ([]*systemproto.AgentModel, error) {
	items, err := a.agentModelDao.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*systemproto.AgentModel, 0, len(items))
	for i := range items {
		out = append(out, modelAgentModelToProto(&items[i], true))
	}
	return out, nil
}

func (a *AgentModelLogic) CreateAgentModel(ctx context.Context, req *systemproto.CreateAgentModelReq) (*systemproto.AgentModel, error) {
	name, modelName, err := validateAgentModelIdentity(req.GetName(), req.GetModel())
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	item := &model.AgentModel{
		ID:                uuid.New().String(),
		Name:              name,
		Model:             modelName,
		BaseURL:           strings.TrimSpace(req.GetBaseUrl()),
		AuthToken:         strings.TrimSpace(req.GetAuthToken()),
		ThinkMode:         strings.TrimSpace(req.GetThinkConfig().GetMode()),
		ThinkBudgetTokens: int(req.GetThinkConfig().GetBudgetTokens()),
		ThinkDisplay:      strings.TrimSpace(req.GetThinkConfig().GetDisplay()),
		MaxThinkingTokens: int(req.GetThinkConfig().GetMaxThinkingTokens()),
		Status:            normalizeAgentModelStatus(int(req.GetStatus())),
		IsDefault:         req.GetIsDefault(),
		Sort:              int(req.GetSort()),
		Remark:            strings.TrimSpace(req.GetRemark()),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if item.IsDefault {
		if err := a.agentModelDao.ClearDefault(ctx, ""); err != nil {
			return nil, err
		}
	}
	if err := a.agentModelDao.Create(ctx, item); err != nil {
		return nil, err
	}
	return modelAgentModelToProto(item, false), nil
}

func (a *AgentModelLogic) UpdateAgentModel(ctx context.Context, req *systemproto.UpdateAgentModelReq) (*systemproto.AgentModel, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("id is required")
	}
	name, modelName, err := validateAgentModelIdentity(req.GetName(), req.GetModel())
	if err != nil {
		return nil, err
	}
	item, err := a.agentModelDao.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("agent model not found")
		}
		return nil, err
	}
	item.Name = name
	item.Model = modelName
	item.BaseURL = strings.TrimSpace(req.GetBaseUrl())
	if token := strings.TrimSpace(req.GetAuthToken()); token != "" && token != maskedAuthToken {
		item.AuthToken = token
	}
	item.ThinkMode = strings.TrimSpace(req.GetThinkConfig().GetMode())
	item.ThinkBudgetTokens = int(req.GetThinkConfig().GetBudgetTokens())
	item.ThinkDisplay = strings.TrimSpace(req.GetThinkConfig().GetDisplay())
	item.MaxThinkingTokens = int(req.GetThinkConfig().GetMaxThinkingTokens())
	item.Status = normalizeAgentModelStatus(int(req.GetStatus()))
	item.IsDefault = req.GetIsDefault()
	item.Sort = int(req.GetSort())
	item.Remark = strings.TrimSpace(req.GetRemark())
	item.UpdatedAt = nowUnixMicro()
	if item.IsDefault {
		if err := a.agentModelDao.ClearDefault(ctx, item.ID); err != nil {
			return nil, err
		}
	}
	if err := a.agentModelDao.Save(ctx, item); err != nil {
		return nil, err
	}
	return modelAgentModelToProto(item, false), nil
}

func (a *AgentModelLogic) DeleteAgentModel(ctx context.Context, req *systemproto.DeleteAgentModelReq) error {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return errors.New("id is required")
	}
	_, err := a.agentModelDao.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("agent model not found")
		}
		return err
	}
	return a.agentModelDao.Delete(ctx, id)
}

func (a *AgentModelLogic) EnableAgentModel(ctx context.Context, req *systemproto.EnableAgentModelReq) (*systemproto.AgentModel, error) {
	return a.setAgentModelStatus(ctx, req.GetId(), model.AgentModelStatusEnabled)
}

func (a *AgentModelLogic) DisableAgentModel(ctx context.Context, req *systemproto.DisableAgentModelReq) (*systemproto.AgentModel, error) {
	return a.setAgentModelStatus(ctx, req.GetId(), model.AgentModelStatusDisabled)
}

func (a *AgentModelLogic) setAgentModelStatus(ctx context.Context, id string, status int) (*systemproto.AgentModel, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is required")
	}
	item, err := a.agentModelDao.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("agent model not found")
		}
		return nil, err
	}
	item.Status = status
	item.UpdatedAt = nowUnixMicro()
	if err := a.agentModelDao.Save(ctx, item); err != nil {
		return nil, err
	}
	return modelAgentModelToProto(item, false), nil
}

func validateAgentModelIdentity(name, modelName string) (string, string, error) {
	name = strings.TrimSpace(name)
	modelName = strings.TrimSpace(modelName)
	if name == "" {
		return "", "", errors.New("name is required")
	}
	if modelName == "" {
		return "", "", errors.New("model is required")
	}
	return name, modelName, nil
}

func normalizeAgentModelStatus(status int) int {
	if status == model.AgentModelStatusDisabled {
		return model.AgentModelStatusDisabled
	}
	return model.AgentModelStatusEnabled
}

func modelAgentModelToProto(item *model.AgentModel, maskToken bool) *systemproto.AgentModel {
	authToken := item.AuthToken
	if maskToken && authToken != "" {
		authToken = maskedAuthToken
	}
	return &systemproto.AgentModel{
		Id:        item.ID,
		Name:      item.Name,
		Model:     item.Model,
		BaseUrl:   item.BaseURL,
		AuthToken: authToken,
		ThinkConfig: &systemproto.AgentModelThinkConfig{
			Mode:              item.ThinkMode,
			BudgetTokens:      int32(item.ThinkBudgetTokens),
			Display:           item.ThinkDisplay,
			MaxThinkingTokens: int32(item.MaxThinkingTokens),
		},
		Status:    int32(item.Status),
		IsDefault: item.IsDefault,
		Sort:      int32(item.Sort),
		Remark:    item.Remark,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
