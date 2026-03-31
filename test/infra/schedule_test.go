package infra_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedule_EveryFiresRepeatedly(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, err := k.SubscribeRaw(ctx, "test.tick", func(msg messages.Message) {
		count.Add(1)
	})
	require.NoError(t, err)
	defer unsub()

	id, err := k.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 200ms",
		Topic:      "test.tick",
		Payload:    json.RawMessage(`{"tick":true}`),
		Source:     "test",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	time.Sleep(700 * time.Millisecond)
	got := count.Load()
	assert.GreaterOrEqual(t, got, int32(3), "every 200ms should fire 3+ times in 700ms, got %d", got)
}

func TestSchedule_InFiresOnce(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, _ := k.SubscribeRaw(ctx, "test.once", func(msg messages.Message) {
		count.Add(1)
	})
	defer unsub()

	_, err := k.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 100ms",
		Topic:      "test.once",
		Payload:    json.RawMessage(`{"once":true}`),
		Source:     "test",
	})
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load(), "in 100ms should fire exactly once")
	assert.Empty(t, k.ListSchedules(), "one-time schedule should be removed after firing")
}

func TestSchedule_Unschedule(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, _ := k.SubscribeRaw(ctx, "test.cancel", func(msg messages.Message) {
		count.Add(1)
	})
	defer unsub()

	id, _ := k.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 100ms",
		Topic:      "test.cancel",
		Payload:    json.RawMessage(`{}`),
		Source:     "test",
	})

	time.Sleep(250 * time.Millisecond)
	k.Unschedule(ctx, id)
	countAtCancel := count.Load()

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, countAtCancel, count.Load(), "unschedule should stop firing")
}

func TestSchedule_InvalidExpression(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := k.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "cron 0 9 * * *",
		Topic:      "test.invalid",
		Source:     "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schedule expression")
}

func TestSchedule_SurvivesRestart(t *testing.T) {
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
		Topic:      "test.hourly",
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
	assert.Equal(t, "test.hourly", schedules[0].Topic)
}

func TestSchedule_MissedRecurringCatchUp(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID:         "missed-1",
		Expression: "every 1h",
		Duration:   time.Hour,
		Topic:      "test.missed",
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

func TestSchedule_ExpiredOneTimeFires(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(brainkit.PersistedSchedule{
		ID:         "expired-1",
		Expression: "in 30s",
		Duration:   30 * time.Second,
		Topic:      "test.expired",
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

func TestSchedule_TeardownCancels(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "scheduler.ts",
		Code:   `bus.schedule("every 200ms", "tick", {});`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	assert.NotEmpty(t, k.ListSchedules())

	tpr, _ := sdk.Publish(k, ctx, messages.KitTeardownMsg{Source: "scheduler.ts"})
	tdCh := make(chan struct{}, 1)
	tunsub, _ := sdk.SubscribeTo[messages.KitTeardownResp](k, ctx, tpr.ReplyTo, func(_ messages.KitTeardownResp, _ messages.Message) { tdCh <- struct{}{} })
	<-tdCh
	tunsub()
	time.Sleep(100 * time.Millisecond)

	assert.Empty(t, k.ListSchedules(), "teardown should cancel all schedules for the deployment")
}

func TestSchedule_DrainSkipsFiring(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, _ := k.SubscribeRaw(ctx, "test.drain", func(msg messages.Message) {
		count.Add(1)
	})
	defer unsub()

	k.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 100ms",
		Topic:      "test.drain",
		Payload:    json.RawMessage(`{}`),
		Source:     "test",
	})

	time.Sleep(350 * time.Millisecond)
	countBefore := count.Load()
	assert.GreaterOrEqual(t, countBefore, int32(2))

	k.Kernel.SetDraining(true)
	time.Sleep(400 * time.Millisecond)

	assert.Equal(t, countBefore, count.Load(), "schedule should not fire during drain")
	k.Kernel.Close()
}
