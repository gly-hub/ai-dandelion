package service

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/grpcep"
)

func (a *AiAgentService) ListAgentModels(ctx context.Context, req *aiagent.ListAgentModelsReq) (
	out *aiagent.ListAgentModelsResp, err error) {
	grpcep.InitResponse(&out)
	out.Models, err = a.agentModelLogic.ListAgentModels(ctx, req)
	return
}
