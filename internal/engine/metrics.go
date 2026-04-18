package engine

import (
	"time"

	"github.com/brainlet/brainkit/internal/types"
)


// Metrics returns a point-in-time snapshot of internal Kernel state.
func (k *Kernel) Metrics() types.KernelMetrics {
	scheduleCount := 0
	if h := k.scheduleHandler; h != nil {
		scheduleCount = len(h.List())
	}
	m := types.KernelMetrics{
		ActiveHandlers:    k.activeHandlers.Load(),
		ActiveDeployments: len(k.ListDeployments()),
		ActiveSchedules:   scheduleCount,
		PumpCycles:        k.pumpCycles.Load(),
	}

	if !k.startedAt.IsZero() {
		m.Uptime = time.Since(k.startedAt)
	}

	// Bus per-topic metrics
	if k.busMetrics != nil {
		snap := k.busMetrics.Snapshot()
		m.Bus = &snap
	}

	// Plugin metrics are now sourced from modules/plugins (which holds the
	// WS server state). metrics.get returns no plugin details when the module
	// is absent; see modules/plugins if richer metrics are needed.

	return m
}
