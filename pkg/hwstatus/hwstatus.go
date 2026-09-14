// Package hwstatus owns the configurable hardware-status push interval that is
// shared between:
//
//   - route/periodical.go — the background push loop (sends hardware status
//     over the socket), which exposes Start/Stop/Get/Set wrappers per the
//     "route" API surface used by main.go, and
//   - route/v1/system.go     — the GET/PUT /sys/utilization/interval endpoints.
//
// The state lives in its own package because route/v1 cannot import the route
// package (route imports route/v1, so that would be an import cycle), yet both
// the loop and the HTTP handlers must act on the same interval state.
package hwstatus

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	minInterval = 250 * time.Millisecond
	maxInterval = 5 * time.Second
	defaultMS   = 5000
)

// intervalNanos holds the current push interval as atomic int64 nanoseconds.
// It is the single source of truth; resetCh is only a wake-up signal so an
// HTTP handler (GetInterval) always reads a race-free value.
var intervalNanos int64 = int64(defaultMS * time.Millisecond)

var (
	resetCh  = make(chan struct{}, 1)
	stopCh   = make(chan struct{})
	stopOnce sync.Once
)

// PushFunc is invoked once per interval tick.
type PushFunc func()

// Start runs the per-interval push loop in a background goroutine until Stop
// is called (or the process exits). Call SetInterval to change the cadence
// from a running loop.
func Start(push PushFunc) {
	go func() {
		d := time.Duration(atomic.LoadInt64(&intervalNanos))
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				push()
			case <-resetCh:
				// Re-read the latest interval; collapsed/wakeup-only signal so
				// a dropped signal can never leave the cadence stale.
				d = time.Duration(atomic.LoadInt64(&intervalNanos))
				ticker.Reset(d)
			case <-stopCh:
				return
			}
		}
	}()
}

// Stop signals the background loop to stop. Safe to call more than once;
// the loop goroutine terminates after the process exits anyway.
func Stop() {
	stopOnce.Do(func() { close(stopCh) })
}

// GetInterval is the current push interval in milliseconds.
func GetInterval() int {
	return int(time.Duration(atomic.LoadInt64(&intervalNanos)) / time.Millisecond)
}

// SetInterval clamps ms to [250, 5000] (0 → default 5000), stores it
// atomically, wakes the running loop, and returns the applied value.
func SetInterval(ms int) int {
	if ms == 0 {
		ms = defaultMS
	}
	if ms < int(minInterval/time.Millisecond) {
		ms = int(minInterval / time.Millisecond)
	}
	if ms > int(maxInterval/time.Millisecond) {
		ms = int(maxInterval / time.Millisecond)
	}
	atomic.StoreInt64(&intervalNanos, int64(time.Duration(ms)*time.Millisecond))
	// Non-blocking wake-up: the loop re-reads intervalNanos on any signal, so
	// a collapsed signal is harmless.
	select {
	case resetCh <- struct{}{}:
	default:
	}
	return ms
}
