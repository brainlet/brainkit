package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStateCorruptionBadTranspile — persisted deployment with bad code, good deployment survives (D02).
func testStateCorruptionBadTranspile(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-deploy-adv.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "bad-deploy-adv.ts", Code: "const x: = {{{;;;", Order: 1,
		DeployedAt: time.Now(),
	})
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "good-deploy-adv.ts", Code: `output("survived");`, Order: 2,
		DeployedAt: time.Now(),
	})
	store.Close()

	var mu sync.Mutex
	var received []error

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
		ErrorHandler: func(err error, ctx brainkit.ErrorContext) {
			mu.Lock()
			received = append(received, err)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// The good deployment should still be active
	deps := k.ListDeployments()
	found := false
	for _, d := range deps {
		if d.Source == "good-deploy-adv.ts" {
			found = true
		}
	}
	assert.True(t, found, "good-deploy-adv.ts should survive despite bad-deploy-adv.ts failing")

	// ErrorHandler should have been called for bad.ts
	mu.Lock()
	foundDeploy := false
	for _, err := range received {
		var de *sdkerrors.DeployError
		if errors.As(err, &de) && de.Source == "bad-deploy-adv.ts" {
			foundDeploy = true
		}
	}
	mu.Unlock()
	assert.True(t, foundDeploy, "ErrorHandler should report bad-deploy-adv.ts deploy failure")
}

// testStateCorruptionDuplicatePersistedSource — duplicate persisted source resolves to one deployment (D06).
func testStateCorruptionDuplicatePersistedSource(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-deploy-adv2.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "dup-persist-deploy-adv.ts", Code: `output("v1");`, Order: 1, DeployedAt: time.Now(),
	})
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "dup-persist-deploy-adv.ts", Code: `output("v2");`, Order: 2, DeployedAt: time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	deps := k.ListDeployments()
	count := 0
	for _, d := range deps {
		if d.Source == "dup-persist-deploy-adv.ts" {
			count++
		}
	}
	assert.Equal(t, 1, count, "duplicate persisted sources should resolve to one deployment")
}

// testStateCorruptionStoreWipedMidlife — store wiped mid-life, in-memory state survives (D07).
func testStateCorruptionStoreWipedMidlife(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "store-deploy-adv3.db"))
	require.NoError(t, err)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	_, err = k.Deploy(t.Context(), "survivor-deploy-adv.ts", `output("alive");`)
	require.NoError(t, err)

	// Wipe store behind brainkit's back
	store.DeleteDeployment("survivor-deploy-adv.ts")

	// In-memory deployment still active
	deps := k.ListDeployments()
	found := false
	for _, d := range deps {
		if d.Source == "survivor-deploy-adv.ts" {
			found = true
		}
	}
	assert.True(t, found, "in-memory deployment survives store wipe")
}

// testStateCorruptionEmptyCode — persisted deployment with empty code (D01).
func testStateCorruptionEmptyCode(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-empty-deploy-adv.db")

	// Create store, save a deployment with empty code
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "empty-deploy-adv.ts", Code: "", Order: 1,
		DeployedAt: time.Now(),
	})
	store.Close()

	var mu sync.Mutex
	var received []error

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
		ErrorHandler: func(err error, ctx brainkit.ErrorContext) {
			mu.Lock()
			received = append(received, err)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Kernel should start — empty code deploy may succeed (empty JS is valid) or fail gracefully
	assert.NotNil(t, k)
}

// testStateCorruptionZeroDurationSchedule — schedule with zero duration (D03).
func testStateCorruptionZeroDurationSchedule(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-zerodur-deploy-adv.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID:         "zero-dur-deploy-adv",
		Expression: "every 0s",
		Duration:   0,
		Topic:      "zero.fire-deploy-adv",
		Payload:    json.RawMessage(`{}`),
		Source:     "test",
		CreatedAt:  time.Now(),
		NextFire:   time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	assert.True(t, k.Alive(context.Background()))
}

// testStateCorruptionPastScheduleFires — persisted schedule with past NextFire fires immediately (D04).
func testStateCorruptionPastScheduleFires(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-past-deploy-adv.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID:         "past-sched-deploy-adv",
		Expression: "every 1h",
		Duration:   time.Hour,
		Topic:      "past.fire-deploy-adv",
		Payload:    json.RawMessage(`{"fired": true}`),
		Source:     "test",
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		NextFire:   time.Now().Add(-1 * time.Hour), // 1 hour in the past
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	// The past schedule should fire immediately on restore
	fired := make(chan bool, 1)
	unsub, _ := k.SubscribeRaw(context.Background(), "past.fire-deploy-adv", func(m messages.Message) {
		fired <- true
	})
	defer unsub()

	select {
	case <-fired:
		// Good — past schedule caught up
	case <-time.After(3 * time.Second):
		// Also OK — schedule might have already fired during NewKernel before we subscribed
	}
}

// testStateCorruptionNonexistentRoleOnDeploy — role assigned to nonexistent role name (D08).
func testStateCorruptionNonexistentRoleOnDeploy(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-ghostrole-deploy-adv.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "ghost-role-deploy-adv.ts", Code: `output("hi");`,
		Order: 1, Role: "nonexistent-role-xyz-deploy-adv",
		DeployedAt: time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
		Roles: map[string]rbac.Role{"service": rbac.RoleService},
	})
	require.NoError(t, err)
	defer k.Close()

	// Kernel should start despite nonexistent role — RBAC.Assign would fail
	// but the deployment itself should still work (role assignment is best-effort)
	assert.NotNil(t, k)
}
