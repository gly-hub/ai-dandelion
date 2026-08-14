package service

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *SystemService) ListMenus(ctx context.Context, req *systemproto.ListMenusReq) (
	out *systemproto.ListMenusResp, err error) {
	grpcep.InitResponse(&out)
	out.Menus, err = s.menuLogic.ListMenus(ctx, req)
	return
}

func (s *SystemService) GetNavMenus(ctx context.Context, req *systemproto.GetNavMenusReq) (
	out *systemproto.GetNavMenusResp, err error) {
	grpcep.InitResponse(&out)
	out.Menus, err = s.menuLogic.GetNavMenus(ctx, req)
	return
}

func (s *SystemService) CreateMenu(ctx context.Context, req *systemproto.CreateMenuReq) (
	out *systemproto.CreateMenuResp, err error) {
	grpcep.InitResponse(&out)
	out.Menu, err = s.menuLogic.CreateMenu(ctx, req)
	return
}

func (s *SystemService) UpdateMenu(ctx context.Context, req *systemproto.UpdateMenuReq) (
	out *systemproto.UpdateMenuResp, err error) {
	grpcep.InitResponse(&out)
	out.Menu, err = s.menuLogic.UpdateMenu(ctx, req)
	return
}

func (s *SystemService) DeleteMenu(ctx context.Context, req *systemproto.DeleteMenuReq) (
	out *systemproto.DeleteMenuResp, err error) {
	grpcep.InitResponse(&out)
	err = s.menuLogic.DeleteMenu(ctx, req)
	return
}

func (s *SystemService) EnableMenu(ctx context.Context, req *systemproto.EnableMenuReq) (
	out *systemproto.EnableMenuResp, err error) {
	grpcep.InitResponse(&out)
	out.Menu, err = s.menuLogic.EnableMenu(ctx, req)
	return
}

func (s *SystemService) DisableMenu(ctx context.Context, req *systemproto.DisableMenuReq) (
	out *systemproto.DisableMenuResp, err error) {
	grpcep.InitResponse(&out)
	out.Menu, err = s.menuLogic.DisableMenu(ctx, req)
	return
}

func (s *SystemService) SyncGeneratedFunctionMenu(ctx context.Context, req *systemproto.SyncGeneratedFunctionMenuReq) (
	out *systemproto.SyncGeneratedFunctionMenuResp, err error) {
	grpcep.InitResponse(&out)
	out.Menu, err = s.menuLogic.SyncGeneratedFunctionMenu(ctx, req)
	return
}

func (s *SystemService) CheckFunctionMenuAccess(ctx context.Context, req *systemproto.CheckFunctionMenuAccessReq) (
	out *systemproto.CheckFunctionMenuAccessResp, err error) {
	grpcep.InitResponse(&out)
	out.Allowed, err = s.menuLogic.CheckFunctionMenuAccess(ctx, req)
	return
}

func (s *SystemService) CheckMenuAccess(ctx context.Context, req *systemproto.CheckMenuAccessReq) (
	out *systemproto.CheckMenuAccessResp, err error) {
	grpcep.InitResponse(&out)
	out.Allowed, err = s.menuLogic.CheckMenuAccess(ctx, req)
	return
}
