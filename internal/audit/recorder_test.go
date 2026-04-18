package audit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memStore is a minimal in-memory Store for recorder tests. The real
// SQLite/Postgres stores moved to modules/audit/stores to avoid the
// internal/audit → internal/store import cycle; the recorder only needs
// Record/Query/Count/CountByCategory for these tests.
type memStore struct {
	mu     sync.Mutex
	events []Event
}

func newMemStore() *memStore { return &memStore{} }

func (s *memStore) Record(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
}

func (s *memStore) Query(q Query) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Event
	for _, e := range s.events {
		if q.Category != "" && e.Category != q.Category {
			continue
		}
		if q.Type != "" && e.Type != q.Type {
			continue
		}
		if q.Source != "" && e.Source != q.Source {
			continue
		}
		out = append(out, e)
	}
	if out == nil {
		out = []Event{}
	}
	return out, nil
}

func (s *memStore) Prune(_ time.Duration) error { return nil }

func (s *memStore) Count() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int64(len(s.events)), nil
}

func (s *memStore) CountByCategory() (map[string]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]int64)
	for _, e := range s.events {
		result[e.Category]++
	}
	return result, nil
}

func (s *memStore) Close() error { return nil }

func TestVerboseMode_BusCommandCompleted(t *testing.T) {
	store := newMemStore()

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
	store := newMemStore()

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
	store := newMemStore()

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
	store := newMemStore()

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
