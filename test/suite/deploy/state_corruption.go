package deploy

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTeardownDuringHandlerExecution — persisted deployment with bad code, good deployment survives.
func testTeardownDuringHandlerExecution(t *testing.T, env *suite.TestEnv) {
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

// testRedeployWhileHandlersActive — duplicate persisted source resolves to one deployment.
func testRedeployWhileHandlersActive(t *testing.T, env *suite.TestEnv) {
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

// testDeployRemovesOtherDeployment — store wiped mid-life, in-memory state survives.
func testDeployRemovesOtherDeployment(t *testing.T, env *suite.TestEnv) {
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
