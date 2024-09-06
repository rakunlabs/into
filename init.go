package into

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	DefaultExitCode  = 0
	DefaultWgTimeout = 1 * time.Minute

	exitCode = 0
	logger   LogAdapter

	shutdown shutdownType
)

func Init(fn func(context.Context) error, options ...Option) {
	opt := newOption(options...)
	logger = opt.logger
	if logger == nil {
		logger = LogNoop{}
	}

	if opt.errExitCode == nil {
		opt.errExitCode = func(_ error) int { return 1 }
	}

	if opt.startFn != nil {
		opt.startFn()
	} else {
		logger.Info("starting " + opt.msg)
	}

	defer func() {
		if opt.stopFn != nil {
			opt.stopFn()
		} else {
			logger.Info("closing " + opt.msg)
		}

		if r := recover(); r != nil {
			panic(r)
		}

		exitC := DefaultExitCode
		if exitCode != 0 {
			exitC = exitCode
		}

		os.Exit(exitC)
	}()

	wg := sync.WaitGroup{}
	ctx, ctxCancel := context.WithCancel(opt.ctx)

	defer func() {
		if opt.wgWaitTimeout > 0 {
			if v := newTimeout(opt.wgWaitTimeout, opt.waitFn).wait(&wg); v {
				if exitCode == 0 {
					exitCode = 1
				}
			}
		} else {
			wg.Wait()
		}
	}()

	// cancel context before wait group
	defer ctxCancel()

	shutdown.setCtxCancel(ctxCancel)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()

	DEFERFOR:
		for {
			select {
			case <-ctx.Done():
				break DEFERFOR
			case <-signalChan:
				if opt.ctxCancelFn != nil {
					opt.ctxCancelFn(ctxCancel)
				} else {
					logger.Warn("received shutdown signal")

					ctxCancel()
				}
				if exitCode == 0 {
					exitCode = 1
				}
			}
		}

		shutdown.Run(true)
	}()

	ctx = setIntoValue(ctx, &intoType{
		wg:  &wg,
		opt: opt,
	})

	if err := fn(ctx); err != nil {
		exitCode = opt.errExitCode(err)

		if opt.runErrFn != nil {
			opt.runErrFn(err)
		} else {
			logger.Error("service closing: "+opt.msg, slog.String("error", err.Error()))
		}
	}
}
