package deploy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTSNamespaceIsolation — deployed .ts services have isolated namespaces.
func testTSNamespaceIsolation(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy two services with same handler topic name
	testutil.Deploy(t, env.Kit, "ns-a-deploy-adv.ts", `
			bus.on("greet", async (msg) => {
				msg.reply({ from: "service-a" });
			});
		`)
	testutil.Deploy(t, env.Kit, "ns-b-deploy-adv.ts", `
			bus.on("greet", async (msg) => {
				msg.reply({ from: "service-b" });
			});
		`)

	time.Sleep(100 * time.Millisecond)

	// Send to service A — should get reply from A, not B
	reply, err := sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   sdk.ResolveServiceTopic("ns-a-deploy-adv.ts", "greet"),
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	var result map[string]string
	json.Unmarshal(reply, &result)
	assert.Equal(t, "service-a", result["from"], "should get reply from service A, not B")

	testutil.Teardown(t, env.Kit, "ns-a-deploy-adv.ts")
	testutil.Teardown(t, env.Kit, "ns-b-deploy-adv.ts")
}

// testTSModuleImports — verify the 4-module import system works from deployed .ts code.
func testTSModuleImports(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "surface-imports-deploy-adv.ts", `
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
		`)

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

	testutil.Teardown(t, env.Kit, "surface-imports-deploy-adv.ts")
}

// testTSAgentEndowments — verify Mastra endowments (createTool, createStep, createWorkflow, z) are available.
func testTSAgentEndowments(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "surface-agent-endowments.ts", `
			var checks = {
				hasAgent: typeof Agent === "function",
				hasCreateTool: typeof createTool === "function",
				hasCreateWorkflow: typeof createWorkflow === "function",
				hasCreateStep: typeof createStep === "function",
				hasZ: typeof z === "object" && typeof z.object === "function",
			};
			output(checks);
		`)

	result := testutil.EvalTS(t, env.Kit, "__read_agent_endowments.ts", `return globalThis.__module_result || "null"`)

	var checks map[string]bool
	require.NoError(t, json.Unmarshal([]byte(result), &checks))
	assert.True(t, checks["hasCreateTool"], "createTool should be available")
	assert.True(t, checks["hasCreateStep"], "createStep should be available")
	assert.True(t, checks["hasCreateWorkflow"], "createWorkflow should be available")
	assert.True(t, checks["hasZ"], "z should be available")

	testutil.Teardown(t, env.Kit, "surface-agent-endowments.ts")
}

// testTSAISDKEndowments — verify AI SDK endowments (model, generateText) are available.
func testTSAISDKEndowments(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "surface-ai-endowments.ts", `
			var checks = {
				hasModel: typeof model === "function",
				hasGenerateText: typeof generateText === "function",
				hasStreamText: typeof streamText === "function",
				hasGenerateObject: typeof generateObject === "function",
			};
			output(checks);
		`)

	result := testutil.EvalTS(t, env.Kit, "__read_ai_endowments.ts", `return globalThis.__module_result || "null"`)

	var checks map[string]bool
	require.NoError(t, json.Unmarshal([]byte(result), &checks))
	assert.True(t, checks["hasModel"], "model should be available")
	assert.True(t, checks["hasGenerateText"], "generateText should be available")

	testutil.Teardown(t, env.Kit, "surface-ai-endowments.ts")
}

// testTSDeployWithTool — deploy .ts that creates a tool via createTool + kit.register, then call it from Go.
func testTSDeployWithTool(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "surface-tool-deploy.ts", `
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
		`)

	// Call the tool from Go
	resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](env.Kit, ctx, toolmsg.ToolCallMsg{Name: "surface-calc-deploy", Input: map[string]any{"a": 10, "b": 32}})
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Equal(t, float64(42), result["sum"])
	assert.Equal(t, "ts-surface", result["source"])

	testutil.Teardown(t, env.Kit, "surface-tool-deploy.ts")
}

// testTSDeployWithWorkflow — deploy .ts that creates a Mastra workflow, runs it, and outputs the result.
func testTSDeployWithWorkflow(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "surface-workflow-deploy.ts", `
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
		`)

	result := testutil.EvalTS(t, env.Kit, "__read_wf_deploy.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, "success", parsed["status"])
	if r, ok := parsed["result"].(map[string]any); ok {
		assert.Equal(t, "DEPLOY TEST!!!", r["result"])
	}

	testutil.Teardown(t, env.Kit, "surface-workflow-deploy.ts")
}

// testTSDeployWithBusService — deploy .ts as a bus service with bus.on, Go sends message, .ts replies.
func testTSDeployWithBusService(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "surface-service-deploy.ts", `
			bus.on("greet", async (msg) => {
				const name = msg.payload && msg.payload.name ? msg.payload.name : "world";
				msg.reply({ greeting: "hello " + name + " from ts service" });
			});
		`)

	time.Sleep(100 * time.Millisecond)

	reply, err := sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   sdk.ResolveServiceTopic("surface-service-deploy.ts", "greet"),
		Payload: json.RawMessage(`{"name":"Go"}`),
	})
	require.NoError(t, err)

	var result map[string]string
	json.Unmarshal(reply, &result)
	assert.Equal(t, "hello Go from ts service", result["greeting"])

	testutil.Teardown(t, env.Kit, "surface-service-deploy.ts")
}

// testTSDeployWithStreaming — deploy .ts service with streaming chunks then final reply.
func testTSDeployWithStreaming(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "surface-streamer-deploy.ts", `
			bus.on("stream", async (msg) => {
				msg.send({ chunk: "one" });
				msg.send({ chunk: "two" });
				msg.send({ chunk: "three" });
				msg.reply({ done: true, count: 3 });
			});
		`)

	time.Sleep(100 * time.Millisecond)

	final, err := sdk.CallStream[sdk.CustomMsg, json.RawMessage, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   sdk.ResolveServiceTopic("surface-streamer-deploy.ts", "stream"),
		Payload: json.RawMessage(`{}`),
	}, func(json.RawMessage) error { return nil })
	require.NoError(t, err)

	var finalPayload map[string]any
	json.Unmarshal(final, &finalPayload)
	assert.Equal(t, true, finalPayload["done"])
	assert.Equal(t, float64(3), finalPayload["count"])

	testutil.Teardown(t, env.Kit, "surface-streamer-deploy.ts")
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
