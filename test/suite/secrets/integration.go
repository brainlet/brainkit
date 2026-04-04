package secrets

import (
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
