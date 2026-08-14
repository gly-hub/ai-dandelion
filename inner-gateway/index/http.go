package index

import (
	"github.com/gofiber/fiber/v2"
	"github.com/team-dandelion/ai-dandelion/inner-gateway/global"
	aiagent "github.com/team-dandelion/ai-dandelion/inner-gateway/index/ai-agent"
	funcoperation "github.com/team-dandelion/ai-dandelion/inner-gateway/index/func-operation"
	realtime "github.com/team-dandelion/ai-dandelion/inner-gateway/index/realtime"
	systemsvc "github.com/team-dandelion/ai-dandelion/inner-gateway/index/system"
	"github.com/team-dandelion/ai-dandelion/inner-gateway/internal/middleware"
)

func RouteHandler(fiberApp *fiber.App) {
	fiberApp.Use(middleware.Auth(global.GetApp().GrpcClientManager()))
	aiagent.RouteHandler(fiberApp)
	funcoperation.RouteHandler(fiberApp)
	systemsvc.RouteHandler(fiberApp)
	realtime.RouteHandler(fiberApp)
}
