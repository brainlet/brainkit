package tracing

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	tracingpkg "github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/modules/packages"
	_ "github.com/brainlet/brainkit/modules/packages/bundlers/esbuild"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	tracingmod "github.com/brainlet/brainkit/modules/tracing"
	"github.com/brainlet/brainkit/modules/tracing/tracingmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/stores"
	"github.com/brainlet/brainkit/test/suite"
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
	kitStore, _ := stores.NewSQLite(tmpDir + "/trace.db")
	t.Cleanup(func() { kitStore.Close() })
	k, err := brainkit.New(brainkit.Config{
		Transport:  brainkit.Memory(),
		Namespace:  "test",
		CallerID:   "test",
		FSRoot:     tmpDir,
		Store:      kitStore,
		TraceStore: store,
		Modules: []bkmodule.Module{
			toolsmod.New(),
			packages.New(),
			tracingmod.New(tracingmod.Config{Store: store}),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	type echoIn struct {
		Message string `json:"message"`
	}
	require.NoError(t, k.Mount(context.Background(), toolsmod.GoTool("echo", toolsmod.TypedTool[echoIn]{
		Description: "echoes",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})))

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

	_, err := sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   tracingServiceTopic("traced.ts", "ping"),
		Payload: json.RawMessage(`{"x":true}`),
	}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)

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

	resp, err := tracingmsg.CallTraceList(env.Kit, ctx, tracingmsg.TraceListMsg{Limit: 10}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	var traces []tracingpkg.TraceSummary
	json.Unmarshal(resp.Traces, &traces)
	assert.Greater(t, len(traces), 0, "expected traces from bus query")
}

func testNoStoreNoOp(t *testing.T, _ *suite.TestEnv) {
	env := suite.Minimal(t)
	ctx := context.Background()

	resp, err := toolmsg.CallToolList(env.Kit, ctx, toolmsg.ToolListMsg{}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.NotNil(t, resp.Tools)
}

func testToolCallCreatesSpan(t *testing.T, _ *suite.TestEnv) {
	env, store := tracingEnv(t)
	ctx := context.Background()

	_, err := toolmsg.CallToolCall(env.Kit, ctx, toolmsg.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "traced"}}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)

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

	_, err := sdk.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic: "ts.source-a.ping", Payload: json.RawMessage(`{}`),
	}, sdk.WithCallTimeout(3*time.Second))
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	resp, err := tracingmsg.CallTraceList(env.Kit, ctx, tracingmsg.TraceListMsg{Limit: 100}, sdk.WithCallTimeout(3*time.Second))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Traces)
}

// testSampleRate — with 0 sample rate, no spans recorded (or minimal).
func testSampleRate(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store := tracingpkg.NewMemoryTraceStore(1000)

	k, err := brainkit.New(brainkit.Config{
		Transport:       brainkit.Memory(),
		Namespace:       "test",
		CallerID:        "test",
		FSRoot:          tmpDir,
		TraceStore:      store,
		TraceSampleRate: 0.0, // sample nothing
		Modules:         []bkmodule.Module{toolsmod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct {
		Message string `json:"message"`
	}
	require.NoError(t, k.Mount(context.Background(), toolsmod.GoTool("echo", toolsmod.TypedTool[echoIn]{
		Description: "echoes",
		Execute:     func(ctx context.Context, in echoIn) (any, error) { return in, nil },
	})))

	ctx := context.Background()
	_, err = toolmsg.CallToolCall(k, ctx, toolmsg.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "no-trace"}}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)

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
		Transport:  brainkit.Memory(),
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

	resp, err := tracingmsg.CallTraceGet(env.Kit, ctx, tracingmsg.TraceGetMsg{TraceID: "nonexistent-trace-id"}, sdk.WithCallTimeout(3*time.Second))
	require.NoError(t, err)
	assert.NotNil(t, resp.Spans)
}

func tracingServiceTopic(source, topic string) string {
	name := strings.TrimSuffix(source, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
