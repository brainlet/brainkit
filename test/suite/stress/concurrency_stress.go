package stress

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// test100DeploysSimultaneously deploys 100 services at once.
func test100DeploysSimultaneously(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit
	var wg sync.WaitGroup
	var succeeded, failed atomic.Int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			src := fmt.Sprintf("stress-100-%d.ts", n)
			err := testutil.DeployErr(tk, src, fmt.Sprintf(`output("stress-100-%d");`, n))
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
	// Verify alive via a simple publish
	_, err := tk.PublishRaw(context.Background(), "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err)

	ctx := context.Background()
	for i := 0; i < 100; i++ {
		sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		sdk.Publish(tk, sctx, sdk.KitTeardownMsg{Source: fmt.Sprintf("stress-100-%d.ts", i)})
		cancel()
	}
}

// test1000BusPublishes fires 1000 publishes from 100 goroutines.
func test1000BusPublishes(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit
	ctx := context.Background()

	var received atomic.Int64
	unsub, _ := tk.SubscribeRaw(ctx, "incoming.stress.pub", func(m sdk.Message) {
		received.Add(1)
	})
	defer unsub()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tk.PublishRaw(ctx, "incoming.stress.pub", json.RawMessage(fmt.Sprintf(`{"g":%d,"j":%d}`, g, j)))
			}
		}(i)
	}
	wg.Wait()

	time.Sleep(1 * time.Second)
	count := received.Load()
	t.Logf("1000 publishes: received %d", count)
	assert.Greater(t, count, int64(500), "should receive majority of messages")
	_, err := tk.PublishRaw(ctx, "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err)
}

// testSecretRotationDuringReads rotates a secret while 50 goroutines read it.
func testSecretRotationDuringReads(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit
	ctx := context.Background()

	// Set initial value
	pr, _ := sdk.Publish(tk, ctx, sdk.SecretsSetMsg{Name: "stress-rotating", Value: "v0"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	<-ch
	unsub()

	var wg sync.WaitGroup
	stop := make(chan struct{})

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
					testutil.EvalTSErr(tk, "__stress_read_rot.ts", `return secrets.get("stress-rotating");`)
					readCount.Add(1)
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 10; i++ {
			pr, _ := sdk.Publish(tk, ctx, sdk.SecretsRotateMsg{
				Name: "stress-rotating", NewValue: fmt.Sprintf("v%d", i),
			})
			ch := make(chan []byte, 1)
			unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
			<-ch
			unsub()
			time.Sleep(50 * time.Millisecond)
		}
		close(stop)
	}()

	wg.Wait()
	t.Logf("reads during rotation: %d", readCount.Load())
	assert.Greater(t, readCount.Load(), int64(0))
	_, err := tk.PublishRaw(ctx, "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err)
}

// testDeployWhileEvalTS deploys new services while EvalTS is running.
func testDeployWhileEvalTS(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			testutil.EvalTSErr(tk, fmt.Sprintf("__stress_eval_%d.ts", i), `return "eval-" + Math.random();`)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx := context.Background()
		for i := 0; i < 20; i++ {
			src := fmt.Sprintf("stress-parallel-deploy-%d.ts", i)
			testutil.DeployErr(tk, src, `output("stress-parallel");`)
			sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			sdk.Publish(tk, sctx, sdk.KitTeardownMsg{Source: src})
			cancel()
		}
	}()

	wg.Wait()
	_, err := tk.PublishRaw(context.Background(), "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err)
}

// testToolCallsUnderLoad fires 100 concurrent tool calls.
func testToolCallsUnderLoad(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit

	var wg sync.WaitGroup
	var succeeded atomic.Int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			payload, ok := sendAndReceive(t, tk, sdk.ToolCallMsg{
				Name:  "echo",
				Input: map[string]any{"message": fmt.Sprintf("stress-%d", n)},
			}, 10*time.Second)
			if ok && !suite.ResponseHasError(payload) {
				succeeded.Add(1)
			}
		}(i)
	}
	wg.Wait()

	t.Logf("100 tool calls: %d succeeded", succeeded.Load())
	assert.Greater(t, succeeded.Load(), int64(50), "majority should succeed under load")
}

// testScheduleStorm creates 50 schedules, all fire within 1 second.
func testScheduleStorm(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit
	ctx := context.Background()

	var received atomic.Int64
	unsub, _ := tk.SubscribeRaw(ctx, "stress.sched.storm", func(m sdk.Message) {
		received.Add(1)
	})
	defer unsub()

	for i := 0; i < 50; i++ {
		sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		sdk.Publish(tk, sctx, sdk.ScheduleCreateMsg{
			Expression: "in 200ms",
			Topic:      "stress.sched.storm",
			Payload:    json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)),
		})
		cancel()
	}

	time.Sleep(3 * time.Second)
	count := received.Load()
	t.Logf("50 schedules: %d fired", count)
	assert.Greater(t, count, int64(25), "majority of schedules should fire")
}

// testMultiSurfaceSimultaneous exercises Go SDK + .ts + EvalTS all at once.
func testMultiSurfaceSimultaneous(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kit
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.Deploy(t, tk, "multi-stress-surface.ts", `
		bus.on("ts-ping", function(msg) { msg.reply({from: "ts"}); });
	`)

	var wg sync.WaitGroup

	// Go SDK surface
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			sendAndReceive(t, tk, sdk.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "go"}}, 5*time.Second)
		}
	}()

	// .ts surface (publish to handler)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			pr, _ := sdk.Publish(tk, ctx, sdk.CustomMsg{
				Topic: "ts.multi-stress-surface.ts-ping", Payload: json.RawMessage(`{}`),
			})
			ch := make(chan []byte, 1)
			unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
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
			testutil.EvalTSErr(tk, "__stress_ms.ts", `return "eval-ok";`)
		}
	}()

	wg.Wait()
	_, err := tk.PublishRaw(ctx, "test.alive", json.RawMessage(`{}`))
	assert.NoError(t, err)
}
