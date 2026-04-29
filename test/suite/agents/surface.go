package agents

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/agents/agentmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSurfaceGenerateTextReal — verify real AI SDK generateText from deployed .ts.
// Requires OPENAI_API_KEY.
func testSurfaceGenerateTextReal(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	testutil.Deploy(t, env.Kit, "surface-ai-gen-adv.ts", `
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
		`)

	result := testutil.EvalTS(t, env.Kit, "__read_ai_gen_adv.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "generateText should return non-empty text")
	assert.Contains(t, parsed["text"], "4", "should contain the answer 4")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	testutil.Teardown(t, env.Kit, "surface-ai-gen-adv.ts")
}

// testSurfaceAgentGenerate — deploy agent, call generate, verify response.
// Requires OPENAI_API_KEY.
func testSurfaceAgentGenerate(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "surface-gen-agent-adv.ts", `
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
		`)

	result := testutil.EvalTS(t, env.Kit, "__read_surface_gen_adv.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Contains(t, parsed["text"], "SURFACE_AGENT_OK")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	// Verify agent was registered
	listResp, err := sdk.Call[agentmsg.AgentListMsg, agentmsg.AgentListResp](env.Kit, ctx, agentmsg.AgentListMsg{})
	require.NoError(t, err)
	found := false
	for _, a := range listResp.Agents {
		if a.Name == "surface-gen-agent-adv" {
			found = true
		}
	}
	assert.True(t, found, "surface-gen-agent-adv should be in agents list")

	testutil.Teardown(t, env.Kit, "surface-gen-agent-adv.ts")
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

	testutil.Deploy(t, env.Kit, "ai-svc-agent-adv.ts", `
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
		`)

	time.Sleep(100 * time.Millisecond)

	// Go sends to the .ts AI service
	reply, err := sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   sdk.ResolveServiceTopic("ai-svc-agent-adv.ts", "generate"),
		Payload: json.RawMessage(`{"prompt":"Reply with exactly: BUS_AI_WORKS"}`),
	})
	require.NoError(t, err)

	var result map[string]any
	json.Unmarshal(reply, &result)
	if errMsg, hasErr := result["error"]; hasErr {
		t.Fatalf("AI service returned error: %v", errMsg)
	}
	assert.NotEmpty(t, result["text"], "AI service should return text")
	assert.NotNil(t, result["usage"], "AI service should return usage")

	testutil.Teardown(t, env.Kit, "ai-svc-agent-adv.ts")
}
