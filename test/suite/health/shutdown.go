package health

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDrainsBeforeClose(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	testutil.Deploy(t, env.Kit, "slow.ts", `bus.on("slow", async (msg) => {
		await new Promise(r => setTimeout(r, 500));
		msg.reply({ done: true });
	});`)
	time.Sleep(100 * time.Millisecond)

	_, err := sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   healthServiceTopic("slow.ts", "slow"),
		Payload: json.RawMessage(`{"x":true}`),
	}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)

	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = env.Kit.Shutdown(shutCtx)
	require.NoError(t, err)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 1*time.Second, "drain should be instant when no handlers active")
}

func testDrainTimeoutForcesClose(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	testutil.Deploy(t, env.Kit, "stuck.ts", `bus.on("stuck", async (msg) => {
		await new Promise(r => setTimeout(r, 10000));
		msg.reply({ done: true });
	});`)
	time.Sleep(100 * time.Millisecond)

	callCtx, callCancel := context.WithCancel(ctx)
	defer callCancel()
	callDone := make(chan struct{}, 1)
	go func() {
		defer close(callDone)
		_, _ = sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, callCtx, sdk.CustomMsg{
			Topic:   healthServiceTopic("stuck.ts", "stuck"),
			Payload: json.RawMessage(`{"x":true}`),
		}, sdk.WithCallTimeout(10*time.Second))
	}()
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Instrument: call Shutdown but log timing
	env.Kit.Shutdown(shutCtx)
	callCancel()
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 5*time.Second, "should force-close after drain timeout")
	select {
	case <-callDone:
	case <-time.After(time.Second):
		t.Fatal("stuck call did not unblock after shutdown")
	}
}

func testCloseStillWorks(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	result := testutil.EvalJS(t, env.Kit, "__test.ts", `return "alive"`)
	assert.Equal(t, "alive", result)

	err := env.Kit.Close()
	require.NoError(t, err)
}

func testMessagesDroppedDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	testutil.Deploy(t, env.Kit, "dropper.ts", `bus.on("ping", (msg) => { msg.reply({ got: true }); });`)
	time.Sleep(100 * time.Millisecond)

	testutil.SetDraining(t, env.Kit, true)

	_, err := sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   healthServiceTopic("dropper.ts", "ping"),
		Payload: json.RawMessage(`{"x":true}`),
	}, sdk.WithCallTimeout(500*time.Millisecond))
	assert.Error(t, err, "message should be dropped during drain")
}

func testEvalJSWorksDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.SetDraining(t, env.Kit, true)

	result := testutil.EvalJS(t, env.Kit, "__test.ts", `return "works"`)
	assert.Equal(t, "works", result)
}

func healthServiceTopic(source, topic string) string {
	name := strings.TrimSuffix(source, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
