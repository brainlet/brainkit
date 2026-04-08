package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Plugin surface tests (from test/adversarial/plugin_surface_test.go) ---

func testPluginSurfaceGoToolFromPlugin(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace:   "plugin-test-cross",
		CallerID:    "host",
		FSRoot:      tmpDir,
		Transport:   tf.Transport,
		NATSURL:     tf.NATSURL,
		NATSName:    tf.NATSName,
		AMQPURL:     tf.AMQPURL,
		RedisURL:    tf.RedisURL,
		PostgresURL: tf.PostgresURL,
		SQLitePath:  tf.SQLitePath,
	})
	require.NoError(t, err)
	defer kit.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(kit, "host-echo", tools.TypedTool[echoIn]{
		Description: "echoes from host",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message, "source": "host"}, nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(kit, ctx, sdk.ToolCallMsg{Name: "host-echo", Input: map[string]any{"message": "from-plugin-surface"}})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
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
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace:   "plugin-ts-cross",
		CallerID:    "host",
		FSRoot:      tmpDir,
		Transport:   tf.Transport,
		NATSURL:     tf.NATSURL,
		NATSName:    tf.NATSName,
		AMQPURL:     tf.AMQPURL,
		RedisURL:    tf.RedisURL,
		PostgresURL: tf.PostgresURL,
		SQLitePath:  tf.SQLitePath,
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a .ts handler
	testutil.Deploy(t, kit, "plugin-target-cross.ts", `
		bus.on("ask", function(msg) { msg.reply({answer: "from-ts", question: msg.payload.q}); });
	`)

	// Simulate plugin calling the .ts via bus
	pr, err := sdk.Publish(kit, ctx, sdk.CustomMsg{
		Topic:   "ts.plugin-target-cross.ask",
		Payload: json.RawMessage(`{"q":"hello?"}`),
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
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
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace:   "plugin-list-cross",
		CallerID:    "host",
		FSRoot:      tmpDir,
		Transport:   tf.Transport,
		NATSURL:     tf.NATSURL,
		NATSName:    tf.NATSName,
		AMQPURL:     tf.AMQPURL,
		RedisURL:    tf.RedisURL,
		PostgresURL: tf.PostgresURL,
		SQLitePath:  tf.SQLitePath,
	})
	require.NoError(t, err)
	defer kit.Close()

	type addIn struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	brainkit.RegisterTool(kit, "add", tools.TypedTool[addIn]{
		Description: "adds numbers",
		Execute: func(ctx context.Context, in addIn) (any, error) {
			return map[string]int{"sum": in.A + in.B}, nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(kit, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
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
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace:   "plugin-err-cross",
		CallerID:    "host",
		FSRoot:      tmpDir,
		Transport:   tf.Transport,
		NATSURL:     tf.NATSURL,
		NATSName:    tf.NATSName,
		AMQPURL:     tf.AMQPURL,
		RedisURL:    tf.RedisURL,
		PostgresURL: tf.PostgresURL,
		SQLitePath:  tf.SQLitePath,
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Call nonexistent tool
	p := publishAndWaitJSON(t, kit, ctx, sdk.ToolCallMsg{Name: "ghost-plugin-tool"})
	code := suite.ResponseCode(p)
	assert.Equal(t, "NOT_FOUND", code)
}

func testPluginSurfaceSecretsFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace:   "plugin-sec-cross",
		CallerID:    "host",
		FSRoot:      tmpDir,
		Transport:   tf.Transport,
		NATSURL:     tf.NATSURL,
		NATSName:    tf.NATSName,
		AMQPURL:     tf.AMQPURL,
		RedisURL:    tf.RedisURL,
		PostgresURL: tf.PostgresURL,
		SQLitePath:  tf.SQLitePath,
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Set secret
	p1 := publishAndWaitRaw(t, kit, ctx, sdk.SecretsSetMsg{Name: "plugin-key", Value: "plugin-val"})
	_ = p1

	// Get secret
	p2 := publishAndWaitRaw(t, kit, ctx, sdk.SecretsGetMsg{Name: "plugin-key"})
	assert.Contains(t, string(p2), "plugin-val")
}

func testPluginSurfaceDeployFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace:   "plugin-deploy-cross",
		CallerID:    "host",
		FSRoot:      tmpDir,
		Transport:   tf.Transport,
		NATSURL:     tf.NATSURL,
		NATSName:    tf.NATSName,
		AMQPURL:     tf.AMQPURL,
		RedisURL:    tf.RedisURL,
		PostgresURL: tf.PostgresURL,
		SQLitePath:  tf.SQLitePath,
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy via bus command
	pr, err := sdk.Publish(kit, ctx, sdk.KitDeployMsg{
		Source: "node-deploy-cross.ts",
		Code:   `const t = createTool({id: "node-tool", description: "test", execute: async () => ({ok:true})}); kit.register("tool", "node-tool", t);`,
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "deployed")
	case <-ctx.Done():
		t.Fatal("timeout deploy")
	}

	// Verify tool is registered
	pr2, _ := sdk.Publish(kit, ctx, sdk.ToolResolveMsg{Name: "node-tool"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := kit.SubscribeRaw(ctx, pr2.ReplyTo, func(m sdk.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		assert.Contains(t, string(p), "node-tool")
	case <-ctx.Done():
		t.Fatal("timeout resolve")
	}
}
