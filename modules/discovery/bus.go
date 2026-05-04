package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk"
)

const presenceTopic = "_brainkit.presence"

// BusConfig configures the bus discovery provider.
type BusConfig struct {
	Transport transport.Presence
	Heartbeat time.Duration // default 10s
	TTL       time.Duration // default 30s
}

// Bus discovers peers via announcements on the transport bus.
// All Kits on the same transport cluster see each other — cross-namespace.
type Bus struct {
	mu      syncx.RWMutex
	self    *Peer
	peers   map[string]peerEntry
	closing bool
	closed  bool

	transport transport.Presence
	unsub     func()
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeMu   sync.Mutex
	closeWait sync.Once
	closeDone chan struct{}

	heartbeat   time.Duration
	ttl         time.Duration
	activeLoops atomic.Int64
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
		closeDone: make(chan struct{}),
	}
}

func (d *Bus) Register(self Peer) error {
	d.closeMu.Lock()
	defer d.closeMu.Unlock()

	d.mu.Lock()
	if d.closing || (!d.closed && (d.cancel != nil || d.unsub != nil)) {
		d.mu.Unlock()
		return fmt.Errorf("discovery bus already registered")
	}
	d.self = &self
	d.closing = false
	d.closed = false
	d.closeWait = sync.Once{}
	d.closeDone = make(chan struct{})
	d.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	// Subscribe to presence announcements (fan-out — all instances receive)
	unsub, err := d.transport.SubscribeRawFanOutGlobal(ctx, presenceTopic, func(payload json.RawMessage) {
		d.handleMessage(payload)
	})
	if err != nil {
		cancel()
		return err
	}
	d.mu.Lock()
	d.cancel = cancel
	d.unsub = unsub
	d.mu.Unlock()

	// Initial announce
	d.announce()

	// Heartbeat goroutine
	d.wg.Add(1)
	d.activeLoops.Add(1)
	go func() {
		defer func() {
			d.activeLoops.Add(-1)
			d.wg.Done()
		}()
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
	d.wg.Add(1)
	d.activeLoops.Add(1)
	go func() {
		defer func() {
			d.activeLoops.Add(-1)
			d.wg.Done()
		}()
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
	return d.CloseContext(context.Background())
}

func (d *Bus) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	d.closeMu.Lock()
	defer d.closeMu.Unlock()

	d.mu.Lock()
	if d.closed && d.cancel == nil && d.unsub == nil && d.activeLoops.Load() == 0 {
		d.mu.Unlock()
		return nil
	}
	self := d.self
	cancel := d.cancel
	unsub := d.unsub
	closeDone := d.closeDone
	d.closing = true
	d.closed = false
	d.closeWait.Do(func() {
		go func() {
			d.wg.Wait()
			close(closeDone)
		}()
	})
	d.mu.Unlock()

	// Stop heartbeat + eviction goroutines before publishing leave. announce()
	// also checks d.closed so a selected heartbeat cannot re-announce after
	// close starts.
	if cancel != nil {
		cancel()
	}
	select {
	case <-closeDone:
	case <-ctx.Done():
		d.mu.Lock()
		d.closing = false
		d.mu.Unlock()
		return ctx.Err()
	}
	if self != nil {
		msg, _ := json.Marshal(presenceMessage{Type: "leave", Name: self.Name})
		leaveCtx, leaveCancel := context.WithTimeout(ctx, 2*time.Second)
		_ = d.transport.PublishRawGlobal(leaveCtx, presenceTopic, msg)
		leaveCancel()
	}
	if unsub != nil {
		unsub()
	}
	d.mu.Lock()
	d.cancel = nil
	d.unsub = nil
	d.closed = true
	d.closing = false
	d.mu.Unlock()
	return nil
}

func (d *Bus) announce() {
	d.mu.RLock()
	self := d.self
	closed := d.closed || d.closing
	d.mu.RUnlock()
	if self == nil || closed {
		return
	}
	msg, _ := json.Marshal(presenceMessage{
		Type:      "announce",
		Name:      self.Name,
		Namespace: self.Namespace,
		Meta:      self.Meta,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_ = d.transport.PublishRawGlobal(ctx, presenceTopic, msg)
	cancel()
}

func (d *Bus) handleMessage(payload json.RawMessage) {
	var msg presenceMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	d.mu.RLock()
	closed := d.closed || d.closing
	selfName := ""
	if d.self != nil {
		selfName = d.self.Name
	}
	d.mu.RUnlock()
	if closed {
		return
	}

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
