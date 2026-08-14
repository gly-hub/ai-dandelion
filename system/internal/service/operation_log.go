package service

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *SystemService) ListOperationLogs(ctx context.Context, req *systemproto.ListOperationLogsReq) (
	out *systemproto.ListOperationLogsResp, err error) {
	grpcep.InitResponse(&out)
	out.Logs, out.Total, err = s.operationLogLogic.ListOperationLogs(ctx, req)
	return
}
