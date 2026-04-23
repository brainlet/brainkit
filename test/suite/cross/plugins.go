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
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/testutil"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
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

	pr1, err := sdk.Publish(rt, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)
	ch1 := make(chan sdk.ToolListResp, 1)
	us1, err := sdk.SubscribeTo[sdk.ToolListResp](rt, ctx, pr1.ReplyTo, func(r sdk.ToolListResp, m sdk.Message) { ch1 <- r })
	require.NoError(t, err)
	defer us1()
	var resp sdk.ToolListResp
	select {
	case resp = <-ch1:
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
}

func testPluginInProcessCallTool(t *testing.T, env *suite.TestEnv) {
	rt := testutil.NewTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr2, err := sdk.Publish(rt, ctx, sdk.ToolCallMsg{
		Name:  "add",
		Input: map[string]any{"a": 100, "b": 200},
	})
	require.NoError(t, err)
	ch2 := make(chan sdk.ToolCallResp, 1)
	us2, err := sdk.SubscribeTo[sdk.ToolCallResp](rt, ctx, pr2.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) { ch2 <- r })
	require.NoError(t, err)
	defer us2()
	var resp sdk.ToolCallResp
	select {
	case resp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	var result map[string]int
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, 300, result["sum"])
}

func testPluginInProcessFSWriteRead(t *testing.T, env *suite.TestEnv) {
	tk := testutil.NewTestKitFull(t)
	fsCtx, fsCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer fsCancel()

	result := testutil.EvalTS(t, tk, "__test-cross.ts", `
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
	pr5, err := sdk.Publish(rt, ctx, sdk.PackageDeployMsg{
		Manifest: pluginManifest,
		Files:    map[string]string{"plugin-created-cross.ts": `const t = createTool({ id: "plugin-tool", description: "from plugin", execute: async () => ({ created: true }) }); kit.register("tool", "plugin-tool", t);`},
	})
	require.NoError(t, err)
	ch5 := make(chan sdk.PackageDeployResp, 1)
	us5, err := sdk.SubscribeTo[sdk.PackageDeployResp](rt, ctx, pr5.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch5 <- r })
	require.NoError(t, err)
	defer us5()
	var deployResp sdk.PackageDeployResp
	select {
	case deployResp = <-ch5:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, deployResp.Deployed)

	pr6, err := sdk.Publish(rt, ctx, sdk.PackageTeardownMsg{Name: "plugin-created-cross"})
	require.NoError(t, err)
	ch6 := make(chan sdk.PackageTeardownResp, 1)
	us6, _ := sdk.SubscribeTo[sdk.PackageTeardownResp](rt, ctx, pr6.ReplyTo, func(r sdk.PackageTeardownResp, m sdk.Message) { ch6 <- r })
	defer us6()
	select {
	case <-ch6:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testPluginInProcessAsyncSubscribe(t *testing.T, env *suite.TestEnv) {
	rt := testutil.NewTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pubResult, err := sdk.Publish(rt, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)
	assert.NotEmpty(t, pubResult.ReplyTo)

	received := make(chan bool, 1)
	unsub, err := sdk.SubscribeTo[sdk.ToolListResp](rt, ctx, pubResult.ReplyTo, func(resp sdk.ToolListResp, msg sdk.Message) {
		received <- true
	})
	require.NoError(t, err)
	defer unsub()

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

	pr1, err := sdk.Publish(kit, ctx, sdk.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "hello from host"},
	})
	require.NoError(t, err)
	ch1 := make(chan sdk.ToolCallResp, 1)
	us1, err := sdk.SubscribeTo[sdk.ToolCallResp](kit, ctx, pr1.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) { ch1 <- r })
	require.NoError(t, err)
	defer us1()
	var resp sdk.ToolCallResp
	select {
	case resp = <-ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

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

	pr2, err := sdk.Publish(kit, ctx, sdk.ToolCallMsg{
		Name:  "concat",
		Input: map[string]any{"a": "foo", "b": "bar"},
	})
	require.NoError(t, err)
	ch2 := make(chan sdk.ToolCallResp, 1)
	us2, err := sdk.SubscribeTo[sdk.ToolCallResp](kit, ctx, pr2.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) { ch2 <- r })
	require.NoError(t, err)
	defer us2()
	var resp sdk.ToolCallResp
	select {
	case resp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

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

	pr3, err := sdk.Publish(kit, ctx, sdk.ToolCallMsg{
		Name:  "host-add",
		Input: map[string]any{"a": 10, "b": 20},
	})
	require.NoError(t, err)
	ch3 := make(chan sdk.ToolCallResp, 1)
	us3, err := sdk.SubscribeTo[sdk.ToolCallResp](kit, ctx, pr3.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) { ch3 <- r })
	require.NoError(t, err)
	defer us3()
	var resp sdk.ToolCallResp
	select {
	case resp = <-ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

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

	pr4, err := sdk.Publish(kit, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)
	ch4 := make(chan sdk.ToolListResp, 1)
	us4, err := sdk.SubscribeTo[sdk.ToolListResp](kit, ctx, pr4.ReplyTo, func(r sdk.ToolListResp, m sdk.Message) { ch4 <- r })
	require.NoError(t, err)
	defer us4()
	var resp sdk.ToolListResp
	select {
	case resp = <-ch4:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

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
		Transport: brainkit.NATS(natsURL, brainkit.WithNATSName("brainkit-test-cross")),
		Modules: []brainkit.Module{
			pluginsmod.NewModule(pluginsmod.Config{
				Plugins: []brainkit.PluginConfig{
					{
						Name:         "testplugin",
						Binary:       pluginBinary,
						StartTimeout: 30 * time.Second,
					},
				},
			}),
		},
	})
	require.NoError(t, err)

	brainkit.RegisterTool(kit, "host-add", tools.TypedTool[testutil.AddInput]{
		Description: "adds two numbers (host-side)",
		Execute: func(ctx context.Context, input testutil.AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	time.Sleep(2 * time.Second)

	t.Cleanup(func() {
		kit.Close()
		natsContainer.Terminate(context.Background())
		cancel()
	})

	return kit
}
