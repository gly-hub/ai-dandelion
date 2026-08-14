package system

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

// GetAgentConfig
// @tags System
// @summary 获取 AI Agent 系统配置
// @router /system/agent-config [GET]
func (s *SystemServerController) GetAgentConfig(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.GetAgentConfigReq{}
	handler := func(ctx context.Context, req *systemproto.GetAgentConfigReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetAgentConfig(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateAgentConfig
// @tags System
// @summary 更新 AI Agent 系统配置
// @router /system/agent-config [PUT]
func (s *SystemServerController) UpdateAgentConfig(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.UpdateAgentConfigReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *systemproto.UpdateAgentConfigReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.UpdateAgentConfig(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
