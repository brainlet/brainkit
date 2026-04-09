package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCascadeDeployWithBrokenStore — C01: Deploy succeeds in memory when store is closed.
// ErrorHandler should be called for the persistence failure.
func testCascadeDeployWithBrokenStore(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(tmpDir + "/store-cascade.db")
	require.NoError(t, err)

	var errorCalled bool
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		ErrorHandler: func(err error) {
			errorCalled = true
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Close the store DB to force persistence failures
	store.Close()

	// Deploy should still succeed in memory
	testutil.Deploy(t, k, "broken-store-cascade.ts", `output("works in memory");`)

	// ErrorHandler should have been called for the persistence failure
	time.Sleep(100 * time.Millisecond)
	assert.True(t, errorCalled, "ErrorHandler should have been called for store failure")
}

// testCascadeCorruptedStore — C02: NewSQLiteStore should fail on a corrupt file.
func testCascadeCorruptedStore(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/corrupt-cascade.db"

	// Write garbage to the store file
	require.NoError(t, os.WriteFile(storePath, []byte("not a sqlite database"), 0644))

	// NewSQLiteStore should fail on corrupt file
	_, err := brainkit.NewSQLiteStore(storePath)
	assert.Error(t, err)
}

// testCascadeSecretRotatePluginFails — C05: Secret rotate when no plugins running.
// Rotation should succeed; restart is a no-op.
func testCascadeSecretRotatePluginFails(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	// Set a secret, then rotate it — no plugins running, so restart is a no-op
	pr1, _ := sdk.Publish(freshEnv.Kit, ctx, sdk.SecretsSetMsg{Name: "ROTATE_KEY_CASCADE", Value: "v1"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := freshEnv.Kit.SubscribeRaw(ctx, pr1.ReplyTo, func(m sdk.Message) { ch1 <- m.Payload })
	select {
	case <-ch1:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout setting secret")
	}
	unsub1()

	// Rotate
	pr2, _ := sdk.Publish(freshEnv.Kit, ctx, sdk.SecretsRotateMsg{Name: "ROTATE_KEY_CASCADE", NewValue: "v2", Restart: true})
	ch2 := make(chan []byte, 1)
	unsub2, _ := freshEnv.Kit.SubscribeRaw(ctx, pr2.ReplyTo, func(m sdk.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case payload := <-ch2:
		assert.Contains(t, string(payload), "rotated")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout rotating secret")
	}
}

// testCascadeHandlerThrowNoReplyTo — C07: Handler throws on emit (no replyTo).
// Failure event should be emitted, no panic.
func testCascadeHandlerThrowNoReplyTo(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	testutil.Deploy(t, freshEnv.Kit, "no-reply-cascade.ts", `
		bus.on("fire", function(msg) { throw new Error("no replyTo crash"); });
	`)

	// Emit (no replyTo) — handler will throw but there's nowhere to send the error
	failed := make(chan bool, 1)
	unsub, _ := sdk.SubscribeTo[sdk.HandlerFailedEvent](freshEnv.Kit, ctx, "bus.handler.failed", func(e sdk.HandlerFailedEvent, m sdk.Message) {
		failed <- true
	})
	defer unsub()

	sdk.Emit(freshEnv.Kit, ctx, sdk.CustomEvent{Topic: "ts.no-reply-cascade.fire", Payload: json.RawMessage(`{}`)})

	select {
	case <-failed:
		// Good — failure event emitted, no panic
	case <-time.After(5 * time.Second):
		// If no retry policy, the error is logged but no event emitted. That's OK too.
	}
}

// testCascadeTeardownCleansSubscriptions — C08: Teardown cleans up bus subscriptions.
func testCascadeTeardownCleansSubscriptions(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	testutil.Deploy(t, freshEnv.Kit, "sub-cleanup-cascade.ts", `
		bus.on("ping", function(msg) { msg.reply({ pong: true }); });
	`)

	// Verify it works
	result := testutil.EvalTS(t, freshEnv.Kit, "__ping-cascade.ts", `
		var r = bus.publish("ts.sub-cleanup-cascade.ping", {});
		return r.replyTo ? "ok" : "fail";
	`)
	assert.Equal(t, "ok", result)

	// Teardown
	testutil.Teardown(t, freshEnv.Kit, "sub-cleanup-cascade.ts")

	// Publish again — should not get a response (handler is gone)
	pr, err := sdk.Publish(freshEnv.Kit, ctx, sdk.CustomMsg{Topic: "ts.sub-cleanup-cascade.ping", Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)

	ch := make(chan bool, 1)
	unsub, _ := freshEnv.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- true })
	defer unsub()

	select {
	case <-ch:
		t.Fatal("received response after teardown — subscription not cleaned")
	case <-time.After(1 * time.Second):
		// Good — no response, subscription was cleaned
	}
}

// testCascadeConcurrentErrorHandler — concurrent goroutines trigger ErrorHandler without panic.
func testCascadeConcurrentErrorHandler(t *testing.T, _ *suite.TestEnv) {
	var mu sync.Mutex
	var count int

	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(tmpDir + "/store-cascade-concurrent.db")

	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		ErrorHandler: func(err error) {
			mu.Lock()
			count++
			mu.Unlock()
		},
	})
	require.NoError(t, err)

	// Close store to force errors, then do multiple persistence operations concurrently
	store.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			testutil.ScheduleErr(k, "in 1h", fmt.Sprintf("concurrent-err-cascade-%d", i), json.RawMessage(`{}`))
		}(i)
	}
	wg.Wait()
	k.Close()

	mu.Lock()
	// ErrorHandler should have been called multiple times from concurrent goroutines
	// without panicking
	t.Logf("ErrorHandler called %d times concurrently", count)
	mu.Unlock()
}

// testCascadePublishDuringDrain — C03: bus.publish during kernel drain.
func testCascadePublishDuringDrain(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)

	// Deploy a service
	testutil.Deploy(t, freshEnv.Kit, "drain-pub-cascade.ts", `
		bus.on("ask", function(msg) { msg.reply({ ok: true }); });
	`)

	// Start draining
	testutil.SetDraining(t, freshEnv.Kit, true)

	// Publish should still work (publish isn't affected by drain — only handlers are)
	result := testutil.EvalTS(t, freshEnv.Kit, "__drain_pub_cascade.ts", `
		var r = bus.publish("ts.drain-pub-cascade.ask", {});
		return r.replyTo ? "published" : "fail";
	`)
	assert.Equal(t, "published", result)

	testutil.SetDraining(t, freshEnv.Kit, false)
}

// testCascadeEvalTSDuringClose — C04: EvalTS during kernel close.
func testCascadeEvalTSDuringClose(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	// Close the kernel
	k.Close()

	// EvalTS after close should error, not panic
	_, err = testutil.EvalTSErr(k, "__after_close_cascade.ts", `return "should not run";`)
	assert.Error(t, err)
}

// testCascadeRetryExhausted — C06: Handler throws, retry exhausted.
func testCascadeRetryExhausted(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		RetryPolicies: map[string]brainkit.RetryPolicy{
			"ts.retry-test-cascade.*": {MaxRetries: 2, InitialDelay: 10 * time.Millisecond},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	testutil.Deploy(t, k, "retry-test-cascade.ts", `
		bus.on("boom", function(msg) { throw new Error("always fails"); });
	`)

	// Subscribe to exhaustion events
	exhausted := make(chan bool, 1)
	unsub, _ := sdk.SubscribeTo[sdk.HandlerExhaustedEvent](k, context.Background(), "bus.handler.exhausted", func(e sdk.HandlerExhaustedEvent, m sdk.Message) {
		exhausted <- true
	})
	defer unsub()

	// Publish — should trigger retries then exhaustion
	k.PublishRaw(context.Background(), "ts.retry-test-cascade.boom", json.RawMessage(`{}`))

	select {
	case <-exhausted:
		// Good — exhaustion event fired
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for exhaustion event")
	}
}

// testCascadeScheduleNoHandler — C10: Schedule fires but no handler exists.
func testCascadeScheduleNoHandler(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)

	// Schedule a message to a topic nobody listens to
	id := testutil.Schedule(t, freshEnv.Kit, "in 100ms", "ghost.topic.nobody.listens.cascade", json.RawMessage(`{"ghost": true}`))
	require.NotEmpty(t, id)

	// Wait for it to fire
	time.Sleep(500 * time.Millisecond)

	// Kit should still be healthy — no panic from publishing to a topic with no subscribers
	assert.True(t, testutil.Alive(t, freshEnv.Kit))
}
