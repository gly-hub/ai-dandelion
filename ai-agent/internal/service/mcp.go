package service

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *AiAgentService) ListMCPServers(ctx context.Context, req *aiagent.ListMCPServersReq) (
	out *aiagent.ListMCPServersResp, err error) {
	grpcep.InitResponse(&out)
	out.Servers, err = s.mcpLogic.ListMCPServers(ctx, req)
	return
}

func (s *AiAgentService) CreateMCPServer(ctx context.Context, req *aiagent.SaveMCPServerReq) (
	out *aiagent.SaveMCPServerResp, err error) {
	grpcep.InitResponse(&out)
	out.Server, err = s.mcpLogic.CreateMCPServer(ctx, req)
	return
}

func (s *AiAgentService) UpdateMCPServer(ctx context.Context, req *aiagent.SaveMCPServerReq) (
	out *aiagent.SaveMCPServerResp, err error) {
	grpcep.InitResponse(&out)
	out.Server, err = s.mcpLogic.UpdateMCPServer(ctx, req)
	return
}

func (s *AiAgentService) DeleteMCPServer(ctx context.Context, req *aiagent.DeleteMCPServerReq) (
	out *aiagent.DeleteMCPServerResp, err error) {
	grpcep.InitResponse(&out)
	out.Id, err = s.mcpLogic.DeleteMCPServer(ctx, req)
	return
}
