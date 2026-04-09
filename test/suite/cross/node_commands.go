package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Node command tests (from test/adversarial/node_commands_test.go) ---

func testNodeCommandsPluginList(t *testing.T, env *suite.TestEnv) {
	kit := makeNode(t, env, "node-pluglist-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitRaw(t, kit, ctx, sdk.PluginListRunningMsg{})
	assert.Contains(t, string(p), "plugins")
}

func testNodeCommandsPluginStopNonexistent(t *testing.T, env *suite.TestEnv) {
	kit := makeNode(t, env, "node-plugstop-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitJSON(t, kit, ctx, sdk.PluginStopMsg{Name: "ghost-plugin"})
	assert.True(t, suite.ResponseHasError(p), "stopping nonexistent plugin should error")
}

func testNodeCommandsPluginRestartNonexistent(t *testing.T, env *suite.TestEnv) {
	kit := makeNode(t, env, "node-plugrestart-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitJSON(t, kit, ctx, sdk.PluginRestartMsg{Name: "ghost-plugin"})
	assert.True(t, suite.ResponseHasError(p))
}

func testNodeCommandsPluginStatusNonexistent(t *testing.T, env *suite.TestEnv) {
	kit := makeNode(t, env, "node-plugstatus-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitJSON(t, kit, ctx, sdk.PluginStatusMsg{Name: "ghost-plugin"})
	assert.True(t, suite.ResponseHasError(p))
}

func testNodeCommandsPackageListEmpty(t *testing.T, env *suite.TestEnv) {
	kit := makeNode(t, env, "node-pkglist-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := publishAndWaitRaw(t, kit, ctx, sdk.PackageListDeployedMsg{})
	assert.Contains(t, string(p), "packages")
}

func testNodeCommandsDeployOnNode(t *testing.T, env *suite.TestEnv) {
	kit := makeNode(t, env, "node-deploy-cross")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy via bus command
	testutil.Deploy(t, kit, "node-deploy-cross.ts", `
		bus.on("hello", function(msg) { msg.reply({from: "node"}); });
	`)

	// Call
	pr, _ := sdk.Publish(kit, ctx, sdk.CustomMsg{
		Topic: "ts.node-deploy-cross.hello", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "node")
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Teardown
	testutil.Teardown(t, kit, "node-deploy-cross.ts")
}

func testNodeCommandsNodeShutdownClean(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "shutdown-test-cross",
		CallerID:  "host",
		FSRoot:    tmpDir,
		Transport: "embedded",
	})
	require.NoError(t, err)

	testutil.Deploy(t, kit, "shutdown-svc-cross.ts", `
		bus.on("ping", function(msg) { msg.reply({ok:true}); });
	`)

	err = kit.Close()
	assert.NoError(t, err)
}
