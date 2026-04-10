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
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E01: Deploy + teardown same source simultaneously
func testConcurrencyDeployTeardownRace(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit

	var wg sync.WaitGroup
	var deployErrs, teardownErrs atomic.Int64

	for i := 0; i < 10; i++ {
		wg.Add(2)
		source := fmt.Sprintf("race-stress-%d.ts", i)

		go func() {
			defer wg.Done()
			err := testutil.DeployErr(tk, source, `output("race-stress");`)
			if err != nil {
				deployErrs.Add(1)
			}
		}()

		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			// Teardown via bus — non-fatal
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			pr, err := sdk.Publish(tk, ctx, sdk.KitTeardownMsg{Source: source})
			if err == nil {
				ch := make(chan struct{}, 1)
				unsub, _ := sdk.SubscribeTo[sdk.KitTeardownResp](tk, ctx, pr.ReplyTo, func(_ sdk.KitTeardownResp, _ sdk.Message) {
					ch <- struct{}{}
				})
				select {
				case <-ch:
				case <-ctx.Done():
				}
				if unsub != nil {
					unsub()
				}
			}
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

	tk := env.Kit

	testutil.Deploy(t, tk, "pubsub-stress-race.ts", `
		bus.on("ping", function(msg) { msg.reply({ pong: true }); });
	`)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testutil.EvalTSErr(tk, "__stress_race_pub.ts", `
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

	tk := env.Kit

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		val := fmt.Sprintf("value-%d", i)

		go func() {
			defer wg.Done()
			testutil.EvalTSErr(tk, "__stress_secret_set.ts", fmt.Sprintf(`
				try {
					__go_brainkit_request("secrets.set", JSON.stringify({name: "stress-race-key", value: %q}));
				} catch(e) {}
				return "ok";
			`, val))
		}()

		go func() {
			defer wg.Done()
			testutil.EvalTSErr(tk, "__stress_secret_get.ts", `
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

	tk := env.Kit

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
			if err := testutil.DeployErr(tk, src, code); err == nil {
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
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			pr, err := sdk.Publish(tk, ctx, sdk.KitTeardownMsg{Source: src})
			if err != nil {
				return
			}
			ch := make(chan bool, 1)
			unsub, _ := sdk.SubscribeTo[sdk.KitTeardownResp](tk, ctx, pr.ReplyTo, func(r sdk.KitTeardownResp, _ sdk.Message) {
				ch <- r.Error == ""
			})
			select {
			case ok := <-ch:
				if ok {
					tornDown.Add(1)
				}
			case <-ctx.Done():
			}
			if unsub != nil {
				unsub()
			}
		}()
	}
	wg.Wait()
	t.Logf("torn down: %d/%d", tornDown.Load(), n)

	deps := testutil.ListDeployments(t, tk)
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

	tk := env.Kit
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sctx, scancel := context.WithTimeout(ctx, 5*time.Second)
			defer scancel()

			// Schedule via bus
			pr, err := sdk.Publish(tk, sctx, sdk.ScheduleCreateMsg{
				Expression: "in 1h",
				Topic:      "stress.race.topic",
				Payload:    json.RawMessage(`{}`),
			})
			if err != nil {
				return
			}
			ch := make(chan string, 1)
			unsub, _ := sdk.SubscribeTo[sdk.ScheduleCreateResp](tk, sctx, pr.ReplyTo, func(r sdk.ScheduleCreateResp, _ sdk.Message) {
				ch <- r.ID
			})
			var id string
			select {
			case id = <-ch:
			case <-sctx.Done():
			}
			if unsub != nil {
				unsub()
			}

			if id != "" {
				// Cancel the schedule via bus
				sdk.Publish(tk, sctx, sdk.ScheduleCancelMsg{ID: id})
			}
		}()
	}
	wg.Wait()

	// Verify all cancelled — list schedules via bus
	lctx, lcancel := context.WithTimeout(ctx, 5*time.Second)
	defer lcancel()
	pr, err := sdk.Publish(tk, lctx, sdk.ScheduleListMsg{})
	if err == nil {
		ch := make(chan []sdk.ScheduleInfo, 1)
		unsub, _ := sdk.SubscribeTo[sdk.ScheduleListResp](tk, lctx, pr.ReplyTo, func(r sdk.ScheduleListResp, _ sdk.Message) {
			ch <- r.Schedules
		})
		select {
		case scheds := <-ch:
			assert.Empty(t, scheds, "all schedules should be cancelled")
		case <-lctx.Done():
		}
		if unsub != nil {
			unsub()
		}
	}
}

// E07: kit.Close() while handlers are active
func testConcurrencyCloseDuringHandlers(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	// Uses a fresh kit (not the shared env) because we close it.
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: t.TempDir(),
	})
	require.NoError(t, err)

	testutil.Deploy(t, k, "slow-stress-handler.ts", `
		bus.on("slow", async function(msg) {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({ done: true });
		});
	`)

	k.PublishRaw(context.Background(), "ts.slow-stress-handler.slow", json.RawMessage(`{}`))

	time.Sleep(50 * time.Millisecond)
	err = k.Close()
	assert.NoError(t, err)
}

// E08: EvalTS from 5 goroutines simultaneously
func testConcurrencyParallelEvalTS(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit

	var wg sync.WaitGroup
	results := make([]string, 5)
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			results[i], errs[i] = testutil.EvalTSErr(tk, fmt.Sprintf("__stress_parallel_%d.ts", i),
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

	tk := env.Kit
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			pr, err := sdk.Publish(tk, sctx, sdk.StorageAddMsg{
				Name:   "stress-race-store",
				Type:   "memory",
				Config: json.RawMessage(`{}`),
			})
			if err == nil {
				ch := make(chan struct{}, 1)
				unsub, _ := sdk.SubscribeTo[sdk.StorageAddResp](tk, sctx, pr.ReplyTo, func(_ sdk.StorageAddResp, _ sdk.Message) {
					ch <- struct{}{}
				})
				select {
				case <-ch:
				case <-sctx.Done():
				}
				if unsub != nil {
					unsub()
				}
			}
		}()
		go func() {
			defer wg.Done()
			sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			pr, err := sdk.Publish(tk, sctx, sdk.StorageRemoveMsg{Name: "stress-race-store"})
			if err == nil {
				ch := make(chan struct{}, 1)
				unsub, _ := sdk.SubscribeTo[sdk.StorageRemoveResp](tk, sctx, pr.ReplyTo, func(_ sdk.StorageRemoveResp, _ sdk.Message) {
					ch <- struct{}{}
				})
				select {
				case <-ch:
				case <-sctx.Done():
				}
				if unsub != nil {
					unsub()
				}
			}
		}()
	}
	wg.Wait()
}

// E10: Metrics during heavy deploy/teardown churn
func testConcurrencyMetricsDuringChurn(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit
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
				testutil.DeployErr(tk, src, `output("churn-stress");`)
				// Teardown via bus — fire and forget
				sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				sdk.Publish(tk, sctx, sdk.KitTeardownMsg{Source: src})
				cancel()
				i++
			}
		}
	}()

	// Query metrics via bus repeatedly
	for i := 0; i < 50; i++ {
		mctx, mcancel := context.WithTimeout(ctx, 2*time.Second)
		pr, err := sdk.Publish(tk, mctx, sdk.MetricsGetMsg{})
		if err == nil {
			ch := make(chan sdk.MetricsGetResp, 1)
			unsub, _ := sdk.SubscribeTo[sdk.MetricsGetResp](tk, mctx, pr.ReplyTo, func(r sdk.MetricsGetResp, _ sdk.Message) {
				ch <- r
			})
			select {
			case m := <-ch:
				assert.NotNil(t, m.Metrics)
			case <-mctx.Done():
			}
			if unsub != nil {
				unsub()
			}
		}
		mcancel()
	}

	close(stop)
	wg.Wait()
}

// E11: Two Kits sharing same SQLite store file
func testConcurrencySharedSQLiteStore(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "stress-shared.db")

	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "stress-kit1", CallerID: "stress-kit1", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)
	defer k1.Close()

	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "stress-kit2", CallerID: "stress-kit2", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			testutil.DeployErr(k1, fmt.Sprintf("stress-k1-%d.ts", n), `output("stress-k1");`)
		}(i)
		go func(n int) {
			defer wg.Done()
			testutil.DeployErr(k2, fmt.Sprintf("stress-k2-%d.ts", n), `output("stress-k2");`)
		}(i)
	}
	wg.Wait()

	// Both kits should still be alive — verify via a simple publish
	_, err = k1.PublishRaw(context.Background(), "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err, "k1 should be alive")
	_, err = k2.PublishRaw(context.Background(), "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err, "k2 should be alive")
}

// E12: Deploy during restorePersistedDeployments
func testConcurrencyDeployDuringRestore(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "stress-store.db")

	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		testutil.DeployErr(k1, fmt.Sprintf("stress-restore-%d.ts", i), fmt.Sprintf(`output("stress-restore-%d");`, i))
	}
	k1.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	for i := 0; i < 3; i++ {
		testutil.DeployErr(k2, fmt.Sprintf("stress-new-%d.ts", i), fmt.Sprintf(`output("stress-new-%d");`, i))
	}

	// Verify alive via a simple publish
	_, err = k2.PublishRaw(context.Background(), "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err, "k2 should be alive")
}

// E04: RBAC assign + checkPermission simultaneously — RBAC removed, test is a no-op.
func testConcurrencyRBACAssignCheckRace(t *testing.T, env *suite.TestEnv) {
	t.Skip("RBAC has been removed")
}
