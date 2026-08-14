package funcoperation

import (
	"context"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

func (f *FuncOperationServerController) ListPublicConfigs(ctx *fiber.Ctx) error {
	param := &funcoperation.ListPublicConfigsReq{}
	handler := func(rpcCtx context.Context, req *funcoperation.ListPublicConfigsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.ListPublicConfigs(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) CreatePublicConfig(ctx *fiber.Ctx) error {
	param := &funcoperation.CreatePublicConfigReq{}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(rpcCtx context.Context, req *funcoperation.CreatePublicConfigReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.CreatePublicConfig(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) UpdatePublicConfig(ctx *fiber.Ctx) error {
	param := &funcoperation.UpdatePublicConfigReq{ConfigKey: ctx.Params("key")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ConfigKey = ctx.Params("key")
	handler := func(rpcCtx context.Context, req *funcoperation.UpdatePublicConfigReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.UpdatePublicConfig(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) ListPublicConfigVersions(ctx *fiber.Ctx) error {
	param := &funcoperation.ListPublicConfigVersionsReq{ConfigKey: ctx.Params("key")}
	handler := func(rpcCtx context.Context, req *funcoperation.ListPublicConfigVersionsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.ListPublicConfigVersions(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) RollbackPublicConfig(ctx *fiber.Ctx) error {
	param := &funcoperation.RollbackPublicConfigReq{ConfigKey: ctx.Params("key")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ConfigKey = ctx.Params("key")
	handler := func(rpcCtx context.Context, req *funcoperation.RollbackPublicConfigReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.RollbackPublicConfig(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}
