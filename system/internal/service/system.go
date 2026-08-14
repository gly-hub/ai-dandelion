package service

import (
	"github.com/team-dandelion/ai-dandelion/system/internal/logic"
)

type SystemService struct {
	userLogic               *logic.UserLogic
	menuLogic               *logic.MenuLogic
	roleLogic               *logic.RoleLogic
	agentModelLogic         *logic.AgentModelLogic
	agentConfigLogic        *logic.AgentConfigLogic
	agentSessionConfigLogic *logic.AgentSessionConfigLogic
	operationLogLogic       *logic.OperationLogLogic
	notificationLogic       *logic.NotificationLogic
	uploadLogic             *logic.UploadLogic
}

func NewSystemService(
	userLogic *logic.UserLogic,
	menuLogic *logic.MenuLogic,
	roleLogic *logic.RoleLogic,
	agentModelLogic *logic.AgentModelLogic,
	agentConfigLogic *logic.AgentConfigLogic,
	agentSessionConfigLogic *logic.AgentSessionConfigLogic,
	operationLogLogic *logic.OperationLogLogic,
	notificationLogic *logic.NotificationLogic,
	uploadLogic *logic.UploadLogic,
) *SystemService {
	return &SystemService{
		userLogic:               userLogic,
		menuLogic:               menuLogic,
		roleLogic:               roleLogic,
		agentModelLogic:         agentModelLogic,
		agentConfigLogic:        agentConfigLogic,
		agentSessionConfigLogic: agentSessionConfigLogic,
		operationLogLogic:       operationLogLogic,
		notificationLogic:       notificationLogic,
		uploadLogic:             uploadLogic,
	}
}
