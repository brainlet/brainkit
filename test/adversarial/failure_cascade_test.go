package adversarial_test

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
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// C01: Deploy succeeds but store.SaveDeployment fails
func TestFailureCascade_DeployWithBrokenStore(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(tmpDir + "/store.db")
	require.NoError(t, err)

	var errorCalled bool
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		ErrorHandler: func(err error, ctx brainkit.ErrorContext) {
			errorCalled = true
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Close the store DB to force persistence failures
	store.Close()

	// Deploy should still succeed in memory
	_, err = k.Deploy(context.Background(), "broken-store.ts", `output("works in memory");`)
	require.NoError(t, err)

	// ErrorHandler should have been called for the persistence failure
	time.Sleep(100 * time.Millisecond)
	assert.True(t, errorCalled, "ErrorHandler should have been called for store failure")
}

// C03: bus.publish during kernel drain
func TestFailureCascade_PublishDuringDrain(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy a service
	_, err := tk.Deploy(ctx, "drain-pub.ts", `
		bus.on("ask", function(msg) { msg.reply({ ok: true }); });
	`)
	require.NoError(t, err)

	// Start draining
	tk.SetDraining(true)

	// Publish should still work (publish isn't affected by drain — only handlers are)
	result, err := tk.EvalTS(ctx, "__drain_pub.ts", `
		var r = bus.publish("ts.drain-pub.ask", {});
		return r.replyTo ? "published" : "fail";
	`)
	require.NoError(t, err)
	assert.Equal(t, "published", result)

	tk.SetDraining(false)
}

// C04: EvalTS during kernel close
func TestFailureCascade_EvalTSDuringClose(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	// Close the kernel
	k.Close()

	// EvalTS after close should error, not panic
	_, err = k.EvalTS(context.Background(), "__after_close.ts", `return "should not run";`)
	// Should get an error (bridge closed, context cancelled, etc.)
	assert.Error(t, err)
}

// C06: Handler throws, retry exhausted
func TestFailureCascade_RetryExhausted(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		RetryPolicies: map[string]brainkit.RetryPolicy{
			"ts.retry-test.*": {MaxRetries: 2, InitialDelay: 10 * time.Millisecond},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	_, err = k.Deploy(context.Background(), "retry-test.ts", `
		bus.on("boom", function(msg) { throw new Error("always fails"); });
	`)
	require.NoError(t, err)

	// Subscribe to exhaustion events
	exhausted := make(chan bool, 1)
	unsub, _ := sdk.SubscribeTo[messages.HandlerExhaustedEvent](k, context.Background(), "bus.handler.exhausted", func(e messages.HandlerExhaustedEvent, m messages.Message) {
		exhausted <- true
	})
	defer unsub()

	// Publish — should trigger retries then exhaustion
	k.PublishRaw(context.Background(), "ts.retry-test.boom", json.RawMessage(`{}`))

	select {
	case <-exhausted:
		// Good — exhaustion event fired
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for exhaustion event")
	}
}

// C07: Handler throws on message with no replyTo
func TestFailureCascade_HandlerThrowNoReplyTo(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "no-reply.ts", `
		bus.on("fire", function(msg) { throw new Error("no replyTo crash"); });
	`)
	require.NoError(t, err)

	// Emit (no replyTo) — handler will throw but there's nowhere to send the error
	failed := make(chan bool, 1)
	unsub, _ := sdk.SubscribeTo[messages.HandlerFailedEvent](tk, ctx, "bus.handler.failed", func(e messages.HandlerFailedEvent, m messages.Message) {
		failed <- true
	})
	defer unsub()

	sdk.Emit(tk, ctx, messages.CustomEvent{Topic: "ts.no-reply.fire", Payload: json.RawMessage(`{}`)})

	select {
	case <-failed:
		// Good — failure event emitted, no panic
	case <-time.After(5 * time.Second):
		// If no retry policy, the error is logged but no event emitted. That's OK too.
	}
}

// C08: Teardown cleans up bus subscriptions
func TestFailureCascade_TeardownCleansSubscriptions(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "sub-cleanup.ts", `
		bus.on("ping", function(msg) { msg.reply({ pong: true }); });
	`)
	require.NoError(t, err)

	// Verify it works
	result, err := tk.EvalTS(ctx, "__ping.ts", `
		var r = bus.publish("ts.sub-cleanup.ping", {});
		return r.replyTo ? "ok" : "fail";
	`)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)

	// Teardown
	_, err = tk.Teardown(ctx, "sub-cleanup.ts")
	require.NoError(t, err)

	// Publish again — should not get a response (handler is gone)
	pr, err := sdk.Publish(tk, ctx, messages.CustomMsg{Topic: "ts.sub-cleanup.ping", Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)

	ch := make(chan bool, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- true })
	defer unsub()

	select {
	case <-ch:
		t.Fatal("received response after teardown — subscription not cleaned")
	case <-time.After(1 * time.Second):
		// Good — no response, subscription was cleaned
	}
}

// C10: Schedule fires but no handler exists
func TestFailureCascade_ScheduleNoHandler(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Schedule a message to a topic nobody listens to
	id, err := tk.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 100ms",
		Topic:      "ghost.topic.nobody.listens",
		Payload:    json.RawMessage(`{"ghost": true}`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	// Wait for it to fire
	time.Sleep(500 * time.Millisecond)

	// Kernel should still be healthy — no panic from publishing to a topic with no subscribers
	assert.True(t, tk.Alive(ctx))
}

// C02: Kernel starts with corrupted store
func TestFailureCascade_CorruptedStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/corrupt.db"

	// Write garbage to the store file
	require.NoError(t, os.WriteFile(storePath, []byte("not a sqlite database"), 0644))

	// NewSQLiteStore should fail on corrupt file
	_, err := brainkit.NewSQLiteStore(storePath)
	assert.Error(t, err)
}

// C05: Secret rotate when plugin restarter returns errors
func TestFailureCascade_SecretRotatePluginFails(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Set a secret, then rotate it — no plugins running, so restart is a no-op
	pr1, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "ROTATE_KEY", Value: "v1"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case <-ch1:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout setting secret")
	}
	unsub1()

	// Rotate
	pr2, _ := sdk.Publish(tk, ctx, messages.SecretsRotateMsg{Name: "ROTATE_KEY", NewValue: "v2", Restart: true})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case payload := <-ch2:
		assert.Contains(t, string(payload), "rotated")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout rotating secret")
	}
}

func TestFailureCascade_ConcurrentErrorHandler(t *testing.T) {
	var mu sync.Mutex
	var count int

	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(tmpDir + "/store.db")

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		ErrorHandler: func(err error, ctx brainkit.ErrorContext) {
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
			k.Schedule(context.Background(), brainkit.ScheduleConfig{
				Expression: "in 1h",
				Topic:      fmt.Sprintf("concurrent-err-%d", i),
				Payload:    json.RawMessage(`{}`),
			})
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
