package discovery

// Provider resolves peer addresses for Kit-to-Kit networking.
type Provider interface {
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

// Config configures the discovery mechanism.
type Config struct {
	Type        string // "static", "multicast", or "" (none)
	ServiceName string // multicast service name (default: "_brainkit._tcp")
}
