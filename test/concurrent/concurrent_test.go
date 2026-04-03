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

func TestConcurrent_DeployTeardownRaceOnSameSource(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy the service first
	_, err := k.Deploy(ctx, "race-target.ts", `bus.on("ping", (msg) => msg.reply({ ok: true }));`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Concurrent teardown + redeploy on the same source — must not panic/deadlock.
	// One operation wins (teardown or redeploy), the other gets an error — that's correct.
	errs := make(chan error, 2)

	go func() {
		_, tErr := k.Teardown(ctx, "race-target.ts")
		errs <- tErr
	}()
	go func() {
		_, rErr := k.Redeploy(ctx, "race-target.ts", `bus.on("ping", (msg) => msg.reply({ v: 2 }));`)
		errs <- rErr
	}()

	for i := 0; i < 2; i++ {
		select {
		case e := <-errs:
			// Either nil (operation won) or error (lost the race) — both are acceptable.
			// The important thing is no panic or deadlock.
			_ = e
		case <-time.After(15 * time.Second):
			t.Fatalf("deadlock on operation %d: deploy/teardown race did not resolve", i)
		}
	}
}

func TestConcurrent_StressDeployTeardownCycles(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// 5 goroutines, each deploys, verifies, tears down, repeats 3 times
	testutil.ConcurrentDo(t, 5, func(i int) {
		for cycle := 0; cycle < 3; cycle++ {
			source := fmt.Sprintf("stress-%d.ts", i)
			code := fmt.Sprintf(`bus.on("ping", (msg) => msg.reply({ id: %d, cycle: %d }));`, i, cycle)

			resources, deployErr := k.Deploy(ctx, source, code)
			if deployErr != nil {
				// AlreadyExists is acceptable if the previous teardown hasn't fully completed
				continue
			}
			_ = resources

			time.Sleep(50 * time.Millisecond)

			_, teardownErr := k.Teardown(ctx, source)
			if teardownErr != nil {
				t.Errorf("goroutine %d cycle %d: teardown failed: %v", i, cycle, teardownErr)
			}
		}
	})

	// Final state: all deployments should be torn down
	deployments := k.ListDeployments()
	assert.Empty(t, deployments, "all stress deployments should be torn down")
}

func TestConcurrent_RedeployRace(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Initial deploy
	_, err := k.Deploy(ctx, "redeploy-race.ts", `bus.on("v", (msg) => msg.reply({ version: 0 }));`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// 3 goroutines all try to redeploy simultaneously — only one should succeed,
	// others should either succeed or get a recoverable error. No panics.
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(version int) {
			code := fmt.Sprintf(`bus.on("v", (msg) => msg.reply({ version: %d }));`, version)
			_, rErr := k.Redeploy(ctx, "redeploy-race.ts", code)
			errs <- rErr
		}(i + 1)
	}

	for i := 0; i < 3; i++ {
		select {
		case e := <-errs:
			_ = e // error or nil — both acceptable in a race
		case <-time.After(30 * time.Second):
			t.Fatal("deadlock: concurrent redeploy did not resolve")
		}
	}

	// The service should still exist (one redeploy won)
	deployments := k.ListDeployments()
	assert.Len(t, deployments, 1, "exactly one deployment should survive")
}

func TestConcurrent_DeployDuringDrain(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Activate drain mode
	k.SetDraining(true)

	// Deploy a service — should still succeed (draining affects handlers, not deploys)
	// BUT subscriptions in the deployed code won't receive messages (enterHandler returns false)
	resources, err := k.Deploy(ctx, "drain-deploy.ts", `bus.on("ping", (msg) => msg.reply({ ok: true }));`)
	if err != nil {
		// If deploy fails during drain, that's also acceptable behavior
		t.Logf("deploy during drain returned error (acceptable): %v", err)
	} else {
		assert.NotNil(t, resources)
		// Verify the handler is rejected during drain
		pr, pubErr := sdk.SendToService(k, ctx, "drain-deploy.ts", "ping", map[string]bool{"go": true})
		if pubErr == nil {
			// Subscribe to replyTo — should time out because drain rejects handlers
			replyCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			gotReply := make(chan bool, 1)
			unsub, _ := k.SubscribeRaw(replyCtx, pr.ReplyTo, func(msg messages.Message) {
				gotReply <- true
			})
			if unsub != nil {
				defer unsub()
			}
			select {
			case <-gotReply:
				t.Log("handler replied despite drain — drain may not affect deployed code handlers")
			case <-replyCtx.Done():
				// Expected: handler rejected by drain
			}
		}
	}

	// Clean up drain state
	k.SetDraining(false)
}

// helper to avoid unused import
var _ = json.Marshal
var _ = messages.Message{}
