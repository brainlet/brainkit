package security

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimingPreemptiveReplySubscribe — subscribe to replyTo BEFORE the legitimate caller.
func testTimingPreemptiveReplySubscribe(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := k.Deploy(ctx, "timing-svc-sec.ts", `
		bus.on("data", function(msg) {
			msg.reply({confidential: "alpha-bravo-charlie"});
		});
	`)
	require.NoError(t, err)

	var attackerGot atomic.Int64
	unsub, _ := k.SubscribeRaw(ctx, "ts.timing-svc-sec.data.reply", func(m messages.Message) {
		attackerGot.Add(1)
	})
	defer unsub()

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.timing-svc-sec.data", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	legitUnsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer legitUnsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "alpha-bravo")
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	t.Logf("Attacker intercepted via broad subscribe: %d", attackerGot.Load())
}

// testTimingDeployTeardownRace — deploy + teardown + deploy same source to corrupt deployment state.
func testTimingDeployTeardownRace(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx := context.Background()

	var wg sync.WaitGroup
	var deployErrors, teardownErrors atomic.Int64

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := k.Deploy(ctx, "race-target-sec.ts", `output("deployed");`)
			if err != nil {
				deployErrors.Add(1)
			}
		}()
		go func() {
			defer wg.Done()
			k.Teardown(ctx, "race-target-sec.ts")
			teardownErrors.Add(1)
		}()
	}
	wg.Wait()

	t.Logf("Deploy errors: %d, Teardown attempts: %d", deployErrors.Load(), teardownErrors.Load())

	deps := k.ListDeployments()
	raceFound := false
	for _, d := range deps {
		if d.Source == "race-target-sec.ts" {
			raceFound = true
		}
	}

	assert.True(t, k.Alive(ctx), "kernel should survive deploy/teardown race")
	t.Logf("Final deployment exists: %v", raceFound)
}

// testTimingMessageDuringRestore — publish message during redeployPersistedDeployments.
func testTimingMessageDuringRestore(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/timing-sec.db"

	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1,
	})
	require.NoError(t, err)

	_, err = k1.Deploy(context.Background(), "slow-restore-sec.ts", `
		bus.on("ping", function(msg) { msg.reply({restored: true}); });
	`)
	require.NoError(t, err)
	k1.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var responded atomic.Int64
	for i := 0; i < 10; i++ {
		go func() {
			pr, _ := sdk.Publish(k2, ctx, messages.CustomMsg{
				Topic: "ts.slow-restore-sec.ping", Payload: json.RawMessage(`{}`),
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

// testTimingConcurrentRedeploy — concurrent Redeploy calls on the same source.
func testTimingConcurrentRedeploy(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "redeploy-race-sec.ts", `output("v0");`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			k.Deploy(ctx, "redeploy-race-sec.ts", fmt.Sprintf(`output("v%d");`, n))
		}(i)
	}
	wg.Wait()

	deps := k.ListDeployments()
	count := 0
	for _, d := range deps {
		if d.Source == "redeploy-race-sec.ts" {
			count++
		}
	}
	assert.Equal(t, 1, count, "should have exactly one deployment after concurrent redeploys")
	assert.True(t, k.Alive(ctx))
}

// testTimingToolCallDuringDeploy — tool.call during deploy of the tool's source (reentrancy).
func testTimingToolCallDuringDeploy(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "reentrant-tool-sec.ts", `
		var t = createTool({
			id: "reentrant-sec",
			description: "test",
			execute: async ({n}) => ({doubled: n * 2}),
		});
		kit.register("tool", "reentrant-sec", t);

		var result = await tools.call("reentrant-sec", {n: 21});
		output(result);
	`)
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__reentrant.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Reentrant tool call result: %s", result)
}

// testTimingScheduleFiresBeforeHandlerReady — schedule fires between deploy and handler registration.
func testTimingScheduleFiresBeforeHandlerReady(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx := context.Background()

	_, err := k.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 200ms",
		Topic:      "ts.timing-handler-sec.trigger",
		Payload:    json.RawMessage(`{"scheduled": true}`),
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	_, err = k.Deploy(ctx, "timing-handler-sec.ts", `
		var received = false;
		bus.on("trigger", function(msg) {
			received = true;
			msg.reply({received: true});
		});
		output("deployed");
	`)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	result, _ := k.EvalTS(ctx, "__timing_h.ts", `
		return "kernel-alive";
	`)
	assert.Equal(t, "kernel-alive", result)
	assert.True(t, k.Alive(ctx))
}

// testTimingCloseWhileToolCallInProgress — Close kernel while a tool.call is in progress.
func testTimingCloseWhileToolCallInProgress(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	ctx := context.Background()

	_, err = k.Deploy(ctx, "slow-tool-sec.ts", `
		var t = createTool({
			id: "slow-sec",
			description: "takes 2s",
			execute: async () => {
				await new Promise(r => setTimeout(r, 2000));
				return {done: true};
			},
		});
		kit.register("tool", "slow-sec", t);
	`)
	require.NoError(t, err)

	go func() {
		secSendAndReceive(t, k, messages.ToolCallMsg{Name: "slow-sec", Input: map[string]any{}}, 5*time.Second)
	}()

	time.Sleep(100 * time.Millisecond)
	err = k.Close()
	assert.NoError(t, err, "Close should succeed even with in-flight tool call")
}

// testTimingRoleChangeWhileHandlerRunning — RBAC role change while a handler is executing.
func testTimingRoleChangeWhileHandlerRunning(t *testing.T, env *suite.TestEnv) {
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

	_, err = k.Deploy(ctx, "role-change-sec.ts", `
		bus.on("slow-op", async function(msg) {
			var r1 = bus.publish("incoming.test-admin-sec", {phase: "before"});
			await new Promise(r => setTimeout(r, 500));
			try {
				var r2 = bus.publish("incoming.test-after-sec", {phase: "after"});
				msg.reply({both: true});
			} catch(e) {
				msg.reply({denied: true, error: e.code});
			}
		});
	`, brainkit.WithRole("admin"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.role-change-sec.slow-op", Payload: json.RawMessage(`{}`),
	})

	time.Sleep(200 * time.Millisecond)
	sdk.Publish(k, ctx, messages.RBACAssignMsg{Source: "role-change-sec.ts", Role: "observer"})

	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		t.Logf("Role-change-during-handler result: %s", string(p))
	case <-time.After(3 * time.Second):
		t.Log("timeout — handler may have deadlocked")
	}
}

// testTimingScheduleUnscheduleRace — concurrent Schedule + Unschedule same ID.
func testTimingScheduleUnscheduleRace(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx := context.Background()

	var fired atomic.Int64
	unsub, _ := k.SubscribeRaw(ctx, "race.sched.topic.sec", func(m messages.Message) {
		fired.Add(1)
	})
	defer unsub()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := k.Schedule(ctx, brainkit.ScheduleConfig{
				Expression: "in 10ms",
				Topic:      "race.sched.topic.sec",
				Payload:    json.RawMessage(`{}`),
			})
			if err == nil {
				k.Unschedule(ctx, id)
			}
		}()
	}
	wg.Wait()

	time.Sleep(500 * time.Millisecond)
	t.Logf("Schedule/unschedule race: %d fired despite unschedule", fired.Load())
	assert.True(t, k.Alive(ctx))
}

// testTimingStorageRaceWithDeploy — concurrent AddStorage + RemoveStorage + Deploy.
func testTimingStorageRaceWithDeploy(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx := context.Background()

	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			k.AddStorage("race-storage-sec", brainkit.InMemoryStorage())
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			k.RemoveStorage("race-storage-sec")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			src := fmt.Sprintf("storage-race-sec-%d.ts", i)
			k.Deploy(ctx, src, `
				try {
					var has = registry.has("storage", "race-storage-sec");
				} catch(e) {}
				output("ok");
			`)
			k.Teardown(ctx, src)
		}
	}()
	wg.Wait()

	assert.True(t, k.Alive(ctx), "kernel should survive storage add/remove/deploy race")
}
