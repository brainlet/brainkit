package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"

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
		Modules:   []bkmodule.Module{toolsmod.New()},
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
	resp, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.plugin-target-cross.ask",
		Payload: json.RawMessage(`{"q":"hello?"}`),
	})
	require.NoError(t, err)
	assert.Contains(t, string(resp), "from-ts")
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

	resp, err := toolmsg.CallToolList(kit, ctx, toolmsg.ToolListMsg{})
	require.NoError(t, err)
	found := false
	for _, tool := range resp.Tools {
		if tool.ShortName == "add" || tool.Name == "add" {
			found = true
		}
	}
	assert.True(t, found, "tools.list should include add")
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
	_, err = toolmsg.CallToolCall(kit, ctx, toolmsg.ToolCallMsg{Name: "ghost-plugin-tool"})
	assertErrorCode(t, err, "NOT_FOUND")
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
		Modules:   []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Set secret
	setResp, err := secretmsg.CallSecretsSet(kit, ctx, secretmsg.SecretsSetMsg{Name: "plugin-key", Value: "plugin-val"})
	require.NoError(t, err)
	assert.True(t, setResp.Stored)

	// Get secret
	getResp, err := secretmsg.CallSecretsGet(kit, ctx, secretmsg.SecretsGetMsg{Name: "plugin-key"})
	require.NoError(t, err)
	assert.Equal(t, "plugin-val", getResp.Value)
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
	deployResp, err := packagemsg.CallPackageDeploy(kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: nodeManifest,
		Files:    map[string]string{"node-deploy-cross.ts": `const t = createTool({id: "node-tool", description: "test", execute: async () => ({ok:true})}); kit.register("tool", "node-tool", t);`},
	})
	require.NoError(t, err)
	assert.True(t, deployResp.Deployed)

	// Verify tool is registered
	resolveResp, err := toolmsg.CallToolResolve(kit, ctx, toolmsg.ToolResolveMsg{Name: "node-tool"})
	require.NoError(t, err)
	assert.Contains(t, resolveResp.Name, "node-tool")
}
