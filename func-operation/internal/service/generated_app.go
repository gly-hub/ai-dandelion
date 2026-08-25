package service

import (
	"context"
	"encoding/json"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/logic"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/runtime/generatedapp"
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
	result, invokeErr := s.appLogic.PreviewInvoke(ctx, req.GetId(), payload)
	return previewInvokeResponse(result, invokeErr, out), nil
}

func (s *FuncOperationService) StreamFunctionPreview(req *funcoperation.StreamFunctionPreviewReq, stream funcoperation.FuncOperationService_StreamFunctionPreviewServer) error {
	payload := json.RawMessage(req.GetPayload())
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	result, invokeErr := s.appLogic.PreviewInvokeWithObserver(stream.Context(), req.GetId(), payload, func(event generatedapp.ExecutionLogEvent) {
		// A later Send will return the terminal transport error. WASI execution
		// is kept isolated from a dropped preview connection.
		_ = stream.Send(&funcoperation.StreamFunctionPreviewResp{Content: previewStreamLogContent(event), Stream: event.Stream})
	})
	completed := previewInvokeResponse(result, invokeErr, &funcoperation.InvokeFunctionPreviewResp{})
	return stream.Send(&funcoperation.StreamFunctionPreviewResp{Content: previewStreamCompletedContent(completed), Stream: "runtime", Done: true, Result: completed})
}

func previewStreamLogContent(event generatedapp.ExecutionLogEvent) string {
	content, _ := json.Marshal(struct {
		Type string `json:"type"`
		generatedapp.ExecutionLogEvent
	}{Type: "log", ExecutionLogEvent: event})
	return string(content)
}

func previewStreamCompletedContent(result *funcoperation.InvokeFunctionPreviewResp) string {
	content, _ := json.Marshal(struct {
		Type   string                                   `json:"type"`
		Result *funcoperation.InvokeFunctionPreviewResp `json:"result"`
	}{Type: "complete", Result: result})
	return string(content)
}

func previewInvokeResponse(result generatedapp.InvokeResult, invokeErr error, out *funcoperation.InvokeFunctionPreviewResp) *funcoperation.InvokeFunctionPreviewResp {
	if out == nil {
		out = &funcoperation.InvokeFunctionPreviewResp{}
	}
	out.AppId, out.Version, out.Export = result.AppID, result.Version, result.Export
	out.Result, out.Response, out.Duration = result.Result, result.Response, result.Duration
	out.Runtime, out.ModuleLen = result.Runtime, int32(result.ModuleLen)
	out.BackendSource, out.BackendModule, out.ExecutionLogId = result.BackendSource, result.BackendModule, result.ExecutionLogID
	if invokeErr != nil {
		out.ErrorCode, out.ErrorMessage, out.Stage, out.Hint = logic.ClassifyInvokeError(invokeErr)
	}
	return out
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
	out.ExecutionLogId = invokeOut.ExecutionLogId
	return
}

func (s *FuncOperationService) ListFunctionExecutionLogs(ctx context.Context, req *funcoperation.ListFunctionExecutionLogsReq) (out *funcoperation.ListFunctionExecutionLogsResp, err error) {
	grpcep.InitResponse(&out)
	out.Logs, out.Total, err = s.appLogic.ListExecutionLogs(ctx, req)
	return
}

func (s *FuncOperationService) GetFunctionExecutionLog(ctx context.Context, req *funcoperation.GetFunctionExecutionLogReq) (out *funcoperation.GetFunctionExecutionLogResp, err error) {
	grpcep.InitResponse(&out)
	out.Log, err = s.appLogic.GetExecutionLog(ctx, req.GetFunctionId(), req.GetId())
	return
}
