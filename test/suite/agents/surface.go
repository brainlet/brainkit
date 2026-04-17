package agents

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSurfaceGenerateTextReal — verify real AI SDK generateText from deployed .ts.
// Requires OPENAI_API_KEY.
func testSurfaceGenerateTextReal(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.KitDeployMsg{
		Source: "surface-ai-gen-adv.ts",
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
	ch := make(chan sdk.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.KitDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.KitDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy should succeed: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying AI generate")
	}

	result := testutil.EvalTS(t, env.Kit, "__read_ai_gen_adv.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "generateText should return non-empty text")
	assert.Contains(t, parsed["text"], "4", "should contain the answer 4")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	sdk.Publish(env.Kit, ctx, sdk.KitTeardownMsg{Source: "surface-ai-gen-adv.ts"})
}

// testSurfaceAgentGenerate — deploy agent, call generate, verify response.
// Requires OPENAI_API_KEY.
func testSurfaceAgentGenerate(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.KitDeployMsg{
		Source: "surface-gen-agent-adv.ts",
		Code: `
			const myAgent = new Agent({
				name: "surface-gen-agent-adv",
				model: model("openai", "gpt-4o-mini"),
				instructions: "Reply with exactly: SURFACE_AGENT_OK",
			});
			kit.register("agent", "surface-gen-agent-adv", myAgent);

			const result = await myAgent.generate("Say the magic word");
			output({
				text: result.text,
				hasUsage: !!result.usage,
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan sdk.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.KitDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.KitDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy should succeed: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying surface agent")
	}

	result := testutil.EvalTS(t, env.Kit, "__read_surface_gen_adv.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Contains(t, parsed["text"], "SURFACE_AGENT_OK")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	// Verify agent was registered
	pr2, err := sdk.Publish(env.Kit, ctx, sdk.AgentListMsg{})
	require.NoError(t, err)
	ch2 := make(chan sdk.AgentListResp, 1)
	unsub2, _ := sdk.SubscribeTo[sdk.AgentListResp](env.Kit, ctx, pr2.ReplyTo, func(r sdk.AgentListResp, m sdk.Message) { ch2 <- r })
	defer unsub2()
	select {
	case listResp := <-ch2:
		found := false
		for _, a := range listResp.Agents {
			if a.Name == "surface-gen-agent-adv" {
				found = true
			}
		}
		assert.True(t, found, "surface-gen-agent-adv should be in agents list")
	case <-ctx.Done():
		t.Fatal("timeout listing agents")
	}

	sdk.Publish(env.Kit, ctx, sdk.KitTeardownMsg{Source: "surface-gen-agent-adv.ts"})
}

// testSurfaceAgentWithTool — deploy agent with a tool, call generate, verify steps.
// Requires OPENAI_API_KEY. Retries up to 3 times for transient LLM API errors.
func testSurfaceAgentWithTool(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	const source = "surface-tool-agent-adv.ts"
	code := `
		const addTool = createTool({
			id: "add-agent-adv",
			description: "adds two numbers",
			inputSchema: z.object({ a: z.number(), b: z.number() }),
			execute: async ({ context: input }) => {
				var a = (input && input.a) || 0;
				var b = (input && input.b) || 0;
				return { sum: a + b };
			},
		});

		const myAgent = new Agent({
			name: "math-agent-adv",
			model: model("openai", "gpt-4o-mini"),
			instructions: "You are a math assistant. Reply with just the number.",
			tools: { add: addTool },
		});

		const result = await myAgent.generate("What is 17 + 25?");
		output({
			text: result.text,
			hasSteps: result.steps && result.steps.length > 0,
		});
	`

	var deployErr error
	for attempt := 0; attempt < 3; attempt++ {
		deployErr = testutil.DeployErr(env.Kit, source, code)
		if deployErr == nil {
			break
		}
		// Teardown the failed attempt before retrying
		testutil.Teardown(t, env.Kit, source)
		if attempt < 2 {
			t.Logf("deploy attempt %d failed (transient LLM error), retrying: %v", attempt+1, deployErr)
		}
	}
	require.NoError(t, deployErr, "deploy should succeed after retries")

	result := testutil.EvalTS(t, env.Kit, "__read_surface_tool_adv.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "agent should return non-empty text")
	assert.True(t, parsed["hasSteps"].(bool), "should have at least one step")

	testutil.Teardown(t, env.Kit, source)
}

// testSurfaceBusServiceAIProxy — deploy .ts as AI service via bus, Go sends message, .ts calls generateText, replies.
// Requires OPENAI_API_KEY.
func testSurfaceBusServiceAIProxy(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.KitDeployMsg{
		Source: "ai-svc-agent-adv.ts",
		Code: `
			bus.on("generate", async (msg) => {
				try {
					var prompt = (msg.payload && msg.payload.prompt) || "say hello";
					const result = await generateText({
						model: model("openai", "gpt-4o-mini"),
						prompt: prompt,
						maxTokens: 20,
					});
					msg.reply({ text: result.text, usage: result.usage });
				} catch (e) {
					msg.reply({ error: e.message || String(e) });
				}
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan sdk.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.KitDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.KitDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy AI service: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying AI service")
	}

	time.Sleep(100 * time.Millisecond)

	// Go sends to the .ts AI service
	pr2, err := sdk.SendToService(env.Kit, ctx, "ai-svc-agent-adv.ts", "generate", json.RawMessage(`{"prompt":"Reply with exactly: BUS_AI_WORKS"}`))
	require.NoError(t, err)

	replyCh := make(chan sdk.Message, 1)
	unsub2, err := env.Kit.SubscribeRaw(ctx, pr2.ReplyTo, func(msg sdk.Message) {
		if msg.Metadata["done"] == "true" {
			select {
			case replyCh <- msg:
			default:
			}
		}
	})
	require.NoError(t, err)
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]any
		json.Unmarshal(suite.ResponseDataFromMsg(msg), &result)
		if errMsg, hasErr := result["error"]; hasErr {
			t.Fatalf("AI service returned error: %v", errMsg)
		}
		assert.NotEmpty(t, result["text"], "AI service should return text")
		assert.NotNil(t, result["usage"], "AI service should return usage")
	case <-ctx.Done():
		t.Fatal("timeout waiting for AI service reply")
	}

	sdk.Publish(env.Kit, ctx, sdk.KitTeardownMsg{Source: "ai-svc-agent-adv.ts"})
}
