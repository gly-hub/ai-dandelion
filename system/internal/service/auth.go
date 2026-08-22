package service

import (
	"context"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *SystemService) Login(ctx context.Context, req *systemproto.LoginReq) (
	out *systemproto.LoginResp, err error) {
	grpcep.InitResponse(&out)
	result, loginErr := s.userLogic.Login(ctx, req)
	if loginErr != nil {
		return out, loginErr
	}
	out.User = result.User
	out.Roles = result.Roles
	out.AccessToken = result.AccessToken
	out.RefreshToken = result.RefreshToken
	out.AccessExpiresIn = result.AccessExpiresIn
	out.RefreshExpiresIn = result.RefreshExpiresIn
	return
}

func (s *SystemService) RefreshToken(ctx context.Context, req *systemproto.RefreshTokenReq) (
	out *systemproto.RefreshTokenResp, err error) {
	grpcep.InitResponse(&out)
	result, refreshErr := s.userLogic.RefreshToken(ctx, req)
	if refreshErr != nil {
		return out, refreshErr
	}
	out.User = result.User
	out.Roles = result.Roles
	out.AccessToken = result.AccessToken
	out.RefreshToken = result.RefreshToken
	out.AccessExpiresIn = result.AccessExpiresIn
	out.RefreshExpiresIn = result.RefreshExpiresIn
	return
}

func (s *SystemService) Logout(ctx context.Context, req *systemproto.LogoutReq) (
	out *systemproto.LogoutResp, err error) {
	grpcep.InitResponse(&out)
	err = s.userLogic.Logout(ctx, req)
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
