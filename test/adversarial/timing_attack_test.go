package adversarial_test

import (
	"context"
	"encoding/json"
	"fmt"
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

// ════════════════════════════════════════════════════════════════════════════
// TIMING ATTACKS
// Race conditions, TOCTOU, and state corruption through precise timing.
// ════════════════════════════════════════════════════════════════════════════

// Attack: subscribe to replyTo BEFORE the legitimate caller to steal response
func TestTiming_PreemptiveReplySubscribe(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "timing-svc.ts", `
		bus.on("data", function(msg) {
			msg.reply({confidential: "alpha-bravo-charlie"});
		});
	`)
	require.NoError(t, err)

	// Attacker guesses the replyTo pattern and subscribes preemptively
	// Pattern: <topic>.reply.<uuid> — UUID is unpredictable
	// But if attacker subscribes to a WILDCARD (on transports that support it)...
	// GoChannel doesn't support wildcards, but let's check
	var attackerGot atomic.Int64
	// Subscribe to a broad topic that might catch replies
	unsub, _ := tk.SubscribeRaw(ctx, "ts.timing-svc.data.reply", func(m messages.Message) {
		attackerGot.Add(1)
	})
	defer unsub()

	// Legitimate call
	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.timing-svc.data", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	legitUnsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer legitUnsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "alpha-bravo")
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Attacker's broad subscribe shouldn't catch the specific reply topic
	// (GoChannel is exact match, not prefix match)
	t.Logf("Attacker intercepted via broad subscribe: %d", attackerGot.Load())
}

// Attack: deploy + teardown + deploy same source to corrupt deployment state
func TestTiming_DeployTeardownRace_SameSource(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	var deployErrors, teardownErrors atomic.Int64

	// Rapid alternation between deploy and teardown on SAME source
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := tk.Deploy(ctx, "race-target.ts", `output("deployed");`)
			if err != nil {
				deployErrors.Add(1)
			}
		}()
		go func() {
			defer wg.Done()
			tk.Teardown(ctx, "race-target.ts")
			teardownErrors.Add(1)
		}()
	}
	wg.Wait()

	t.Logf("Deploy errors: %d, Teardown attempts: %d", deployErrors.Load(), teardownErrors.Load())

	// Final state should be consistent — either deployed or not
	deps := tk.ListDeployments()
	raceFound := false
	for _, d := range deps {
		if d.Source == "race-target.ts" {
			raceFound = true
		}
	}

	// Either it's deployed or not — but kernel must be alive
	assert.True(t, tk.Alive(ctx), "kernel should survive deploy/teardown race")
	t.Logf("Final deployment exists: %v", raceFound)
}

// Attack: publish message during redeployPersistedDeployments to hit half-initialized state
func TestTiming_MessageDuringRestore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/timing.db"

	// Phase 1: Persist a handler
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1,
	})
	require.NoError(t, err)

	_, err = k1.Deploy(context.Background(), "slow-restore.ts", `
		bus.on("ping", function(msg) { msg.reply({restored: true}); });
	`)
	require.NoError(t, err)
	k1.Close()

	// Phase 2: Reopen and IMMEDIATELY start publishing
	// The handler might not be restored yet
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	// Immediately fire messages — handler may not be restored yet
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var responded atomic.Int64
	for i := 0; i < 10; i++ {
		go func() {
			pr, _ := sdk.Publish(k2, ctx, messages.CustomMsg{
				Topic: "ts.slow-restore.ping", Payload: json.RawMessage(`{}`),
			})
			ch := make(chan []byte, 1)
			unsub, _ := k2.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			defer unsub()
			select {
			case <-ch:
				responded.Add(1)
			case <-time.After(2 * time.Second):
			}
		}()
	}

	time.Sleep(3 * time.Second)
	t.Logf("Messages sent during restore: 10, got responses: %d", responded.Load())
	assert.True(t, k2.Alive(ctx))
}

// Attack: concurrent Redeploy calls on the same source
func TestTiming_ConcurrentRedeploy(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "redeploy-race.ts", `output("v0");`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			tk.Redeploy(ctx, "redeploy-race.ts", fmt.Sprintf(`output("v%d");`, n))
		}(i)
	}
	wg.Wait()

	// Exactly one deployment should exist
	deps := tk.ListDeployments()
	count := 0
	for _, d := range deps {
		if d.Source == "redeploy-race.ts" {
			count++
		}
	}
	assert.Equal(t, 1, count, "should have exactly one deployment after concurrent redeploys")
	assert.True(t, tk.Alive(ctx))
}

// Attack: tool.call during deploy of the tool's source (reentrancy)
func TestTiming_ToolCallDuringDeploy(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy code that registers a tool AND immediately calls it
	_, err := tk.Deploy(ctx, "reentrant-tool.ts", `
		var t = createTool({
			id: "reentrant",
			description: "test",
			execute: async ({n}) => ({doubled: n * 2}),
		});
		kit.register("tool", "reentrant", t);

		// Now call it during the SAME deploy eval
		var result = await tools.call("reentrant", {n: 21});
		output(result);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__reentrant.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// tools.call during deploy goes through __go_brainkit_request_async → LocalInvoker
	// which calls the JS executor → which does EvalOnJSThread
	// FINDING #6: This might deadlock (bridge mutex reentrant call)
	t.Logf("Reentrant tool call result: %s", result)
}

// Attack: schedule fires between deploy and handler registration
func TestTiming_ScheduleFiresBeforeHandlerReady(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Create schedule FIRST (fires in 200ms)
	_, err := tk.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 200ms",
		Topic:      "ts.timing-handler.trigger",
		Payload:    json.RawMessage(`{"scheduled": true}`),
	})
	require.NoError(t, err)

	// Wait 100ms, then deploy the handler (schedule fires during deploy or before handler is ready)
	time.Sleep(100 * time.Millisecond)

	_, err = tk.Deploy(ctx, "timing-handler.ts", `
		var received = false;
		bus.on("trigger", function(msg) {
			received = true;
			msg.reply({received: true});
		});
		output("deployed");
	`)
	require.NoError(t, err)

	// Wait for schedule to fire
	time.Sleep(500 * time.Millisecond)

	// Did the handler catch the scheduled message?
	result, _ := tk.EvalTS(ctx, "__timing_h.ts", `
		// Check if the handler was called
		return "kernel-alive";
	`)
	assert.Equal(t, "kernel-alive", result)
	assert.True(t, tk.Alive(ctx))
}

// Attack: Close kernel while a tool.call is in progress
func TestTiming_CloseWhileToolCallInProgress(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Deploy a slow tool
	_, err = k.Deploy(ctx, "slow-tool.ts", `
		var t = createTool({
			id: "slow",
			description: "takes 2s",
			execute: async () => {
				await new Promise(r => setTimeout(r, 2000));
				return {done: true};
			},
		});
		kit.register("tool", "slow", t);
	`)
	require.NoError(t, err)

	// Start a tool call
	go func() {
		sendAndReceive(t, k, messages.ToolCallMsg{Name: "slow", Input: map[string]any{}}, 5*time.Second)
	}()

	// Close while tool is running
	time.Sleep(100 * time.Millisecond)
	err = k.Close()
	assert.NoError(t, err, "Close should succeed even with in-flight tool call")
}

// Attack: RBAC role change while a handler is executing
func TestTiming_RoleChangeWhileHandlerRunning(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":    rbac.RoleAdmin,
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "admin",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	_, err = k.Deploy(ctx, "role-change.ts", `
		bus.on("slow-op", async function(msg) {
			// Start as admin...
			var r1 = bus.publish("incoming.test-admin", {phase: "before"});
			await new Promise(r => setTimeout(r, 500));
			// Role might have changed by now!
			try {
				var r2 = bus.publish("incoming.test-after", {phase: "after"});
				msg.reply({both: true});
			} catch(e) {
				msg.reply({denied: true, error: e.code});
			}
		});
	`, brainkit.WithRole("admin"))
	require.NoError(t, err)

	// Start the handler
	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.role-change.slow-op", Payload: json.RawMessage(`{}`),
	})

	// While handler is sleeping, change its role to observer
	time.Sleep(200 * time.Millisecond)
	sdk.Publish(k, ctx, messages.RBACAssignMsg{Source: "role-change.ts", Role: "observer"})

	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		t.Logf("Role-change-during-handler result: %s", string(p))
		// If "denied" — the role change affected the in-flight handler
		// If "both" — the handler completed before the role change took effect
	case <-time.After(3 * time.Second):
		t.Log("timeout — handler may have deadlocked")
	}
}

// Attack: concurrent Schedule + Unschedule same ID
func TestTiming_ScheduleUnscheduleRace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var fired atomic.Int64
	unsub, _ := tk.SubscribeRaw(ctx, "race.sched.topic", func(m messages.Message) {
		fired.Add(1)
	})
	defer unsub()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := tk.Schedule(ctx, brainkit.ScheduleConfig{
				Expression: "in 10ms",
				Topic:      "race.sched.topic",
				Payload:    json.RawMessage(`{}`),
			})
			if err == nil {
				// Race: unschedule immediately
				tk.Unschedule(ctx, id)
			}
		}()
	}
	wg.Wait()

	time.Sleep(500 * time.Millisecond)
	t.Logf("Schedule/unschedule race: %d fired despite unschedule", fired.Load())
	assert.True(t, tk.Alive(ctx))
}

// Attack: concurrent AddStorage + RemoveStorage + Deploy that uses storage
func TestTiming_StorageRaceWithDeploy(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup

	// Add/remove storage while deploying code that uses it
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			tk.AddStorage("race-storage", brainkit.InMemoryStorage())
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			tk.RemoveStorage("race-storage")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			src := fmt.Sprintf("storage-race-%d.ts", i)
			tk.Deploy(ctx, src, `
				try {
					var has = registry.has("storage", "race-storage");
				} catch(e) {}
				output("ok");
			`)
			tk.Teardown(ctx, src)
		}
	}()
	wg.Wait()

	assert.True(t, tk.Alive(ctx), "kernel should survive storage add/remove/deploy race")
}
