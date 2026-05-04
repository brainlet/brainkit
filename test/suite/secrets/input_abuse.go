package secrets

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInputAbuseEmptyName — empty secret name returns VALIDATION_ERROR.
func testInputAbuseEmptyName(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	payload, err := env.PublishAndWait(t, secretmsg.SecretsSetMsg{Name: "", Value: "val"}, 5*time.Second)
	require.NoError(t, err)
	code := suite.ResponseCode(payload)
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// testInputAbuseLargeValue — storing a 100KB secret succeeds cleanly.
func testInputAbuseLargeValue(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	big := strings.Repeat("x", 100000) // 100KB secret
	ctx := context.Background()
	resp := callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "big-secret-sec-adv", Value: big})
	assert.True(t, resp.Stored)
}

// testInputAbuseSpecialCharsInName — secret names with special characters don't panic.
func testInputAbuseSpecialCharsInName(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	names := []string{"key/with/slashes", "key.with.dots", "key with spaces", "key=with=equals"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			payload, err := env.PublishAndWait(t, secretmsg.SecretsSetMsg{Name: name, Value: "val"}, 5*time.Second)
			require.NoError(t, err)
			// Should succeed or error cleanly — never panic
			_ = payload
		})
	}
}

// testInputAbuseBulkOperations — set 20 secrets via individual calls, verify via list.
func testInputAbuseBulkOperations(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := env.T.Context()

	for i := 0; i < 20; i++ {
		name := strings.Join([]string{"bulk-sec-adv", strings.Repeat("x", i%5)}, "-")
		resp := callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{
			Name: name, Value: strings.Repeat("v", i+1),
		})
		assert.True(t, resp.Stored)
	}

	// List should return without error or hang
	resp := callSecretList(t, env.Kit, ctx, secretmsg.SecretsListMsg{})
	assert.Greater(t, len(resp.Secrets), 0, "should list at least some secrets")
}
