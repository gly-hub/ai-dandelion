package service

import "github.com/gly-hub/ai-dandelion/func-operation/internal/logic"

type FuncOperationService struct {
	functionLogic     *logic.FunctionLogic
	appLogic          *logic.GeneratedAppLogic
	outboxLogic       *logic.OutboxManagementLogic
	publicConfigLogic *logic.PublicConfigLogic
	externalAPILogic  *logic.ExternalAPILogic
}

func NewFuncOperationService(functionLogic *logic.FunctionLogic, appLogic *logic.GeneratedAppLogic, outboxLogic *logic.OutboxManagementLogic, publicConfigLogic *logic.PublicConfigLogic, externalAPILogic *logic.ExternalAPILogic) *FuncOperationService {
	return &FuncOperationService{
		functionLogic:     functionLogic,
		appLogic:          appLogic,
		outboxLogic:       outboxLogic,
		publicConfigLogic: publicConfigLogic,
		externalAPILogic:  externalAPILogic,
	}
}
