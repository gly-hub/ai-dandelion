package system

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

func (s *SystemServerController) CreateUpload(ctx *fiber.Ctx) error {
	req := &systemproto.CreateUploadReq{}
	if err := s.baseHandler.ParseJson(ctx, req); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	return s.baseHandler.GRPCCall(ctx, req, func(ctx context.Context, req *systemproto.CreateUploadReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CreateUpload(ctx, req)
	})
}
func (s *SystemServerController) GetUploadPartURL(ctx *fiber.Ctx) error {
	part, err := ctx.ParamsInt("part_number")
	if err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, "part_number is illegal"))
	}
	req := &systemproto.GetUploadPartURLReq{Uuid: ctx.Params("uuid"), UploadId: ctx.Query("upload_id"), PartNumber: int32(part), FileSize: int64(ctx.QueryInt("file_size", 0))}
	return s.baseHandler.GRPCCall(ctx, req, func(ctx context.Context, req *systemproto.GetUploadPartURLReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetUploadPartURL(ctx, req)
	})
}
func (s *SystemServerController) CompleteUpload(ctx *fiber.Ctx) error {
	req := &systemproto.CompleteUploadReq{Uuid: ctx.Params("uuid")}
	if err := s.baseHandler.ParseJson(ctx, req); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	req.Uuid = ctx.Params("uuid")
	return s.baseHandler.GRPCCall(ctx, req, func(ctx context.Context, req *systemproto.CompleteUploadReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CompleteUpload(ctx, req)
	})
}
func (s *SystemServerController) GetUploadPreviewURL(ctx *fiber.Ctx) error {
	req := &systemproto.GetUploadURLReq{Uuid: ctx.Params("uuid"), ExpiresSeconds: int64(ctx.QueryInt("expires_seconds", 0))}
	return s.baseHandler.GRPCCall(ctx, req, func(ctx context.Context, req *systemproto.GetUploadURLReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetUploadPreviewURL(ctx, req)
	})
}
func (s *SystemServerController) GetUploadDownloadURL(ctx *fiber.Ctx) error {
	req := &systemproto.GetUploadURLReq{Uuid: ctx.Params("uuid"), ExpiresSeconds: int64(ctx.QueryInt("expires_seconds", 0)), FileName: ctx.Query("file_name")}
	return s.baseHandler.GRPCCall(ctx, req, func(ctx context.Context, req *systemproto.GetUploadURLReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetUploadDownloadURL(ctx, req)
	})
}
