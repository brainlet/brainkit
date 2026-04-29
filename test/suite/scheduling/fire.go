package scheduling

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEveryFiresRepeatedly(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, err := env.Kit.SubscribeRaw(ctx, "test.tick", func(msg sdk.Message) {
		count.Add(1)
	})
	require.NoError(t, err)
	defer unsub()

	id := testutil.Schedule(t, env.Kit, "every 200ms", "test.tick", json.RawMessage(`{"tick":true}`))
	assert.NotEmpty(t, id)

	time.Sleep(700 * time.Millisecond)
	got := count.Load()
	assert.GreaterOrEqual(t, got, int32(3), "every 200ms should fire 3+ times in 700ms, got %d", got)
}

// testInFiresOnce needs fresh kernel because it asserts schedule count.
func testInFiresOnce(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, _ := freshEnv.Kit.SubscribeRaw(ctx, "test.once", func(msg sdk.Message) {
		count.Add(1)
	})
	defer unsub()

	testutil.Schedule(t, freshEnv.Kit, "in 100ms", "test.once", json.RawMessage(`{"once":true}`))

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load(), "in 100ms should fire exactly once")

	assert.Empty(t, testutil.ListSchedules(t, freshEnv.Kit), "one-time schedule should be removed after firing")
}

func testUnschedule(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count atomic.Int32
	unsub, _ := env.Kit.SubscribeRaw(ctx, "test.cancel", func(msg sdk.Message) {
		count.Add(1)
	})
	defer unsub()

	id := testutil.Schedule(t, env.Kit, "every 100ms", "test.cancel", json.RawMessage(`{}`))

	time.Sleep(250 * time.Millisecond)

	testutil.Unschedule(t, env.Kit, id)

	countAtCancel := count.Load()

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, countAtCancel, count.Load(), "unschedule should stop firing")
}

func testInvalidExpression(t *testing.T, env *suite.TestEnv) {
	_, err := testutil.ScheduleErr(env.Kit, "cron 0 9 * * *", "test.invalid", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schedule expression")
}

// testTeardownCancelsSchedules needs fresh kernel because it asserts schedule count.
func testTeardownCancelsSchedules(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)

	err := freshEnv.Deploy("sched-teardown.ts", `
		bus.schedule("every 100ms", "tick", {});
	`)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	assert.Greater(t, len(testutil.ListSchedules(t, freshEnv.Kit)), 0, "should have at least one schedule")

	testutil.Teardown(t, freshEnv.Kit, "sched-teardown.ts")
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, len(testutil.ListSchedules(t, freshEnv.Kit)), "teardown should cancel all schedules from the deployment")
}

// testE2EScheduleFires — schedule a message, verify handler receives it.
func testE2EScheduleFires(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	// Deploy handler
	testutil.Deploy(t, freshEnv.Kit, "sched-handler-e2e.ts", `
		bus.on("tick", function(msg) {
			msg.reply({ticked: true, payload: msg.payload});
		});
	`)

	// Subscribe to know when schedule fires
	fired := make(chan []byte, 1)
	unsub, _ := freshEnv.Kit.SubscribeRaw(ctx, "ts.sched-handler-e2e.tick", func(m sdk.Message) {
		fired <- m.Payload
	})
	defer unsub()

	// Schedule in 200ms
	id := testutil.Schedule(t, freshEnv.Kit, "in 200ms", "ts.sched-handler-e2e.tick", json.RawMessage(`{"scheduled":true}`))
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
	_, err := testutil.ScheduleErr(freshEnv.Kit, "bananas at midnight", "test", nil)
	assert.Error(t, err)
}

// testInputAbuseScheduleEmptyTopic — empty topic schedule should work or error cleanly.
func testInputAbuseScheduleEmptyTopic(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	id, err := testutil.ScheduleErr(freshEnv.Kit, "in 1h", "", nil)
	// Either succeeds or errors — no panic
	if err == nil {
		testutil.Unschedule(t, freshEnv.Kit, id)
	}
}

func testDrainSkipsFiring(t *testing.T, env *suite.TestEnv) {
	// Use a fresh kernel since we need to drain it
	freshEnv := suite.Full(t)
	ctx := context.Background()

	var count atomic.Int32
	unsub, _ := freshEnv.Kit.SubscribeRaw(ctx, "test.drain", func(msg sdk.Message) {
		count.Add(1)
	})
	defer unsub()

	testutil.Schedule(t, freshEnv.Kit, "every 100ms", "test.drain", json.RawMessage(`{}`))

	time.Sleep(250 * time.Millisecond)
	testutil.SetDraining(t, freshEnv.Kit, true)
	countAtDrain := count.Load()

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, countAtDrain, count.Load(), "schedules should not fire during drain")
}
