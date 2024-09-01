package into

import (
	"context"
	"fmt"
	"time"

	"github.com/rakunlabs/logi/logadapter"
)

type option struct {
	msg           string
	ctx           context.Context //nolint:containedctx // temporary
	logger        logadapter.Adapter
	wgWaitTimeout time.Duration
	errExitCode   func(error) int
	ctxCancelFn   func(cancel context.CancelFunc)

	runErrFn func(error)
	startFn  func()
	stopFn   func()
	waitFn   func()
}

func (o *option) SetContextCancelFn(fn func(cancel context.CancelFunc)) {
	o.ctxCancelFn = fn
}

type Option func(options *option)

// WithMsg is a function that sets the message to be logged when the application starts and stops.
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

// WithLogger is a function that sets the logger to be used.
// If not set, it is no-op.
func WithLogger(logger logadapter.Adapter) Option {
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
func WithStartFn(fn func()) Option {
	return func(options *option) {
		if fn == nil {
			fn = func() {}
		}

		options.startFn = fn
	}
}

// WithStopFn function for replace custom stopping log message.
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
func WithWaitFn(fn func()) Option {
	return func(options *option) {
		if fn == nil {
			fn = func() {}
		}

		options.waitFn = fn
	}
}

func newOption(options ...Option) *option {
	option := &option{
		ctx:           context.Background(),
		wgWaitTimeout: DefaultWgTimeout,
	}

	for _, opt := range options {
		opt(option)
	}

	return option
}
