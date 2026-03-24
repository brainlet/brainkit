package test

import (
	"context"
	"encoding/json"
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
				_pr1, err := sdk.Publish(rt, ctx, messages.KitListMsg{})
				require.NoError(t, err)
				_ch1 := make(chan messages.KitListResp, 1)
				_us1, err := sdk.SubscribeTo[messages.KitListResp](rt, ctx, _pr1.ReplyTo, func(r messages.KitListResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.KitListResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.NotNil(t, resp.Deployments)
			})

			t.Run("Deploy_Teardown", func(t *testing.T) {
				_pr2, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
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
				_ch2 := make(chan messages.KitDeployResp, 1)
				_us2, err := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr2.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var deployResp messages.KitDeployResp
				select {
				case deployResp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, deployResp.Deployed)

				// List should show it
				_pr3, err := sdk.Publish(rt, ctx, messages.KitListMsg{})
				require.NoError(t, err)
				_ch3 := make(chan messages.KitListResp, 1)
				_us3, err := sdk.SubscribeTo[messages.KitListResp](rt, ctx, _pr3.ReplyTo, func(r messages.KitListResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var listResp messages.KitListResp
				select {
				case listResp = <-_ch3:
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
				_pr4, err := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "kit-test-1.ts"})
				require.NoError(t, err)
				_ch4 := make(chan messages.KitTeardownResp, 1)
				_us4, err := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _pr4.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _ch4 <- r })
				require.NoError(t, err)
				defer _us4()
				var teardownResp messages.KitTeardownResp
				select {
				case teardownResp = <-_ch4:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.GreaterOrEqual(t, teardownResp.Removed, 0)
			})

			t.Run("Redeploy", func(t *testing.T) {
				// Deploy v1
				_pr5, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "kit-redeploy.ts",
					Code:   `const t = createTool({ id: "redeploy-v1", description: "v1", execute: async () => ({ version: 1 }) }); kit.register("tool", "redeploy-v1", t);`,
				})
				require.NoError(t, err)
				_ch5 := make(chan messages.KitDeployResp, 1)
				_us5, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr5.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch5 <- r })
				defer _us5()
				select {
				case <-_ch5:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				// Redeploy v2
				_pr6, err := sdk.Publish(rt, ctx, messages.KitRedeployMsg{
					Source: "kit-redeploy.ts",
					Code:   `const t = createTool({ id: "redeploy-v2", description: "v2", execute: async () => ({ version: 2 }) }); kit.register("tool", "redeploy-v2", t);`,
				})
				require.NoError(t, err)
				_ch6 := make(chan messages.KitRedeployResp, 1)
				_us6, err := sdk.SubscribeTo[messages.KitRedeployResp](rt, ctx, _pr6.ReplyTo, func(r messages.KitRedeployResp, m messages.Message) { _ch6 <- r })
				require.NoError(t, err)
				defer _us6()
				var redeployResp messages.KitRedeployResp
				select {
				case redeployResp = <-_ch6:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, redeployResp.Deployed)

				// Teardown
				_, _ = sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "kit-redeploy.ts"})
			})

			t.Run("Deploy_InvalidCode", func(t *testing.T) {
				_epr1, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "bad-code.ts",
					Code:   `throw new Error("intentional failure");`,
				})
				require.NoError(t, err)
				_ech1 := make(chan string, 1)
				_eun1, _ := rt.SubscribeRaw(ctx, _epr1.ReplyTo, func(msg messages.Message) {
					var r struct { Error string `json:"error"` }
					json.Unmarshal(msg.Payload, &r)
					_ech1 <- r.Error
				})
				defer _eun1()
				select {
				case errMsg := <-_ech1:
					assert.NotEmpty(t, errMsg)
				case <-ctx.Done():
					t.Fatal("timeout")
				}
			})

			t.Run("Deploy_Duplicate", func(t *testing.T) {
				_pr9, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "dup.ts",
					Code:   `const t = createTool({ id: "dup-tool", description: "dup", execute: async () => ({}) }); kit.register("tool", "dup-tool", t);`,
				})
				require.NoError(t, err)
				_ch9 := make(chan messages.KitDeployResp, 1)
				_us9, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr9.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch9 <- r })
				defer _us9()
				select {
				case <-_ch9:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				// Deploy again with same source — should fail
				_epr2, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "dup.ts",
					Code:   `const t = createTool({ id: "dup-tool-2", description: "dup2", execute: async () => ({}) }); kit.register("tool", "dup-tool-2", t);`,
				})
				require.NoError(t, err)
				_ech2 := make(chan string, 1)
				_eun2, _ := rt.SubscribeRaw(ctx, _epr2.ReplyTo, func(msg messages.Message) {
					var r struct { Error string `json:"error"` }
					json.Unmarshal(msg.Payload, &r)
					_ech2 <- r.Error
				})
				defer _eun2()
				select {
				case errMsg := <-_ech2:
					assert.NotEmpty(t, errMsg)
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				_, _ = sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "dup.ts"})
			})
		})
	}
}
