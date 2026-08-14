package system

import (
	"context"
	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

func (s *SystemServerController) ListNotifications(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ListNotificationsReq{Page: int32(ctx.QueryInt("page", 1)), PageSize: int32(ctx.QueryInt("pageSize", 20)), UnreadOnly: ctx.Query("unreadOnly") == "1" || ctx.Query("unreadOnly") == "true"}
	return s.baseHandler.GRPCCall(ctx, rpcParam, func(ctx context.Context, req *systemproto.ListNotificationsReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListNotifications(ctx, req)
	})
}
func (s *SystemServerController) SendNotification(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.SendNotificationReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, func(ctx context.Context, req *systemproto.SendNotificationReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.SendNotification(ctx, req)
	})
}
func (s *SystemServerController) ReadNotification(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ReadNotificationReq{Id: ctx.Params("id")}
	return s.baseHandler.GRPCCall(ctx, rpcParam, func(ctx context.Context, req *systemproto.ReadNotificationReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ReadNotification(ctx, req)
	})
}
