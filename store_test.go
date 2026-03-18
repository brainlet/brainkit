package brainkit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSQLiteStore_ModuleRoundTrip(t *testing.T) {
	store := newTestStore(t)

	info := WASMModuleInfo{
		Name:       "calc",
		Size:       1024,
		Exports:    []string{"run", "__new"},
		CompiledAt: time.Now().Format(time.RFC3339),
		SourceHash: "abc123",
	}
	binary := []byte{0x00, 0x61, 0x73, 0x6d} // WASM magic

	err := store.SaveModule("calc", binary, info)
	require.NoError(t, err)

	modules, err := store.LoadModules()
	require.NoError(t, err)
	require.Len(t, modules, 1)

	mod := modules["calc"]
	require.NotNil(t, mod)
	require.Equal(t, "calc", mod.Name)
	require.Equal(t, binary, mod.Binary)
	require.Equal(t, "abc123", mod.SourceHash)
	require.Equal(t, 1024, mod.Size)
	require.Contains(t, mod.Exports, "run")
	require.Contains(t, mod.Exports, "__new")
}

func TestSQLiteStore_ShardRoundTrip(t *testing.T) {
	store := newTestStore(t)

	desc := ShardDescriptor{
		Module:     "processor",
		Mode:       "keyed",
		StateKey:   "orderId",
		Handlers:   map[string]string{"order.new": "onNew", "order.done": "onDone"},
		DeployedAt: time.Now(),
	}

	err := store.SaveShard("processor", desc)
	require.NoError(t, err)

	shards, err := store.LoadShards()
	require.NoError(t, err)
	require.Len(t, shards, 1)

	loaded := shards["processor"]
	require.Equal(t, "keyed", loaded.Mode)
	require.Equal(t, "orderId", loaded.StateKey)
	require.Equal(t, "onNew", loaded.Handlers["order.new"])
	require.Equal(t, "onDone", loaded.Handlers["order.done"])
}

func TestSQLiteStore_StateRoundTrip(t *testing.T) {
	store := newTestStore(t)

	// Shared mode (key = "")
	err := store.SaveState("counter", "", map[string]string{"count": "42"})
	require.NoError(t, err)

	state, err := store.LoadState("counter", "")
	require.NoError(t, err)
	require.Equal(t, "42", state["count"])

	// Keyed mode
	err = store.SaveState("orders", "abc-123", map[string]string{"status": "pending"})
	require.NoError(t, err)
	err = store.SaveState("orders", "def-456", map[string]string{"status": "shipped"})
	require.NoError(t, err)

	s1, err := store.LoadState("orders", "abc-123")
	require.NoError(t, err)
	require.Equal(t, "pending", s1["status"])

	s2, err := store.LoadState("orders", "def-456")
	require.NoError(t, err)
	require.Equal(t, "shipped", s2["status"])

	// Non-existent key
	s3, err := store.LoadState("orders", "nope")
	require.NoError(t, err)
	require.Nil(t, s3)
}

func TestSQLiteStore_Delete(t *testing.T) {
	store := newTestStore(t)

	// Module
	store.SaveModule("temp", []byte{1, 2}, WASMModuleInfo{Name: "temp", Size: 2, Exports: []string{}, CompiledAt: time.Now().Format(time.RFC3339)})
	store.DeleteModule("temp")
	modules, _ := store.LoadModules()
	require.Empty(t, modules)

	// Shard
	store.SaveShard("temp", ShardDescriptor{Module: "temp", Mode: "stateless", Handlers: map[string]string{}})
	store.DeleteShard("temp")
	shards, _ := store.LoadShards()
	require.Empty(t, shards)

	// State
	store.SaveState("temp", "k1", map[string]string{"a": "1"})
	store.SaveState("temp", "k2", map[string]string{"b": "2"})
	store.DeleteState("temp") // deletes ALL state for shard
	s, _ := store.LoadState("temp", "k1")
	require.Nil(t, s)
	s2, _ := store.LoadState("temp", "k2")
	require.Nil(t, s2)
}

func TestSQLiteStore_EmptyLoad(t *testing.T) {
	store := newTestStore(t)

	modules, err := store.LoadModules()
	require.NoError(t, err)
	require.Empty(t, modules)

	shards, err := store.LoadShards()
	require.NoError(t, err)
	require.Empty(t, shards)

	state, err := store.LoadState("nope", "")
	require.NoError(t, err)
	require.Nil(t, state)
}

func TestSQLiteStore_PathCreatesDir(t *testing.T) {
	dir := t.TempDir()
	deepPath := filepath.Join(dir, "a", "b", "c", "store.db")

	store, err := NewSQLiteStore(deepPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify file exists
	_, err = os.Stat(deepPath)
	require.NoError(t, err)
}
