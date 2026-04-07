package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Plugin surface tests (from test/adversarial/plugin_surface_test.go) ---

func testPluginSurfaceGoToolFromPlugin(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-test-cross", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(node.Kernel, "host-echo", tools.TypedTool[echoIn]{
		Description: "echoes from host",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message, "source": "host"}, nil
		},
	})

	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

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

func testPluginSurfaceTSFromPlugin(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-ts-cross", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a .ts handler
	_, err = node.Kernel.Deploy(ctx, "plugin-target-cross.ts", `
		bus.on("ask", function(msg) { msg.reply({answer: "from-ts", question: msg.payload.q}); });
	`)
	require.NoError(t, err)

	// Simulate plugin calling the .ts via bus
	pr, err := sdk.Publish(node, ctx, messages.CustomMsg{
		Topic:   "ts.plugin-target-cross.ask",
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

func testPluginSurfaceToolsList(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-list-cross", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()

	type addIn struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	brainkit.RegisterTool(node.Kernel, "add", tools.TypedTool[addIn]{
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

func testPluginSurfaceErrorCodeFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-err-cross", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Call nonexistent tool
	p := publishAndWaitJSON(t, node, ctx, messages.ToolCallMsg{Name: "ghost-plugin-tool"})
	code := suite.ResponseCode(p)
	assert.Equal(t, "NOT_FOUND", code)
}

func testPluginSurfaceSecretsFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-sec-cross", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Set secret
	p1 := publishAndWaitRaw(t, node, ctx, messages.SecretsSetMsg{Name: "plugin-key", Value: "plugin-val"})
	_ = p1

	// Get secret
	p2 := publishAndWaitRaw(t, node, ctx, messages.SecretsGetMsg{Name: "plugin-key"})
	assert.Contains(t, string(p2), "plugin-val")
}

func testPluginSurfaceDeployFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "plugin-deploy-cross", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	defer node.Close()
	require.NoError(t, node.Start(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy via bus command
	pr, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
		Source: "node-deploy-cross.ts",
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
