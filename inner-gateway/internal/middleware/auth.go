package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/team-dandelion/ai-dandelion/inner-gateway/global"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"google.golang.org/grpc"
)

type ClientManager interface {
	GetConn(ctx context.Context, serviceName string) (*grpc.ClientConn, error)
}

func Auth(clientMgr ClientManager) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if isPublicRequest(ctx) {
			return ctx.Next()
		}
		token := bearerToken(ctx.Get(fiber.HeaderAuthorization))
		if token == "" {
			token = strings.TrimSpace(ctx.Get("X-Access-Token"))
		}
		if token == "" {
			return unauthorized(ctx, "missing token")
		}
		if isBridgeToken(ctx, token) {
			setUserValue(ctx, authctx.MetadataUserID, "bridge")
			setUserValue(ctx, authctx.MetadataUsername, "channel-bridge")
			clearExternalCredentialHeaders(ctx)
			return ctx.Next()
		}

		conn, err := clientMgr.GetConn(ctx.UserContext(), "system")
		if err != nil {
			return unauthorized(ctx, "auth service unavailable")
		}
		client := systemproto.NewSystemServiceClient(conn)

		resp, err := client.ValidateToken(ctx.UserContext(), &systemproto.ValidateTokenReq{Token: token})
		if err != nil {
			return unauthorized(ctx, "invalid token")
		}
		user := resp.GetUser()
		if user == nil || strings.TrimSpace(user.GetId()) == "" {
			return unauthorized(ctx, "invalid token")
		}

		roleIDs := make([]string, 0, len(resp.GetRoles()))
		for _, role := range resp.GetRoles() {
			if strings.TrimSpace(role.GetId()) != "" {
				roleIDs = append(roleIDs, role.GetId())
			}
		}
		setUserValue(ctx, authctx.MetadataUserID, user.GetId())
		setUserValue(ctx, authctx.MetadataUsername, user.GetUsername())
		setUserValue(ctx, authctx.MetadataRoleIDs, strings.Join(roleIDs, ","))
		clearExternalCredentialHeaders(ctx)
		return ctx.Next()
	}
}

// The gateway has already validated the browser credential. Downstream gRPC
// services receive only the trusted identity metadata populated above, never
// the external bearer token itself.
func clearExternalCredentialHeaders(ctx *fiber.Ctx) {
	ctx.Request().Header.Del(fiber.HeaderAuthorization)
	ctx.Request().Header.Del("X-Access-Token")
}

func isBridgeToken(ctx *fiber.Ctx, bearer string) bool {
	expected := strings.TrimSpace(global.GetConfig().AuthConfig.BridgeToken)
	if expected == "" {
		return false
	}
	token := strings.TrimSpace(ctx.Get("X-Bridge-Token"))
	if token == "" {
		token = strings.TrimSpace(bearer)
	}
	return token == expected
}

func isPublicRequest(ctx *fiber.Ctx) bool {
	if strings.EqualFold(ctx.Method(), fiber.MethodOptions) {
		return true
	}
	path := strings.TrimRight(ctx.Path(), "/")
	if path == "/system/auth/login" {
		return true
	}
	if path == "/realtime/ws" {
		return true
	}
	if path == "/ai-agent/agent-bots/runtime" {
		return true
	}
	if strings.EqualFold(ctx.Method(), fiber.MethodPost) && strings.HasPrefix(path, "/func-operation/swagger-import/") {
		return true
	}
	return false
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return header
}

func setUserValue(ctx *fiber.Ctx, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	ctx.Context().SetUserValue(key, value)
	// BaseHandler.RPCCtx forwards only explicitly namespaced user values to
	// gRPC metadata. Keep the local value for Fiber handlers and add the
	// namespaced copy for downstream authorization.
	ctx.Context().SetUserValue("grpc-metadata-"+key, value)
	ctx.Locals(key, value)
}

func unauthorized(ctx *fiber.Ctx, message string) error {
	return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"code": fiber.StatusUnauthorized,
		"msg":  message,
	})
}
