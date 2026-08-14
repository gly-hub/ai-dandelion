package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc/metadata"
)

func TestSetUserValueForwardsIdentityToGRPCMetadata(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		setUserValue(ctx, authctx.MetadataUserID, "user-1")
		setUserValue(ctx, authctx.MetadataUsername, "admin")
		setUserValue(ctx, authctx.MetadataRoleIDs, "role-admin")

		rpcCtx := (&grpcep.BaseHandler{}).RPCCtx(ctx)
		md, ok := metadata.FromOutgoingContext(rpcCtx)
		if !ok {
			t.Fatal("outgoing gRPC metadata is missing")
		}
		if got := md.Get(authctx.MetadataUserID); len(got) != 1 || got[0] != "user-1" {
			t.Fatalf("x-user-id = %v, want [user-1]", got)
		}
		if got := md.Get(authctx.MetadataRoleIDs); len(got) != 1 || got[0] != "role-admin" {
			t.Fatalf("x-role-ids = %v, want [role-admin]", got)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("run request: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("response status = %d, want %d", resp.StatusCode, fiber.StatusNoContent)
	}
}

func TestClearExternalCredentialHeadersKeepsIdentityButDoesNotForwardJWT(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		setUserValue(ctx, authctx.MetadataUserID, "user-1")
		setUserValue(ctx, authctx.MetadataUsername, "admin")
		clearExternalCredentialHeaders(ctx)

		rpcCtx := (&grpcep.BaseHandler{}).RPCCtx(ctx)
		md, ok := metadata.FromOutgoingContext(rpcCtx)
		if !ok {
			t.Fatal("outgoing gRPC metadata is missing")
		}
		if got := md.Get("authorization"); len(got) != 0 {
			t.Fatalf("authorization was forwarded to gRPC: %v", got)
		}
		if got := md.Get(authctx.MetadataUserID); len(got) != 1 || got[0] != "user-1" {
			t.Fatalf("x-user-id = %v, want [user-1]", got)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set(fiber.HeaderAuthorization, "Bearer external-jwt")
	req.Header.Set("X-Access-Token", "external-jwt")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("run request: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("response status = %d, want %d", resp.StatusCode, fiber.StatusNoContent)
	}
}

func TestSwaggerUploadIsPublicOnlyForPost(t *testing.T) {
	app := fiber.New()
	app.Use(func(ctx *fiber.Ctx) error {
		if !isPublicRequest(ctx) {
			return ctx.SendStatus(fiber.StatusUnauthorized)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})
	post, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/func-operation/swagger-import/game-server", nil))
	if err != nil || post.StatusCode != fiber.StatusNoContent {
		t.Fatalf("Swagger upload public route = %#v, %v", post, err)
	}
	get, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/func-operation/swagger-import/game-server", nil))
	if err != nil || get.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("Swagger upload GET route = %#v, %v", get, err)
	}
}
