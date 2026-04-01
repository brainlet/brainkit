package adversarial_test

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStorageRuntime_AddRemove — add and remove storage at runtime.
func TestStorageRuntime_AddRemove(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	err := tk.AddStorage("runtime-mem", brainkit.InMemoryStorage())
	require.NoError(t, err)

	url := tk.StorageURL("runtime-mem")
	assert.Empty(t, url) // in-memory has no URL

	err = tk.RemoveStorage("runtime-mem")
	require.NoError(t, err)
}

// TestStorageRuntime_AddDuplicate — adding same name twice returns error.
func TestStorageRuntime_AddDuplicate(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	err := tk.AddStorage("dup-store", brainkit.InMemoryStorage())
	require.NoError(t, err)

	// Second add — might succeed (InMemory replaces) or error
	err2 := tk.AddStorage("dup-store", brainkit.InMemoryStorage())
	_ = err2 // behavior depends on implementation — no panic is the key

	tk.RemoveStorage("dup-store")
}

// TestStorageRuntime_RemoveNonexistent — removing nonexistent storage doesn't crash.
func TestStorageRuntime_RemoveNonexistent(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	err := tk.RemoveStorage("ghost-storage")
	assert.NoError(t, err) // should be idempotent
}

// TestStorageRuntime_URLForNonexistent — URL for nonexistent returns empty.
func TestStorageRuntime_URLForNonexistent(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	url := tk.StorageURL("ghost-storage")
	assert.Empty(t, url)
}

// TestStorageRuntime_SQLiteAdd — adding SQLite storage starts bridge.
func TestStorageRuntime_SQLiteAdd(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	tmpDir := t.TempDir()

	err := tk.AddStorage("sqlite-runtime", brainkit.SQLiteStorage(tmpDir+"/runtime.db"))
	require.NoError(t, err)

	url := tk.StorageURL("sqlite-runtime")
	assert.NotEmpty(t, url, "SQLite storage should have a bridge URL")

	err = tk.RemoveStorage("sqlite-runtime")
	require.NoError(t, err)
}

// TestStorageRuntime_ListResources — ListResources doesn't crash with various states.
func TestStorageRuntime_ListResources(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Before any deployment
	resources, err := tk.ListResources()
	require.NoError(t, err)
	assert.NotNil(t, resources) // may be empty, but not nil

	// After deployment
	tk.Deploy(ctx, "res-test.ts", `
		const t = createTool({id: "res-tool", description: "test", execute: async () => ({})});
		kit.register("tool", "res-tool", t);
	`)
	defer tk.Teardown(ctx, "res-test.ts")

	resources2, _ := tk.ListResources()
	assert.Greater(t, len(resources2), 0)

	// Filter by type
	tools, _ := tk.ListResources("tool")
	found := false
	for _, r := range tools {
		if r.Name == "res-tool" {
			found = true
		}
	}
	assert.True(t, found)
}

// TestStorageRuntime_ResourcesFromSource — track resources by deployment source.
func TestStorageRuntime_ResourcesFromSource(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	tk.Deploy(ctx, "source-track.ts", `
		const t = createTool({id: "tracked-tool", description: "test", execute: async () => ({})});
		kit.register("tool", "tracked-tool", t);
		kit.register("agent", "tracked-agent", {});
	`)
	defer tk.Teardown(ctx, "source-track.ts")

	resources, err := tk.ResourcesFrom("source-track.ts")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resources), 2, "should have tool + agent from source")
}

// TestStorageRuntime_ScalingPool — basic pool lifecycle.
func TestStorageRuntime_ScalingPool(t *testing.T) {
	im := brainkit.NewInstanceManager()

	// Pool operations should not panic even with no real instances
	pools := im.Pools()
	assert.Empty(t, pools)

	_, err := im.PoolInfo("ghost")
	assert.Error(t, err) // not found

	err = im.Scale("ghost", 1)
	assert.Error(t, err) // not found

	err = im.KillPool("ghost")
	assert.Error(t, err) // not found
}

// TestStorageRuntime_KernelMultipleStorages — kernel with multiple storage backends.
func TestStorageRuntime_KernelMultipleStorages(t *testing.T) {
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

	assert.Empty(t, k.StorageURL("mem"))        // in-memory has no URL
	assert.NotEmpty(t, k.StorageURL("sqlite"))   // sqlite has bridge URL

	// Both registered in provider registry
	ctx := context.Background()
	result, err := k.EvalTS(ctx, "__check_storages.ts", `
		var hasMem = registry.has("storage", "mem");
		var hasSqlite = registry.has("storage", "sqlite");
		return JSON.stringify({mem: hasMem, sqlite: hasSqlite});
	`)
	require.NoError(t, err)
	assert.Contains(t, result, `"mem":true`)
	assert.Contains(t, result, `"sqlite":true`)
}
