package service

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *SystemService) Login(ctx context.Context, req *systemproto.LoginReq) (
	out *systemproto.LoginResp, err error) {
	grpcep.InitResponse(&out)
	out.User, out.Roles, out.Token, out.ExpiresIn, err = s.userLogic.Login(ctx, req)
	return
}

func (s *SystemService) ValidateToken(ctx context.Context, req *systemproto.ValidateTokenReq) (
	out *systemproto.ValidateTokenResp, err error) {
	grpcep.InitResponse(&out)
	out.User, out.Roles, err = s.userLogic.ValidateToken(ctx, req)
	return
}

func (s *SystemService) GetUserRoles(ctx context.Context, req *systemproto.GetUserRolesReq) (
	out *systemproto.GetUserRolesResp, err error) {
	grpcep.InitResponse(&out)
	out.RoleIds, out.Roles, err = s.userLogic.GetUserRoles(ctx, req)
	return
}

func (s *SystemService) SetUserRoles(ctx context.Context, req *systemproto.SetUserRolesReq) (
	out *systemproto.SetUserRolesResp, err error) {
	grpcep.InitResponse(&out)
	out.RoleIds, err = s.userLogic.SetUserRoles(ctx, req)
	return
}
