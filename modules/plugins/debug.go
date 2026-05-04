package plugins

// DebugSnapshot is the plugins module lifecycle debug view. It reports only
// plugin-owned in-memory state so inspection stays cheap and non-blocking.
type DebugSnapshot struct {
	Closing                      bool                         `json:"closing"`
	ConfiguredPlugins            int                          `json:"configuredPlugins"`
	ManagerStopping              bool                         `json:"managerStopping"`
	RunningProcesses             int                          `json:"runningProcesses"`
	StoppingProcesses            int                          `json:"stoppingProcesses"`
	RestartCount                 int                          `json:"restartCount"`
	RegisteredPlugins            int                          `json:"registeredPlugins"`
	RegisteredTools              int                          `json:"registeredTools"`
	PluginToolOwners             int                          `json:"pluginToolOwners"`
	PluginToolReferences         int                          `json:"pluginToolReferences"`
	ReplayTimers                 int                          `json:"replayTimers"`
	StoreConfigured              bool                         `json:"storeConfigured"`
	PluginCheckerLeaseAttached   bool                         `json:"pluginCheckerLeaseAttached"`
	PluginRestarterLeaseAttached bool                         `json:"pluginRestarterLeaseAttached"`
	WebSocket                    PluginWebSocketDebugSnapshot `json:"websocket"`
}

// PluginWebSocketDebugSnapshot reports WebSocket control-plane ownership
// counters without exposing connection internals.
type PluginWebSocketDebugSnapshot struct {
	Listening         bool   `json:"listening"`
	Address           string `json:"address,omitempty"`
	Closing           bool   `json:"closing"`
	ServeRunning      bool   `json:"serveRunning"`
	ActiveHandlers    int    `json:"activeHandlers"`
	PingLoops         int    `json:"pingLoops"`
	ActiveConnections int    `json:"activeConnections"`
	PendingToolCalls  int    `json:"pendingToolCalls"`
	Subscriptions     int    `json:"subscriptions"`
	RegisteredTools   int    `json:"registeredTools"`
}

type pluginManagerDebugSnapshot struct {
	managerStopping   bool
	runningProcesses  int
	stoppingProcesses int
	restartCount      int
	websocket         PluginWebSocketDebugSnapshot
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.RLock()
	manager := m.manager
	configuredPlugins := len(m.cfg.Plugins)
	storeConfigured := m.cfg.Store != nil
	closing := m.closing.Load()
	checkerLeaseAttached := m.pluginCheckerLeaseActive.Load()
	restarterLeaseAttached := m.pluginRestarterLeaseActive.Load()
	m.mu.RUnlock()

	managerSnapshot := manager.debugSnapshot()

	m.regMu.Lock()
	registeredPlugins := len(m.registrations)
	m.regMu.Unlock()

	m.toolMu.Lock()
	registeredTools := len(m.toolRefs)
	pluginToolOwners := len(m.pluginToolRefs)
	pluginToolReferences := 0
	for _, refs := range m.pluginToolRefs {
		for _, count := range refs {
			pluginToolReferences += count
		}
	}
	m.toolMu.Unlock()

	m.timerMu.Lock()
	replayTimers := len(m.replayTimers)
	m.timerMu.Unlock()

	return DebugSnapshot{
		Closing:                      closing,
		ConfiguredPlugins:            configuredPlugins,
		ManagerStopping:              managerSnapshot.managerStopping,
		RunningProcesses:             managerSnapshot.runningProcesses,
		StoppingProcesses:            managerSnapshot.stoppingProcesses,
		RestartCount:                 managerSnapshot.restartCount,
		RegisteredPlugins:            registeredPlugins,
		RegisteredTools:              registeredTools,
		PluginToolOwners:             pluginToolOwners,
		PluginToolReferences:         pluginToolReferences,
		ReplayTimers:                 replayTimers,
		StoreConfigured:              storeConfigured,
		PluginCheckerLeaseAttached:   checkerLeaseAttached,
		PluginRestarterLeaseAttached: restarterLeaseAttached,
		WebSocket:                    managerSnapshot.websocket,
	}
}

func (pm *pluginManager) debugSnapshot() pluginManagerDebugSnapshot {
	if pm == nil {
		return pluginManagerDebugSnapshot{}
	}
	pm.mu.Lock()
	managerStopping := pm.stopping
	plugins := make([]*pluginConn, 0, len(pm.plugins))
	for _, pc := range pm.plugins {
		plugins = append(plugins, pc)
	}
	ws := pm.wsServer
	pm.mu.Unlock()

	snapshot := pluginManagerDebugSnapshot{
		managerStopping:  managerStopping,
		runningProcesses: len(plugins),
	}
	for _, pc := range plugins {
		if pc == nil {
			continue
		}
		pc.mu.Lock()
		if pc.stopping {
			snapshot.stoppingProcesses++
		}
		snapshot.restartCount += pc.restarts
		pc.mu.Unlock()
	}
	snapshot.websocket = ws.debugSnapshot()
	return snapshot
}

func (s *pluginWSServer) debugSnapshot() PluginWebSocketDebugSnapshot {
	if s == nil {
		return PluginWebSocketDebugSnapshot{}
	}
	s.mu.Lock()
	conns := make([]*pluginWSConn, 0, len(s.conns))
	for _, pc := range s.conns {
		conns = append(conns, pc)
	}
	listener := s.listener
	closing := s.closing
	serveDone := s.serveDone
	closingSubs := len(s.closingSubs)
	s.mu.Unlock()

	snapshot := PluginWebSocketDebugSnapshot{
		Closing:        closing,
		ActiveHandlers: int(s.activeHandlers.Load()),
		PingLoops:      int(s.activePingLoops.Load()),
	}
	if listener != nil {
		snapshot.Listening = true
		snapshot.Address = listener.Addr().String()
	}
	if serveDone != nil {
		select {
		case <-serveDone:
		default:
			snapshot.ServeRunning = true
		}
	}
	for _, pc := range conns {
		if pc == nil {
			continue
		}
		pc.mu.Lock()
		if !pc.closed {
			snapshot.ActiveConnections++
		}
		snapshot.PendingToolCalls += len(pc.pending)
		snapshot.Subscriptions += len(pc.subs)
		snapshot.RegisteredTools += len(pc.tools)
		pc.mu.Unlock()
	}
	snapshot.Subscriptions += closingSubs
	return snapshot
}
