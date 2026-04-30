package topology

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/stretchr/testify/require"
)

func TestTopologyModuleHotMountsCommands(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{Transport: brainkit.Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	mod := NewModule(Config{Peers: []Peer{{Name: "peer-a", Namespace: "peer-a-ns"}}})
	require.NoError(t, k.Mount(ctx, mod))

	resp, err := brainkit.Call[PeersResolveMsg, PeersResolveResp](
		k,
		ctx,
		PeersResolveMsg{Name: "peer-a"},
		brainkit.WithCallTimeout(2*time.Second),
	)
	require.NoError(t, err)
	require.Equal(t, "peer-a-ns", resp.Namespace)

	require.True(t, mountedCommand(k, "topology", "peers.resolve"))
	require.NoError(t, k.Unmount(ctx, "topology"))
	require.False(t, mountedCommand(k, "topology", "peers.resolve"))
}

func mountedCommand(k *brainkit.Kit, moduleID, topic string) bool {
	for _, desc := range k.MountedModules() {
		if desc.Name != moduleID {
			continue
		}
		for _, cmd := range desc.Commands {
			if cmd.Topic == topic {
				return true
			}
		}
	}
	return false
}
