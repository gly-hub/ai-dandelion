package service

import (
	"context"
	funcoperation "github.com/team-dandelion/ai-dandelion/proto/func-operation"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *FuncOperationService) ListOutboxEvents(ctx context.Context, req *funcoperation.ListOutboxEventsReq) (out *funcoperation.ListOutboxEventsResp, err error) {
	grpcep.InitResponse(&out)
	out.Events, err = s.outboxLogic.List(ctx, req.GetLimit())
	return
}
func (s *FuncOperationService) ReplayOutboxEvents(ctx context.Context, req *funcoperation.ReplayOutboxEventsReq) (out *funcoperation.ReplayOutboxEventsResp, err error) {
	grpcep.InitResponse(&out)
	out.ProcessedCount, err = s.outboxLogic.Replay(ctx, req.GetLimit())
	return
}
