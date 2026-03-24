package plugin_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlugin_InProcess verifies that the unified sdk.Runtime interface works
// the same way a plugin would use it — via Publish over the transport.
// Uses a Node with memory transport to simulate the plugin side.
//
// In a real plugin, the sdk.Runtime is backed by a pluginClient connected
// via NATS. Here we test the same contract through the Node, which delegates
// to the same Kernel. The key insight: if all three (Kernel, Node, pluginClient)
// implement sdk.Runtime, and the tests pass on Node, the plugin path works too.
func TestPlugin_InProcess(t *testing.T) {
	// Use Node as the "plugin-side" Runtime — same interface, same behavior
	rt := testutil.NewTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Plugin typical flow: list tools, call a tool, read/write files

	t.Run("Plugin_ListTools", func(t *testing.T) {
		_pr1, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
		require.NoError(t, err)
		_ch1 := make(chan messages.ToolListResp, 1)
		_us1, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, _pr1.ReplyTo, func(r messages.ToolListResp, m messages.Message) { _ch1 <- r })
		require.NoError(t, err)
		defer _us1()
		var resp messages.ToolListResp
		select {
		case resp = <-_ch1:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		found := false
		for _, tool := range resp.Tools {
			if tool.ShortName == "echo" {
				found = true
			}
		}
		assert.True(t, found, "plugin should see registered tools")
	})

	t.Run("Plugin_CallTool", func(t *testing.T) {
		_pr2, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
			Name:  "add",
			Input: map[string]any{"a": 100, "b": 200},
		})
		require.NoError(t, err)
		_ch2 := make(chan messages.ToolCallResp, 1)
		_us2, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch2 <- r })
		require.NoError(t, err)
		defer _us2()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch2:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]int
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, 300, result["sum"])
	})

	t.Run("Plugin_FS_WriteRead", func(t *testing.T) {
		_pr3, err := sdk.Publish(rt, ctx, messages.FsWriteMsg{
			Path: "plugin-data.json", Data: `{"status":"ok"}`,
		})
		require.NoError(t, err)
		_ch3 := make(chan messages.FsWriteResp, 1)
		_us3, _ := sdk.SubscribeTo[messages.FsWriteResp](rt, ctx, _pr3.ReplyTo, func(r messages.FsWriteResp, m messages.Message) { _ch3 <- r })
		defer _us3()
		select {
		case <-_ch3:
		case <-ctx.Done():
			t.Fatal("timeout")
		}

		_pr4, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "plugin-data.json"})
		require.NoError(t, err)
		_ch4 := make(chan messages.FsReadResp, 1)
		_us4, err := sdk.SubscribeTo[messages.FsReadResp](rt, ctx, _pr4.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch4 <- r })
		require.NoError(t, err)
		defer _us4()
		var resp messages.FsReadResp
		select {
		case resp = <-_ch4:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Equal(t, `{"status":"ok"}`, resp.Data)
	})

	t.Run("Plugin_Deploy_Teardown", func(t *testing.T) {
		_pr5, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
			Source: "plugin-created.ts",
			Code:   `const t = createTool({ id: "plugin-tool", description: "from plugin", execute: async () => ({ created: true }) }); kit.register("tool", "plugin-tool", t);`,
		})
		require.NoError(t, err)
		_ch5 := make(chan messages.KitDeployResp, 1)
		_us5, err := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr5.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch5 <- r })
		require.NoError(t, err)
		defer _us5()
		var deployResp messages.KitDeployResp
		select {
		case deployResp = <-_ch5:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, deployResp.Deployed)

		_pr6, err := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "plugin-created.ts"})
		require.NoError(t, err)
		_ch6 := make(chan messages.KitTeardownResp, 1)
		_us6, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _pr6.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _ch6 <- r })
		defer _us6()
		select {
		case <-_ch6:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	})

	t.Run("Plugin_Async_Subscribe", func(t *testing.T) {
		// Simulate plugin subscribing to tool list results asynchronously
		pubResult, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
		require.NoError(t, err)
		assert.NotEmpty(t, pubResult.ReplyTo)

		received := make(chan bool, 1)
		cancel, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, pubResult.ReplyTo, func(resp messages.ToolListResp, msg messages.Message) {
			received <- true
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
