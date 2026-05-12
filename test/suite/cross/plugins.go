package cross

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/brainlet/brainkit/transports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// --- In-process plugin tests (from test/plugin/inprocess_test.go) ---

func testPluginInProcessListTools(t *testing.T, env *suite.TestEnv) {
	rt := testutil.NewTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp := callAndWait[toolmsg.ToolListMsg, toolmsg.ToolListResp](t, rt, ctx, toolmsg.ToolListMsg{})
	found := false
	for _, tool := range resp.Tools {
		if tool.ShortName == "echo" {
			found = true
		}
	}
	assert.True(t, found, "plugin should see registered tools")
}

func testPluginInProcessCallTool(t *testing.T, env *suite.TestEnv) {
	rt := testutil.NewTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp := callAndWait[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](t, rt, ctx, toolmsg.ToolCallMsg{
		Name:  "add",
		Input: map[string]any{"a": 100, "b": 200},
	})
	var result map[string]int
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, 300, result["sum"])
}

func testPluginInProcessFSWriteRead(t *testing.T, env *suite.TestEnv) {
	tk := testutil.NewTestKitFull(t)
	fsCtx, fsCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer fsCancel()

	result := testutil.EvalJS(t, tk, "__test-cross.ts", `
		fs.writeFileSync("plugin-data.json", '{"status":"ok"}');
		return fs.readFileSync("plugin-data.json", "utf8");
	`)
	_ = fsCtx
	assert.Equal(t, `{"status":"ok"}`, result)
}

func testPluginInProcessDeployTeardown(t *testing.T, env *suite.TestEnv) {
	rt := testutil.NewTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pluginManifest, _ := json.Marshal(map[string]string{"name": "plugin-created-cross", "entry": "plugin-created-cross.ts"})
	deployResp := callAndWait[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](t, rt, ctx, packagemsg.PackageDeployMsg{
		Manifest: pluginManifest,
		Files:    map[string]string{"plugin-created-cross.ts": `const t = createTool({ id: "plugin-tool", description: "from plugin", execute: async () => ({ created: true }) }); kit.register("tool", "plugin-tool", t);`},
	})
	assert.True(t, deployResp.Deployed)

	callAndWait[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](t, rt, ctx, packagemsg.PackageTeardownMsg{Name: "plugin-created-cross"})
}

func testPluginInProcessAsyncSubscribe(t *testing.T, env *suite.TestEnv) {
	rt := testutil.NewTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	replyTo := fmt.Sprintf("tools.list.reply.%d", time.Now().UnixNano())

	received := make(chan bool, 1)
	unsub, err := sdk.SubscribeTo[toolmsg.ToolListResp](rt, ctx, replyTo, func(resp toolmsg.ToolListResp, msg sdk.Message) {
		received <- true
	})
	require.NoError(t, err)
	defer unsub()
	_, err = protocol.Publish(rt, ctx, toolmsg.ToolListMsg{}, protocol.WithReplyTo(replyTo))
	require.NoError(t, err)

	select {
	case <-received:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("plugin async subscribe: timeout waiting for response")
	}
}

// --- Subprocess plugin tests (from test/plugin/subprocess_test.go) ---

func testPluginSubprocessEcho(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping subprocess plugin test in short mode")
	}
	kit := buildSubprocessKit(t, env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := callAndWait[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](t, kit, ctx, toolmsg.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "hello from host"},
	})

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, "hello from host", result["echoed"])
	assert.Equal(t, "testplugin", result["plugin"])
}

func testPluginSubprocessConcat(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping subprocess plugin test in short mode")
	}
	kit := buildSubprocessKit(t, env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := callAndWait[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](t, kit, ctx, toolmsg.ToolCallMsg{
		Name:  "concat",
		Input: map[string]any{"a": "foo", "b": "bar"},
	})

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, "foobar", result["result"])
}

func testPluginSubprocessHostToolStillWorks(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping subprocess plugin test in short mode")
	}
	kit := buildSubprocessKit(t, env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := callAndWait[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](t, kit, ctx, toolmsg.ToolCallMsg{
		Name:  "host-add",
		Input: map[string]any{"a": 10, "b": 20},
	})

	var result map[string]int
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, 30, result["sum"])
}

func testPluginSubprocessToolsListShowsBoth(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping subprocess plugin test in short mode")
	}
	kit := buildSubprocessKit(t, env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := callAndWait[toolmsg.ToolListMsg, toolmsg.ToolListResp](t, kit, ctx, toolmsg.ToolListMsg{})

	names := make(map[string]bool)
	for _, tool := range resp.Tools {
		names[tool.ShortName] = true
	}
	assert.True(t, names["echo"], "plugin echo tool should be listed")
	assert.True(t, names["concat"], "plugin concat tool should be listed")
	assert.True(t, names["host-add"], "host-side tool should be listed")
}

// buildSubprocessKit creates a full subprocess plugin Kit with NATS.
// Returns the started kit. Cleans up on test completion.
func buildSubprocessKit(t *testing.T, env *suite.TestEnv) *brainkit.Kit {
	t.Helper()
	env.RequirePodman(t)

	pluginBinary := testutil.BuildTestPlugin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// Configure testcontainers for Podman
	testutil.EnsurePodmanSocket(t)

	if os.Getenv("DOCKER_HOST") == "" {
		cancel()
		t.Skip("DOCKER_HOST not set and podman socket not found")
	}

	natsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:latest",
			ExposedPorts: []string{"4222/tcp"},
			Cmd:          []string{"-js"},
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		cancel()
		t.Skipf("failed to start NATS container: %v", err)
	}

	natsHost, err := natsContainer.Host(ctx)
	require.NoError(t, err)
	natsPort, err := natsContainer.MappedPort(ctx, "4222")
	require.NoError(t, err)
	natsURL := fmt.Sprintf("nats://%s:%s", natsHost, natsPort.Port())

	// Verify NATS is accepting connections
	natsReadyCtx, natsReadyCancel := context.WithTimeout(ctx, 15*time.Second)
	defer natsReadyCancel()
	for {
		_, connErr := exec.CommandContext(natsReadyCtx, "nc", "-z", natsHost, natsPort.Port()).CombinedOutput()
		if connErr == nil {
			break
		}
		select {
		case <-natsReadyCtx.Done():
			cancel()
			t.Fatalf("NATS never became ready: %v", natsReadyCtx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}

	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-e2e-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: transports.NATS(natsURL, transports.WithNATSName("brainkit-test-cross")),
		Modules: packageModules(
			pluginsmod.NewModule(pluginsmod.Config{
				Plugins: []pluginsmod.PluginConfig{
					{
						Name:         "testplugin",
						Binary:       pluginBinary,
						StartTimeout: 30 * time.Second,
					},
				},
			}),
		),
	})
	require.NoError(t, err)

	require.NoError(t, kit.Mount(context.Background(), toolsmod.GoTool("host-add", toolsmod.TypedTool[testutil.AddInput]{
		Description: "adds two numbers (host-side)",
		Execute: func(ctx context.Context, input testutil.AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})))

	time.Sleep(2 * time.Second)

	t.Cleanup(func() {
		kit.Close()
		natsContainer.Terminate(context.Background())
		cancel()
	})

	return kit
}
