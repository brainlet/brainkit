package secrets

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSecretsRotation — integration: set secret, rotate, verify new value.
func testSecretsRotation(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := env.T.Context()

	// Set
	pub1, err := protocol.Publish(env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "rotate-key-sec-adv", Value: "old-value"})
	require.NoError(t, err)
	setCh := make(chan secretmsg.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[secretmsg.SecretsSetResp](env.Kit, ctx, pub1.ReplyTo, func(resp secretmsg.SecretsSetResp, _ sdk.Message) { setCh <- resp })
	select {
	case <-setCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout setting secret")
	}
	cancelSet()

	// Rotate
	pub2, _ := protocol.Publish(env.Kit, ctx, secretmsg.SecretsRotateMsg{Name: "rotate-key-sec-adv", NewValue: "new-value"})
	rotateCh := make(chan secretmsg.SecretsRotateResp, 1)
	cancelRotate, _ := sdk.SubscribeTo[secretmsg.SecretsRotateResp](env.Kit, ctx, pub2.ReplyTo, func(resp secretmsg.SecretsRotateResp, _ sdk.Message) { rotateCh <- resp })
	select {
	case resp := <-rotateCh:
		assert.True(t, resp.Rotated)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout rotating secret")
	}
	cancelRotate()

	// Verify the new value is returned
	pub3, _ := protocol.Publish(env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "rotate-key-sec-adv"})
	getCh := make(chan secretmsg.SecretsGetResp, 1)
	cancelGet, _ := sdk.SubscribeTo[secretmsg.SecretsGetResp](env.Kit, ctx, pub3.ReplyTo, func(resp secretmsg.SecretsGetResp, _ sdk.Message) { getCh <- resp })
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
	pr1, _ := protocol.Publish(env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "rotate-key-e2e", Value: "v1"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := env.Kit.SubscribeRaw(ctx, pr1.ReplyTo, func(m sdk.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Rotate
	pr2, _ := protocol.Publish(env.Kit, ctx, secretmsg.SecretsRotateMsg{Name: "rotate-key-e2e", NewValue: "v2"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := env.Kit.SubscribeRaw(ctx, pr2.ReplyTo, func(m sdk.Message) { ch2 <- m.Payload })
	p2 := <-ch2
	unsub2()
	assert.Contains(t, string(p2), "rotated")

	// Get — should be v2
	pr3, _ := protocol.Publish(env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "rotate-key-e2e"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := env.Kit.SubscribeRaw(ctx, pr3.ReplyTo, func(m sdk.Message) { ch3 <- m.Payload })
	defer unsub3()

	p3 := <-ch3
	var resp struct {
		Value string `json:"value"`
	}
	json.Unmarshal(suite.ResponseData(p3), &resp)
	assert.Equal(t, "v2", resp.Value)
}
