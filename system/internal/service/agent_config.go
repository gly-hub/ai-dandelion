package service

import (
	"context"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *SystemService) GetAgentConfig(ctx context.Context, req *systemproto.GetAgentConfigReq) (
	out *systemproto.GetAgentConfigResp, err error) {
	grpcep.InitResponse(&out)
	out.Config, err = s.agentConfigLogic.GetAgentConfig(ctx, req)
	return
}

func (s *SystemService) UpdateAgentConfig(ctx context.Context, req *systemproto.UpdateAgentConfigReq) (
	out *systemproto.UpdateAgentConfigResp, err error) {
	grpcep.InitResponse(&out)
	out.Config, err = s.agentConfigLogic.UpdateAgentConfig(ctx, req)
	return
}
