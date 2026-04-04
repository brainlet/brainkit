package health

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDrainsBeforeClose(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "slow.ts",
		Code: `bus.on("slow", async (msg) => {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	replyCh := make(chan struct{}, 1)
	sendPR, _ := sdk.SendToService(env.Kernel, ctx, "slow.ts", "slow", map[string]bool{"x": true})
	replyUnsub, _ := env.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) { replyCh <- struct{}{} })
	defer replyUnsub()

	select {
	case <-replyCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not reply")
	}

	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := env.Kernel.Shutdown(shutCtx)
	require.NoError(t, err)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 1*time.Second, "drain should be instant when no handlers active")
}

func testDrainTimeoutForcesClose(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "stuck.ts",
		Code: `bus.on("stuck", async (msg) => {
			await new Promise(r => setTimeout(r, 10000));
			msg.reply({ done: true });
		});`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	sdk.SendToService(env.Kernel, ctx, "stuck.ts", "stuck", map[string]bool{"x": true})
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	env.Kernel.Shutdown(shutCtx)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 2*time.Second, "should force-close after drain timeout")
}

func testCloseStillWorks(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `return "alive"`)
	require.NoError(t, err)
	assert.Equal(t, "alive", result)

	err = env.Kernel.Close()
	require.NoError(t, err)
}

func testMessagesDroppedDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "dropper.ts",
		Code:   `bus.on("ping", (msg) => { msg.reply({ got: true }); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	env.Kernel.SetDraining(true)

	sendPR, _ := sdk.SendToService(env.Kernel, ctx, "dropper.ts", "ping", map[string]bool{"x": true})
	var replied atomic.Bool
	replyUnsub, _ := env.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) { replied.Store(true) })
	defer replyUnsub()

	time.Sleep(500 * time.Millisecond)
	assert.False(t, replied.Load(), "message should be dropped during drain")
}

func testEvalTSWorksDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	env.Kernel.SetDraining(true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `return "works"`)
	require.NoError(t, err)
	assert.Equal(t, "works", result)
}
