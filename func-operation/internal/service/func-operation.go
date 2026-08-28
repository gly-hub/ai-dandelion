package service

import "github.com/gly-hub/ai-dandelion/func-operation/internal/logic"

type FuncOperationService struct {
	functionLogic              *logic.FunctionLogic
	appLogic                   *logic.GeneratedAppLogic
	outboxLogic                *logic.OutboxManagementLogic
	publicConfigLogic          *logic.PublicConfigLogic
	externalAPILogic           *logic.ExternalAPILogic
	functionSkillLogic         *logic.FunctionSkillLogic
	conversationOperationLogic *logic.ConversationOperationLogic
}

func NewFuncOperationService(functionLogic *logic.FunctionLogic, appLogic *logic.GeneratedAppLogic, outboxLogic *logic.OutboxManagementLogic, publicConfigLogic *logic.PublicConfigLogic, externalAPILogic *logic.ExternalAPILogic, functionSkillLogic *logic.FunctionSkillLogic, conversationOperationLogic *logic.ConversationOperationLogic) *FuncOperationService {
	return &FuncOperationService{
		functionLogic:              functionLogic,
		appLogic:                   appLogic,
		outboxLogic:                outboxLogic,
		publicConfigLogic:          publicConfigLogic,
		externalAPILogic:           externalAPILogic,
		functionSkillLogic:         functionSkillLogic,
		conversationOperationLogic: conversationOperationLogic,
	}
}
