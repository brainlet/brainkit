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
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStress_100DeploysSimultaneously — deploy 100 services at once.
func TestStress_100DeploysSimultaneously(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	var succeeded, failed atomic.Int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			src := fmt.Sprintf("stress-%d.ts", n)
			_, err := tk.Deploy(ctx, src, fmt.Sprintf(`output("stress-%d");`, n))
			if err != nil {
				failed.Add(1)
			} else {
				succeeded.Add(1)
			}
		}(i)
	}
	wg.Wait()

	t.Logf("100 deploys: %d succeeded, %d failed", succeeded.Load(), failed.Load())
	assert.Greater(t, succeeded.Load(), int64(0))
	assert.True(t, tk.Alive(ctx))

	// Teardown all
	for i := 0; i < 100; i++ {
		tk.Teardown(ctx, fmt.Sprintf("stress-%d.ts", i))
	}
}

// TestStress_1000BusPublishes — 1000 publishes from 100 goroutines.
func TestStress_1000BusPublishes(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var received atomic.Int64
	unsub, _ := tk.SubscribeRaw(ctx, "incoming.stress", func(m messages.Message) {
		received.Add(1)
	})
	defer unsub()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tk.PublishRaw(ctx, "incoming.stress", json.RawMessage(fmt.Sprintf(`{"g":%d,"j":%d}`, g, j)))
			}
		}(i)
	}
	wg.Wait()

	time.Sleep(1 * time.Second)
	count := received.Load()
	t.Logf("1000 publishes: received %d", count)
	assert.Greater(t, count, int64(500), "should receive majority of messages")
	assert.True(t, tk.Alive(ctx))
}

// TestStress_SecretRotationDuringReads — rotate a secret while 50 goroutines read it.
func TestStress_SecretRotationDuringReads(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Set initial value
	pr, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "rotating", Value: "v0"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Readers
	var readCount atomic.Int64
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					tk.EvalTS(ctx, "__read_rot.ts", `return secrets.get("rotating");`)
					readCount.Add(1)
				}
			}
		}()
	}

	// Rotator
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 10; i++ {
			pr, _ := sdk.Publish(tk, ctx, messages.SecretsRotateMsg{
				Name: "rotating", NewValue: fmt.Sprintf("v%d", i),
			})
			ch := make(chan []byte, 1)
			unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			<-ch
			unsub()
			time.Sleep(50 * time.Millisecond)
		}
		close(stop)
	}()

	wg.Wait()
	t.Logf("reads during rotation: %d", readCount.Load())
	assert.Greater(t, readCount.Load(), int64(0))
	assert.True(t, tk.Alive(ctx))
}

// TestStress_DeployWhileEvalTS — deploy new services while EvalTS is running.
func TestStress_DeployWhileEvalTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var wg sync.WaitGroup

	// EvalTS in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			tk.EvalTS(ctx, fmt.Sprintf("__eval_%d.ts", i), `return "eval-" + Math.random();`)
		}
	}()

	// Deploy in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			src := fmt.Sprintf("parallel-deploy-%d.ts", i)
			tk.Deploy(ctx, src, `output("parallel");`)
			tk.Teardown(ctx, src)
		}
	}()

	wg.Wait()
	assert.True(t, tk.Alive(ctx))
}

// TestStress_ToolCallsUnderLoad — 100 concurrent tool calls.
func TestStress_ToolCallsUnderLoad(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	

	var wg sync.WaitGroup
	var succeeded atomic.Int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			payload, ok := sendAndReceive(t, tk, messages.ToolCallMsg{
				Name:  "echo",
				Input: map[string]any{"message": fmt.Sprintf("stress-%d", n)},
			}, 10*time.Second)
			if ok && !responseHasError(payload) {
				succeeded.Add(1)
			}
		}(i)
	}
	wg.Wait()

	t.Logf("100 tool calls: %d succeeded", succeeded.Load())
	assert.Greater(t, succeeded.Load(), int64(50), "majority should succeed under load")
}

// TestStress_ScheduleStorm — create 50 schedules, all fire within 1 second.
func TestStress_ScheduleStorm(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	var received atomic.Int64
	unsub, _ := tk.SubscribeRaw(ctx, "stress.sched", func(m messages.Message) {
		received.Add(1)
	})
	defer unsub()

	for i := 0; i < 50; i++ {
		tk.Schedule(ctx, brainkit.ScheduleConfig{
			Expression: "in 200ms",
			Topic:      "stress.sched",
			Payload:    json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)),
		})
	}

	time.Sleep(3 * time.Second)
	count := received.Load()
	t.Logf("50 schedules: %d fired", count)
	assert.Greater(t, count, int64(25), "majority of schedules should fire")
}

// TestStress_MultiSurfaceSimultaneous — Go SDK + .ts + EvalTS all at once.
func TestStress_MultiSurfaceSimultaneous(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "multi-surface.ts", `
		bus.on("ts-ping", function(msg) { msg.reply({from: "ts"}); });
	`)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Go SDK surface
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			sendAndReceive(t, tk, messages.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "go"}}, 5*time.Second)
		}
	}()

	// .ts surface (publish to handler)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
				Topic: "ts.multi-surface.ts-ping", Payload: json.RawMessage(`{}`),
			})
			ch := make(chan []byte, 1)
			unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			select {
			case <-ch:
			case <-time.After(5 * time.Second):
			}
			unsub()
		}
	}()

	// EvalTS surface
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			tk.EvalTS(ctx, "__ms.ts", `return "eval-ok";`)
		}
	}()

	wg.Wait()
	assert.True(t, tk.Alive(ctx))
}
