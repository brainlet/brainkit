package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShardFixture_DeployDefaultMode(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { on, reply } from "brainkit";

export function init(): void {
  on("test.default", "handle");
}

export function handle(topic: string, payload: string): void {
  reply('{"mode":"default"}');
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"default-mode\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("default-mode")
	require.NoError(t, err)
	require.Equal(t, "stateless", desc.Mode, "no setMode() should default to stateless")

	result := injectEvent(t, kit, "default-mode", "test.default", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"mode":"default"`)
}

func TestShardFixture_DeployInvalidMode(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode, on } from "brainkit";

export function init(): void {
  setMode("bogus");
  on("test.x", "handle");
}

export function handle(topic: string, payload: string): void {}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"bad-mode\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("bad-mode")
	if err != nil {
		require.Contains(t, err.Error(), "invalid shard mode")
	} else {
		require.Empty(t, desc.Module, "invalid mode should not produce a valid descriptor")
	}
}

func TestShardFixture_DeployNoHandlers(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode } from "brainkit";

export function init(): void {
  setMode("stateless");
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"empty-shard\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("empty-shard")
	require.NoError(t, err)
	require.Equal(t, "stateless", desc.Mode)
	require.Empty(t, desc.Handlers, "shard with no on() calls should have empty handlers")
}

func TestShardFixture_UndeployAndRedeploy(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-echo.ts", "redeploy-test")

	result := injectEvent(t, kit, "redeploy-test", "test.echo", map[string]string{"v": "1"})
	require.Equal(t, `{"v":"1"}`, result.ReplyPayload)

	err := kit.UndeployWASM("redeploy-test")
	require.NoError(t, err)

	_, err = kit.InjectWASMEvent("redeploy-test", "test.echo", json.RawMessage(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not deployed")

	desc, err := kit.DeployWASM("redeploy-test")
	require.NoError(t, err)
	require.Equal(t, "stateless", desc.Mode)

	result = injectEvent(t, kit, "redeploy-test", "test.echo", map[string]string{"v": "2"})
	require.Equal(t, `{"v":"2"}`, result.ReplyPayload)
}
