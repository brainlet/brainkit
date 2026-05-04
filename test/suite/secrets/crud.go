package secrets

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// secretsEnv creates a fresh kernel with persistence + secret key.
func secretsEnv(t *testing.T) *suite.TestEnv {
	t.Helper()
	return suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-master-key-for-secrets!!!"))
}

func testSetAndGet(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	resp := callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "api-key", Value: "sk-test-12345"})
	assert.True(t, resp.Stored)
	assert.Equal(t, 1, resp.Version)

	getResp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "api-key"})
	assert.Equal(t, "sk-test-12345", getResp.Value)
}

func testDelete(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "temp", Value: "val"})
	delResp := callSecretDelete(t, env.Kit, ctx, secretmsg.SecretsDeleteMsg{Name: "temp"})
	assert.True(t, delResp.Deleted)
	getResp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "temp"})
	assert.Empty(t, getResp.Value)
}

func testList(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	for _, name := range []string{"key-a", "key-b"} {
		callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: name, Value: "val-" + name})
	}

	resp := callSecretList(t, env.Kit, ctx, secretmsg.SecretsListMsg{})
	assert.Len(t, resp.Secrets, 2)
	for _, s := range resp.Secrets {
		assert.NotEmpty(t, s.Name)
	}
}

func testRotate(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "rotate-me", Value: "old-value"})
	rotateResp := callSecretRotate(t, env.Kit, ctx, secretmsg.SecretsRotateMsg{Name: "rotate-me", NewValue: "new-value", Restart: false})
	assert.True(t, rotateResp.Rotated)
	assert.Equal(t, 2, rotateResp.Version)
	getResp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "rotate-me"})
	assert.Equal(t, "new-value", getResp.Value)
}

func testJSBridge(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "js-test-token", Value: "tok_abc123"})

	result := testutil.EvalTS(t, env.Kit, "__test_secret.ts", `
		var val = secrets.get("js-test-token");
		return val;
	`)
	assert.Equal(t, "tok_abc123", result)
}

func testAuditEvents(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	storedCh := make(chan secretmsg.SecretsStoredEvent, 1)
	cancelStored, _ := sdk.SubscribeTo[secretmsg.SecretsStoredEvent](env.Kit, ctx, "secrets.stored", func(evt secretmsg.SecretsStoredEvent, _ sdk.Message) { storedCh <- evt })
	defer cancelStored()

	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "audit-test", Value: "val"})

	select {
	case evt := <-storedCh:
		assert.Equal(t, "audit-test", evt.Name)
		assert.Equal(t, 1, evt.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testConcurrentAccess(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "concurrent", Value: "v0"})

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			resp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "concurrent"})
			if resp.Value != "v0" {
				t.Errorf("concurrent get: expected %q, got %q", "v0", resp.Value)
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func testDevModeNoEncryption(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence()) // no secret key
	ctx := context.Background()

	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "dev-secret", Value: "unencrypted"})
	resp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "dev-secret"})
	assert.Equal(t, "unencrypted", resp.Value)
}

func testListNeverLeaksValues(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "sensitive-key", Value: "sk-super-secret-do-not-leak"})
	resp := callSecretList(t, env.Kit, ctx, secretmsg.SecretsListMsg{})
	raw, _ := json.Marshal(resp)
	assert.False(t, strings.Contains(string(raw), "sk-super-secret-do-not-leak"), "list response must never contain secret value")
	assert.Len(t, resp.Secrets, 1)
	assert.Equal(t, "sensitive-key", resp.Secrets[0].Name)
}
