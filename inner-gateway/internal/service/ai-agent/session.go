package aiagent

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

// ListSessions
// @tags AI Agent
// @summary 获取会话列表
// @description 获取会话列表
// @accept json
// @param body body aiagent.SearchMessageReq true "Search Message Request"
// @success 200 {object} aiagent.SearchMessageResp "Search Message Response"
// @router /ai-agent/session [GET]
func (a *AIAgentServerController) ListSessions(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.SearchMessageReq{
		SessionType: int32(ctx.QueryInt("session_type", 0)),
	}
	handler := func(ctx context.Context, req *aiagent.SearchMessageReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ListSessions(ctx, req)
	}

	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateSession
// @tags AI Agent
// @summary 创建会话
// @description 创建会话
// @accept json
// @param body body aiagent.CreateSessionReq true "Create Session Request"
// @success 200 {object} aiagent.CreateSessionResp "Create Session Response"
// @router /ai-agent/session [POST]
func (a *AIAgentServerController) CreateSession(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.CreateSessionReq{}
	handler := func(ctx context.Context, req *aiagent.CreateSessionReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.CreateSession(ctx, req)
	}

	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// EnsureSession
// @tags AI Agent
// @summary 确保会话存在
// @description 外部渠道按稳定 ID 复用或创建会话
// @accept json
// @param body body aiagent.EnsureSessionReq true "Ensure Session Request"
// @success 200 {object} aiagent.EnsureSessionResp "Ensure Session Response"
// @router /ai-agent/session/ensure [POST]
func (a *AIAgentServerController) EnsureSession(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.EnsureSessionReq{}
	if err := a.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *aiagent.EnsureSessionReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.EnsureSession(ctx, req)
	}

	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateSession
// @tags AI Agent
// @summary 更新会话
// @description 更新会话标题
// @accept json
// @param id path string true "Session ID"
// @param body body aiagent.UpdateSessionReq true "Update Session Request"
// @success 200 {object} aiagent.UpdateSessionResp "Update Session Response"
// @router /ai-agent/session/{id} [PUT]
func (a *AIAgentServerController) UpdateSession(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.UpdateSessionReq{Id: ctx.Params("id")}
	if err := a.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *aiagent.UpdateSessionReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.UpdateSession(ctx, req)
	}

	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteSession
// @tags AI Agent
// @summary 删除会话
// @description 删除会话
// @accept json
// @param id path string true "Session ID"
// @success 200 {object} aiagent.DeleteSessionResp "Delete Session Response"
// @router /ai-agent/session/{id} [DELETE]
func (a *AIAgentServerController) DeleteSession(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.DeleteSessionReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *aiagent.DeleteSessionReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.DeleteSession(ctx, req)
	}

	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ListMessages
// @tags AI Agent
// @summary 获取消息列表
// @description 获取消息列表
// @accept json
// @param id path string true "Session ID"
// @param limit query int false "Limit"
// @param before query string false "Before Message ID"
// @success 200 {object} aiagent.GetMessageResp "Get Message Response"
// @router /ai-agent/session/{id}/messages [GET]
func (a *AIAgentServerController) ListMessages(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.GetMessageReq{
		SessionId: ctx.Params("id"),
		Limit:     int32(ctx.QueryInt("limit", 0)),
		Before:    ctx.Query("before"),
	}
	handler := func(ctx context.Context, req *aiagent.GetMessageReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ListMessages(ctx, req)
	}

	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// StreamMessage
// @tags AI Agent
// @summary 流式对话
// @description 流式对话
// @accept json
// @produce text/event-stream
// @param id path string true "Session ID"
// @param body body aiagent.StreamMessageReq true "Stream Message Request"
// @success 200 {object} aiagent.StreamMessageResp "Stream Message Response"
// @router /ai-agent/session/{id}/messages/stream [POST]
func (a *AIAgentServerController) StreamMessage(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.StreamMessageReq{}
	if err := a.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.SessionId = ctx.Params("id")

	aiAgentClient, err := a.getAiAgentClient(ctx.Context())
	if err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}

	streamFunc := func(rpcCtx context.Context, req interface{}) (interface{}, error) {
		return aiAgentClient.StreamMessage(rpcCtx, req.(*aiagent.StreamMessageReq))
	}
	return a.baseHandler.RPCStream(ctx, rpcParam, streamFunc)
}

// SubmitAskUserQuestion
// @tags AI Agent
// @summary 提交 AskUserQuestion 回答
// @accept json
// @param id path string true "Session ID"
// @param toolId path string true "Tool Use ID"
// @router /ai-agent/session/{id}/tool-requests/{toolId}/answer [POST]
func (a *AIAgentServerController) SubmitAskUserQuestion(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.SubmitAskUserQuestionReq{}
	if err := a.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.SessionId = ctx.Params("id")
	rpcParam.ToolId = ctx.Params("toolId")
	handler := func(ctx context.Context, req *aiagent.SubmitAskUserQuestionReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.SubmitAskUserQuestion(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// SubmitToolPermission
// @tags AI Agent
// @summary 提交工具执行授权决定
// @accept json
// @param id path string true "Session ID"
// @param toolId path string true "Tool Use ID"
// @router /ai-agent/session/{id}/tool-requests/{toolId}/permission [POST]
func (a *AIAgentServerController) SubmitToolPermission(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.SubmitToolPermissionReq{}
	if err := a.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.SessionId = ctx.Params("id")
	rpcParam.ToolId = ctx.Params("toolId")
	handler := func(ctx context.Context, req *aiagent.SubmitToolPermissionReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.SubmitToolPermission(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
