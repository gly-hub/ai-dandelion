package system

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
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

// RefreshToken
// @tags System
// @summary 刷新登录 token
// @router /system/auth/refresh [POST]
func (s *SystemServerController) RefreshToken(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.RefreshTokenReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *systemproto.RefreshTokenReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.RefreshToken(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// Logout
// @tags System
// @summary 退出登录
// @router /system/auth/logout [POST]
func (s *SystemServerController) Logout(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.LogoutReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *systemproto.LogoutReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.Logout(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
