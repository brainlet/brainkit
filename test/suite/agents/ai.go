package agents

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

// testDeployAgentThenList — deploy an agent via .ts, then list agents and verify it appears.
// Requires OPENAI_API_KEY.
func testDeployAgentThenList(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy .ts that creates a Mastra Agent and registers it
	pr, err := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "ai-agent-agent-adv.ts",
		Code: `
			const myAgent = new Agent({
				name: "ai-list-agent-adv",
				model: model("openai", "gpt-4o-mini"),
				instructions: "Reply with exactly: AGENT_LISTED",
			});
			kit.register("agent", "ai-list-agent-adv", myAgent);

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
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy should succeed: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying AI agent")
	}

	// Verify output from generate
	result, err := env.Kernel.EvalTS(ctx, "__read_ai_agent_adv.ts", `return globalThis.__module_result || "null"`)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "generateText should return non-empty text")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	// Verify agent was registered via AgentList
	pr2, err := sdk.Publish(env.Kernel, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	ch2 := make(chan messages.AgentListResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.AgentListResp](env.Kernel, ctx, pr2.ReplyTo, func(r messages.AgentListResp, m messages.Message) { ch2 <- r })
	defer unsub2()
	select {
	case listResp := <-ch2:
		found := false
		for _, a := range listResp.Agents {
			if a.Name == "ai-list-agent-adv" {
				found = true
			}
		}
		assert.True(t, found, "ai-list-agent-adv should be in agents list")
	case <-ctx.Done():
		t.Fatal("timeout listing agents")
	}

	sdk.Publish(env.Kernel, ctx, messages.KitTeardownMsg{Source: "ai-agent-agent-adv.ts"})
}
