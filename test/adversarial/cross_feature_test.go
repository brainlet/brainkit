package adversarial_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossFeature_DeployCallsGoTool — .ts calls Go tool during init, verifies result.
func TestCrossFeature_DeployCallsGoTool(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "call-go.ts", `
		var result = await tools.call("echo", {message: "from-deploy"});
		output({toolResult: result, calledDuringDeploy: true});
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__cg.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "from-deploy")
	assert.Contains(t, result, "calledDuringDeploy")
}

// TestCrossFeature_TSToolCallsAnotherTSTool — service A registers tool, service B calls it.
func TestCrossFeature_TSToolCallsAnotherTSTool(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Service A: register a tool
	_, err := tk.Deploy(ctx, "svc-a-tool.ts", `
		const doubler = createTool({
			id: "doubler",
			description: "doubles a number",
			inputSchema: z.object({n: z.number()}),
			execute: async ({n}) => ({doubled: n * 2}),
		});
		kit.register("tool", "doubler", doubler);
	`)
	require.NoError(t, err)

	// Service B: call A's tool
	_, err = tk.Deploy(ctx, "svc-b-caller.ts", `
		var result = await tools.call("doubler", {n: 21});
		output(result);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__ab.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "42")
}

// TestCrossFeature_HandlerCallsTool — bus handler calls a tool during message processing.
func TestCrossFeature_HandlerCallsTool(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "handler-tool.ts", `
		bus.on("process", async function(msg) {
			var toolResult = await tools.call("echo", {message: msg.payload.data});
			msg.reply({processed: true, toolResult: toolResult});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.handler-tool.process", Payload: json.RawMessage(`{"data":"chain-test"}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "chain-test")
		assert.Contains(t, string(p), "processed")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// TestCrossFeature_HandlerReadsSecret — bus handler reads a secret during processing.
func TestCrossFeature_HandlerReadsSecret(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Set a secret first
	pr0, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "API_KEY", Value: "sk-test-123"})
	ch0 := make(chan []byte, 1)
	unsub0, _ := tk.SubscribeRaw(ctx, pr0.ReplyTo, func(m messages.Message) { ch0 <- m.Payload })
	<-ch0
	unsub0()

	_, err := tk.Deploy(ctx, "secret-handler.ts", `
		bus.on("check", function(msg) {
			var key = secrets.get("API_KEY");
			msg.reply({hasKey: key.length > 0, keyPrefix: key.substring(0, 7)});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.secret-handler.check", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), `"hasKey":true`)
		assert.Contains(t, string(p), "sk-test")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// TestCrossFeature_HandlerWritesFS — bus handler writes to filesystem.
func TestCrossFeature_HandlerWritesFS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "fs-handler.ts", `
		bus.on("save", async function(msg) {
			await fs.write("handler-output.txt", msg.payload.content);
			var read = await fs.read("handler-output.txt");
			msg.reply({written: true, readBack: read.data});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.fs-handler.save", Payload: json.RawMessage(`{"content":"from handler"}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "from handler")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// TestCrossFeature_GoToolEmitsBusEvent — Go tool publishes bus event as side effect.
func TestCrossFeature_GoToolEmitsBusEvent(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	var eventCount atomic.Int64

	type processIn struct{ Data string `json:"data"` }
	brainkit.RegisterTool(k, "process-and-emit", registry.TypedTool[processIn]{
		Description: "processes and emits event",
		Execute: func(ctx context.Context, in processIn) (any, error) {
			// Side effect: emit an event
			sdk.Emit(k, ctx, messages.CustomEvent{
				Topic:   "events.processed",
				Payload: json.RawMessage(fmt.Sprintf(`{"processed":"%s"}`, in.Data)),
			})
			return map[string]string{"result": "done"}, nil
		},
	})

	ctx := context.Background()

	// Subscribe to the side-effect event
	unsub, _ := sdk.SubscribeTo[messages.CustomEvent](k, ctx, "events.processed", func(e messages.CustomEvent, m messages.Message) {
		eventCount.Add(1)
	})
	defer unsub()

	// Call the tool
	pr, _ := sdk.Publish(k, ctx, messages.ToolCallMsg{Name: "process-and-emit", Input: map[string]any{"data": "hello"}})
	ch := make(chan []byte, 1)
	unsub2, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "done")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	time.Sleep(200 * time.Millisecond)
	assert.Greater(t, eventCount.Load(), int64(0), "event should have been emitted")
}

// TestCrossFeature_TracedToolCall — tool call creates span, queryable via trace.get.
func TestCrossFeature_TracedToolCall(t *testing.T) {
	tmpDir := t.TempDir()
	traceStore := tracing.NewMemoryTraceStore(1000)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		TraceStore: traceStore,
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "traced-echo", registry.TypedTool[echoIn]{
		Description: "echoes with tracing",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})

	ctx := context.Background()
	pr, _ := sdk.Publish(k, ctx, messages.ToolCallMsg{Name: "traced-echo", Input: map[string]any{"message": "traced"}})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	<-ch

	time.Sleep(200 * time.Millisecond)

	// Query traces — should find span for the tool call
	traces, _ := traceStore.ListTraces(tracing.TraceQuery{Limit: 10})
	assert.Greater(t, len(traces), 0, "tool call should be traced")

	// Get the specific trace
	if len(traces) > 0 {
		spans, _ := traceStore.GetTrace(traces[0].TraceID)
		assert.Greater(t, len(spans), 0, "trace should have spans")

		foundToolSpan := false
		for _, s := range spans {
			if s.Name == "tools.call:traced-echo" || s.Attributes["tool"] == "traced-echo" {
				foundToolSpan = true
			}
		}
		assert.True(t, foundToolSpan, "should find a span for traced-echo tool call")
	}
}

// TestCrossFeature_HealthDuringDeployChurn — health stays good during rapid deploy/teardown.
func TestCrossFeature_HealthDuringDeployChurn(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		src := fmt.Sprintf("churn-%d.ts", i)
		tk.Deploy(ctx, src, `output("churn");`)
		assert.True(t, tk.Alive(ctx), "alive during churn iteration %d", i)
		tk.Teardown(ctx, src)
	}

	health := tk.Health(ctx)
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
}

// TestCrossFeature_MetricsTrackSchedules — metrics reflect schedule count.
func TestCrossFeature_MetricsTrackSchedules(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	m0 := tk.Metrics()
	assert.Equal(t, 0, m0.ActiveSchedules)

	id1, _ := tk.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 1h", Topic: "m.sched1", Payload: json.RawMessage(`{}`)})
	id2, _ := tk.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 2h", Topic: "m.sched2", Payload: json.RawMessage(`{}`)})

	m1 := tk.Metrics()
	assert.Equal(t, 2, m1.ActiveSchedules)

	tk.Unschedule(ctx, id1)
	m2 := tk.Metrics()
	assert.Equal(t, 1, m2.ActiveSchedules)

	tk.Unschedule(ctx, id2)
	m3 := tk.Metrics()
	assert.Equal(t, 0, m3.ActiveSchedules)
}

// TestCrossFeature_DeployWithPersistenceAndRestart — deploy, persist, restart, verify handler works.
func TestCrossFeature_DeployWithPersistenceAndRestart(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/cross.db"

	// Phase 1: Deploy handler, persist
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k1, "echo", registry.TypedTool[echoIn]{
		Description: "echoes",
		Execute:     func(ctx context.Context, in echoIn) (any, error) { return map[string]string{"echoed": in.Message}, nil },
	})

	_, err = k1.Deploy(context.Background(), "persist-handler.ts", `
		bus.on("ask", async function(msg) {
			var r = await tools.call("echo", {message: "persisted:" + msg.payload.q});
			msg.reply(r);
		});
	`)
	require.NoError(t, err)
	k1.Close()

	// Phase 2: Restart, call handler, verify it works
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	brainkit.RegisterTool(k2, "echo", registry.TypedTool[echoIn]{
		Description: "echoes",
		Execute:     func(ctx context.Context, in echoIn) (any, error) { return map[string]string{"echoed": in.Message}, nil },
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(k2, ctx, messages.CustomMsg{
		Topic: "ts.persist-handler.ask", Payload: json.RawMessage(`{"q":"hello"}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k2.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "persisted:hello")
	case <-ctx.Done():
		t.Fatal("timeout — handler should work after restart")
	}
}
