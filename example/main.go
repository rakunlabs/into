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
		into.WithHealthCheck(),
		into.WithKill(),
	)
}

func run(ctx context.Context) error {
	into.HealthCheck = func(ctx context.Context) error {
		return errors.New("some problem")
	}

	<-ctx.Done()

	return nil
}
