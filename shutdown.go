package into

import (
	"context"
	"sync"
)

type shutdownType struct {
	ctxCancel context.CancelFunc
	funcs     []shutdownInfo

	mutex sync.Mutex
}

type shutdownInfo struct {
	name string
	fn   func() error
}

func ShutdownAdd(fn func() error, options ...OptionShutdownAdd) {
	shutdown.Add(fn, options...)
}

func (s *shutdownType) setCtxCancel(ctxCancel context.CancelFunc) {
	s.ctxCancel = ctxCancel
}

// Cancel is a function that cancels the root context.
//
// This helps to stop the application gracefully without any errors.
func (s *shutdownType) CtxCancel() {
	if s.ctxCancel == nil {
		return
	}

	s.ctxCancel()
}

func (s *shutdownType) Add(fn func() error, options ...OptionShutdownAdd) {
	option := optionShutdownAdd{}
	for _, opt := range options {
		opt(&option)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.funcs = append(s.funcs, shutdownInfo{
		name: option.name,
		fn:   fn,
	})
}

func (s *shutdownType) Run(once bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// run opposite order
	for i := len(s.funcs) - 1; i >= 0; i-- {
		inf := s.funcs[i]

		if err := inf.fn(); err != nil {
			logger.Error("shutdown error", "name", inf.name, "error", err.Error())
		}
	}
}

type optionShutdownAdd struct {
	name string
}

type OptionShutdownAdd func(options *optionShutdownAdd)

func WithShutdownName(name string) OptionShutdownAdd {
	return func(options *optionShutdownAdd) {
		options.name = name
	}
}
