package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testToolPipeline — full pipeline: deploy .ts tool → list → call → teardown → verify gone.
func testToolPipeline(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(env.T.Context(), 30*time.Second)
	defer cancel()
	rt := env.Kit

	// 1. Deploy .ts code that creates a new tool
	mp1, _ := json.Marshal(map[string]string{"name": "pipeline-tool-adv", "entry": "pipeline-tool-adv.ts"})
	deployResp, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](rt, ctx, packagemsg.PackageDeployMsg{
		Manifest: mp1,
		Files: map[string]string{"pipeline-tool-adv.ts": `
			const greeter = createTool({
				id: "greeter-tool-adv",
				description: "greets a person by name",
				execute: async ({ context: input }) => {
					return { greeting: "Hello, " + (input.name || "world") + "!" };
				}
			});
			kit.register("tool", "greeter-tool-adv", greeter);
		`},
	})
	require.NoError(t, err)
	assert.True(t, deployResp.Deployed)

	// 2. Verify "greeter-tool-adv" appears in tools.list
	listResp, err := sdk.Call[toolmsg.ToolListMsg, toolmsg.ToolListResp](rt, ctx, toolmsg.ToolListMsg{})
	require.NoError(t, err)
	found := false
	for _, tool := range listResp.Tools {
		if tool.ShortName == "greeter-tool-adv" {
			found = true
		}
	}
	assert.True(t, found, "deployed 'greeter-tool-adv' tool should appear")

	// 3. Call the deployed tool
	callResp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](rt, ctx, toolmsg.ToolCallMsg{
		Name:  "greeter-tool-adv",
		Input: map[string]any{"name": "Brainkit"},
	})
	require.NoError(t, err)
	var result map[string]string
	json.Unmarshal(callResp.Result, &result)
	assert.Equal(t, "Hello, Brainkit!", result["greeting"])

	// 4. Teardown
	_, err = sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](rt, ctx, packagemsg.PackageTeardownMsg{Name: "pipeline-tool-adv"})
	require.NoError(t, err)
}
