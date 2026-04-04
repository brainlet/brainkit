package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
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

// testE2EDeployWithErrorRecovery — deploy bad code, recover, deploy good code.
func testE2EDeployWithErrorRecovery(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	// Deploy bad code — should fail
	_, err := freshEnv.Kernel.Deploy(ctx, "recovery-deploy.ts", `throw new Error("intentional failure");`)
	assert.Error(t, err)

	// Deploy good code to same source — should succeed
	_, err = freshEnv.Kernel.Deploy(ctx, "recovery-deploy.ts", `
		const t = createTool({id: "recovered-deploy", description: "test", execute: async () => ({ok: true})});
		kit.register("tool", "recovered-deploy", t);
	`)
	require.NoError(t, err)

	// Verify tool works
	payload, ok := freshEnv.SendAndReceive(t, messages.ToolCallMsg{Name: "recovered-deploy", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "ok")
}

// testE2EDeployListRedeployTeardown — deploy → list → redeploy → teardown → list cycle.
func testE2EDeployListRedeployTeardown(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy v1
	pr1, err := sdk.Publish(freshEnv.Kernel, ctx, messages.KitDeployMsg{
		Source: "lifecycle-e2e-deploy.ts",
		Code:   `const v1 = createTool({ id: "version-check-e2e", description: "v1", execute: async () => ({ version: 1 }) }); kit.register("tool", "version-check-e2e", v1);`,
	})
	require.NoError(t, err)
	ch1 := make(chan messages.KitDeployResp, 1)
	us1, _ := sdk.SubscribeTo[messages.KitDeployResp](freshEnv.Kernel, ctx, pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch1 <- r })
	defer us1()
	select {
	case <-ch1:
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	// List — should show lifecycle-e2e-deploy.ts
	pr2, err := sdk.Publish(freshEnv.Kernel, ctx, messages.KitListMsg{})
	require.NoError(t, err)
	ch2 := make(chan messages.KitListResp, 1)
	us2, err := sdk.SubscribeTo[messages.KitListResp](freshEnv.Kernel, ctx, pr2.ReplyTo, func(r messages.KitListResp, m messages.Message) { ch2 <- r })
	require.NoError(t, err)
	defer us2()
	var listResp messages.KitListResp
	select {
	case listResp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout listing")
	}
	sources := make(map[string]bool)
	for _, d := range listResp.Deployments {
		sources[d.Source] = true
	}
	assert.True(t, sources["lifecycle-e2e-deploy.ts"])

	// Redeploy with v2
	pr3, err := sdk.Publish(freshEnv.Kernel, ctx, messages.KitRedeployMsg{
		Source: "lifecycle-e2e-deploy.ts",
		Code:   `const v2 = createTool({ id: "version-check-e2e-v2", description: "v2", execute: async () => ({ version: 2 }) }); kit.register("tool", "version-check-e2e-v2", v2);`,
	})
	require.NoError(t, err)
	ch3 := make(chan messages.KitRedeployResp, 1)
	us3, _ := sdk.SubscribeTo[messages.KitRedeployResp](freshEnv.Kernel, ctx, pr3.ReplyTo, func(r messages.KitRedeployResp, m messages.Message) { ch3 <- r })
	defer us3()
	select {
	case <-ch3:
	case <-ctx.Done():
		t.Fatal("timeout redeploying")
	}

	// Teardown
	pr4, err := sdk.Publish(freshEnv.Kernel, ctx, messages.KitTeardownMsg{Source: "lifecycle-e2e-deploy.ts"})
	require.NoError(t, err)
	ch4 := make(chan messages.KitTeardownResp, 1)
	us4, err := sdk.SubscribeTo[messages.KitTeardownResp](freshEnv.Kernel, ctx, pr4.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { ch4 <- r })
	require.NoError(t, err)
	defer us4()
	select {
	case <-ch4:
	case <-ctx.Done():
		t.Fatal("timeout tearing down")
	}

	// List — should not contain lifecycle-e2e-deploy.ts
	pr5, err := sdk.Publish(freshEnv.Kernel, ctx, messages.KitListMsg{})
	require.NoError(t, err)
	ch5 := make(chan messages.KitListResp, 1)
	us5, err := sdk.SubscribeTo[messages.KitListResp](freshEnv.Kernel, ctx, pr5.ReplyTo, func(r messages.KitListResp, m messages.Message) { ch5 <- r })
	require.NoError(t, err)
	defer us5()
	select {
	case listResp = <-ch5:
	case <-ctx.Done():
		t.Fatal("timeout listing after teardown")
	}
	for _, d := range listResp.Deployments {
		assert.NotEqual(t, "lifecycle-e2e-deploy.ts", d.Source, "should be torn down")
	}
}
