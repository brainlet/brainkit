package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
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

	_, err = pluginmsg.CallPluginListRunning(k, ctx, pluginmsg.PluginListRunningMsg{})
	require.Error(t, err)
}
