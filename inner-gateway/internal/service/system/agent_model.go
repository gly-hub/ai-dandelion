package system

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

// ListAgentModels
// @tags System
// @summary 获取 AI 模型配置列表
// @router /system/agent-models [GET]
func (s *SystemServerController) ListAgentModels(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ListAgentModelsReq{}
	handler := func(ctx context.Context, req *systemproto.ListAgentModelsReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListAgentModels(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateAgentModel
// @tags System
// @summary 创建 AI 模型配置
// @router /system/agent-models [POST]
func (s *SystemServerController) CreateAgentModel(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.CreateAgentModelReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *systemproto.CreateAgentModelReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CreateAgentModel(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateAgentModel
// @tags System
// @summary 更新 AI 模型配置
// @router /system/agent-models/{id} [PUT]
func (s *SystemServerController) UpdateAgentModel(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.UpdateAgentModelReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *systemproto.UpdateAgentModelReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.UpdateAgentModel(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteAgentModel
// @tags System
// @summary 删除 AI 模型配置
// @router /system/agent-models/{id} [DELETE]
func (s *SystemServerController) DeleteAgentModel(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DeleteAgentModelReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.DeleteAgentModelReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DeleteAgentModel(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// EnableAgentModel
// @tags System
// @summary 启用 AI 模型配置
// @router /system/agent-models/{id}/enable [POST]
func (s *SystemServerController) EnableAgentModel(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.EnableAgentModelReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.EnableAgentModelReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.EnableAgentModel(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DisableAgentModel
// @tags System
// @summary 禁用 AI 模型配置
// @router /system/agent-models/{id}/disable [POST]
func (s *SystemServerController) DisableAgentModel(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DisableAgentModelReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.DisableAgentModelReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DisableAgentModel(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
