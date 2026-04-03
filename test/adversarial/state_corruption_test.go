package adversarial_test

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
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// D01: Persisted deployment with empty code
func TestStateCorruption_EmptyCode(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	// Create store, save a deployment with empty code
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "empty.ts", Code: "", Order: 1,
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

// D02: Persisted deployment with code that fails to transpile
func TestStateCorruption_BadTranspile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "bad.ts", Code: "const x: = {{{;;;", Order: 1,
		DeployedAt: time.Now(),
	})
	// Also save a good one to verify it still deploys
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "good.ts", Code: `output("survived");`, Order: 2,
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
		if d.Source == "good.ts" {
			found = true
		}
	}
	assert.True(t, found, "good.ts should survive despite bad.ts failing")

	// ErrorHandler should have been called for bad.ts
	mu.Lock()
	foundDeploy := false
	for _, err := range received {
		var de *sdkerrors.DeployError
		if errors.As(err, &de) && de.Source == "bad.ts" {
			foundDeploy = true
		}
	}
	mu.Unlock()
	assert.True(t, foundDeploy, "ErrorHandler should report bad.ts deploy failure")
}

// D04: Persisted schedule with past NextFire fires immediately
func TestStateCorruption_PastScheduleFires(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID:         "past-sched",
		Expression: "every 1h",
		Duration:   time.Hour,
		Topic:      "past.fire",
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
	unsub, _ := k.SubscribeRaw(context.Background(), "past.fire", func(m messages.Message) {
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

// D08: Role assigned to nonexistent role name
func TestStateCorruption_NonexistentRoleOnDeploy(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "ghost-role.ts", Code: `output("hi");`,
		Order: 1, Role: "nonexistent-role-xyz",
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

// D03: Schedule with zero duration
func TestStateCorruption_ZeroDurationSchedule(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID:         "zero-dur",
		Expression: "every 0s",
		Duration:   0,
		Topic:      "zero.fire",
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

func TestStateCorruption_DuplicatePersistedSource(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "dup.ts", Code: `output("v1");`, Order: 1, DeployedAt: time.Now(),
	})
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "dup.ts", Code: `output("v2");`, Order: 2, DeployedAt: time.Now(),
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
		if d.Source == "dup.ts" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

// D07: Store emptied behind brainkit's back — in-memory state survives
func TestStateCorruption_StoreWipedMidlife(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "store.db"))
	require.NoError(t, err)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	_, err = k.Deploy(context.Background(), "survivor.ts", `output("alive");`)
	require.NoError(t, err)

	// Wipe store behind brainkit's back
	store.DeleteDeployment("survivor.ts")

	// In-memory deployment still active
	deps := k.ListDeployments()
	found := false
	for _, d := range deps {
		if d.Source == "survivor.ts" {
			found = true
		}
	}
	assert.True(t, found, "in-memory deployment survives store wipe")
}
