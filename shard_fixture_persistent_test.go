package brainkit

import (
	"fmt"
	"testing"

	"github.com/brainlet/brainkit/bus"
	"github.com/stretchr/testify/require"
)

func TestShardFixture_PersistentCounter(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/persistent-counter.ts", "counter")

	require.Equal(t, "persistent", desc.Mode)
	require.Len(t, desc.Handlers, 3)

	result := injectEvent(t, kit, "counter", "counter.get", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":0`)

	for i := 1; i <= 5; i++ {
		result = injectEvent(t, kit, "counter", "counter.inc", map[string]string{})
		require.Contains(t, result.ReplyPayload, fmt.Sprintf(`"count":%d`, i))
	}

	result = injectEvent(t, kit, "counter", "counter.get", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":5`)

	result = injectEvent(t, kit, "counter", "counter.reset", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":0`)

	result = injectEvent(t, kit, "counter", "counter.get", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":0`)
}

func TestShardFixture_PersistentKVStore(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/persistent-kv-store.ts", "kvstore")

	require.Equal(t, "persistent", desc.Mode)

	result := injectEvent(t, kit, "kvstore", "kv.has", map[string]string{"key": "name"})
	require.Contains(t, result.ReplyPayload, `"exists":false`)

	result = injectEvent(t, kit, "kvstore", "kv.set", map[string]string{"key": "name", "value": "david"})
	require.Contains(t, result.ReplyPayload, `"ok":true`)

	result = injectEvent(t, kit, "kvstore", "kv.get", map[string]string{"key": "name"})
	require.Contains(t, result.ReplyPayload, `"value":"david"`)

	result = injectEvent(t, kit, "kvstore", "kv.has", map[string]string{"key": "name"})
	require.Contains(t, result.ReplyPayload, `"exists":true`)

	result = injectEvent(t, kit, "kvstore", "kv.get", map[string]string{"key": "missing"})
	require.Contains(t, result.ReplyPayload, `"value":""`)
}

func TestShardFixture_PersistentEventLog(t *testing.T) {
	kit := newTestKitNoKey(t)

	notifications := make(chan string, 10)
	kit.Bus.On("audit.logged", func(msg bus.Message, _ bus.ReplyFunc) {
		notifications <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/persistent-event-log.ts", "auditlog")

	injectEvent(t, kit, "auditlog", "audit.event", map[string]string{"action": "login"})
	injectEvent(t, kit, "auditlog", "audit.event", map[string]string{"action": "access"})
	injectEvent(t, kit, "auditlog", "audit.event", map[string]string{"action": "logout"})

	result := injectEvent(t, kit, "auditlog", "audit.stats", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"eventCount":3`)

	require.Len(t, notifications, 3)
}
