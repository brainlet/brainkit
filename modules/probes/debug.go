package probes

import "time"

// DebugSnapshot is the probes module lifecycle debug view. It reports only
// module-owned loop state and never runs probes itself.
type DebugSnapshot struct {
	RunnerAttached    bool       `json:"runnerAttached"`
	LoopRunning       bool       `json:"loopRunning"`
	Closing           bool       `json:"closing"`
	Closed            bool       `json:"closed"`
	ActiveSweeps      int64      `json:"activeSweeps"`
	IntervalSeconds   int64      `json:"intervalSeconds,omitempty"`
	ProbeOnRegister   bool       `json:"probeOnRegister"`
	LastSweepStarted  *time.Time `json:"lastSweepStarted,omitempty"`
	LastSweepFinished *time.Time `json:"lastSweepFinished,omitempty"`
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.Lock()
	snapshot := DebugSnapshot{
		RunnerAttached:  m.runner != nil,
		LoopRunning:     m.loopRunning,
		Closing:         m.closing,
		Closed:          m.closed,
		ActiveSweeps:    m.activeSweeps.Load(),
		IntervalSeconds: int64(m.cfg.Interval.Seconds()),
		ProbeOnRegister: m.probeOnRegister(),
	}
	if !m.lastSweepStarted.IsZero() {
		started := m.lastSweepStarted
		snapshot.LastSweepStarted = &started
	}
	if !m.lastSweepFinished.IsZero() {
		finished := m.lastSweepFinished
		snapshot.LastSweepFinished = &finished
	}
	m.mu.Unlock()
	return snapshot
}
