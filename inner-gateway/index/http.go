package index

import (
	"github.com/gly-hub/ai-dandelion/inner-gateway/global"
	aiagent "github.com/gly-hub/ai-dandelion/inner-gateway/index/ai-agent"
	funcoperation "github.com/gly-hub/ai-dandelion/inner-gateway/index/func-operation"
	realtime "github.com/gly-hub/ai-dandelion/inner-gateway/index/realtime"
	systemsvc "github.com/gly-hub/ai-dandelion/inner-gateway/index/system"
	"github.com/gly-hub/ai-dandelion/inner-gateway/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

func RouteHandler(fiberApp *fiber.App) {
	fiberApp.Use(middleware.Auth(global.GetApp().GrpcClientManager()))
	aiagent.RouteHandler(fiberApp)
	funcoperation.RouteHandler(fiberApp)
	systemsvc.RouteHandler(fiberApp)
	realtime.RouteHandler(fiberApp)
}
