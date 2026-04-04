package stress

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
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E01: Deploy + teardown same source simultaneously
func testConcurrencyDeployTeardownRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	var wg sync.WaitGroup
	var deployErrs, teardownErrs atomic.Int64

	for i := 0; i < 10; i++ {
		wg.Add(2)
		source := fmt.Sprintf("race-stress-%d.ts", i)

		go func() {
			defer wg.Done()
			_, err := tk.Deploy(ctx, source, `output("race-stress");`)
			if err != nil {
				deployErrs.Add(1)
			}
		}()

		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			tk.Teardown(ctx, source)
			teardownErrs.Add(1)
		}()
	}

	wg.Wait()
	t.Logf("deploy errors: %d, teardowns: %d", deployErrs.Load(), teardownErrs.Load())
}

// E02: Publish + unsubscribe on same topic simultaneously
func testConcurrencyPublishUnsubscribeRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "pubsub-stress-race.ts", `
		bus.on("ping", function(msg) { msg.reply({ pong: true }); });
	`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tk.EvalTS(ctx, "__stress_race_pub.ts", `
				try { bus.publish("ts.pubsub-stress-race.ping", {}); } catch(e) {}
				return "ok";
			`)
		}()
	}
	wg.Wait()
}

// E03: Secret set + get on same key simultaneously
func testConcurrencySecretSetGetRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		val := fmt.Sprintf("value-%d", i)

		go func() {
			defer wg.Done()
			tk.EvalTS(ctx, "__stress_secret_set.ts", fmt.Sprintf(`
				try {
					__go_brainkit_request("secrets.set", JSON.stringify({name: "stress-race-key", value: %q}));
				} catch(e) {}
				return "ok";
			`, val))
		}()

		go func() {
			defer wg.Done()
			tk.EvalTS(ctx, "__stress_secret_get.ts", `
				try { secrets.get("stress-race-key"); } catch(e) {}
				return "ok";
			`)
		}()
	}
	wg.Wait()
}

// E05: Deploy 10 services simultaneously, teardown all simultaneously
func testConcurrencyMassDeployTeardown(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	n := 10
	sources := make([]string, n)
	for i := 0; i < n; i++ {
		sources[i] = fmt.Sprintf("mass-stress-%d.ts", i)
	}

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

	deps := tk.ListDeployments()
	for _, d := range deps {
		for _, src := range sources {
			assert.NotEqual(t, src, d.Source, "deployment %s should be torn down", src)
		}
	}
}

// E06: Schedule + unschedule same ID simultaneously
func testConcurrencyScheduleUnscheduleRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := tk.Schedule(ctx, brainkit.ScheduleConfig{
				Expression: "in 1h",
				Topic:      "stress.race.topic",
				Payload:    json.RawMessage(`{}`),
			})
			if err == nil {
				tk.Unschedule(ctx, id)
			}
		}()
	}
	wg.Wait()
	scheds := tk.ListSchedules()
	assert.Empty(t, scheds, "all schedules should be cancelled")
}

// E07: kernel.Close() while handlers are active
func testConcurrencyCloseDuringHandlers(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	// Uses a fresh kernel (not the shared env) because we close it.
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = k.Deploy(ctx, "slow-stress-handler.ts", `
		bus.on("slow", async function(msg) {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});
	`)
	require.NoError(t, err)

	k.PublishRaw(ctx, "ts.slow-stress-handler.slow", json.RawMessage(`{}`))

	time.Sleep(50 * time.Millisecond)
	err = k.Close()
	assert.NoError(t, err)
}

// E08: EvalTS from 5 goroutines simultaneously
func testConcurrencyParallelEvalTS(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([]string, 5)
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			results[i], errs[i] = tk.EvalTS(ctx, fmt.Sprintf("__stress_parallel_%d.ts", i),
				fmt.Sprintf(`return "result-%d";`, i))
		}()
	}
	wg.Wait()

	for i := 0; i < 5; i++ {
		require.NoError(t, errs[i], "goroutine %d failed", i)
		assert.Equal(t, fmt.Sprintf("result-%d", i), results[i])
	}
}

// E09: AddStorage + RemoveStorage on same name simultaneously
func testConcurrencyStorageAddRemoveRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tk.AddStorage("stress-race-store", brainkit.InMemoryStorage())
		}()
		go func() {
			defer wg.Done()
			tk.RemoveStorage("stress-race-store")
		}()
	}
	wg.Wait()
}

// E10: Kernel.Metrics() during heavy deploy/teardown churn
func testConcurrencyMetricsDuringChurn(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	var wg sync.WaitGroup

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
				src := fmt.Sprintf("churn-stress-%d.ts", i)
				tk.Deploy(ctx, src, `output("churn-stress");`)
				tk.Teardown(ctx, src)
				i++
			}
		}
	}()

	for i := 0; i < 50; i++ {
		m := tk.Metrics()
		assert.GreaterOrEqual(t, m.PumpCycles, int64(0))
	}

	close(stop)
	wg.Wait()
}

// E11: Two Kernels sharing same SQLite store file
func testConcurrencySharedSQLiteStore(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "stress-shared.db")

	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-kit1", CallerID: "stress-kit1", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)
	defer k1.Close()

	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-kit2", CallerID: "stress-kit2", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			k1.Deploy(ctx, fmt.Sprintf("stress-k1-%d.ts", n), `output("stress-k1");`)
		}(i)
		go func(n int) {
			defer wg.Done()
			k2.Deploy(ctx, fmt.Sprintf("stress-k2-%d.ts", n), `output("stress-k2");`)
		}(i)
	}
	wg.Wait()

	assert.True(t, k1.Alive(ctx))
	assert.True(t, k2.Alive(ctx))
}

// E12: Deploy during restorePersistedDeployments
func testConcurrencyDeployDuringRestore(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "stress-store.db")

	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		k1.Deploy(context.Background(), fmt.Sprintf("stress-restore-%d.ts", i), fmt.Sprintf(`output("stress-restore-%d");`, i))
	}
	k1.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := k2.Deploy(ctx, fmt.Sprintf("stress-new-%d.ts", i), fmt.Sprintf(`output("stress-new-%d");`, i))
		_ = err
	}

	assert.True(t, k2.Alive(ctx))
}

// E04: RBAC assign + checkPermission simultaneously
func testConcurrencyRBACAssignCheckRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	// Uses a fresh kernel with RBAC config.
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":   rbac.RoleAdmin,
			"service": rbac.RoleService,
		},
		DefaultRole: "service",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	_, err = k.Deploy(ctx, "rbac-stress-race.ts", `
		bus.on("ping", function(msg) { msg.reply({ ok: true }); });
	`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			sdk.Publish(k, ctx, messages.RBACAssignMsg{Source: "rbac-stress-race.ts", Role: "admin"})
			sdk.Publish(k, ctx, messages.RBACRevokeMsg{Source: "rbac-stress-race.ts"})
		}()
		go func() {
			defer wg.Done()
			k.EvalTS(ctx, "__stress_rbac_race.ts", `
				try { bus.publish("ts.rbac-stress-race.ping", {}); } catch(e) {}
				return "ok";
			`)
		}()
	}
	wg.Wait()
}

