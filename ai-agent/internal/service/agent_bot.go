package service

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *AiAgentService) ListAgentBots(ctx context.Context, req *aiagent.ListAgentBotsReq) (
	out *aiagent.ListAgentBotsResp, err error) {
	grpcep.InitResponse(&out)
	out.Bots, err = s.agentBotLogic.ListAgentBots(ctx, req)
	return
}

func (s *AiAgentService) ListAgentBotRuntimeConfigs(ctx context.Context, req *aiagent.ListAgentBotRuntimeConfigsReq) (
	out *aiagent.ListAgentBotRuntimeConfigsResp, err error) {
	grpcep.InitResponse(&out)
	out.Bots, err = s.agentBotLogic.ListAgentBotRuntimeConfigs(ctx, req)
	return
}

func (s *AiAgentService) CreateAgentBot(ctx context.Context, req *aiagent.CreateAgentBotReq) (
	out *aiagent.CreateAgentBotResp, err error) {
	grpcep.InitResponse(&out)
	out.Bot, err = s.agentBotLogic.CreateAgentBot(ctx, req)
	return
}

func (s *AiAgentService) UpdateAgentBot(ctx context.Context, req *aiagent.UpdateAgentBotReq) (
	out *aiagent.UpdateAgentBotResp, err error) {
	grpcep.InitResponse(&out)
	out.Bot, err = s.agentBotLogic.UpdateAgentBot(ctx, req)
	return
}

func (s *AiAgentService) DeleteAgentBot(ctx context.Context, req *aiagent.DeleteAgentBotReq) (
	out *aiagent.DeleteAgentBotResp, err error) {
	grpcep.InitResponse(&out)
	err = s.agentBotLogic.DeleteAgentBot(ctx, req)
	return
}

func (s *AiAgentService) EnableAgentBot(ctx context.Context, req *aiagent.EnableAgentBotReq) (
	out *aiagent.EnableAgentBotResp, err error) {
	grpcep.InitResponse(&out)
	out.Bot, err = s.agentBotLogic.EnableAgentBot(ctx, req)
	return
}

func (s *AiAgentService) DisableAgentBot(ctx context.Context, req *aiagent.DisableAgentBotReq) (
	out *aiagent.DisableAgentBotResp, err error) {
	grpcep.InitResponse(&out)
	out.Bot, err = s.agentBotLogic.DisableAgentBot(ctx, req)
	return
}
