package aiagent

import (
	"context"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func TestGetAiAgentClientUsesConnectionPoolForEveryRequest(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///unused", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("create client connection: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	manager := &countingClientManager{conn: conn}
	controller := NewAIAgentServerController(manager)
	const requests = 128

	var wg sync.WaitGroup
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := controller.getAiAgentClient(context.Background()); err != nil {
				t.Errorf("get client: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := manager.callCount(); got != requests {
		t.Fatalf("GetConn calls = %d, want %d", got, requests)
	}
}
