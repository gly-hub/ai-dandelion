package funcoperation

import (
	"context"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/grpcep"
	"google.golang.org/grpc"
)

type ClientManager interface {
	GetConn(ctx context.Context, serviceName string) (*grpc.ClientConn, error)
}

type FuncOperationServerController struct {
	baseHandler *grpcep.BaseHandler
	clientMgr   ClientManager
}

func NewFuncOperationServerController(clientMgr ClientManager) *FuncOperationServerController {
	return &FuncOperationServerController{
		baseHandler: &grpcep.BaseHandler{},
		clientMgr:   clientMgr,
	}
}

func (f *FuncOperationServerController) getFuncOperationClient(ctx context.Context) (funcoperation.FuncOperationServiceClient, error) {
	conn, err := f.clientMgr.GetConn(ctx, "func-operation")
	if err != nil {
		return nil, err
	}
	return funcoperation.NewFuncOperationServiceClient(conn), nil
}
