package into

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	DefaultExitCode  = 0
	DefaultWgTimeout = 1 * time.Minute

	// exitCode is the failure code. It is set when the run function returns
	// an error or when shutdown exceeds the wait-group timeout. Zero means
	// "no failure". It is written by the main goroutine and read in the exit
	// defer, hence atomic.
	exitCode atomic.Int32
	// signalCode carries the conventional 128+signum status when a caught
	// signal initiated the shutdown. It is written by the signal goroutine
	// and read in the exit defer, hence atomic.
	signalCode atomic.Int32

	// logger defaults to a no-op so any code path reached before Init, such as a
	// direct handler registration, cannot nil dereference.
	logger LogAdapter = LogNoop{}

	shutdown shutdownType
)

// signalExitCode maps a caught signal to its conventional exit status,
// 128+signum: 130 for SIGINT, 143 for SIGTERM.
//
// A graceful shutdown that ran to completion is not a failure, so this is
// deliberately distinct from the generic failure code 1 that reports a run
// error or a shutdown that timed out. Supervisors should treat it as success:
// with systemd, add `SuccessExitStatus=143` to the unit.
func signalExitCode(sig os.Signal) int32 {
	if s, ok := sig.(syscall.Signal); ok && s > 0 {
		return 128 + int32(s)
	}

	return 128
}

// resolveExitCode picks the process status from the failure code, the signal
// code and the configured default, in that order of precedence. A real failure
// outranks "stopped by signal" so a shutdown timeout stays visible.
func resolveExitCode(failure, signal int32, def int) int {
	switch {
	case failure != 0:
		return int(failure)
	case signal != 0:
		return int(signal)
	default:
		return def
	}
}

func Init(fn func(context.Context) error, options ...Option) {
	opt := newOption(options...)

	logger = opt.logger
	if logger == nil {
		logger = LogNoop{}
	}

	// args to check commands
	commands(opt.ctx, opt)

	if opt.errExitCode == nil {
		opt.errExitCode = func(_ error) int { return 1 }
	}

	if opt.signalExitCode == nil {
		opt.signalExitCode = signalExitCode
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

		os.Exit(resolveExitCode(exitCode.Load(), signalCode.Load(), DefaultExitCode))
	}()

	wg := sync.WaitGroup{}
	ctx, ctxCancel := context.WithCancel(opt.ctx)

	defer func() {
		if opt.wgWaitTimeout > 0 {
			if v := newTimeout(opt.wgWaitTimeout, opt.waitFn).wait(&wg); v {
				// Shutdown did not finish in time: a genuine failure, so it
				// must outrank any signal code already recorded.
				exitCode.CompareAndSwap(0, 1)
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
			case sig := <-signalChan:
				if opt.ctxCancelFn != nil {
					opt.ctxCancelFn(ctxCancel)
				} else {
					logger.Warn("received shutdown signal", "signal", sig.String())

					ctxCancel()
				}

				signalCode.CompareAndSwap(0, opt.signalExitCode(sig))
			}
		}

		shutdown.Run()
	}()

	// Derive the value-carrying context into a new variable instead of
	// rebinding ctx: the signal goroutine above captured ctx, so assigning to
	// it here is a data race on the variable itself. Cancellation is shared,
	// because runCtx wraps ctx.
	runCtx := setIntoValue(ctx, &intoType{
		wg:  &wg,
		opt: opt,
	})

	// health check server. A bind failure is fatal: running on without the
	// endpoints an orchestrator probes only produces a slower, more confusing
	// outage than refusing to start.
	if err := serve(runCtx, opt.serverOptions); err != nil {
		logger.Error("init server: "+opt.msg, "error", err.Error())
		exitCode.Store(1)

		return
	}

	// run main function
	if err := fn(runCtx); err != nil {
		exitCode.Store(int32(opt.errExitCode(err)))

		if opt.runErrFn != nil {
			opt.runErrFn(err)
		} else {
			logger.Error("service closing: "+opt.msg, "error", err.Error())
		}
	}
}
