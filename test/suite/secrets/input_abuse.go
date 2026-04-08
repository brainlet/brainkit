package secrets

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInputAbuseEmptyName — empty secret name returns VALIDATION_ERROR.
func testInputAbuseEmptyName(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	payload, err := env.PublishAndWait(t, messages.SecretsSetMsg{Name: "", Value: "val"}, 5*time.Second)
	require.NoError(t, err)
	code := suite.ResponseCode(payload)
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// testInputAbuseLargeValue — storing a 100KB secret succeeds cleanly.
func testInputAbuseLargeValue(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	big := strings.Repeat("x", 100000) // 100KB secret
	ctx := context.Background()
	pub, err := sdk.Publish(env.Kit, ctx, messages.SecretsSetMsg{Name: "big-secret-sec-adv", Value: big})
	require.NoError(t, err)

	ch := make(chan messages.SecretsSetResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { ch <- resp })
	defer unsub()
	select {
	case resp := <-ch:
		assert.True(t, resp.Stored)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout storing large secret")
	}
}

// testInputAbuseSpecialCharsInName — secret names with special characters don't panic.
func testInputAbuseSpecialCharsInName(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	names := []string{"key/with/slashes", "key.with.dots", "key with spaces", "key=with=equals"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			payload, err := env.PublishAndWait(t, messages.SecretsSetMsg{Name: name, Value: "val"}, 5*time.Second)
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
		pub, err := sdk.Publish(env.Kit, ctx, messages.SecretsSetMsg{
			Name: name, Value: strings.Repeat("v", i+1),
		})
		require.NoError(t, err)
		ch := make(chan messages.SecretsSetResp, 1)
		unsub, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { ch <- resp })
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout on bulk set %d", i)
		}
		unsub()
	}

	// List should return without error or hang
	pub, _ := sdk.Publish(env.Kit, ctx, messages.SecretsListMsg{})
	listCh := make(chan messages.SecretsListResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.SecretsListResp](env.Kit, ctx, pub.ReplyTo, func(resp messages.SecretsListResp, _ messages.Message) { listCh <- resp })
	defer unsub()
	select {
	case resp := <-listCh:
		assert.Greater(t, len(resp.Secrets), 0, "should list at least some secrets")
	case <-time.After(10 * time.Second):
		t.Fatal("timeout on bulk list")
	}
}
