package service

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *AiAgentService) ListMessages(ctx context.Context, req *aiagent.GetMessageReq) (
	out *aiagent.GetMessageResp, err error) {
	grpcep.InitResponse(&out)
	out.Messages, out.HasMore, out.NextBefore, err = s.messageLogic.ListMessages(ctx, req)
	if err != nil {
		return
	}
	return
}

func (s *AiAgentService) StreamMessage(req *aiagent.StreamMessageReq, stream aiagent.AiAgentService_StreamMessageServer) error {
	return s.messageLogic.StreamMessage(stream.Context(), req, stream.Send)
}

func (s *AiAgentService) SubmitAskUserQuestion(ctx context.Context, req *aiagent.SubmitAskUserQuestionReq) (
	out *aiagent.SubmitAskUserQuestionResp, err error) {
	grpcep.InitResponse(&out)
	err = s.messageLogic.SubmitAskUserQuestion(ctx, req)
	return
}

func (s *AiAgentService) SubmitToolPermission(ctx context.Context, req *aiagent.SubmitToolPermissionReq) (
	out *aiagent.SubmitToolPermissionResp, err error) {
	grpcep.InitResponse(&out)
	err = s.messageLogic.SubmitToolPermission(ctx, req)
	return
}
