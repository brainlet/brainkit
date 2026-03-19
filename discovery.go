package brainkit

// Discovery resolves peer addresses for Kit-to-Kit networking.
type Discovery interface {
	Resolve(name string) (addr string, err error)
	Browse() ([]Peer, error)
	Register(self Peer) error
	Close() error
}

// Peer represents a discoverable Kit instance.
type Peer struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// DiscoveryConfig configures the discovery mechanism.
type DiscoveryConfig struct {
	Type        string // "static", "multicast", or "" (none)
	ServiceName string // multicast service name (default: "_brainkit._tcp")
}
