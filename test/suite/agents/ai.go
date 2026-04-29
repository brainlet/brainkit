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

// testDeployAgentThenList — deploy an agent via .ts, then list agents and verify it appears.
// Requires OPENAI_API_KEY.
func testDeployAgentThenList(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy .ts that creates a Mastra Agent and registers it
	testutil.Deploy(t, env.Kit, "ai-agent-agent-adv.ts", `
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
		`)

	// Verify output from generate
	result := testutil.EvalTS(t, env.Kit, "__read_ai_agent_adv.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "generateText should return non-empty text")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	// Verify agent was registered via AgentList
	listResp, err := sdk.Call[agentmsg.AgentListMsg, agentmsg.AgentListResp](env.Kit, ctx, agentmsg.AgentListMsg{})
	require.NoError(t, err)
	found := false
	for _, a := range listResp.Agents {
		if a.Name == "ai-list-agent-adv" {
			found = true
		}
	}
	assert.True(t, found, "ai-list-agent-adv should be in agents list")

	// Get status — should be "idle" by default
	statusResp, err := sdk.Call[agentmsg.AgentGetStatusMsg, agentmsg.AgentGetStatusResp](env.Kit, ctx, agentmsg.AgentGetStatusMsg{Name: "ai-list-agent-adv"})
	require.NoError(t, err)
	assert.Equal(t, "idle", statusResp.Status)

	// Set status to "busy"
	_, err = sdk.Call[agentmsg.AgentSetStatusMsg, agentmsg.AgentSetStatusResp](env.Kit, ctx, agentmsg.AgentSetStatusMsg{
		Name: "ai-list-agent-adv", Status: "busy",
	})
	require.NoError(t, err)

	// Re-get status — should be "busy"
	statusResp, err = sdk.Call[agentmsg.AgentGetStatusMsg, agentmsg.AgentGetStatusResp](env.Kit, ctx, agentmsg.AgentGetStatusMsg{Name: "ai-list-agent-adv"})
	require.NoError(t, err)
	assert.Equal(t, "busy", statusResp.Status)

	testutil.Teardown(t, env.Kit, "ai-agent-agent-adv.ts")
}
