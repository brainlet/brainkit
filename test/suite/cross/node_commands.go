package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Node command tests (from test/adversarial/node_commands_test.go) ---

func testNodeCommandsPluginList(t *testing.T, env *suite.TestEnv) {
	node := makeNode(t, env, "node-pluglist-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitRaw(t, node, ctx, messages.PluginListRunningMsg{})
	assert.Contains(t, string(p), "plugins")
}

func testNodeCommandsPluginStopNonexistent(t *testing.T, env *suite.TestEnv) {
	node := makeNode(t, env, "node-plugstop-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitJSON(t, node, ctx, messages.PluginStopMsg{Name: "ghost-plugin"})
	assert.True(t, suite.ResponseHasError(p), "stopping nonexistent plugin should error")
}

func testNodeCommandsPluginRestartNonexistent(t *testing.T, env *suite.TestEnv) {
	node := makeNode(t, env, "node-plugrestart-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitJSON(t, node, ctx, messages.PluginRestartMsg{Name: "ghost-plugin"})
	assert.True(t, suite.ResponseHasError(p))
}

func testNodeCommandsPluginStatusNonexistent(t *testing.T, env *suite.TestEnv) {
	node := makeNode(t, env, "node-plugstatus-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitJSON(t, node, ctx, messages.PluginStatusMsg{Name: "ghost-plugin"})
	assert.True(t, suite.ResponseHasError(p))
}

func testNodeCommandsPluginStateGetSet(t *testing.T, env *suite.TestEnv) {
	node := makeNode(t, env, "node-plugstate-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set state
	p1 := publishAndWaitRaw(t, node, ctx, messages.PluginStateSetMsg{Key: "test-key", Value: "test-value"})
	assert.Contains(t, string(p1), "ok")

	// Get state
	p2 := publishAndWaitRaw(t, node, ctx, messages.PluginStateGetMsg{Key: "test-key"})
	assert.Contains(t, string(p2), "test-value")
}

func testNodeCommandsPackageListEmpty(t *testing.T, env *suite.TestEnv) {
	node := makeNode(t, env, "node-pkglist-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitRaw(t, node, ctx, messages.PackageListDeployedMsg{})
	assert.Contains(t, string(p), "packages")
}

func testNodeCommandsDeployOnNode(t *testing.T, env *suite.TestEnv) {
	node := makeNode(t, env, "node-deploy-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy
	_, err := node.Kernel.Deploy(ctx, "node-deploy-cross.ts", `
		bus.on("hello", function(msg) { msg.reply({from: "node"}); });
	`)
	require.NoError(t, err)

	// Call
	pr, _ := sdk.Publish(node, ctx, messages.CustomMsg{
		Topic: "ts.node-deploy-cross.hello", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := node.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "node")
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Teardown
	_, err = node.Kernel.Teardown(ctx, "node-deploy-cross.ts")
	require.NoError(t, err)
}

func testNodeCommandsNodeShutdownClean(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)
	msgCfg := messagingCfgForBackend(t, "nats")
	tmpDir := t.TempDir()

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel:    brainkit.KernelConfig{Namespace: "shutdown-test-cross", CallerID: "host", FSRoot: tmpDir},
		Messaging: msgCfg,
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))

	node.Kernel.Deploy(context.Background(), "shutdown-svc-cross.ts", `
		bus.on("ping", function(msg) { msg.reply({ok:true}); });
	`)

	err = node.Close()
	assert.NoError(t, err)
}
