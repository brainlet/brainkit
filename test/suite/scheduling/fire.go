package scheduling

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEveryFiresRepeatedly(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, err := env.Kernel.SubscribeRaw(ctx, "test.tick", func(msg messages.Message) {
		count.Add(1)
	})
	require.NoError(t, err)
	defer unsub()

	id, err := env.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
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

// testInFiresOnce needs fresh kernel because it asserts ListSchedules is empty.
func testInFiresOnce(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, _ := freshEnv.Kernel.SubscribeRaw(ctx, "test.once", func(msg messages.Message) {
		count.Add(1)
	})
	defer unsub()

	_, err := freshEnv.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 100ms",
		Topic:      "test.once",
		Payload:    json.RawMessage(`{"once":true}`),
		Source:     "test",
	})
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load(), "in 100ms should fire exactly once")
	assert.Empty(t, freshEnv.Kernel.ListSchedules(), "one-time schedule should be removed after firing")
}

func testUnschedule(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, _ := env.Kernel.SubscribeRaw(ctx, "test.cancel", func(msg messages.Message) {
		count.Add(1)
	})
	defer unsub()

	id, _ := env.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 100ms",
		Topic:      "test.cancel",
		Payload:    json.RawMessage(`{}`),
		Source:     "test",
	})

	time.Sleep(250 * time.Millisecond)
	env.Kernel.Unschedule(ctx, id)
	countAtCancel := count.Load()

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, countAtCancel, count.Load(), "unschedule should stop firing")
}

func testInvalidExpression(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	_, err := env.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "cron 0 9 * * *",
		Topic:      "test.invalid",
		Source:     "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schedule expression")
}

// testTeardownCancelsSchedules needs fresh kernel because it asserts schedule count.
func testTeardownCancelsSchedules(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	err := freshEnv.Deploy("sched-teardown.ts", `
		bus.schedule("every 100ms", "tick", {});
	`)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	schedsBefore := freshEnv.Kernel.ListSchedules()
	assert.Greater(t, len(schedsBefore), 0, "should have at least one schedule")

	freshEnv.Kernel.Teardown(ctx, "sched-teardown.ts")
	time.Sleep(200 * time.Millisecond)

	schedsAfter := freshEnv.Kernel.ListSchedules()
	assert.Equal(t, 0, len(schedsAfter), "teardown should cancel all schedules from the deployment")
}

// testE2EScheduleFires — schedule a message, verify handler receives it.
func testE2EScheduleFires(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	// Deploy handler
	_, err := freshEnv.Kernel.Deploy(ctx, "sched-handler-e2e.ts", `
		bus.on("tick", function(msg) {
			msg.reply({ticked: true, payload: msg.payload});
		});
	`)
	require.NoError(t, err)

	// Subscribe to know when schedule fires
	fired := make(chan []byte, 1)
	unsub, _ := freshEnv.Kernel.SubscribeRaw(ctx, "ts.sched-handler-e2e.tick", func(m messages.Message) {
		fired <- m.Payload
	})
	defer unsub()

	// Schedule in 200ms
	id, err := freshEnv.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 200ms",
		Topic:      "ts.sched-handler-e2e.tick",
		Payload:    json.RawMessage(`{"scheduled":true}`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	select {
	case p := <-fired:
		assert.Contains(t, string(p), "scheduled")
	case <-time.After(5 * time.Second):
		t.Fatal("schedule didn't fire within 5s")
	}
}

// testInputAbuseScheduleInvalidExpression — invalid schedule expression should error.
func testInputAbuseScheduleInvalidExpression(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	_, err := freshEnv.Kernel.Schedule(context.Background(), brainkit.ScheduleConfig{
		Expression: "bananas at midnight",
		Topic:      "test",
	})
	assert.Error(t, err)
}

// testInputAbuseScheduleEmptyTopic — empty topic schedule should work or error cleanly.
func testInputAbuseScheduleEmptyTopic(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	id, err := freshEnv.Kernel.Schedule(context.Background(), brainkit.ScheduleConfig{
		Expression: "in 1h",
		Topic:      "",
	})
	// Either succeeds or errors — no panic
	if err == nil {
		freshEnv.Kernel.Unschedule(context.Background(), id)
	}
}

func testDrainSkipsFiring(t *testing.T, env *suite.TestEnv) {
	// Use a fresh kernel since we need to drain it
	freshEnv := suite.Full(t)
	ctx := context.Background()

	var count atomic.Int32
	unsub, _ := freshEnv.Kernel.SubscribeRaw(ctx, "test.drain", func(msg messages.Message) {
		count.Add(1)
	})
	defer unsub()

	freshEnv.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 100ms",
		Topic:      "test.drain",
		Payload:    json.RawMessage(`{}`),
		Source:     "test",
	})

	time.Sleep(250 * time.Millisecond)
	freshEnv.Kernel.SetDraining(true)
	countAtDrain := count.Load()

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, countAtDrain, count.Load(), "schedules should not fire during drain")
}
