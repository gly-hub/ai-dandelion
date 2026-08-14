package system

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"github.com/team-dandelion/quickgo/gerr"
	"github.com/team-dandelion/quickgo/grpcep"
)

// GetNavMenus
// @tags System
// @summary 获取导航菜单树
// @router /system/menus/nav [GET]
func (s *SystemServerController) GetNavMenus(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.GetNavMenusReq{
		Module: strings.TrimSpace(ctx.Query("module")),
	}
	if userID, ok := ctx.Locals(authctx.MetadataUserID).(string); ok {
		rpcParam.UserId = strings.TrimSpace(userID)
	}
	handler := func(ctx context.Context, req *systemproto.GetNavMenusReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.GetNavMenus(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// ListMenus
// @tags System
// @summary 获取菜单列表
// @router /system/menus [GET]
func (s *SystemServerController) ListMenus(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ListMenusReq{
		Module:    ctx.Query("module"),
		Placement: ctx.Query("placement"),
		Tree:      ctx.Query("tree") == "1" || ctx.Query("tree") == "true",
		Status:    int32(ctx.QueryInt("status", 0)),
	}
	handler := func(ctx context.Context, req *systemproto.ListMenusReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListMenus(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// CreateMenu
// @tags System
// @summary 创建菜单
// @router /system/menus [POST]
func (s *SystemServerController) CreateMenu(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.CreateMenuReq{}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	handler := func(ctx context.Context, req *systemproto.CreateMenuReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.CreateMenu(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// UpdateMenu
// @tags System
// @summary 更新菜单
// @router /system/menus/{id} [PUT]
func (s *SystemServerController) UpdateMenu(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.UpdateMenuReq{Id: ctx.Params("id")}
	if err := s.baseHandler.ParseJson(ctx, rpcParam); err != nil {
		return s.baseHandler.Response(ctx, grpcep.JsonResponse{}, gerr.NewGErr(grpcep.ParamsErrCode, err.Error()))
	}
	rpcParam.Id = ctx.Params("id")
	handler := func(ctx context.Context, req *systemproto.UpdateMenuReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.UpdateMenu(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DeleteMenu
// @tags System
// @summary 删除菜单
// @router /system/menus/{id} [DELETE]
func (s *SystemServerController) DeleteMenu(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DeleteMenuReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.DeleteMenuReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DeleteMenu(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// EnableMenu
// @tags System
// @summary 启用菜单
// @router /system/menus/{id}/enable [POST]
func (s *SystemServerController) EnableMenu(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.EnableMenuReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.EnableMenuReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.EnableMenu(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

// DisableMenu
// @tags System
// @summary 禁用菜单
// @router /system/menus/{id}/disable [POST]
func (s *SystemServerController) DisableMenu(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.DisableMenuReq{Id: ctx.Params("id")}
	handler := func(ctx context.Context, req *systemproto.DisableMenuReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.DisableMenu(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}
