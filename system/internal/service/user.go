package service

import (
	"context"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *SystemService) ListUsers(ctx context.Context, req *systemproto.ListUsersReq) (
	out *systemproto.ListUsersResp, err error) {
	grpcep.InitResponse(&out)
	out.Users, err = s.userLogic.ListUsers(ctx, req)
	return
}

func (s *SystemService) CreateUser(ctx context.Context, req *systemproto.CreateUserReq) (
	out *systemproto.CreateUserResp, err error) {
	grpcep.InitResponse(&out)
	out.User, err = s.userLogic.CreateUser(ctx, req)
	return
}

func (s *SystemService) UpdateUser(ctx context.Context, req *systemproto.UpdateUserReq) (
	out *systemproto.UpdateUserResp, err error) {
	grpcep.InitResponse(&out)
	out.User, err = s.userLogic.UpdateUser(ctx, req)
	return
}

func (s *SystemService) DeleteUser(ctx context.Context, req *systemproto.DeleteUserReq) (
	out *systemproto.DeleteUserResp, err error) {
	grpcep.InitResponse(&out)
	err = s.userLogic.DeleteUser(ctx, req)
	return
}

func (s *SystemService) EnableUser(ctx context.Context, req *systemproto.EnableUserReq) (
	out *systemproto.EnableUserResp, err error) {
	grpcep.InitResponse(&out)
	out.User, err = s.userLogic.EnableUser(ctx, req)
	return
}

func (s *SystemService) DisableUser(ctx context.Context, req *systemproto.DisableUserReq) (
	out *systemproto.DisableUserResp, err error) {
	grpcep.InitResponse(&out)
	out.User, err = s.userLogic.DisableUser(ctx, req)
	return
}
