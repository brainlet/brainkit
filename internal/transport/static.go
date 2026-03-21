package transport

import (
	"fmt"
	"sync"
)

// StaticDiscovery resolves peers from a fixed configuration map.
type StaticDiscovery struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

func NewStaticDiscovery(peers map[string]string) *StaticDiscovery {
	sd := &StaticDiscovery{
		peers: make(map[string]Peer),
	}
	for name, addr := range peers {
		sd.peers[name] = Peer{Name: name, Address: addr}
	}
	return sd
}

func (d *StaticDiscovery) Resolve(name string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	peer, ok := d.peers[name]
	if !ok {
		return "", fmt.Errorf("discovery: peer %q not found", name)
	}
	return peer.Address, nil
}

func (d *StaticDiscovery) Browse() ([]Peer, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]Peer, 0, len(d.peers))
	for _, peer := range d.peers {
		result = append(result, peer)
	}
	return result, nil
}

func (d *StaticDiscovery) Register(self Peer) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.peers[self.Name] = self
	return nil
}

func (d *StaticDiscovery) Close() error {
	return nil
}
