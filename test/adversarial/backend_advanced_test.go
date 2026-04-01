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
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackendAdvanced_ConcurrentPublish — 50 concurrent publishes on every backend.
func TestBackendAdvanced_ConcurrentPublish(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test-" + backend,
				FSRoot: tmpDir, Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			ctx := context.Background()

			// Deploy handler that counts messages
			var received atomic.Int64
			unsub, _ := k.SubscribeRaw(ctx, "incoming.concurrent", func(m messages.Message) {
				received.Add(1)
			})
			defer unsub()

			// Fire 50 publishes concurrently
			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(1)
				go func(n int) {
					defer wg.Done()
					k.PublishRaw(ctx, "incoming.concurrent", json.RawMessage(fmt.Sprintf(`{"n":%d}`, n)))
				}(i)
			}
			wg.Wait()

			// Wait for delivery
			time.Sleep(500 * time.Millisecond)
			count := received.Load()
			assert.Greater(t, count, int64(0), "should receive messages on %s", backend)
			t.Logf("%s: received %d/50 messages", backend, count)
		})
	}
}

// TestBackendAdvanced_ScheduleFire — schedule fires on every backend.
func TestBackendAdvanced_ScheduleFire(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test-" + backend,
				FSRoot: tmpDir, Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			fired := make(chan []byte, 1)
			unsub, _ := k.SubscribeRaw(ctx, "sched.backend.fire", func(m messages.Message) {
				fired <- m.Payload
			})
			defer unsub()

			_, err = k.Schedule(ctx, brainkit.ScheduleConfig{
				Expression: "in 200ms",
				Topic:      "sched.backend.fire",
				Payload:    json.RawMessage(`{"backend":"` + backend + `"}`),
			})
			require.NoError(t, err)

			select {
			case p := <-fired:
				assert.Contains(t, string(p), backend)
			case <-ctx.Done():
				t.Fatalf("schedule didn't fire on %s", backend)
			}
		})
	}
}

// TestBackendAdvanced_DeployHandlerCall — deploy .ts handler, publish, get reply on every backend.
func TestBackendAdvanced_DeployHandlerCall(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test-" + backend,
				FSRoot: tmpDir, Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			type echoIn struct{ Message string `json:"message"` }
			brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
				Description: "echoes",
				Execute: func(ctx context.Context, in echoIn) (any, error) {
					return map[string]string{"echoed": in.Message}, nil
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err = k.Deploy(ctx, "backend-handler.ts", `
				bus.on("ask", async function(msg) {
					var r = await tools.call("echo", {message: "via-` + backend + `"});
					msg.reply(r);
				});
			`)
			require.NoError(t, err)

			pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
				Topic: "ts.backend-handler.ask", Payload: json.RawMessage(`{}`),
			})
			ch := make(chan []byte, 1)
			unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			defer unsub()

			select {
			case p := <-ch:
				assert.Contains(t, string(p), "via-"+backend)
			case <-ctx.Done():
				t.Fatalf("timeout on deploy+handler+call via %s", backend)
			}
		})
	}
}

// TestBackendAdvanced_SecretsOnBackend — secrets work on every backend transport.
func TestBackendAdvanced_SecretsOnBackend(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test-" + backend,
				FSRoot: tmpDir, Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Set
			pr1, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "backend-key", Value: "backend-val-" + backend})
			ch1 := make(chan []byte, 1)
			unsub1, _ := k.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
			select {
			case <-ch1:
			case <-ctx.Done():
				t.Fatalf("timeout set on %s", backend)
			}
			unsub1()

			// Get
			pr2, _ := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "backend-key"})
			ch2 := make(chan []byte, 1)
			unsub2, _ := k.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
			defer unsub2()

			select {
			case p := <-ch2:
				assert.Contains(t, string(p), "backend-val-"+backend)
			case <-ctx.Done():
				t.Fatalf("timeout get on %s", backend)
			}
		})
	}
}
