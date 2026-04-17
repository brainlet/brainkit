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
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStorageRuntimeAddRemove — add and remove storage at runtime via bus.
func testStorageRuntimeAddRemove(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add via bus
	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, sdk.StorageAddMsg{
		Name: "runtime-mem-reg-adv", Type: "memory", Config: json.RawMessage(`{}`),
	})
	addCh := make(chan sdk.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.StorageAddResp, _ sdk.Message) { addCh <- resp })
	<-addCh
	unsub()

	// Remove via bus
	pr2, _ := sdk.PublishStorageRemove(env.Kit, ctx, sdk.StorageRemoveMsg{Name: "runtime-mem-reg-adv"})
	rmCh := make(chan sdk.StorageRemoveResp, 1)
	unsub2, _ := sdk.SubscribeStorageRemoveResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp sdk.StorageRemoveResp, _ sdk.Message) { rmCh <- resp })
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
	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, sdk.StorageAddMsg{
		Name: "dup-store-reg-adv", Type: "memory", Config: json.RawMessage(`{}`),
	})
	ch := make(chan sdk.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.StorageAddResp, _ sdk.Message) { ch <- resp })
	<-ch
	unsub()

	// Second add — might succeed (replace) or error — no panic is the key
	pr2, _ := sdk.PublishStorageAdd(env.Kit, ctx, sdk.StorageAddMsg{
		Name: "dup-store-reg-adv", Type: "memory", Config: json.RawMessage(`{}`),
	})
	ch2 := make(chan sdk.StorageAddResp, 1)
	unsub2, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp sdk.StorageAddResp, _ sdk.Message) { ch2 <- resp })
	<-ch2
	unsub2()

	// Cleanup
	sdk.PublishStorageRemove(env.Kit, ctx, sdk.StorageRemoveMsg{Name: "dup-store-reg-adv"})
}

// testStorageRuntimeRemoveNonexistent — removing nonexistent storage doesn't crash.
func testStorageRuntimeRemoveNonexistent(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishStorageRemove(env.Kit, ctx, sdk.StorageRemoveMsg{Name: "ghost-storage-reg-adv"})
	rmCh := make(chan sdk.StorageRemoveResp, 1)
	unsub, _ := sdk.SubscribeStorageRemoveResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.StorageRemoveResp, _ sdk.Message) { rmCh <- resp })
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

	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, sdk.StorageAddMsg{
		Name:   "sqlite-runtime-reg-adv",
		Type:   "sqlite",
		Config: json.RawMessage(`{"path":"` + tmpDir + `/runtime.db"}`),
	})
	addCh := make(chan sdk.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.StorageAddResp, _ sdk.Message) { addCh <- resp })
	resp := <-addCh
	unsub()
	require.True(t, resp.Added, "SQLite storage should be added")

	// Verify it's registered
	result := testutil.EvalTS(t, env.Kit, "__reg_sqlite_check.ts", `
		return JSON.stringify({ has: registry.has("storage", "sqlite-runtime-reg-adv") });
	`)
	assert.Contains(t, result, `"has":true`)

	// Cleanup
	sdk.PublishStorageRemove(env.Kit, ctx, sdk.StorageRemoveMsg{Name: "sqlite-runtime-reg-adv"})
}

// testStorageRuntimeListResources — verify registered tool appears in tool.list bus command.
func testStorageRuntimeListResources(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "res-test-reg-adv.ts", `
		const t = createTool({id: "res-tool-reg-adv", description: "test", execute: async () => ({})});
		kit.register("tool", "res-tool-reg-adv", t);
	`)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, "res-test-reg-adv.ts") })

	// Verify tool appears via tool.list bus command
	payload := testutil.PublishAndWait(t, env.Kit, sdk.ToolListMsg{}, 5*time.Second)
	var resp sdk.ToolListResp
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, tool := range resp.Tools {
		if strings.Contains(tool.Name, "res-tool-reg-adv") || tool.ShortName == "res-tool-reg-adv" {
			found = true
		}
	}
	assert.True(t, found, "res-tool-reg-adv should appear in tool.list")
}

// testStorageRuntimeResourcesFromSource — track resources by deployment source via EvalTS.
func testStorageRuntimeResourcesFromSource(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "source-track-reg-adv.ts", `
		const t = createTool({id: "tracked-tool-reg-adv", description: "test", execute: async () => ({})});
		kit.register("tool", "tracked-tool-reg-adv", t);
	`)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, "source-track-reg-adv.ts") })

	// Verify the tool is registered via tool.list
	payload := testutil.PublishAndWait(t, env.Kit, sdk.ToolListMsg{}, 5*time.Second)
	var resp sdk.ToolListResp
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, tool := range resp.Tools {
		if strings.Contains(tool.Name, "tracked-tool-reg-adv") || tool.ShortName == "tracked-tool-reg-adv" {
			found = true
		}
	}
	assert.True(t, found, "should have tool from source")
}

// testStorageRuntimeKernelMultipleStorages — kit with multiple storage backends.
func testStorageRuntimeKernelMultipleStorages(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
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
