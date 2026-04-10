package discovery

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport is a simple in-memory PresenceTransport for unit tests.
type mockTransport struct {
	mu       sync.Mutex
	handlers []func(payload json.RawMessage)
}

func newMockTransport() *mockTransport {
	return &mockTransport{}
}

func (m *mockTransport) PublishRawGlobal(_ context.Context, topic string, payload json.RawMessage) error {
	m.mu.Lock()
	handlers := make([]func(json.RawMessage), len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.Unlock()
	for _, h := range handlers {
		h(payload)
	}
	return nil
}

func (m *mockTransport) SubscribeRawFanOutGlobal(_ context.Context, topic string, handler func(payload json.RawMessage)) (func(), error) {
	m.mu.Lock()
	m.handlers = append(m.handlers, handler)
	m.mu.Unlock()
	return func() {}, nil
}

func TestBus_AnnounceAndDiscover(t *testing.T) {
	tr := newMockTransport()

	d1 := NewBus(BusConfig{Transport: tr, Heartbeat: 100 * time.Millisecond, TTL: 1 * time.Second})
	d2 := NewBus(BusConfig{Transport: tr, Heartbeat: 100 * time.Millisecond, TTL: 1 * time.Second})

	require.NoError(t, d1.Register(Peer{Name: "kit-1", Namespace: "agents"}))
	require.NoError(t, d2.Register(Peer{Name: "kit-2", Namespace: "workers"}))

	// Wait for one heartbeat cycle
	time.Sleep(200 * time.Millisecond)

	peers1, _ := d1.Browse()
	assert.Len(t, peers1, 1)
	assert.Equal(t, "kit-2", peers1[0].Name)
	assert.Equal(t, "workers", peers1[0].Namespace)

	peers2, _ := d2.Browse()
	assert.Len(t, peers2, 1)
	assert.Equal(t, "kit-1", peers2[0].Name)

	d1.Close()
	d2.Close()
}

func TestBus_SkipSelf(t *testing.T) {
	tr := newMockTransport()
	d := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})
	require.NoError(t, d.Register(Peer{Name: "self-kit", Namespace: "ns"}))

	time.Sleep(150 * time.Millisecond)

	peers, _ := d.Browse()
	assert.Len(t, peers, 0, "should not discover self")
	d.Close()
}

func TestBus_TTLEviction(t *testing.T) {
	tr := newMockTransport()
	d1 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 200 * time.Millisecond})
	d2 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 200 * time.Millisecond})

	require.NoError(t, d1.Register(Peer{Name: "kit-1", Namespace: "ns"}))
	require.NoError(t, d2.Register(Peer{Name: "kit-2", Namespace: "ns"}))

	time.Sleep(100 * time.Millisecond)
	peers, _ := d1.Browse()
	assert.Len(t, peers, 1, "should see kit-2")

	// Close d2 without graceful leave — just stop heartbeating
	d2.cancel() // stop goroutines but don't publish leave

	// Wait for TTL to expire
	time.Sleep(400 * time.Millisecond)

	peers, _ = d1.Browse()
	assert.Len(t, peers, 0, "kit-2 should be evicted after TTL")
	d1.Close()
}

func TestBus_GracefulLeave(t *testing.T) {
	tr := newMockTransport()
	d1 := NewBus(BusConfig{Transport: tr, Heartbeat: 100 * time.Millisecond, TTL: 5 * time.Second})
	d2 := NewBus(BusConfig{Transport: tr, Heartbeat: 100 * time.Millisecond, TTL: 5 * time.Second})

	require.NoError(t, d1.Register(Peer{Name: "kit-1", Namespace: "ns"}))
	require.NoError(t, d2.Register(Peer{Name: "kit-2", Namespace: "ns"}))
	time.Sleep(200 * time.Millisecond)

	peers, _ := d1.Browse()
	assert.Len(t, peers, 1)

	// Graceful close — d2 publishes leave
	d2.Close()

	// d1 should see d2 gone immediately (leave message, not TTL)
	time.Sleep(50 * time.Millisecond)
	peers, _ = d1.Browse()
	assert.Len(t, peers, 0, "kit-2 should be removed by leave message")
	d1.Close()
}

func TestBus_ResolveReturnsNamespace(t *testing.T) {
	tr := newMockTransport()
	d1 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})
	d2 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})

	require.NoError(t, d1.Register(Peer{Name: "kit-1", Namespace: "ns-1"}))
	require.NoError(t, d2.Register(Peer{Name: "kit-2", Namespace: "ns-2"}))
	time.Sleep(100 * time.Millisecond)

	ns, err := d1.Resolve("kit-2")
	require.NoError(t, err)
	assert.Equal(t, "ns-2", ns)

	_, err = d1.Resolve("nonexistent")
	assert.Error(t, err)

	d1.Close()
	d2.Close()
}

func TestBus_BrowseNamespacesDedup(t *testing.T) {
	tr := newMockTransport()
	d1 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})
	d2 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})
	d3 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})
	d4 := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})

	// 3 nodes, 2 namespaces: agents (2 replicas) + gateway (1)
	require.NoError(t, d1.Register(Peer{Name: "observer", Namespace: "observer"}))
	require.NoError(t, d2.Register(Peer{Name: "agents-1", Namespace: "agents"}))
	require.NoError(t, d3.Register(Peer{Name: "agents-2", Namespace: "agents"}))
	require.NoError(t, d4.Register(Peer{Name: "gateway-1", Namespace: "gateway"}))
	time.Sleep(150 * time.Millisecond)

	namespaces, err := d1.BrowseNamespaces()
	require.NoError(t, err)
	assert.Len(t, namespaces, 2, "should see agents + gateway (not observer = self, not duplicate agents)")

	nsMap := map[string]bool{}
	for _, ns := range namespaces {
		nsMap[ns] = true
	}
	assert.True(t, nsMap["agents"])
	assert.True(t, nsMap["gateway"])

	d1.Close()
	d2.Close()
	d3.Close()
	d4.Close()
}

func TestBus_ConcurrentBrowse(t *testing.T) {
	tr := newMockTransport()
	d := NewBus(BusConfig{Transport: tr, Heartbeat: 50 * time.Millisecond, TTL: 1 * time.Second})
	require.NoError(t, d.Register(Peer{Name: "self", Namespace: "ns"}))

	// Simulate external announcements
	go func() {
		for i := 0; i < 100; i++ {
			msg, _ := json.Marshal(presenceMessage{Type: "announce", Name: "peer-x", Namespace: "ns-x"})
			tr.PublishRawGlobal(context.Background(), presenceTopic, msg)
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				d.Browse()
				d.BrowseNamespaces()
				d.Resolve("peer-x")
			}
		}()
	}
	wg.Wait()
	d.Close()
}
