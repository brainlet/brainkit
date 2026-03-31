package concurrent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrent_ParallelDeploy(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	testutil.ConcurrentDo(t, 10, func(i int) {
		source := fmt.Sprintf("svc-%d.ts", i)
		code := fmt.Sprintf(`bus.on("ping", (msg) => msg.reply({ id: %d }));`, i)
		_, err := k.Deploy(ctx, source, code)
		require.NoError(t, err, "deploy %s failed", source)
	})

	// Verify all deployed
	deployments := k.ListDeployments()
	require.Len(t, deployments, 10, "all 10 deploys should succeed")
}

func TestConcurrent_ParallelPublish(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy an echo handler
	_, err := k.Deploy(ctx, "echo.ts", `
		bus.on("echo", (msg) => {
			msg.reply({ echoed: msg.payload.id });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// 10 goroutines each publish and wait for reply
	results := make([]bool, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		sendPR, err := sdk.SendToService(k, ctx, "echo.ts", "echo", map[string]int{"id": i})
		if err != nil {
			t.Errorf("goroutine %d: publish failed: %v", i, err)
			return
		}
		msg := testutil.WaitForBusMessage(t, k.Kernel, sendPR.ReplyTo, 10*time.Second)
		if len(msg.Payload) > 0 {
			results[i] = true
		}
	})

	for i, got := range results {
		assert.True(t, got, "goroutine %d did not get a response", i)
	}
}

func TestConcurrent_ParallelEvalTS(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	results := make([]string, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		code := fmt.Sprintf(`return JSON.stringify({ id: %d });`, i)
		result, err := k.EvalTS(ctx, fmt.Sprintf("eval-%d.ts", i), code)
		if err != nil {
			t.Errorf("EvalTS %d failed: %v", i, err)
			return
		}
		results[i] = result
	})

	// Each goroutine should get its own result (correct isolation)
	for i, r := range results {
		expected := fmt.Sprintf(`{"id":%d}`, i)
		assert.Equal(t, expected, r, "goroutine %d got wrong result", i)
	}
}

func TestConcurrent_DeployDuringHandler(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy a slow handler
	_, err := k.Deploy(ctx, "slow.ts", `
		bus.on("slow", async (msg) => {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Trigger the slow handler
	sdk.SendToService(k, ctx, "slow.ts", "slow", map[string]bool{"go": true})

	// Deploy another service while handler is running — should not deadlock
	done := make(chan error, 1)
	go func() {
		_, err := k.Deploy(ctx, "fast.ts", `bus.on("fast", (msg) => msg.reply({}));`)
		done <- err
	}()

	select {
	case err := <-done:
		require.NoError(t, err, "deploy during active handler should not deadlock")
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: deploy blocked by active handler")
	}
}

func TestConcurrent_TeardownDuringHandler(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "teardown-target.ts", `
		bus.on("work", async (msg) => {
			await new Promise(r => setTimeout(r, 300));
			msg.reply({ done: true });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Trigger handler
	sdk.SendToService(k, ctx, "teardown-target.ts", "work", map[string]bool{"go": true})
	time.Sleep(50 * time.Millisecond) // let handler start

	// Teardown while handler is running — should not crash or deadlock
	done := make(chan struct{}, 1)
	go func() {
		k.Teardown(ctx, "teardown-target.ts")
		done <- struct{}{}
	}()

	select {
	case <-done:
		// Teardown completed — no deadlock, no panic
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: teardown blocked by active handler")
	}
}

// helper to avoid unused import
var _ = json.Marshal
var _ = messages.Message{}
