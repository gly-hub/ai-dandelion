package service

import (
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/logic"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
)

type AiAgentService struct {
	aiagent.UnimplementedAiAgentServiceServer
	sessionLogic    *logic.SessionLogic
	messageLogic    *logic.MessageLogic
	agentModelLogic *logic.AgentModelLogic
	skillLogic      *logic.SkillLogic
	mcpLogic        *logic.MCPLogic
	agentBotLogic   *logic.AgentBotLogic
}

func NewAiAgentService(
	sessionLogic *logic.SessionLogic,
	messageLogic *logic.MessageLogic,
	agentModelLogic *logic.AgentModelLogic,
	skillLogic *logic.SkillLogic,
	mcpLogic *logic.MCPLogic,
	agentBotLogic *logic.AgentBotLogic,
) *AiAgentService {
	return &AiAgentService{
		sessionLogic:    sessionLogic,
		messageLogic:    messageLogic,
		agentModelLogic: agentModelLogic,
		skillLogic:      skillLogic,
		mcpLogic:        mcpLogic,
		agentBotLogic:   agentBotLogic,
	}
}
