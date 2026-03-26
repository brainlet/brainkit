package discovery

import (
	"sync"

	"github.com/brainlet/brainkit/sdk"
)

// Static resolves peers from a fixed configuration.
type Static struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

// NewStatic creates a provider from a map of name→address (legacy).
func NewStatic(peers map[string]string) *Static {
	sd := &Static{peers: make(map[string]Peer)}
	for name, addr := range peers {
		sd.peers[name] = Peer{Name: name, Address: addr}
	}
	return sd
}

// NewStaticFromConfig creates a provider from PeerConfig entries.
func NewStaticFromConfig(configs []PeerConfig) *Static {
	sd := &Static{peers: make(map[string]Peer)}
	for _, cfg := range configs {
		sd.peers[cfg.Name] = Peer{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
			Address:   cfg.Address,
			Meta:      cfg.Meta,
		}
	}
	return sd
}

func (d *Static) Resolve(name string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	peer, ok := d.peers[name]
	if !ok {
		return "", &sdk.NotFoundError{Resource: "peer", Name: name}
	}
	// Return namespace if set, otherwise address (backward compat)
	if peer.Namespace != "" {
		return peer.Namespace, nil
	}
	return peer.Address, nil
}

func (d *Static) Browse() ([]Peer, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]Peer, 0, len(d.peers))
	for _, peer := range d.peers {
		result = append(result, peer)
	}
	return result, nil
}

func (d *Static) Register(self Peer) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.peers[self.Name] = self
	return nil
}

func (d *Static) Close() error {
	return nil
}
