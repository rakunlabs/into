package into

import (
	"context"
	"sync"
)

type ctxKey string

const ctxKeyInto ctxKey = "into"

type intoType struct {
	opt *option
	wg  *sync.WaitGroup
}

// SetContextWaitGroup sets the into's signal cancel function.
func SetCtxCancelFn(ctx context.Context, fn func(cancel context.CancelFunc)) {
	t, _ := ctx.Value(ctxKeyInto).(*intoType)

	if t != nil {
		t.opt.SetContextCancelFn(fn)
	}
}

// WaitGroup returns the into main wait group from the context.
func WaitGroup(ctx context.Context) *sync.WaitGroup {
	t, _ := ctx.Value(ctxKeyInto).(*intoType)

	if t != nil {
		return t.wg
	}

	return nil
}

func setIntoValue(ctx context.Context, into *intoType) context.Context {
	return context.WithValue(ctx, ctxKeyInto, into)
}
