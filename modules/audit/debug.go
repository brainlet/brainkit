package audit

import "fmt"

// DebugSnapshot is the audit module lifecycle debug view. It reports only
// module-owned store/lease state and does not query audit rows.
type DebugSnapshot struct {
	Closing                bool   `json:"closing"`
	StoreConfigured        bool   `json:"storeConfigured"`
	StoreAttached          bool   `json:"storeAttached"`
	StoreClosing           bool   `json:"storeClosing"`
	StoreType              string `json:"storeType,omitempty"`
	StoreCloseable         bool   `json:"storeCloseable"`
	OwnStore               bool   `json:"ownStore"`
	VerboseConfigured      bool   `json:"verboseConfigured"`
	StoreLeaseAttached     bool   `json:"storeLeaseAttached"`
	VerbosityLeaseAttached bool   `json:"verbosityLeaseAttached"`
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	store := m.cfg.Store
	snapshot := DebugSnapshot{
		Closing:                m.closing.Load(),
		StoreConfigured:        store != nil,
		StoreAttached:          m.storeAttached.Load(),
		StoreClosing:           m.storeClosing.Load() || m.storeCloseJob.Running(),
		OwnStore:               m.cfg.OwnStore,
		VerboseConfigured:      m.cfg.Verbose,
		StoreLeaseAttached:     m.storeAttached.Load(),
		VerbosityLeaseAttached: m.verboseAttached.Load(),
	}
	if store == nil {
		return snapshot
	}
	snapshot.StoreType = fmt.Sprintf("%T", store)
	_, snapshot.StoreCloseable = store.(interface{ Close() error })
	return snapshot
}
