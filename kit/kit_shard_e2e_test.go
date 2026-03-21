package kit

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/bus"
	"github.com/stretchr/testify/require"
)

func TestShardE2E_ChainReaction(t *testing.T) {
	kit := newTestKitNoKey(t)

	desc := deployShard(t, kit, "testdata/as/shard/chain-reaction.ts", "chain")
	require.Equal(t, "persistent", desc.Mode)
	require.Len(t, desc.Handlers, 2)

	// Listen for step2 events (chain reaction)
	step2Received := make(chan string, 5)
	kit.Bus.On("chain.step2", func(msg bus.Message, _ bus.ReplyFunc) {
		step2Received <- string(msg.Payload)
	})

	result := injectEvent(t, kit, "chain", "chain.start", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"started":true`)

	// Wait for the chain to fire step2
	select {
	case payload := <-step2Received:
		require.Contains(t, payload, "step1")
	case <-time.After(5 * time.Second):
		t.Fatal("chain.step2 not received")
	}
}

func TestShardE2E_DeployUndeployRedeploy(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Deploy
	deployShard(t, kit, "testdata/as/shard/stateless-echo.ts", "echo-cycle")

	// Verify it works
	result := injectEvent(t, kit, "echo-cycle", "test.echo", map[string]string{"msg": "first"})
	require.Contains(t, result.ReplyPayload, "first")

	// Undeploy
	err := kit.UndeployWASM("echo-cycle")
	require.NoError(t, err)

	// Inject should fail — shard not deployed
	_, err = kit.InjectWASMEvent("echo-cycle", "test.echo", json.RawMessage(`{"msg":"dead"}`))
	require.Error(t, err)

	// Redeploy
	deployShard(t, kit, "testdata/as/shard/stateless-echo.ts", "echo-cycle")

	// Works again
	result = injectEvent(t, kit, "echo-cycle", "test.echo", map[string]string{"msg": "second"})
	require.Contains(t, result.ReplyPayload, "second")
}

func TestShardE2E_ConcurrentShards(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Deploy 3 persistent counters with different names
	for i := range 3 {
		name := fmt.Sprintf("counter-%d", i)
		deployShard(t, kit, "testdata/as/shard/persistent-counter.ts", name)
	}

	// Send 10 events to each shard concurrently
	var wg sync.WaitGroup
	const eventsPerShard = 10

	for i := range 3 {
		shard := fmt.Sprintf("counter-%d", i)
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			for j := range eventsPerShard {
				result, err := kit.InjectWASMEvent(s, "counter.inc", json.RawMessage(fmt.Sprintf(`{"n":%d}`, j)))
				if err != nil {
					t.Errorf("shard %s event %d: %v", s, j, err)
					return
				}
				if result.Error != "" {
					t.Errorf("shard %s event %d error: %s", s, j, result.Error)
				}
			}
		}(shard)
	}

	wg.Wait()

	// Each counter should be at 10
	for i := range 3 {
		shard := fmt.Sprintf("counter-%d", i)
		result := injectEvent(t, kit, shard, "counter.get", map[string]string{})
		require.Contains(t, result.ReplyPayload, fmt.Sprintf(`"count":%d`, eventsPerShard))
	}
}

func TestShardE2E_PersistentStateAccumulates(t *testing.T) {
	kit := newTestKitNoKey(t)

	deployShard(t, kit, "testdata/as/shard/persistent-counter.ts", "acc")

	// Send 20 increments
	for range 20 {
		result, err := kit.InjectWASMEvent("acc", "counter.inc", json.RawMessage(`{}`))
		require.NoError(t, err)
		require.Empty(t, result.Error)
	}

	// Verify count
	result := injectEvent(t, kit, "acc", "counter.get", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":20`)
}

// injectEvent is a test helper (defined in shard_fixture_test.go)
// deployShard is a test helper (defined in shard_fixture_test.go)
