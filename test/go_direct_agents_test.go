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

func TestGoDirect_Agents(t *testing.T) {
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
				_pr1, err := sdk.Publish(rt, ctx, messages.AgentListMsg{})
				require.NoError(t, err)
				_ch1 := make(chan messages.AgentListResp, 1)
				_us1, err := sdk.SubscribeTo[messages.AgentListResp](rt, ctx, _pr1.ReplyTo, func(r messages.AgentListResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.AgentListResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Empty(t, resp.Agents)
			})

			t.Run("Discover_NoMatch", func(t *testing.T) {
				_pr1, err := sdk.Publish(rt, ctx, messages.AgentDiscoverMsg{
					Capability: "teleportation",
				})
				require.NoError(t, err)
				_ch1 := make(chan messages.AgentDiscoverResp, 1)
				_us1, err := sdk.SubscribeTo[messages.AgentDiscoverResp](rt, ctx, _pr1.ReplyTo, func(r messages.AgentDiscoverResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.AgentDiscoverResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Empty(t, resp.Agents)
			})

			t.Run("GetStatus_NotFound", func(t *testing.T) {
				_pr2, err := sdk.Publish(rt, ctx, messages.AgentGetStatusMsg{
					Name: "ghost-agent",
				})
				assert.Error(t, err)
			})

			t.Run("SetStatus_NotFound", func(t *testing.T) {
				_pr3, err := sdk.Publish(rt, ctx, messages.AgentSetStatusMsg{
					Name: "ghost-agent", Status: "busy",
				})
				assert.Error(t, err)
			})

			t.Run("SetStatus_InvalidStatus", func(t *testing.T) {
				_pr4, err := sdk.Publish(rt, ctx, messages.AgentSetStatusMsg{
					Name: "any", Status: "flying",
				})
				assert.Error(t, err)
			})

			t.Run("Deploy_Agent_Then_List", func(t *testing.T) {
				if !hasAIKey() {
					t.Skip("OPENAI_API_KEY required for agent deployment")
				}

				// Deploy .ts that creates an agent with string model reference
				_pr5, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "test-agent.ts",
					Code: `
						const a = agent({
							name: "test-helper",
							instructions: "You are a test helper. Reply with exactly: OK",
							model: "openai:gpt-4o-mini",
						});
					`,
				})
				if err != nil {
					t.Skipf("agent deployment failed (provider may not be configured): %v", err)
				}

				// List should find it
				_pr2, err := sdk.Publish(rt, ctx, messages.AgentListMsg{})
				require.NoError(t, err)
				_ch2 := make(chan messages.AgentListResp, 1)
				_us2, err := sdk.SubscribeTo[messages.AgentListResp](rt, ctx, _pr2.ReplyTo, func(r messages.AgentListResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var listResp messages.AgentListResp
				select {
				case listResp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				found := false
				for _, a := range listResp.Agents {
					if a.Name == "test-helper" {
						found = true
					}
				}
				assert.True(t, found, "deployed agent should appear in list")

				// Get status
				_pr3, err := sdk.Publish(rt, ctx, messages.AgentGetStatusMsg{Name: "test-helper"})
				require.NoError(t, err)
				_ch3 := make(chan messages.AgentGetStatusResp, 1)
				_us3, err := sdk.SubscribeTo[messages.AgentGetStatusResp](rt, ctx, _pr3.ReplyTo, func(r messages.AgentGetStatusResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var statusResp messages.AgentGetStatusResp
				select {
				case statusResp = <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "idle", statusResp.Status)

				// Set status
				_pr6, err := sdk.Publish(rt, ctx, messages.AgentSetStatusMsg{
					Name: "test-helper", Status: "busy",
				})
				require.NoError(t, err)
				_ch6 := make(chan messages.AgentSetStatusResp, 1)
				_us6, _ := sdk.SubscribeTo[messages.AgentSetStatusResp](rt, ctx, _pr6.ReplyTo, func(r messages.AgentSetStatusResp, m messages.Message) { _ch6 <- r })
				defer _us6()
				select {
				case <-_ch6:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				_pr4, err := sdk.Publish(rt, ctx, messages.AgentGetStatusMsg{Name: "test-helper"})
				require.NoError(t, err)
				_ch4 := make(chan messages.AgentGetStatusResp, 1)
				_us4, err := sdk.SubscribeTo[messages.AgentGetStatusResp](rt, ctx, _pr4.ReplyTo, func(r messages.AgentGetStatusResp, m messages.Message) { _ch4 <- r })
				require.NoError(t, err)
				defer _us4()
				var statusResp messages.AgentGetStatusResp
				select {
				case statusResp = <-_ch4:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "busy", statusResp.Status)

				// Teardown
				_pr5, _ := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "test-agent.ts"})
			})

			t.Run("Request_NotFound", func(t *testing.T) {
				_pr7, err := sdk.Publish(rt, ctx, messages.AgentRequestMsg{
					Name: "ghost-agent", Prompt: "hello",
				})
				assert.Error(t, err)
			})

			t.Run("Message_NotFound", func(t *testing.T) {
				_pr8, err := sdk.Publish(rt, ctx, messages.AgentMessageMsg{
					Target: "ghost-agent", Payload: "hello",
				})
				assert.Error(t, err)
			})

			t.Run("Message_Delivered", func(t *testing.T) {
				if !hasAIKey() {
					t.Skip("OPENAI_API_KEY required for agent deployment")
				}

				// Deploy an agent so it exists in the registry
				_pr9, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "msg-agent.ts",
					Code: `
						const a = agent({
							name: "msg-target",
							instructions: "You are a test agent.",
							model: "openai/gpt-4o-mini",
						});
					`,
				})
				if err != nil {
					t.Skipf("agent deploy failed: %v", err)
				}
				defer sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "msg-agent.ts"})

				// Send a message — fire-and-forget, should return delivered: true
				_pr10, err := sdk.Publish(rt, ctx, messages.AgentMessageMsg{
					Target: "msg-target", Payload: map[string]string{"text": "hello agent"},
				})
				require.NoError(t, err)
				_ch10 := make(chan messages.AgentMessageResp, 1)
				_us10, err := sdk.SubscribeTo[messages.AgentMessageResp](rt, ctx, _pr10.ReplyTo, func(r messages.AgentMessageResp, m messages.Message) { _ch10 <- r })
				require.NoError(t, err)
				defer _us10()
				var resp messages.AgentMessageResp
				select {
				case resp = <-_ch10:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, resp.Delivered)
			})
		})
	}
}
