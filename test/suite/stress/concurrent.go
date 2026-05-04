package stress

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testParallelDeploy(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kit
	cleanupStressDeployments(t, k)
	t.Cleanup(func() { cleanupStressDeployments(t, k) })

	testutil.ConcurrentDo(t, 10, func(i int) {
		source := fmt.Sprintf("svc-stress-%d.ts", i)
		code := fmt.Sprintf(`bus.on("ping", (msg) => msg.reply({ id: %d }));`, i)
		err := testutil.DeployErr(k, source, code)
		require.NoError(t, err, "deploy %s failed", source)
	})

	deployments := testutil.ListDeployments(t, k)
	require.Len(t, deployments, 10, "all 10 deploys should succeed")
}

func testParallelPublish(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kit
	ctx := context.Background()
	cleanupStressDeployments(t, k)
	t.Cleanup(func() { cleanupStressDeployments(t, k) })

	testutil.Deploy(t, k, "echo-stress.ts", `
		bus.on("echo", (msg) => {
			msg.reply({ echoed: msg.payload.id });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	results := make([]bool, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		payload, err := callStressService(t, k, ctx, "echo-stress.ts", "echo", json.RawMessage(fmt.Sprintf(`{"id":%d}`, i)), 10*time.Second)
		if err != nil {
			t.Errorf("goroutine %d: call failed: %v", i, err)
			return
		}
		if len(payload) > 0 {
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

	k := env.Kit

	results := make([]string, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		code := fmt.Sprintf(`return JSON.stringify({ id: %d });`, i)
		result, err := testutil.EvalTSErr(k, fmt.Sprintf("eval-stress-%d.ts", i), code)
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

	k := env.Kit
	ctx := context.Background()
	cleanupStressDeployments(t, k)
	t.Cleanup(func() { cleanupStressDeployments(t, k) })

	testutil.Deploy(t, k, "slow-stress.ts", `
		bus.on("slow", async (msg) => {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	go func() {
		_, _ = callStressService(t, k, ctx, "slow-stress.ts", "slow", json.RawMessage(`{"go":true}`), 2*time.Second)
	}()

	done := make(chan error, 1)
	go func() {
		done <- testutil.DeployErr(k, "fast-stress.ts", `bus.on("fast", (msg) => msg.reply({}));`)
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

	k := env.Kit
	ctx := context.Background()

	testutil.Deploy(t, k, "teardown-stress-target.ts", `
		bus.on("work", async (msg) => {
			await new Promise(r => setTimeout(r, 300));
			msg.reply({ done: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	go func() {
		_, _ = callStressService(t, k, ctx, "teardown-stress-target.ts", "work", json.RawMessage(`{"go":true}`), 2*time.Second)
	}()
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{}, 1)
	go func() {
		testutil.Teardown(t, k, "teardown-stress-target.ts")
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

	k := env.Kit
	cleanupStressDeployments(t, k)
	t.Cleanup(func() { cleanupStressDeployments(t, k) })

	testutil.Deploy(t, k, "race-stress-target.ts", `bus.on("ping", (msg) => msg.reply({ ok: true }));`)
	time.Sleep(200 * time.Millisecond)

	errs := make(chan error, 2)

	go func() {
		errs <- stressTeardown(t, k, "race-stress-target.ts")
	}()
	go func() {
		errs <- testutil.DeployErr(k, "race-stress-target.ts", `bus.on("ping", (msg) => msg.reply({ v: 2 }));`)
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

	k := env.Kit
	cleanupStressDeployments(t, k)
	t.Cleanup(func() { cleanupStressDeployments(t, k) })

	testutil.ConcurrentDo(t, 5, func(i int) {
		for cycle := 0; cycle < 3; cycle++ {
			source := fmt.Sprintf("stress-cycle-%d.ts", i)
			code := fmt.Sprintf(`bus.on("ping", (msg) => msg.reply({ id: %d, cycle: %d }));`, i, cycle)

			deployErr := testutil.DeployErr(k, source, code)
			if deployErr != nil {
				continue
			}

			time.Sleep(50 * time.Millisecond)

			if err := stressTeardown(t, k, source); err != nil {
				t.Errorf("goroutine %d cycle %d: teardown failed: %v", i, cycle, err)
			}
		}
	})

	deployments := testutil.ListDeployments(t, k)
	assert.Empty(t, deployments, "all stress deployments should be torn down")
}

func testRedeployRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kit
	cleanupStressDeployments(t, k)
	t.Cleanup(func() { cleanupStressDeployments(t, k) })

	testutil.Deploy(t, k, "redeploy-stress-race.ts", `bus.on("v", (msg) => msg.reply({ version: 0 }));`)
	time.Sleep(200 * time.Millisecond)

	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(version int) {
			code := fmt.Sprintf(`bus.on("v", (msg) => msg.reply({ version: %d }));`, version)
			errs <- testutil.DeployErr(k, "redeploy-stress-race.ts", code)
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

	deployments := testutil.ListDeployments(t, k)
	assert.Len(t, deployments, 1, "exactly one deployment should survive")
}

func testDeployDuringDrain(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k := env.Kit
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.SetDraining(t, k, false)
		cleanupStressDeployments(t, k)
	})

	testutil.SetDraining(t, k, true)

	err := testutil.DeployErr(k, "drain-stress-deploy.ts", `bus.on("ping", (msg) => msg.reply({ ok: true }));`)
	if err != nil {
		t.Logf("deploy during drain returned error (acceptable): %v", err)
	} else {
		if _, callErr := callStressService(t, k, ctx, "drain-stress-deploy.ts", "ping", json.RawMessage(`{"go":true}`), 2*time.Second); callErr == nil {
			t.Log("handler replied despite drain")
		}
	}

	testutil.SetDraining(t, k, false)
}
