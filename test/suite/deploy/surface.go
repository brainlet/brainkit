package deploy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTSNamespaceIsolation — deployed .ts services have isolated namespaces.
func testTSNamespaceIsolation(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy two services with same handler topic name
	pr1, err := sdk.Publish(env.Kit, ctx, pkgDeploy("ns-a-deploy-adv.ts", `
			bus.on("greet", async (msg) => {
				msg.reply({ from: "service-a" });
			});
		`))
	require.NoError(t, err)
	ch1 := make(chan sdk.PackageDeployResp, 1)
	unsub1, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr1.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch1 <- r })
	defer unsub1()
	select {
	case resp := <-ch1:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying ns-a")
	}

	pr2, err := sdk.Publish(env.Kit, ctx, pkgDeploy("ns-b-deploy-adv.ts", `
			bus.on("greet", async (msg) => {
				msg.reply({ from: "service-b" });
			});
		`))
	require.NoError(t, err)
	ch2 := make(chan sdk.PackageDeployResp, 1)
	unsub2, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr2.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch2 <- r })
	defer unsub2()
	select {
	case resp := <-ch2:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying ns-b")
	}

	time.Sleep(100 * time.Millisecond)

	// Send to service A — should get reply from A, not B
	pr3, err := sdk.SendToService(env.Kit, ctx, "ns-a-deploy-adv.ts", "greet", json.RawMessage(`{}`))
	require.NoError(t, err)

	replyCh := make(chan sdk.Message, 1)
	unsub3, err := env.Kit.SubscribeRaw(ctx, pr3.ReplyTo, func(msg sdk.Message) {
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
		json.Unmarshal(suite.ResponseDataFromMsg(msg), &result)
		assert.Equal(t, "service-a", result["from"], "should get reply from service A, not B")
	case <-ctx.Done():
		t.Fatal("timeout waiting for namespace isolation reply")
	}

	sdk.Publish(env.Kit, ctx, pkgTeardown("ns-a-deploy-adv.ts"))
	sdk.Publish(env.Kit, ctx, pkgTeardown("ns-b-deploy-adv.ts"))
}

// testTSModuleImports — verify the 4-module import system works from deployed .ts code.
func testTSModuleImports(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeploy("surface-imports-deploy-adv.ts", `
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
		`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying module imports check")
	}

	result := testutil.EvalTS(t, env.Kit, "__read_imports_adv.ts", `return globalThis.__module_result || "null"`)

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

	sdk.Publish(env.Kit, ctx, pkgTeardown("surface-imports-deploy-adv.ts"))
}

// testTSAgentEndowments — verify Mastra endowments (createTool, createStep, createWorkflow, z) are available.
func testTSAgentEndowments(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeploy("surface-agent-endowments.ts", `
			var checks = {
				hasAgent: typeof Agent === "function",
				hasCreateTool: typeof createTool === "function",
				hasCreateWorkflow: typeof createWorkflow === "function",
				hasCreateStep: typeof createStep === "function",
				hasZ: typeof z === "object" && typeof z.object === "function",
			};
			output(checks);
		`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying agent endowments check")
	}

	result := testutil.EvalTS(t, env.Kit, "__read_agent_endowments.ts", `return globalThis.__module_result || "null"`)

	var checks map[string]bool
	require.NoError(t, json.Unmarshal([]byte(result), &checks))
	assert.True(t, checks["hasCreateTool"], "createTool should be available")
	assert.True(t, checks["hasCreateStep"], "createStep should be available")
	assert.True(t, checks["hasCreateWorkflow"], "createWorkflow should be available")
	assert.True(t, checks["hasZ"], "z should be available")

	sdk.Publish(env.Kit, ctx, pkgTeardown("surface-agent-endowments.ts"))
}

// testTSAISDKEndowments — verify AI SDK endowments (model, generateText) are available.
func testTSAISDKEndowments(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeploy("surface-ai-endowments.ts", `
			var checks = {
				hasModel: typeof model === "function",
				hasGenerateText: typeof generateText === "function",
				hasStreamText: typeof streamText === "function",
				hasGenerateObject: typeof generateObject === "function",
			};
			output(checks);
		`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed)
	case <-ctx.Done():
		t.Fatal("timeout deploying AI SDK endowments check")
	}

	result := testutil.EvalTS(t, env.Kit, "__read_ai_endowments.ts", `return globalThis.__module_result || "null"`)

	var checks map[string]bool
	require.NoError(t, json.Unmarshal([]byte(result), &checks))
	assert.True(t, checks["hasModel"], "model should be available")
	assert.True(t, checks["hasGenerateText"], "generateText should be available")

	sdk.Publish(env.Kit, ctx, pkgTeardown("surface-ai-endowments.ts"))
}

// testTSDeployWithTool — deploy .ts that creates a tool via createTool + kit.register, then call it from Go.
func testTSDeployWithTool(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeploy("surface-tool-deploy.ts", `
			const calc = createTool({
				id: "surface-calc-deploy",
				description: "adds two numbers (surface test)",
				execute: async ({ context: input }) => {
					var a = (input && input.a) || 0;
					var b = (input && input.b) || 0;
					return { sum: a + b, source: "ts-surface" };
				},
			});
			kit.register("tool", "surface-calc-deploy", calc);
		`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying tool")
	}

	// Call the tool from Go
	pr2, err := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{Name: "surface-calc-deploy", Input: map[string]any{"a": 10, "b": 32}})
	require.NoError(t, err)
	ch2 := make(chan sdk.ToolCallResp, 1)
	mch2 := make(chan sdk.Message, 1)
	unsub2, _ := sdk.SubscribeTo[sdk.ToolCallResp](env.Kit, ctx, pr2.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) { ch2 <- r; mch2 <- m })
	defer unsub2()
	var resp sdk.ToolCallResp
	select {
	case resp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout calling tool")
	}
	m2 := <-mch2
	require.Empty(t, suite.ResponseErrorMessage(m2.Payload))

	var result map[string]any
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Equal(t, float64(42), result["sum"])
	assert.Equal(t, "ts-surface", result["source"])

	sdk.Publish(env.Kit, ctx, pkgTeardown("surface-tool-deploy.ts"))
}

// testTSDeployWithWorkflow — deploy .ts that creates a Mastra workflow, runs it, and outputs the result.
func testTSDeployWithWorkflow(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeploy("surface-workflow-deploy.ts", `
			const step1 = createStep({
				id: "uppercase-deploy",
				inputSchema: z.object({ text: z.string() }),
				outputSchema: z.object({ upper: z.string() }),
				execute: async ({ inputData }) => {
					return { upper: inputData.text.toUpperCase() };
				},
			});

			const step2 = createStep({
				id: "exclaim-deploy",
				inputSchema: z.object({ upper: z.string() }),
				outputSchema: z.object({ result: z.string() }),
				execute: async ({ inputData }) => {
					return { result: inputData.upper + "!!!" };
				},
			});

			const wf = createWorkflow({
				id: "surface-wf-deploy",
				inputSchema: z.object({ text: z.string() }),
				outputSchema: z.object({ result: z.string() }),
			}).then(step1).then(step2).commit();

			const run = await wf.createRun();
			const result = await run.start({ inputData: { text: "deploy test" } });
			output({ status: result.status, result: result.result });
		`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying workflow")
	}

	result := testutil.EvalTS(t, env.Kit, "__read_wf_deploy.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, "success", parsed["status"])
	if r, ok := parsed["result"].(map[string]any); ok {
		assert.Equal(t, "DEPLOY TEST!!!", r["result"])
	}

	sdk.Publish(env.Kit, ctx, pkgTeardown("surface-workflow-deploy.ts"))
}

// testTSDeployWithBusService — deploy .ts as a bus service with bus.on, Go sends message, .ts replies.
func testTSDeployWithBusService(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeploy("surface-service-deploy.ts", `
			bus.on("greet", async (msg) => {
				const name = msg.payload && msg.payload.name ? msg.payload.name : "world";
				msg.reply({ greeting: "hello " + name + " from ts service" });
			});
		`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying service")
	}

	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.SendToService(env.Kit, ctx, "surface-service-deploy.ts", "greet", json.RawMessage(`{"name":"Go"}`))
	require.NoError(t, err)

	replyCh := make(chan sdk.Message, 1)
	unsub2, err := sdk.SubscribeTo[json.RawMessage](env.Kit, ctx, pr2.ReplyTo, func(payload json.RawMessage, msg sdk.Message) {
		replyCh <- msg
	})
	require.NoError(t, err)
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]string
		json.Unmarshal(suite.ResponseDataFromMsg(msg), &result)
		assert.Equal(t, "hello Go from ts service", result["greeting"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for service reply")
	}

	sdk.Publish(env.Kit, ctx, pkgTeardown("surface-service-deploy.ts"))
}

// testTSDeployWithStreaming — deploy .ts service with streaming chunks then final reply.
func testTSDeployWithStreaming(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeploy("surface-streamer-deploy.ts", `
			bus.on("stream", async (msg) => {
				msg.send({ chunk: "one" });
				msg.send({ chunk: "two" });
				msg.send({ chunk: "three" });
				msg.reply({ done: true, count: 3 });
			});
		`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying streamer")
	}

	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.SendToService(env.Kit, ctx, "surface-streamer-deploy.ts", "stream", json.RawMessage(`{}`))
	require.NoError(t, err)

	finalCh := make(chan sdk.Message, 1)
	unsub2, err := env.Kit.SubscribeRaw(ctx, pr2.ReplyTo, func(msg sdk.Message) {
		if msg.Metadata["done"] == "true" {
			select {
			case finalCh <- msg:
			default:
			}
		}
	})
	require.NoError(t, err)
	defer unsub2()

	select {
	case final := <-finalCh:
		assert.Equal(t, "true", final.Metadata["done"])
		var finalPayload map[string]any
		json.Unmarshal(suite.ResponseDataFromMsg(final), &finalPayload)
		assert.Equal(t, true, finalPayload["done"])
		assert.Equal(t, float64(3), finalPayload["count"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for streaming completion")
	}

	sdk.Publish(env.Kit, ctx, pkgTeardown("surface-streamer-deploy.ts"))
}

// testTSFileExtensionHandling — deploy .js vs .ts file extension handling.
func testTSFileExtensionHandling(t *testing.T, env *suite.TestEnv) {
	// .ts should work (transpiled)
	testutil.Deploy(t, env.Kit, "ext-ts-deploy-adv.ts", `
		const typed: string = "ts works";
		output({ result: typed });
	`)

	// .js should work (executed directly)
	testutil.Deploy(t, env.Kit, "ext-js-deploy-adv.js", `output("js works");`)

	result := testutil.EvalTS(t, env.Kit, "__read_ext_adv.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "js works", result)

	testutil.Teardown(t, env.Kit, "ext-ts-deploy-adv.ts")
	testutil.Teardown(t, env.Kit, "ext-js-deploy-adv.js")
}
