package tools

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testToolPipeline — full pipeline: deploy .ts tool → list → call → teardown → verify gone.
func testToolPipeline(t *testing.T, env *suite.TestEnv) {
	ctx := env.T.Context()
	rt := env.Kit

	// 1. Deploy .ts code that creates a new tool
	pr1, err := sdk.Publish(rt, ctx, sdk.KitDeployMsg{
		Source: "pipeline-tool-adv.ts",
		Code: `
			const greeter = createTool({
				id: "greeter-tool-adv",
				description: "greets a person by name",
				execute: async ({ context: input }) => {
					return { greeting: "Hello, " + (input.name || "world") + "!" };
				}
			});
			kit.register("tool", "greeter-tool-adv", greeter);
		`,
	})
	require.NoError(t, err)
	deployCh := make(chan sdk.KitDeployResp, 1)
	cancelDeploy, err := sdk.SubscribeTo[sdk.KitDeployResp](rt, ctx, pr1.ReplyTo, func(r sdk.KitDeployResp, _ sdk.Message) { deployCh <- r })
	require.NoError(t, err)
	defer cancelDeploy()
	select {
	case resp := <-deployCh:
		assert.True(t, resp.Deployed)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout deploying pipeline tool")
	}

	// 2. Verify "greeter-tool-adv" appears in tools.list
	pr2, err := sdk.Publish(rt, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)
	listCh := make(chan sdk.ToolListResp, 1)
	cancelList, err := sdk.SubscribeTo[sdk.ToolListResp](rt, ctx, pr2.ReplyTo, func(r sdk.ToolListResp, _ sdk.Message) { listCh <- r })
	require.NoError(t, err)
	defer cancelList()
	select {
	case listResp := <-listCh:
		found := false
		for _, tool := range listResp.Tools {
			if tool.ShortName == "greeter-tool-adv" {
				found = true
			}
		}
		assert.True(t, found, "deployed 'greeter-tool-adv' tool should appear")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout listing tools")
	}

	// 3. Call the deployed tool
	pr3, err := sdk.Publish(rt, ctx, sdk.ToolCallMsg{
		Name:  "greeter-tool-adv",
		Input: map[string]any{"name": "Brainkit"},
	})
	require.NoError(t, err)
	callCh := make(chan sdk.ToolCallResp, 1)
	cancelCall, err := sdk.SubscribeTo[sdk.ToolCallResp](rt, ctx, pr3.ReplyTo, func(r sdk.ToolCallResp, _ sdk.Message) { callCh <- r })
	require.NoError(t, err)
	defer cancelCall()
	select {
	case callResp := <-callCh:
		var result map[string]string
		json.Unmarshal(callResp.Result, &result)
		assert.Equal(t, "Hello, Brainkit!", result["greeting"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout calling tool")
	}

	// 4. Teardown
	pr4, err := sdk.Publish(rt, ctx, sdk.KitTeardownMsg{Source: "pipeline-tool-adv.ts"})
	require.NoError(t, err)
	tearCh := make(chan sdk.KitTeardownResp, 1)
	cancelTear, _ := sdk.SubscribeTo[sdk.KitTeardownResp](rt, ctx, pr4.ReplyTo, func(r sdk.KitTeardownResp, _ sdk.Message) { tearCh <- r })
	defer cancelTear()
	select {
	case <-tearCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on teardown")
	}
}
