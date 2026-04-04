package registry

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStorageRuntimeAddRemove — add and remove storage at runtime.
func testStorageRuntimeAddRemove(t *testing.T, env *suite.TestEnv) {
	err := env.Kernel.AddStorage("runtime-mem-reg-adv", brainkit.InMemoryStorage())
	require.NoError(t, err)

	url := env.Kernel.StorageURL("runtime-mem-reg-adv")
	assert.Empty(t, url) // in-memory has no URL

	err = env.Kernel.RemoveStorage("runtime-mem-reg-adv")
	require.NoError(t, err)
}

// testStorageRuntimeAddDuplicate — adding same name twice returns error or replaces.
func testStorageRuntimeAddDuplicate(t *testing.T, env *suite.TestEnv) {
	err := env.Kernel.AddStorage("dup-store-reg-adv", brainkit.InMemoryStorage())
	require.NoError(t, err)

	// Second add — might succeed (InMemory replaces) or error
	err2 := env.Kernel.AddStorage("dup-store-reg-adv", brainkit.InMemoryStorage())
	_ = err2 // behavior depends on implementation — no panic is the key

	env.Kernel.RemoveStorage("dup-store-reg-adv")
}

// testStorageRuntimeRemoveNonexistent — removing nonexistent storage doesn't crash.
func testStorageRuntimeRemoveNonexistent(t *testing.T, env *suite.TestEnv) {
	err := env.Kernel.RemoveStorage("ghost-storage-reg-adv")
	assert.NoError(t, err) // should be idempotent
}

// testStorageRuntimeURLForNonexistent — URL for nonexistent returns empty.
func testStorageRuntimeURLForNonexistent(t *testing.T, env *suite.TestEnv) {
	url := env.Kernel.StorageURL("ghost-storage-reg-adv")
	assert.Empty(t, url)
}

// testStorageRuntimeSQLiteAdd — adding SQLite storage starts bridge.
func testStorageRuntimeSQLiteAdd(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()

	err := env.Kernel.AddStorage("sqlite-runtime-reg-adv", brainkit.SQLiteStorage(tmpDir+"/runtime.db"))
	require.NoError(t, err)

	url := env.Kernel.StorageURL("sqlite-runtime-reg-adv")
	assert.NotEmpty(t, url, "SQLite storage should have a bridge URL")

	err = env.Kernel.RemoveStorage("sqlite-runtime-reg-adv")
	require.NoError(t, err)
}

// testStorageRuntimeListResources — ListResources doesn't crash with various states.
func testStorageRuntimeListResources(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// Before any deployment
	resources, err := env.Kernel.ListResources()
	require.NoError(t, err)
	assert.NotNil(t, resources) // may be empty, but not nil

	// After deployment
	env.Kernel.Deploy(ctx, "res-test-reg-adv.ts", `
		const t = createTool({id: "res-tool-reg-adv", description: "test", execute: async () => ({})});
		kit.register("tool", "res-tool-reg-adv", t);
	`)
	defer env.Kernel.Teardown(ctx, "res-test-reg-adv.ts")

	resources2, _ := env.Kernel.ListResources()
	assert.Greater(t, len(resources2), 0)

	// Filter by type
	tools, _ := env.Kernel.ListResources("tool")
	found := false
	for _, r := range tools {
		if r.Name == "res-tool-reg-adv" {
			found = true
		}
	}
	assert.True(t, found)
}

// testStorageRuntimeResourcesFromSource — track resources by deployment source.
func testStorageRuntimeResourcesFromSource(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	env.Kernel.Deploy(ctx, "source-track-reg-adv.ts", `
		const t = createTool({id: "tracked-tool-reg-adv", description: "test", execute: async () => ({})});
		kit.register("tool", "tracked-tool-reg-adv", t);
		kit.register("agent", "tracked-agent-reg-adv", {});
	`)
	defer env.Kernel.Teardown(ctx, "source-track-reg-adv.ts")

	resources, err := env.Kernel.ResourcesFrom("source-track-reg-adv.ts")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resources), 2, "should have tool + agent from source")
}

// testStorageRuntimeScalingPool — basic pool lifecycle.
func testStorageRuntimeScalingPool(t *testing.T, _ *suite.TestEnv) {
	im := brainkit.NewInstanceManager()

	// Pool operations should not panic even with no real instances
	pools := im.Pools()
	assert.Empty(t, pools)

	_, err := im.PoolInfo("ghost-reg-adv")
	assert.Error(t, err) // not found

	err = im.Scale("ghost-reg-adv", 1)
	assert.Error(t, err) // not found

	err = im.KillPool("ghost-reg-adv")
	assert.Error(t, err) // not found
}

// testStorageRuntimeKernelMultipleStorages — kernel with multiple storage backends.
func testStorageRuntimeKernelMultipleStorages(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"mem":    brainkit.InMemoryStorage(),
			"sqlite": brainkit.SQLiteStorage(tmpDir + "/multi.db"),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	assert.Empty(t, k.StorageURL("mem"))      // in-memory has no URL
	assert.NotEmpty(t, k.StorageURL("sqlite")) // sqlite has bridge URL

	// Both registered in provider registry
	ctx := context.Background()
	result, err := k.EvalTS(ctx, "__check_storages_reg_adv.ts", `
		var hasMem = registry.has("storage", "mem");
		var hasSqlite = registry.has("storage", "sqlite");
		return JSON.stringify({mem: hasMem, sqlite: hasSqlite});
	`)
	require.NoError(t, err)
	assert.Contains(t, result, `"mem":true`)
	assert.Contains(t, result, `"sqlite":true`)
}
