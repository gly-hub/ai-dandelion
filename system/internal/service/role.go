package service

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *SystemService) ListRoles(ctx context.Context, req *systemproto.ListRolesReq) (
	out *systemproto.ListRolesResp, err error) {
	grpcep.InitResponse(&out)
	out.Roles, err = s.roleLogic.ListRoles(ctx, req)
	return
}

func (s *SystemService) CreateRole(ctx context.Context, req *systemproto.CreateRoleReq) (
	out *systemproto.CreateRoleResp, err error) {
	grpcep.InitResponse(&out)
	out.Role, err = s.roleLogic.CreateRole(ctx, req)
	return
}

func (s *SystemService) UpdateRole(ctx context.Context, req *systemproto.UpdateRoleReq) (
	out *systemproto.UpdateRoleResp, err error) {
	grpcep.InitResponse(&out)
	out.Role, err = s.roleLogic.UpdateRole(ctx, req)
	return
}

func (s *SystemService) DeleteRole(ctx context.Context, req *systemproto.DeleteRoleReq) (
	out *systemproto.DeleteRoleResp, err error) {
	grpcep.InitResponse(&out)
	err = s.roleLogic.DeleteRole(ctx, req)
	return
}

func (s *SystemService) EnableRole(ctx context.Context, req *systemproto.EnableRoleReq) (
	out *systemproto.EnableRoleResp, err error) {
	grpcep.InitResponse(&out)
	out.Role, err = s.roleLogic.EnableRole(ctx, req)
	return
}

func (s *SystemService) DisableRole(ctx context.Context, req *systemproto.DisableRoleReq) (
	out *systemproto.DisableRoleResp, err error) {
	grpcep.InitResponse(&out)
	out.Role, err = s.roleLogic.DisableRole(ctx, req)
	return
}

func (s *SystemService) GetRoleMenus(ctx context.Context, req *systemproto.GetRoleMenusReq) (
	out *systemproto.GetRoleMenusResp, err error) {
	grpcep.InitResponse(&out)
	out.MenuIds, err = s.roleLogic.GetRoleMenus(ctx, req)
	return
}

func (s *SystemService) SetRoleMenus(ctx context.Context, req *systemproto.SetRoleMenusReq) (
	out *systemproto.SetRoleMenusResp, err error) {
	grpcep.InitResponse(&out)
	out.MenuIds, err = s.roleLogic.SetRoleMenus(ctx, req)
	return
}
