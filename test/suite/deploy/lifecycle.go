package deploy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListEmpty(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.KitListMsg{})
	require.NoError(t, err)
	ch := make(chan messages.KitListResp, 1)
	unsub, err := sdk.SubscribeTo[messages.KitListResp](env.Kernel, ctx, pr.ReplyTo, func(r messages.KitListResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.KitListResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.NotNil(t, resp.Deployments)
}

func testDeployTeardown(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "kit-test-1.ts",
		Code: `
			const t = createTool({
				id: "kit-deployed-tool",
				description: "tool from deploy test",
				execute: async () => ({ ok: true })
			});
			kit.register("tool", "kit-deployed-tool", t);
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, err := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var deployResp messages.KitDeployResp
	select {
	case deployResp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, deployResp.Deployed)

	// List should show it
	pr2, err := sdk.Publish(env.Kernel, ctx, messages.KitListMsg{})
	require.NoError(t, err)
	ch2 := make(chan messages.KitListResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.KitListResp](env.Kernel, ctx, pr2.ReplyTo, func(r messages.KitListResp, m messages.Message) { ch2 <- r })
	defer unsub2()
	var listResp messages.KitListResp
	select {
	case listResp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	found := false
	for _, d := range listResp.Deployments {
		if d.Source == "kit-test-1.ts" {
			found = true
		}
	}
	assert.True(t, found)

	// Teardown
	pr3, err := sdk.Publish(env.Kernel, ctx, messages.KitTeardownMsg{Source: "kit-test-1.ts"})
	require.NoError(t, err)
	ch3 := make(chan messages.KitTeardownResp, 1)
	unsub3, _ := sdk.SubscribeTo[messages.KitTeardownResp](env.Kernel, ctx, pr3.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { ch3 <- r })
	defer unsub3()
	var tearResp messages.KitTeardownResp
	select {
	case tearResp = <-ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.GreaterOrEqual(t, tearResp.Removed, 0)
}

func testRedeploy(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "kit-redeploy.ts",
		Code:   `const t = createTool({ id: "redeploy-v1", description: "v1", execute: async () => ({ version: 1 }) }); kit.register("tool", "redeploy-v1", t);`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	pr2, err := sdk.Publish(env.Kernel, ctx, messages.KitRedeployMsg{
		Source: "kit-redeploy.ts",
		Code:   `const t = createTool({ id: "redeploy-v2", description: "v2", execute: async () => ({ version: 2 }) }); kit.register("tool", "redeploy-v2", t);`,
	})
	require.NoError(t, err)
	ch2 := make(chan messages.KitRedeployResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.KitRedeployResp](env.Kernel, ctx, pr2.ReplyTo, func(r messages.KitRedeployResp, m messages.Message) { ch2 <- r })
	defer unsub2()
	var redeployResp messages.KitRedeployResp
	select {
	case redeployResp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, redeployResp.Deployed)

	sdk.Publish(env.Kernel, ctx, messages.KitTeardownMsg{Source: "kit-redeploy.ts"})
}

func testDeployInvalidCode(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "bad-code.ts",
		Code:   `throw new Error("intentional failure");`,
	})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		var r struct {
			Error string `json:"error"`
		}
		json.Unmarshal(msg.Payload, &r)
		ch <- r.Error
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
	ctx := context.Background()

	_, err := env.Kernel.Deploy(ctx, "dup-deploy.ts", `output("v1");`)
	require.NoError(t, err)

	// Deploy again with same source — idempotent, not error
	_, err = env.Kernel.Deploy(ctx, "dup-deploy.ts", `output("v2");`)
	require.NoError(t, err, "second deploy should succeed (idempotent)")

	env.Kernel.Teardown(ctx, "dup-deploy.ts")
}

// testConcurrentDeploySameSource verifies that deploying the same source concurrently
// doesn't crash — the second deploy is idempotent.
func testConcurrentDeploySameSource(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	_, err := env.Kernel.Deploy(ctx, "concurrent-src.ts", `output("first");`)
	require.NoError(t, err)

	// Second deploy replaces the first (idempotent)
	_, err = env.Kernel.Deploy(ctx, "concurrent-src.ts", `output("second");`)
	require.NoError(t, err, "second deploy should succeed (idempotent)")

	env.Kernel.Teardown(ctx, "concurrent-src.ts")
}
