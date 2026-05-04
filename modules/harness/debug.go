package harness

// DebugSnapshot is the harness module lifecycle debug view. It reports only
// Go-owned harness state and never calls into the JS runtime.
type DebugSnapshot struct {
	ConfiguredHeartbeats    int   `json:"configuredHeartbeats"`
	InstanceAttached        bool  `json:"instanceAttached"`
	Initialized             bool  `json:"initialized"`
	Closing                 bool  `json:"closing"`
	Closed                  bool  `json:"closed"`
	HeartbeatTimers         int   `json:"heartbeatTimers"`
	ActiveHeartbeatWorkers  int64 `json:"activeHeartbeatWorkers"`
	SubscriberCount         int   `json:"subscriberCount"`
	DisplayStateInitialized bool  `json:"displayStateInitialized"`
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.RLock()
	instance := m.instance
	m.mu.RUnlock()
	snapshot := DebugSnapshot{
		ConfiguredHeartbeats: len(m.cfg.Harness.HeartbeatHandlers),
		InstanceAttached:     instance != nil,
	}
	if instance == nil {
		return snapshot
	}
	harnessSnapshot := instance.debugSnapshot()
	harnessSnapshot.ConfiguredHeartbeats = len(m.cfg.Harness.HeartbeatHandlers)
	harnessSnapshot.InstanceAttached = true
	return harnessSnapshot
}

func (h *Harness) debugSnapshot() DebugSnapshot {
	if h == nil {
		return DebugSnapshot{}
	}
	h.hbMu.Lock()
	heartbeatTimers := len(h.heartbeats)
	h.hbMu.Unlock()

	h.subMu.RLock()
	subscriberCount := len(h.subscribers)
	h.subMu.RUnlock()

	return DebugSnapshot{
		Initialized:             h.initialized,
		Closing:                 h.closing.Load(),
		Closed:                  h.closed.Load(),
		HeartbeatTimers:         heartbeatTimers,
		ActiveHeartbeatWorkers:  h.activeHB.Load(),
		SubscriberCount:         subscriberCount,
		DisplayStateInitialized: h.displayState != nil,
	}
}
