//go:build stress

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func TestBusStress_ConcurrentAsks(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/echo", ShortName: "echo",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return input, nil
			},
		},
	})

	const N = 1000
	var wg sync.WaitGroup
	var errCount atomic.Int32

	for i := range N {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload := fmt.Sprintf(`{"name":"echo","input":{"n":%d}}`, idx)
			resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
				Topic:    "tools.call",
				CallerID: "stress",
				Payload:  json.RawMessage(payload),
			})
			if err != nil {
				errCount.Add(1)
				return
			}
			var result map[string]any
			json.Unmarshal(resp.Payload, &result)
			if result["n"] != float64(idx) {
				errCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if c := errCount.Load(); c > 0 {
		t.Fatalf("%d/%d concurrent asks failed", c, N)
	}
}

func TestBusStress_InterceptorPipelineUnderLoad(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Register 3 interceptors that each add metadata
	var order atomic.Int32

	for i := range 3 {
		priority := (i + 1) * 100
		name := fmt.Sprintf("interceptor-%d", i)
		kit.Bus.AddInterceptor(&testInterceptor{
			name:     name,
			priority: priority,
			fn: func(msg *bus.Message) error {
				if msg.Metadata == nil {
					msg.Metadata = make(map[string]string)
				}
				msg.Metadata[name] = fmt.Sprintf("p%d", priority)
				order.Add(1)
				return nil
			},
		})
	}

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/check", ShortName: "check",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(map[string]string{"ok": "true"})
			},
		},
	})

	const N = 500
	var wg sync.WaitGroup
	var errCount atomic.Int32

	for i := range N {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
				Topic:    "tools.call",
				CallerID: "stress",
				Payload:  json.RawMessage(`{"name":"check","input":{}}`),
			})
			if err != nil {
				errCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if c := errCount.Load(); c > 0 {
		t.Fatalf("%d/%d interceptor calls failed", c, N)
	}

	// Each message triggers 3 interceptors
	total := order.Load()
	expected := int32(N * 3)
	if total != expected {
		t.Errorf("expected %d interceptor invocations, got %d", expected, total)
	}
}

func TestBusStress_WorkerGroupDistribution(t *testing.T) {
	b := bus.NewBus(bus.NewInProcessTransport())
	defer b.Close()

	const workers = 3
	const messages = 300
	var counts [workers]atomic.Int32

	for i := range workers {
		idx := i
		b.On("work.queue", func(msg bus.Message, _ bus.ReplyFunc) {
			counts[idx].Add(1)
		}, bus.AsWorker("processors"))
	}

	// Send messages
	for i := range messages {
		b.Send(bus.Message{
			Topic:   "work.queue",
			Payload: json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)),
		})
	}

	// Wait for delivery
	time.Sleep(500 * time.Millisecond)

	// Check distribution — each worker should get roughly equal share
	total := 0
	for i := range workers {
		c := int(counts[i].Load())
		total += c
		t.Logf("worker %d: %d messages", i, c)
	}

	if total != messages {
		t.Errorf("expected %d total, got %d", messages, total)
	}

	// Allow ±50% variance (worker groups are load-balanced, not perfectly even)
	expected := messages / workers
	for i := range workers {
		c := int(counts[i].Load())
		if c < expected/2 || c > expected*2 {
			t.Errorf("worker %d got %d messages (expected ~%d ±50%%)", i, c, expected)
		}
	}
}

// testInterceptor is a bus.Interceptor implementation for testing.
type testInterceptor struct {
	name     string
	priority int
	fn       func(msg *bus.Message) error
}

func (i *testInterceptor) Name() string             { return i.name }
func (i *testInterceptor) Priority() int             { return i.priority }
func (i *testInterceptor) Match(topic string) bool   { return bus.TopicMatches("tools.*", topic) }
func (i *testInterceptor) Process(msg *bus.Message) error { return i.fn(msg) }
