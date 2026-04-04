package cross

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/discovery"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
func testDiscoveryStaticPeersBus(t *testing.T, _ *suite.TestEnv) {
	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace: "test-disc-cross",
			CallerID:  "test-node",
		},
		Messaging: brainkit.MessagingConfig{Transport: "memory"},
		Discovery: discovery.Config{
			Type: "static",
			StaticPeers: []discovery.PeerConfig{
				{Name: "peer-a", Namespace: "ns-a", Address: "localhost:4222"},
				{Name: "peer-b", Namespace: "ns-b", Address: "localhost:4223"},
			},
		},
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))
	defer node.Close()

	ctx := context.Background()

	// peers.list via bus
	pr, err := sdk.Publish(node, ctx, messages.PeersListMsg{})
	require.NoError(t, err)

	listCh := make(chan messages.PeersListResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.PeersListResp](node, ctx, pr.ReplyTo, func(resp messages.PeersListResp, _ messages.Message) {
		listCh <- resp
	})
	defer unsub()

	select {
	case resp := <-listCh:
		assert.GreaterOrEqual(t, len(resp.Peers), 2, "should have at least 2 static peers")
		names := make(map[string]bool)
		for _, p := range resp.Peers {
			names[p.Name] = true
		}
		assert.True(t, names["peer-a"], "should find peer-a")
		assert.True(t, names["peer-b"], "should find peer-b")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for peers.list response")
	}
}
