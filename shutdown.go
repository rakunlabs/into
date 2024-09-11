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

// CtxCancel is a function that cancels the root context.
func CtxCancel() {
	shutdown.CtxCancel()
}

// Add is a function that adds a function to the shutdown. This function will be called when the context is done.
func ShutdownAdd(fn func() error, name string) {
	shutdown.Add(fn, name)
}

// FnWarp is a function that wraps a function to be used in the shutdown.
func FnWarp(fn func()) func() error {
	return func() error {
		if fn != nil {
			fn()
		}

		return nil
	}
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

func (s *shutdownType) Add(fn func() error, name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.funcs = append(s.funcs, shutdownInfo{
		name: name,
		fn:   fn,
	})
}

func (s *shutdownType) Run() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// run opposite order
	for i := len(s.funcs) - 1; i >= 0; i-- {
		inf := s.funcs[i]

		if inf.fn == nil {
			logger.Warn("shutdown function is nil", "name", inf.name)
			continue
		}

		if err := inf.fn(); err != nil {
			logger.Error("shutdown error", "name", inf.name, "error", err.Error())
		}
	}
}
