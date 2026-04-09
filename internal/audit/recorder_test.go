package audit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerboseMode_BusCommandCompleted(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	// Normal mode — BusCommandCompleted should NOT record
	normal := NewRecorderWithConfig(RecorderConfig{
		Store: store, RuntimeID: "rt1", Namespace: "ns1", Verbosity: VerbosityNormal,
	})
	normal.BusCommandCompleted("tools.call", "caller-1", 50*time.Millisecond)

	events, _ := store.Query(Query{Type: "bus.command.completed"})
	assert.Len(t, events, 0, "normal mode should not record bus command completions")

	// Verbose mode — BusCommandCompleted SHOULD record
	verbose := NewRecorderWithConfig(RecorderConfig{
		Store: store, RuntimeID: "rt1", Namespace: "ns1", Verbosity: VerbosityVerbose,
	})
	verbose.BusCommandCompleted("kit.deploy", "caller-2", 100*time.Millisecond)

	events, _ = store.Query(Query{Type: "bus.command.completed"})
	assert.Len(t, events, 1, "verbose mode should record bus command completions")
	assert.Equal(t, "kit.deploy", events[0].Source)
	assert.Equal(t, 100*time.Millisecond, events[0].Duration)
}

func TestVerboseMode_MetricsSnapshot(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	// Normal mode — skipped
	normal := NewRecorder(store, "rt1", "ns1")
	normal.MetricsSnapshot(map[string]any{"activeHandlers": 5})
	events, _ := store.Query(Query{Type: "metrics.snapshot"})
	assert.Len(t, events, 0)

	// Verbose mode — recorded
	verbose := NewRecorderWithConfig(RecorderConfig{
		Store: store, RuntimeID: "rt1", Namespace: "ns1", Verbosity: VerbosityVerbose,
	})
	verbose.MetricsSnapshot(map[string]any{"activeHandlers": 5, "activeDeployments": 3})
	events, _ = store.Query(Query{Type: "metrics.snapshot"})
	assert.Len(t, events, 1)
	assert.Contains(t, string(events[0].Data), "activeHandlers")
}

func TestIsVerbose(t *testing.T) {
	store, _ := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	defer store.Close()

	normal := NewRecorder(store, "rt1", "ns1")
	assert.False(t, normal.IsVerbose())

	verbose := NewRecorderWithConfig(RecorderConfig{
		Store: store, Verbosity: VerbosityVerbose,
	})
	assert.True(t, verbose.IsVerbose())

	var nilRec *Recorder
	assert.False(t, nilRec.IsVerbose())
}

func TestCountByCategory(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "audit.db"))
	require.NoError(t, err)
	defer store.Close()

	store.Record(Event{Category: "plugin", Type: "plugin.started", Source: "a"})
	store.Record(Event{Category: "plugin", Type: "plugin.stopped", Source: "a"})
	store.Record(Event{Category: "security", Type: "tools.call.denied", Source: "b"})
	store.Record(Event{Category: "tools", Type: "tools.call.completed", Source: "c"})
	store.Record(Event{Category: "tools", Type: "tools.call.completed", Source: "d"})
	store.Record(Event{Category: "tools", Type: "tools.call.completed", Source: "e"})

	counts, err := store.CountByCategory()
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts["plugin"])
	assert.Equal(t, int64(1), counts["security"])
	assert.Equal(t, int64(3), counts["tools"])
}
