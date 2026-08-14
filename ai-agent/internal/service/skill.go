package service

import (
	"context"

	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *AiAgentService) ListSkills(ctx context.Context, req *aiagent.ListSkillsReq) (
	out *aiagent.ListSkillsResp, err error) {
	grpcep.InitResponse(&out)
	out.Skills, err = s.skillLogic.ListSkills(ctx, req)
	return
}

func (s *AiAgentService) ImportSkillPackage(ctx context.Context, req *aiagent.ImportSkillPackageReq) (
	out *aiagent.ImportSkillPackageResp, err error) {
	grpcep.InitResponse(&out)
	out.Skill, err = s.skillLogic.ImportSkillPackage(ctx, req)
	return
}

func (s *AiAgentService) UpdateSkill(ctx context.Context, req *aiagent.UpdateSkillReq) (
	out *aiagent.UpdateSkillResp, err error) {
	grpcep.InitResponse(&out)
	out.Skill, err = s.skillLogic.UpdateSkill(ctx, req)
	return
}

func (s *AiAgentService) DeleteSkill(ctx context.Context, req *aiagent.DeleteSkillReq) (
	out *aiagent.DeleteSkillResp, err error) {
	grpcep.InitResponse(&out)
	out.Id, err = s.skillLogic.DeleteSkill(ctx, req)
	return
}
