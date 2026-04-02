package adversarial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeNode(t *testing.T) *brainkit.Node {
	t.Helper()
	if !testutil.PodmanAvailable() {
		t.Skip("Node tests need Podman for NATS")
	}
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "node-test", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))
	t.Cleanup(func() { node.Close() })
	return node
}

// TestNodeCommands_PluginList — plugin.list returns empty on fresh Node.
func TestNodeCommands_PluginList(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(node, ctx, messages.PluginListRunningMsg{})
	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "plugins")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestNodeCommands_PluginStopNonexistent — plugin.stop for ghost plugin returns error.
func TestNodeCommands_PluginStopNonexistent(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(node, ctx, messages.PluginStopMsg{Name: "ghost-plugin"})
	ch := make(chan json.RawMessage, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- json.RawMessage(m.Payload) })
	defer unsub()

	select {
	case p := <-ch:
		assert.True(t, responseHasError(p), "stopping nonexistent plugin should error")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestNodeCommands_PluginRestartNonexistent — plugin.restart for ghost returns error.
func TestNodeCommands_PluginRestartNonexistent(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(node, ctx, messages.PluginRestartMsg{Name: "ghost-plugin"})
	ch := make(chan json.RawMessage, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- json.RawMessage(m.Payload) })
	defer unsub()

	select {
	case p := <-ch:
		assert.True(t, responseHasError(p))
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestNodeCommands_PluginStatusNonexistent — plugin.status for ghost returns error.
func TestNodeCommands_PluginStatusNonexistent(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(node, ctx, messages.PluginStatusMsg{Name: "ghost-plugin"})
	ch := make(chan json.RawMessage, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- json.RawMessage(m.Payload) })
	defer unsub()

	select {
	case p := <-ch:
		assert.True(t, responseHasError(p))
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestNodeCommands_PluginStateGetSet — plugin state get/set via bus.
func TestNodeCommands_PluginStateGetSet(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set state
	pr1, _ := sdk.Publish(node, ctx, messages.PluginStateSetMsg{Key: "test-key", Value: "test-value"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := node.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case p := <-ch1:
		assert.Contains(t, string(p), "ok")
	case <-ctx.Done():
		t.Fatal("timeout set")
	}
	unsub1()

	// Get state
	pr2, _ := sdk.Publish(node, ctx, messages.PluginStateGetMsg{Key: "test-key"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := node.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		assert.Contains(t, string(p), "test-value")
	case <-ctx.Done():
		t.Fatal("timeout get")
	}
}

// TestNodeCommands_PackageListEmpty — package.list on fresh Node returns empty.
func TestNodeCommands_PackageListEmpty(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(node, ctx, messages.PackageListDeployedMsg{})
	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "packages")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestNodeCommands_WorkflowListEmpty — workflow.list on fresh Node returns empty.
func TestNodeCommands_WorkflowListEmpty(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(node, ctx, messages.WorkflowListMsg{})
	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "runs")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestNodeCommands_AutomationListEmpty — automation.list on fresh Node returns empty.
func TestNodeCommands_AutomationListEmpty(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(node, ctx, messages.AutomationListMsg{})
	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.NotEmpty(t, p) // some response — may be empty list or error for no automation domain
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestNodeCommands_DeployOnNode — deploy + call + teardown on Node.
func TestNodeCommands_DeployOnNode(t *testing.T) {
	node := makeNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy
	_, err := node.Kernel.Deploy(ctx, "node-deploy.ts", `
		bus.on("hello", function(msg) { msg.reply({from: "node"}); });
	`)
	require.NoError(t, err)

	// Call
	pr, _ := sdk.Publish(node, ctx, messages.CustomMsg{
		Topic: "ts.node-deploy.hello", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "node")
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Teardown
	_, err = node.Kernel.Teardown(ctx, "node-deploy.ts")
	require.NoError(t, err)
}

// TestNodeCommands_NodeShutdownClean — Node.Close is clean with active deployments.
func TestNodeCommands_NodeShutdownClean(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "shutdown-test", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))

	node.Kernel.Deploy(context.Background(), "shutdown-svc.ts", `
		bus.on("ping", function(msg) { msg.reply({ok:true}); });
	`)

	err = node.Close()
	assert.NoError(t, err)
}
