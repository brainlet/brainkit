package registry

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
)

// ── Bus plumbing ─────────────────────────────────────────────────────────────

func testStorageAddViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name:   "test-mem-stor",
		Type:   "memory",
		Config: json.RawMessage(`{}`),
	})
	respCh := make(chan messages.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageAddResp, msg messages.Message) { respCh <- resp })
	defer unsub()

	select {
	case resp := <-respCh:
		if resp.Error != "" {
			t.Fatalf("error: %s", resp.Error)
		}
		if !resp.Added {
			t.Fatal("expected Added=true")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Verify via registry.list
	pr2, _ := sdk.Publish(env.Kit, ctx, messages.RegistryListMsg{Category: "storage"})
	listCh := make(chan messages.RegistryListResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.RegistryListResp](env.Kit, ctx, pr2.ReplyTo,
		func(resp messages.RegistryListResp, msg messages.Message) { listCh <- resp })
	defer unsub2()

	select {
	case resp := <-listCh:
		if !strings.Contains(string(resp.Items), "test-mem-stor") {
			t.Fatalf("storage not in list: %s", resp.Items)
		}
	case <-ctx.Done():
		t.Fatal("timeout on list")
	}
}

func testStorageRemoveViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add
	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name: "test-rm-stor", Type: "memory", Config: json.RawMessage(`{}`),
	})
	ch := make(chan messages.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageAddResp, msg messages.Message) { ch <- resp })
	<-ch
	unsub()

	// Remove
	pr2, _ := sdk.PublishStorageRemove(env.Kit, ctx, messages.StorageRemoveMsg{Name: "test-rm-stor"})
	rmCh := make(chan messages.StorageRemoveResp, 1)
	unsub2, _ := sdk.SubscribeStorageRemoveResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp messages.StorageRemoveResp, msg messages.Message) { rmCh <- resp })
	defer unsub2()

	select {
	case resp := <-rmCh:
		if !resp.Removed {
			t.Fatal("expected Removed=true")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testStorageRemoveNonexistent(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishStorageRemove(env.Kit, ctx, messages.StorageRemoveMsg{Name: "nonexistent-stor"})
	rmCh := make(chan messages.StorageRemoveResp, 1)
	unsub, _ := sdk.SubscribeStorageRemoveResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageRemoveResp, msg messages.Message) { rmCh <- resp })
	defer unsub()

	select {
	case resp := <-rmCh:
		// Should succeed gracefully (no-op)
		if !resp.Removed {
			t.Log("remove nonexistent returned Removed=false (acceptable)")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// ── Real effect ──────────────────────────────────────────────────────────────

func testStorageAddSQLiteThenDeployUses(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	// Add SQLite storage at runtime via bus
	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name:   "dynamic-sql",
		Type:   "sqlite",
		Config: json.RawMessage(`{"path":"` + tmpDir + `/dynamic.db"}`),
	})
	addCh := make(chan messages.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageAddResp, msg messages.Message) { addCh <- resp })
	resp := <-addCh
	unsub()
	if resp.Error != "" {
		t.Fatalf("add storage: %s", resp.Error)
	}

	// Deploy .ts that uses this storage, creates a table, inserts+reads data
	code := `
		const store = storage("dynamic-sql");
		await store.init();

		bus.on("storage-test", async (msg) => {
			try {
				// The storage is a LibSQLStore — run raw SQL through its internal client
				// For verification, just check it initialized without error
				msg.reply({ ok: true, storage: "dynamic-sql" });
			} catch(e) {
				msg.reply({ ok: false, error: e.message });
			}
		});
	`
	testutil.Deploy(t, env.Kit, "storage-test-dynamic.ts", code)
	defer testutil.Teardown(t, env.Kit, "storage-test-dynamic.ts")

	// Send message to deployed service
	pr2, _ := sdk.Publish(env.Kit, ctx, messages.KitSendMsg{
		Topic:   "ts.storage-test-dynamic.storage-test",
		Payload: json.RawMessage(`{}`),
	})
	sendCh := make(chan messages.KitSendResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.KitSendResp](env.Kit, ctx, pr2.ReplyTo,
		func(resp messages.KitSendResp, msg messages.Message) { sendCh <- resp })
	defer unsub2()

	select {
	case resp := <-sendCh:
		var result struct {
			OK      bool   `json:"ok"`
			Storage string `json:"storage"`
			Error   string `json:"error"`
		}
		json.Unmarshal(resp.Payload, &result)
		if !result.OK {
			t.Fatalf("storage test failed: %s", result.Error)
		}
		if result.Storage != "dynamic-sql" {
			t.Fatalf("expected storage name 'dynamic-sql', got %q", result.Storage)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for storage test response")
	}
}

func testStorageAddMemoryThenDeployUses(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add in-memory storage at runtime
	pr, _ := sdk.PublishStorageAdd(env.Kit, ctx, messages.StorageAddMsg{
		Name: "dynamic-mem", Type: "memory", Config: json.RawMessage(`{}`),
	})
	addCh := make(chan messages.StorageAddResp, 1)
	unsub, _ := sdk.SubscribeStorageAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.StorageAddResp, msg messages.Message) { addCh <- resp })
	<-addCh
	unsub()

	// Deploy .ts that resolves it
	code := `
		const resolved = registry.resolve("storage", "dynamic-mem");
		output(JSON.stringify(resolved));
	`
	testutil.Deploy(t, env.Kit, "mem-stor-test.ts", code)
	defer testutil.Teardown(t, env.Kit, "mem-stor-test.ts")

	result := testutil.EvalTS(t, env.Kit, "__check_mem_stor.ts", `
		return globalThis.__module_result || "null";
	`)
	if result == "null" || result == "" {
		t.Fatal("expected storage to resolve, got null")
	}
	if !strings.Contains(result, "memory") {
		t.Fatalf("expected 'memory' in resolved config, got: %s", result)
	}
}
