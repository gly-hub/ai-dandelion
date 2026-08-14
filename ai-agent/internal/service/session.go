package service

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *AiAgentService) ListSessions(ctx context.Context, req *aiagent.SearchMessageReq) (
	out *aiagent.SearchMessageResp, err error) {
	grpcep.InitResponse(&out)
	out.Sessions, err = s.sessionLogic.ListSessions(ctx, req)
	if err != nil {
		return
	}
	return
}

func (s *AiAgentService) CreateSession(ctx context.Context, req *aiagent.CreateSessionReq) (
	out *aiagent.CreateSessionResp, err error) {
	grpcep.InitResponse(&out)
	out.Session, err = s.sessionLogic.CreateSession(ctx, req)
	if err != nil {
		return
	}
	return
}

func (s *AiAgentService) EnsureSession(ctx context.Context, req *aiagent.EnsureSessionReq) (
	out *aiagent.EnsureSessionResp, err error) {
	grpcep.InitResponse(&out)
	out.Session, out.Created, err = s.sessionLogic.EnsureSession(ctx, req)
	if err != nil {
		return
	}
	return
}

func (s *AiAgentService) UpdateSession(ctx context.Context, req *aiagent.UpdateSessionReq) (
	out *aiagent.UpdateSessionResp, err error) {
	grpcep.InitResponse(&out)
	out.Session, err = s.sessionLogic.UpdateSession(ctx, req)
	if err != nil {
		return
	}
	return
}

func (s *AiAgentService) DeleteSession(ctx context.Context, req *aiagent.DeleteSessionReq) (
	out *aiagent.DeleteSessionResp, err error) {
	grpcep.InitResponse(&out)
	err = s.sessionLogic.DeleteSession(ctx, req)
	if err != nil {
		return
	}
	return
}
