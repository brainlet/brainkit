package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/discovery"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscovery_StaticPeers(t *testing.T) {
	node, err := kit.NewNode(kit.NodeConfig{
		Kernel: kit.KernelConfig{
			Namespace: "test",
			CallerID:  "test-node",
		},
		Messaging: kit.MessagingConfig{Transport: "memory"},
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
		// Should have 3 peers: peer-a, peer-b, and self (test-node)
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
