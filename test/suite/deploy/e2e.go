package deploy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
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
	payload, ok := env.SendAndReceive(t, toolmsg.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "1")

	// Teardown
	testutil.Teardown(t, env.Kit, "lifecycle-deploy-adv.ts")

	// Tool should be gone
	payload2, ok2 := env.SendAndReceive(t, toolmsg.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok2)
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload2))

	// Redeploy v2
	testutil.Deploy(t, env.Kit, "lifecycle-deploy-adv.ts", `
		const t = createTool({id: "lc-adv-tool", description: "v2", execute: async () => ({version: 2})});
		kit.register("tool", "lc-adv-tool", t);
	`)

	// Call v2
	payload3, ok3 := env.SendAndReceive(t, toolmsg.ToolCallMsg{Name: "lc-adv-tool", Input: map[string]any{}}, 5*time.Second)
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
	payload, ok := freshEnv.SendAndReceive(t, toolmsg.ToolCallMsg{Name: "recovered-deploy", Input: map[string]any{}}, 5*time.Second)
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
	_, err := packagemsg.CallPackageDeploy(freshEnv.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: mp1,
		Files:    map[string]string{"lifecycle-e2e-deploy.ts": `const v1 = createTool({ id: "version-check-e2e", description: "v1", execute: async () => ({ version: 1 }) }); kit.register("tool", "version-check-e2e", v1);`},
	}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)

	// List — should show lifecycle-e2e-deploy
	listResp, err := packagemsg.CallPackageListDeployed(freshEnv.Kit, ctx, packagemsg.PackageListDeployedMsg{}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	sources := make(map[string]bool)
	for _, d := range listResp.Packages {
		sources[d.Source] = true
	}
	assert.True(t, sources["lifecycle-e2e-deploy.ts"])

	// Redeploy with v2 (hot-replace via same deploy message)
	mp3, _ := json.Marshal(map[string]string{"name": "lifecycle-e2e-deploy", "entry": "lifecycle-e2e-deploy.ts"})
	_, err = packagemsg.CallPackageDeploy(freshEnv.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: mp3,
		Files:    map[string]string{"lifecycle-e2e-deploy.ts": `const v2 = createTool({ id: "version-check-e2e-v2", description: "v2", execute: async () => ({ version: 2 }) }); kit.register("tool", "version-check-e2e-v2", v2);`},
	}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)

	// Teardown
	_, err = packagemsg.CallPackageTeardown(freshEnv.Kit, ctx, packagemsg.PackageTeardownMsg{Name: "lifecycle-e2e-deploy"}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)

	// List — should not contain lifecycle-e2e-deploy.ts
	listResp, err = packagemsg.CallPackageListDeployed(freshEnv.Kit, ctx, packagemsg.PackageListDeployedMsg{}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	for _, d := range listResp.Packages {
		assert.NotEqual(t, "lifecycle-e2e-deploy.ts", d.Source, "should be torn down")
	}
}
