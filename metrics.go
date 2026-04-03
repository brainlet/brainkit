package brainkit

import (
	"time"
)

// KernelMetrics is a point-in-time snapshot of internal Kernel state.
type KernelMetrics struct {
	// Counters
	ActiveHandlers    int64         `json:"activeHandlers"`
	ActiveDeployments int           `json:"activeDeployments"`
	ActiveSchedules   int           `json:"activeSchedules"`
	PumpCycles        int64         `json:"pumpCycles"`

	// Uptime
	Uptime time.Duration `json:"uptime"`
}

// Metrics returns a point-in-time snapshot of internal Kernel state.
func (k *Kernel) Metrics() KernelMetrics {
	m := KernelMetrics{
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
