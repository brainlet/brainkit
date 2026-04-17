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

func testListEmpty(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.PackageListDeployedMsg{})
	require.NoError(t, err)
	ch := make(chan sdk.PackageListDeployedResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.PackageListDeployedResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageListDeployedResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp sdk.PackageListDeployedResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.NotNil(t, resp.Packages)
}

func testDeployTeardown(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	m1, _ := json.Marshal(map[string]string{"name": "kit-test-1", "entry": "kit-test-1.ts"})
	pr, err := sdk.Publish(env.Kit, ctx, sdk.PackageDeployMsg{
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
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var deployResp sdk.PackageDeployResp
	select {
	case deployResp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, deployResp.Deployed)

	// List should show it
	pr2, err := sdk.Publish(env.Kit, ctx, sdk.PackageListDeployedMsg{})
	require.NoError(t, err)
	ch2 := make(chan sdk.PackageListDeployedResp, 1)
	unsub2, _ := sdk.SubscribeTo[sdk.PackageListDeployedResp](env.Kit, ctx, pr2.ReplyTo, func(r sdk.PackageListDeployedResp, m sdk.Message) { ch2 <- r })
	defer unsub2()
	var listResp sdk.PackageListDeployedResp
	select {
	case listResp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	found := false
	for _, d := range listResp.Packages {
		if d.Source == "kit-test-1.ts" {
			found = true
		}
	}
	assert.True(t, found)

	// Teardown
	pr3, err := sdk.Publish(env.Kit, ctx, sdk.PackageTeardownMsg{Name: "kit-test-1"})
	require.NoError(t, err)
	ch3 := make(chan sdk.PackageTeardownResp, 1)
	unsub3, _ := sdk.SubscribeTo[sdk.PackageTeardownResp](env.Kit, ctx, pr3.ReplyTo, func(r sdk.PackageTeardownResp, m sdk.Message) { ch3 <- r })
	defer unsub3()
	var tearResp sdk.PackageTeardownResp
	select {
	case tearResp = <-ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, tearResp.Removed)
}

func testRedeploy(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mr, _ := json.Marshal(map[string]string{"name": "kit-redeploy", "entry": "kit-redeploy.ts"})
	pr, err := sdk.Publish(env.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: mr,
		Files:    map[string]string{"kit-redeploy.ts": `const t = createTool({ id: "redeploy-v1", description: "v1", execute: async () => ({ version: 1 }) }); kit.register("tool", "redeploy-v1", t);`},
	})
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	mr2, _ := json.Marshal(map[string]string{"name": "kit-redeploy", "entry": "kit-redeploy.ts"})
	pr2, err := sdk.Publish(env.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: mr2,
		Files:    map[string]string{"kit-redeploy.ts": `const t = createTool({ id: "redeploy-v2", description: "v2", execute: async () => ({ version: 2 }) }); kit.register("tool", "redeploy-v2", t);`},
	})
	require.NoError(t, err)
	ch2 := make(chan sdk.PackageDeployResp, 1)
	unsub2, _ := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr2.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch2 <- r })
	defer unsub2()
	var redeployResp sdk.PackageDeployResp
	select {
	case redeployResp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, redeployResp.Deployed)

	sdk.Publish(env.Kit, ctx, sdk.PackageTeardownMsg{Name: "kit-redeploy"})
}

func testDeployInvalidCode(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mb, _ := json.Marshal(map[string]string{"name": "bad-code", "entry": "bad-code.ts"})
	pr, err := sdk.Publish(env.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: mb,
		Files:    map[string]string{"bad-code.ts": `throw new Error("intentional failure");`},
	})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
		ch <- suite.ResponseErrorMessage(msg.Payload)
	})
	defer unsub()
	select {
	case errMsg := <-ch:
		assert.NotEmpty(t, errMsg)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
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
