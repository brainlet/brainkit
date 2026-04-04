package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDeployLifecycle — full lifecycle: deploy->call->teardown->redeploy->call->verify.
func testDeployLifecycle(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// Deploy v1
	_, err := env.Kernel.Deploy(ctx, "lifecycle-deploy-adv.ts", `
		const t = createTool({id: "lc-adv-tool", description: "v1", execute: async () => ({version: 1})});
		kit.register("tool", "lc-adv-tool", t);
	`)
	require.NoError(t, err)

	// Call v1
	payload, ok := env.SendAndReceive(t, messages.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "1")

	// Teardown
	_, err = env.Kernel.Teardown(ctx, "lifecycle-deploy-adv.ts")
	require.NoError(t, err)

	// Tool should be gone
	payload2, ok2 := env.SendAndReceive(t, messages.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok2)
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload2))

	// Redeploy v2
	_, err = env.Kernel.Deploy(ctx, "lifecycle-deploy-adv.ts", `
		const t = createTool({id: "lc-adv-tool", description: "v2", execute: async () => ({version: 2})});
		kit.register("tool", "lc-adv-tool", t);
	`)
	require.NoError(t, err)

	// Call v2
	payload3, ok3 := env.SendAndReceive(t, messages.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok3)
	assert.Contains(t, string(payload3), "2")

	env.Kernel.Teardown(ctx, "lifecycle-deploy-adv.ts")
}
