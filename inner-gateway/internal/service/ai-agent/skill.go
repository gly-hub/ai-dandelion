package aiagent

import (
	"context"
	"io"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

// ListSkills
// @tags AI Agent
// @summary 获取用户技能列表
// @router /ai-agent/skills [GET]
func (a *AIAgentServerController) ListSkills(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.ListSkillsReq{}
	handler := func(ctx context.Context, req *aiagent.ListSkillsReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ListSkills(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ListFunctionSkills returns functions that the current user can operate through Agent.
// @tags AI Agent
// @summary 获取可用功能技能
// @router /ai-agent/function-skills [GET]
func (a *AIAgentServerController) ListFunctionSkills(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.ListFunctionSkillsReq{}
	handler := func(rpcCtx context.Context, req *aiagent.ListFunctionSkillsReq) (interface{}, error) {
		client, err := a.getAiAgentClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.ListFunctionSkills(rpcCtx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ImportSkillPackage
// @tags AI Agent
// @summary 导入用户技能包
// @router /ai-agent/skills/import [POST]
func (a *AIAgentServerController) ImportSkillPackage(ctx *fiber.Ctx) error {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, "file is required"))
	}
	file, err := fileHeader.Open()
	if err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return a.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}

	rpcParam := &aiagent.ImportSkillPackageReq{
		FileName: fileHeader.Filename,
		Data:     data,
	}
	handler := func(ctx context.Context, req *aiagent.ImportSkillPackageReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.ImportSkillPackage(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateSkill
// @tags AI Agent
// @summary 更新用户技能
// @router /ai-agent/skills/{id} [PUT]
func (a *AIAgentServerController) UpdateSkill(ctx *fiber.Ctx) error {
	skillID := ctx.Params("id")
	rpcParam := &aiagent.UpdateSkillReq{}
	handler := func(ctx context.Context, req *aiagent.UpdateSkillReq) (interface{}, error) {
		if req.Skill == nil {
			req.Skill = &aiagent.AgentSkill{}
		}
		req.Skill.Id = skillID
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.UpdateSkill(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteSkill
// @tags AI Agent
// @summary 删除用户技能
// @router /ai-agent/skills/{id} [DELETE]
func (a *AIAgentServerController) DeleteSkill(ctx *fiber.Ctx) error {
	rpcParam := &aiagent.DeleteSkillReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *aiagent.DeleteSkillReq) (interface{}, error) {
		aiAgentClient, err := a.getAiAgentClient(ctx)
		if err != nil {
			return nil, err
		}
		return aiAgentClient.DeleteSkill(ctx, req)
	}
	return a.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
