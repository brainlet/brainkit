package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

// TestPluginsNoModule verifies that a Kit built without the plugins module
// does not attach a handler for plugin.list — a PluginListRunningMsg round
// trip times out. Mirror of the audit/scheduling no-module checks: proves
// the lifecycle is fully owned by modules/plugins and nothing else wires it
// up behind the scenes.
func TestPluginsNoModule(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-plugins-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pr, err := sdk.PublishPluginListRunning(k, ctx, sdk.PluginListRunningMsg{})
	require.NoError(t, err)

	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m })
	require.NoError(t, err)
	defer unsub()

	select {
	case m := <-ch:
		t.Fatalf("expected no reply for plugin.list without plugins module, got payload=%s", string(m.Payload))
	case <-ctx.Done():
		// Expected — no handler, no reply.
	}
}
