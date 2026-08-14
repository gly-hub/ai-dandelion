package service

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
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
