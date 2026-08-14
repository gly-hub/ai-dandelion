package service

import (
	"context"
	"encoding/json"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *FuncOperationService) GetFunctionPreviewFrontend(ctx context.Context, req *funcoperation.GetFunctionPreviewFrontendReq) (out *funcoperation.GetFunctionPreviewFrontendResp, err error) {
	grpcep.InitResponse(&out)
	out.Code, err = s.appLogic.PreviewFrontend(ctx, req.GetId(), req.GetPath())
	return
}

func (s *FuncOperationService) GetFunctionPreviewBundle(ctx context.Context, req *funcoperation.GetFunctionPreviewBundleReq) (out *funcoperation.GetFunctionPreviewBundleResp, err error) {
	grpcep.InitResponse(&out)
	out.Bundle, err = s.appLogic.PreviewBundle(ctx, req.GetId())
	return
}

func (s *FuncOperationService) InvokeFunctionPreview(ctx context.Context, req *funcoperation.InvokeFunctionPreviewReq) (out *funcoperation.InvokeFunctionPreviewResp, err error) {
	grpcep.InitResponse(&out)
	payload := json.RawMessage(req.GetPayload())
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	result, err := s.appLogic.PreviewInvoke(ctx, req.GetId(), payload)
	if err != nil {
		return out, err
	}
	out.AppId, out.Version, out.Export = result.AppID, result.Version, result.Export
	out.Result, out.Response, out.Duration = result.Result, result.Response, result.Duration
	out.Runtime, out.ModuleLen = result.Runtime, int32(result.ModuleLen)
	out.BackendSource, out.BackendModule = result.BackendSource, result.BackendModule
	return out, nil
}

func (s *FuncOperationService) GetGeneratedAppFrontendBundle(ctx context.Context, req *funcoperation.GetGeneratedAppFrontendBundleReq) (out *funcoperation.GetGeneratedAppFrontendBundleResp, err error) {
	grpcep.InitResponse(&out)
	out.Bundle, err = s.appLogic.GetFrontendBundle(ctx, req.GetId())
	return
}

func (s *FuncOperationService) ListGeneratedApps(ctx context.Context, req *funcoperation.ListGeneratedAppsReq) (
	out *funcoperation.ListGeneratedAppsResp, err error) {
	grpcep.InitResponse(&out)
	out.Apps, err = s.appLogic.ListGeneratedApps(ctx, req)
	return
}

func (s *FuncOperationService) ReloadGeneratedApps(ctx context.Context, req *funcoperation.ReloadGeneratedAppsReq) (
	out *funcoperation.ReloadGeneratedAppsResp, err error) {
	grpcep.InitResponse(&out)
	out.Apps, err = s.appLogic.ReloadGeneratedApps(ctx, req)
	return
}

func (s *FuncOperationService) CleanupGeneratedFunctionMenus(ctx context.Context, req *funcoperation.CleanupGeneratedFunctionMenusReq) (
	out *funcoperation.CleanupGeneratedFunctionMenusResp, err error) {
	grpcep.InitResponse(&out)
	out, err = s.appLogic.CleanupGeneratedFunctionMenus(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) GetGeneratedAppFrontend(ctx context.Context, req *funcoperation.GetGeneratedAppFrontendReq) (
	out *funcoperation.GetGeneratedAppFrontendResp, err error) {
	grpcep.InitResponse(&out)
	out.Code, err = s.appLogic.GetFrontend(ctx, req)
	return
}

func (s *FuncOperationService) InvokeGeneratedApp(ctx context.Context, req *funcoperation.InvokeGeneratedAppReq) (
	out *funcoperation.InvokeGeneratedAppResp, err error) {
	grpcep.InitResponse(&out)
	invokeOut, err := s.appLogic.Invoke(ctx, req)
	if err != nil {
		return
	}
	out.AppId = invokeOut.AppId
	out.Version = invokeOut.Version
	out.Export = invokeOut.Export
	out.Result = invokeOut.Result
	out.Response = invokeOut.Response
	out.Duration = invokeOut.Duration
	out.Runtime = invokeOut.Runtime
	out.ModuleLen = invokeOut.ModuleLen
	out.BackendSource = invokeOut.BackendSource
	out.BackendModule = invokeOut.BackendModule
	out.ErrorCode = invokeOut.ErrorCode
	out.ErrorMessage = invokeOut.ErrorMessage
	out.Stage = invokeOut.Stage
	out.Hint = invokeOut.Hint
	return
}
