package audit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStore_RecordAndQuery(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	store.Record(Event{Category: "plugin", Type: "plugin.started", Source: "kv"})
	store.Record(Event{Category: "plugin", Type: "plugin.stopped", Source: "kv"})
	store.Record(Event{Category: "security", Type: "tools.call.denied", Source: "read-db"})
	store.Record(Event{Category: "secrets", Type: "secrets.set", Source: "api-key"})

	// Query all
	all, err := store.Query(Query{})
	require.NoError(t, err)
	assert.Len(t, all, 4)

	// Query by category
	plugins, err := store.Query(Query{Category: "plugin"})
	require.NoError(t, err)
	assert.Len(t, plugins, 2)

	// Query by type
	denied, err := store.Query(Query{Type: "tools.call.denied"})
	require.NoError(t, err)
	assert.Len(t, denied, 1)
	assert.Equal(t, "read-db", denied[0].Source)

	// Query by source
	kvEvents, err := store.Query(Query{Source: "kv"})
	require.NoError(t, err)
	assert.Len(t, kvEvents, 2)

	// Count
	count, err := store.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(4), count)
}

func TestSQLiteStore_TimeRangeQuery(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	store.Record(Event{Timestamp: now.Add(-2 * time.Hour), Category: "plugin", Type: "plugin.started", Source: "old"})
	store.Record(Event{Timestamp: now.Add(-30 * time.Minute), Category: "plugin", Type: "plugin.started", Source: "recent"})
	store.Record(Event{Timestamp: now, Category: "plugin", Type: "plugin.started", Source: "now"})

	// Since 1 hour ago
	recent, err := store.Query(Query{Since: now.Add(-1 * time.Hour)})
	require.NoError(t, err)
	assert.Len(t, recent, 2)
	assert.Equal(t, "now", recent[0].Source)       // newest first
	assert.Equal(t, "recent", recent[1].Source)
}

func TestSQLiteStore_Prune(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	store.Record(Event{Timestamp: now.Add(-48 * time.Hour), Category: "plugin", Type: "old", Source: "old"})
	store.Record(Event{Timestamp: now, Category: "plugin", Type: "recent", Source: "recent"})

	// Prune events older than 24h
	err = store.Prune(24 * time.Hour)
	require.NoError(t, err)

	remaining, err := store.Query(Query{})
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
	assert.Equal(t, "recent", remaining[0].Source)
}

func TestSQLiteStore_DurationAndError(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	store.Record(Event{
		Category: "tools",
		Type:     "tools.call.completed",
		Source:   "echo",
		Duration: 150 * time.Millisecond,
	})
	store.Record(Event{
		Category: "tools",
		Type:     "tools.call.failed",
		Source:   "broken",
		Duration: 50 * time.Millisecond,
		Error:    "connection refused",
	})

	events, err := store.Query(Query{Category: "tools"})
	require.NoError(t, err)
	require.Len(t, events, 2)

	// Newest first
	assert.Equal(t, "tools.call.failed", events[0].Type)
	assert.Equal(t, 50*time.Millisecond, events[0].Duration)
	assert.Equal(t, "connection refused", events[0].Error)

	assert.Equal(t, "tools.call.completed", events[1].Type)
	assert.Equal(t, 150*time.Millisecond, events[1].Duration)
	assert.Empty(t, events[1].Error)
}

func TestRecorder_NilSafe(t *testing.T) {
	// nil Recorder should not panic
	var r *Recorder
	r.PluginStarted("test", 123)
	r.ToolCallCompleted("echo", "caller", time.Second)
	r.PermissionDenied("src", "call", "topic", "role")
}

func TestRecorder_RecordsEvents(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	r := NewRecorder(store, "runtime-1", "my-ns")

	r.PluginStarted("kv", 123)
	r.PluginStopped("kv", "stopped")
	r.ToolCallCompleted("echo", "caller-1", 50*time.Millisecond)
	r.ToolCallDenied("read-db", "attacker-runtime", "local-only")
	r.PermissionDenied("evil.ts", "publish", "secrets.list", "observer")
	r.SecretSet("api-key", "admin")
	r.Deployed("greeter.ts", 3)
	r.BusHandlerFailed("workflow.start", assert.AnError)

	all, err := store.Query(Query{})
	require.NoError(t, err)
	assert.Len(t, all, 8)

	// Verify runtimeID and namespace are stamped
	for _, e := range all {
		assert.Equal(t, "runtime-1", e.RuntimeID)
		assert.Equal(t, "my-ns", e.Namespace)
	}

	// Verify categories
	security, _ := store.Query(Query{Category: "security"})
	assert.Len(t, security, 2) // tools.call.denied + bus.permission.denied

	plugins, _ := store.Query(Query{Category: "plugin"})
	assert.Len(t, plugins, 2)

	tools, _ := store.Query(Query{Category: "tools"})
	assert.Len(t, tools, 1) // completed

	secrets, _ := store.Query(Query{Category: "secrets"})
	assert.Len(t, secrets, 1)
}
