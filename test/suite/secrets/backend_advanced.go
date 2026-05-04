package secrets

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// testSecretsOnTransport — secrets set+get roundtrip on the transport.
// Ported from adversarial/backend_advanced_test.go:TestBackendAdvanced_SecretsOnBackend.
func testSecretsOnTransport(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set
	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "transport-key-suite", Value: "transport-val"})

	// Get
	resp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "transport-key-suite"})
	assert.Equal(t, "transport-val", resp.Value)
}
