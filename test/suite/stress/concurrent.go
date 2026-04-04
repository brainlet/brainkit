package stress

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testParallelDeploy(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	testutil.ConcurrentDo(t, 10, func(i int) {
		source := fmt.Sprintf("svc-stress-%d.ts", i)
		code := fmt.Sprintf(`bus.on("ping", (msg) => msg.reply({ id: %d }));`, i)
		_, err := k.Deploy(ctx, source, code)
		require.NoError(t, err, "deploy %s failed", source)
	})

	deployments := k.ListDeployments()
	require.GreaterOrEqual(t, len(deployments), 10, "all 10 deploys should succeed")
}

func testParallelPublish(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "echo-stress.ts", `
		bus.on("echo", (msg) => {
			msg.reply({ echoed: msg.payload.id });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	results := make([]bool, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		sendPR, err := sdk.SendToService(k, ctx, "echo-stress.ts", "echo", map[string]int{"id": i})
		if err != nil {
			t.Errorf("goroutine %d: publish failed: %v", i, err)
			return
		}
		msg := testutil.WaitForBusMessage(t, k, sendPR.ReplyTo, 10*time.Second)
		if len(msg.Payload) > 0 {
			results[i] = true
		}
	})

	for i, got := range results {
		assert.True(t, got, "goroutine %d did not get a response", i)
	}
}

func testParallelEvalTS(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	results := make([]string, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		code := fmt.Sprintf(`return JSON.stringify({ id: %d });`, i)
		result, err := k.EvalTS(ctx, fmt.Sprintf("eval-stress-%d.ts", i), code)
		if err != nil {
			t.Errorf("EvalTS %d failed: %v", i, err)
			return
		}
		results[i] = result
	})

	for i, r := range results {
		expected := fmt.Sprintf(`{"id":%d}`, i)
		assert.Equal(t, expected, r, "goroutine %d got wrong result", i)
	}
}

func testDeployDuringHandler(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "slow-stress.ts", `
		bus.on("slow", async (msg) => {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sdk.SendToService(k, ctx, "slow-stress.ts", "slow", map[string]bool{"go": true})

	done := make(chan error, 1)
	go func() {
		_, err := k.Deploy(ctx, "fast-stress.ts", `bus.on("fast", (msg) => msg.reply({}));`)
		done <- err
	}()

	select {
	case err := <-done:
		require.NoError(t, err, "deploy during active handler should not deadlock")
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: deploy blocked by active handler")
	}
}

func testTeardownDuringHandler(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "teardown-stress-target.ts", `
		bus.on("work", async (msg) => {
			await new Promise(r => setTimeout(r, 300));
			msg.reply({ done: true });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sdk.SendToService(k, ctx, "teardown-stress-target.ts", "work", map[string]bool{"go": true})
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{}, 1)
	go func() {
		k.Teardown(ctx, "teardown-stress-target.ts")
		done <- struct{}{}
	}()

	select {
	case <-done:
		// Teardown completed
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: teardown blocked by active handler")
	}
}

func testDeployTeardownRaceOnSameSource(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "race-stress-target.ts", `bus.on("ping", (msg) => msg.reply({ ok: true }));`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	errs := make(chan error, 2)

	go func() {
		_, tErr := k.Teardown(ctx, "race-stress-target.ts")
		errs <- tErr
	}()
	go func() {
		_, rErr := k.Redeploy(ctx, "race-stress-target.ts", `bus.on("ping", (msg) => msg.reply({ v: 2 }));`)
		errs <- rErr
	}()

	for i := 0; i < 2; i++ {
		select {
		case e := <-errs:
			_ = e
		case <-time.After(15 * time.Second):
			t.Fatalf("deadlock on operation %d: deploy/teardown race did not resolve", i)
		}
	}
}

func testStressDeployTeardownCycles(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	testutil.ConcurrentDo(t, 5, func(i int) {
		for cycle := 0; cycle < 3; cycle++ {
			source := fmt.Sprintf("stress-cycle-%d.ts", i)
			code := fmt.Sprintf(`bus.on("ping", (msg) => msg.reply({ id: %d, cycle: %d }));`, i, cycle)

			resources, deployErr := k.Deploy(ctx, source, code)
			if deployErr != nil {
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

	deployments := k.ListDeployments()
	// Some may remain from concurrent timing -- just verify no crash
	_ = deployments
}

func testRedeployRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "redeploy-stress-race.ts", `bus.on("v", (msg) => msg.reply({ version: 0 }));`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(version int) {
			code := fmt.Sprintf(`bus.on("v", (msg) => msg.reply({ version: %d }));`, version)
			_, rErr := k.Redeploy(ctx, "redeploy-stress-race.ts", code)
			errs <- rErr
		}(i + 1)
	}

	for i := 0; i < 3; i++ {
		select {
		case e := <-errs:
			_ = e
		case <-time.After(30 * time.Second):
			t.Fatal("deadlock: concurrent redeploy did not resolve")
		}
	}

	deployments := k.ListDeployments()
	assert.GreaterOrEqual(t, len(deployments), 1, "at least one deployment should survive")
}

func testDeployDuringDrain(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kernel
	ctx := context.Background()

	k.SetDraining(true)

	resources, err := k.Deploy(ctx, "drain-stress-deploy.ts", `bus.on("ping", (msg) => msg.reply({ ok: true }));`)
	if err != nil {
		t.Logf("deploy during drain returned error (acceptable): %v", err)
	} else {
		assert.NotNil(t, resources)
		pr, pubErr := sdk.SendToService(k, ctx, "drain-stress-deploy.ts", "ping", map[string]bool{"go": true})
		if pubErr == nil {
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
				t.Log("handler replied despite drain")
			case <-replyCtx.Done():
				// Expected: handler rejected by drain
			}
		}
	}

	k.SetDraining(false)
}
