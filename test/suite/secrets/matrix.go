package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMatrixSetGetDeleteList — full lifecycle: set → get → list → delete → verify gone.
func testMatrixSetGetDeleteList(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set
	pub1, err := sdk.Publish(env.Kit, ctx, messages.SecretsSetMsg{Name: "lifecycle-key-sec-adv", Value: "lifecycle-val"})
	require.NoError(t, err)
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kit, ctx, pub1.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	select {
	case resp := <-setCh:
		assert.True(t, resp.Stored)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on set")
	}
	cancelSet()

	// Get
	pub2, _ := sdk.Publish(env.Kit, ctx, messages.SecretsGetMsg{Name: "lifecycle-key-sec-adv"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancelGet, _ := sdk.SubscribeTo[messages.SecretsGetResp](env.Kit, ctx, pub2.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) { getCh <- resp })
	select {
	case resp := <-getCh:
		assert.Equal(t, "lifecycle-val", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on get")
	}
	cancelGet()

	// List
	pub3, _ := sdk.Publish(env.Kit, ctx, messages.SecretsListMsg{})
	listCh := make(chan messages.SecretsListResp, 1)
	cancelList, _ := sdk.SubscribeTo[messages.SecretsListResp](env.Kit, ctx, pub3.ReplyTo, func(resp messages.SecretsListResp, _ messages.Message) { listCh <- resp })
	select {
	case resp := <-listCh:
		found := false
		for _, s := range resp.Secrets {
			if s.Name == "lifecycle-key-sec-adv" {
				found = true
			}
		}
		assert.True(t, found, "lifecycle key should appear in list")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on list")
	}
	cancelList()

	// Delete
	pub4, _ := sdk.Publish(env.Kit, ctx, messages.SecretsDeleteMsg{Name: "lifecycle-key-sec-adv"})
	delCh := make(chan messages.SecretsDeleteResp, 1)
	cancelDel, _ := sdk.SubscribeTo[messages.SecretsDeleteResp](env.Kit, ctx, pub4.ReplyTo, func(resp messages.SecretsDeleteResp, _ messages.Message) { delCh <- resp })
	select {
	case resp := <-delCh:
		assert.True(t, resp.Deleted)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on delete")
	}
	cancelDel()

	// Get after delete — should be empty
	pub5, _ := sdk.Publish(env.Kit, ctx, messages.SecretsGetMsg{Name: "lifecycle-key-sec-adv"})
	getCh2 := make(chan messages.SecretsGetResp, 1)
	cancelGet2, _ := sdk.SubscribeTo[messages.SecretsGetResp](env.Kit, ctx, pub5.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) { getCh2 <- resp })
	defer cancelGet2()
	select {
	case resp := <-getCh2:
		assert.Empty(t, resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on get-after-delete")
	}
}

// testMatrixRotate — set then rotate, verify version increments.
func testMatrixRotate(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set v1
	pub1, _ := sdk.Publish(env.Kit, ctx, messages.SecretsSetMsg{Name: "rot-key-sec-adv", Value: "v1"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kit, ctx, pub1.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	<-setCh
	cancelSet()

	// Rotate to v2
	pub2, _ := sdk.Publish(env.Kit, ctx, messages.SecretsRotateMsg{Name: "rot-key-sec-adv", NewValue: "v2"})
	rotateCh := make(chan messages.SecretsRotateResp, 1)
	cancelRotate, _ := sdk.SubscribeTo[messages.SecretsRotateResp](env.Kit, ctx, pub2.ReplyTo, func(resp messages.SecretsRotateResp, _ messages.Message) { rotateCh <- resp })
	select {
	case resp := <-rotateCh:
		assert.True(t, resp.Rotated)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on rotate")
	}
	cancelRotate()

	// Get — should be v2
	pub3, _ := sdk.Publish(env.Kit, ctx, messages.SecretsGetMsg{Name: "rot-key-sec-adv"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancelGet, _ := sdk.SubscribeTo[messages.SecretsGetResp](env.Kit, ctx, pub3.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) { getCh <- resp })
	defer cancelGet()
	select {
	case resp := <-getCh:
		assert.Equal(t, "v2", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on get")
	}
}

// testMatrixManySecrets — set 20 secrets, list them all.
func testMatrixManySecrets(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "bulk.db"))
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "bulk-test-key-32-characters!!",
	})
	require.NoError(t, err)
	defer k.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for i := 0; i < 20; i++ {
		pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{
			Name: fmt.Sprintf("bulk-key-sec-adv-%d", i), Value: fmt.Sprintf("val-%d", i),
		})
		ch := make(chan messages.SecretsSetResp, 1)
		unsub, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { ch <- resp })
		<-ch
		unsub()
	}

	pub, _ := sdk.Publish(k, ctx, messages.SecretsListMsg{})
	listCh := make(chan messages.SecretsListResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.SecretsListResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsListResp, _ messages.Message) { listCh <- resp })
	defer unsub()

	select {
	case resp := <-listCh:
		names := make(map[string]bool)
		for _, s := range resp.Secrets {
			names[s.Name] = true
		}
		for i := 0; i < 20; i++ {
			assert.True(t, names[fmt.Sprintf("bulk-key-sec-adv-%d", i)], "missing bulk-key-%d", i)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout on list")
	}
}

// testMatrixEncryptedPersistence — secrets survive restart with encryption.
func testMatrixEncryptedPersistence(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.db")
	masterKey := "test-encryption-key-32chars!!"

	// Phase 1: Set encrypted secret
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: masterKey,
	})
	require.NoError(t, err)

	ctx := context.Background()
	pub1, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "enc-key-sec-adv", Value: "enc-secret-val"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](k1, ctx, pub1.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	<-setCh
	cancelSet()
	k1.Close()

	// Phase 2: Reopen with same key, retrieve
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: masterKey,
	})
	require.NoError(t, err)
	defer k2.Close()

	pub2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "enc-key-sec-adv"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancelGet, _ := sdk.SubscribeTo[messages.SecretsGetResp](k2, ctx, pub2.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) { getCh <- resp })
	defer cancelGet()
	select {
	case resp := <-getCh:
		assert.Equal(t, "enc-secret-val", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout on get after restart")
	}
}

// testMatrixWrongKeyCannotDecrypt — wrong master key fails to decrypt.
func testMatrixWrongKeyCannotDecrypt(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.db")

	// Set with key A
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "correct-key-32-characters-long!",
	})
	require.NoError(t, err)

	ctx := context.Background()
	pub1, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "protected-sec-adv", Value: "sensitive"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](k1, ctx, pub1.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	<-setCh
	cancelSet()
	k1.Close()

	// Reopen with WRONG key
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: "wrong-key-32-characters-long-!",
	})
	require.NoError(t, err)
	defer k2.Close()

	pub2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "protected-sec-adv"})
	getCh := make(chan []byte, 1)
	unsub, _ := k2.SubscribeRaw(ctx, pub2.ReplyTo, func(m messages.Message) { getCh <- m.Payload })
	defer unsub()

	select {
	case p := <-getCh:
		var resp struct {
			Value string `json:"value"`
			Error string `json:"error"`
		}
		json.Unmarshal(p, &resp)
		// Should either return empty/error or garbage (wrong key = bad decrypt)
		assert.NotEqual(t, "sensitive", resp.Value, "wrong key should not decrypt correctly")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// testMatrixAuditEvents — secrets operations emit audit events.
func testMatrixAuditEvents(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	storedCh := make(chan messages.SecretsStoredEvent, 1)
	cancelStored, _ := sdk.SubscribeTo[messages.SecretsStoredEvent](env.Kit, ctx, "secrets.stored", func(evt messages.SecretsStoredEvent, _ messages.Message) { storedCh <- evt })
	defer cancelStored()

	accessedCh := make(chan messages.SecretsAccessedEvent, 1)
	cancelAccessed, _ := sdk.SubscribeTo[messages.SecretsAccessedEvent](env.Kit, ctx, "secrets.accessed", func(evt messages.SecretsAccessedEvent, _ messages.Message) { accessedCh <- evt })
	defer cancelAccessed()

	deletedCh := make(chan messages.SecretsDeletedEvent, 1)
	cancelDeleted, _ := sdk.SubscribeTo[messages.SecretsDeletedEvent](env.Kit, ctx, "secrets.deleted", func(evt messages.SecretsDeletedEvent, _ messages.Message) { deletedCh <- evt })
	defer cancelDeleted()

	// Set — triggers secrets.stored
	pub1, _ := sdk.Publish(env.Kit, ctx, messages.SecretsSetMsg{Name: "audit-key-sec-adv", Value: "v"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kit, ctx, pub1.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	<-setCh
	cancelSet()

	// Get — triggers secrets.accessed
	pub2, _ := sdk.Publish(env.Kit, ctx, messages.SecretsGetMsg{Name: "audit-key-sec-adv"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancelGet, _ := sdk.SubscribeTo[messages.SecretsGetResp](env.Kit, ctx, pub2.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) { getCh <- resp })
	<-getCh
	cancelGet()

	// Delete — triggers secrets.deleted
	pub3, _ := sdk.Publish(env.Kit, ctx, messages.SecretsDeleteMsg{Name: "audit-key-sec-adv"})
	delCh := make(chan messages.SecretsDeleteResp, 1)
	cancelDel, _ := sdk.SubscribeTo[messages.SecretsDeleteResp](env.Kit, ctx, pub3.ReplyTo, func(resp messages.SecretsDeleteResp, _ messages.Message) { delCh <- resp })
	<-delCh
	cancelDel()

	time.Sleep(300 * time.Millisecond)

	select {
	case evt := <-storedCh:
		assert.Equal(t, "audit-key-sec-adv", evt.Name)
	case <-time.After(2 * time.Second):
		t.Error("did not receive secrets.stored event")
	}

	select {
	case <-accessedCh:
		// received
	case <-time.After(2 * time.Second):
		t.Error("did not receive secrets.accessed event")
	}

	select {
	case <-deletedCh:
		// received
	case <-time.After(2 * time.Second):
		t.Error("did not receive secrets.deleted event")
	}
}

// testMatrixFromTS — secrets accessible from .ts surface.
func testMatrixFromTS(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx := context.Background()

	// Set via bus first
	pub, _ := sdk.Publish(env.Kit, ctx, messages.SecretsSetMsg{Name: "ts-secret-sec-adv", Value: "ts-value"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kit, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	<-setCh
	cancelSet()

	// Read from .ts
	result := testutil.EvalTS(t, env.Kit, "__sec_adv_read.ts", `
		var val = secrets.get("ts-secret-sec-adv");
		return val;
	`)
	assert.Equal(t, "ts-value", result)
}
