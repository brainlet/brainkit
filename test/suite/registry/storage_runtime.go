package registry

import (
	"context"
	"encoding/json"
	"strings"
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

// testStorageRuntimeAddRemove — add and remove storage at runtime via bus.
func testStorageRuntimeAddRemove(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add via bus
	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name: "runtime-mem-reg-adv", Type: "memory", Config: json.RawMessage(`{}`),
	})
	addCh := make(chan messages.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageAddResp, _ messages.Message) { addCh <- resp })
	<-addCh
	unsub()

	// Remove via bus
	pr2, _ := sdk.PublishStorageRemove(env.Kit, ctx, messages.StorageRemoveMsg{Name: "runtime-mem-reg-adv"})
	rmCh := make(chan messages.StorageRemoveResp, 1)
	unsub2, _ := sdk.SubscribeStorageRemoveResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp messages.StorageRemoveResp, _ messages.Message) { rmCh <- resp })
	defer unsub2()

	select {
	case resp := <-rmCh:
		assert.True(t, resp.Removed)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testStorageRuntimeAddDuplicate — adding same name twice returns error or replaces.
func testStorageRuntimeAddDuplicate(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First add
	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name: "dup-store-reg-adv", Type: "memory", Config: json.RawMessage(`{}`),
	})
	ch := make(chan messages.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageAddResp, _ messages.Message) { ch <- resp })
	<-ch
	unsub()

	// Second add — might succeed (replace) or error — no panic is the key
	pr2, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name: "dup-store-reg-adv", Type: "memory", Config: json.RawMessage(`{}`),
	})
	ch2 := make(chan messages.StorageAddResp, 1)
	unsub2, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp messages.StorageAddResp, _ messages.Message) { ch2 <- resp })
	<-ch2
	unsub2()

	// Cleanup
	sdk.PublishStorageRemove(env.Kit, ctx, messages.StorageRemoveMsg{Name: "dup-store-reg-adv"})
}

// testStorageRuntimeRemoveNonexistent — removing nonexistent storage doesn't crash.
func testStorageRuntimeRemoveNonexistent(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishStorageRemove(env.Kit, ctx, messages.StorageRemoveMsg{Name: "ghost-storage-reg-adv"})
	rmCh := make(chan messages.StorageRemoveResp, 1)
	unsub, _ := sdk.SubscribeStorageRemoveResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageRemoveResp, _ messages.Message) { rmCh <- resp })
	defer unsub()

	select {
	case <-rmCh:
		// Should succeed gracefully (no-op or removed=false)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testStorageRuntimeURLForNonexistent — resolving nonexistent storage returns empty via JS bridge.
func testStorageRuntimeURLForNonexistent(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__reg_url_nonexist.ts", `
		var resolved = registry.has("storage", "ghost-storage-reg-adv");
		return JSON.stringify({ exists: resolved });
	`)
	assert.Contains(t, result, `"exists":false`)
}

// testStorageRuntimeSQLiteAdd — adding SQLite storage via bus.
func testStorageRuntimeSQLiteAdd(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name:   "sqlite-runtime-reg-adv",
		Type:   "sqlite",
		Config: json.RawMessage(`{"path":"` + tmpDir + `/runtime.db"}`),
	})
	addCh := make(chan messages.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageAddResp, _ messages.Message) { addCh <- resp })
	resp := <-addCh
	unsub()
	require.True(t, resp.Added, "SQLite storage should be added")

	// Verify it's registered
	result := testutil.EvalTS(t, env.Kit, "__reg_sqlite_check.ts", `
		return JSON.stringify({ has: registry.has("storage", "sqlite-runtime-reg-adv") });
	`)
	assert.Contains(t, result, `"has":true`)

	// Cleanup
	sdk.PublishStorageRemove(env.Kit, ctx, messages.StorageRemoveMsg{Name: "sqlite-runtime-reg-adv"})
}

// testStorageRuntimeListResources — ListResources equivalent via registry.list bus command.
func testStorageRuntimeListResources(t *testing.T, env *suite.TestEnv) {
	// After deployment
	testutil.Deploy(t, env.Kit, "res-test-reg-adv.ts", `
		const t = createTool({id: "res-tool-reg-adv", description: "test", execute: async () => ({})});
		kit.register("tool", "res-tool-reg-adv", t);
	`)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, "res-test-reg-adv.ts") })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify tool appears in registry list
	pr, _ := sdk.Publish(env.Kit, ctx, messages.RegistryListMsg{Category: "tool"})
	listCh := make(chan messages.RegistryListResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.RegistryListResp](env.Kit, ctx, pr.ReplyTo,
		func(resp messages.RegistryListResp, _ messages.Message) { listCh <- resp })
	defer unsub()

	select {
	case resp := <-listCh:
		assert.Contains(t, string(resp.Items), "res-tool-reg-adv")
	case <-ctx.Done():
		t.Fatal("timeout listing tools")
	}
}

// testStorageRuntimeResourcesFromSource — track resources by deployment source via EvalTS.
func testStorageRuntimeResourcesFromSource(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "source-track-reg-adv.ts", `
		const t = createTool({id: "tracked-tool-reg-adv", description: "test", execute: async () => ({})});
		kit.register("tool", "tracked-tool-reg-adv", t);
	`)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, "source-track-reg-adv.ts") })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify the tool is registered
	pr, _ := sdk.Publish(env.Kit, ctx, messages.RegistryListMsg{Category: "tool"})
	listCh := make(chan messages.RegistryListResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.RegistryListResp](env.Kit, ctx, pr.ReplyTo,
		func(resp messages.RegistryListResp, _ messages.Message) { listCh <- resp })
	defer unsub()

	select {
	case resp := <-listCh:
		assert.True(t, strings.Contains(string(resp.Items), "tracked-tool-reg-adv"),
			"should have tool from source")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
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

// testStorageRuntimeKernelMultipleStorages — kit with multiple storage backends.
func testStorageRuntimeKernelMultipleStorages(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"mem":    brainkit.InMemoryStorage(),
			"sqlite": brainkit.SQLiteStorage(tmpDir + "/multi.db"),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Both registered in provider registry — verify via JS bridge
	result := testutil.EvalTS(t, k, "__check_storages_reg_adv.ts", `
		var hasMem = registry.has("storage", "mem");
		var hasSqlite = registry.has("storage", "sqlite");
		return JSON.stringify({mem: hasMem, sqlite: hasSqlite});
	`)
	assert.Contains(t, result, `"mem":true`)
	assert.Contains(t, result, `"sqlite":true`)
}
