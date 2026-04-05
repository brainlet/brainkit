package secrets

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSecretsRotation — integration: set secret, rotate, verify new value.
func testSecretsRotation(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := env.T.Context()

	// Set
	pub1, err := sdk.Publish(env.Kernel, ctx, messages.SecretsSetMsg{Name: "rotate-key-sec-adv", Value: "old-value"})
	require.NoError(t, err)
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kernel, ctx, pub1.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	select {
	case <-setCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout setting secret")
	}
	cancelSet()

	// Rotate
	pub2, _ := sdk.Publish(env.Kernel, ctx, messages.SecretsRotateMsg{Name: "rotate-key-sec-adv", NewValue: "new-value"})
	rotateCh := make(chan messages.SecretsRotateResp, 1)
	cancelRotate, _ := sdk.SubscribeTo[messages.SecretsRotateResp](env.Kernel, ctx, pub2.ReplyTo, func(resp messages.SecretsRotateResp, _ messages.Message) { rotateCh <- resp })
	select {
	case resp := <-rotateCh:
		assert.True(t, resp.Rotated)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout rotating secret")
	}
	cancelRotate()

	// Verify the new value is returned
	pub3, _ := sdk.Publish(env.Kernel, ctx, messages.SecretsGetMsg{Name: "rotate-key-sec-adv"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancelGet, _ := sdk.SubscribeTo[messages.SecretsGetResp](env.Kernel, ctx, pub3.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) { getCh <- resp })
	defer cancelGet()
	select {
	case resp := <-getCh:
		assert.Equal(t, "new-value", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout getting rotated secret")
	}
}

// testE2ESecretsRotateAndVerify — E2E: set secret, rotate, verify new value via raw bus.
// Faithfully migrated from adversarial/e2e_scenarios_test.go TestE2E_SecretsRotateAndVerify.
func testE2ESecretsRotateAndVerify(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := env.T.Context()

	// Set
	pr1, _ := sdk.Publish(env.Kernel, ctx, messages.SecretsSetMsg{Name: "rotate-key-e2e", Value: "v1"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := env.Kernel.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Rotate
	pr2, _ := sdk.Publish(env.Kernel, ctx, messages.SecretsRotateMsg{Name: "rotate-key-e2e", NewValue: "v2"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := env.Kernel.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	p2 := <-ch2
	unsub2()
	assert.Contains(t, string(p2), "rotated")

	// Get — should be v2
	pr3, _ := sdk.Publish(env.Kernel, ctx, messages.SecretsGetMsg{Name: "rotate-key-e2e"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := env.Kernel.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	defer unsub3()

	p3 := <-ch3
	var resp struct{ Value string `json:"value"` }
	json.Unmarshal(p3, &resp)
	assert.Equal(t, "v2", resp.Value)
}
