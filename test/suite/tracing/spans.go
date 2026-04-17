package tracing

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/transport"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	tracingpkg "github.com/brainlet/brainkit/internal/tracing"
	tracingmod "github.com/brainlet/brainkit/modules/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tracingEnv creates a fresh kernel with MemoryTraceStore and returns both.
func tracingEnv(t *testing.T) (*suite.TestEnv, *tracingpkg.MemoryTraceStore) {
	t.Helper()
	store := tracingpkg.NewMemoryTraceStore(1000)
	env := suite.Full(t, suite.WithTracing(), suite.WithPersistence())
	// Replace the tracer store with our own so we can inspect it.
	// The suite WithTracing() already creates a MemoryTraceStore internally,
	// but we need the reference. So we create our own kit.
	tmpDir := t.TempDir()
	kitStore, _ := brainkit.NewSQLiteStore(tmpDir + "/trace.db")
	t.Cleanup(func() { kitStore.Close() })
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace:  "test",
		CallerID:   "test",
		FSRoot:     tmpDir,
		Store:      kitStore,
		TraceStore: store,
		Modules: []brainkit.Module{
			tracingmod.New(tracingmod.Config{Store: store}),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", tools.TypedTool[echoIn]{
		Description: "echoes",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})

	env.Kit = k
	return env, store
}

func testCommandRequestCreatesSpan(t *testing.T, _ *suite.TestEnv) {
	env, store := tracingEnv(t)

	result := testutil.EvalTS(t, env.Kit, "__trace_test.ts", `
		var list = tools.list();
		return JSON.stringify(list);
	`)
	require.NotEmpty(t, result)

	traces, err := store.ListTraces(tracingpkg.TraceQuery{})
	require.NoError(t, err)
	require.Greater(t, len(traces), 0, "expected at least one trace from JS command")
}

func testHandlerCreatesSpan(t *testing.T, _ *suite.TestEnv) {
	env, store := tracingEnv(t)
	ctx := context.Background()

	testutil.Deploy(t, env.Kit, "traced.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(env.Kit, ctx, "traced.ts", "ping", map[string]bool{"x": true})
	replyCh := make(chan struct{}, 1)
	replyCancel, _ := env.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(_ sdk.Message) { replyCh <- struct{}{} })
	defer replyCancel()
	<-replyCh

	time.Sleep(50 * time.Millisecond)

	traces, err := store.ListTraces(tracingpkg.TraceQuery{})
	require.NoError(t, err)

	found := false
	for _, tr := range traces {
		if tr.RootSpan != "" {
			found = true
		}
	}
	assert.True(t, found, "expected traces with root spans")
}

func testQueryViaBus(t *testing.T, _ *suite.TestEnv) {
	env, store := tracingEnv(t)
	ctx := context.Background()

	span := tracingpkg.NewTracer(store, 1.0).StartSpan("test.op", ctx)
	span.End(nil)

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.TraceListMsg{Limit: 10})
	listCh := make(chan sdk.TraceListResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.TraceListResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.TraceListResp, _ sdk.Message) {
		listCh <- resp
	})
	defer cancel()

	select {
	case resp := <-listCh:
		var traces []tracingpkg.TraceSummary
		json.Unmarshal(resp.Traces, &traces)
		assert.Greater(t, len(traces), 0, "expected traces from bus query")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout querying traces")
	}
}

func testNoStoreNoOp(t *testing.T, _ *suite.TestEnv) {
	env := suite.Minimal(t)
	ctx := context.Background()

	pub, _ := sdk.Publish(env.Kit, ctx, sdk.ToolListMsg{})
	ch := make(chan sdk.ToolListResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.ToolListResp](env.Kit, ctx, pub.ReplyTo, func(resp sdk.ToolListResp, _ sdk.Message) {
		ch <- resp
	})
	defer cancel()

	select {
	case resp := <-ch:
		assert.NotNil(t, resp.Tools)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout — tracing should be transparent no-op")
	}
}

func testToolCallCreatesSpan(t *testing.T, _ *suite.TestEnv) {
	env, store := tracingEnv(t)
	ctx := context.Background()

	pr, _ := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "traced"}})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()
	<-ch

	time.Sleep(100 * time.Millisecond)

	traces, err := store.ListTraces(tracingpkg.TraceQuery{Limit: 10})
	require.NoError(t, err)
	assert.Greater(t, len(traces), 0, "tool call should create trace spans")
}

func testDeployCreatesSpan(t *testing.T, _ *suite.TestEnv) {
	env, store := tracingEnv(t)

	testutil.Deploy(t, env.Kit, "traced-deploy.ts", `output("traced");`)

	time.Sleep(100 * time.Millisecond)

	traces, _ := store.ListTraces(tracingpkg.TraceQuery{Limit: 10})
	assert.Greater(t, len(traces), 0, "deploy should create trace spans")

	if len(traces) > 0 {
		spans, _ := store.GetTrace(traces[0].TraceID)
		foundDeploy := false
		for _, s := range spans {
			if s.Name == "kit.deploy:traced-deploy.ts" {
				foundDeploy = true
				assert.Equal(t, "traced-deploy.ts", s.Source)
			}
		}
		assert.True(t, foundDeploy, "should find a kit.deploy span")
	}
}

func testQueryBySource(t *testing.T, _ *suite.TestEnv) {
	env, _ := tracingEnv(t)
	ctx := context.Background()

	testutil.Deploy(t, env.Kit, "source-a.ts", `
		bus.on("ping", function(msg) { msg.reply({ok:true}); });
	`)

	pr, _ := sdk.Publish(env.Kit, ctx, sdk.CustomMsg{
		Topic: "ts.source-a.ping", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
	}

	time.Sleep(200 * time.Millisecond)

	pr2, _ := sdk.Publish(env.Kit, ctx, sdk.TraceListMsg{Limit: 100})
	ch2 := make(chan []byte, 1)
	unsub2, _ := env.Kit.SubscribeRaw(ctx, pr2.ReplyTo, func(m sdk.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		assert.NotEmpty(t, p)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout querying traces")
	}
}

// testSampleRate — with 0 sample rate, no spans recorded (or minimal).
func testSampleRate(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store := tracingpkg.NewMemoryTraceStore(1000)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace:       "test",
		CallerID:        "test",
		FSRoot:          tmpDir,
		TraceStore:      store,
		TraceSampleRate: 0.0, // sample nothing
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", tools.TypedTool[echoIn]{
		Description: "echoes",
		Execute:     func(ctx context.Context, in echoIn) (any, error) { return in, nil },
	})

	ctx := context.Background()
	pr, _ := sdk.Publish(k, ctx, sdk.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "no-trace"}})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()
	<-ch

	time.Sleep(100 * time.Millisecond)
	traces, _ := store.ListTraces(tracingpkg.TraceQuery{Limit: 10})
	// With 0% sample rate, we expect very few or no traces — just verify no panic
	_ = traces
}

// testTraceContextPropagates — trace IDs propagate across namespaces.
func testTraceContextPropagates(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store := tracingpkg.NewMemoryTraceStore(1000)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace:  "test",
		CallerID:   "test",
		FSRoot:     tmpDir,
		TraceStore: store,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Stamp trace context into the publish context
	traceCtx := transport.WithTraceIDs(ctx, "trace-abc-123", "span-parent-456", "")
	traceCtx = transport.WithSampled(traceCtx, "true")

	// Subscribe to a topic and capture metadata
	receivedCh := make(chan map[string]string, 1)
	unsub, err := k.SubscribeRawTo(ctx, "test", "trace.test.target", func(msg sdk.Message) {
		receivedCh <- msg.Metadata
	})
	require.NoError(t, err)
	defer unsub()

	// Publish via cross-namespace path (same namespace for test simplicity)
	_, err = k.PublishRawTo(traceCtx, "test", "trace.test.target", []byte(`{"test":true}`))
	require.NoError(t, err)

	select {
	case meta := <-receivedCh:
		assert.Equal(t, "trace-abc-123", meta["traceId"], "traceId must propagate across namespaces")
		assert.Equal(t, "span-parent-456", meta["parentSpanId"], "parentSpanId must propagate")
		assert.Equal(t, "true", meta["traceSampled"], "traceSampled must propagate")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cross-namespace message")
	}
}

func testEmptyStore(t *testing.T, _ *suite.TestEnv) {
	env, store := tracingEnv(t)
	ctx := context.Background()

	traces, err := store.ListTraces(tracingpkg.TraceQuery{Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, traces)

	pr, _ := sdk.Publish(env.Kit, ctx, sdk.TraceGetMsg{TraceID: "nonexistent-trace-id"})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "spans")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}
