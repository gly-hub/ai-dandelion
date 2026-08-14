package aiagent

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gofiber/fiber/v2"
)

// ListAgentModels
// @tags AI Agent
// @summary 获取可用模型列表
// @router /ai-agent/models [GET]
func (a *AIAgentServerController) ListAgentModels(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.ListAgentModelsReq{}
	handler := func(ctx context.Context, req *aiagent.ListAgentModelsReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ListAgentModels(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
