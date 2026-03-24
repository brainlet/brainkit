package surface_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTS_ModuleImports verifies the 4-module import system works from deployed .ts code.
// Tests that "kit", "ai", "agent" modules export the correct symbols.
func TestTS_ModuleImports(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("KitModule_ExportsInfrastructure", func(t *testing.T) {
		// Deploy .ts that verifies kit module exports are available as endowments
		pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
			Source: "surface-kit-imports.ts",
			Code: `
				// In Compartment, endowments are globals (not ES module imports)
				// Verify all kit infrastructure is available
				var checks = {
					hasBus: typeof bus === "object" && typeof bus.publish === "function",
					hasKit: typeof kit === "object" && typeof kit.register === "function",
					hasModel: typeof model === "function",
					hasTools: typeof tools === "object" && typeof tools.call === "function",
					hasFs: typeof fs === "object" && typeof fs.read === "function",
					hasMcp: typeof mcp === "object",
					hasOutput: typeof output === "function",
					hasRegistry: typeof registry === "object",
				};
				output(checks);
			`,
		})
		require.NoError(t, err)
		ch := make(chan messages.KitDeployResp, 1)
		unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
		defer unsub()
		select {
		case resp := <-ch:
			assert.True(t, resp.Deployed)
		case <-ctx.Done():
			t.Fatal("timeout")
		}

		// Read the output
		result, err := rt.EvalTS(ctx, "__read_result.ts", `return globalThis.__module_result || "null"`)
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

		sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-kit-imports.ts"})
	})

	t.Run("AgentModule_ExportsMastra", func(t *testing.T) {
		// Deploy .ts that verifies Mastra exports are available as endowments
		pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
			Source: "surface-agent-imports.ts",
			Code: `
				var checks = {
					hasAgent: typeof Agent === "function",
					hasCreateTool: typeof createTool === "function",
					hasCreateWorkflow: typeof createWorkflow === "function",
					hasCreateStep: typeof createStep === "function",
					hasMemory: typeof Memory === "function",
					hasInMemoryStore: typeof InMemoryStore === "function",
					hasLibSQLStore: typeof LibSQLStore === "function",
					hasWorkspace: typeof Workspace === "function",
					hasMDocument: typeof MDocument === "function",
					hasCreateScorer: typeof createScorer === "function",
				};
				output(checks);
			`,
		})
		require.NoError(t, err)
		ch := make(chan messages.KitDeployResp, 1)
		unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
		defer unsub()
		select {
		case resp := <-ch:
			assert.True(t, resp.Deployed)
		case <-ctx.Done():
			t.Fatal("timeout")
		}

		result, err := rt.EvalTS(ctx, "__read_result2.ts", `return globalThis.__module_result || "null"`)
		require.NoError(t, err)

		var checks map[string]bool
		require.NoError(t, json.Unmarshal([]byte(result), &checks))
		assert.True(t, checks["hasAgent"], "Agent class should be available")
		assert.True(t, checks["hasCreateTool"], "createTool should be available")
		assert.True(t, checks["hasCreateWorkflow"], "createWorkflow should be available")
		assert.True(t, checks["hasCreateStep"], "createStep should be available")
		assert.True(t, checks["hasMemory"], "Memory should be available")
		assert.True(t, checks["hasInMemoryStore"], "InMemoryStore should be available")
		assert.True(t, checks["hasWorkspace"], "Workspace should be available")
		assert.True(t, checks["hasMDocument"], "MDocument should be available")

		sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-agent-imports.ts"})
	})

	t.Run("AIModule_ExportsSDK", func(t *testing.T) {
		// Deploy .ts that verifies AI SDK exports are available as endowments
		pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
			Source: "surface-ai-imports.ts",
			Code: `
				var checks = {
					hasGenerateText: typeof generateText === "function",
					hasStreamText: typeof streamText === "function",
					hasGenerateObject: typeof generateObject === "function",
					hasStreamObject: typeof streamObject === "function",
					hasEmbed: typeof embed === "function",
					hasEmbedMany: typeof embedMany === "function",
					hasZ: typeof z === "object" && typeof z.object === "function",
				};
				output(checks);
			`,
		})
		require.NoError(t, err)
		ch := make(chan messages.KitDeployResp, 1)
		unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
		defer unsub()
		select {
		case resp := <-ch:
			assert.True(t, resp.Deployed)
		case <-ctx.Done():
			t.Fatal("timeout")
		}

		result, err := rt.EvalTS(ctx, "__read_result3.ts", `return globalThis.__module_result || "null"`)
		require.NoError(t, err)

		var checks map[string]bool
		require.NoError(t, json.Unmarshal([]byte(result), &checks))
		assert.True(t, checks["hasGenerateText"], "generateText should be available")
		assert.True(t, checks["hasStreamText"], "streamText should be available")
		assert.True(t, checks["hasGenerateObject"], "generateObject should be available")
		assert.True(t, checks["hasStreamObject"], "streamObject should be available")
		assert.True(t, checks["hasEmbed"], "embed should be available")
		assert.True(t, checks["hasZ"], "z should be available")

		sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-ai-imports.ts"})
	})
}

// TestTS_DeployWithTool verifies deploying .ts that creates a tool via createTool + kit.register,
// then calling it from Go.
func TestTS_DeployWithTool(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy .ts that creates a tool using Mastra createTool + registers via kit.register
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "surface-tool.ts",
		Code: `
			const calc = createTool({
				id: "surface-calc",
				description: "adds two numbers (surface test)",
				execute: async ({ context: input }) => {
					var a = (input && input.a) || 0;
					var b = (input && input.b) || 0;
					return { sum: a + b, source: "ts-surface" };
				},
			});
			kit.register("tool", "surface-calc", calc);
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	// Call the tool from Go
	pr2, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{Name: "surface-calc", Input: map[string]any{"a": 10, "b": 32}})
	require.NoError(t, err)
	ch2 := make(chan messages.ToolCallResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch2 <- r })
	defer unsub2()
	var resp messages.ToolCallResp
	select {
	case resp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout calling tool")
	}
	require.Empty(t, resp.Error)

	var result map[string]any
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Equal(t, float64(42), result["sum"])
	assert.Equal(t, "ts-surface", result["source"])

	// Teardown
	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-tool.ts"})
}

// TestTS_DeployWithWorkflow verifies deploying .ts that creates a Mastra workflow,
// runs it, and outputs the result.
func TestTS_DeployWithWorkflow(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy .ts that creates and runs a workflow
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "surface-workflow.ts",
		Code: `
			const step1 = createStep({
				id: "uppercase",
				inputSchema: z.object({ text: z.string() }),
				outputSchema: z.object({ upper: z.string() }),
				execute: async ({ inputData }) => {
					return { upper: inputData.text.toUpperCase() };
				},
			});

			const step2 = createStep({
				id: "exclaim",
				inputSchema: z.object({ upper: z.string() }),
				outputSchema: z.object({ result: z.string() }),
				execute: async ({ inputData }) => {
					return { result: inputData.upper + "!!!" };
				},
			});

			const wf = createWorkflow({
				id: "surface-wf",
				inputSchema: z.object({ text: z.string() }),
				outputSchema: z.object({ result: z.string() }),
			}).then(step1).then(step2).commit();

			const run = await wf.createRun();
			const result = await run.start({ inputData: { text: "surface test" } });
			output({ status: result.status, result: result.result });
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying workflow")
	}

	result, err := rt.EvalTS(ctx, "__read_wf.ts", `return globalThis.__module_result || "null"`)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, "success", parsed["status"])
	if r, ok := parsed["result"].(map[string]any); ok {
		assert.Equal(t, "SURFACE TEST!!!", r["result"])
	}

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-workflow.ts"})
}

// TestTS_DeployWithBusService verifies the full .ts service pattern:
// deploy .ts → bus.on() creates mailbox → Go sends message → .ts replies.
func TestTS_DeployWithBusService(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy .ts as a bus service with bus.on
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "surface-service.ts",
		Code: `
			bus.on("greet", async (msg) => {
				const name = msg.payload && msg.payload.name ? msg.payload.name : "world";
				msg.reply({ greeting: "hello " + name + " from ts service" });
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying service")
	}

	// Wait for subscription to be active
	time.Sleep(100 * time.Millisecond)

	// Send a message to the .ts service mailbox: ts.surface-service.greet
	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.surface-service.greet",
		Payload: json.RawMessage(`{"name":"Go"}`),
	})
	require.NoError(t, err)

	// Subscribe for the reply
	replyCh := make(chan messages.Message, 1)
	unsub2, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, pr2.ReplyTo, func(payload json.RawMessage, msg messages.Message) {
		replyCh <- msg
	})
	require.NoError(t, err)
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]string
		json.Unmarshal(msg.Payload, &result)
		assert.Equal(t, "hello Go from ts service", result["greeting"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for service reply")
	}

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-service.ts"})
}

// TestTS_DeployWithStreaming verifies .ts service streaming pattern:
// deploy .ts → bus.on → msg.send() chunks → msg.reply() final.
func TestTS_DeployWithStreaming(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy .ts that streams chunks then sends final reply
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "surface-streamer.ts",
		Code: `
			bus.on("stream", async (msg) => {
				msg.send({ chunk: "one" });
				msg.send({ chunk: "two" });
				msg.send({ chunk: "three" });
				msg.reply({ done: true, count: 3 });
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout deploying streamer")
	}

	time.Sleep(100 * time.Millisecond)

	// Send message to the streaming service
	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.surface-streamer.stream",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	// Wait for the final message (done=true) on the reply topic
	finalCh := make(chan messages.Message, 1)
	chunkCount := 0
	unsub2, err := rt.SubscribeRaw(ctx, pr2.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			select {
			case finalCh <- msg:
			default:
			}
		} else {
			chunkCount++
		}
	})
	require.NoError(t, err)
	defer unsub2()

	select {
	case final := <-finalCh:
		// The final message should have done=true and count=3
		assert.Equal(t, "true", final.Metadata["done"])
		var finalPayload map[string]any
		json.Unmarshal(final.Payload, &finalPayload)
		assert.Equal(t, true, finalPayload["done"])
		assert.Equal(t, float64(3), finalPayload["count"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for streaming completion")
	}

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-streamer.ts"})
}

// TestTS_GenerateText_Real verifies real AI SDK generateText from deployed .ts.
// Requires OPENAI_API_KEY.
func TestTS_GenerateText_Real(t *testing.T) {
	testutil.LoadEnv(t)
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy .ts that uses generateText from AI SDK directly
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "surface-ai-generate.ts",
		Code: `
			const result = await generateText({
				model: model("openai", "gpt-4o-mini"),
				prompt: "What is 2+2? Reply with just the number.",
				maxTokens: 10,
			});
			output({
				text: result.text,
				hasUsage: !!result.usage,
				finishReason: result.finishReason,
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy should succeed: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying AI generate")
	}

	result, err := rt.EvalTS(ctx, "__read_ai.ts", `return globalThis.__module_result || "null"`)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "generateText should return non-empty text")
	assert.Contains(t, parsed["text"], "4", "should contain the answer 4")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-ai-generate.ts"})
}

// TestTS_Agent_Real verifies real Mastra Agent from deployed .ts.
// Requires OPENAI_API_KEY.
func TestTS_Agent_Real(t *testing.T) {
	testutil.LoadEnv(t)
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy .ts that creates a Mastra Agent and calls generate
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "surface-agent.ts",
		Code: `
			const myAgent = new Agent({
				name: "surface-agent",
				model: model("openai", "gpt-4o-mini"),
				instructions: "Reply with exactly: AGENT_WORKS",
			});
			kit.register("agent", "surface-agent", myAgent);

			const result = await myAgent.generate("Say the magic word");
			output({
				text: result.text,
				hasUsage: !!result.usage,
				finishReason: result.finishReason,
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy should succeed: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying agent")
	}

	result, err := rt.EvalTS(ctx, "__read_agent.ts", `return globalThis.__module_result || "null"`)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Contains(t, parsed["text"], "AGENT_WORKS")
	assert.True(t, parsed["hasUsage"].(bool))

	// Verify agent was registered
	pr2, err := sdk.Publish(rt, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	ch2 := make(chan messages.AgentListResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.AgentListResp](rt, ctx, pr2.ReplyTo, func(r messages.AgentListResp, m messages.Message) { ch2 <- r })
	defer unsub2()
	select {
	case listResp := <-ch2:
		found := false
		for _, a := range listResp.Agents {
			if a.Name == "surface-agent" {
				found = true
			}
		}
		assert.True(t, found, "surface-agent should be in agents list")
	case <-ctx.Done():
		t.Fatal("timeout listing agents")
	}

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-agent.ts"})
}

// TestTS_AgentWithTool_Real verifies Agent with a createTool from deployed .ts.
// Requires OPENAI_API_KEY.
func TestTS_AgentWithTool_Real(t *testing.T) {
	testutil.LoadEnv(t)
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy .ts: Agent with a local tool
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "surface-agent-tool.ts",
		Code: `
			const addTool = createTool({
				id: "add",
				description: "adds two numbers",
				execute: async ({ context: input }) => {
					var a = (input && input.a) || 0;
					var b = (input && input.b) || 0;
					return { sum: a + b };
				},
			});

			const myAgent = new Agent({
				name: "math-agent",
				model: model("openai", "gpt-4o-mini"),
				instructions: "You are a math assistant. Reply with just the number.",
				tools: { add: addTool },
			});

			const result = await myAgent.generate("What is 17 + 25?");
			output({
				text: result.text,
				hasSteps: result.steps && result.steps.length > 0,
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy should succeed: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying agent with tool")
	}

	result, err := rt.EvalTS(ctx, "__read_agent_tool.ts", `return globalThis.__module_result || "null"`)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "agent should return non-empty text")
	// The agent may or may not use the tool — just verify it responded
	assert.True(t, parsed["hasSteps"].(bool), "should have at least one step")

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "surface-agent-tool.ts"})
}

// TestTS_BusServiceAsAIProxy verifies the canonical .ts service pattern:
// deploy .ts as an AI service → Go sends message → .ts calls generateText → replies.
// This is the primary use case for the .ts service architecture.
// Requires OPENAI_API_KEY.
//
// KNOWN ISSUE: generateText (fetch → HTTP) called inside a bus.on handler
// processed by the job pump's ProcessScheduledJobs does not complete.
// The fetch goroutine's ctx.Schedule resolve callback is not picked up
// by the Await loop nested inside ProcessJobs. This needs investigation
// in the jsbridge/quickjs-go interaction layer.
// Direct EvalTS/Deploy with generateText works fine (tested above).
func TestTS_BusServiceAsAIProxy(t *testing.T) {
	t.Skip("KNOWN ISSUE: generateText inside bus.on handler — fetch resolve not picked up by nested Await")
	testutil.LoadEnv(t)
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy .ts AI service: receives prompt via bus, calls generateText, replies with result
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "ai-service.ts",
		Code: `
			bus.on("generate", async (msg) => {
				console.log("ai-service: received request");
				try {
					var prompt = (msg.payload && msg.payload.prompt) || "say hello";
					console.log("ai-service: calling generateText with prompt: " + prompt);
					const result = await generateText({
						model: model("openai", "gpt-4o-mini"),
						prompt: prompt,
						maxTokens: 20,
					});
					console.log("ai-service: got result: " + result.text);
					msg.reply({ text: result.text, usage: result.usage });
				} catch (e) {
					console.error("ai-service error: " + (e.message || String(e)));
					msg.reply({ error: e.message || String(e) });
				}
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy AI service: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying AI service")
	}

	time.Sleep(100 * time.Millisecond)

	// Go sends a message to the .ts AI service
	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.ai-service.generate",
		Payload: json.RawMessage(`{"prompt":"Reply with exactly: BUS_AI_WORKS"}`),
	})
	require.NoError(t, err)

	replyCh := make(chan messages.Message, 1)
	unsub2, err := rt.SubscribeRaw(ctx, pr2.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			replyCh <- msg
		}
	})
	require.NoError(t, err)
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]any
		json.Unmarshal(msg.Payload, &result)
		if errMsg, hasErr := result["error"]; hasErr {
			t.Fatalf("AI service returned error: %v", errMsg)
		}
		assert.NotEmpty(t, result["text"], "AI service should return text")
		assert.NotNil(t, result["usage"], "AI service should return usage")
	case <-ctx.Done():
		t.Fatal("timeout waiting for AI service reply — generateText inside bus.on handler may not complete")
	}

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "ai-service.ts"})
}
