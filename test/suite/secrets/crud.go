package secrets

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// secretsEnv creates a fresh kernel with persistence + secret key.
func secretsEnv(t *testing.T) *suite.TestEnv {
	t.Helper()
	return suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-master-key-for-secrets!!!"))
}

func testSetAndGet(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	pub, err := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "api-key", Value: "sk-test-12345"})
	require.NoError(t, err)
	respCh := make(chan sdk.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { respCh <- resp })
	defer cancel()

	select {
	case resp := <-respCh:
		assert.True(t, resp.Stored)
		assert.Equal(t, 1, resp.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	pub2, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsGetMsg{Name: "api-key"})
	getCh := make(chan sdk.SecretsGetResp, 1)
	cancel2, _ := sdk.SubscribeTo[sdk.SecretsGetResp](env.Kit, ctx, pub2.ReplyTo, func(resp sdk.SecretsGetResp, _ sdk.Message) { getCh <- resp })
	defer cancel2()

	select {
	case resp := <-getCh:
		assert.Equal(t, "sk-test-12345", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testDelete(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "temp", Value: "val"})
	setCh := make(chan sdk.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { setCh <- resp })
	<-setCh
	cancel()

	pub2, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsDeleteMsg{Name: "temp"})
	delCh := make(chan sdk.SecretsDeleteResp, 1)
	cancel2, _ := sdk.SubscribeTo[sdk.SecretsDeleteResp](env.Kit, ctx, pub2.ReplyTo, func(resp sdk.SecretsDeleteResp, _ sdk.Message) { delCh <- resp })
	defer cancel2()

	select {
	case resp := <-delCh:
		assert.True(t, resp.Deleted)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	pub3, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsGetMsg{Name: "temp"})
	getCh := make(chan sdk.SecretsGetResp, 1)
	cancel3, _ := sdk.SubscribeTo[sdk.SecretsGetResp](env.Kit, ctx, pub3.ReplyTo, func(resp sdk.SecretsGetResp, _ sdk.Message) { getCh <- resp })
	defer cancel3()

	select {
	case resp := <-getCh:
		assert.Empty(t, resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testList(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	for _, name := range []string{"key-a", "key-b"} {
		pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: name, Value: "val-" + name})
		ch := make(chan sdk.SecretsSetResp, 1)
		cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { ch <- resp })
		<-ch
		cancel()
	}

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsListMsg{})
	listCh := make(chan sdk.SecretsListResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsListResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsListResp, _ sdk.Message) { listCh <- resp })
	defer cancel()

	select {
	case resp := <-listCh:
		assert.Len(t, resp.Secrets, 2)
		for _, s := range resp.Secrets {
			assert.NotEmpty(t, s.Name)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testRotate(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "rotate-me", Value: "old-value"})
	ch := make(chan sdk.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { ch <- resp })
	<-ch
	cancel()

	pub2, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsRotateMsg{Name: "rotate-me", NewValue: "new-value", Restart: false})
	rotateCh := make(chan sdk.SecretsRotateResp, 1)
	cancel2, _ := sdk.SubscribeTo[sdk.SecretsRotateResp](env.Kit, ctx, pub2.ReplyTo, func(resp sdk.SecretsRotateResp, _ sdk.Message) { rotateCh <- resp })
	defer cancel2()

	select {
	case resp := <-rotateCh:
		assert.True(t, resp.Rotated)
		assert.Equal(t, 2, resp.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	pub3, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsGetMsg{Name: "rotate-me"})
	getCh := make(chan sdk.SecretsGetResp, 1)
	cancel3, _ := sdk.SubscribeTo[sdk.SecretsGetResp](env.Kit, ctx, pub3.ReplyTo, func(resp sdk.SecretsGetResp, _ sdk.Message) { getCh <- resp })
	defer cancel3()

	select {
	case resp := <-getCh:
		assert.Equal(t, "new-value", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testJSBridge(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "js-test-token", Value: "tok_abc123"})
	ch := make(chan sdk.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { ch <- resp })
	<-ch
	cancel()

	result := testutil.EvalTS(t, env.Kit, "__test_secret.ts", `
		var val = secrets.get("js-test-token");
		return val;
	`)
	assert.Equal(t, "tok_abc123", result)
}

func testAuditEvents(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	storedCh := make(chan sdk.SecretsStoredEvent, 1)
	cancelStored, _ := sdk.SubscribeTo[sdk.SecretsStoredEvent](env.Kit, ctx, "secrets.stored", func(evt sdk.SecretsStoredEvent, _ sdk.Message) { storedCh <- evt })
	defer cancelStored()

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "audit-test", Value: "val"})
	setCh := make(chan sdk.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { setCh <- resp })
	<-setCh
	cancelSet()

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

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "concurrent", Value: "v0"})
	ch := make(chan sdk.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { ch <- resp })
	<-ch
	cancel()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsGetMsg{Name: "concurrent"})
			getCh := make(chan sdk.SecretsGetResp, 1)
			cancel, _ := sdk.SubscribeTo[sdk.SecretsGetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsGetResp, _ sdk.Message) { getCh <- resp })
			select {
			case resp := <-getCh:
				cancel()
				if resp.Value != "v0" {
					t.Errorf("concurrent get: expected %q, got %q", "v0", resp.Value)
				}
			case <-time.After(5 * time.Second):
				cancel()
				t.Error("concurrent get: timeout")
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

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "dev-secret", Value: "unencrypted"})
	setCh := make(chan sdk.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { setCh <- resp })
	<-setCh
	cancel()

	pub2, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsGetMsg{Name: "dev-secret"})
	getCh := make(chan sdk.SecretsGetResp, 1)
	cancel2, _ := sdk.SubscribeTo[sdk.SecretsGetResp](env.Kit, ctx, pub2.ReplyTo, func(resp sdk.SecretsGetResp, _ sdk.Message) { getCh <- resp })
	defer cancel2()

	select {
	case resp := <-getCh:
		assert.Equal(t, "unencrypted", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testListNeverLeaksValues(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "sensitive-key", Value: "sk-super-secret-do-not-leak"})
	ch := make(chan sdk.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.SecretsSetResp, _ sdk.Message) { ch <- resp })
	<-ch
	cancel()

	pub2, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsListMsg{})
	listCh := make(chan sdk.SecretsListResp, 1)
	cancel2, _ := sdk.SubscribeTo[sdk.SecretsListResp](env.Kit, ctx, pub2.ReplyTo, func(resp sdk.SecretsListResp, _ sdk.Message) { listCh <- resp })
	defer cancel2()

	select {
	case resp := <-listCh:
		raw, _ := json.Marshal(resp)
		assert.False(t, strings.Contains(string(raw), "sk-super-secret-do-not-leak"), "list response must never contain secret value")
		assert.Len(t, resp.Secrets, 1)
		assert.Equal(t, "sensitive-key", resp.Secrets[0].Name)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
