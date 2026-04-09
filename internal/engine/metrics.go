package engine

import (
	"time"

	"github.com/brainlet/brainkit/internal/types"
)


// Metrics returns a point-in-time snapshot of internal Kernel state.
func (k *Kernel) Metrics() types.KernelMetrics {
	m := types.KernelMetrics{
		ActiveHandlers:    k.activeHandlers.Load(),
		ActiveDeployments: len(k.ListDeployments()),
		ActiveSchedules:   len(k.ListSchedules()),
		PumpCycles:        k.pumpCycles.Load(),
	}

	if !k.startedAt.IsZero() {
		m.Uptime = time.Since(k.startedAt)
	}

	return m
}
