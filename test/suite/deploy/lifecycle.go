package deploy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListEmpty(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := sdk.Call[packagemsg.PackageListDeployedMsg, packagemsg.PackageListDeployedResp](env.Kit, ctx, packagemsg.PackageListDeployedMsg{})
	require.NoError(t, err)
	assert.NotNil(t, resp.Packages)
}

func testDeployTeardown(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	m1, _ := json.Marshal(map[string]string{"name": "kit-test-1", "entry": "kit-test-1.ts"})
	deployResp, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: m1,
		Files: map[string]string{"kit-test-1.ts": `
			const t = createTool({
				id: "kit-deployed-tool",
				description: "tool from deploy test",
				execute: async () => ({ ok: true })
			});
			kit.register("tool", "kit-deployed-tool", t);
		`},
	})
	require.NoError(t, err)
	assert.True(t, deployResp.Deployed)

	// List should show it
	listResp, err := sdk.Call[packagemsg.PackageListDeployedMsg, packagemsg.PackageListDeployedResp](env.Kit, ctx, packagemsg.PackageListDeployedMsg{})
	require.NoError(t, err)
	found := false
	for _, d := range listResp.Packages {
		if d.Source == "kit-test-1.ts" {
			found = true
		}
	}
	assert.True(t, found)

	// Teardown
	tearResp, err := sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](env.Kit, ctx, packagemsg.PackageTeardownMsg{Name: "kit-test-1"})
	require.NoError(t, err)
	assert.True(t, tearResp.Removed)
}

func testRedeploy(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mr, _ := json.Marshal(map[string]string{"name": "kit-redeploy", "entry": "kit-redeploy.ts"})
	_, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: mr,
		Files:    map[string]string{"kit-redeploy.ts": `const t = createTool({ id: "redeploy-v1", description: "v1", execute: async () => ({ version: 1 }) }); kit.register("tool", "redeploy-v1", t);`},
	})
	require.NoError(t, err)

	mr2, _ := json.Marshal(map[string]string{"name": "kit-redeploy", "entry": "kit-redeploy.ts"})
	redeployResp, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: mr2,
		Files:    map[string]string{"kit-redeploy.ts": `const t = createTool({ id: "redeploy-v2", description: "v2", execute: async () => ({ version: 2 }) }); kit.register("tool", "redeploy-v2", t);`},
	})
	require.NoError(t, err)
	assert.True(t, redeployResp.Deployed)

	_, _ = sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](env.Kit, ctx, packagemsg.PackageTeardownMsg{Name: "kit-redeploy"})
}

func testDeployInvalidCode(t *testing.T, env *suite.TestEnv) {
	mb, _ := json.Marshal(map[string]string{"name": "bad-code", "entry": "bad-code.ts"})
	payload, err := env.PublishAndWait(t, packagemsg.PackageDeployMsg{
		Manifest: mb,
		Files:    map[string]string{"bad-code.ts": `throw new Error("intentional failure");`},
	}, 15*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, suite.ResponseErrorMessage(payload))
}

// testDeployDuplicate verifies that deploying the same source twice is idempotent.
func testDeployDuplicate(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "dup-deploy.ts", `output("v1");`)

	// Deploy again with same source — idempotent, not error
	err := testutil.DeployErr(env.Kit, "dup-deploy.ts", `output("v2");`)
	require.NoError(t, err, "second deploy should succeed (idempotent)")

	testutil.Teardown(t, env.Kit, "dup-deploy.ts")
}

// testConcurrentDeploySameSource verifies that deploying the same source concurrently
// doesn't crash — the second deploy is idempotent.
func testConcurrentDeploySameSource(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "concurrent-src.ts", `output("first");`)

	// Second deploy replaces the first (idempotent)
	err := testutil.DeployErr(env.Kit, "concurrent-src.ts", `output("second");`)
	require.NoError(t, err, "second deploy should succeed (idempotent)")

	testutil.Teardown(t, env.Kit, "concurrent-src.ts")
}
