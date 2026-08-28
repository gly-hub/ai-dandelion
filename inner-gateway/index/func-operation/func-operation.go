package funcoperation

import (
	"github.com/gly-hub/ai-dandelion/inner-gateway/global"
	funcoperation "github.com/gly-hub/ai-dandelion/inner-gateway/internal/service/func-operation"
	"github.com/gofiber/fiber/v2"
)

func RouteHandler(fiberApp *fiber.App) {
	funcOperationController := funcoperation.NewFuncOperationServerController(global.GetApp().GrpcClientManager())
	baseRouter := fiberApp.Group("/func-operation")

	functionRouter := baseRouter.Group("/functions")
	{
		functionRouter.Get("/", funcOperationController.ListFunctions)
		functionRouter.Post("/", funcOperationController.CreateFunction)
		functionRouter.Put("/:id", funcOperationController.UpdateFunction)
		functionRouter.Post("/:id/documents/load", funcOperationController.LoadFunctionDocument)
		functionRouter.Post("/:id/documents/commit", funcOperationController.CommitFunctionDocument)
		functionRouter.Post("/:id/materialize", funcOperationController.MaterializeFunctionApp)
		functionRouter.Post("/:id/code/load", funcOperationController.LoadFunctionCodeState)
		functionRouter.Post("/:id/code/draft", funcOperationController.TouchFunctionCodeDraft)
		functionRouter.Post("/:id/code/apply", funcOperationController.ApplyFunctionCode)
		functionRouter.Post("/:id/sessions/ensure", funcOperationController.EnsureFunctionSession)
		functionRouter.Post("/:id/conversation-operations", funcOperationController.StartFunctionConversationOperation)
		functionRouter.Get("/:id/conversation-operations/latest", funcOperationController.GetLatestFunctionConversationOperation)
		functionRouter.Get("/:id/data-forms", funcOperationController.ListFunctionDataForms)
		functionRouter.Delete("/:id/data-forms/:name", funcOperationController.DeleteFunctionDataForm)
		functionRouter.Get("/:id/preview/frontend.js", funcOperationController.GetFunctionPreviewFrontend)
		functionRouter.Get("/:id/preview/frontend/*", funcOperationController.GetFunctionPreviewFrontend)
		functionRouter.Get("/:id/preview/bundle", funcOperationController.GetFunctionPreviewBundle)
		functionRouter.Post("/:id/preview/invoke", funcOperationController.InvokeFunctionPreview)
		functionRouter.Post("/:id/preview/invoke/stream", funcOperationController.StreamFunctionPreview)
		functionRouter.Get("/:id/execution-logs", funcOperationController.ListFunctionExecutionLogs)
		functionRouter.Get("/:id/execution-logs/:logId", funcOperationController.GetFunctionExecutionLog)
		functionRouter.Put("/:id/skill", funcOperationController.SetFunctionSkillEnabled)
		functionRouter.Get("/:id/skill-executions", funcOperationController.ListFunctionSkillExecutions)
		functionRouter.Delete("/:id", funcOperationController.DeleteFunction)
	}

	configRouter := baseRouter.Group("/configs")
	{
		configRouter.Get("/", funcOperationController.ListPublicConfigs)
		configRouter.Post("/", funcOperationController.CreatePublicConfig)
		configRouter.Put("/:key", funcOperationController.UpdatePublicConfig)
		configRouter.Get("/:key/versions", funcOperationController.ListPublicConfigVersions)
		configRouter.Post("/:key/rollback", funcOperationController.RollbackPublicConfig)
		configRouter.Post("/import-key/rotate", funcOperationController.RotatePublicConfigImportKey)
	}

	externalAPIRouter := baseRouter.Group("/external-api-clients")
	{
		externalAPIRouter.Get("/", funcOperationController.ListExternalAPIClients)
		externalAPIRouter.Post("/", funcOperationController.CreateExternalAPIClient)
		externalAPIRouter.Get("/recycle-bin", funcOperationController.ListDeletedExternalAPIClients)
		externalAPIRouter.Delete("/recycle-bin/:key", funcOperationController.PurgeExternalAPIClient)
		externalAPIRouter.Post("/:key/import-key/rotate", funcOperationController.RotateExternalAPIImportKey)
		externalAPIRouter.Put("/:key", funcOperationController.UpdateExternalAPIClient)
		externalAPIRouter.Delete("/:key", funcOperationController.DeleteExternalAPIClient)
		externalAPIRouter.Get("/:key/apis", funcOperationController.ListExternalAPIs)
		externalAPIRouter.Get("/:key/groups", funcOperationController.ListExternalAPIGroups)
		externalAPIRouter.Post("/:key/groups", funcOperationController.CreateExternalAPIGroup)
		externalAPIRouter.Post("/:key/apis", funcOperationController.CreateExternalAPI)
		externalAPIRouter.Put("/:key/apis/:apiKey", funcOperationController.UpdateExternalAPI)
		externalAPIRouter.Delete("/:key/apis/:apiKey", funcOperationController.DeleteExternalAPI)
		externalAPIRouter.Post("/:key/import", funcOperationController.ImportExternalAPIDocument)
		externalAPIRouter.Post("/:key/apis/:apiKey/test", funcOperationController.TestExternalAPI)
	}
	baseRouter.Post("/swagger-import/:key", funcOperationController.UploadExternalAPIDocument)
	baseRouter.Post("/import_configs", funcOperationController.ImportPublicConfigs)

	generatedAppRouter := baseRouter.Group("/generated-apps")
	{
		generatedAppRouter.Get("/", funcOperationController.ListGeneratedApps)
		generatedAppRouter.Post("/reload", funcOperationController.ReloadGeneratedApps)
		generatedAppRouter.Post("/cleanup-menus", funcOperationController.CleanupGeneratedFunctionMenus)
		generatedAppRouter.Get("/:id/frontend.bundle", funcOperationController.GetGeneratedAppFrontendBundle)
		generatedAppRouter.Get("/:id/bundle", funcOperationController.GetGeneratedAppFrontendBundle)
		generatedAppRouter.Get("/:id/frontend.js", funcOperationController.GetGeneratedAppFrontend)
		generatedAppRouter.Get("/:id/frontend/*", funcOperationController.GetGeneratedAppFrontend)
		generatedAppRouter.Post("/:id/invoke", funcOperationController.InvokeGeneratedApp)
	}
	outboxRouter := baseRouter.Group("/outbox")
	{
		outboxRouter.Get("/", funcOperationController.ListOutboxEvents)
		outboxRouter.Post("/replay", funcOperationController.ReplayOutboxEvents)
	}
}
