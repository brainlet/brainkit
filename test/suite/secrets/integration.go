package secrets

import (
	"testing"

	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// testSecretsRotation — integration: set secret, rotate, verify new value.
func testSecretsRotation(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := env.T.Context()

	// Set
	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "rotate-key-sec-adv", Value: "old-value"})

	// Rotate
	rotateResp := callSecretRotate(t, env.Kit, ctx, secretmsg.SecretsRotateMsg{Name: "rotate-key-sec-adv", NewValue: "new-value"})
	assert.True(t, rotateResp.Rotated)

	// Verify the new value is returned
	resp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "rotate-key-sec-adv"})
	assert.Equal(t, "new-value", resp.Value)
}

// testE2ESecretsRotateAndVerify — E2E: set secret, rotate, verify new value via typed bus calls.
// Faithfully migrated from adversarial/e2e_scenarios_test.go TestE2E_SecretsRotateAndVerify.
func testE2ESecretsRotateAndVerify(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := env.T.Context()

	// Set
	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "rotate-key-e2e", Value: "v1"})

	// Rotate
	rotateResp := callSecretRotate(t, env.Kit, ctx, secretmsg.SecretsRotateMsg{Name: "rotate-key-e2e", NewValue: "v2"})
	assert.True(t, rotateResp.Rotated)

	// Get — should be v2
	resp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "rotate-key-e2e"})
	assert.Equal(t, "v2", resp.Value)
}
