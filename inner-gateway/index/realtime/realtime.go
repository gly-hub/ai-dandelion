package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/team-dandelion/ai-dandelion/inner-gateway/global"
	realtimehandler "github.com/team-dandelion/ai-dandelion/inner-gateway/internal/realtime"
	funcoperation "github.com/team-dandelion/ai-dandelion/proto/func-operation"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/toolbox/eventbus"
	"time"
)

func RouteHandler(app *fiber.App) {
	tickets := realtimehandler.NewTicketManager(time.Minute)
	if redisClient, err := global.GetApp().RedisManager().GetRedisClient("ai-dandelion"); err == nil && redisClient != nil {
		tickets = realtimehandler.NewRedisTicketManager(time.Minute, redisClient)
	}
	handler := realtimehandler.NewHandler(global.GetApp().GrpcClientManager(), tickets)
	handler.SetAllowedOrigins(global.GetConfig().RealtimeConfig.AllowedOrigins)
	handler.RegisterNamespace("system", func(ctx context.Context, in realtimehandler.Envelope, write realtimehandler.Writer) {
		conn, err := global.GetApp().GrpcClientManager().GetConn(ctx, "system")
		if err == nil {
			switch in.Type {
			case "system.nav.list":
				var req systemproto.GetNavMenusReq
				if len(in.Payload) > 0 {
					err = json.Unmarshal(in.Payload, &req)
				}
				if err == nil {
					var resp *systemproto.GetNavMenusResp
					resp, err = systemproto.NewSystemServiceClient(conn).GetNavMenus(ctx, &req)
					if err == nil {
						payload, _ := json.Marshal(resp)
						_ = write(realtimehandler.Envelope{ProtocolVersion: 1, Type: "system.nav.list.result", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli(), Payload: payload})
						return
					}
				}
			case "system.ping":
				_ = write(realtimehandler.Envelope{ProtocolVersion: 1, Type: "system.pong", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli()})
				return
			default:
				err = fmt.Errorf("unsupported system realtime command: %s", in.Type)
			}
		}
		_ = write(realtimehandler.RealtimeError(in.RequestID, err))
	})
	handler.RegisterNamespace("func-operation", func(ctx context.Context, in realtimehandler.Envelope, write realtimehandler.Writer) {
		conn, err := global.GetApp().GrpcClientManager().GetConn(ctx, "func-operation")
		if err == nil {
			switch in.Type {
			case "func-operation.functions.list":
				resp, callErr := funcoperation.NewFuncOperationServiceClient(conn).ListFunctions(ctx, &funcoperation.ListFunctionsReq{})
				if callErr == nil {
					payload, _ := json.Marshal(resp)
					_ = write(realtimehandler.Envelope{ProtocolVersion: 1, Type: "func-operation.functions.list.result", RequestID: in.RequestID, Timestamp: time.Now().UnixMilli(), Payload: payload})
					return
				}
				err = callErr
			default:
				err = fmt.Errorf("unsupported func-operation realtime command: %s", in.Type)
			}
		}
		_ = write(realtimehandler.RealtimeError(in.RequestID, err))
	})
	if redisClient, err := global.GetApp().RedisManager().GetRedisClient("ai-dandelion"); err == nil && redisClient != nil {
		if bus, busErr := eventbus.NewRedisStreams(redisClient); busErr == nil {
			handler.SubscribeEvents(context.Background(), bus, eventbus.Subscription{Topic: "realtime.events", Group: fmt.Sprintf("inner-gateway-%s", uuid.NewString()), Consumer: "gateway-local"})
		}
	}
	app.Post("/realtime/ticket", handler.IssueTicket)
	app.Get("/realtime/ws", handler.UpgradeCheck, fiberws.New(handler.Serve))
}
