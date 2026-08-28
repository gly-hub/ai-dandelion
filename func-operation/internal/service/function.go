package service

import (
	"context"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *FuncOperationService) ListFunctions(ctx context.Context, req *funcoperation.ListFunctionsReq) (
	out *funcoperation.ListFunctionsResp, err error) {
	grpcep.InitResponse(&out)
	out.Functions, err = s.functionLogic.ListFunctions(ctx, req)
	return
}

func (s *FuncOperationService) CreateFunction(ctx context.Context, req *funcoperation.CreateFunctionReq) (
	out *funcoperation.CreateFunctionResp, err error) {
	grpcep.InitResponse(&out)
	out.Function, err = s.functionLogic.CreateFunction(ctx, req)
	return
}

func (s *FuncOperationService) UpdateFunction(ctx context.Context, req *funcoperation.UpdateFunctionReq) (
	out *funcoperation.UpdateFunctionResp, err error) {
	grpcep.InitResponse(&out)
	out.Function, err = s.functionLogic.UpdateFunction(ctx, req)
	return
}

func (s *FuncOperationService) LoadFunctionDocument(ctx context.Context, req *funcoperation.LoadFunctionDocumentReq) (
	out *funcoperation.LoadFunctionDocumentResp, err error) {
	out, err = s.functionLogic.LoadFunctionDocument(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) CommitFunctionDocument(ctx context.Context, req *funcoperation.CommitFunctionDocumentReq) (
	out *funcoperation.CommitFunctionDocumentResp, err error) {
	out, err = s.functionLogic.CommitFunctionDocument(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) MaterializeFunctionApp(ctx context.Context, req *funcoperation.MaterializeFunctionAppReq) (
	out *funcoperation.MaterializeFunctionAppResp, err error) {
	out, err = s.functionLogic.MaterializeFunctionApp(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) LoadFunctionCodeState(ctx context.Context, req *funcoperation.LoadFunctionCodeStateReq) (
	out *funcoperation.LoadFunctionCodeStateResp, err error) {
	out, err = s.functionLogic.LoadFunctionCodeState(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) TouchFunctionCodeDraft(ctx context.Context, req *funcoperation.TouchFunctionCodeDraftReq) (
	out *funcoperation.TouchFunctionCodeDraftResp, err error) {
	out, err = s.functionLogic.TouchFunctionCodeDraft(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) ApplyFunctionCode(ctx context.Context, req *funcoperation.ApplyFunctionCodeReq) (
	out *funcoperation.ApplyFunctionCodeResp, err error) {
	out, err = s.functionLogic.ApplyFunctionCode(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) EnsureFunctionSession(ctx context.Context, req *funcoperation.EnsureFunctionSessionReq) (
	out *funcoperation.EnsureFunctionSessionResp, err error) {
	out, err = s.functionLogic.EnsureFunctionSession(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) StartFunctionConversationOperation(ctx context.Context, req *funcoperation.StartFunctionConversationOperationReq) (
	out *funcoperation.StartFunctionConversationOperationResp, err error) {
	grpcep.InitResponse(&out)
	out.Operation, err = s.conversationOperationLogic.Start(ctx, req)
	return
}

func (s *FuncOperationService) GetLatestFunctionConversationOperation(ctx context.Context, req *funcoperation.GetLatestFunctionConversationOperationReq) (
	out *funcoperation.GetLatestFunctionConversationOperationResp, err error) {
	grpcep.InitResponse(&out)
	out.Operation, err = s.conversationOperationLogic.GetLatest(ctx, req)
	return
}

func (s *FuncOperationService) FinishFunctionConversationOperation(ctx context.Context, req *funcoperation.FinishFunctionConversationOperationReq) (
	out *funcoperation.FinishFunctionConversationOperationResp, err error) {
	grpcep.InitResponse(&out)
	out.Operation, err = s.conversationOperationLogic.Finish(ctx, req)
	return
}

func (s *FuncOperationService) SubmitFunctionConversationProgress(ctx context.Context, req *funcoperation.SubmitFunctionConversationProgressReq) (
	out *funcoperation.SubmitFunctionConversationProgressResp, err error) {
	grpcep.InitResponse(&out)
	out.Operation, out.AlreadySubmitted, err = s.conversationOperationLogic.SubmitProgress(ctx, req)
	return
}

func (s *FuncOperationService) ListFunctionDataForms(ctx context.Context, req *funcoperation.ListFunctionDataFormsReq) (
	out *funcoperation.ListFunctionDataFormsResp, err error) {
	out, err = s.functionLogic.ListFunctionDataForms(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) DeleteFunctionDataForm(ctx context.Context, req *funcoperation.DeleteFunctionDataFormReq) (
	out *funcoperation.DeleteFunctionDataFormResp, err error) {
	out, err = s.functionLogic.DeleteFunctionDataForm(ctx, req)
	if out == nil {
		grpcep.InitResponse(&out)
	} else if out.CommonResp == nil {
		out.CommonResp = &grpcep.CommonResp{}
	}
	return
}

func (s *FuncOperationService) DeleteFunction(ctx context.Context, req *funcoperation.DeleteFunctionReq) (
	out *funcoperation.DeleteFunctionResp, err error) {
	grpcep.InitResponse(&out)
	err = s.functionLogic.DeleteFunction(ctx, req)
	return
}
