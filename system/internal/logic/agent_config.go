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

type AgentConfigLogic struct {
	agentConfigDao *dao.AgentConfig
}

func NewAgentConfigLogic(agentConfigDao *dao.AgentConfig) *AgentConfigLogic {
	return &AgentConfigLogic{agentConfigDao: agentConfigDao}
}

func (a *AgentConfigLogic) GetAgentConfig(ctx context.Context, _ *systemproto.GetAgentConfigReq) (*systemproto.AgentSystemConfig, error) {
	item, err := a.agentConfigDao.Get(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &systemproto.AgentSystemConfig{}, nil
		}
		return nil, err
	}
	return modelAgentConfigToProto(item), nil
}

func (a *AgentConfigLogic) UpdateAgentConfig(ctx context.Context, req *systemproto.UpdateAgentConfigReq) (*systemproto.AgentSystemConfig, error) {
	permissionMode := strings.TrimSpace(req.GetPermissionMode())
	if permissionMode == "" {
		permissionMode = "bypassPermissions"
	}
	maxTurns := int(req.GetMaxTurns())
	if maxTurns <= 0 {
		maxTurns = 20
	}
	now := nowUnixMicro()
	item, err := a.agentConfigDao.Get(ctx)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		item = &model.AgentSystemConfig{
			ID:        model.AgentSystemConfigID,
			CreatedAt: now,
		}
	}
	item.SystemPrompt = strings.TrimSpace(req.GetSystemPrompt())
	item.PermissionMode = permissionMode
	item.MaxTurns = maxTurns
	item.ImageToolEnabled = req.GetImageToolEnabled()
	item.ImageModelID = strings.TrimSpace(req.GetImageModelId())
	item.UpdatedAt = now
	if err := a.agentConfigDao.Save(ctx, item); err != nil {
		return nil, err
	}
	return modelAgentConfigToProto(item), nil
}

func (a *AgentConfigLogic) EnsureSeedAgentConfig(ctx context.Context, systemPrompt, permissionMode string, maxTurns int) error {
	count, err := a.agentConfigDao.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if strings.TrimSpace(permissionMode) == "" {
		permissionMode = "bypassPermissions"
	}
	if maxTurns <= 0 {
		maxTurns = 20
	}
	now := nowUnixMicro()
	item := &model.AgentSystemConfig{
		ID:             model.AgentSystemConfigID,
		SystemPrompt:   strings.TrimSpace(systemPrompt),
		PermissionMode: permissionMode,
		MaxTurns:       maxTurns,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return a.agentConfigDao.Save(ctx, item)
}

func modelAgentConfigToProto(item *model.AgentSystemConfig) *systemproto.AgentSystemConfig {
	if item == nil {
		return &systemproto.AgentSystemConfig{}
	}
	return &systemproto.AgentSystemConfig{
		SystemPrompt:   item.SystemPrompt,
		PermissionMode: item.PermissionMode,
		MaxTurns:       int32(item.MaxTurns),
		ImageToolEnabled: item.ImageToolEnabled,
		ImageModelId:    item.ImageModelID,
		UpdatedAt:      item.UpdatedAt,
	}
}
