package schedules

// DebugSnapshot is the schedules module lifecycle debug view. It reports only
// module-owned in-memory state so inspection stays cheap and non-blocking.
type DebugSnapshot struct {
	ConfiguredSchedules          int  `json:"configuredSchedules"`
	ActiveTimers                 int  `json:"activeTimers"`
	ActiveFires                  int  `json:"activeFires"`
	OneTimeSchedules             int  `json:"oneTimeSchedules"`
	RepeatingSchedules           int  `json:"repeatingSchedules"`
	Closing                      bool `json:"closing"`
	Closed                       bool `json:"closed"`
	StoreConfigured              bool `json:"storeConfigured"`
	StoreClosing                 bool `json:"storeClosing"`
	ScheduleHandlerLeaseAttached bool `json:"scheduleHandlerLeaseAttached"`
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.RLock()
	scheduler := m.scheduler
	storeConfigured := m.cfg.Store != nil
	storeClosing := m.storeClosing.Load() || m.storeCloseJob.Running()
	leaseAttached := m.scheduleHandlerLeaseActive.Load()
	moduleClosing := m.closing.Load()
	m.mu.RUnlock()

	schedulerSnapshot := scheduler.debugSnapshot()
	return DebugSnapshot{
		ConfiguredSchedules:          schedulerSnapshot.configuredSchedules,
		ActiveTimers:                 schedulerSnapshot.activeTimers,
		ActiveFires:                  schedulerSnapshot.activeFires,
		OneTimeSchedules:             schedulerSnapshot.oneTimeSchedules,
		RepeatingSchedules:           schedulerSnapshot.repeatingSchedules,
		Closing:                      moduleClosing || schedulerSnapshot.closing,
		Closed:                       schedulerSnapshot.closed,
		StoreConfigured:              storeConfigured,
		StoreClosing:                 storeClosing,
		ScheduleHandlerLeaseAttached: leaseAttached,
	}
}

type schedulerDebugSnapshot struct {
	configuredSchedules int
	activeTimers        int
	activeFires         int
	oneTimeSchedules    int
	repeatingSchedules  int
	closing             bool
	closed              bool
}

func (s *Scheduler) debugSnapshot() schedulerDebugSnapshot {
	if s == nil {
		return schedulerDebugSnapshot{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := schedulerDebugSnapshot{
		configuredSchedules: len(s.schedules),
		activeFires:         int(s.activeFires.Load()),
		closing:             s.closing,
		closed:              s.closed,
	}
	for _, entry := range s.schedules {
		if entry == nil {
			continue
		}
		if entry.timer != nil {
			snapshot.activeTimers++
		}
		if entry.OneTime {
			snapshot.oneTimeSchedules++
		} else {
			snapshot.repeatingSchedules++
		}
	}
	return snapshot
}
