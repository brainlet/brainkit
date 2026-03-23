package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

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

	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			requiresNetworkTransport(t, backend)

			if backend != "nats" {
				t.Skipf("TS↔Plugin cross-surface currently tested on NATS only")
			}

			pluginBinary := filepath.Join(t.TempDir(), "testplugin")
			buildCmd := exec.Command("go", "build", "-o", pluginBinary, "./test/testplugin/")
			buildCmd.Dir = filepath.Join("..")
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				t.Fatalf("build test plugin: %v", err)
			}

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
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](node, ctx, messages.KitDeployMsg{
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
					`,
				})
				require.NoError(t, err)

				callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
				defer callCancel()

				resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](node, callCtx, messages.ToolCallMsg{
					Name:  "plugin-caller",
					Input: map[string]any{"x": "foo", "y": "bar"},
				})
				require.NoError(t, err)

				var result map[string]any
				json.Unmarshal(resp.Result, &result)
				inner, _ := result["fromPlugin"].(map[string]any)
				assert.Equal(t, "foobar", inner["result"])

				sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](node, ctx, messages.KitTeardownMsg{Source: "ts-calls-plugin.ts"})
			})

			t.Run("TS_deployed_tool_visible_alongside_plugin", func(t *testing.T) {
				// Deploy .ts tool
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](node, ctx, messages.KitDeployMsg{
					Source: "ts-alongside.ts",
					Code: `
						const tsTool = createTool({
							id: "ts-side-tool",
							description: "a TS-side tool",
							execute: async () => ({ from: "ts" })
						});
					`,
				})
				require.NoError(t, err)

				listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
				defer listCancel()

				resp, err := sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](node, listCtx, messages.ToolListMsg{})
				require.NoError(t, err)

				names := make(map[string]bool)
				for _, tool := range resp.Tools {
					names[tool.ShortName] = true
				}
				assert.True(t, names["echo"], "plugin echo tool")
				assert.True(t, names["concat"], "plugin concat tool")
				assert.True(t, names["ts-side-tool"], "TS-deployed tool")

				sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](node, ctx, messages.KitTeardownMsg{Source: "ts-alongside.ts"})
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
