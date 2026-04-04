package deploy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTSNamespaceIsolation — deployed .ts services have isolated namespaces.
func testTSNamespaceIsolation(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy two services with same handler topic name
	pr1, err := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "ns-a-deploy-adv.ts",
		Code: `
			bus.on("greet", async (msg) => {
				msg.reply({ from: "service-a" });
			});
		`,
	})
	require.NoError(t, err)
	ch1 := make(chan messages.KitDeployResp, 1)
	unsub1, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch1 <- r })
	defer unsub1()
	select {
	case resp := <-ch1:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying ns-a")
	}

	pr2, err := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "ns-b-deploy-adv.ts",
		Code: `
			bus.on("greet", async (msg) => {
				msg.reply({ from: "service-b" });
			});
		`,
	})
	require.NoError(t, err)
	ch2 := make(chan messages.KitDeployResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr2.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch2 <- r })
	defer unsub2()
	select {
	case resp := <-ch2:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying ns-b")
	}

	time.Sleep(100 * time.Millisecond)

	// Send to service A — should get reply from A, not B
	pr3, err := sdk.SendToService(env.Kernel, ctx, "ns-a-deploy-adv.ts", "greet", json.RawMessage(`{}`))
	require.NoError(t, err)

	replyCh := make(chan messages.Message, 1)
	unsub3, err := env.Kernel.SubscribeRaw(ctx, pr3.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			select {
			case replyCh <- msg:
			default:
			}
		}
	})
	require.NoError(t, err)
	defer unsub3()

	select {
	case msg := <-replyCh:
		var result map[string]string
		json.Unmarshal(msg.Payload, &result)
		assert.Equal(t, "service-a", result["from"], "should get reply from service A, not B")
	case <-ctx.Done():
		t.Fatal("timeout waiting for namespace isolation reply")
	}

	sdk.Publish(env.Kernel, ctx, messages.KitTeardownMsg{Source: "ns-a-deploy-adv.ts"})
	sdk.Publish(env.Kernel, ctx, messages.KitTeardownMsg{Source: "ns-b-deploy-adv.ts"})
}

// testTSModuleImports — verify the 4-module import system works from deployed .ts code.
func testTSModuleImports(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "surface-imports-deploy-adv.ts",
		Code: `
			var checks = {
				hasBus: typeof bus === "object" && typeof bus.publish === "function",
				hasKit: typeof kit === "object" && typeof kit.register === "function",
				hasModel: typeof model === "function",
				hasTools: typeof tools === "object" && typeof tools.call === "function",
				hasFs: typeof fs === "object" && typeof fs.promises === "object" && typeof fs.promises.readFile === "function",
				hasMcp: typeof mcp === "object",
				hasOutput: typeof output === "function",
				hasRegistry: typeof registry === "object",
			};
			output(checks);
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying module imports check")
	}

	result, err := env.Kernel.EvalTS(ctx, "__read_imports_adv.ts", `return globalThis.__module_result || "null"`)
	require.NoError(t, err)

	var checks map[string]bool
	require.NoError(t, json.Unmarshal([]byte(result), &checks))
	assert.True(t, checks["hasBus"], "bus should be available")
	assert.True(t, checks["hasKit"], "kit should be available")
	assert.True(t, checks["hasModel"], "model should be available")
	assert.True(t, checks["hasTools"], "tools should be available")
	assert.True(t, checks["hasFs"], "fs should be available")
	assert.True(t, checks["hasMcp"], "mcp should be available")
	assert.True(t, checks["hasOutput"], "output should be available")
	assert.True(t, checks["hasRegistry"], "registry should be available")

	sdk.Publish(env.Kernel, ctx, messages.KitTeardownMsg{Source: "surface-imports-deploy-adv.ts"})
}

// testTSFileExtensionHandling — deploy .js vs .ts file extension handling.
func testTSFileExtensionHandling(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// .ts should work (transpiled)
	_, err := env.Kernel.Deploy(ctx, "ext-ts-deploy-adv.ts", `
		const typed: string = "ts works";
		output({ result: typed });
	`)
	require.NoError(t, err)

	// .js should work (executed directly)
	_, err = env.Kernel.Deploy(ctx, "ext-js-deploy-adv.js", `output("js works");`)
	require.NoError(t, err)

	result, _ := env.Kernel.EvalTS(ctx, "__read_ext_adv.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "js works", result)

	env.Kernel.Teardown(ctx, "ext-ts-deploy-adv.ts")
	env.Kernel.Teardown(ctx, "ext-js-deploy-adv.js")
}
