package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrossKit_TraceContextPropagates(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Stamp trace context into the publish context
	traceCtx := messaging.WithTraceIDs(ctx, "trace-abc-123", "span-parent-456", "")
	traceCtx = messaging.WithSampled(traceCtx, "true")

	// Subscribe to a topic and capture metadata
	receivedCh := make(chan map[string]string, 1)
	unsub, err := k.SubscribeRawTo(ctx, k.Namespace(), "trace.test.target", func(msg messages.Message) {
		receivedCh <- msg.Metadata
	})
	require.NoError(t, err)
	defer unsub()

	// Publish via cross-namespace path (same namespace for test simplicity)
	_, err = k.PublishRawTo(traceCtx, k.Namespace(), "trace.test.target", []byte(`{"test":true}`))
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
