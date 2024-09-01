package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/rakunlabs/into"
)

func main() {
	into.Run(run,
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

	return nil
}
