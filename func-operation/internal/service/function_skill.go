package service

import (
	"context"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/quickgo/grpcep"
)

func (s *FuncOperationService) ListFunctionSkills(ctx context.Context, req *funcoperation.ListFunctionSkillsReq) (out *funcoperation.ListFunctionSkillsResp, err error) {
	grpcep.InitResponse(&out)
	out.Skills, err = s.functionSkillLogic.List(ctx, req)
	return
}

func (s *FuncOperationService) SetFunctionSkillEnabled(ctx context.Context, req *funcoperation.SetFunctionSkillEnabledReq) (out *funcoperation.SetFunctionSkillEnabledResp, err error) {
	grpcep.InitResponse(&out)
	out.Skill, err = s.functionSkillLogic.SetEnabled(ctx, req)
	return
}

func (s *FuncOperationService) IssueFunctionSkillGrant(ctx context.Context, req *funcoperation.IssueFunctionSkillGrantReq) (out *funcoperation.IssueFunctionSkillGrantResp, err error) {
	grpcep.InitResponse(&out)
	out.GrantToken, out.ExpiresAt, err = s.functionSkillLogic.IssueGrant(ctx, req)
	return
}

func (s *FuncOperationService) GetFunctionSkillTools(ctx context.Context, req *funcoperation.GetFunctionSkillToolsReq) (out *funcoperation.GetFunctionSkillToolsResp, err error) {
	grpcep.InitResponse(&out)
	out.Tools, err = s.functionSkillLogic.GetTools(ctx, req)
	return
}

func (s *FuncOperationService) CreateFunctionSkillApproval(ctx context.Context, req *funcoperation.CreateFunctionSkillApprovalReq) (out *funcoperation.CreateFunctionSkillApprovalResp, err error) {
	grpcep.InitResponse(&out)
	out.ApprovalToken, err = s.functionSkillLogic.CreateApproval(ctx, req)
	return
}

func (s *FuncOperationService) ExecuteFunctionSkill(ctx context.Context, req *funcoperation.ExecuteFunctionSkillReq) (out *funcoperation.ExecuteFunctionSkillResp, err error) {
	grpcep.InitResponse(&out)
	result, err := s.functionSkillLogic.Execute(ctx, req)
	if err != nil {
		return out, err
	}
	return result, nil
}

func (s *FuncOperationService) ListFunctionSkillExecutions(ctx context.Context, req *funcoperation.ListFunctionSkillExecutionsReq) (out *funcoperation.ListFunctionSkillExecutionsResp, err error) {
	grpcep.InitResponse(&out)
	out.Executions, err = s.functionSkillLogic.ListExecutions(ctx, req)
	return
}
