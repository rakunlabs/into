package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/rakunlabs/into"
)

func main() {
	into.Init(run,
		into.WithLogger(slog.Default()),
		into.WithMsgf("myservice [%s]", "v0.1.0"),
		into.WithWaitTimeout(5*time.Second),
		into.WithWaitFn(func() {
			slog.Warn("timeout")
		}),
	)
}

func run(ctx context.Context) error {
	wg := into.WaitGroup(ctx)
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-time.After(6 * time.Second)
	}()

	into.SetCtxCancelFn(ctx, func(cancel context.CancelFunc) {
		slog.Warn("canceled")
		cancel()
	})

	time.Sleep(2 * time.Second)

	return nil
}
