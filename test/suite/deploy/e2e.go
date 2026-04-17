package deploy

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

// testDeployLifecycle — full lifecycle: deploy->call->teardown->redeploy->call->verify.
func testDeployLifecycle(t *testing.T, env *suite.TestEnv) {
	// Deploy v1
	testutil.Deploy(t, env.Kit, "lifecycle-deploy-adv.ts", `
		const t = createTool({id: "lc-adv-tool", description: "v1", execute: async () => ({version: 1})});
		kit.register("tool", "lc-adv-tool", t);
	`)

	// Call v1
	payload, ok := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "1")

	// Teardown
	testutil.Teardown(t, env.Kit, "lifecycle-deploy-adv.ts")

	// Tool should be gone
	payload2, ok2 := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok2)
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload2))

	// Redeploy v2
	testutil.Deploy(t, env.Kit, "lifecycle-deploy-adv.ts", `
		const t = createTool({id: "lc-adv-tool", description: "v2", execute: async () => ({version: 2})});
		kit.register("tool", "lc-adv-tool", t);
	`)

	// Call v2
	payload3, ok3 := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok3)
	assert.Contains(t, string(payload3), "2")

	testutil.Teardown(t, env.Kit, "lifecycle-deploy-adv.ts")
}

// testE2EDeployWithErrorRecovery — deploy bad code, recover, deploy good code.
func testE2EDeployWithErrorRecovery(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)

	// Deploy bad code — should fail
	err := testutil.DeployErr(freshEnv.Kit, "recovery-deploy.ts", `throw new Error("intentional failure");`)
	assert.Error(t, err)

	// Deploy good code to same source — should succeed
	testutil.Deploy(t, freshEnv.Kit, "recovery-deploy.ts", `
		const t = createTool({id: "recovered-deploy", description: "test", execute: async () => ({ok: true})});
		kit.register("tool", "recovered-deploy", t);
	`)

	// Verify tool works
	payload, ok := freshEnv.SendAndReceive(t, sdk.ToolCallMsg{Name: "recovered-deploy", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "ok")
}

// testE2EDeployListRedeployTeardown — deploy -> list -> redeploy -> teardown -> list cycle.
func testE2EDeployListRedeployTeardown(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy v1
	mp1, _ := json.Marshal(map[string]string{"name": "lifecycle-e2e-deploy", "entry": "lifecycle-e2e-deploy.ts"})
	pr1, err := sdk.Publish(freshEnv.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: mp1,
		Files:    map[string]string{"lifecycle-e2e-deploy.ts": `const v1 = createTool({ id: "version-check-e2e", description: "v1", execute: async () => ({ version: 1 }) }); kit.register("tool", "version-check-e2e", v1);`},
	})
	require.NoError(t, err)
	ch1 := make(chan sdk.PackageDeployResp, 1)
	us1, _ := sdk.SubscribeTo[sdk.PackageDeployResp](freshEnv.Kit, ctx, pr1.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch1 <- r })
	defer us1()
	select {
	case <-ch1:
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	// List — should show lifecycle-e2e-deploy
	pr2, err := sdk.Publish(freshEnv.Kit, ctx, sdk.PackageListDeployedMsg{})
	require.NoError(t, err)
	ch2 := make(chan sdk.PackageListDeployedResp, 1)
	us2, err := sdk.SubscribeTo[sdk.PackageListDeployedResp](freshEnv.Kit, ctx, pr2.ReplyTo, func(r sdk.PackageListDeployedResp, m sdk.Message) { ch2 <- r })
	require.NoError(t, err)
	defer us2()
	var listResp sdk.PackageListDeployedResp
	select {
	case listResp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout listing")
	}
	sources := make(map[string]bool)
	for _, d := range listResp.Packages {
		sources[d.Source] = true
	}
	assert.True(t, sources["lifecycle-e2e-deploy.ts"])

	// Redeploy with v2 (hot-replace via same deploy message)
	mp3, _ := json.Marshal(map[string]string{"name": "lifecycle-e2e-deploy", "entry": "lifecycle-e2e-deploy.ts"})
	pr3, err := sdk.Publish(freshEnv.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: mp3,
		Files:    map[string]string{"lifecycle-e2e-deploy.ts": `const v2 = createTool({ id: "version-check-e2e-v2", description: "v2", execute: async () => ({ version: 2 }) }); kit.register("tool", "version-check-e2e-v2", v2);`},
	})
	require.NoError(t, err)
	ch3 := make(chan sdk.PackageDeployResp, 1)
	us3, _ := sdk.SubscribeTo[sdk.PackageDeployResp](freshEnv.Kit, ctx, pr3.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch3 <- r })
	defer us3()
	select {
	case <-ch3:
	case <-ctx.Done():
		t.Fatal("timeout redeploying")
	}

	// Teardown
	pr4, err := sdk.Publish(freshEnv.Kit, ctx, sdk.PackageTeardownMsg{Name: "lifecycle-e2e-deploy"})
	require.NoError(t, err)
	ch4 := make(chan sdk.PackageTeardownResp, 1)
	us4, err := sdk.SubscribeTo[sdk.PackageTeardownResp](freshEnv.Kit, ctx, pr4.ReplyTo, func(r sdk.PackageTeardownResp, m sdk.Message) { ch4 <- r })
	require.NoError(t, err)
	defer us4()
	select {
	case <-ch4:
	case <-ctx.Done():
		t.Fatal("timeout tearing down")
	}

	// List — should not contain lifecycle-e2e-deploy.ts
	pr5, err := sdk.Publish(freshEnv.Kit, ctx, sdk.PackageListDeployedMsg{})
	require.NoError(t, err)
	ch5 := make(chan sdk.PackageListDeployedResp, 1)
	us5, err := sdk.SubscribeTo[sdk.PackageListDeployedResp](freshEnv.Kit, ctx, pr5.ReplyTo, func(r sdk.PackageListDeployedResp, m sdk.Message) { ch5 <- r })
	require.NoError(t, err)
	defer us5()
	select {
	case listResp = <-ch5:
	case <-ctx.Done():
		t.Fatal("timeout listing after teardown")
	}
	for _, d := range listResp.Packages {
		assert.NotEqual(t, "lifecycle-e2e-deploy.ts", d.Source, "should be torn down")
	}
}
