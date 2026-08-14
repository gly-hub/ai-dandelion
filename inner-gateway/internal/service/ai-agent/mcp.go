package aiagent

import (
	"context"

	"github.com/gofiber/fiber/v2"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
)

// ListMCPServers
// @tags AI Agent
// @summary 获取用户 MCP 列表
// @router /ai-agent/mcp-servers [GET]
func (a *AIAgentServerController) ListMCPServers(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.ListMCPServersReq{}
	handler := func(ctx context.Context, req *aiagent.ListMCPServersReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ListMCPServers(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateMCPServer
// @tags AI Agent
// @summary 创建 MCP Server
// @router /ai-agent/mcp-servers [POST]
func (a *AIAgentServerController) CreateMCPServer(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.SaveMCPServerReq{}
	handler := func(ctx context.Context, req *aiagent.SaveMCPServerReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.CreateMCPServer(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateMCPServer
// @tags AI Agent
// @summary 更新 MCP Server
// @router /ai-agent/mcp-servers/{id} [PUT]
func (a *AIAgentServerController) UpdateMCPServer(ctx *fiber.Ctx) error {
	serverID := ctx.Params("id")
	rpcParam := &aiagent.SaveMCPServerReq{}
	handler := func(ctx context.Context, req *aiagent.SaveMCPServerReq) (interface{}, error) {
		if req.Server == nil {
			req.Server = &aiagent.AgentMCPServer{}
		}
		req.Server.Id = serverID
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.UpdateMCPServer(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteMCPServer
// @tags AI Agent
// @summary 删除 MCP Server
// @router /ai-agent/mcp-servers/{id} [DELETE]
func (a *AIAgentServerController) DeleteMCPServer(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.DeleteMCPServerReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *aiagent.DeleteMCPServerReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.DeleteMCPServer(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
