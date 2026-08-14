package system

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
	"google.golang.org/grpc"
)

type ClientManager interface {
	GetConn(ctx context.Context, serviceName string) (*grpc.ClientConn, error)
}

type SystemServerController struct {
	baseHandler *grpcep.BaseHandler
	clientMgr   ClientManager
}

func NewSystemServerController(clientMgr ClientManager) *SystemServerController {
	return &SystemServerController{
		baseHandler: &grpcep.BaseHandler{},
		clientMgr:   clientMgr,
	}
}

func (s *SystemServerController) getSystemClient(ctx context.Context) (systemproto.SystemServiceClient, error) {
	conn, err := s.clientMgr.GetConn(ctx, "system")
	if err != nil {
		return nil, err
	}
	return systemproto.NewSystemServiceClient(conn), nil
}
