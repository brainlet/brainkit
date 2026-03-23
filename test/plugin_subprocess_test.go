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

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestPlugin_Subprocess is a full e2e test:
// Podman NATS → Node → plugin subprocess → tool call → result.
func TestPlugin_Subprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess plugin test in short mode")
	}

	// Check Podman availability
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not available")
	}

	// Build the test plugin binary
	pluginBinary := filepath.Join(t.TempDir(), "testplugin")
	// Build from the module root so Go can resolve the module path
	moduleRoot := filepath.Join("..")
	buildCmd := exec.Command("go", "build", "-o", pluginBinary, "./test/testplugin/")
	buildCmd.Dir = moduleRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("build test plugin: %v", err)
	}
	t.Logf("Built test plugin: %s", pluginBinary)

	// NATS JetStream auto-provisioning for 48+ command topics is slow.
	// Allow up to 5 minutes for the full e2e lifecycle.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Configure testcontainers for Podman
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true") // Podman doesn't support Ryuk

	// Set DOCKER_HOST to Podman socket if not already set
	if os.Getenv("DOCKER_HOST") == "" {
		// Try to find Podman socket
		socketPaths := []string{
			"/var/folders/h2/lww7d7p5049dx4qzhxgk33640000gn/T/podman/podman-machine-default-api.sock",
		}
		// Also try `podman machine inspect`
		if out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output(); err == nil {
			socketPaths = append([]string{string(out[:len(out)-1])}, socketPaths...) // prepend
		}
		for _, sp := range socketPaths {
			if _, err := os.Stat(sp); err == nil {
				os.Setenv("DOCKER_HOST", "unix://"+sp)
				t.Logf("Set DOCKER_HOST to unix://%s", sp)
				break
			}
		}
	}

	// Start NATS via Podman
	natsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:latest",
			ExposedPorts: []string{"4222/tcp"},
			Cmd:          []string{"-js"}, // Enable JetStream
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("failed to start NATS container (podman may not be running): %v", err)
	}
	defer natsContainer.Terminate(ctx)

	natsHost, err := natsContainer.Host(ctx)
	require.NoError(t, err)
	natsPort, err := natsContainer.MappedPort(ctx, "4222")
	require.NoError(t, err)
	natsURL := fmt.Sprintf("nats://%s:%s", natsHost, natsPort.Port())
	t.Logf("NATS URL: %s", natsURL)

	// Verify NATS is actually accepting connections before proceeding
	natsReadyCtx, natsReadyCancel := context.WithTimeout(ctx, 15*time.Second)
	defer natsReadyCancel()
	for {
		conn, err := exec.CommandContext(natsReadyCtx, "nc", "-z", natsHost, natsPort.Port()).CombinedOutput()
		if err == nil {
			t.Log("NATS accepting connections")
			break
		}
		_ = conn
		select {
		case <-natsReadyCtx.Done():
			t.Fatalf("NATS never became ready: %v", natsReadyCtx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}

	tmpDir := t.TempDir()
	t.Log("Creating Node with NATS transport...")

	// Create Node with NATS transport and plugin config
	node, err := kit.NewNode(kit.NodeConfig{
		Kernel: kit.KernelConfig{
			Namespace:    "plugin-e2e",
			CallerID:     "host",
			WorkspaceDir: tmpDir,
		},
		Messaging: kit.MessagingConfig{
			Transport: "nats",
			NATSURL:   natsURL,
			NATSName:  "brainkit-test",
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

	// Register a host-side tool
	kit.RegisterTool(node.Kernel, "host-add", registry.TypedTool[addInput]{
		Description: "adds two numbers (host-side)",
		Execute: func(ctx context.Context, input addInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	// Start Node — this starts the plugin subprocess
	err = node.Start(ctx)
	require.NoError(t, err)

	// Wait a moment for the plugin to register its manifest
	time.Sleep(2 * time.Second)

	// --- Tests ---

	t.Run("PluginTool_Echo", func(t *testing.T) {
		toolCtx, toolCancel := context.WithTimeout(ctx, 10*time.Second)
		defer toolCancel()

		_pr1, err := sdk.Publish(node, toolCtx, messages.ToolCallMsg{
			Name:  "echo",
			Input: map[string]any{"message": "hello from host"},
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
		assert.Equal(t, "hello from host", result["echoed"])
		assert.Equal(t, "testplugin", result["plugin"])
	})

	t.Run("PluginTool_Concat", func(t *testing.T) {
		toolCtx, toolCancel := context.WithTimeout(ctx, 10*time.Second)
		defer toolCancel()

		_pr2, err := sdk.Publish(node, toolCtx, messages.ToolCallMsg{
			Name:  "concat",
			Input: map[string]any{"a": "foo", "b": "bar"},
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

		var result map[string]string
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "foobar", result["result"])
	})

	t.Run("HostTool_StillWorks", func(t *testing.T) {
		toolCtx, toolCancel := context.WithTimeout(ctx, 10*time.Second)
		defer toolCancel()

		_pr3, err := sdk.Publish(node, toolCtx, messages.ToolCallMsg{
			Name:  "host-add",
			Input: map[string]any{"a": 10, "b": 20},
		})
		require.NoError(t, err)
		_ch3 := make(chan messages.ToolCallResp, 1)
		_us3, err := sdk.SubscribeTo[messages.ToolCallResp](node, ctx, _pr3.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch3 <- r })
		require.NoError(t, err)
		defer _us3()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch3:
		case <-ctx.Done():
			t.Fatal("timeout")
		}

		var result map[string]int
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, 30, result["sum"])
	})

	t.Run("ToolsList_ShowsBoth", func(t *testing.T) {
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
		assert.True(t, names["echo"], "plugin echo tool should be listed")
		assert.True(t, names["concat"], "plugin concat tool should be listed")
		assert.True(t, names["host-add"], "host-side tool should be listed")
	})
}

