package aiagent

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/team-dandelion/ai-dandelion/inner-gateway/global"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"github.com/team-dandelion/quickgo/gerr"
	"github.com/team-dandelion/quickgo/grpcep"
)

// ListAgentBots
// @tags AI Agent
// @summary 获取智能机器人列表
// @router /ai-agent/agent-bots [GET]
func (a *AIAgentServerController) ListAgentBots(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.ListAgentBotsReq{}
	handler := func(ctx context.Context, req *aiagent.ListAgentBotsReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ListAgentBots(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ListAgentBotRuntimeConfigs
// @tags AI Agent
// @summary 获取智能机器人运行时配置
// @router /ai-agent/agent-bots/runtime [GET]
func (a *AIAgentServerController) ListAgentBotRuntimeConfigs(ctx *fiber.Ctx) error {
	if !validBridgeToken(ctx) {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code": fiber.StatusUnauthorized,
			"msg":  "invalid bridge token",
		})
	}
	clearRuntimeBridgeCredentialHeaders(ctx)
	rpcParam := &aiagent.ListAgentBotRuntimeConfigsReq{}
	handler := func(ctx context.Context, req *aiagent.ListAgentBotRuntimeConfigsReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ListAgentBotRuntimeConfigs(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateAgentBot
// @tags AI Agent
// @summary 创建智能机器人
// @router /ai-agent/agent-bots [POST]
func (a *AIAgentServerController) CreateAgentBot(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.CreateAgentBotReq{}
	if err := a.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *aiagent.CreateAgentBotReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.CreateAgentBot(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateAgentBot
// @tags AI Agent
// @summary 更新智能机器人
// @router /ai-agent/agent-bots/{id} [PUT]
func (a *AIAgentServerController) UpdateAgentBot(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.UpdateAgentBotReq{}
	if err := a.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *aiagent.UpdateAgentBotReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.UpdateAgentBot(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteAgentBot
// @tags AI Agent
// @summary 删除智能机器人
// @router /ai-agent/agent-bots/{id} [DELETE]
func (a *AIAgentServerController) DeleteAgentBot(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.DeleteAgentBotReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *aiagent.DeleteAgentBotReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.DeleteAgentBot(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// EnableAgentBot
// @tags AI Agent
// @summary 启用智能机器人
// @router /ai-agent/agent-bots/{id}/enable [POST]
func (a *AIAgentServerController) EnableAgentBot(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.EnableAgentBotReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *aiagent.EnableAgentBotReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.EnableAgentBot(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DisableAgentBot
// @tags AI Agent
// @summary 禁用智能机器人
// @router /ai-agent/agent-bots/{id}/disable [POST]
func (a *AIAgentServerController) DisableAgentBot(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.DisableAgentBotReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *aiagent.DisableAgentBotReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.DisableAgentBot(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

func validBridgeToken(ctx *fiber.Ctx) bool {
	expected := strings.TrimSpace(global.GetConfig().AuthConfig.BridgeToken)
	if expected == "" {
		return false
	}
	actual := strings.TrimSpace(ctx.Get("X-Bridge-Token"))
	if actual == "" {
		actual = strings.TrimSpace(ctx.Get(fiber.HeaderAuthorization))
		actual = strings.TrimPrefix(strings.TrimPrefix(actual, "Bearer "), "bearer ")
	}
	return actual == expected
}

// The bridge token only authenticates the public HTTP boundary. Do not carry
// it into the internal gRPC metadata created by BaseHandler.RPCCtx.
func clearRuntimeBridgeCredentialHeaders(ctx *fiber.Ctx) {
	ctx.Request().Header.Del(fiber.HeaderAuthorization)
	ctx.Request().Header.Del("X-Bridge-Token")
}
