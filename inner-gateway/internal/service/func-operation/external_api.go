package funcoperation

import (
	"context"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

func (f *FuncOperationServerController) ListExternalAPIClients(ctx *fiber.Ctx) error {
	param := &funcoperation.ListExternalAPIClientsReq{}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.ListExternalAPIClientsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.ListExternalAPIClients(r, req)
	})
}
func (f *FuncOperationServerController) CreateExternalAPIClient(ctx *fiber.Ctx) error {
	param := &funcoperation.CreateExternalAPIClientReq{}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.CreateExternalAPIClientReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.CreateExternalAPIClient(r, req)
	})
}
func (f *FuncOperationServerController) UpdateExternalAPIClient(ctx *fiber.Ctx) error {
	param := &funcoperation.UpdateExternalAPIClientReq{ClientKey: ctx.Params("key")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ClientKey = ctx.Params("key")
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.UpdateExternalAPIClientReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.UpdateExternalAPIClient(r, req)
	})
}
func (f *FuncOperationServerController) DeleteExternalAPIClient(ctx *fiber.Ctx) error {
	param := &funcoperation.DeleteExternalAPIClientReq{ClientKey: ctx.Params("key")}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.DeleteExternalAPIClientReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.DeleteExternalAPIClient(r, req)
	})
}
func (f *FuncOperationServerController) RotateExternalAPIImportKey(ctx *fiber.Ctx) error {
	param := &funcoperation.RotateExternalAPIImportKeyReq{ClientKey: ctx.Params("key")}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.RotateExternalAPIImportKeyReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.RotateExternalAPIImportKey(r, req)
	})
}
func (f *FuncOperationServerController) ListDeletedExternalAPIClients(ctx *fiber.Ctx) error {
	param := &funcoperation.ListDeletedExternalAPIClientsReq{}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.ListDeletedExternalAPIClientsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.ListDeletedExternalAPIClients(r, req)
	})
}
func (f *FuncOperationServerController) PurgeExternalAPIClient(ctx *fiber.Ctx) error {
	param := &funcoperation.PurgeExternalAPIClientReq{ClientKey: ctx.Params("key")}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.PurgeExternalAPIClientReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.PurgeExternalAPIClient(r, req)
	})
}
func (f *FuncOperationServerController) ListExternalAPIs(ctx *fiber.Ctx) error {
	param := &funcoperation.ListExternalAPIsReq{ClientKey: ctx.Params("key")}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.ListExternalAPIsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.ListExternalAPIs(r, req)
	})
}
func (f *FuncOperationServerController) ListExternalAPIGroups(ctx *fiber.Ctx) error {
	param := &funcoperation.ListExternalAPIGroupsReq{ClientKey: ctx.Params("key")}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.ListExternalAPIGroupsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.ListExternalAPIGroups(r, req)
	})
}
func (f *FuncOperationServerController) CreateExternalAPIGroup(ctx *fiber.Ctx) error {
	param := &funcoperation.CreateExternalAPIGroupReq{ClientKey: ctx.Params("key")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ClientKey = ctx.Params("key")
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.CreateExternalAPIGroupReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.CreateExternalAPIGroup(r, req)
	})
}
func (f *FuncOperationServerController) CreateExternalAPI(ctx *fiber.Ctx) error {
	param := &funcoperation.CreateExternalAPIReq{ClientKey: ctx.Params("key")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ClientKey = ctx.Params("key")
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.CreateExternalAPIReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.CreateExternalAPI(r, req)
	})
}
func (f *FuncOperationServerController) UpdateExternalAPI(ctx *fiber.Ctx) error {
	param := &funcoperation.UpdateExternalAPIReq{ClientKey: ctx.Params("key"), ApiKey: ctx.Params("apiKey")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ClientKey = ctx.Params("key")
	param.ApiKey = ctx.Params("apiKey")
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.UpdateExternalAPIReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.UpdateExternalAPI(r, req)
	})
}
func (f *FuncOperationServerController) DeleteExternalAPI(ctx *fiber.Ctx) error {
	param := &funcoperation.DeleteExternalAPIReq{ClientKey: ctx.Params("key"), ApiKey: ctx.Params("apiKey")}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.DeleteExternalAPIReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.DeleteExternalAPI(r, req)
	})
}
func (f *FuncOperationServerController) ImportExternalAPIDocument(ctx *fiber.Ctx) error {
	param := &funcoperation.ImportExternalAPIDocumentReq{ClientKey: ctx.Params("key")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ClientKey = ctx.Params("key")
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.ImportExternalAPIDocumentReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.ImportExternalAPIDocument(r, req)
	})
}
func (f *FuncOperationServerController) UploadExternalAPIDocument(ctx *fiber.Ctx) error {
	param := &funcoperation.UploadExternalAPIDocumentReq{
		ClientKey:    ctx.Params("key"),
		ApiKey:       ctx.Get("X-API-Key"),
		DocumentJson: string(ctx.Body()),
	}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.UploadExternalAPIDocumentReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.UploadExternalAPIDocument(r, req)
	})
}
func (f *FuncOperationServerController) RotatePublicConfigImportKey(ctx *fiber.Ctx) error {
	param := &funcoperation.RotatePublicConfigImportKeyReq{}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.RotatePublicConfigImportKeyReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.RotatePublicConfigImportKey(r, req)
	})
}
func (f *FuncOperationServerController) ImportPublicConfigs(ctx *fiber.Ctx) error {
	param := &funcoperation.ImportPublicConfigsReq{ApiKey: ctx.Get("X-API-Key"), ConfigsJson: string(ctx.Body())}
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.ImportPublicConfigsReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.ImportPublicConfigs(r, req)
	})
}
func (f *FuncOperationServerController) TestExternalAPI(ctx *fiber.Ctx) error {
	param := &funcoperation.TestExternalAPIReq{ClientKey: ctx.Params("key"), ApiKey: ctx.Params("apiKey")}
	if err := f.baseHandler.ParseJson(ctx, param); err != nil {
		return f.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	param.ClientKey = ctx.Params("key")
	param.ApiKey = ctx.Params("apiKey")
	return f.baseHandler.GRPCCall(ctx, param, func(r context.Context, req *funcoperation.TestExternalAPIReq) (interface{}, error) {
		client, err := f.getFuncOperationClient(r)
		if err != nil {
			return nil, err
		}
		return client.TestExternalAPI(r, req)
	})
}
