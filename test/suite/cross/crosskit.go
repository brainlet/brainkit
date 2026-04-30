package cross

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/brainlet/brainkit/transports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// --- TS <-> Go cross-kit (from test/cross/ts_go_test.go) ---

func testTSDeploysToolGoCallsIt(t *testing.T, env *suite.TestEnv) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			rt := testutil.NewTestKitFullWithBackend(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// TS surface: deploy .ts that creates a tool
			m1, _ := json.Marshal(map[string]string{"name": "cross-ts-tool-cross", "entry": "cross-ts-tool-cross.ts"})
			_, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](rt, ctx, packagemsg.PackageDeployMsg{
				Manifest: m1,
				Files: map[string]string{"cross-ts-tool-cross.ts": `
					const myTool = createTool({
						id: "ts-greeter",
						description: "greets from TS",
						execute: async ({ context: input }) => {
							return { greeting: "hello from TS, " + (input.name || "world") };
						}
					});
					kit.register("tool", "ts-greeter", myTool);
					`},
			})
			require.NoError(t, err)

			// Go surface: call the TS-created tool via the shared reply inbox.
			resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](rt, ctx, toolmsg.ToolCallMsg{
				Name:  "ts-greeter",
				Input: map[string]any{"name": "Go"},
			})
			require.NoError(t, err)

			var result map[string]string
			json.Unmarshal(resp.Result, &result)
			assert.Equal(t, "hello from TS, Go", result["greeting"])

			// Cleanup
			_, err = sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](rt, ctx, packagemsg.PackageTeardownMsg{Name: "cross-ts-tool-cross"})
			require.NoError(t, err)
		})
	}
}

func testGoRegistersToolTSCallsViaDeploy(t *testing.T, env *suite.TestEnv) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			rt := testutil.NewTestKitFullWithBackend(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Go surface: "echo" tool is already registered by helpers

			// TS surface: deploy .ts that calls the Go-registered "echo" tool
			m3, _ := json.Marshal(map[string]string{"name": "cross-go-call-cross", "entry": "cross-go-call-cross.ts"})
			_, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](rt, ctx, packagemsg.PackageDeployMsg{
				Manifest: m3,
				Files: map[string]string{"cross-go-call-cross.ts": `
					const wrapper = createTool({
						id: "echo-wrapper",
						description: "calls Go echo tool from TS",
						execute: async ({ context: input }) => {
							const result = await tools.call("echo", { message: input.msg || "default" });
							return { wrapped: true, inner: result };
						}
					});
					kit.register("tool", "echo-wrapper", wrapper);
					`},
			})
			require.NoError(t, err)

			// Go surface: call the TS wrapper which internally calls the Go echo tool
			resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](rt, ctx, toolmsg.ToolCallMsg{
				Name:  "echo-wrapper",
				Input: map[string]any{"msg": "from TS to Go"},
			})
			require.NoError(t, err)

			var result map[string]any
			json.Unmarshal(resp.Result, &result)
			assert.Equal(t, true, result["wrapped"])
			inner, _ := result["inner"].(map[string]any)
			assert.Equal(t, "from TS to Go", inner["echoed"])

			_, err = sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](rt, ctx, packagemsg.PackageTeardownMsg{Name: "cross-go-call-cross"})
			require.NoError(t, err)
		})
	}
}

// --- Plugin <-> Go cross-kit (from test/cross/plugin_go_test.go) ---

func testPluginToolCalledFromGo(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping plugin tests in short mode")
	}
	env.RequirePodman(t)

	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			testutil.RequiresNetworkTransport(t, backend)

			pluginBinary := testutil.BuildTestPlugin(t)
			var natsURL string
			if backend == "nats" {
				natsURL = crossKitStartNATS(t)
			} else {
				t.Skipf("plugin cross-surface test only implemented for NATS backend currently")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			tmpDir := t.TempDir()
			kit, err := brainkit.New(brainkit.Config{
				Namespace: "plugin-cross",
				CallerID:  "host",
				FSRoot:    tmpDir,
				Transport: transports.NATS(natsURL, transports.WithNATSName("brainkit-cross-plugin")),
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
			defer kit.Close()

			require.NoError(t, kit.Mount(context.Background(), toolsmod.GoTool("host-multiply", toolsmod.TypedTool[struct {
				A int `json:"a"`
				B int `json:"b"`
			}]{
				Description: "multiplies two numbers",
				Execute: func(ctx context.Context, input struct {
					A int `json:"a"`
					B int `json:"b"`
				}) (any, error) {
					return map[string]int{"product": input.A * input.B}, nil
				},
			})))

			time.Sleep(2 * time.Second)

			toolCtx, toolCancel := context.WithTimeout(ctx, 10*time.Second)
			defer toolCancel()

			resp := callAndWait[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](t, kit, toolCtx, toolmsg.ToolCallMsg{
				Name:  "echo",
				Input: map[string]any{"message": "plugin->go test"},
			})

			var result map[string]string
			json.Unmarshal(resp.Result, &result)
			assert.Equal(t, "plugin->go test", result["echoed"])
			assert.Equal(t, "testplugin", result["plugin"])
		})
	}
}

func testGoToolVisibleInList(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping plugin tests in short mode")
	}
	env.RequirePodman(t)

	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			testutil.RequiresNetworkTransport(t, backend)

			pluginBinary := testutil.BuildTestPlugin(t)
			var natsURL string
			if backend == "nats" {
				natsURL = crossKitStartNATS(t)
			} else {
				t.Skipf("plugin cross-surface test only implemented for NATS backend currently")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			tmpDir := t.TempDir()
			kit, err := brainkit.New(brainkit.Config{
				Namespace: "plugin-cross-list",
				CallerID:  "host",
				FSRoot:    tmpDir,
				Transport: transports.NATS(natsURL, transports.WithNATSName("brainkit-cross-plugin-list")),
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
			defer kit.Close()

			require.NoError(t, kit.Mount(context.Background(), toolsmod.GoTool("host-multiply", toolsmod.TypedTool[struct {
				A int `json:"a"`
				B int `json:"b"`
			}]{
				Description: "multiplies two numbers",
				Execute: func(ctx context.Context, input struct {
					A int `json:"a"`
					B int `json:"b"`
				}) (any, error) {
					return map[string]int{"product": input.A * input.B}, nil
				},
			})))

			time.Sleep(2 * time.Second)

			listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
			defer listCancel()

			resp := callAndWait[toolmsg.ToolListMsg, toolmsg.ToolListResp](t, kit, listCtx, toolmsg.ToolListMsg{})

			names := make(map[string]bool)
			for _, tool := range resp.Tools {
				names[tool.ShortName] = true
			}
			assert.True(t, names["echo"], "plugin echo tool")
			assert.True(t, names["concat"], "plugin concat tool")
			assert.True(t, names["host-multiply"], "host-side tool")
		})
	}
}

// --- TS <-> Plugin cross-kit (from test/cross/ts_plugin_test.go) ---

func testTSCallsPluginTool(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping plugin tests in short mode")
	}
	env.RequirePodman(t)

	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			testutil.RequiresNetworkTransport(t, backend)
			if backend != "nats" {
				t.Skipf("TS<->Plugin cross-surface currently tested on NATS only")
			}

			pluginBinary := testutil.BuildTestPlugin(t)
			natsURL := startNATSContainer(t)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			tmpDir := t.TempDir()
			kit, err := brainkit.New(brainkit.Config{
				Namespace: "ts-plugin-cross",
				CallerID:  "host",
				FSRoot:    tmpDir,
				Transport: transports.NATS(natsURL, transports.WithNATSName("brainkit-ts-plugin")),
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
			defer kit.Close()

			time.Sleep(2 * time.Second)

			// Deploy .ts that calls the plugin's "concat" tool
			mfst1, _ := json.Marshal(map[string]string{"name": "ts-calls-plugin-cross", "entry": "ts-calls-plugin-cross.ts"})
			callAndWait[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](t, kit, ctx, packagemsg.PackageDeployMsg{
				Manifest: mfst1,
				Files: map[string]string{"ts-calls-plugin-cross.ts": `
					const pluginCaller = createTool({
						id: "plugin-caller",
						description: "calls plugin concat tool from TS",
						execute: async ({ context: input }) => {
							const result = await tools.call("concat", { a: input.x || "hello", b: input.y || "world" });
							return { fromPlugin: result };
						}
					});
					kit.register("tool", "plugin-caller", pluginCaller);
				`},
			})

			callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
			defer callCancel()

			resp := callAndWait[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](t, kit, callCtx, toolmsg.ToolCallMsg{
				Name:  "plugin-caller",
				Input: map[string]any{"x": "foo", "y": "bar"},
			})

			var result map[string]any
			json.Unmarshal(resp.Result, &result)
			inner, _ := result["fromPlugin"].(map[string]any)
			assert.Equal(t, "foobar", inner["result"])

			callAndWait[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](t, kit, ctx, packagemsg.PackageTeardownMsg{Name: "ts-calls-plugin-cross"})
		})
	}
}

func testTSDeployedToolVisibleAlongsidePlugin(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping plugin tests in short mode")
	}
	env.RequirePodman(t)

	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			testutil.RequiresNetworkTransport(t, backend)
			if backend != "nats" {
				t.Skipf("TS<->Plugin cross-surface currently tested on NATS only")
			}

			pluginBinary := testutil.BuildTestPlugin(t)
			natsURL := startNATSContainer(t)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			tmpDir := t.TempDir()
			kit, err := brainkit.New(brainkit.Config{
				Namespace: "ts-plugin-alongside-cross",
				CallerID:  "host",
				FSRoot:    tmpDir,
				Transport: transports.NATS(natsURL, transports.WithNATSName("brainkit-ts-plugin-alongside")),
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
			defer kit.Close()

			time.Sleep(2 * time.Second)

			// Deploy .ts tool
			mfst3, _ := json.Marshal(map[string]string{"name": "ts-alongside-cross", "entry": "ts-alongside-cross.ts"})
			callAndWait[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](t, kit, ctx, packagemsg.PackageDeployMsg{
				Manifest: mfst3,
				Files: map[string]string{"ts-alongside-cross.ts": `
					const tsTool = createTool({
						id: "ts-side-tool",
						description: "a TS-side tool",
						execute: async () => ({ from: "ts" })
					});
					kit.register("tool", "ts-side-tool", tsTool);
				`},
			})

			listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
			defer listCancel()

			resp := callAndWait[toolmsg.ToolListMsg, toolmsg.ToolListResp](t, kit, listCtx, toolmsg.ToolListMsg{})

			names := make(map[string]bool)
			for _, tool := range resp.Tools {
				names[tool.ShortName] = true
			}
			assert.True(t, names["echo"], "plugin echo tool")
			assert.True(t, names["concat"], "plugin concat tool")
			assert.True(t, names["ts-side-tool"], "TS-deployed tool")

			callAndWait[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](t, kit, ctx, packagemsg.PackageTeardownMsg{Name: "ts-alongside-cross"})
		})
	}
}

// crossKitStartNATS starts NATS with Podman environment setup.
func crossKitStartNATS(t *testing.T) string {
	t.Helper()
	testutil.EnsurePodmanSocket(t)

	ctx := context.Background()
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
		t.Skipf("failed to start NATS container: %v", err)
	}
	t.Cleanup(func() { natsContainer.Terminate(context.Background()) })

	host, _ := natsContainer.Host(ctx)
	port, _ := natsContainer.MappedPort(ctx, "4222")
	return fmt.Sprintf("nats://%s:%s", host, port.Port())
}
