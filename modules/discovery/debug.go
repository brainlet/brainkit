package discovery

// DebugSnapshot is the discovery module lifecycle debug view. It reports
// provider ownership and bus-mode loop/subscription state without contacting
// peers.
type DebugSnapshot struct {
	ProviderAttached bool              `json:"providerAttached"`
	ProviderType     string            `json:"providerType,omitempty"`
	Static           *StaticDebugState `json:"static,omitempty"`
	Bus              *BusDebugState    `json:"bus,omitempty"`
}

// StaticDebugState reports static provider peer state.
type StaticDebugState struct {
	PeerCount int `json:"peerCount"`
}

// BusDebugState reports bus provider lifecycle state.
type BusDebugState struct {
	Registered           bool  `json:"registered"`
	Closing              bool  `json:"closing"`
	Closed               bool  `json:"closed"`
	KnownPeers           int   `json:"knownPeers"`
	SubscriptionAttached bool  `json:"subscriptionAttached"`
	ActiveLoops          int64 `json:"activeLoops"`
	HeartbeatSeconds     int64 `json:"heartbeatSeconds"`
	TTLSeconds           int64 `json:"ttlSeconds"`
}

// DebugSnapshot returns a point-in-time view of module-owned lifecycle state.
func (m *Module) DebugSnapshot() DebugSnapshot {
	if m == nil || m.provider == nil {
		return DebugSnapshot{}
	}
	switch provider := m.provider.(type) {
	case *Bus:
		state := provider.DebugSnapshot()
		return DebugSnapshot{ProviderAttached: true, ProviderType: "bus", Bus: &state}
	case *Static:
		state := provider.DebugSnapshot()
		return DebugSnapshot{ProviderAttached: true, ProviderType: "static", Static: &state}
	default:
		return DebugSnapshot{ProviderAttached: true, ProviderType: "custom"}
	}
}

// DebugSnapshot returns bus-mode lifecycle counters.
func (d *Bus) DebugSnapshot() BusDebugState {
	if d == nil {
		return BusDebugState{}
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return BusDebugState{
		Registered:           d.self != nil,
		Closing:              d.closing,
		Closed:               d.closed,
		KnownPeers:           len(d.peers),
		SubscriptionAttached: d.unsub != nil,
		ActiveLoops:          d.activeLoops.Load(),
		HeartbeatSeconds:     int64(d.heartbeat.Seconds()),
		TTLSeconds:           int64(d.ttl.Seconds()),
	}
}

// DebugSnapshot returns static provider lifecycle counters.
func (d *Static) DebugSnapshot() StaticDebugState {
	if d == nil {
		return StaticDebugState{}
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return StaticDebugState{PeerCount: len(d.peers)}
}
