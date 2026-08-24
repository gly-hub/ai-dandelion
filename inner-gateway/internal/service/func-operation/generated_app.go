package funcoperation

import (
	"context"
	"encoding/json"
	"path"
	"strings"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

func (f *FuncOperationServerController) ListGeneratedApps(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.ListGeneratedAppsReq{}
	handler := func(ctx context.Context, req *funcoperation.ListGeneratedAppsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListGeneratedApps(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

func (f *FuncOperationServerController) ReloadGeneratedApps(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.ReloadGeneratedAppsReq{}
	handler := func(ctx context.Context, req *funcoperation.ReloadGeneratedAppsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ReloadGeneratedApps(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

func (f *FuncOperationServerController) CleanupGeneratedFunctionMenus(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.CleanupGeneratedFunctionMenusReq{}
	handler := func(ctx context.Context, req *funcoperation.CleanupGeneratedFunctionMenusReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CleanupGeneratedFunctionMenus(ctx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

func (f *FuncOperationServerController) ListOutboxEvents(ctx *fiber.Ctx) error {
	param := &funcoperation.ListOutboxEventsReq{}
	handler := func(rpcCtx context.Context, req *funcoperation.ListOutboxEventsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.ListOutboxEvents(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) ReplayOutboxEvents(ctx *fiber.Ctx) error {
	param := &funcoperation.ReplayOutboxEventsReq{}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(rpcCtx context.Context, req *funcoperation.ReplayOutboxEventsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.ReplayOutboxEvents(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) GetGeneratedAppFrontend(ctx *fiber.Ctx) error {
	filePath := strings.TrimPrefix(path.Clean("/frontend/"+ctx.Params("*")), "/")
	if ctx.Params("*") == "" {
		filePath = "frontend.js"
	}
	rpcParam := &funcoperation.GetGeneratedAppFrontendReq{
		Id:   ctx.Params("id"),
		Path: filePath,
	}
	client, err := f.getFuncOperationClient(ctx.Context())
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	resp, err := client.GetGeneratedAppFrontend(f.baseHandler.RPCCtx(ctx), rpcParam)
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	ctx.Set("Content-Type", "text/javascript; charset=utf-8")
	ctx.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	return ctx.SendString(resp.GetCode())
}

func (f *FuncOperationServerController) GetFunctionPreviewFrontend(ctx *fiber.Ctx) error {
	filePath := strings.TrimPrefix(path.Clean("/frontend/"+ctx.Params("*")), "/")
	if ctx.Params("*") == "" {
		filePath = "frontend.js"
	}
	client, err := f.getFuncOperationClient(ctx.Context())
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	resp, err := client.GetFunctionPreviewFrontend(f.baseHandler.RPCCtx(ctx), &funcoperation.GetFunctionPreviewFrontendReq{Id: ctx.Params("id"), Path: filePath})
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	ctx.Set("Content-Type", "text/javascript; charset=utf-8")
	ctx.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	return ctx.SendString(resp.GetCode())
}

func (f *FuncOperationServerController) GetFunctionPreviewBundle(ctx *fiber.Ctx) error {
	param := &funcoperation.GetFunctionPreviewBundleReq{Id: ctx.Params("id")}
	handler := func(rpcCtx context.Context, req *funcoperation.GetFunctionPreviewBundleReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.GetFunctionPreviewBundle(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) InvokeFunctionPreview(ctx *fiber.Ctx) error {
	forwardGeneratedAppRequestID(ctx)
	payload := "{}"
	if len(ctx.Body()) > 0 {
		var input struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(ctx.Body(), &input); err != nil {
			return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
		}
		if len(input.Payload) > 0 {
			payload = string(input.Payload)
		}
	}
	client, err := f.getFuncOperationClient(ctx.Context())
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	resp, err := client.InvokeFunctionPreview(f.baseHandler.RPCCtx(ctx), &funcoperation.InvokeFunctionPreviewReq{Id: ctx.Params("id"), Payload: payload})
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	return f.baseHandler.Response(ctx, grpcep.JsonResponse{Data: resp}, nil)
}

func (f *FuncOperationServerController) StreamFunctionPreview(ctx *fiber.Ctx) error {
	forwardGeneratedAppRequestID(ctx)
	rpcParam := &funcoperation.StreamFunctionPreviewReq{Id: ctx.Params("id")}
	if err := f.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	streamFunc := func(rpcCtx context.Context, req interface{}) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.StreamFunctionPreview(rpcCtx, req.(*funcoperation.StreamFunctionPreviewReq))
	}
	return f.baseHandler.RPCStream(ctx, rpcParam, streamFunc)
}

func (f *FuncOperationServerController) ListFunctionExecutionLogs(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.ListFunctionExecutionLogsReq{
		FunctionId: ctx.Params("id"), Limit: int32(ctx.QueryInt("limit", 0)), Page: int32(ctx.QueryInt("page", 1)), Query: ctx.Query("query"), Status: ctx.Query("status"), InvocationType: ctx.Query("invocationType"),
		StartTime: int64(ctx.QueryInt("startTime", 0)), EndTime: int64(ctx.QueryInt("endTime", 0)), RequestId: ctx.Query("requestId"),
	}
	handler := func(rpcCtx context.Context, req *funcoperation.ListFunctionExecutionLogsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.ListFunctionExecutionLogs(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

func (f *FuncOperationServerController) GetFunctionExecutionLog(ctx *fiber.Ctx) error {
	rpcParam := &funcoperation.GetFunctionExecutionLogReq{FunctionId: ctx.Params("id"), Id: ctx.Params("logId")}
	handler := func(rpcCtx context.Context, req *funcoperation.GetFunctionExecutionLogReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.GetFunctionExecutionLog(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

func (f *FuncOperationServerController) GetGeneratedAppFrontendBundle(ctx *fiber.Ctx) error {
	param := &funcoperation.GetGeneratedAppFrontendBundleReq{Id: ctx.Params("id")}
	handler := func(rpcCtx context.Context, req *funcoperation.GetGeneratedAppFrontendBundleReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(rpcCtx)
		if err != nil {
			return nil, err
		}
		return client.GetGeneratedAppFrontendBundle(rpcCtx, req)
	}
	return f.baseHandler.GRPCCall(ctx, param, handler)
}

func (f *FuncOperationServerController) InvokeGeneratedApp(ctx *fiber.Ctx) error {
	forwardGeneratedAppRequestID(ctx)
	rpcParam := &funcoperation.InvokeGeneratedAppReq{Id: ctx.Params("id")}
	rpcParam.Payload = "{}"
	if len(ctx.Body()) > 0 {
		var payload struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(ctx.Body(), &payload); err != nil {
			return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
		}
		if len(payload.Payload) > 0 {
			rpcParam.Payload = string(payload.Payload)
		}
	}

	client, err := f.getFuncOperationClient(ctx.Context())
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	resp, err := client.InvokeGeneratedApp(f.baseHandler.RPCCtx(ctx), rpcParam)
	if err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.InternalErrCode, err.Error()))
	}
	return f.baseHandler.Response(ctx, grpcep.JsonResponse{Data: resp}, nil)
}

// TraceMiddleware assigns the external request ID to this local value. The
// base gRPC adapter forwards request headers, so copy generated IDs into the
// trace header when the caller did not provide one themselves.
func forwardGeneratedAppRequestID(ctx *fiber.Ctx) {
	if strings.TrimSpace(ctx.Get("X-Trace-ID")) != "" {
		return
	}
	if requestID, ok := ctx.Locals("request_id").(string); ok && strings.TrimSpace(requestID) != "" {
		ctx.Request().Header.Set("X-Trace-ID", requestID)
	}
}
