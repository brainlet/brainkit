package adversarial_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E01: Deploy + teardown same source simultaneously
func TestConcurrency_DeployTeardownRace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	var deployErrs, teardownErrs atomic.Int64

	for i := 0; i < 10; i++ {
		wg.Add(2)
		source := fmt.Sprintf("race-%d.ts", i)

		go func() {
			defer wg.Done()
			_, err := tk.Deploy(ctx, source, `output("race");`)
			if err != nil {
				deployErrs.Add(1)
			}
		}()

		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond) // slight delay
			tk.Teardown(ctx, source)
			teardownErrs.Add(1) // teardown is always "successful" (idempotent)
		}()
	}

	wg.Wait()
	// No panics = pass. Some deploys may fail (source exists) — that's fine.
	t.Logf("deploy errors: %d, teardowns: %d", deployErrs.Load(), teardownErrs.Load())
}

// E02: Publish + unsubscribe on same topic simultaneously
func TestConcurrency_PublishUnsubscribeRace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "pubsub-race.ts", `
		bus.on("ping", function(msg) { msg.reply({ pong: true }); });
	`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tk.EvalTS(ctx, "__race_pub.ts", `
				try { bus.publish("ts.pubsub-race.ping", {}); } catch(e) {}
				return "ok";
			`)
		}()
	}
	wg.Wait()
	// No panics, no deadlocks = pass
}

// E03: Secret set + get on same key simultaneously
func TestConcurrency_SecretSetGetRace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		val := fmt.Sprintf("value-%d", i)

		go func() {
			defer wg.Done()
			tk.EvalTS(ctx, "__secret_set.ts", fmt.Sprintf(`
				try {
					__go_brainkit_request("secrets.set", JSON.stringify({name: "race-key", value: %q}));
				} catch(e) {}
				return "ok";
			`, val))
		}()

		go func() {
			defer wg.Done()
			tk.EvalTS(ctx, "__secret_get.ts", `
				try { secrets.get("race-key"); } catch(e) {}
				return "ok";
			`)
		}()
	}
	wg.Wait()
	// No panics, no deadlocks = pass
}

// E05: Deploy 10 services simultaneously, teardown all simultaneously
func TestConcurrency_MassDeployTeardown(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	n := 10
	sources := make([]string, n)
	for i := 0; i < n; i++ {
		sources[i] = fmt.Sprintf("mass-%d.ts", i)
	}

	// Deploy all simultaneously
	var wg sync.WaitGroup
	var deployed atomic.Int64
	for _, src := range sources {
		wg.Add(1)
		src := src
		go func() {
			defer wg.Done()
			code := fmt.Sprintf(`output("deployed %s");`, src)
			if _, err := tk.Deploy(ctx, src, code); err == nil {
				deployed.Add(1)
			}
		}()
	}
	wg.Wait()
	t.Logf("deployed: %d/%d", deployed.Load(), n)

	// Teardown all simultaneously
	var tornDown atomic.Int64
	for _, src := range sources {
		wg.Add(1)
		src := src
		go func() {
			defer wg.Done()
			if _, err := tk.Teardown(ctx, src); err == nil {
				tornDown.Add(1)
			}
		}()
	}
	wg.Wait()
	t.Logf("torn down: %d/%d", tornDown.Load(), n)

	// Verify all cleaned up
	deps := tk.ListDeployments()
	for _, d := range deps {
		for _, src := range sources {
			assert.NotEqual(t, src, d.Source, "deployment %s should be torn down", src)
		}
	}
}

// E06: Schedule + unschedule same ID simultaneously
func TestConcurrency_ScheduleUnscheduleRace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := tk.Schedule(ctx, brainkit.ScheduleConfig{
				Expression: "in 1h",
				Topic:      "race.topic",
				Payload:    json.RawMessage(`{}`),
			})
			if err == nil {
				tk.Unschedule(ctx, id)
			}
		}()
	}
	wg.Wait()
	// No panics, no leaked timers
	scheds := tk.ListSchedules()
	assert.Empty(t, scheds, "all schedules should be cancelled")
}

// E07: kernel.Close() while handlers are active
func TestConcurrency_CloseDuringHandlers(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = k.Deploy(ctx, "slow-handler.ts", `
		bus.on("slow", async function(msg) {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});
	`)
	require.NoError(t, err)

	// Fire a message to the slow handler
	k.PublishRaw(ctx, "ts.slow-handler.slow", json.RawMessage(`{}`))

	// Immediately close — should drain gracefully
	time.Sleep(50 * time.Millisecond) // let the handler start
	err = k.Close()
	assert.NoError(t, err)
}

// E08: EvalTS from 5 goroutines simultaneously
func TestConcurrency_ParallelEvalTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([]string, 5)
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			results[i], errs[i] = tk.EvalTS(ctx, fmt.Sprintf("__parallel_%d.ts", i),
				fmt.Sprintf(`return "result-%d";`, i))
		}()
	}
	wg.Wait()

	// All should succeed — EvalTS serializes via bridge mutex
	for i := 0; i < 5; i++ {
		require.NoError(t, errs[i], "goroutine %d failed", i)
		assert.Equal(t, fmt.Sprintf("result-%d", i), results[i])
	}
}

// E09: AddStorage + RemoveStorage on same name simultaneously
func TestConcurrency_StorageAddRemoveRace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()
	_ = ctx

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tk.AddStorage("race-store", brainkit.InMemoryStorage())
		}()
		go func() {
			defer wg.Done()
			tk.RemoveStorage("race-store")
		}()
	}
	wg.Wait()
	// No panics, no deadlocks
}

// E10: Kernel.Metrics() during heavy deploy/teardown churn
func TestConcurrency_MetricsDuringChurn(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup

	// Background: deploy and teardown rapidly
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				src := fmt.Sprintf("churn-%d.ts", i)
				tk.Deploy(ctx, src, `output("churn");`)
				tk.Teardown(ctx, src)
				i++
			}
		}
	}()

	// Foreground: call Metrics() repeatedly
	for i := 0; i < 50; i++ {
		m := tk.Metrics()
		assert.GreaterOrEqual(t, m.PumpCycles, int64(0))
	}

	close(stop)
	wg.Wait()
}

// E11: Two Kernels sharing same SQLite store file
func TestConcurrency_SharedSQLiteStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "shared.db")

	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "kit1", CallerID: "kit1", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)
	defer k1.Close()

	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "kit2", CallerID: "kit2", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Both kernels deploy and schedule simultaneously on same store
	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			k1.Deploy(ctx, fmt.Sprintf("k1-%d.ts", n), `output("k1");`)
		}(i)
		go func(n int) {
			defer wg.Done()
			k2.Deploy(ctx, fmt.Sprintf("k2-%d.ts", n), `output("k2");`)
		}(i)
	}
	wg.Wait()

	// Both kernels should be alive — SQLite WAL handles concurrent access
	assert.True(t, k1.Alive(ctx))
	assert.True(t, k2.Alive(ctx))
}

// E12: Deploy during restorePersistedDeployments
func TestConcurrency_DeployDuringRestore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	// Phase 1: Persist several deployments
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		k1.Deploy(context.Background(), fmt.Sprintf("restore-%d.ts", i), fmt.Sprintf(`output("restore-%d");`, i))
	}
	k1.Close()

	// Phase 2: Reopen — while persisted deployments are restoring, also deploy new ones
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	// Immediately deploy new ones (restore may still be in progress on the JS thread)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := k2.Deploy(ctx, fmt.Sprintf("new-%d.ts", i), fmt.Sprintf(`output("new-%d");`, i))
		// May succeed or fail with AlreadyExists — both are fine, no panic
		_ = err
	}

	// Kernel should be healthy
	assert.True(t, k2.Alive(ctx))
}

// E04: RBAC assign + checkPermission simultaneously
func TestConcurrency_RBACAssignCheckRace(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":   rbac.RoleAdmin,
			"service": rbac.RoleService,
		},
		DefaultRole: "service",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	_, err = k.Deploy(ctx, "rbac-race.ts", `
		bus.on("ping", function(msg) { msg.reply({ ok: true }); });
	`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	// Rapidly assign/revoke roles while publishing
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			// Toggle role between admin and service
			sdk.Publish(k, ctx, messages.RBACAssignMsg{Source: "rbac-race.ts", Role: "admin"})
			sdk.Publish(k, ctx, messages.RBACRevokeMsg{Source: "rbac-race.ts"})
		}()
		go func() {
			defer wg.Done()
			k.EvalTS(ctx, "__rbac_race.ts", `
				try { bus.publish("ts.rbac-race.ping", {}); } catch(e) {}
				return "ok";
			`)
		}()
	}
	wg.Wait()
	// No panics, no deadlocks = pass
}
