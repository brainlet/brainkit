package security

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStateNonexistentRoleOnDeploy — role assigned to nonexistent role name.
func testStateNonexistentRoleOnDeploy(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-sec.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	store.SaveDeployment(types.PersistedDeployment{
		Source: "ghost-role-sec.ts", Code: `output("hi");`,
		Order: 1, Role: "nonexistent-role-xyz",
		DeployedAt: time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
		Roles: map[string]rbac.Role{"service": rbac.RoleService},
	})
	require.NoError(t, err)
	defer k.Close()

	assert.NotNil(t, k)
}

// testStateStoreWipedMidlife — store emptied behind brainkit's back — in-memory state survives.
func testStateStoreWipedMidlife(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "store-sec.db"))
	require.NoError(t, err)

	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	secDeploy(t, k, "survivor-sec.ts", `output("alive");`)

	store.DeleteDeployment("survivor-sec.ts")

	deps := secListDeployments(t, k)
	found := false
	for _, d := range deps {
		if d.Source == "survivor-sec.ts" {
			found = true
		}
	}
	assert.True(t, found, "in-memory deployment survives store wipe")
}
