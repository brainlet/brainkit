package tracing

import "fmt"

// DebugSnapshot is the tracing module lifecycle debug view. It reports only
// module-owned store/lease state and does not query trace rows.
type DebugSnapshot struct {
	Closing                 bool                      `json:"closing"`
	StoreConfigured         bool                      `json:"storeConfigured"`
	StoreAttached           bool                      `json:"storeAttached"`
	StoreClosing            bool                      `json:"storeClosing"`
	StoreType               string                    `json:"storeType,omitempty"`
	StoreCloseable          bool                      `json:"storeCloseable"`
	TraceStoreLeaseAttached bool                      `json:"traceStoreLeaseAttached"`
	SQLite                  *SQLiteTraceDebugSnapshot `json:"sqlite,omitempty"`
}

// SQLiteTraceDebugSnapshot reports static state owned by SQLiteTraceStore.
type SQLiteTraceDebugSnapshot struct {
	RetentionEnabled bool  `json:"retentionEnabled"`
	RetentionSeconds int64 `json:"retentionSeconds,omitempty"`
	CleanupEnabled   bool  `json:"cleanupEnabled"`
	CleanupRunning   bool  `json:"cleanupRunning"`
	Closing          bool  `json:"closing"`
	Closed           bool  `json:"closed"`
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	store := m.store
	snapshot := DebugSnapshot{
		Closing:                 m.closing.Load(),
		StoreConfigured:         m.cfg.Store != nil,
		StoreAttached:           store != nil,
		StoreClosing:            m.storeClosing.Load() || m.storeCloseJob.Running(),
		TraceStoreLeaseAttached: m.traceStoreLeaseActive.Load(),
	}
	if store == nil {
		return snapshot
	}
	snapshot.StoreType = fmt.Sprintf("%T", store)
	_, snapshot.StoreCloseable = store.(interface{ Close() error })
	if sqliteStore, ok := store.(*SQLiteTraceStore); ok {
		snapshot.SQLite = sqliteStore.debugSnapshot()
	}
	return snapshot
}

func (s *SQLiteTraceStore) debugSnapshot() *SQLiteTraceDebugSnapshot {
	if s == nil {
		return nil
	}
	s.closeMu.Lock()
	closing := s.closing
	closed := s.closed
	s.closeMu.Unlock()
	seconds := int64(0)
	if s.retention > 0 {
		seconds = int64(s.retention.Seconds())
	}
	return &SQLiteTraceDebugSnapshot{
		RetentionEnabled: s.retention > 0,
		RetentionSeconds: seconds,
		CleanupEnabled:   s.retention > 0,
		CleanupRunning:   s.cleanupRunning.Load(),
		Closing:          closing,
		Closed:           closed,
	}
}
