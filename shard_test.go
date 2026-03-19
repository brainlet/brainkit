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
import { setMode, on, log, setState, reply } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.ping", "handlePing");
}

export function handlePing(topic: string, payload: string): void {
  log("got ping: " + payload);
  setState("seen", "yes");
  reply('{"status":"ok"}');
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
	require.Empty(t, result.Error)

	// Undeploy
	err = kit.UndeployWASM("pinger")
	require.NoError(t, err, "undeploy")

	// Verify undeploy worked
	deployed := kit.ListDeployedWASM()
	require.Empty(t, deployed)
}

func TestShard_Deploy_Persistent(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode, on, setState, getState, reply } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("counter.increment", "handleIncrement");
  on("counter.get", "handleGet");
}

export function handleIncrement(topic: string, payload: string): void {
  var count: i32 = 0;
  var raw = getState("count");
  if (raw.length > 0) {
    count = I32.parseInt(raw);
  }
  count++;
  setState("count", count.toString());
  reply('{"ok":true}');
}

export function handleGet(topic: string, payload: string): void {
  var raw = getState("count");
  reply('{"count":' + (raw.length > 0 ? raw : '0') + '}');
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"counter\" });",
		"`"+source+"`"))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("counter")
	require.NoError(t, err)
	require.Equal(t, "persistent", desc.Mode)

	// Increment 3 times
	for i := 0; i < 3; i++ {
		result, err := kit.InjectWASMEvent("counter", "counter.increment", json.RawMessage(`{}`))
		require.NoError(t, err)
		require.Empty(t, result.Error)
	}

	// Get should return 3
	result, err := kit.InjectWASMEvent("counter", "counter.get", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, result.ReplyPayload, `"count":3`)

	kit.UndeployWASM("counter")
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

	t.Run("missing_export", func(t *testing.T) {
		err := validateShardDescriptor(&ShardDescriptor{
			Mode:     "stateless",
			Handlers: map[string]string{"topic": "nonexistent"},
		}, []string{"init", "run"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found in module exports")
	})

	t.Run("valid_stateless", func(t *testing.T) {
		err := validateShardDescriptor(&ShardDescriptor{
			Mode:     "stateless",
			Handlers: map[string]string{"topic": "handleIt"},
		}, []string{"init", "handleIt"})
		require.NoError(t, err)
	})

	t.Run("valid_persistent", func(t *testing.T) {
		err := validateShardDescriptor(&ShardDescriptor{
			Mode:     "persistent",
			Handlers: map[string]string{"topic": "handleIt"},
		}, []string{"init", "handleIt"})
		require.NoError(t, err)
	})
}

func TestShard_RemoveInterlock(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode, on, log } from "brainkit";
export function init(): void {
  setMode("stateless");
  on("test.x", "handle");
}
export function handle(topic: string, payload: string): void { log("handled"); }
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
