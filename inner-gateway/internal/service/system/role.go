package system

import (
	"context"

	"github.com/gofiber/fiber/v2"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/gerr"
	"github.com/team-dandelion/quickgo/grpcep"
)

// ListRoles
// @tags System
// @summary 获取角色列表
// @router /system/roles [GET]
func (s *SystemServerController) ListRoles(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ListRolesReq{}
	handler := func(ctx context.Context, req *systemproto.ListRolesReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListRoles(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateRole
// @tags System
// @summary 创建角色
// @router /system/roles [POST]
func (s *SystemServerController) CreateRole(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.CreateRoleReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *systemproto.CreateRoleReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CreateRole(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateRole
// @tags System
// @summary 更新角色
// @router /system/roles/{id} [PUT]
func (s *SystemServerController) UpdateRole(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.UpdateRoleReq{Id: ctx.Params("id")}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *systemproto.UpdateRoleReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.UpdateRole(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteRole
// @tags System
// @summary 删除角色
// @router /system/roles/{id} [DELETE]
func (s *SystemServerController) DeleteRole(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DeleteRoleReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.DeleteRoleReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DeleteRole(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// EnableRole
// @tags System
// @summary 启用角色
// @router /system/roles/{id}/enable [POST]
func (s *SystemServerController) EnableRole(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.EnableRoleReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.EnableRoleReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.EnableRole(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DisableRole
// @tags System
// @summary 禁用角色
// @router /system/roles/{id}/disable [POST]
func (s *SystemServerController) DisableRole(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DisableRoleReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.DisableRoleReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DisableRole(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// GetRoleMenus
// @tags System
// @summary 获取角色菜单
// @router /system/roles/{id}/menus [GET]
func (s *SystemServerController) GetRoleMenus(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.GetRoleMenusReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.GetRoleMenusReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetRoleMenus(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// SetRoleMenus
// @tags System
// @summary 设置角色菜单
// @router /system/roles/{id}/menus [PUT]
func (s *SystemServerController) SetRoleMenus(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.SetRoleMenusReq{Id: ctx.Params("id")}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *systemproto.SetRoleMenusReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.SetRoleMenus(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
