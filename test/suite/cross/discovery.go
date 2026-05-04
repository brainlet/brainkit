package cross

import (
	"context"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/discovery"
	"github.com/brainlet/brainkit/modules/topology"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/brainlet/brainkit/transports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// busDiscoveryModules pairs a bus-discovery module with a topology
// module that reads from it. Every peer-ing test uses this pair so
// tests actually exercise the topology bus surface end-to-end.
func busDiscoveryModules(heartbeat, ttl time.Duration) []bkmodule.Module {
	d := discovery.NewModule(discovery.ModuleConfig{
		Type:      "bus",
		Heartbeat: heartbeat,
		TTL:       ttl,
	})
	return []bkmodule.Module{d, topology.NewModule(topology.Config{Discovery: d})}
}

// --- Discovery tests (from test/adversarial/discovery_test.go + test/infra/discovery_test.go) ---

func testDiscoveryStaticPeers(t *testing.T, _ *suite.TestEnv) {
	provider := discovery.NewStaticFromConfig([]discovery.PeerConfig{
		{Name: "kit-a", Namespace: "ns-a", Address: "localhost:4222"},
		{Name: "kit-b", Namespace: "ns-b", Address: "localhost:4223"},
	})

	ns, err := provider.Resolve("kit-a")
	require.NoError(t, err)
	assert.Equal(t, "ns-a", ns)

	ns2, err := provider.Resolve("kit-b")
	require.NoError(t, err)
	assert.Equal(t, "ns-b", ns2)

	_, err = provider.Resolve("ghost")
	assert.Error(t, err)
}

func testDiscoveryBrowse(t *testing.T, _ *suite.TestEnv) {
	provider := discovery.NewStaticFromConfig([]discovery.PeerConfig{
		{Name: "a", Namespace: "ns-a"},
		{Name: "b", Namespace: "ns-b"},
		{Name: "c", Namespace: "ns-c"},
	})

	peers, err := provider.Browse()
	require.NoError(t, err)
	assert.Len(t, peers, 3)

	names := make(map[string]bool)
	for _, p := range peers {
		names[p.Name] = true
	}
	assert.True(t, names["a"])
	assert.True(t, names["b"])
	assert.True(t, names["c"])
}

func testDiscoveryRegister(t *testing.T, _ *suite.TestEnv) {
	provider := discovery.NewStaticFromConfig(nil)

	err := provider.Register(discovery.Peer{Name: "self", Namespace: "my-ns", Address: "localhost:5000"})
	require.NoError(t, err)

	peers, _ := provider.Browse()
	assert.Len(t, peers, 1)
	assert.Equal(t, "self", peers[0].Name)
}

func testDiscoveryResolveNonexistent(t *testing.T, _ *suite.TestEnv) {
	provider := discovery.NewStaticFromConfig(nil)

	_, err := provider.Resolve("nobody")
	assert.Error(t, err)
}

func testDiscoveryClose(t *testing.T, _ *suite.TestEnv) {
	provider := discovery.NewStaticFromConfig([]discovery.PeerConfig{
		{Name: "x", Namespace: "ns-x"},
	})

	err := provider.Close()
	assert.NoError(t, err)
}

// testDiscoveryStaticPeersBus — bus-level discovery (from test/infra/discovery_test.go)
// testDiscoveryBusPeers — two Kits with different namespaces discover each other via bus.
// Requires Podman (shared NATS for cross-namespace communication).
func testDiscoveryBusPeers(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)

	// Shared NATS container — both Kits must be on the SAME transport
	natsURL := startNATSContainer(t)

	kit1, err := brainkit.New(brainkit.Config{
		Namespace: "disc-agents-cross",
		CallerID:  "host",
		Transport: transports.NATS(natsURL),
		Modules:   busDiscoveryModules(1*time.Second, 5*time.Second),
	})
	require.NoError(t, err)
	defer kit1.Close()

	kit2, err := brainkit.New(brainkit.Config{
		Namespace: "disc-workers-cross",
		CallerID:  "host",
		Transport: transports.NATS(natsURL),
		Modules:   busDiscoveryModules(1*time.Second, 5*time.Second),
	})
	require.NoError(t, err)
	defer kit2.Close()

	// Wait for convergence (2 heartbeat cycles)
	time.Sleep(3 * time.Second)

	// peers.list from kit1 should see kit2
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := topology.CallPeersList(kit1, ctx, topology.PeersListMsg{})
	require.NoError(t, err)
	found := false
	for _, p := range resp.Peers {
		if p.Namespace == "disc-workers-cross" {
			found = true
		}
	}
	assert.True(t, found, "kit1 should discover kit2's namespace")
	assert.Contains(t, resp.Namespaces, "disc-workers-cross", "namespaces should include kit2")
}

// testDiscoveryBusLeave — Kit discovered, then closed, verify evicted.
func testDiscoveryBusLeave(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)

	natsURL := startNATSContainer(t)

	kit1, err := brainkit.New(brainkit.Config{
		Namespace: "disc-stay-cross",
		CallerID:  "host",
		Transport: transports.NATS(natsURL),
		Modules:   busDiscoveryModules(1*time.Second, 5*time.Second),
	})
	require.NoError(t, err)
	defer kit1.Close()

	kit2, err := brainkit.New(brainkit.Config{
		Namespace: "disc-leave-cross",
		CallerID:  "host",
		Transport: transports.NATS(natsURL),
		Modules:   busDiscoveryModules(1*time.Second, 5*time.Second),
	})
	require.NoError(t, err)

	// Wait for mutual discovery
	time.Sleep(3 * time.Second)

	// Verify kit2 is discovered
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp1, err := topology.CallPeersList(kit1, ctx, topology.PeersListMsg{})
	require.NoError(t, err)
	assert.Contains(t, peerNamespaces(resp1.Peers), "disc-leave-cross", "kit2 should be discovered before leave")

	// Graceful close — sends leave message
	kit2.Close()
	time.Sleep(1 * time.Second)

	// Verify kit2 is removed immediately (leave message, not TTL)
	resp2, err := topology.CallPeersList(kit1, ctx, topology.PeersListMsg{})
	require.NoError(t, err)
	assert.NotContains(t, peerNamespaces(resp2.Peers), "disc-leave-cross", "kit2 should be gone after graceful leave")
}

// testDiscoveryBusNamespaces — 3 Kits (2 "agents" replicas + 1 "gateway"), verify BrowseNamespaces dedup.
func testDiscoveryBusNamespaces(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)

	natsURL := startNATSContainer(t)

	makeKit := func(ns string) *brainkit.Kit {
		kit, err := brainkit.New(brainkit.Config{
			Namespace: ns,
			CallerID:  "host",
			Transport: transports.NATS(natsURL),
			Modules:   busDiscoveryModules(1*time.Second, 5*time.Second),
		})
		require.NoError(t, err)
		t.Cleanup(func() { kit.Close() })
		return kit
	}

	observer := makeKit("disc-observer-cross")
	makeKit("disc-agents-ns-cross") // replica 1
	makeKit("disc-agents-ns-cross") // replica 2 — same namespace
	makeKit("disc-gateway-ns-cross")

	// Wait for convergence
	time.Sleep(4 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := topology.CallPeersList(observer, ctx, topology.PeersListMsg{})
	require.NoError(t, err)
	// 3 individual peers (not counting self)
	assert.GreaterOrEqual(t, len(resp.Peers), 3, "should see 3 peer nodes")
	// But only 2 unique namespaces (agents deduped)
	assert.Len(t, resp.Namespaces, 2, "should see agents + gateway (deduplicated)")
	nsMap := map[string]bool{}
	for _, ns := range resp.Namespaces {
		nsMap[ns] = true
	}
	assert.True(t, nsMap["disc-agents-ns-cross"], "should find agents namespace")
	assert.True(t, nsMap["disc-gateway-ns-cross"], "should find gateway namespace")
}

func testDiscoveryStaticPeersBus(t *testing.T, _ *suite.TestEnv) {
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test-disc-cross",
		CallerID:  "test-node",
		Transport: transports.EmbeddedNATS(),
		Modules: []bkmodule.Module{
			topology.NewModule(topology.Config{
				Peers: []topology.Peer{
					{Name: "peer-a", Namespace: "ns-a", Address: "localhost:4222"},
					{Name: "peer-b", Namespace: "ns-b", Address: "localhost:4223"},
				},
			}),
		},
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := topology.CallPeersList(kit, ctx, topology.PeersListMsg{})
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(resp.Peers), 2, "should have at least 2 static peers")
	names := make(map[string]bool)
	for _, p := range resp.Peers {
		names[p.Name] = true
	}
	assert.True(t, names["peer-a"], "should find peer-a")
	assert.True(t, names["peer-b"], "should find peer-b")
}

func peerNamespaces(peers []topology.PeerInfo) []string {
	namespaces := make([]string, 0, len(peers))
	for _, peer := range peers {
		namespaces = append(namespaces, peer.Namespace)
	}
	return namespaces
}
