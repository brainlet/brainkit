package cross_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCross_Plugin_Go(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping plugin tests in short mode")
	}

	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			testutil.RequiresNetworkTransport(t, backend)

			// Build testplugin binary
			pluginBinary := testutil.BuildTestPlugin(t)

			// For NATS, start a container
			var natsURL string
			if backend == "nats" {
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
				defer natsContainer.Terminate(ctx)

				host, _ := natsContainer.Host(ctx)
				port, _ := natsContainer.MappedPort(ctx, "4222")
				natsURL = fmt.Sprintf("nats://%s:%s", host, port.Port())
			} else {
				t.Skipf("plugin cross-surface test only implemented for NATS backend currently")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			tmpDir := t.TempDir()

			node, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel: brainkit.KernelConfig{
					Namespace:    "plugin-cross",
					CallerID:     "host",
					FSRoot: tmpDir,
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

			// Register a host-side tool
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

			// Wait for plugin manifest registration
			time.Sleep(2 * time.Second)

			t.Run("Plugin_tool_called_from_Go", func(t *testing.T) {
				toolCtx, toolCancel := context.WithTimeout(ctx, 10*time.Second)
				defer toolCancel()

				_pr1, err := sdk.Publish(node, toolCtx, messages.ToolCallMsg{
					Name:  "echo",
					Input: map[string]any{"message": "plugin→go test"},
				})
				require.NoError(t, err)
				_ch1 := make(chan messages.ToolCallResp, 1)
				_us1, err := sdk.SubscribeTo[messages.ToolCallResp](node, ctx, _pr1.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "plugin→go test", result["echoed"])
				assert.Equal(t, "testplugin", result["plugin"])
			})

			t.Run("Go_tool_visible_in_list", func(t *testing.T) {
				listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
				defer listCancel()

				_pr2, err := sdk.Publish(node, listCtx, messages.ToolListMsg{})
				require.NoError(t, err)
				_ch2 := make(chan messages.ToolListResp, 1)
				_us2, err := sdk.SubscribeTo[messages.ToolListResp](node, ctx, _pr2.ReplyTo, func(r messages.ToolListResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var resp messages.ToolListResp
				select {
				case resp = <-_ch2:
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
		})
	}
}
