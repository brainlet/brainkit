package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/brainlet/brainkit/transports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Plugin surface tests (from test/adversarial/plugin_surface_test.go) ---

func testPluginSurfaceGoToolFromPlugin(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-test-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: transports.EmbeddedNATS(),
		Modules:   []brainkit.Module{toolsmod.New()},
	})
	require.NoError(t, err)
	defer kit.Close()

	type echoIn struct {
		Message string `json:"message"`
	}
	require.NoError(t, kit.Mount(context.Background(), toolsmod.GoTool("host-echo", toolsmod.TypedTool[echoIn]{
		Description: "echoes from host",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message, "source": "host"}, nil
		},
	})))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := brainkit.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](kit, ctx, toolmsg.ToolCallMsg{Name: "host-echo", Input: map[string]any{"message": "from-plugin-surface"}})
	require.NoError(t, err)

	assert.Contains(t, string(resp.Result), "from-plugin-surface")
	assert.Contains(t, string(resp.Result), "host")
}

func testPluginSurfaceTSFromPlugin(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-ts-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: tf.Transport,
		Modules:   packageModules(),
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
	p := publishAndWaitRaw(t, kit, ctx, sdk.CustomMsg{
		Topic:   "ts.plugin-target-cross.ask",
		Payload: json.RawMessage(`{"q":"hello?"}`),
	})
	assert.Contains(t, string(p), "from-ts")
}

func testPluginSurfaceToolsList(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-list-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: tf.Transport,
		Modules:   packageModules(),
	})
	require.NoError(t, err)
	defer kit.Close()

	type addIn struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	require.NoError(t, kit.Mount(context.Background(), toolsmod.GoTool("add", toolsmod.TypedTool[addIn]{
		Description: "adds numbers",
		Execute: func(ctx context.Context, in addIn) (any, error) {
			return map[string]int{"sum": in.A + in.B}, nil
		},
	})))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	p := publishAndWaitRaw(t, kit, ctx, toolmsg.ToolListMsg{})
	assert.Contains(t, string(p), "add")
}

func testPluginSurfaceErrorCodeFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-err-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: tf.Transport,
		Modules:   packageModules(),
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Call nonexistent tool
	p := publishAndWaitJSON(t, kit, ctx, toolmsg.ToolCallMsg{Name: "ghost-plugin-tool"})
	code := suite.ResponseCode(p)
	assert.Equal(t, "NOT_FOUND", code)
}

func testPluginSurfaceSecretsFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-sec-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: tf.Transport,
		Modules:   []brainkit.Module{secretsmod.New()},
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Set secret
	p1 := publishAndWaitRaw(t, kit, ctx, secretmsg.SecretsSetMsg{Name: "plugin-key", Value: "plugin-val"})
	_ = p1

	// Get secret
	p2 := publishAndWaitRaw(t, kit, ctx, secretmsg.SecretsGetMsg{Name: "plugin-key"})
	assert.Contains(t, string(p2), "plugin-val")
}

func testPluginSurfaceDeployFromNode(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	tf := transportFieldsForBackend(t, "nats")
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-deploy-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: tf.Transport,
		Modules:   packageModules(),
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy via bus command
	nodeManifest, _ := json.Marshal(map[string]string{"name": "node-deploy-cross", "entry": "node-deploy-cross.ts"})
	p := publishAndWaitRaw(t, kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: nodeManifest,
		Files:    map[string]string{"node-deploy-cross.ts": `const t = createTool({id: "node-tool", description: "test", execute: async () => ({ok:true})}); kit.register("tool", "node-tool", t);`},
	})
	assert.Contains(t, string(p), "deployed")

	// Verify tool is registered
	p = publishAndWaitRaw(t, kit, ctx, toolmsg.ToolResolveMsg{Name: "node-tool"})
	assert.Contains(t, string(p), "node-tool")
}
