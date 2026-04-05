package secrets

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSecretsOnTransport — secrets set+get roundtrip on the transport.
// Ported from adversarial/backend_advanced_test.go:TestBackendAdvanced_SecretsOnBackend.
func testSecretsOnTransport(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set
	pr1, err := sdk.Publish(env.Kernel, ctx, messages.SecretsSetMsg{Name: "transport-key-suite", Value: "transport-val"})
	require.NoError(t, err)
	ch1 := make(chan []byte, 1)
	unsub1, err := env.Kernel.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	require.NoError(t, err)
	select {
	case <-ch1:
	case <-ctx.Done():
		t.Fatal("timeout set")
	}
	unsub1()

	// Get
	pr2, err := sdk.Publish(env.Kernel, ctx, messages.SecretsGetMsg{Name: "transport-key-suite"})
	require.NoError(t, err)
	ch2 := make(chan []byte, 1)
	unsub2, err := env.Kernel.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	require.NoError(t, err)
	defer unsub2()

	select {
	case p := <-ch2:
		assert.Contains(t, string(p), "transport-val")
	case <-ctx.Done():
		t.Fatal("timeout get")
	}
}
