package into

import (
	"context"
	"sync"
)

type ctxKey string

const (
	ctxKeyWaitGroup ctxKey = "waitGroup"
)

func WaitGroup(ctx context.Context) *sync.WaitGroup {
	wg, _ := ctx.Value(ctxKeyWaitGroup).(*sync.WaitGroup)

	return wg
}

func setWaitGroup(ctx context.Context, wg *sync.WaitGroup) context.Context {
	return context.WithValue(ctx, ctxKeyWaitGroup, wg)
}
