package runtime

import (
	"context"
	"fmt"
	"runtime/debug"
)

func GOSafe(ctx context.Context, tag string, f func()) {
	go func(gCtx context.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())
				fmt.Println(stack)
			}
		}()
		f()
	}(ctx)
}
