package secrets

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/stores"
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
	setResp := callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "lifecycle-key-sec-adv", Value: "lifecycle-val"})
	assert.True(t, setResp.Stored)

	// Get
	getResp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "lifecycle-key-sec-adv"})
	assert.Equal(t, "lifecycle-val", getResp.Value)

	// List
	listResp := callSecretList(t, env.Kit, ctx, secretmsg.SecretsListMsg{})
	found := false
	for _, s := range listResp.Secrets {
		if s.Name == "lifecycle-key-sec-adv" {
			found = true
		}
	}
	assert.True(t, found, "lifecycle key should appear in list")

	// Delete
	delResp := callSecretDelete(t, env.Kit, ctx, secretmsg.SecretsDeleteMsg{Name: "lifecycle-key-sec-adv"})
	assert.True(t, delResp.Deleted)

	// Get after delete — should be empty
	getAfterDeleteResp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "lifecycle-key-sec-adv"})
	assert.Empty(t, getAfterDeleteResp.Value)
}

// testMatrixRotate — set then rotate, verify version increments.
func testMatrixRotate(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set v1
	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "rot-key-sec-adv", Value: "v1"})

	// Rotate to v2
	rotateResp := callSecretRotate(t, env.Kit, ctx, secretmsg.SecretsRotateMsg{Name: "rot-key-sec-adv", NewValue: "v2"})
	assert.True(t, rotateResp.Rotated)

	// Get — should be v2
	resp := callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "rot-key-sec-adv"})
	assert.Equal(t, "v2", resp.Value)
}

// testMatrixManySecrets — set 20 secrets, list them all.
func testMatrixManySecrets(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, _ := stores.NewSQLite(filepath.Join(tmpDir, "bulk.db"))
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "bulk-test-key-32-characters!!",
		Modules: []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)
	defer k.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for i := 0; i < 20; i++ {
		resp := callSecretSet(t, k, ctx, secretmsg.SecretsSetMsg{
			Name: fmt.Sprintf("bulk-key-sec-adv-%d", i), Value: fmt.Sprintf("val-%d", i),
		})
		assert.True(t, resp.Stored)
	}

	listResp := callSecretList(t, k, ctx, secretmsg.SecretsListMsg{})
	names := make(map[string]bool)
	for _, s := range listResp.Secrets {
		names[s.Name] = true
	}
	for i := 0; i < 20; i++ {
		assert.True(t, names[fmt.Sprintf("bulk-key-sec-adv-%d", i)], "missing bulk-key-%d", i)
	}
}

// testMatrixEncryptedPersistence — secrets survive restart with encryption.
func testMatrixEncryptedPersistence(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.db")
	masterKey := "test-encryption-key-32chars!!"

	// Phase 1: Set encrypted secret
	store1, _ := stores.NewSQLite(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: masterKey,
		Modules: []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)

	ctx := context.Background()
	callSecretSet(t, k1, ctx, secretmsg.SecretsSetMsg{Name: "enc-key-sec-adv", Value: "enc-secret-val"})
	k1.Close()

	// Phase 2: Reopen with same key, retrieve
	store2, _ := stores.NewSQLite(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: masterKey,
		Modules: []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)
	defer k2.Close()

	resp := callSecretGet(t, k2, ctx, secretmsg.SecretsGetMsg{Name: "enc-key-sec-adv"})
	assert.Equal(t, "enc-secret-val", resp.Value)
}

// testMatrixWrongKeyCannotDecrypt — wrong master key fails to decrypt.
func testMatrixWrongKeyCannotDecrypt(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.db")

	// Set with key A
	store1, _ := stores.NewSQLite(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "correct-key-32-characters-long!",
		Modules: []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)

	ctx := context.Background()
	callSecretSet(t, k1, ctx, secretmsg.SecretsSetMsg{Name: "protected-sec-adv", Value: "sensitive"})
	k1.Close()

	// Reopen with WRONG key
	store2, _ := stores.NewSQLite(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: "wrong-key-32-characters-long-!",
		Modules: []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)
	defer k2.Close()

	resp, err := secretmsg.CallSecretsGet(k2, ctx, secretmsg.SecretsGetMsg{Name: "protected-sec-adv"}, sdk.WithCallTimeout(5*time.Second))
	if err == nil {
		assert.NotEqual(t, "sensitive", resp.Value, "wrong key should not decrypt correctly")
	}
}

// testMatrixAuditEvents — secrets operations emit audit events.
func testMatrixAuditEvents(t *testing.T, _ *suite.TestEnv) {
	env := secretsEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	storedCh := make(chan secretmsg.SecretsStoredEvent, 1)
	cancelStored, _ := sdk.SubscribeTo[secretmsg.SecretsStoredEvent](env.Kit, ctx, "secrets.stored", func(evt secretmsg.SecretsStoredEvent, _ sdk.Message) { storedCh <- evt })
	defer cancelStored()

	accessedCh := make(chan secretmsg.SecretsAccessedEvent, 1)
	cancelAccessed, _ := sdk.SubscribeTo[secretmsg.SecretsAccessedEvent](env.Kit, ctx, "secrets.accessed", func(evt secretmsg.SecretsAccessedEvent, _ sdk.Message) { accessedCh <- evt })
	defer cancelAccessed()

	deletedCh := make(chan secretmsg.SecretsDeletedEvent, 1)
	cancelDeleted, _ := sdk.SubscribeTo[secretmsg.SecretsDeletedEvent](env.Kit, ctx, "secrets.deleted", func(evt secretmsg.SecretsDeletedEvent, _ sdk.Message) { deletedCh <- evt })
	defer cancelDeleted()

	// Set — triggers secrets.stored
	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "audit-key-sec-adv", Value: "v"})

	// Get — triggers secrets.accessed
	callSecretGet(t, env.Kit, ctx, secretmsg.SecretsGetMsg{Name: "audit-key-sec-adv"})

	// Delete — triggers secrets.deleted
	callSecretDelete(t, env.Kit, ctx, secretmsg.SecretsDeleteMsg{Name: "audit-key-sec-adv"})

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
	callSecretSet(t, env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "ts-secret-sec-adv", Value: "ts-value"})

	// Read from .ts
	result := testutil.EvalTS(t, env.Kit, "__sec_adv_read.ts", `
		var val = secrets.get("ts-secret-sec-adv");
		return val;
	`)
	assert.Equal(t, "ts-value", result)
}
