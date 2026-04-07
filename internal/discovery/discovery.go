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
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`          // Kit namespace for cross-Kit routing
	Address   string            `json:"address"`            // network address (host:port) for future direct connections
	Meta      map[string]string `json:"meta,omitempty"`
}


