package infra

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/tracing"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startKernelWithTracing(t *testing.T) (*brainkit.Kernel, *tracing.MemoryTraceStore) {
	t.Helper()
	store := tracing.NewMemoryTraceStore(1000)
	storePath := t.TempDir() + "/trace-test.db"
	kitStore, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Store:      kitStore,
		TraceStore: store,
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return k, store
}

func TestTracing_CommandRequestCreatesSpan(t *testing.T) {
	k, store := startKernelWithTracing(t)
	ctx := context.Background()

	// Call a command from JS (goes through bridge → creates span)
	result, err := k.EvalTS(ctx, "__trace_test.ts", `
		var list = tools.list();
		return JSON.stringify(list);
	`)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// Verify span was recorded
	traces, err := store.ListTraces(tracing.TraceQuery{})
	require.NoError(t, err)
	require.Greater(t, len(traces), 0, "expected at least one trace from JS command")
}

func TestTracing_HandlerCreatesSpan(t *testing.T) {
	k, store := startKernelWithTracing(t)
	ctx := context.Background()

	// Deploy a handler
	_, err := k.Deploy(ctx, "traced.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: true });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Send a message — handler invocation should create a span
	sendPR, _ := sdk.SendToService(k, ctx, "traced.ts", "ping", map[string]bool{"x": true})
	replyCh := make(chan struct{}, 1)
	replyCancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(_ messages.Message) { replyCh <- struct{}{} })
	defer replyCancel()
	<-replyCh

	// Wait a tick for span recording
	time.Sleep(50 * time.Millisecond)

	traces, err := store.ListTraces(tracing.TraceQuery{})
	require.NoError(t, err)

	// Should have spans from the command (kit.deploy) + handler invocation
	found := false
	for _, tr := range traces {
		if tr.RootSpan != "" {
			found = true
		}
	}
	assert.True(t, found, "expected traces with root spans")
}

func TestTracing_QueryViaBus(t *testing.T) {
	k, store := startKernelWithTracing(t)
	ctx := context.Background()

	// Create some spans
	span := tracing.NewTracer(store, 1.0).StartSpan("test.op", ctx)
	span.End(nil)

	// Query via bus command
	pub, _ := sdk.Publish(k, ctx, messages.TraceListMsg{Limit: 10})
	listCh := make(chan messages.TraceListResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.TraceListResp](k, ctx, pub.ReplyTo, func(resp messages.TraceListResp, _ messages.Message) {
		listCh <- resp
	})
	defer cancel()

	select {
	case resp := <-listCh:
		var traces []tracing.TraceSummary
		json.Unmarshal(resp.Traces, &traces)
		assert.Greater(t, len(traces), 0, "expected traces from bus query")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout querying traces")
	}
}

func TestTracing_NoStoreNoOp(t *testing.T) {
	// Kernel without TraceStore — tracing should be no-op
	k, err := brainkit.NewKernel(brainkit.KernelConfig{})
	require.NoError(t, err)
	defer k.Close()
	ctx := context.Background()

	// Commands should still work without error
	pub, _ := sdk.Publish(k, ctx, messages.ToolListMsg{})
	ch := make(chan messages.ToolListResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.ToolListResp](k, ctx, pub.ReplyTo, func(resp messages.ToolListResp, _ messages.Message) {
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
