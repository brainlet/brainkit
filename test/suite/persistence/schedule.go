package persistence

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testScheduleSurvivesRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store1,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = k1.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 1h",
		Topic:      "test.hourly.persist",
		Payload:    json.RawMessage(`{"hourly":true}`),
		Source:     "test",
	})
	require.NoError(t, err)
	assert.Len(t, k1.ListSchedules(), 1)
	k1.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	schedules := k2.ListSchedules()
	require.Len(t, schedules, 1)
	assert.Equal(t, "test.hourly.persist", schedules[0].Topic)
}

func testMissedRecurringCatchUp(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(brainkit.PersistedSchedule{
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
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	schedules := k.ListSchedules()
	require.Len(t, schedules, 1)
	assert.True(t, schedules[0].NextFire.After(time.Now()), "NextFire should be in the future after catch-up")
}

func testExpiredOneTimeFires(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(brainkit.PersistedSchedule{
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
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	time.Sleep(300 * time.Millisecond)
	assert.Empty(t, k.ListSchedules(), "expired one-time should be removed after firing")

	loaded, _ := store2.LoadSchedules()
	assert.Empty(t, loaded)
}
