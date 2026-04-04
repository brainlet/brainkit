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
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// --- TS <-> Go cross-kit (from test/cross/ts_go_test.go) ---

func testTSDeploysToolGoCallsIt(t *testing.T, env *suite.TestEnv) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			rt := testutil.NewTestKernelFullWithBackend(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// TS surface: deploy .ts that creates a tool
			pr1, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
				Source: "cross-ts-tool-cross.ts",
				Code: `
					const myTool = createTool({
						id: "ts-greeter",
						description: "greets from TS",
						execute: async ({ context: input }) => {
							return { greeting: "hello from TS, " + (input.name || "world") };
						}
					});
					kit.register("tool", "ts-greeter", myTool);
				`,
			})
			require.NoError(t, err)
			ch1 := make(chan messages.KitDeployResp, 1)
			us1, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch1 <- r })
			defer us1()
			select {
			case <-ch1:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			// Go surface: call the TS-created tool via Publish
			pr2, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
				Name:  "ts-greeter",
				Input: map[string]any{"name": "Go"},
			})
			require.NoError(t, err)
			ch2 := make(chan messages.ToolCallResp, 1)
			us2, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch2 <- r })
			require.NoError(t, err)
			defer us2()
			var resp messages.ToolCallResp
			select {
			case resp = <-ch2:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			var result map[string]string
			json.Unmarshal(resp.Result, &result)
			assert.Equal(t, "hello from TS, Go", result["greeting"])

			// Cleanup
			spr1, _ := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "cross-ts-tool-cross.ts"})
			sch1 := make(chan messages.KitTeardownResp, 1)
			sun1, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, spr1.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { sch1 <- r })
			defer sun1()
			select {
			case <-sch1:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
		})
	}
}

func testGoRegistersToolTSCallsViaDeploy(t *testing.T, env *suite.TestEnv) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			rt := testutil.NewTestKernelFullWithBackend(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Go surface: "echo" tool is already registered by helpers

			// TS surface: deploy .ts that calls the Go-registered "echo" tool
			pr3, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
				Source: "cross-go-call-cross.ts",
				Code: `
					const wrapper = createTool({
						id: "echo-wrapper",
						description: "calls Go echo tool from TS",
						execute: async ({ context: input }) => {
							const result = await tools.call("echo", { message: input.msg || "default" });
							return { wrapped: true, inner: result };
						}
					});
					kit.register("tool", "echo-wrapper", wrapper);
				`,
			})
			require.NoError(t, err)
			ch3 := make(chan messages.KitDeployResp, 1)
			us3, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr3.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch3 <- r })
			defer us3()
			select {
			case <-ch3:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			// Go surface: call the TS wrapper which internally calls the Go echo tool
			pr4, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
				Name:  "echo-wrapper",
				Input: map[string]any{"msg": "from TS to Go"},
			})
			require.NoError(t, err)
			ch4 := make(chan messages.ToolCallResp, 1)
			us4, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, pr4.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch4 <- r })
			require.NoError(t, err)
			defer us4()
			var resp messages.ToolCallResp
			select {
			case resp = <-ch4:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			var result map[string]any
			json.Unmarshal(resp.Result, &result)
			assert.Equal(t, true, result["wrapped"])
			inner, _ := result["inner"].(map[string]any)
			assert.Equal(t, "from TS to Go", inner["echoed"])

			spr2, _ := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "cross-go-call-cross.ts"})
			sch2 := make(chan messages.KitTeardownResp, 1)
			sun2, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, spr2.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { sch2 <- r })
			defer sun2()
			select {
			case <-sch2:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
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
			node, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel: brainkit.KernelConfig{
					Namespace: "plugin-cross",
					CallerID:  "host",
					FSRoot:    tmpDir,
				},
				Messaging: brainkit.MessagingConfig{
					Transport: "nats",
					NATSURL:   natsURL,
					NATSName:  "brainkit-cross-plugin",
				},
				Plugins: []brainkit.PluginConfig{
					{
						Name:         "testplugin",
						Binary:       pluginBinary,
						StartTimeout: 30 * time.Second,
					},
				},
			})
			require.NoError(t, err)
			defer node.Close()

			brainkit.RegisterTool(node.Kernel, "host-multiply", registry.TypedTool[struct {
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
			})

			err = node.Start(ctx)
			require.NoError(t, err)
			time.Sleep(2 * time.Second)

			toolCtx, toolCancel := context.WithTimeout(ctx, 10*time.Second)
			defer toolCancel()

			pr1, err := sdk.Publish(node, toolCtx, messages.ToolCallMsg{
				Name:  "echo",
				Input: map[string]any{"message": "plugin->go test"},
			})
			require.NoError(t, err)
			ch1 := make(chan messages.ToolCallResp, 1)
			us1, err := sdk.SubscribeTo[messages.ToolCallResp](node, ctx, pr1.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch1 <- r })
			require.NoError(t, err)
			defer us1()
			var resp messages.ToolCallResp
			select {
			case resp = <-ch1:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

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
			node, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel: brainkit.KernelConfig{
					Namespace: "plugin-cross-list",
					CallerID:  "host",
					FSRoot:    tmpDir,
				},
				Messaging: brainkit.MessagingConfig{
					Transport: "nats",
					NATSURL:   natsURL,
					NATSName:  "brainkit-cross-plugin-list",
				},
				Plugins: []brainkit.PluginConfig{
					{
						Name:         "testplugin",
						Binary:       pluginBinary,
						StartTimeout: 30 * time.Second,
					},
				},
			})
			require.NoError(t, err)
			defer node.Close()

			brainkit.RegisterTool(node.Kernel, "host-multiply", registry.TypedTool[struct {
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
			})

			err = node.Start(ctx)
			require.NoError(t, err)
			time.Sleep(2 * time.Second)

			listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
			defer listCancel()

			pr2, err := sdk.Publish(node, listCtx, messages.ToolListMsg{})
			require.NoError(t, err)
			ch2 := make(chan messages.ToolListResp, 1)
			us2, err := sdk.SubscribeTo[messages.ToolListResp](node, ctx, pr2.ReplyTo, func(r messages.ToolListResp, m messages.Message) { ch2 <- r })
			require.NoError(t, err)
			defer us2()
			var resp messages.ToolListResp
			select {
			case resp = <-ch2:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

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
			node, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel: brainkit.KernelConfig{
					Namespace: "ts-plugin-cross",
					CallerID:  "host",
					FSRoot:    tmpDir,
				},
				Messaging: brainkit.MessagingConfig{
					Transport: "nats",
					NATSURL:   natsURL,
					NATSName:  "brainkit-ts-plugin",
				},
				Plugins: []brainkit.PluginConfig{
					{
						Name:         "testplugin",
						Binary:       pluginBinary,
						StartTimeout: 30 * time.Second,
					},
				},
			})
			require.NoError(t, err)
			defer node.Close()

			err = node.Start(ctx)
			require.NoError(t, err)
			time.Sleep(2 * time.Second)

			// Deploy .ts that calls the plugin's "concat" tool
			pr1, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
				Source: "ts-calls-plugin-cross.ts",
				Code: `
					const pluginCaller = createTool({
						id: "plugin-caller",
						description: "calls plugin concat tool from TS",
						execute: async ({ context: input }) => {
							const result = await tools.call("concat", { a: input.x || "hello", b: input.y || "world" });
							return { fromPlugin: result };
						}
					});
					kit.register("tool", "plugin-caller", pluginCaller);
				`,
			})
			require.NoError(t, err)
			ch1 := make(chan messages.KitDeployResp, 1)
			us1, _ := sdk.SubscribeTo[messages.KitDeployResp](node, ctx, pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch1 <- r })
			defer us1()
			select {
			case <-ch1:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
			defer callCancel()

			pr2, err := sdk.Publish(node, callCtx, messages.ToolCallMsg{
				Name:  "plugin-caller",
				Input: map[string]any{"x": "foo", "y": "bar"},
			})
			require.NoError(t, err)
			ch2 := make(chan messages.ToolCallResp, 1)
			us2, err := sdk.SubscribeTo[messages.ToolCallResp](node, ctx, pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch2 <- r })
			require.NoError(t, err)
			defer us2()
			var resp messages.ToolCallResp
			select {
			case resp = <-ch2:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			var result map[string]any
			json.Unmarshal(resp.Result, &result)
			inner, _ := result["fromPlugin"].(map[string]any)
			assert.Equal(t, "foobar", inner["result"])

			spr1, _ := sdk.Publish(node, ctx, messages.KitTeardownMsg{Source: "ts-calls-plugin-cross.ts"})
			sch1 := make(chan messages.KitTeardownResp, 1)
			sun1, _ := sdk.SubscribeTo[messages.KitTeardownResp](node, ctx, spr1.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { sch1 <- r })
			defer sun1()
			select {
			case <-sch1:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
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
			node, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel: brainkit.KernelConfig{
					Namespace: "ts-plugin-alongside-cross",
					CallerID:  "host",
					FSRoot:    tmpDir,
				},
				Messaging: brainkit.MessagingConfig{
					Transport: "nats",
					NATSURL:   natsURL,
					NATSName:  "brainkit-ts-plugin-alongside",
				},
				Plugins: []brainkit.PluginConfig{
					{
						Name:         "testplugin",
						Binary:       pluginBinary,
						StartTimeout: 30 * time.Second,
					},
				},
			})
			require.NoError(t, err)
			defer node.Close()

			err = node.Start(ctx)
			require.NoError(t, err)
			time.Sleep(2 * time.Second)

			// Deploy .ts tool
			pr3, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
				Source: "ts-alongside-cross.ts",
				Code: `
					const tsTool = createTool({
						id: "ts-side-tool",
						description: "a TS-side tool",
						execute: async () => ({ from: "ts" })
					});
					kit.register("tool", "ts-side-tool", tsTool);
				`,
			})
			require.NoError(t, err)
			ch3 := make(chan messages.KitDeployResp, 1)
			us3, _ := sdk.SubscribeTo[messages.KitDeployResp](node, ctx, pr3.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch3 <- r })
			defer us3()
			select {
			case <-ch3:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
			defer listCancel()

			pr4, err := sdk.Publish(node, listCtx, messages.ToolListMsg{})
			require.NoError(t, err)
			ch4 := make(chan messages.ToolListResp, 1)
			us4, err := sdk.SubscribeTo[messages.ToolListResp](node, ctx, pr4.ReplyTo, func(r messages.ToolListResp, m messages.Message) { ch4 <- r })
			require.NoError(t, err)
			defer us4()
			var resp messages.ToolListResp
			select {
			case resp = <-ch4:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			names := make(map[string]bool)
			for _, tool := range resp.Tools {
				names[tool.ShortName] = true
			}
			assert.True(t, names["echo"], "plugin echo tool")
			assert.True(t, names["concat"], "plugin concat tool")
			assert.True(t, names["ts-side-tool"], "TS-deployed tool")

			spr2, _ := sdk.Publish(node, ctx, messages.KitTeardownMsg{Source: "ts-alongside-cross.ts"})
			sch2 := make(chan messages.KitTeardownResp, 1)
			sun2, _ := sdk.SubscribeTo[messages.KitTeardownResp](node, ctx, spr2.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { sch2 <- r })
			defer sun2()
			select {
			case <-sch2:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
		})
	}
}

// crossKitStartNATS starts NATS with Podman environment setup.
func crossKitStartNATS(t *testing.T) string {
	t.Helper()
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	if os.Getenv("DOCKER_HOST") == "" {
		if out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output(); err == nil {
			os.Setenv("DOCKER_HOST", "unix://"+string(out[:len(out)-1]))
		}
	}

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
