package discovery

import (
	"fmt"
	"sync"
)

// Static resolves peers from a fixed configuration map.
type Static struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

func NewStatic(peers map[string]string) *Static {
	sd := &Static{
		peers: make(map[string]Peer),
	}
	for name, addr := range peers {
		sd.peers[name] = Peer{Name: name, Address: addr}
	}
	return sd
}

func (d *Static) Resolve(name string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	peer, ok := d.peers[name]
	if !ok {
		return "", fmt.Errorf("discovery: peer %q not found", name)
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
