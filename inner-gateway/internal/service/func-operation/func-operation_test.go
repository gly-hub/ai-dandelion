package funcoperation

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gly-hub/quickgo/grpcep"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type countingClientManager struct {
	conn  *grpc.ClientConn
	mu    sync.Mutex
	calls int
}

func (m *countingClientManager) GetConn(context.Context, string) (*grpc.ClientConn, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return m.conn, nil
}

func (m *countingClientManager) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestGetFuncOperationClientUsesConnectionPoolForEveryRequest(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///unused", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("create client connection: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	manager := &countingClientManager{conn: conn}
	controller := NewFuncOperationServerController(manager)
	const requests = 128

	var wg sync.WaitGroup
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := controller.getFuncOperationClient(context.Background()); err != nil {
				t.Errorf("get client: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := manager.callCount(); got != requests {
		t.Fatalf("GetConn calls = %d, want %d", got, requests)
	}
}

func TestForwardGeneratedAppRequestIDPropagatesMiddlewareRequestID(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		ctx.Locals("request_id", "generated-request-id")
		forwardGeneratedAppRequestID(ctx)
		rpcCtx := (&grpcep.BaseHandler{}).RPCCtx(ctx)
		metadata, _ := metadata.FromOutgoingContext(rpcCtx)
		if got := metadata.Get("x-trace-id"); len(got) != 1 || got[0] != "generated-request-id" {
			t.Fatalf("x-trace-id = %v, want generated request ID", got)
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
