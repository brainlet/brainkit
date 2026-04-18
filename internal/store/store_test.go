package store

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- KitStore tests (run against any backend) ---

func testKitStoreDeployments(t *testing.T, store types.KitStore) {
	now := time.Now().Truncate(time.Second)

	// Save
	err := store.SaveDeployment(types.PersistedDeployment{
		Source: "hello.ts", Code: "export default {}", Order: 1,
		DeployedAt: now, PackageName: "test",
	})
	require.NoError(t, err)

	// Load all
	deps, err := store.LoadDeployments()
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "hello.ts", deps[0].Source)
	assert.Equal(t, "export default {}", deps[0].Code)
	assert.Equal(t, 1, deps[0].Order)
	assert.Equal(t, "test", deps[0].PackageName)

	// Load one
	dep, err := store.LoadDeployment("hello.ts")
	require.NoError(t, err)
	assert.Equal(t, "hello.ts", dep.Source)

	// Upsert
	err = store.SaveDeployment(types.PersistedDeployment{
		Source: "hello.ts", Code: "updated code", Order: 2,
		DeployedAt: now, PackageName: "test",
	})
	require.NoError(t, err)
	dep, _ = store.LoadDeployment("hello.ts")
	assert.Equal(t, "updated code", dep.Code)

	// Delete
	err = store.DeleteDeployment("hello.ts")
	require.NoError(t, err)
	deps, _ = store.LoadDeployments()
	assert.Len(t, deps, 0)
}

func testKitStoreSchedules(t *testing.T, store types.KitStore) {
	now := time.Now().Truncate(time.Second)

	err := store.SaveSchedule(types.PersistedSchedule{
		ID: "sched-1", Expression: "every 5m", Duration: 5 * time.Minute,
		Topic: "test.topic", Payload: json.RawMessage(`{"key":"val"}`),
		Source: "cron.ts", CreatedAt: now, NextFire: now.Add(5 * time.Minute),
	})
	require.NoError(t, err)

	schedules, err := store.LoadSchedules()
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	assert.Equal(t, "sched-1", schedules[0].ID)
	assert.Equal(t, 5*time.Minute, schedules[0].Duration)
	assert.Contains(t, string(schedules[0].Payload), "key")

	// Delete
	store.DeleteSchedule("sched-1")
	schedules, _ = store.LoadSchedules()
	assert.Len(t, schedules, 0)
}

func testKitStoreScheduleFires(t *testing.T, store types.KitStore) {
	now := time.Now()

	// First claim succeeds
	claimed, err := store.ClaimScheduleFire("sched-1", now)
	require.NoError(t, err)
	assert.True(t, claimed, "first claim should succeed")

	// Second claim for same time fails (dedup)
	claimed, err = store.ClaimScheduleFire("sched-1", now)
	require.NoError(t, err)
	assert.False(t, claimed, "duplicate claim should fail")

	// Different time succeeds
	claimed, err = store.ClaimScheduleFire("sched-1", now.Add(time.Second))
	require.NoError(t, err)
	assert.True(t, claimed, "different fire time should succeed")
}

func testKitStorePlugins(t *testing.T, store types.KitStore) {
	now := time.Now().Truncate(time.Second)

	// Installed plugins
	err := store.SaveInstalledPlugin(types.InstalledPlugin{
		Name: "kv", Owner: "brainlet", Version: "1.0.0",
		BinaryPath: "/usr/bin/kv", Manifest: `{"tools":["set","get"]}`,
		InstalledAt: now,
	})
	require.NoError(t, err)

	plugins, err := store.LoadInstalledPlugins()
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, "kv", plugins[0].Name)

	store.DeleteInstalledPlugin("kv")
	plugins, _ = store.LoadInstalledPlugins()
	assert.Len(t, plugins, 0)

	// Running plugins
	err = store.SaveRunningPlugin(types.RunningPluginRecord{
		Name: "kv", BinaryPath: "/usr/bin/kv",
		Env: map[string]string{"KEY": "val"}, Config: json.RawMessage(`{"port":8080}`),
		StartOrder: 1, StartedAt: now,
	})
	require.NoError(t, err)

	running, err := store.LoadRunningPlugins()
	require.NoError(t, err)
	require.Len(t, running, 1)
	assert.Equal(t, "kv", running[0].Name)
	assert.Equal(t, "val", running[0].Env["KEY"])

	store.DeleteRunningPlugin("kv")
	running, _ = store.LoadRunningPlugins()
	assert.Len(t, running, 0)
}

// AuditStore tests moved to modules/audit/stores.

// --- SQLite backend tests ---

func TestSQLiteKitStore(t *testing.T) {
	makeStore := func(t *testing.T) types.KitStore {
		t.Helper()
		s, err := NewSQLiteKitStore(filepath.Join(t.TempDir(), "kit.db"))
		require.NoError(t, err)
		t.Cleanup(func() { s.Close() })
		return s
	}

	t.Run("deployments", func(t *testing.T) { testKitStoreDeployments(t, makeStore(t)) })
	t.Run("schedules", func(t *testing.T) { testKitStoreSchedules(t, makeStore(t)) })
	t.Run("schedule_fires", func(t *testing.T) { testKitStoreScheduleFires(t, makeStore(t)) })
	t.Run("plugins", func(t *testing.T) { testKitStorePlugins(t, makeStore(t)) })
}

// Audit store tests moved to modules/audit/stores (stores.SQLite / stores.Postgres).

// --- Factory tests ---

func TestFactory_SQLite(t *testing.T) {
	dir := t.TempDir()

	kitStore, err := NewKitStore(Config{Backend: "sqlite", SQLitePath: filepath.Join(dir, "kit.db")})
	require.NoError(t, err)
	defer kitStore.Close()

	// Verify it works
	err = kitStore.SaveDeployment(types.PersistedDeployment{Source: "test.ts", Code: "code", DeployedAt: time.Now()})
	require.NoError(t, err)
}

func TestFactory_UnknownBackend(t *testing.T) {
	_, err := NewKitStore(Config{Backend: "redis"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}
