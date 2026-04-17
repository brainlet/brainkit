package persistence

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/internal/types"
	schedulesmod "github.com/brainlet/brainkit/modules/schedules"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testScheduleSurvivesRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store1,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store1})},
	})
	require.NoError(t, err)

	testutil.Schedule(t, k1, "every 1h", "test.hourly.persist", json.RawMessage(`{"hourly":true}`))
	assert.Len(t, listSchedules(t, k1), 1)
	k1.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store2,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store2})},
	})
	require.NoError(t, err)
	defer k2.Close()

	schedules := listSchedules(t, k2)
	require.Len(t, schedules, 1)
	assert.Equal(t, "test.hourly.persist", schedules[0].Topic)
}

func testMissedRecurringCatchUp(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(types.PersistedSchedule{
		ID:         "missed-persist-1",
		Expression: "every 1h",
		Duration:   time.Hour,
		Topic:      "test.missed.persist",
		Payload:    json.RawMessage(`{"catchup":true}`),
		Source:     "test",
		CreatedAt:  time.Now().Add(-3 * time.Hour),
		NextFire:   time.Now().Add(-2 * time.Hour),
		OneTime:    false,
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store2,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store2})},
	})
	require.NoError(t, err)
	defer k.Close()

	// Verify schedule survived via store-level load (bus ListSchedules returns ScheduleInfo, not NextFire as time.Time)
	loaded, err := store2.LoadSchedules()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.True(t, loaded[0].NextFire.After(time.Now()), "NextFire should be in the future after catch-up")
}

func testExpiredOneTimeFires(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(types.PersistedSchedule{
		ID:         "expired-persist-1",
		Expression: "in 30s",
		Duration:   30 * time.Second,
		Topic:      "test.expired.persist",
		Payload:    json.RawMessage(`{"expired":true}`),
		Source:     "test",
		CreatedAt:  time.Now().Add(-1 * time.Minute),
		NextFire:   time.Now().Add(-30 * time.Second),
		OneTime:    true,
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store2,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store2})},
	})
	require.NoError(t, err)
	defer k.Close()

	time.Sleep(300 * time.Millisecond)
	assert.Empty(t, listSchedules(t, k), "expired one-time should be removed after firing")

	loaded, _ := store2.LoadSchedules()
	assert.Empty(t, loaded)
}
