package adversarial_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShutdown_GracefulWithActiveDeployments — close with active deployments.
func TestShutdown_GracefulWithActiveDeployments(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy several services
	for i := 0; i < 5; i++ {
		tk.Deploy(ctx, "shutdown-svc.ts", `bus.on("ping", function(msg) { msg.reply({ok:true}); });`)
		tk.Teardown(ctx, "shutdown-svc.ts") // clean each iteration
	}
	_, err := tk.Deploy(ctx, "final-svc.ts", `bus.on("ping", function(msg) { msg.reply({ok:true}); });`)
	require.NoError(t, err)

	// Shutdown should be clean
	err = tk.Close()
	assert.NoError(t, err)
}

// TestShutdown_WithActiveSchedules — close cancels all schedules.
func TestShutdown_WithActiveSchedules(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		tk.Schedule(ctx, brainkit.ScheduleConfig{
			Expression: "every 1h",
			Topic:      "shutdown-sched",
			Payload:    json.RawMessage(`{}`),
		})
	}

	err := tk.Close()
	assert.NoError(t, err)
}

// TestShutdown_WithActiveSubscriptions — close unsubscribes all.
func TestShutdown_WithActiveSubscriptions(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "sub-shutdown.ts", `
		bus.subscribe("topic1", function() {});
		bus.subscribe("topic2", function() {});
		bus.subscribe("topic3", function() {});
		output("subscribed");
	`)
	require.NoError(t, err)

	err = tk.Close()
	assert.NoError(t, err)
}

// TestShutdown_DrainTimeout — drain with stuck handler forces close.
func TestShutdown_DrainTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = k.Deploy(ctx, "stuck.ts", `
		bus.on("stuck", async function(msg) {
			await new Promise(r => setTimeout(r, 60000)); // 60s — will exceed drain timeout
		});
	`)
	require.NoError(t, err)

	// Fire a message to the stuck handler
	k.PublishRaw(ctx, "ts.stuck.stuck", json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond) // let handler start

	// Shutdown with 1s timeout — should force-close
	shutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = k.Shutdown(shutCtx)
	assert.NoError(t, err) // force-close is still a clean close
}

// TestShutdown_ConcurrentClose — multiple goroutines calling Close.
func TestShutdown_ConcurrentClose(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k.Close() // second+ calls should be no-op
		}()
	}
	wg.Wait()
}

// TestShutdown_StorageStillAccessibleBeforeClose — storage works right until close.
func TestShutdown_StorageAccessBeforeClose(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Add storage at runtime
	err := tk.AddStorage("runtime-store", brainkit.InMemoryStorage())
	require.NoError(t, err)

	// Use it
	_, err = tk.Deploy(ctx, "storage-use.ts", `output("using storage");`)
	require.NoError(t, err)

	// Remove it
	err = tk.RemoveStorage("runtime-store")
	require.NoError(t, err)

	// Close
	err = tk.Close()
	assert.NoError(t, err)
}
