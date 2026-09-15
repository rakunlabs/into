package into

import (
	"sync"
	"time"
)

type timeout struct {
	Duration time.Duration
	WaitFn   func()
}

func newTimeout(duration time.Duration, waitFn func()) *timeout {
	return &timeout{Duration: duration, WaitFn: waitFn}
}

// wait blocks until wg is done or Duration elapses, and reports whether the
// timeout was reached.
//
// The watcher goroutine below outlives wait when the wait group never completes.
// That is inherent: a WaitGroup cannot be abandoned. Nothing else is left
// running, and the result is returned without sharing mutable state between
// goroutines.
func (t *timeout) wait(wg *sync.WaitGroup) bool {
	done := make(chan struct{})

	go func() {
		defer close(done)

		wg.Wait()
	}()

	timer := time.NewTimer(t.Duration)
	defer timer.Stop()

	select {
	case <-done:
		return false
	case <-timer.C:
		if t.WaitFn != nil {
			t.WaitFn()
		} else {
			logger.Warn("timeout reached while waiting WaitGroup")
		}

		return true
	}
}
