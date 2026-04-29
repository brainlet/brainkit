package jsruntime

import (
	"context"
	"time"
)

// --- Job pump ---
//
// Lives in core (not in modules/schedules) because it drives QuickJS
// microtasks — it's required whenever the runtime has any pending
// Promise/async activity, not just when user-level schedules exist.

// startJobPump starts a background goroutine that processes QuickJS scheduled
// callbacks AND JS microtasks. Wakes immediately when Schedule'd callbacks are
// pending (via pumpSignal), with a 100ms fallback for pure-JS microtasks.
//
// Uses bridge.Go() so the goroutine is tracked by bridge.wg — Close() waits
// for it to finish before touching the QuickJS context.
func (r *Runtime) startJobPump() {
	if r.bridge == nil {
		return
	}
	fallback := time.NewTicker(100 * time.Millisecond)
	pumpSignal := r.bridge.PumpSignal()

	r.bridge.Go(func(goCtx context.Context) {
		defer fallback.Stop()
		for {
			select {
			case <-pumpSignal:
				r.processScheduledJobs()
			case <-fallback.C:
				r.processScheduledJobs()
			case <-goCtx.Done():
				return
			}
		}
	})
}

func (r *Runtime) processScheduledJobs() {
	if r.host.IsClosed() {
		return
	}
	r.host.IncrementPumpCycles()
	if r.bridge == nil {
		return
	}
	r.bridge.ProcessScheduledJobs()
}
