package index

import (
	"context"
	"sync"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/logic"
	"github.com/gly-hub/quickgo/logger"
)

var (
	outboxRuntimeMu sync.Mutex
	outboxCancel    context.CancelFunc
	outboxDone      chan struct{}
)

func StartOutboxRuntime(processor *logic.OutboxProcessor) {
	if processor == nil {
		return
	}
	outboxRuntimeMu.Lock()
	if outboxCancel != nil {
		outboxRuntimeMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	outboxCancel, outboxDone = cancel, done
	outboxRuntimeMu.Unlock()
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := processor.ProcessReady(ctx, 50); err != nil {
					logger.Error(ctx, "process function outbox: %v", err)
				}
			}
		}
	}()
}

func StopOutboxRuntime() {
	outboxRuntimeMu.Lock()
	cancel, done := outboxCancel, outboxDone
	outboxCancel, outboxDone = nil, nil
	outboxRuntimeMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
