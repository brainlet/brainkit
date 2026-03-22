package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlugin_InProcess verifies that the unified sdk.Runtime interface works
// the same way a plugin would use it — via PublishAwait over the transport.
// Uses a Node with memory transport to simulate the plugin side.
//
// In a real plugin, the sdk.Runtime is backed by a pluginClient connected
// via NATS. Here we test the same contract through the Node, which delegates
// to the same Kernel. The key insight: if all three (Kernel, Node, pluginClient)
// implement sdk.Runtime, and the tests pass on Node, the plugin path works too.
func TestPlugin_InProcess(t *testing.T) {
	// Use Node as the "plugin-side" Runtime — same interface, same behavior
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Plugin typical flow: list tools, call a tool, read/write files

	t.Run("Plugin_ListTools", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](rt, ctx, messages.ToolListMsg{})
		require.NoError(t, err)
		found := false
		for _, tool := range resp.Tools {
			if tool.ShortName == "echo" {
				found = true
			}
		}
		assert.True(t, found, "plugin should see registered tools")
	})

	t.Run("Plugin_CallTool", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
			Name:  "add",
			Input: map[string]any{"a": 100, "b": 200},
		})
		require.NoError(t, err)
		var result map[string]int
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, 300, result["sum"])
	})

	t.Run("Plugin_FS_WriteRead", func(t *testing.T) {
		_, err := sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{
			Path: "plugin-data.json", Data: `{"status":"ok"}`,
		})
		require.NoError(t, err)

		resp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "plugin-data.json"})
		require.NoError(t, err)
		assert.Equal(t, `{"status":"ok"}`, resp.Data)
	})

	t.Run("Plugin_Deploy_Teardown", func(t *testing.T) {
		deployResp, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
			Source: "plugin-created.ts",
			Code:   `const t = createTool({ id: "plugin-tool", description: "from plugin", execute: async () => ({ created: true }) });`,
		})
		require.NoError(t, err)
		assert.True(t, deployResp.Deployed)

		_, err = sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "plugin-created.ts"})
		require.NoError(t, err)
	})

	t.Run("Plugin_Async_Subscribe", func(t *testing.T) {
		// Simulate plugin subscribing to tool list results asynchronously
		corrID, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
		require.NoError(t, err)
		assert.NotEmpty(t, corrID)

		received := make(chan bool, 1)
		cancel, err := sdk.Subscribe[messages.ToolListResp](rt, ctx, func(resp messages.ToolListResp, msg messages.Message) {
			if msg.Metadata["correlationId"] == corrID {
				received <- true
			}
		})
		require.NoError(t, err)
		defer cancel()

		select {
		case <-received:
			// OK
		case <-time.After(5 * time.Second):
			t.Fatal("plugin async subscribe: timeout waiting for response")
		}
	})
}
