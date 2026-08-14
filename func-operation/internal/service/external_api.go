package service

import (
	"context"
	funcoperation "github.com/team-dandelion/ai-dandelion/proto/func-operation"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *FuncOperationService) ListExternalAPIClients(ctx context.Context, req *funcoperation.ListExternalAPIClientsReq) (out *funcoperation.ListExternalAPIClientsResp, err error) {
	grpcep.InitResponse(&out)
	out.Clients, err = s.externalAPILogic.ListClients(ctx, req)
	return
}
func (s *FuncOperationService) CreateExternalAPIClient(ctx context.Context, req *funcoperation.CreateExternalAPIClientReq) (out *funcoperation.CreateExternalAPIClientResp, err error) {
	grpcep.InitResponse(&out)
	out.Client, err = s.externalAPILogic.CreateClient(ctx, req)
	return
}
func (s *FuncOperationService) UpdateExternalAPIClient(ctx context.Context, req *funcoperation.UpdateExternalAPIClientReq) (out *funcoperation.UpdateExternalAPIClientResp, err error) {
	grpcep.InitResponse(&out)
	out.Client, err = s.externalAPILogic.UpdateClient(ctx, req)
	return
}
func (s *FuncOperationService) DeleteExternalAPIClient(ctx context.Context, req *funcoperation.DeleteExternalAPIClientReq) (out *funcoperation.DeleteExternalAPIClientResp, err error) {
	grpcep.InitResponse(&out)
	err = s.externalAPILogic.DeleteClient(ctx, req)
	return
}
func (s *FuncOperationService) ListDeletedExternalAPIClients(ctx context.Context, req *funcoperation.ListDeletedExternalAPIClientsReq) (out *funcoperation.ListDeletedExternalAPIClientsResp, err error) {
	grpcep.InitResponse(&out)
	out.Clients, err = s.externalAPILogic.ListDeletedClients(ctx, req)
	return
}
func (s *FuncOperationService) PurgeExternalAPIClient(ctx context.Context, req *funcoperation.PurgeExternalAPIClientReq) (out *funcoperation.PurgeExternalAPIClientResp, err error) {
	grpcep.InitResponse(&out)
	err = s.externalAPILogic.PurgeClient(ctx, req)
	return
}
func (s *FuncOperationService) RotateExternalAPIImportKey(ctx context.Context, req *funcoperation.RotateExternalAPIImportKeyReq) (out *funcoperation.RotateExternalAPIImportKeyResp, err error) {
	grpcep.InitResponse(&out)
	out.SwaggerImportKey, err = s.externalAPILogic.RotateImportKey(ctx, req)
	return
}
func (s *FuncOperationService) ListExternalAPIs(ctx context.Context, req *funcoperation.ListExternalAPIsReq) (out *funcoperation.ListExternalAPIsResp, err error) {
	grpcep.InitResponse(&out)
	out.Apis, err = s.externalAPILogic.ListAPIs(ctx, req)
	return
}
func (s *FuncOperationService) ListExternalAPIGroups(ctx context.Context, req *funcoperation.ListExternalAPIGroupsReq) (out *funcoperation.ListExternalAPIGroupsResp, err error) {
	grpcep.InitResponse(&out)
	out.Groups, err = s.externalAPILogic.ListGroups(ctx, req)
	return
}
func (s *FuncOperationService) CreateExternalAPIGroup(ctx context.Context, req *funcoperation.CreateExternalAPIGroupReq) (out *funcoperation.CreateExternalAPIGroupResp, err error) {
	grpcep.InitResponse(&out)
	out.Group, err = s.externalAPILogic.CreateGroup(ctx, req)
	return
}
func (s *FuncOperationService) CreateExternalAPI(ctx context.Context, req *funcoperation.CreateExternalAPIReq) (out *funcoperation.CreateExternalAPIResp, err error) {
	grpcep.InitResponse(&out)
	out.Api, err = s.externalAPILogic.CreateAPI(ctx, req)
	return
}
func (s *FuncOperationService) UpdateExternalAPI(ctx context.Context, req *funcoperation.UpdateExternalAPIReq) (out *funcoperation.UpdateExternalAPIResp, err error) {
	grpcep.InitResponse(&out)
	out.Api, err = s.externalAPILogic.UpdateAPI(ctx, req)
	return
}
func (s *FuncOperationService) DeleteExternalAPI(ctx context.Context, req *funcoperation.DeleteExternalAPIReq) (out *funcoperation.DeleteExternalAPIResp, err error) {
	grpcep.InitResponse(&out)
	err = s.externalAPILogic.DeleteAPI(ctx, req)
	return
}
func (s *FuncOperationService) ImportExternalAPIDocument(ctx context.Context, req *funcoperation.ImportExternalAPIDocumentReq) (out *funcoperation.ImportExternalAPIDocumentResp, err error) {
	grpcep.InitResponse(&out)
	out.CreatedCount, out.UpdatedCount, out.Groups, out.Apis, err = s.externalAPILogic.ImportDocument(ctx, req)
	return
}
func (s *FuncOperationService) UploadExternalAPIDocument(ctx context.Context, req *funcoperation.UploadExternalAPIDocumentReq) (out *funcoperation.UploadExternalAPIDocumentResp, err error) {
	grpcep.InitResponse(&out)
	out.CreatedCount, out.UpdatedCount, err = s.externalAPILogic.UploadDocument(ctx, req)
	return
}
func (s *FuncOperationService) TestExternalAPI(ctx context.Context, req *funcoperation.TestExternalAPIReq) (out *funcoperation.TestExternalAPIResp, err error) {
	grpcep.InitResponse(&out)
	testResult, err := s.externalAPILogic.TestExternalAPI(ctx, req)
	if err == nil {
		out.StatusCode = testResult.StatusCode
		out.ResponseHeadersJson = testResult.ResponseHeadersJson
		out.ResponseBodyJson = testResult.ResponseBodyJson
		out.DurationMs = testResult.DurationMs
	}
	return out, err
}
