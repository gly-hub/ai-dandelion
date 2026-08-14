package aiagent

import (
	"context"

	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/quickgo/grpcep"
	"google.golang.org/grpc"
)

type ClientManager interface {
	GetConn(ctx context.Context, serviceName string) (*grpc.ClientConn, error)
}

type AIAgentServerController struct {
	baseHandler *grpcep.BaseHandler
	clientMgr   ClientManager
}

func NewAIAgentServerController(clientMgr ClientManager) *AIAgentServerController {
	return &AIAgentServerController{
		baseHandler: &grpcep.BaseHandler{},
		clientMgr:   clientMgr,
	}
}

func (a *AIAgentServerController) getAiAgentClient(ctx context.Context) (aiagent.AiAgentServiceClient, error) {
	conn, err := a.clientMgr.GetConn(ctx, "ai-agent")
	if err != nil {
		return nil, err
	}
	return aiagent.NewAiAgentServiceClient(conn), nil
}
