package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/brainlet/brainkit/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCrossDeployCallsGoTool — .ts calls Go tool during init, verifies result.
func testCrossDeployCallsGoTool(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	_, err := env.Kernel.Deploy(ctx, "call-go-adv.ts", `
		var result = await tools.call("echo", {message: "from-deploy"});
		output({toolResult: result, calledDuringDeploy: true});
	`)
	require.NoError(t, err)

	result, _ := env.Kernel.EvalTS(ctx, "__cg_adv.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "from-deploy")
	assert.Contains(t, result, "calledDuringDeploy")
}

// testCrossTSToolCallsAnotherTSTool — service A registers tool, service B calls it.
func testCrossTSToolCallsAnotherTSTool(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// Service A: register a tool
	_, err := env.Kernel.Deploy(ctx, "svc-a-tool-adv.ts", `
		const doubler = createTool({
			id: "doubler-adv",
			description: "doubles a number",
			inputSchema: z.object({n: z.number()}),
			execute: async ({n}) => ({doubled: n * 2}),
		});
		kit.register("tool", "doubler-adv", doubler);
	`)
	require.NoError(t, err)

	// Service B: call A's tool
	_, err = env.Kernel.Deploy(ctx, "svc-b-caller-adv.ts", `
		var result = await tools.call("doubler-adv", {n: 21});
		output(result);
	`)
	require.NoError(t, err)

	result, _ := env.Kernel.EvalTS(ctx, "__ab_adv.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "42")
}

// testCrossHandlerCallsTool — bus handler calls a tool during message processing.
func testCrossHandlerCallsTool(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("handler-tool-adv.ts", `
		bus.on("process", async function(msg) {
			var toolResult = await tools.call("echo", {message: msg.payload.data});
			msg.reply({processed: true, toolResult: toolResult});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.CustomMsg{
		Topic: "ts.handler-tool-adv.process", Payload: json.RawMessage(`{"data":"chain-test"}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "chain-test")
		assert.Contains(t, string(p), "processed")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// testCrossHandlerReadsSecret — bus handler reads a secret during processing.
func testCrossHandlerReadsSecret(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// Set a secret first
	pr0, _ := sdk.Publish(env.Kernel, ctx, messages.SecretsSetMsg{Name: "API_KEY_ADV", Value: "sk-test-123"})
	ch0 := make(chan []byte, 1)
	unsub0, _ := env.Kernel.SubscribeRaw(ctx, pr0.ReplyTo, func(m messages.Message) { ch0 <- m.Payload })
	<-ch0
	unsub0()

	err := env.Deploy("secret-handler-adv.ts", `
		bus.on("check", function(msg) {
			var key = secrets.get("API_KEY_ADV");
			msg.reply({hasKey: key.length > 0, keyPrefix: key.substring(0, 7)});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.CustomMsg{
		Topic: "ts.secret-handler-adv.check", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), `"hasKey":true`)
		assert.Contains(t, string(p), "sk-test")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// testCrossHandlerWritesFS — bus handler writes to filesystem.
func testCrossHandlerWritesFS(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("fs-handler-adv.ts", `
		bus.on("save", function(msg) {
			fs.writeFileSync("handler-output-adv.txt", msg.payload.content);
			var readBack = fs.readFileSync("handler-output-adv.txt", "utf8");
			msg.reply({written: true, readBack: readBack});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.CustomMsg{
		Topic: "ts.fs-handler-adv.save", Payload: json.RawMessage(`{"content":"from handler"}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "from handler")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// testCrossGoToolEmitsBusEvent — Go tool publishes bus event as side effect.
func testCrossGoToolEmitsBusEvent(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	k := freshEnv.Kernel

	var eventCount atomic.Int64

	type processIn struct{ Data string `json:"data"` }
	brainkit.RegisterTool(k, "process-and-emit-adv", registry.TypedTool[processIn]{
		Description: "processes and emits event",
		Execute: func(ctx context.Context, in processIn) (any, error) {
			sdk.Emit(k, ctx, messages.CustomEvent{
				Topic:   "events.processed-adv",
				Payload: json.RawMessage(fmt.Sprintf(`{"processed":"%s"}`, in.Data)),
			})
			return map[string]string{"result": "done"}, nil
		},
	})

	ctx := context.Background()

	unsub, _ := sdk.SubscribeTo[messages.CustomEvent](k, ctx, "events.processed-adv", func(e messages.CustomEvent, m messages.Message) {
		eventCount.Add(1)
	})
	defer unsub()

	pr, _ := sdk.Publish(k, ctx, messages.ToolCallMsg{Name: "process-and-emit-adv", Input: map[string]any{"data": "hello"}})
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

// testCrossTracedToolCall — tool call creates span, queryable via trace store.
func testCrossTracedToolCall(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	traceStore := tracing.NewMemoryTraceStore(1000)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		TraceStore: traceStore,
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "traced-echo-adv", registry.TypedTool[echoIn]{
		Description: "echoes with tracing",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})

	ctx := context.Background()
	pr, _ := sdk.Publish(k, ctx, messages.ToolCallMsg{Name: "traced-echo-adv", Input: map[string]any{"message": "traced"}})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	<-ch

	time.Sleep(200 * time.Millisecond)

	traces, _ := traceStore.ListTraces(tracing.TraceQuery{Limit: 10})
	assert.Greater(t, len(traces), 0, "tool call should be traced")

	if len(traces) > 0 {
		spans, _ := traceStore.GetTrace(traces[0].TraceID)
		assert.Greater(t, len(spans), 0, "trace should have spans")

		foundToolSpan := false
		for _, s := range spans {
			if s.Name == "tools.call:traced-echo-adv" || s.Attributes["tool"] == "traced-echo-adv" {
				foundToolSpan = true
			}
		}
		assert.True(t, foundToolSpan, "should find a span for traced-echo-adv tool call")
	}
}

// testCrossHealthDuringDeployChurn — health stays good during rapid deploy/teardown.
func testCrossHealthDuringDeployChurn(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	k := freshEnv.Kernel
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		src := fmt.Sprintf("churn-%d-adv.ts", i)
		k.Deploy(ctx, src, `output("churn");`)
		assert.True(t, k.Alive(ctx), "alive during churn iteration %d", i)
		k.Teardown(ctx, src)
	}

	health := k.Health(ctx)
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
}

// testCrossMetricsTrackSchedules — metrics reflect schedule count.
func testCrossMetricsTrackSchedules(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	k := freshEnv.Kernel
	ctx := context.Background()

	m0 := k.Metrics()
	assert.Equal(t, 0, m0.ActiveSchedules)

	id1, _ := k.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 1h", Topic: "m.sched1-adv", Payload: json.RawMessage(`{}`)})
	id2, _ := k.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 2h", Topic: "m.sched2-adv", Payload: json.RawMessage(`{}`)})

	m1 := k.Metrics()
	assert.Equal(t, 2, m1.ActiveSchedules)

	k.Unschedule(ctx, id1)
	m2 := k.Metrics()
	assert.Equal(t, 1, m2.ActiveSchedules)

	k.Unschedule(ctx, id2)
	m3 := k.Metrics()
	assert.Equal(t, 0, m3.ActiveSchedules)
}

// testCrossDeployWithPersistenceAndRestart — deploy, persist, restart, verify handler works.
func testCrossDeployWithPersistenceAndRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/cross-adv.db"

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

	_, err = k1.Deploy(context.Background(), "persist-handler-adv.ts", `
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
		Topic: "ts.persist-handler-adv.ask", Payload: json.RawMessage(`{"q":"hello"}`),
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
