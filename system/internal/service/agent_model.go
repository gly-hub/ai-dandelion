package service

import (
	"context"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *SystemService) ListAgentModels(ctx context.Context, req *systemproto.ListAgentModelsReq) (
	out *systemproto.ListAgentModelsResp, err error) {
	grpcep.InitResponse(&out)
	out.Models, err = s.agentModelLogic.ListAgentModels(ctx, req)
	return
}

func (s *SystemService) CreateAgentModel(ctx context.Context, req *systemproto.CreateAgentModelReq) (
	out *systemproto.CreateAgentModelResp, err error) {
	grpcep.InitResponse(&out)
	out.Model, err = s.agentModelLogic.CreateAgentModel(ctx, req)
	return
}

func (s *SystemService) UpdateAgentModel(ctx context.Context, req *systemproto.UpdateAgentModelReq) (
	out *systemproto.UpdateAgentModelResp, err error) {
	grpcep.InitResponse(&out)
	out.Model, err = s.agentModelLogic.UpdateAgentModel(ctx, req)
	return
}

func (s *SystemService) DeleteAgentModel(ctx context.Context, req *systemproto.DeleteAgentModelReq) (
	out *systemproto.DeleteAgentModelResp, err error) {
	grpcep.InitResponse(&out)
	err = s.agentModelLogic.DeleteAgentModel(ctx, req)
	return
}

func (s *SystemService) EnableAgentModel(ctx context.Context, req *systemproto.EnableAgentModelReq) (
	out *systemproto.EnableAgentModelResp, err error) {
	grpcep.InitResponse(&out)
	out.Model, err = s.agentModelLogic.EnableAgentModel(ctx, req)
	return
}

func (s *SystemService) DisableAgentModel(ctx context.Context, req *systemproto.DisableAgentModelReq) (
	out *systemproto.DisableAgentModelResp, err error) {
	grpcep.InitResponse(&out)
	out.Model, err = s.agentModelLogic.DisableAgentModel(ctx, req)
	return
}
