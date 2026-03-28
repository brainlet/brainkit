package infra_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdown_DrainsBeforeClose(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy a handler that takes 500ms
	pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "slow.ts",
		Code: `bus.on("slow", async (msg) => {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	// Subscribe to reply
	replyCh := make(chan struct{}, 1)
	sendPR, _ := sdk.SendToService(k, ctx, "slow.ts", "slow", map[string]bool{"x": true})
	replyUnsub, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		replyCh <- struct{}{}
	})
	defer replyUnsub()

	// Wait for reply — this proves handler completes normally
	select {
	case <-replyCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not reply")
	}

	// Now Shutdown with drain — should be instant since handler already completed
	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := k.Shutdown(shutCtx)
	require.NoError(t, err)
	elapsed := time.Since(start)

	// Drain should complete instantly (no active handlers)
	assert.Less(t, elapsed, 1*time.Second, "drain should be instant when no handlers active")
}

func TestShutdown_DrainTimeout_ForcesClose(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "stuck.ts",
		Code: `bus.on("stuck", async (msg) => {
			await new Promise(r => setTimeout(r, 10000));
			msg.reply({ done: true });
		});`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	sdk.SendToService(k, ctx, "stuck.ts", "stuck", map[string]bool{"x": true})
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	shutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	k.Shutdown(shutCtx)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 2*time.Second, "should force-close after drain timeout")
}

func TestShutdown_CloseStillWorks(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := k.EvalTS(ctx, "__test.ts", `return "alive"`)
	require.NoError(t, err)
	assert.Equal(t, "alive", result)

	err = k.Close()
	require.NoError(t, err)
}

func TestShutdown_MessagesDroppedDuringDrain(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "dropper.ts",
		Code:   `bus.on("ping", (msg) => { msg.reply({ got: true }); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	// Start draining
	k.Kernel.SetDraining(true)

	// Send message — should be dropped
	sendPR, _ := sdk.SendToService(k, ctx, "dropper.ts", "ping", map[string]bool{"x": true})
	var replied atomic.Bool
	replyUnsub, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		replied.Store(true)
	})
	defer replyUnsub()

	time.Sleep(500 * time.Millisecond)
	assert.False(t, replied.Load(), "message should be dropped during drain")
	k.Kernel.Close()
}

func TestShutdown_EvalTSWorksDuringDrain(t *testing.T) {
	k := testutil.NewTestKernelFull(t)

	k.Kernel.SetDraining(true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := k.EvalTS(ctx, "__test.ts", `return "works"`)
	require.NoError(t, err)
	assert.Equal(t, "works", result)

	k.Kernel.Close()
}
