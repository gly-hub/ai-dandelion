package system

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

// ListAgentSessionConfigs
// @tags System
// @summary 获取搭建会话配置列表
// @router /system/agent-session-configs [GET]
func (s *SystemServerController) ListAgentSessionConfigs(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ListAgentSessionConfigsReq{}
	handler := func(ctx context.Context, req *systemproto.ListAgentSessionConfigsReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListAgentSessionConfigs(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateAgentSessionConfig
// @tags System
// @summary 更新搭建会话配置
// @router /system/agent-session-configs/{sessionType} [PUT]
func (s *SystemServerController) UpdateAgentSessionConfig(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.UpdateAgentSessionConfigReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.SessionType = ctx.Params("sessionType")
	handler := func(ctx context.Context, req *systemproto.UpdateAgentSessionConfigReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.UpdateAgentSessionConfig(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
