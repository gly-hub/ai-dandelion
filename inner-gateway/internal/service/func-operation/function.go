package funcoperation

import (
	"context"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

// ListFunctions
// @tags Func Operation
// @summary 获取功能列表
// @router /func-operation/functions [GET]
func (f *FuncOperationServerController) ListFunctions(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.ListFunctionsReq{}
	handler := func(ctx context.Context, req *funcoperation.ListFunctionsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListFunctions(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateFunction
// @tags Func Operation
// @summary 创建功能草稿
// @router /func-operation/functions [POST]
func (f *FuncOperationServerController) CreateFunction(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.CreateFunctionReq{}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}

	handler := func(ctx context.Context, req *funcoperation.CreateFunctionReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CreateFunction(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateFunction
// @tags Func Operation
// @summary 更新功能配置
// @router /func-operation/functions/{id} [PUT]
func (f *FuncOperationServerController) UpdateFunction(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.UpdateFunctionReq{
		Id: ctx.Params("id"),
	}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")

	handler := func(ctx context.Context, req *funcoperation.UpdateFunctionReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.UpdateFunction(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// SetFunctionSkillEnabled enables or disables Agent operations for a function.
// @tags Func Operation
// @summary 设置功能技能状态
// @router /func-operation/functions/{id}/skill [PUT]
func (f *FuncOperationServerController) SetFunctionSkillEnabled(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.SetFunctionSkillEnabledReq{FunctionId: ctx.Params("id")}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.FunctionId = ctx.Params("id")
	handler := func(rpcCtx context.Context, req *funcoperation.SetFunctionSkillEnabledReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.SetFunctionSkillEnabled(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ListFunctionSkillExecutions returns the administrator audit trail for a function skill.
// @tags Func Operation
// @summary 查询功能技能执行记录
// @router /func-operation/functions/{id}/skill-executions [GET]
func (f *FuncOperationServerController) ListFunctionSkillExecutions(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.ListFunctionSkillExecutionsReq{FunctionId: ctx.Params("id"), Limit: int32(ctx.QueryInt("limit", 0))}
	handler := func(rpcCtx context.Context, req *funcoperation.ListFunctionSkillExecutionsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.ListFunctionSkillExecutions(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// LoadFunctionDocument
// @tags Func Operation
// @summary 加载功能文档
// @router /func-operation/functions/{id}/documents/load [POST]
func (f *FuncOperationServerController) LoadFunctionDocument(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.LoadFunctionDocumentReq{
		Id: ctx.Params("id"),
	}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *funcoperation.LoadFunctionDocumentReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.LoadFunctionDocument(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CommitFunctionDocument
// @tags Func Operation
// @summary 保存并提交功能文档
// @router /func-operation/functions/{id}/documents/commit [POST]
func (f *FuncOperationServerController) CommitFunctionDocument(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.CommitFunctionDocumentReq{
		Id: ctx.Params("id"),
	}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *funcoperation.CommitFunctionDocumentReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CommitFunctionDocument(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// MaterializeFunctionApp
// @tags Func Operation
// @summary 生成或更新功能页面
// @router /func-operation/functions/{id}/materialize [POST]
func (f *FuncOperationServerController) MaterializeFunctionApp(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.MaterializeFunctionAppReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *funcoperation.MaterializeFunctionAppReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.MaterializeFunctionApp(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// LoadFunctionCodeState
// @tags Func Operation
// @summary 加载功能代码版本状态
// @router /func-operation/functions/{id}/code/load [POST]
func (f *FuncOperationServerController) LoadFunctionCodeState(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.LoadFunctionCodeStateReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *funcoperation.LoadFunctionCodeStateReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.LoadFunctionCodeState(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// TouchFunctionCodeDraft
// @tags Func Operation
// @summary 标记功能代码最新生成版本
// @router /func-operation/functions/{id}/code/draft [POST]
func (f *FuncOperationServerController) TouchFunctionCodeDraft(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.TouchFunctionCodeDraftReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *funcoperation.TouchFunctionCodeDraftReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.TouchFunctionCodeDraft(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ApplyFunctionCode
// @tags Func Operation
// @summary 应用功能代码最新生成版本
// @router /func-operation/functions/{id}/code/apply [POST]
func (f *FuncOperationServerController) ApplyFunctionCode(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.ApplyFunctionCodeReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *funcoperation.ApplyFunctionCodeReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ApplyFunctionCode(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// EnsureFunctionSession
// @tags Func Operation
// @summary 确保功能会话存在
// @router /func-operation/functions/{id}/sessions/ensure [POST]
func (f *FuncOperationServerController) EnsureFunctionSession(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.EnsureFunctionSessionReq{
		Id: ctx.Params("id"),
	}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *funcoperation.EnsureFunctionSessionReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.EnsureFunctionSession(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// StartFunctionConversationOperation starts a new business request or resumes
// the current one after clarification or a continuation boundary.
// @tags Func Operation
// @summary 开始功能会话操作
// @router /func-operation/functions/{id}/conversation-operations [POST]
func (f *FuncOperationServerController) StartFunctionConversationOperation(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.StartFunctionConversationOperationReq{Id: ctx.Params("id")}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *funcoperation.StartFunctionConversationOperationReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.StartFunctionConversationOperation(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// GetLatestFunctionConversationOperation
// @tags Func Operation
// @summary 获取功能会话当前操作
// @router /func-operation/functions/{id}/conversation-operations/latest [GET]
func (f *FuncOperationServerController) GetLatestFunctionConversationOperation(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.GetLatestFunctionConversationOperationReq{Id: ctx.Params("id"), Conversation: ctx.Query("conversation")}
	handler := func(ctx context.Context, req *funcoperation.GetLatestFunctionConversationOperationReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetLatestFunctionConversationOperation(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ListFunctionDataForms
// @tags Func Operation
// @summary 获取功能数据库表单列表
// @router /func-operation/functions/{id}/data-forms [GET]
func (f *FuncOperationServerController) ListFunctionDataForms(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.ListFunctionDataFormsReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *funcoperation.ListFunctionDataFormsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListFunctionDataForms(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteFunctionDataForm
// @tags Func Operation
// @summary 删除功能数据库表单
// @router /func-operation/functions/{id}/data-forms/{name} [DELETE]
func (f *FuncOperationServerController) DeleteFunctionDataForm(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.DeleteFunctionDataFormReq{Id: ctx.Params("id"), Name: ctx.Params("name")}
	handler := func(ctx context.Context, req *funcoperation.DeleteFunctionDataFormReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DeleteFunctionDataForm(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteFunction
// @tags Func Operation
// @summary 删除功能
// @router /func-operation/functions/{id} [DELETE]
func (f *FuncOperationServerController) DeleteFunction(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.DeleteFunctionReq{
		Id: ctx.Params("id"),
	}
	handler := func(ctx context.Context, req *funcoperation.DeleteFunctionReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DeleteFunction(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
