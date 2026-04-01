package adversarial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginSurface_GoToolFromPlugin — plugin calls a Go-registered tool on the host.
func TestPluginSurface_GoToolFromPlugin(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Fatal("plugin tests need Podman for NATS")
	}

	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-test", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(node.Kernel, "host-echo", registry.TypedTool[echoIn]{
		Description: "echoes from host",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message, "source": "host"}, nil
		},
	})

	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Call the host tool via bus (same as plugin would)
	pr, err := sdk.Publish(node, ctx, messages.ToolCallMsg{Name: "host-echo", Input: map[string]any{"message": "from-plugin-surface"}})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "from-plugin-surface")
		assert.Contains(t, string(p), "host")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestPluginSurface_TSFromPlugin — .ts deployed on Node, plugin calls it via bus.
func TestPluginSurface_TSFromPlugin(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Fatal("plugin tests need Podman for NATS")
	}

	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-ts", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a .ts handler
	_, err = node.Kernel.Deploy(ctx, "plugin-target.ts", `
		bus.on("ask", function(msg) { msg.reply({answer: "from-ts", question: msg.payload.q}); });
	`)
	require.NoError(t, err)

	// Simulate plugin calling the .ts via bus
	pr, err := sdk.Publish(node, ctx, messages.CustomMsg{
		Topic:   "ts.plugin-target.ask",
		Payload: json.RawMessage(`{"q":"hello?"}`),
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "from-ts")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestPluginSurface_ToolsList — list tools from Node (includes both host tools and plugin tools).
func TestPluginSurface_ToolsList(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Fatal("plugin tests need Podman for NATS")
	}

	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-list", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()

	type addIn struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	brainkit.RegisterTool(node.Kernel, "add", registry.TypedTool[addIn]{
		Description: "adds numbers",
		Execute: func(ctx context.Context, in addIn) (any, error) {
			return map[string]int{"sum": in.A + in.B}, nil
		},
	})

	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(node, ctx, messages.ToolListMsg{})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "add")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestPluginSurface_ErrorCodeFromNode — error codes work on Node (not just Kernel).
func TestPluginSurface_ErrorCodeFromNode(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Fatal("plugin tests need Podman for NATS")
	}

	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-err", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Call nonexistent tool — should get NOT_FOUND with code
	pr, err := sdk.Publish(node, ctx, messages.ToolCallMsg{Name: "ghost-plugin-tool"})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	defer unsub()

	select {
	case payload := <-ch:
		code := responseCode(payload)
		assert.Equal(t, "NOT_FOUND", code)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestPluginSurface_SecretsFromNode — secrets work on Node.
func TestPluginSurface_SecretsFromNode(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Fatal("plugin tests need Podman for NATS")
	}

	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-sec", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Set secret
	pr1, _ := sdk.Publish(node, ctx, messages.SecretsSetMsg{Name: "plugin-key", Value: "plugin-val"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := node.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case <-ch1:
	case <-ctx.Done():
		t.Fatal("timeout set")
	}
	unsub1()

	// Get secret
	pr2, _ := sdk.Publish(node, ctx, messages.SecretsGetMsg{Name: "plugin-key"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := node.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		assert.Contains(t, string(p), "plugin-val")
	case <-ctx.Done():
		t.Fatal("timeout get")
	}
}

// TestPluginSurface_DeployFromNode — .ts deployment works on Node.
func TestPluginSurface_DeployFromNode(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Fatal("plugin tests need Podman for NATS")
	}

	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-deploy", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy via bus command
	pr, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
		Source: "node-deploy.ts",
		Code:   `const t = createTool({id: "node-tool", description: "test", execute: async () => ({ok:true})}); kit.register("tool", "node-tool", t);`,
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "deployed")
	case <-ctx.Done():
		t.Fatal("timeout deploy")
	}

	// Verify tool is registered
	pr2, _ := sdk.Publish(node, ctx, messages.ToolResolveMsg{Name: "node-tool"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := node.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		assert.Contains(t, string(p), "node-tool")
	case <-ctx.Done():
		t.Fatal("timeout resolve")
	}
}
