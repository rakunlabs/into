package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/rakunlabs/into"
)

func main() {
	into.Init(run,
		into.WithLogger(slog.Default()),
		into.WithMsgf("myservice [%s]", "v0.1.0"),
		// into.WithWaitTimeout(5*time.Second),
		// into.WithWaitFn(func() {
		// 	slog.Warn("timeout")
		// }),
		into.WithHealthCheck(),
	)
}

func run(ctx context.Context) error {
	// wg := into.WaitGroup(ctx)
	// wg.Add(1)

	// go func() {
	// 	defer wg.Done()
	// 	<-time.After(6 * time.Second)
	// }()

	// into.SetCtxCancelFn(ctx, func(cancel context.CancelFunc) {
	// 	slog.Warn("canceled")
	// 	cancel()
	// })

	into.HealthCheck = func(ctx context.Context) error {
		return errors.New("some problem")
	}

	<-ctx.Done()

	return nil
}
