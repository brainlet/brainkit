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
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimingPreemptiveReplySubscribe — subscribe to replyTo BEFORE the legitimate caller.
func testTimingPreemptiveReplySubscribe(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secDeploy(t, k, "timing-svc-sec.ts", `
		bus.on("data", function(msg) {
			msg.reply({confidential: "alpha-bravo-charlie"});
		});
	`)

	var attackerGot atomic.Int64
	unsub, _ := k.SubscribeRaw(ctx, "ts.timing-svc-sec.data.reply", func(m sdk.Message) {
		attackerGot.Add(1)
	})
	defer unsub()

	pr, _ := sdk.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.timing-svc-sec.data", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	legitUnsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
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
	k := suite.Full(t).Kit

	var wg sync.WaitGroup
	var deployErrors, teardownErrors atomic.Int64

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			err := secDeployErr(k, "race-target-sec.ts", `output("deployed");`)
			if err != nil {
				deployErrors.Add(1)
			}
		}()
		go func() {
			defer wg.Done()
			secTeardown(t, k, "race-target-sec.ts")
			teardownErrors.Add(1)
		}()
	}
	wg.Wait()

	t.Logf("Deploy errors: %d, Teardown attempts: %d", deployErrors.Load(), teardownErrors.Load())

	deps := secListDeployments(t, k)
	raceFound := false
	for _, d := range deps {
		if d.Source == "race-target-sec.ts" {
			raceFound = true
		}
	}

	assert.True(t, secAlive(t, k), "kit should survive deploy/teardown race")
	t.Logf("Final deployment exists: %v", raceFound)
}

// testTimingMessageDuringRestore — publish message during redeployPersistedDeployments.
func testTimingMessageDuringRestore(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/timing-sec.db"

	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1,
	})
	require.NoError(t, err)

	secDeploy(t, k1, "slow-restore-sec.ts", `
		bus.on("ping", function(msg) { msg.reply({restored: true}); });
	`)
	k1.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.New(brainkit.Config{
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
			pr, _ := sdk.Publish(k2, ctx, sdk.CustomMsg{
				Topic: "ts.slow-restore-sec.ping", Payload: json.RawMessage(`{}`),
			})
			ch := make(chan []byte, 1)
			unsub, _ := k2.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
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
	assert.True(t, secAlive(t, k2))
}

// testTimingConcurrentRedeploy — concurrent Redeploy calls on the same source.
func testTimingConcurrentRedeploy(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "redeploy-race-sec.ts", `output("v0");`)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			secDeployErr(k, "redeploy-race-sec.ts", fmt.Sprintf(`output("v%d");`, n))
		}(i)
	}
	wg.Wait()

	deps := secListDeployments(t, k)
	count := 0
	for _, d := range deps {
		if d.Source == "redeploy-race-sec.ts" {
			count++
		}
	}
	assert.Equal(t, 1, count, "should have exactly one deployment after concurrent redeploys")
	assert.True(t, secAlive(t, k))
}

// testTimingToolCallDuringDeploy — tool.call during deploy of the tool's source (reentrancy).
func testTimingToolCallDuringDeploy(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "reentrant-tool-sec.ts", `
		var t = createTool({
			id: "reentrant-sec",
			description: "test",
			execute: async ({n}) => ({doubled: n * 2}),
		});
		kit.register("tool", "reentrant-sec", t);

		var result = await tools.call("reentrant-sec", {n: 21});
		output(result);
	`)

	result, _ := secEvalTSErr(k, "__reentrant.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Reentrant tool call result: %s", result)
}

// testTimingScheduleFiresBeforeHandlerReady — schedule fires between deploy and handler registration.
func testTimingScheduleFiresBeforeHandlerReady(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	_, err := secSchedule(t, k, brainkit.ScheduleConfig{
		Expression: "in 200ms",
		Topic:      "ts.timing-handler-sec.trigger",
		Payload:    json.RawMessage(`{"scheduled": true}`),
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	secDeploy(t, k, "timing-handler-sec.ts", `
		var received = false;
		bus.on("trigger", function(msg) {
			received = true;
			msg.reply({received: true});
		});
		output("deployed");
	`)

	time.Sleep(500 * time.Millisecond)

	result, _ := secEvalTSErr(k, "__timing_h.ts", `
		return "kernel-alive";
	`)
	assert.Equal(t, "kernel-alive", result)
	assert.True(t, secAlive(t, k))
}

// testTimingCloseWhileToolCallInProgress — Close kit while a tool.call is in progress.
func testTimingCloseWhileToolCallInProgress(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	secDeploy(t, k, "slow-tool-sec.ts", `
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

	go func() {
		secSendAndReceive(t, k, sdk.ToolCallMsg{Name: "slow-sec", Input: map[string]any{}}, 5*time.Second)
	}()

	time.Sleep(100 * time.Millisecond)
	err = k.Close()
	assert.NoError(t, err, "Close should succeed even with in-flight tool call")
}

// testTimingRoleChangeWhileHandlerRunning — RBAC role change while a handler is executing.
func testTimingRoleChangeWhileHandlerRunning(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
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

	require.NoError(t, secDeployWithRole(k, "role-change-sec.ts", `
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
	`, "admin"))

	pr, _ := sdk.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.role-change-sec.slow-op", Payload: json.RawMessage(`{}`),
	})

	time.Sleep(200 * time.Millisecond)
	sdk.Publish(k, ctx, sdk.RBACAssignMsg{Source: "role-change-sec.ts", Role: "observer"})

	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
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
	k := suite.Full(t).Kit
	ctx := context.Background()

	var fired atomic.Int64
	unsub, _ := k.SubscribeRaw(ctx, "race.sched.topic.sec", func(m sdk.Message) {
		fired.Add(1)
	})
	defer unsub()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := secSchedule(t, k, brainkit.ScheduleConfig{
				Expression: "in 10ms",
				Topic:      "race.sched.topic.sec",
				Payload:    json.RawMessage(`{}`),
			})
			if err == nil {
				secUnschedule(t, k, id)
			}
		}()
	}
	wg.Wait()

	time.Sleep(500 * time.Millisecond)
	t.Logf("Schedule/unschedule race: %d fired despite unschedule", fired.Load())
	assert.True(t, secAlive(t, k))
}

// testTimingStorageRaceWithDeploy — concurrent AddStorage + RemoveStorage + Deploy.
func testTimingStorageRaceWithDeploy(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx := context.Background()

	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			sdk.Publish(k, ctx, sdk.StorageAddMsg{
				Name: "race-storage-sec",
				Type: "memory",
			})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			sdk.Publish(k, ctx, sdk.StorageRemoveMsg{
				Name: "race-storage-sec",
			})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			src := fmt.Sprintf("storage-race-sec-%d.ts", i)
			secDeployErr(k, src, `
				try {
					var has = registry.has("storage", "race-storage-sec");
				} catch(e) {}
				output("ok");
			`)
			secTeardown(t, k, src)
		}
	}()
	wg.Wait()

	assert.True(t, secAlive(t, k), "kit should survive storage add/remove/deploy race")
}
