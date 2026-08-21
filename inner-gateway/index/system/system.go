package system

import (
	"github.com/gly-hub/ai-dandelion/inner-gateway/global"
	systemsvc "github.com/gly-hub/ai-dandelion/inner-gateway/internal/service/system"
	"github.com/gofiber/fiber/v2"
)

func RouteHandler(fiberApp *fiber.App) {
	baseRouter := fiberApp.Group("/system")
	controller := systemsvc.NewSystemServerController(global.GetApp().GrpcClientManager())
	baseRouter.Post("/uploads", controller.CreateUpload)
	baseRouter.Get("/uploads/:uuid/parts/:part_number", controller.GetUploadPartURL)
	baseRouter.Post("/uploads/:uuid/complete", controller.CompleteUpload)
	baseRouter.Get("/uploads/:uuid/preview", controller.GetUploadPreviewURL)
	baseRouter.Get("/uploads/:uuid/download", controller.GetUploadDownloadURL)

	authRouter := baseRouter.Group("/auth")
	{
		authRouter.Post("/login", controller.Login)
		authRouter.Post("/refresh", controller.RefreshToken)
		authRouter.Post("/logout", controller.Logout)
	}

	userRouter := baseRouter.Group("/users")
	{
		userRouter.Get("/", controller.ListUsers)
		userRouter.Post("/", controller.CreateUser)
		userRouter.Put("/:id", controller.UpdateUser)
		userRouter.Delete("/:id", controller.DeleteUser)
		userRouter.Post("/:id/enable", controller.EnableUser)
		userRouter.Post("/:id/disable", controller.DisableUser)
		userRouter.Get("/:id/roles", controller.GetUserRoles)
		userRouter.Put("/:id/roles", controller.SetUserRoles)
	}

	roleRouter := baseRouter.Group("/roles")
	{
		roleRouter.Get("/", controller.ListRoles)
		roleRouter.Post("/", controller.CreateRole)
		roleRouter.Put("/:id", controller.UpdateRole)
		roleRouter.Delete("/:id", controller.DeleteRole)
		roleRouter.Post("/:id/enable", controller.EnableRole)
		roleRouter.Post("/:id/disable", controller.DisableRole)
		roleRouter.Get("/:id/menus", controller.GetRoleMenus)
		roleRouter.Put("/:id/menus", controller.SetRoleMenus)
	}

	menuRouter := baseRouter.Group("/menus")
	{
		menuRouter.Get("/nav", controller.GetNavMenus)
		menuRouter.Get("/", controller.ListMenus)
		menuRouter.Post("/", controller.CreateMenu)
		menuRouter.Put("/:id", controller.UpdateMenu)
		menuRouter.Delete("/:id", controller.DeleteMenu)
		menuRouter.Post("/:id/enable", controller.EnableMenu)
		menuRouter.Post("/:id/disable", controller.DisableMenu)
	}

	agentModelRouter := baseRouter.Group("/agent-models")
	{
		agentModelRouter.Get("/", controller.ListAgentModels)
		agentModelRouter.Post("/", controller.CreateAgentModel)
		agentModelRouter.Put("/:id", controller.UpdateAgentModel)
		agentModelRouter.Delete("/:id", controller.DeleteAgentModel)
		agentModelRouter.Post("/:id/enable", controller.EnableAgentModel)
		agentModelRouter.Post("/:id/disable", controller.DisableAgentModel)
	}

	agentConfigRouter := baseRouter.Group("/agent-config")
	{
		agentConfigRouter.Get("/", controller.GetAgentConfig)
		agentConfigRouter.Put("/", controller.UpdateAgentConfig)
	}

	agentSessionConfigRouter := baseRouter.Group("/agent-session-configs")
	{
		agentSessionConfigRouter.Get("/", controller.ListAgentSessionConfigs)
		agentSessionConfigRouter.Put("/:sessionType", controller.UpdateAgentSessionConfig)
	}

	operationLogRouter := baseRouter.Group("/operation-logs")
	{
		operationLogRouter.Get("/", controller.ListOperationLogs)
	}
	notificationRouter := baseRouter.Group("/notifications")
	{
		notificationRouter.Get("/", controller.ListNotifications)
		notificationRouter.Post("/", controller.SendNotification)
		notificationRouter.Post("/:id/read", controller.ReadNotification)
	}

}
