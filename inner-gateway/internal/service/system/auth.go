package system

import (
	"context"

	"github.com/gofiber/fiber/v2"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/gerr"
	"github.com/team-dandelion/quickgo/grpcep"
)

// Login
// @tags System
// @summary 用户登录
// @router /system/auth/login [POST]
func (s *SystemServerController) Login(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.LoginReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *systemproto.LoginReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.Login(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
