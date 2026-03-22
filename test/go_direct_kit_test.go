package test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDirect_Kit(t *testing.T) {
	for _, factory := range []struct {
		name string
		make func(t *testing.T) sdk.Runtime
	}{
		{"Kernel", newTestKernel},
		{"Node", newTestNode},
	} {
		t.Run(factory.name, func(t *testing.T) {
			rt := factory.make(t)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			t.Run("List_Empty", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.KitListMsg, messages.KitListResp](rt, ctx, messages.KitListMsg{})
				require.NoError(t, err)
				assert.NotNil(t, resp.Deployments)
			})

			t.Run("Deploy_Teardown", func(t *testing.T) {
				deployResp, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "kit-test-1.ts",
					Code: `
						const t = createTool({
							id: "kit-deployed-tool",
							description: "tool from deploy test",
							execute: async () => ({ ok: true })
						});
					`,
				})
				require.NoError(t, err)
				assert.True(t, deployResp.Deployed)

				// List should show it
				listResp, err := sdk.PublishAwait[messages.KitListMsg, messages.KitListResp](rt, ctx, messages.KitListMsg{})
				require.NoError(t, err)
				found := false
				for _, d := range listResp.Deployments {
					if d.Source == "kit-test-1.ts" {
						found = true
					}
				}
				assert.True(t, found)

				// Teardown
				teardownResp, err := sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "kit-test-1.ts"})
				require.NoError(t, err)
				assert.GreaterOrEqual(t, teardownResp.Removed, 0)
			})

			t.Run("Redeploy", func(t *testing.T) {
				// Deploy v1
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "kit-redeploy.ts",
					Code:   `const t = createTool({ id: "redeploy-v1", description: "v1", execute: async () => ({ version: 1 }) });`,
				})
				require.NoError(t, err)

				// Redeploy v2
				redeployResp, err := sdk.PublishAwait[messages.KitRedeployMsg, messages.KitRedeployResp](rt, ctx, messages.KitRedeployMsg{
					Source: "kit-redeploy.ts",
					Code:   `const t = createTool({ id: "redeploy-v2", description: "v2", execute: async () => ({ version: 2 }) });`,
				})
				require.NoError(t, err)
				assert.True(t, redeployResp.Deployed)

				// Teardown
				_, _ = sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "kit-redeploy.ts"})
			})

			t.Run("Deploy_InvalidCode", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "bad-code.ts",
					Code:   `throw new Error("intentional failure");`,
				})
				assert.Error(t, err)
			})

			t.Run("Deploy_Duplicate", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "dup.ts",
					Code:   `const t = createTool({ id: "dup-tool", description: "dup", execute: async () => ({}) });`,
				})
				require.NoError(t, err)

				// Deploy again with same source — should fail
				_, err = sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "dup.ts",
					Code:   `const t = createTool({ id: "dup-tool-2", description: "dup2", execute: async () => ({}) });`,
				})
				assert.Error(t, err, "duplicate deploy should fail")

				_, _ = sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "dup.ts"})
			})
		})
	}
}
