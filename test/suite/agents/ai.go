package agents

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
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
	pr, err := sdk.Publish(env.Kit, ctx, messages.KitDeployMsg{
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
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kit, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		require.True(t, resp.Deployed, "deploy should succeed: %s", resp.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying AI agent")
	}

	// Verify output from generate
	result := testutil.EvalTS(t, env.Kit, "__read_ai_agent_adv.ts", `return globalThis.__module_result || "null"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.NotEmpty(t, parsed["text"], "generateText should return non-empty text")
	assert.True(t, parsed["hasUsage"].(bool), "should have token usage")

	// Verify agent was registered via AgentList
	pr2, err := sdk.Publish(env.Kit, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	ch2 := make(chan messages.AgentListResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.AgentListResp](env.Kit, ctx, pr2.ReplyTo, func(r messages.AgentListResp, m messages.Message) { ch2 <- r })
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

	// Get status — should be "idle" by default
	pr3, err := sdk.Publish(env.Kit, ctx, messages.AgentGetStatusMsg{Name: "ai-list-agent-adv"})
	require.NoError(t, err)
	ch3 := make(chan messages.AgentGetStatusResp, 1)
	unsub3, err := sdk.SubscribeTo[messages.AgentGetStatusResp](env.Kit, ctx, pr3.ReplyTo, func(r messages.AgentGetStatusResp, m messages.Message) { ch3 <- r })
	require.NoError(t, err)
	defer unsub3()
	var statusResp messages.AgentGetStatusResp
	select {
	case statusResp = <-ch3:
	case <-ctx.Done():
		t.Fatal("timeout getting initial status")
	}
	assert.Equal(t, "idle", statusResp.Status)

	// Set status to "busy"
	pr4, err := sdk.Publish(env.Kit, ctx, messages.AgentSetStatusMsg{
		Name: "ai-list-agent-adv", Status: "busy",
	})
	require.NoError(t, err)
	ch4 := make(chan messages.AgentSetStatusResp, 1)
	unsub4, _ := sdk.SubscribeTo[messages.AgentSetStatusResp](env.Kit, ctx, pr4.ReplyTo, func(r messages.AgentSetStatusResp, m messages.Message) { ch4 <- r })
	defer unsub4()
	select {
	case <-ch4:
	case <-ctx.Done():
		t.Fatal("timeout setting status")
	}

	// Re-get status — should be "busy"
	pr5, err := sdk.Publish(env.Kit, ctx, messages.AgentGetStatusMsg{Name: "ai-list-agent-adv"})
	require.NoError(t, err)
	ch5 := make(chan messages.AgentGetStatusResp, 1)
	unsub5, err := sdk.SubscribeTo[messages.AgentGetStatusResp](env.Kit, ctx, pr5.ReplyTo, func(r messages.AgentGetStatusResp, m messages.Message) { ch5 <- r })
	require.NoError(t, err)
	defer unsub5()
	select {
	case statusResp = <-ch5:
	case <-ctx.Done():
		t.Fatal("timeout getting updated status")
	}
	assert.Equal(t, "busy", statusResp.Status)

	sdk.Publish(env.Kit, ctx, messages.KitTeardownMsg{Source: "ai-agent-agent-adv.ts"})
}
