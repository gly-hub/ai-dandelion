package service

import (
	"context"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *SystemService) ListOperationLogs(ctx context.Context, req *systemproto.ListOperationLogsReq) (
	out *systemproto.ListOperationLogsResp, err error) {
	grpcep.InitResponse(&out)
	out.Logs, out.Total, err = s.operationLogLogic.ListOperationLogs(ctx, req)
	return
}
