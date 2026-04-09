package cross

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// transportFieldsForBackend converts a testutil transport config into brainkit.Config transport fields.
type transportFields struct {
	Transport string
	NATSURL   string
	NATSName  string
	AMQPURL   string
	RedisURL  string
}

func transportFieldsForBackend(t *testing.T, backend string) transportFields {
	t.Helper()
	tcfg := testutil.TransportConfigForBackend(t, backend)
	return transportFields{
		Transport: tcfg.Type,
		NATSURL:   tcfg.NATSURL,
		NATSName:  tcfg.NATSName,
		AMQPURL:   tcfg.AMQPURL,
		RedisURL:  tcfg.RedisURL,
	}
}

// Run executes all cross-kit, plugin, and discovery tests.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("cross", func(t *testing.T) {
		// crosskit.go — TS<->Go cross-kit
		t.Run("ts_go/ts_deploys_tool_go_calls_it", func(t *testing.T) { testTSDeploysToolGoCallsIt(t, env) })
		t.Run("ts_go/go_registers_tool_ts_calls_via_deploy", func(t *testing.T) { testGoRegistersToolTSCallsViaDeploy(t, env) })
		t.Run("plugin_go/plugin_tool_called_from_go", func(t *testing.T) { testPluginToolCalledFromGo(t, env) })
		t.Run("plugin_go/go_tool_visible_in_list", func(t *testing.T) { testGoToolVisibleInList(t, env) })
		t.Run("ts_plugin/ts_calls_plugin_tool", func(t *testing.T) { testTSCallsPluginTool(t, env) })
		t.Run("ts_plugin/ts_deployed_tool_visible_alongside_plugin", func(t *testing.T) { testTSDeployedToolVisibleAlongsidePlugin(t, env) })

		// plugins.go — in-process + subprocess plugin tests
		t.Run("plugin_inprocess/list_tools", func(t *testing.T) { testPluginInProcessListTools(t, env) })
		t.Run("plugin_inprocess/call_tool", func(t *testing.T) { testPluginInProcessCallTool(t, env) })
		t.Run("plugin_inprocess/fs_write_read", func(t *testing.T) { testPluginInProcessFSWriteRead(t, env) })
		t.Run("plugin_inprocess/deploy_teardown", func(t *testing.T) { testPluginInProcessDeployTeardown(t, env) })
		t.Run("plugin_inprocess/async_subscribe", func(t *testing.T) { testPluginInProcessAsyncSubscribe(t, env) })
		t.Run("plugin_subprocess/echo", func(t *testing.T) { testPluginSubprocessEcho(t, env) })
		t.Run("plugin_subprocess/concat", func(t *testing.T) { testPluginSubprocessConcat(t, env) })
		t.Run("plugin_subprocess/host_tool_still_works", func(t *testing.T) { testPluginSubprocessHostToolStillWorks(t, env) })
		t.Run("plugin_subprocess/tools_list_shows_both", func(t *testing.T) { testPluginSubprocessToolsListShowsBoth(t, env) })

		// node_commands.go — Node command tests
		t.Run("node_commands/plugin_list", func(t *testing.T) { testNodeCommandsPluginList(t, env) })
		t.Run("node_commands/plugin_stop_nonexistent", func(t *testing.T) { testNodeCommandsPluginStopNonexistent(t, env) })
		t.Run("node_commands/plugin_restart_nonexistent", func(t *testing.T) { testNodeCommandsPluginRestartNonexistent(t, env) })
		t.Run("node_commands/plugin_status_nonexistent", func(t *testing.T) { testNodeCommandsPluginStatusNonexistent(t, env) })
		t.Run("node_commands/package_list_empty", func(t *testing.T) { testNodeCommandsPackageListEmpty(t, env) })
		t.Run("node_commands/deploy_on_node", func(t *testing.T) { testNodeCommandsDeployOnNode(t, env) })
		t.Run("node_commands/node_shutdown_clean", func(t *testing.T) { testNodeCommandsNodeShutdownClean(t, env) })

		// plugin_surface.go — Plugin surface tests
		t.Run("plugin_surface/go_tool_from_plugin", func(t *testing.T) { testPluginSurfaceGoToolFromPlugin(t, env) })
		t.Run("plugin_surface/ts_from_plugin", func(t *testing.T) { testPluginSurfaceTSFromPlugin(t, env) })
		t.Run("plugin_surface/tools_list", func(t *testing.T) { testPluginSurfaceToolsList(t, env) })
		t.Run("plugin_surface/error_code_from_node", func(t *testing.T) { testPluginSurfaceErrorCodeFromNode(t, env) })
		t.Run("plugin_surface/secrets_from_node", func(t *testing.T) { testPluginSurfaceSecretsFromNode(t, env) })
		t.Run("plugin_surface/deploy_from_node", func(t *testing.T) { testPluginSurfaceDeployFromNode(t, env) })

		// discovery.go — Discovery tests
		t.Run("discovery/static_peers", func(t *testing.T) { testDiscoveryStaticPeers(t, env) })
		t.Run("discovery/browse", func(t *testing.T) { testDiscoveryBrowse(t, env) })
		t.Run("discovery/register", func(t *testing.T) { testDiscoveryRegister(t, env) })
		t.Run("discovery/resolve_nonexistent", func(t *testing.T) { testDiscoveryResolveNonexistent(t, env) })
		t.Run("discovery/close", func(t *testing.T) { testDiscoveryClose(t, env) })
		t.Run("discovery/static_peers_bus", func(t *testing.T) { testDiscoveryStaticPeersBus(t, env) })

		// backend_matrix.go — ported from adversarial/crosskit_matrix_test.go
		t.Run("crosskit/publish_reply", func(t *testing.T) { testCrossKitPublishReply(t, env) })
		t.Run("crosskit/error_propagation", func(t *testing.T) { testCrossKitErrorPropagation(t, env) })
	})
}

// --- Shared helpers ---

// makeNode creates a Kit with NATS transport. Skips if Podman is unavailable.
func makeNode(t *testing.T, env *suite.TestEnv, namespace string) *brainkit.Kit {
	t.Helper()
	return makeNodeWithConfig(t, env, namespace, transportFieldsForBackend(t, "nats"))
}

// makeNodeWithConfig creates a Kit with explicit transport fields.
// Used by cross-Kit tests where multiple nodes must share the same transport.
func makeNodeWithConfig(t *testing.T, env *suite.TestEnv, namespace string, tf transportFields) *brainkit.Kit {
	t.Helper()
	env.RequirePodman(t)
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: namespace,
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: tf.Transport,
		NATSURL:   tf.NATSURL,
		NATSName:  tf.NATSName,
		AMQPURL:   tf.AMQPURL,
		RedisURL:  tf.RedisURL,
	})
	if err != nil {
		t.Fatalf("makeNode: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}

// startNATSContainer starts a NATS JetStream container and returns the URL.
func startNATSContainer(t *testing.T) string {
	t.Helper()
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	if os.Getenv("DOCKER_HOST") == "" {
		if out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output(); err == nil {
			sp := strings.TrimSpace(string(out))
			if _, statErr := os.Stat(sp); statErr == nil {
				os.Setenv("DOCKER_HOST", "unix://"+sp)
			}
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

// publishAndWaitRaw publishes on a Kit and waits for raw payload.
func publishAndWaitRaw(t *testing.T, kit *brainkit.Kit, ctx context.Context, msg sdk.BrainkitMessage) []byte {
	t.Helper()
	pr, err := sdk.Publish(kit, ctx, msg)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	ch := make(chan []byte, 1)
	unsub, err := kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	select {
	case p := <-ch:
		return p
	case <-ctx.Done():
		t.Fatal("timeout waiting for response")
		return nil
	}
}

// publishAndWaitJSON publishes on a Kit and returns the raw JSON payload.
func publishAndWaitJSON(t *testing.T, kit *brainkit.Kit, ctx context.Context, msg sdk.BrainkitMessage) json.RawMessage {
	t.Helper()
	return json.RawMessage(publishAndWaitRaw(t, kit, ctx, msg))
}
