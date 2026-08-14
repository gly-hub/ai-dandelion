package service

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *SystemService) CreateUpload(ctx context.Context, req *systemproto.CreateUploadReq) (out *systemproto.CreateUploadResp, err error) {
	grpcep.InitResponse(&out)
	out.Upload, err = s.uploadLogic.Create(ctx, req)
	return
}
func (s *SystemService) GetUploadPartURL(ctx context.Context, req *systemproto.GetUploadPartURLReq) (out *systemproto.GetUploadPartURLResp, err error) {
	grpcep.InitResponse(&out)
	out.Url, err = s.uploadLogic.PartURL(ctx, req)
	return
}
func (s *SystemService) CompleteUpload(ctx context.Context, req *systemproto.CompleteUploadReq) (out *systemproto.CompleteUploadResp, err error) {
	grpcep.InitResponse(&out)
	out.Url, err = s.uploadLogic.Complete(ctx, req)
	return
}
func (s *SystemService) GetUploadPreviewURL(ctx context.Context, req *systemproto.GetUploadURLReq) (out *systemproto.GetUploadURLResp, err error) {
	grpcep.InitResponse(&out)
	out.Url, err = s.uploadLogic.PreviewURL(ctx, req)
	return
}
func (s *SystemService) GetUploadDownloadURL(ctx context.Context, req *systemproto.GetUploadURLReq) (out *systemproto.GetUploadURLResp, err error) {
	grpcep.InitResponse(&out)
	out.Url, err = s.uploadLogic.DownloadURL(ctx, req)
	return
}
