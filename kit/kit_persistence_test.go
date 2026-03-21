package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func newPersistentKit(t *testing.T, storePath string) *Kit {
	t.Helper()
	store, err := NewSQLiteStore(storePath)
	require.NoError(t, err)

	kit, err := New(Config{
		Providers: map[string]ProviderConfig{},
		Store:     store,
	})
	require.NoError(t, err)
	t.Cleanup(func() { kit.Close() })
	return kit
}

func TestPersistence_ModuleSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "test.db")
	ctx := context.Background()

	// Kit 1: compile a module
	kit1 := newPersistentKit(t, storePath)
	_, err := kit1.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+`export function run(): i32 { return 42; }`+"`"+`, { name: "survivor", runtime: "stub" });
	`)
	require.NoError(t, err)
	kit1.Close()

	// Kit 2: same store — module should be loaded
	store2, err := NewSQLiteStore(storePath)
	require.NoError(t, err)
	kit2, err := New(Config{Store: store2})
	require.NoError(t, err)
	defer kit2.Close()

	info, err := kit2.GetWASMModule("survivor")
	require.NoError(t, err)
	require.NotNil(t, info, "module should survive restart")
	require.Equal(t, "survivor", info.Name)

	// Run the restored module
	result, err := kit2.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("survivor");
		return JSON.stringify(r);
	`)
	require.NoError(t, err)
	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	require.Equal(t, 42, rr.ExitCode, "module should execute correctly after restart")
}

func TestPersistence_ShardAutoRedeploy(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "test.db")
	ctx := context.Background()

	source := `
import { setMode, on, log, reply } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.ping", "handlePing");
}

export function handlePing(topic: string, payload: string): void {
  log("ping received");
  reply('{"ok":true}');
}
`

	// Kit 1: compile + deploy shard
	kit1 := newPersistentKit(t, storePath)
	_, err := kit1.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"pinger\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)
	_, err = kit1.DeployWASM("pinger")
	require.NoError(t, err)
	kit1.Close()

	// Kit 2: shard should auto-redeploy
	store2, _ := NewSQLiteStore(storePath)
	kit2, err := New(Config{Store: store2})
	require.NoError(t, err)
	defer kit2.Close()

	deployed := kit2.ListDeployedWASM()
	require.Len(t, deployed, 1, "shard should auto-redeploy")
	require.Equal(t, "pinger", deployed[0].Module)

	// Inject event — should work
	result, err := kit2.InjectWASMEvent("pinger", "test.ping", json.RawMessage(`{"msg":"hello"}`))
	require.NoError(t, err)
	require.Empty(t, result.Error)
}

func TestPersistence_PersistentState(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "test.db")
	ctx := context.Background()

	source := `
import { setMode, on, setState, getState, reply } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("counter.inc", "handleInc");
  on("counter.get", "handleGet");
}

export function handleInc(topic: string, payload: string): void {
  var c: i32 = 0;
  var raw = getState("count");
  if (raw.length > 0) c = I32.parseInt(raw);
  c++;
  setState("count", c.toString());
  reply('{"ok":true}');
}

export function handleGet(topic: string, payload: string): void {
  var raw = getState("count");
  reply('{"count":' + (raw.length > 0 ? raw : '0') + '}');
}
`

	// Kit 1: compile + deploy + increment 3 times
	kit1 := newPersistentKit(t, storePath)
	_, err := kit1.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"counter\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)
	kit1.DeployWASM("counter")
	kit1.InjectWASMEvent("counter", "counter.inc", json.RawMessage(`{}`))
	kit1.InjectWASMEvent("counter", "counter.inc", json.RawMessage(`{}`))
	kit1.InjectWASMEvent("counter", "counter.inc", json.RawMessage(`{}`))
	kit1.Close()

	// Kit 2: state should be preserved (count=3)
	store2, _ := NewSQLiteStore(storePath)
	kit2, err := New(Config{Store: store2})
	require.NoError(t, err)
	defer kit2.Close()

	result, err := kit2.InjectWASMEvent("counter", "counter.get", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, result.ReplyPayload, `"count":3`, "persistent state should survive restart")
}

func TestPersistence_UndeployClears(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "test.db")
	ctx := context.Background()

	source := `
import { setMode, on, log } from "brainkit";
export function init(): void {
  setMode("stateless");
  on("test.x", "handle");
}
export function handle(topic: string, payload: string): void { log("handled"); }
`

	// Kit 1: compile + deploy + undeploy
	kit1 := newPersistentKit(t, storePath)
	_, err := kit1.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"ephemeral\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)
	kit1.DeployWASM("ephemeral")
	kit1.UndeployWASM("ephemeral")
	kit1.Close()

	// Kit 2: shard should NOT be redeployed
	store2, _ := NewSQLiteStore(storePath)
	kit2, err := New(Config{Store: store2})
	require.NoError(t, err)
	defer kit2.Close()

	deployed := kit2.ListDeployedWASM()
	require.Empty(t, deployed, "undeployed shard should not be redeployed")

	// Module should still exist though (only shard removed, not module)
	info, _ := kit2.GetWASMModule("ephemeral")
	require.NotNil(t, info, "module should still exist after undeploy")
}
