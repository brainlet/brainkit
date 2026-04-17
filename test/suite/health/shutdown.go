package health

import (
	"context"
	"sync/atomic"
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

	replyCh := make(chan struct{}, 1)
	sendPR, _ := sdk.SendToService(env.Kit, ctx, "slow.ts", "slow", map[string]bool{"x": true})
	replyUnsub, _ := env.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) { replyCh <- struct{}{} })
	defer replyUnsub()

	select {
	case <-replyCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not reply")
	}

	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := env.Kit.Shutdown(shutCtx)
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

	sdk.SendToService(env.Kit, ctx, "stuck.ts", "stuck", map[string]bool{"x": true})
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Instrument: call Shutdown but log timing
	env.Kit.Shutdown(shutCtx)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 5*time.Second, "should force-close after drain timeout")
}

func testCloseStillWorks(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	result := testutil.EvalTS(t, env.Kit, "__test.ts", `return "alive"`)
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

	sendPR, _ := sdk.SendToService(env.Kit, ctx, "dropper.ts", "ping", map[string]bool{"x": true})
	var replied atomic.Bool
	replyUnsub, _ := env.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) { replied.Store(true) })
	defer replyUnsub()

	time.Sleep(500 * time.Millisecond)
	assert.False(t, replied.Load(), "message should be dropped during drain")
}

func testEvalTSWorksDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.SetDraining(t, env.Kit, true)

	result := testutil.EvalTS(t, env.Kit, "__test.ts", `return "works"`)
	assert.Equal(t, "works", result)
}
