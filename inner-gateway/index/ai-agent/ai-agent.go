package aiagent

import (
	"github.com/gly-hub/ai-dandelion/inner-gateway/global"
	aiagent "github.com/gly-hub/ai-dandelion/inner-gateway/internal/service/ai-agent"
	"github.com/gofiber/fiber/v2"
)

func RouteHandler(fiberApp *fiber.App) {
	baseRouter := fiberApp.Group("/ai-agent")
	aiAgentServerController := aiagent.NewAIAgentServerController(global.GetApp().GrpcClientManager())

	sessionRouter := baseRouter.Group("/session")
	{
		sessionRouter.Get("/", aiAgentServerController.ListSessions)
		sessionRouter.Post("/", aiAgentServerController.CreateSession)
		sessionRouter.Post("/ensure", aiAgentServerController.EnsureSession)
		sessionRouter.Put("/:id", aiAgentServerController.UpdateSession)
		sessionRouter.Delete("/:id", aiAgentServerController.DeleteSession)
		sessionRouter.Get("/:id/messages", aiAgentServerController.ListMessages)
		sessionRouter.Post("/:id/messages/stream", aiAgentServerController.StreamMessage)
		sessionRouter.Post("/:id/tool-requests/:toolId/answer", aiAgentServerController.SubmitAskUserQuestion)
		sessionRouter.Post("/:id/tool-requests/:toolId/permission", aiAgentServerController.SubmitToolPermission)
	}
	baseRouter.Get("/models", aiAgentServerController.ListAgentModels)
	baseRouter.Get("/skills", aiAgentServerController.ListSkills)
	baseRouter.Get("/function-skills", aiAgentServerController.ListFunctionSkills)
	baseRouter.Post("/skills/import", aiAgentServerController.ImportSkillPackage)
	baseRouter.Put("/skills/:id", aiAgentServerController.UpdateSkill)
	baseRouter.Delete("/skills/:id", aiAgentServerController.DeleteSkill)
	baseRouter.Get("/mcp-servers", aiAgentServerController.ListMCPServers)
	baseRouter.Post("/mcp-servers", aiAgentServerController.CreateMCPServer)
	baseRouter.Put("/mcp-servers/:id", aiAgentServerController.UpdateMCPServer)
	baseRouter.Delete("/mcp-servers/:id", aiAgentServerController.DeleteMCPServer)

	agentBotRouter := baseRouter.Group("/agent-bots")
	{
		agentBotRouter.Get("/runtime", aiAgentServerController.ListAgentBotRuntimeConfigs)
		agentBotRouter.Get("/", aiAgentServerController.ListAgentBots)
		agentBotRouter.Post("/", aiAgentServerController.CreateAgentBot)
		agentBotRouter.Put("/:id", aiAgentServerController.UpdateAgentBot)
		agentBotRouter.Delete("/:id", aiAgentServerController.DeleteAgentBot)
		agentBotRouter.Post("/:id/enable", aiAgentServerController.EnableAgentBot)
		agentBotRouter.Post("/:id/disable", aiAgentServerController.DisableAgentBot)
	}
}
