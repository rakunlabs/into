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

// ShutdownAdd is a function that adds a function to the shutdown. This function will be called when the context is done.
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
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ctxCancel = ctxCancel
}

// CtxCancel is a function that cancels the root context.
//
// This helps to stop the application gracefully without any errors.
func (s *shutdownType) CtxCancel() {
	s.mutex.Lock()
	ctxCancel := s.ctxCancel
	s.mutex.Unlock()

	if ctxCancel == nil {
		return
	}

	// Called outside the lock: cancelling wakes the signal goroutine, which
	// runs Run and would deadlock on a held mutex.
	ctxCancel()
}

func (s *shutdownType) Add(fn func() error, name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i := range s.funcs {
		if s.funcs[i].name == name {
			s.funcs[i].fn = fn

			return
		}
	}

	s.funcs = append(s.funcs, shutdownInfo{
		name: name,
		fn:   fn,
	})
}

func (s *shutdownType) Run() {
	// Snapshot under the lock and release it before calling out: the mutex is
	// not reentrant, so a shutdown function calling ShutdownAdd would deadlock.
	s.mutex.Lock()
	funcs := make([]shutdownInfo, len(s.funcs))
	copy(funcs, s.funcs)
	s.mutex.Unlock()

	// run opposite order
	for i := len(funcs) - 1; i >= 0; i-- {
		inf := funcs[i]

		if inf.fn == nil {
			logger.Warn("shutdown function is nil", "name", inf.name)
			continue
		}

		if err := inf.fn(); err != nil {
			logger.Error("shutdown error", "name", inf.name, "error", err.Error())
		}
	}
}
