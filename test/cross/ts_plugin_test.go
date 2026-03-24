package cross_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestCross_TS_Plugin tests .ts code calling plugin-registered tools and
// plugin registering tools that .ts-deployed code can invoke.
func TestCross_TS_Plugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping plugin tests in short mode")
	}

	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			testutil.RequiresNetworkTransport(t, backend)

			if backend != "nats" {
				t.Skipf("TS↔Plugin cross-surface currently tested on NATS only")
			}

			pluginBinary := testutil.BuildTestPlugin(t)

			natsURL := startNATSContainer(t)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			tmpDir := t.TempDir()
			node, err := kit.NewNode(kit.NodeConfig{
				Kernel: kit.KernelConfig{
					Namespace:    "ts-plugin-cross",
					CallerID:     "host",
					WorkspaceDir: tmpDir,
				},
				Messaging: kit.MessagingConfig{
					Transport: "nats",
					NATSURL:   natsURL,
					NATSName:  "brainkit-ts-plugin",
				},
				Plugins: []kit.PluginConfig{
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

			t.Run("TS_calls_plugin_tool", func(t *testing.T) {
				// Deploy .ts that creates a wrapper tool calling the plugin's "concat" tool
				_pr1, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
					Source: "ts-calls-plugin.ts",
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
				_ch1 := make(chan messages.KitDeployResp, 1)
				_us1, _ := sdk.SubscribeTo[messages.KitDeployResp](node, ctx, _pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch1 <- r })
				defer _us1()
				select {
				case <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
				defer callCancel()

				_pr2, err := sdk.Publish(node, callCtx, messages.ToolCallMsg{
					Name:  "plugin-caller",
					Input: map[string]any{"x": "foo", "y": "bar"},
				})
				require.NoError(t, err)
				_ch2 := make(chan messages.ToolCallResp, 1)
				_us2, err := sdk.SubscribeTo[messages.ToolCallResp](node, ctx, _pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				var result map[string]any
				json.Unmarshal(resp.Result, &result)
				inner, _ := result["fromPlugin"].(map[string]any)
				assert.Equal(t, "foobar", inner["result"])

				_spr1, _ := sdk.Publish(node, ctx, messages.KitTeardownMsg{Source: "ts-calls-plugin.ts"})
				_sch1 := make(chan messages.KitTeardownResp, 1)
				_sun1, _ := sdk.SubscribeTo[messages.KitTeardownResp](node, ctx, _spr1.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _sch1 <- r })
				defer _sun1()
				select { case <-_sch1: case <-ctx.Done(): t.Fatal("timeout") }
			})

			t.Run("TS_deployed_tool_visible_alongside_plugin", func(t *testing.T) {
				// Deploy .ts tool
				_pr3, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
					Source: "ts-alongside.ts",
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
				_ch3 := make(chan messages.KitDeployResp, 1)
				_us3, _ := sdk.SubscribeTo[messages.KitDeployResp](node, ctx, _pr3.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch3 <- r })
				defer _us3()
				select {
				case <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
				defer listCancel()

				_pr4, err := sdk.Publish(node, listCtx, messages.ToolListMsg{})
				require.NoError(t, err)
				_ch4 := make(chan messages.ToolListResp, 1)
				_us4, err := sdk.SubscribeTo[messages.ToolListResp](node, ctx, _pr4.ReplyTo, func(r messages.ToolListResp, m messages.Message) { _ch4 <- r })
				require.NoError(t, err)
				defer _us4()
				var resp messages.ToolListResp
				select {
				case resp = <-_ch4:
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

				_spr2, _ := sdk.Publish(node, ctx, messages.KitTeardownMsg{Source: "ts-alongside.ts"})
				_sch2 := make(chan messages.KitTeardownResp, 1)
				_sun2, _ := sdk.SubscribeTo[messages.KitTeardownResp](node, ctx, _spr2.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _sch2 <- r })
				defer _sun2()
				select { case <-_sch2: case <-ctx.Done(): t.Fatal("timeout") }
			})
		})
	}
}

// startNATSContainer starts a NATS JetStream container and returns the URL.
func startNATSContainer(t *testing.T) string {
	t.Helper()
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	if os.Getenv("DOCKER_HOST") == "" {
		if out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output(); err == nil {
			os.Setenv("DOCKER_HOST", "unix://"+string(out[:len(out)-1]))
		}
	}

	natsContainer, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
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

	host, _ := natsContainer.Host(context.Background())
	port, _ := natsContainer.MappedPort(context.Background(), "4222")
	return fmt.Sprintf("nats://%s:%s", host, port.Port())
}
