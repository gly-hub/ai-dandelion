package system

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/gerr"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
)

// ListUsers
// @tags System
// @summary 获取用户列表
// @router /system/users [GET]
func (s *SystemServerController) ListUsers(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ListUsersReq{}
	handler := func(ctx context.Context, req *systemproto.ListUsersReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListUsers(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateUser
// @tags System
// @summary 创建用户
// @router /system/users [POST]
func (s *SystemServerController) CreateUser(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.CreateUserReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}

	handler := func(ctx context.Context, req *systemproto.CreateUserReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CreateUser(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateUser
// @tags System
// @summary 更新用户
// @router /system/users/{id} [PUT]
func (s *SystemServerController) UpdateUser(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.UpdateUserReq{
		Id: ctx.Params("id"),
	}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")

	handler := func(ctx context.Context, req *systemproto.UpdateUserReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.UpdateUser(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteUser
// @tags System
// @summary 删除用户
// @router /system/users/{id} [DELETE]
func (s *SystemServerController) DeleteUser(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DeleteUserReq{
		Id: ctx.Params("id"),
	}

	handler := func(ctx context.Context, req *systemproto.DeleteUserReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DeleteUser(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// EnableUser
// @tags System
// @summary 启用用户
// @router /system/users/{id}/enable [POST]
func (s *SystemServerController) EnableUser(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.EnableUserReq{
		Id: ctx.Params("id"),
	}

	handler := func(ctx context.Context, req *systemproto.EnableUserReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.EnableUser(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DisableUser
// @tags System
// @summary 禁用用户
// @router /system/users/{id}/disable [POST]
func (s *SystemServerController) DisableUser(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DisableUserReq{
		Id: ctx.Params("id"),
	}

	handler := func(ctx context.Context, req *systemproto.DisableUserReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DisableUser(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// GetUserRoles
// @tags System
// @summary 获取用户角色
// @router /system/users/{id}/roles [GET]
func (s *SystemServerController) GetUserRoles(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.GetUserRolesReq{UserId: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.GetUserRolesReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetUserRoles(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// SetUserRoles
// @tags System
// @summary 设置用户角色
// @router /system/users/{id}/roles [PUT]
func (s *SystemServerController) SetUserRoles(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.SetUserRolesReq{UserId: ctx.Params("id")}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.UserId = ctx.Params("id")
	handler := func(ctx context.Context, req *systemproto.SetUserRolesReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.SetUserRoles(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
