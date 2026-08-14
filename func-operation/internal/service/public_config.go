package service

import (
	"context"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *FuncOperationService) ListPublicConfigs(ctx context.Context, req *funcoperation.ListPublicConfigsReq) (out *funcoperation.ListPublicConfigsResp, err error) {
	grpcep.InitResponse(&out)
	out.Configs, err = s.publicConfigLogic.List(ctx, req)
	return
}

func (s *FuncOperationService) CreatePublicConfig(ctx context.Context, req *funcoperation.CreatePublicConfigReq) (out *funcoperation.CreatePublicConfigResp, err error) {
	grpcep.InitResponse(&out)
	out.Config, err = s.publicConfigLogic.Create(ctx, req)
	return
}

func (s *FuncOperationService) UpdatePublicConfig(ctx context.Context, req *funcoperation.UpdatePublicConfigReq) (out *funcoperation.UpdatePublicConfigResp, err error) {
	grpcep.InitResponse(&out)
	out.Config, err = s.publicConfigLogic.Update(ctx, req)
	return
}

func (s *FuncOperationService) ListPublicConfigVersions(ctx context.Context, req *funcoperation.ListPublicConfigVersionsReq) (out *funcoperation.ListPublicConfigVersionsResp, err error) {
	grpcep.InitResponse(&out)
	out.Versions, err = s.publicConfigLogic.ListVersions(ctx, req)
	return
}

func (s *FuncOperationService) RollbackPublicConfig(ctx context.Context, req *funcoperation.RollbackPublicConfigReq) (out *funcoperation.RollbackPublicConfigResp, err error) {
	grpcep.InitResponse(&out)
	out.Config, err = s.publicConfigLogic.Rollback(ctx, req)
	return
}

func (s *FuncOperationService) RotatePublicConfigImportKey(ctx context.Context, req *funcoperation.RotatePublicConfigImportKeyReq) (out *funcoperation.RotatePublicConfigImportKeyResp, err error) {
	grpcep.InitResponse(&out)
	out.ImportKey, err = s.publicConfigLogic.RotateImportKey(ctx, req)
	return
}

func (s *FuncOperationService) ImportPublicConfigs(ctx context.Context, req *funcoperation.ImportPublicConfigsReq) (out *funcoperation.ImportPublicConfigsResp, err error) {
	grpcep.InitResponse(&out)
	out.Configs, err = s.publicConfigLogic.Import(ctx, req)
	return
}
