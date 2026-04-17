package discovery

import "time"

// Provider resolves peer addresses for Kit-to-Kit networking.
type Provider interface {
	Resolve(name string) (namespace string, err error)
	Browse() ([]Peer, error)
	BrowseNamespaces() ([]string, error)
	Register(self Peer) error
	Close() error
}

// Peer represents a discoverable Kit instance.
type Peer struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Address   string            `json:"address"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// PeerConfig configures a known peer for static discovery.
type PeerConfig struct {
	Name      string
	Namespace string
	Address   string
	Meta      map[string]string
}

// ModuleConfig configures the discovery brainkit.Module. Pass to NewModule.
// Type:
//   - "static": peers are fixed at boot, taken from StaticPeers.
//   - "bus":    peers are learned from presence announcements on the transport.
//   - "":       disabled (NewModule returns a no-op Module).
type ModuleConfig struct {
	Type        string
	StaticPeers []PeerConfig

	// Bus-mode tunables.
	Heartbeat time.Duration // default 10s
	TTL       time.Duration // default 30s

	// Name overrides the self-peer name. Empty = a per-instance UUID, so
	// replicas with the same CallerID remain distinguishable on the bus.
	Name string
}
