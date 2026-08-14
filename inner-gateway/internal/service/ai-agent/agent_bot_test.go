package aiagent

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/team-dandelion/quickgo/grpcep"
	"google.golang.org/grpc/metadata"
)

func TestClearRuntimeBridgeCredentialHeadersDoesNotForwardAuthorization(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		clearRuntimeBridgeCredentialHeaders(ctx)
		rpcCtx := (&grpcep.BaseHandler{}).RPCCtx(ctx)
		md, _ := metadata.FromOutgoingContext(rpcCtx)
		if got := md.Get("authorization"); len(got) != 0 {
			t.Fatalf("authorization was forwarded to gRPC: %v", got)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set(fiber.HeaderAuthorization, "Bearer bridge-token")
	req.Header.Set("X-Bridge-Token", "bridge-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("run request: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("response status = %d, want %d", resp.StatusCode, fiber.StatusNoContent)
	}
}
