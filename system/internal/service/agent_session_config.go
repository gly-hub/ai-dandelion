package service

import (
	"context"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *SystemService) ListAgentSessionConfigs(ctx context.Context, req *systemproto.ListAgentSessionConfigsReq) (
	out *systemproto.ListAgentSessionConfigsResp, err error) {
	grpcep.InitResponse(&out)
	out.Configs, err = s.agentSessionConfigLogic.ListAgentSessionConfigs(ctx, req)
	return
}

func (s *SystemService) UpdateAgentSessionConfig(ctx context.Context, req *systemproto.UpdateAgentSessionConfigReq) (
	out *systemproto.UpdateAgentSessionConfigResp, err error) {
	grpcep.InitResponse(&out)
	out.Config, err = s.agentSessionConfigLogic.UpdateAgentSessionConfig(ctx, req)
	return
}
