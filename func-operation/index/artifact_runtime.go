package index

import (
	"context"
	"sync"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/logic"
	"github.com/gly-hub/quickgo/logger"
)

var (
	artifactRuntimeMu sync.Mutex
	artifactCancel    context.CancelFunc
	artifactDone      chan struct{}
)

func StartArtifactRuntime(releases *logic.ReleaseLogic, interval, staleStagingAfter time.Duration) {
	if releases == nil {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	if staleStagingAfter <= 0 {
		staleStagingAfter = time.Hour
	}
	artifactRuntimeMu.Lock()
	if artifactCancel != nil {
		artifactRuntimeMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	artifactCancel, artifactDone = cancel, done
	artifactRuntimeMu.Unlock()
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := releases.ReconcileArtifactStore(ctx, staleStagingAfter); err != nil {
					logger.Error(ctx, "reconcile function artifacts: %v", err)
				}
			}
		}
	}()
}

func StopArtifactRuntime() {
	artifactRuntimeMu.Lock()
	cancel, done := artifactCancel, artifactDone
	artifactCancel, artifactDone = nil, nil
	artifactRuntimeMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
