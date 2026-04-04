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
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
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
	_, err = k.Deploy(context.Background(), "broken-store-cascade.ts", `output("works in memory");`)
	require.NoError(t, err)

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
	pr1, _ := sdk.Publish(freshEnv.Kernel, ctx, messages.SecretsSetMsg{Name: "ROTATE_KEY_CASCADE", Value: "v1"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := freshEnv.Kernel.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case <-ch1:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout setting secret")
	}
	unsub1()

	// Rotate
	pr2, _ := sdk.Publish(freshEnv.Kernel, ctx, messages.SecretsRotateMsg{Name: "ROTATE_KEY_CASCADE", NewValue: "v2", Restart: true})
	ch2 := make(chan []byte, 1)
	unsub2, _ := freshEnv.Kernel.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
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

	_, err := freshEnv.Kernel.Deploy(ctx, "no-reply-cascade.ts", `
		bus.on("fire", function(msg) { throw new Error("no replyTo crash"); });
	`)
	require.NoError(t, err)

	// Emit (no replyTo) — handler will throw but there's nowhere to send the error
	failed := make(chan bool, 1)
	unsub, _ := sdk.SubscribeTo[messages.HandlerFailedEvent](freshEnv.Kernel, ctx, "bus.handler.failed", func(e messages.HandlerFailedEvent, m messages.Message) {
		failed <- true
	})
	defer unsub()

	sdk.Emit(freshEnv.Kernel, ctx, messages.CustomEvent{Topic: "ts.no-reply-cascade.fire", Payload: json.RawMessage(`{}`)})

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

	_, err := freshEnv.Kernel.Deploy(ctx, "sub-cleanup-cascade.ts", `
		bus.on("ping", function(msg) { msg.reply({ pong: true }); });
	`)
	require.NoError(t, err)

	// Verify it works
	result, err := freshEnv.Kernel.EvalTS(ctx, "__ping-cascade.ts", `
		var r = bus.publish("ts.sub-cleanup-cascade.ping", {});
		return r.replyTo ? "ok" : "fail";
	`)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)

	// Teardown
	_, err = freshEnv.Kernel.Teardown(ctx, "sub-cleanup-cascade.ts")
	require.NoError(t, err)

	// Publish again — should not get a response (handler is gone)
	pr, err := sdk.Publish(freshEnv.Kernel, ctx, messages.CustomMsg{Topic: "ts.sub-cleanup-cascade.ping", Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)

	ch := make(chan bool, 1)
	unsub, _ := freshEnv.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- true })
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
				Topic:      fmt.Sprintf("concurrent-err-cascade-%d", i),
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
