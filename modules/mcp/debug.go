package mcp

import "sort"

// DebugSnapshot is the MCP module lifecycle debug view. It reports configured
// server counts, live client counts, and cached tool counts without calling
// any remote MCP server.
type DebugSnapshot struct {
	ConfiguredServers int      `json:"configuredServers"`
	Closing           bool     `json:"closing"`
	ManagerAttached   bool     `json:"managerAttached"`
	ConnectedServers  int      `json:"connectedServers"`
	ClosingClients    int      `json:"closingClients"`
	CachedTools       int      `json:"cachedTools"`
	Servers           []string `json:"servers,omitempty"`
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	snapshot := DebugSnapshot{ConfiguredServers: len(m.servers), Closing: m.closing.Load()}
	manager := m.currentManager()
	if manager == nil {
		return snapshot
	}
	managerSnapshot := manager.DebugSnapshot()
	managerSnapshot.ConfiguredServers = len(m.servers)
	managerSnapshot.Closing = snapshot.Closing || managerSnapshot.Closing
	managerSnapshot.ManagerAttached = true
	return managerSnapshot
}

// DebugSnapshot returns manager-owned connection and tool counters.
func (m *MCPManager) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	servers := make([]string, 0, len(m.clients))
	toolCount := 0
	closingClients := 0
	for name, entry := range m.clients {
		servers = append(servers, name)
		if entry != nil && entry.closing {
			closingClients++
		}
	}
	for _, tools := range m.tools {
		toolCount += len(tools)
	}
	sort.Strings(servers)
	return DebugSnapshot{
		ManagerAttached:  true,
		Closing:          m.closing,
		ConnectedServers: len(m.clients),
		ClosingClients:   closingClients,
		CachedTools:      toolCount,
		Servers:          servers,
	}
}
