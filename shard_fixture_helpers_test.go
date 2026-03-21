package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// compileShard compiles an AS fixture file, returning the module name.
func compileShard(t *testing.T, kit *Kit, fixturePath, name string) {
	t.Helper()
	source := loadFixture(t, fixturePath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: %q });",
		"`"+source+"`", name,
	))
	require.NoError(t, err, "compile %s", fixturePath)
}

// deployShard compiles + deploys and returns the descriptor.
func deployShard(t *testing.T, kit *Kit, fixturePath, name string) *ShardDescriptor {
	t.Helper()
	compileShard(t, kit, fixturePath, name)
	desc, err := kit.DeployWASM(name)
	require.NoError(t, err, "deploy %s", name)
	return desc
}

// injectEvent sends an event to a shard and returns the result.
func injectEvent(t *testing.T, kit *Kit, shard, topic string, payload interface{}) *WASMEventResult {
	t.Helper()
	data, _ := json.Marshal(payload)
	result, err := kit.InjectWASMEvent(shard, topic, json.RawMessage(data))
	require.NoError(t, err, "inject %s → %s", shard, topic)
	require.Empty(t, result.Error, "handler error for %s → %s", shard, topic)
	return result
}
