package discovery

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/sdk"
)

const presenceTopic = "_brainkit.presence"

// BusConfig configures the bus discovery provider.
type BusConfig struct {
	Transport PresenceTransport
	Heartbeat time.Duration // default 10s
	TTL       time.Duration // default 30s
}

// Bus discovers peers via announcements on the transport bus.
// All Kits on the same transport cluster see each other — cross-namespace.
type Bus struct {
	mu    syncx.RWMutex
	self  *Peer
	peers map[string]peerEntry

	transport PresenceTransport
	unsub     func()
	cancel    context.CancelFunc

	heartbeat time.Duration
	ttl       time.Duration
}

type peerEntry struct {
	Peer
	LastSeen time.Time
}

type presenceMessage struct {
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	RuntimeID string            `json:"runtimeID,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
	StartedAt string            `json:"startedAt,omitempty"`
}

func NewBus(cfg BusConfig) *Bus {
	heartbeat := cfg.Heartbeat
	if heartbeat <= 0 {
		heartbeat = 10 * time.Second
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Bus{
		peers:     make(map[string]peerEntry),
		transport: cfg.Transport,
		heartbeat: heartbeat,
		ttl:       ttl,
	}
}

func (d *Bus) Register(self Peer) error {
	d.mu.Lock()
	d.self = &self
	d.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	// Subscribe to presence announcements (fan-out — all instances receive)
	unsub, err := d.transport.SubscribeRawFanOutGlobal(ctx, presenceTopic, func(payload json.RawMessage) {
		d.handleMessage(payload)
	})
	if err != nil {
		cancel()
		return err
	}
	d.unsub = unsub

	// Initial announce
	d.announce()

	// Heartbeat goroutine
	go func() {
		ticker := time.NewTicker(d.heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.announce()
			}
		}
	}()

	// Eviction goroutine
	go func() {
		ticker := time.NewTicker(d.heartbeat) // check at heartbeat interval
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.evict()
			}
		}
	}()

	return nil
}

func (d *Bus) Resolve(name string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entry, ok := d.peers[name]
	if !ok {
		return "", &sdk.NotFoundError{Resource: "peer", Name: name}
	}
	return entry.Namespace, nil
}

func (d *Bus) Browse() ([]Peer, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]Peer, 0, len(d.peers))
	for _, entry := range d.peers {
		result = append(result, entry.Peer)
	}
	return result, nil
}

func (d *Bus) BrowseNamespaces() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	seen := make(map[string]bool)
	for _, entry := range d.peers {
		if entry.Namespace != "" {
			seen[entry.Namespace] = true
		}
	}
	result := make([]string, 0, len(seen))
	for ns := range seen {
		result = append(result, ns)
	}
	return result, nil
}

func (d *Bus) Close() error {
	// 1. Best-effort leave announcement (before cancel so no heartbeat races re-announce)
	d.mu.RLock()
	self := d.self
	d.mu.RUnlock()
	if self != nil {
		msg, _ := json.Marshal(presenceMessage{Type: "leave", Name: self.Name})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		d.transport.PublishRawGlobal(ctx, presenceTopic, msg)
		cancel()
	}
	// 2. Cancel heartbeat + eviction goroutines
	if d.cancel != nil {
		d.cancel()
	}
	// 3. Unsubscribe
	if d.unsub != nil {
		d.unsub()
	}
	return nil
}

func (d *Bus) announce() {
	d.mu.RLock()
	self := d.self
	d.mu.RUnlock()
	if self == nil {
		return
	}
	msg, _ := json.Marshal(presenceMessage{
		Type:      "announce",
		Name:      self.Name,
		Namespace: self.Namespace,
		Meta:      self.Meta,
	})
	d.transport.PublishRawGlobal(context.Background(), presenceTopic, msg)
}

func (d *Bus) handleMessage(payload json.RawMessage) {
	var msg presenceMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	d.mu.RLock()
	selfName := ""
	if d.self != nil {
		selfName = d.self.Name
	}
	d.mu.RUnlock()

	switch msg.Type {
	case "announce":
		if msg.Name == selfName {
			return // skip self
		}
		d.mu.Lock()
		d.peers[msg.Name] = peerEntry{
			Peer: Peer{
				Name:      msg.Name,
				Namespace: msg.Namespace,
				Meta:      msg.Meta,
			},
			LastSeen: time.Now(),
		}
		d.mu.Unlock()

	case "leave":
		if msg.Name == selfName {
			return
		}
		d.mu.Lock()
		delete(d.peers, msg.Name)
		d.mu.Unlock()
	}
}

func (d *Bus) evict() {
	cutoff := time.Now().Add(-d.ttl)
	d.mu.Lock()
	for name, entry := range d.peers {
		if entry.LastSeen.Before(cutoff) {
			delete(d.peers, name)
		}
	}
	d.mu.Unlock()
}
