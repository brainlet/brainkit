package adversarial_test

import (
	"testing"

	"github.com/brainlet/brainkit/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscovery_StaticPeers — static discovery resolves configured peers.
func TestDiscovery_StaticPeers(t *testing.T) {
	provider := discovery.NewStaticFromConfig([]discovery.PeerConfig{
		{Name: "kit-a", Namespace: "ns-a", Address: "localhost:4222"},
		{Name: "kit-b", Namespace: "ns-b", Address: "localhost:4223"},
	})

	// Resolve returns the namespace, not the address — used for cross-Kit routing
	ns, err := provider.Resolve("kit-a")
	require.NoError(t, err)
	assert.Equal(t, "ns-a", ns)

	ns2, err := provider.Resolve("kit-b")
	require.NoError(t, err)
	assert.Equal(t, "ns-b", ns2)

	_, err = provider.Resolve("ghost")
	assert.Error(t, err)
}

// TestDiscovery_Browse — browse returns all registered peers.
func TestDiscovery_Browse(t *testing.T) {
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

// TestDiscovery_Register — register self adds to browse results.
func TestDiscovery_Register(t *testing.T) {
	provider := discovery.NewStaticFromConfig(nil)

	err := provider.Register(discovery.Peer{Name: "self", Namespace: "my-ns", Address: "localhost:5000"})
	require.NoError(t, err)

	peers, _ := provider.Browse()
	assert.Len(t, peers, 1)
	assert.Equal(t, "self", peers[0].Name)
}

// TestDiscovery_ResolveNonexistent — resolve unknown peer returns error.
func TestDiscovery_ResolveNonexistent(t *testing.T) {
	provider := discovery.NewStaticFromConfig(nil)

	_, err := provider.Resolve("nobody")
	assert.Error(t, err)
}

// TestDiscovery_Close — close doesn't panic.
func TestDiscovery_Close(t *testing.T) {
	provider := discovery.NewStaticFromConfig([]discovery.PeerConfig{
		{Name: "x", Namespace: "ns-x"},
	})

	err := provider.Close()
	assert.NoError(t, err)
}
