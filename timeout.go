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

func (t *timeout) wait(wg *sync.WaitGroup) bool {
	timerWait := time.NewTimer(t.Duration)
	defer timerWait.Stop()

	wgTimeout := &sync.WaitGroup{}
	wgTimeout.Add(2)

	mutex := &sync.Mutex{}
	canceled := false

	timeoutReached := false

	go func() {
		defer func() {
			mutex.Lock()
			defer mutex.Unlock()

			if t.WaitFn != nil {
				t.WaitFn()
			} else {
				logger.Warn("timeout reached while waiting WaitGroup")
			}

			timeoutReached = true

			if !canceled {
				wgTimeout.Add(-2)
			}

			canceled = true
		}()

		<-timerWait.C
	}()

	go func() {
		defer func() {
			mutex.Lock()
			defer mutex.Unlock()

			if !canceled {
				wgTimeout.Add(-2)
			}

			canceled = true
		}()

		wg.Wait()
	}()

	wgTimeout.Wait()

	return timeoutReached
}
