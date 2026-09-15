package into

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

type option struct {
	msg           string
	ctx           context.Context //nolint:containedctx // temporary
	logger        LogAdapter
	wgWaitTimeout time.Duration
	errExitCode   func(error) int
	// signalExitCode is nil until Init defaults it to the package-level
	// signalExitCode (128+signum).
	signalExitCode func(os.Signal) int32
	ctxCancelFn    func(cancel context.CancelFunc)

	runErrFn func(error)
	startFn  func()
	stopFn   func()
	waitFn   func()

	serverOptions *optionServer
}

func newOption(options ...Option) *option {
	option := &option{
		ctx:           context.Background(),
		wgWaitTimeout: DefaultWgTimeout,
		logger:        slog.Default(),
	}

	for _, opt := range options {
		opt(option)
	}

	return option
}

func (o *option) SetContextCancelFn(fn func(cancel context.CancelFunc)) {
	o.ctxCancelFn = fn
}

type Option func(options *option)

func WithHealthCheck(opts ...OptionHealthCheck) Option {
	return func(options *option) {
		if options.serverOptions == nil {
			options.serverOptions = &optionServer{}
		}

		if options.serverOptions.healthCheckOption == nil {
			// Enabled by default: the sub-options below can still turn the
			// endpoint off, e.g. WithHealthCheckCustom.
			options.serverOptions.healthCheckOption = &optionHealthCheck{EndpointEnabled: true}
		}

		for _, opt := range opts {
			opt(options.serverOptions.healthCheckOption)
		}
	}
}

func WithKill(opts ...OptionKill) Option {
	return func(options *option) {
		if options.serverOptions == nil {
			options.serverOptions = &optionServer{}
		}

		if options.serverOptions.killOption == nil {
			options.serverOptions.killOption = &optionKill{}
		}

		for _, opt := range opts {
			opt(options.serverOptions.killOption)
		}
	}
}

func WithServer(opts ...OptionServer) Option {
	return func(options *option) {
		if options.serverOptions == nil {
			options.serverOptions = &optionServer{}
		}

		for _, opt := range opts {
			opt(options.serverOptions)
		}
	}
}

// WithMsgf is a function that sets the message to be logged when the application starts and stops.
//
// This will override the default message.
func WithMsgf(format string, a ...any) Option {
	return func(options *option) {
		options.msg = fmt.Sprintf(format, a...)
	}
}

// WithContext is a function that sets the context to be used as parent context.
func WithContext(ctx context.Context) Option {
	return func(options *option) {
		if ctx != nil {
			options.ctx = ctx
		}
	}
}

// WithWaitTimeout is a function that sets the wait timeout for the wait group.
func WithWaitTimeout(duration time.Duration) Option {
	return func(options *option) {
		options.wgWaitTimeout = duration
	}
}

// WithErrExitCode is a function that sets the exit code when an error occurs from main function.
func WithErrExitCode(fn func(err error) int) Option {
	return func(options *option) {
		options.errExitCode = fn
	}
}

// WithSignalExitCode sets the exit code used when a caught signal initiates
// the shutdown and that shutdown completes without error.
//
// The default is the conventional 128+signum (130 for SIGINT, 143 for SIGTERM).
// Pass `func(os.Signal) int32 { return 1 }` to restore the pre-v0.6.0 behaviour
// of reporting a generic failure. A run error or a shutdown timeout always
// outranks this code.
func WithSignalExitCode(fn func(sig os.Signal) int32) Option {
	return func(options *option) {
		if fn == nil {
			return
		}

		options.signalExitCode = fn
	}
}

// WithLogger is a function that sets the logger to be used.
// If not set, it is no-op.
func WithLogger(logger LogAdapter) Option {
	return func(options *option) {
		options.logger = logger
	}
}

// WithCtxCancelFn is a function that sets the cancel function to be called when the shutdown signal is received.
//   - This is useful when you want to cancel the context manually like multiple ctrl+c signals.
func WithCtxCancelFn(fn func(cancel context.CancelFunc)) Option {
	return func(options *option) {
		options.ctxCancelFn = fn
	}
}

// WithRunErrFn function for replace custom error message return from `Run` function.
func WithRunErrFn(fn func(error)) Option {
	return func(options *option) {
		if fn == nil {
			fn = func(_ error) {}
		}

		options.runErrFn = fn
	}
}

// WithStartFn function for replace custom starting log message.
//   - Set nil to disable.
func WithStartFn(fn func()) Option {
	return func(options *option) {
		if fn == nil {
			fn = func() {}
		}

		options.startFn = fn
	}
}

// WithStopFn function for replace custom stopping log message.
//   - Set nil to disable.
func WithStopFn(fn func()) Option {
	return func(options *option) {
		if fn == nil {
			fn = func() {}
		}

		options.stopFn = fn
	}
}

// WithWaitFn function for replace custom waiting log message.
// This function will be called after timeout of wait group.
//   - Set nil to disable.
func WithWaitFn(fn func()) Option {
	return func(options *option) {
		if fn == nil {
			fn = func() {}
		}

		options.waitFn = fn
	}
}
