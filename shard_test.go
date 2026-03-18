package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShard_Deploy_Stateless(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode, onEvent, log, setState, getState } from "wasm";

export function init(): void {
  setMode("stateless");
  onEvent("test.ping", "handlePing");
}

export function handlePing(payload: string): i32 {
  log("got ping: " + payload);
  setState("seen", "yes");
  return 0;
}
`
	// Compile
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "pinger", runtime: "incremental" });
	`)
	require.NoError(t, err, "compile")

	// Deploy
	desc, err := kit.DeployWASM("pinger")
	require.NoError(t, err, "deploy")
	require.Equal(t, "stateless", desc.Mode)
	require.Contains(t, desc.Handlers, "test.ping")
	require.Equal(t, "handlePing", desc.Handlers["test.ping"])

	// Inject event
	result, err := kit.InjectWASMEvent("pinger", "test.ping", json.RawMessage(`{"msg":"hello"}`))
	require.NoError(t, err, "inject")
	require.Equal(t, 0, result.ExitCode)

	// Undeploy
	err = kit.UndeployWASM("pinger")
	require.NoError(t, err, "undeploy")

	// Verify undeploy worked
	deployed := kit.ListDeployedWASM()
	require.Empty(t, deployed)
}

func TestShard_Deploy_Shared(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode, onEvent, setState, getState } from "wasm";

export function init(): void {
  setMode("shared");
  onEvent("counter.increment", "handleIncrement");
  onEvent("counter.get", "handleGet");
}

export function handleIncrement(payload: string): i32 {
  var count: i32 = 0;
  var raw = getState("count");
  if (raw.length > 0) {
    count = I32.parseInt(raw);
  }
  count++;
  setState("count", count.toString());
  return 0;
}

export function handleGet(payload: string): i32 {
  var raw = getState("count");
  if (raw.length == 0) return 1;
  return I32.parseInt(raw);
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"counter\" });",
		"`"+source+"`"))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("counter")
	require.NoError(t, err)
	require.Equal(t, "shared", desc.Mode)

	// Increment 3 times
	for i := 0; i < 3; i++ {
		result, err := kit.InjectWASMEvent("counter", "counter.increment", json.RawMessage(`{}`))
		require.NoError(t, err)
		require.Equal(t, 0, result.ExitCode)
	}

	// Get should return 3
	result, err := kit.InjectWASMEvent("counter", "counter.get", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Equal(t, 3, result.ExitCode, "expected count=3")

	kit.UndeployWASM("counter")
}

func TestShard_Deploy_Keyed(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setModeKeyed, onEvent, setState, getState } from "wasm";

export function init(): void {
  setModeKeyed("userId");
  onEvent("user.visit", "handleVisit");
  onEvent("user.check", "handleCheck");
}

export function handleVisit(payload: string): i32 {
  var count: i32 = 0;
  var raw = getState("visits");
  if (raw.length > 0) {
    count = I32.parseInt(raw);
  }
  count++;
  setState("visits", count.toString());
  return 0;
}

export function handleCheck(payload: string): i32 {
  var raw = getState("visits");
  if (raw.length == 0) return 0;
  return I32.parseInt(raw);
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"visitor\" });",
		"`"+source+"`"))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("visitor")
	require.NoError(t, err)
	require.Equal(t, "keyed", desc.Mode)
	require.Equal(t, "userId", desc.StateKey)

	// Alice visits 2 times
	kit.InjectWASMEvent("visitor", "user.visit", json.RawMessage(`{"userId":"alice"}`))
	kit.InjectWASMEvent("visitor", "user.visit", json.RawMessage(`{"userId":"alice"}`))

	// Bob visits 1 time
	kit.InjectWASMEvent("visitor", "user.visit", json.RawMessage(`{"userId":"bob"}`))

	// Check Alice = 2
	r1, _ := kit.InjectWASMEvent("visitor", "user.check", json.RawMessage(`{"userId":"alice"}`))
	require.Equal(t, 2, r1.ExitCode, "alice should have 2 visits")

	// Check Bob = 1
	r2, _ := kit.InjectWASMEvent("visitor", "user.check", json.RawMessage(`{"userId":"bob"}`))
	require.Equal(t, 1, r2.ExitCode, "bob should have 1 visit")

	// Check unknown user = 0
	r3, _ := kit.InjectWASMEvent("visitor", "user.check", json.RawMessage(`{"userId":"charlie"}`))
	require.Equal(t, 0, r3.ExitCode, "charlie should have 0 visits")

	kit.UndeployWASM("visitor")
}

func TestShard_Validation(t *testing.T) {
	t.Run("invalid_mode", func(t *testing.T) {
		err := validateShardDescriptor(&ShardDescriptor{
			Mode:     "bogus",
			Handlers: map[string]string{},
		}, []string{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid shard mode")
	})

	t.Run("keyed_no_key", func(t *testing.T) {
		err := validateShardDescriptor(&ShardDescriptor{
			Mode:     "keyed",
			StateKey: "",
			Handlers: map[string]string{},
		}, []string{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "state key field")
	})

	t.Run("missing_export", func(t *testing.T) {
		err := validateShardDescriptor(&ShardDescriptor{
			Mode:     "stateless",
			Handlers: map[string]string{"topic": "nonexistent"},
		}, []string{"init", "run"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found in module exports")
	})

	t.Run("valid", func(t *testing.T) {
		err := validateShardDescriptor(&ShardDescriptor{
			Mode:     "keyed",
			StateKey: "id",
			Handlers: map[string]string{"topic": "handleIt"},
		}, []string{"init", "handleIt"})
		require.NoError(t, err)
	})
}

func TestShard_RemoveInterlock(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode, onEvent, log } from "wasm";
export function init(): void {
  setMode("stateless");
  onEvent("test.x", "handle");
}
export function handlePing(payload: string): i32 { return 0; }
export function handle(payload: string): i32 { return 0; }
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"guarded\" });",
		"`"+source+"`"))
	require.NoError(t, err)

	_, err = kit.DeployWASM("guarded")
	require.NoError(t, err)

	// Remove should fail while deployed
	err = kit.RemoveWASMModule("guarded")
	require.Error(t, err)
	require.Contains(t, err.Error(), "shard is deployed")

	// Undeploy then remove should work
	kit.UndeployWASM("guarded")
	err = kit.RemoveWASMModule("guarded")
	require.NoError(t, err)
}
