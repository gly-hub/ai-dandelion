package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/dao"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"gorm.io/gorm"
)

const maskedAgentBotSecret = "******"

type AgentBotLogic struct {
	agentBotDao *dao.AgentBot
	reload      func(context.Context) error
}

func NewAgentBotLogic(agentBotDao *dao.AgentBot, reload func(context.Context) error) *AgentBotLogic {
	return &AgentBotLogic{agentBotDao: agentBotDao, reload: reload}
}

func (a *AgentBotLogic) ListAgentBots(ctx context.Context, _ *aiagent.ListAgentBotsReq) ([]*aiagent.AgentBot, error) {
	items, err := a.agentBotDao.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*aiagent.AgentBot, 0, len(items))
	for i := range items {
		out = append(out, agentBotAggregateToProto(&items[i], true))
	}
	return out, nil
}

func (a *AgentBotLogic) ListAgentBotRuntimeConfigs(ctx context.Context, _ *aiagent.ListAgentBotRuntimeConfigsReq) ([]*aiagent.AgentBot, error) {
	items, err := a.agentBotDao.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*aiagent.AgentBot, 0, len(items))
	for i := range items {
		if items[i].Bot.Status != model.AgentBotStatusEnabled {
			continue
		}
		out = append(out, agentBotAggregateToProto(&items[i], false))
	}
	return out, nil
}

func (a *AgentBotLogic) CreateAgentBot(ctx context.Context, req *aiagent.CreateAgentBotReq) (*aiagent.AgentBot, error) {
	name, code, err := validateAgentBotIdentity(req.GetName(), req.GetCode())
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	bot := &model.AgentBot{
		ID:             uuid.NewString(),
		Name:           name,
		Code:           code,
		Status:         normalizeAgentBotStatus(int(req.GetStatus())),
		Description:    strings.TrimSpace(req.GetDescription()),
		BusinessScene:  strings.TrimSpace(req.GetBusinessScene()),
		WelcomeMessage: strings.TrimSpace(req.GetWelcomeMessage()),
		ModelID:        strings.TrimSpace(req.GetModelId()),
		SystemPrompt:   strings.TrimSpace(req.GetSystemPrompt()),
		PermissionMode: normalizePermissionMode(req.GetPermissionMode()),
		MaxTurns:       normalizeAgentBotMaxTurns(int(req.GetMaxTurns())),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	channels := buildAgentBotChannels(bot.ID, req.GetChannels(), now, nil)
	capabilities := buildAgentBotCapabilities(bot.ID, req.GetCapabilities(), now)
	if err := a.agentBotDao.Create(ctx, bot, channels, capabilities); err != nil {
		return nil, err
	}
	a.reloadRuntime(ctx)
	return agentBotAggregateToProto(&dao.AgentBotAggregate{Bot: *bot, Channels: channels, Capabilities: capabilities}, true), nil
}

func (a *AgentBotLogic) UpdateAgentBot(ctx context.Context, req *aiagent.UpdateAgentBotReq) (*aiagent.AgentBot, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("id is required")
	}
	name, code, err := validateAgentBotIdentity(req.GetName(), req.GetCode())
	if err != nil {
		return nil, err
	}
	existing, err := a.agentBotDao.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("agent bot not found")
		}
		return nil, err
	}
	now := nowUnixMicro()
	bot := existing.Bot
	bot.Name = name
	bot.Code = code
	bot.Status = normalizeAgentBotStatus(int(req.GetStatus()))
	bot.Description = strings.TrimSpace(req.GetDescription())
	bot.BusinessScene = strings.TrimSpace(req.GetBusinessScene())
	bot.WelcomeMessage = strings.TrimSpace(req.GetWelcomeMessage())
	bot.ModelID = strings.TrimSpace(req.GetModelId())
	bot.SystemPrompt = strings.TrimSpace(req.GetSystemPrompt())
	bot.PermissionMode = normalizePermissionMode(req.GetPermissionMode())
	bot.MaxTurns = normalizeAgentBotMaxTurns(int(req.GetMaxTurns()))
	bot.UpdatedAt = now
	channels := buildAgentBotChannels(bot.ID, req.GetChannels(), now, existing.Channels)
	capabilities := buildAgentBotCapabilities(bot.ID, req.GetCapabilities(), now)
	if err := a.agentBotDao.Update(ctx, &bot, channels, capabilities); err != nil {
		return nil, err
	}
	a.reloadRuntime(ctx)
	return agentBotAggregateToProto(&dao.AgentBotAggregate{Bot: bot, Channels: channels, Capabilities: capabilities}, true), nil
}

func (a *AgentBotLogic) DeleteAgentBot(ctx context.Context, req *aiagent.DeleteAgentBotReq) error {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return errors.New("id is required")
	}
	if err := a.agentBotDao.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("agent bot not found")
		}
		return err
	}
	a.reloadRuntime(ctx)
	return nil
}

func (a *AgentBotLogic) EnableAgentBot(ctx context.Context, req *aiagent.EnableAgentBotReq) (*aiagent.AgentBot, error) {
	return a.setAgentBotStatus(ctx, req.GetId(), model.AgentBotStatusEnabled)
}

func (a *AgentBotLogic) DisableAgentBot(ctx context.Context, req *aiagent.DisableAgentBotReq) (*aiagent.AgentBot, error) {
	return a.setAgentBotStatus(ctx, req.GetId(), model.AgentBotStatusDisabled)
}

func (a *AgentBotLogic) setAgentBotStatus(ctx context.Context, id string, status int) (*aiagent.AgentBot, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is required")
	}
	item, err := a.agentBotDao.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("agent bot not found")
		}
		return nil, err
	}
	item.Bot.Status = status
	item.Bot.UpdatedAt = nowUnixMicro()
	if err := a.agentBotDao.SaveBot(ctx, &item.Bot); err != nil {
		return nil, err
	}
	a.reloadRuntime(ctx)
	return agentBotAggregateToProto(item, true), nil
}

func (a *AgentBotLogic) reloadRuntime(ctx context.Context) {
	if a == nil || a.reload == nil {
		return
	}
	_ = a.reload(ctx)
}

func validateAgentBotIdentity(name string, code string) (string, string, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	if name == "" {
		return "", "", errors.New("name is required")
	}
	if code == "" {
		return "", "", errors.New("code is required")
	}
	return name, code, nil
}

func normalizeAgentBotStatus(status int) int {
	if status == model.AgentBotStatusDisabled {
		return model.AgentBotStatusDisabled
	}
	return model.AgentBotStatusEnabled
}

func normalizeAgentBotChannelStatus(status int) int {
	if status == model.AgentBotChannelStatusDisabled {
		return model.AgentBotChannelStatusDisabled
	}
	return model.AgentBotChannelStatusEnabled
}

func normalizeAgentBotCapabilityEnabled(enabled int) int {
	if enabled == model.AgentBotCapabilityDisabled {
		return model.AgentBotCapabilityDisabled
	}
	return model.AgentBotCapabilityEnabled
}

func normalizeAgentBotMaxTurns(maxTurns int) int {
	if maxTurns <= 0 {
		return 20
	}
	if maxTurns > 200 {
		return 200
	}
	return maxTurns
}

func normalizePermissionMode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "bypassPermissions"
	}
	return value
}

func buildAgentBotChannels(botID string, inputs []*aiagent.AgentBotChannel, now int64, existing []model.AgentBotChannel) []model.AgentBotChannel {
	secretByID := make(map[string]string, len(existing))
	for i := range existing {
		secretByID[existing[i].ID] = existing[i].Secret
	}
	out := make([]model.AgentBotChannel, 0, len(inputs))
	for _, input := range inputs {
		channel := strings.TrimSpace(input.GetChannel())
		name := strings.TrimSpace(input.GetName())
		if channel == "" && name == "" && strings.TrimSpace(input.GetExternalBotId()) == "" {
			continue
		}
		id := strings.TrimSpace(input.GetId())
		if id == "" {
			id = uuid.NewString()
		}
		secret := strings.TrimSpace(input.GetSecret())
		if secret == maskedAgentBotSecret {
			secret = secretByID[id]
		}
		out = append(out, model.AgentBotChannel{
			ID:            id,
			BotID:         botID,
			Channel:       channel,
			Name:          name,
			Status:        normalizeAgentBotChannelStatus(int(input.GetStatus())),
			ExternalBotID: strings.TrimSpace(input.GetExternalBotId()),
			Secret:        secret,
			EndpointURL:   strings.TrimSpace(input.GetEndpointUrl()),
			ConfigJSON:    strings.TrimSpace(input.GetConfigJson()),
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return out
}

func buildAgentBotCapabilities(botID string, inputs []*aiagent.AgentBotCapability, now int64) []model.AgentBotCapability {
	out := make([]model.AgentBotCapability, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		capabilityType := strings.TrimSpace(input.GetCapabilityType())
		capabilityID := strings.TrimSpace(input.GetCapabilityId())
		if capabilityType == "" || capabilityID == "" {
			continue
		}
		key := capabilityType + "\x00" + capabilityID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		id := strings.TrimSpace(input.GetId())
		if id == "" {
			id = uuid.NewString()
		}
		out = append(out, model.AgentBotCapability{
			ID:             id,
			BotID:          botID,
			CapabilityType: capabilityType,
			CapabilityID:   capabilityID,
			Name:           strings.TrimSpace(input.GetName()),
			Enabled:        normalizeAgentBotCapabilityEnabled(int(input.GetEnabled())),
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	return out
}

func agentBotAggregateToProto(item *dao.AgentBotAggregate, maskSecret bool) *aiagent.AgentBot {
	if item == nil {
		return &aiagent.AgentBot{}
	}
	return &aiagent.AgentBot{
		Id:             item.Bot.ID,
		Name:           item.Bot.Name,
		Code:           item.Bot.Code,
		Status:         int32(item.Bot.Status),
		Description:    item.Bot.Description,
		BusinessScene:  item.Bot.BusinessScene,
		WelcomeMessage: item.Bot.WelcomeMessage,
		ModelId:        item.Bot.ModelID,
		SystemPrompt:   item.Bot.SystemPrompt,
		PermissionMode: item.Bot.PermissionMode,
		MaxTurns:       int32(item.Bot.MaxTurns),
		CreatedAt:      item.Bot.CreatedAt,
		UpdatedAt:      item.Bot.UpdatedAt,
		Channels:       agentBotChannelsToProto(item.Channels, maskSecret),
		Capabilities:   agentBotCapabilitiesToProto(item.Capabilities),
	}
}

func agentBotChannelsToProto(items []model.AgentBotChannel, maskSecret bool) []*aiagent.AgentBotChannel {
	out := make([]*aiagent.AgentBotChannel, 0, len(items))
	for i := range items {
		secret := items[i].Secret
		if maskSecret && secret != "" {
			secret = maskedAgentBotSecret
		}
		out = append(out, &aiagent.AgentBotChannel{
			Id:            items[i].ID,
			BotId:         items[i].BotID,
			Channel:       items[i].Channel,
			Name:          items[i].Name,
			Status:        int32(items[i].Status),
			ExternalBotId: items[i].ExternalBotID,
			Secret:        secret,
			EndpointUrl:   items[i].EndpointURL,
			ConfigJson:    items[i].ConfigJSON,
			CreatedAt:     items[i].CreatedAt,
			UpdatedAt:     items[i].UpdatedAt,
		})
	}
	return out
}

func agentBotCapabilitiesToProto(items []model.AgentBotCapability) []*aiagent.AgentBotCapability {
	out := make([]*aiagent.AgentBotCapability, 0, len(items))
	for i := range items {
		out = append(out, &aiagent.AgentBotCapability{
			Id:             items[i].ID,
			BotId:          items[i].BotID,
			CapabilityType: items[i].CapabilityType,
			CapabilityId:   items[i].CapabilityID,
			Name:           items[i].Name,
			Enabled:        int32(items[i].Enabled),
			CreatedAt:      items[i].CreatedAt,
			UpdatedAt:      items[i].UpdatedAt,
		})
	}
	return out
}
