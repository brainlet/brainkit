package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsync_CorrelationIDFiltering(t *testing.T) {
	rt := newTestKernel(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to tool list results
	received := make(chan messages.ToolListResp, 1)
	var receivedCorrID string

	corrID, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
	require.NoError(t, err)
	assert.NotEmpty(t, corrID, "Publish must return a correlationID")

	unsub, err := sdk.Subscribe[messages.ToolListResp](rt, ctx, func(resp messages.ToolListResp, msg messages.Message) {
		if msg.Metadata["correlationId"] == corrID {
			receivedCorrID = msg.Metadata["correlationId"]
			received <- resp
		}
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case resp := <-received:
		assert.Equal(t, corrID, receivedCorrID)
		assert.NotNil(t, resp.Tools)
	case <-ctx.Done():
		t.Fatal("timeout waiting for correlated response")
	}
}

func TestAsync_MultipleInFlight(t *testing.T) {
	rt := newTestKernel(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Fire 10 concurrent PublishAwait calls — stress test for correlationID filtering.
	// GoChannel fan-out delivers every result to every subscriber. Each goroutine
	// must filter by its own correlationID. The resultCh buffer (16) must handle
	// receiving other goroutines' results without dropping its own.
	const n = 10
	var wg sync.WaitGroup
	results := make([]messages.ToolListResp, n)
	errors := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errors[idx] = sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](rt, ctx, messages.ToolListMsg{})
		}(i)
	}

	wg.Wait()

	for i := 0; i < n; i++ {
		assert.NoError(t, errors[i], "request %d should succeed", i)
		assert.NotNil(t, results[i].Tools, "request %d should have tools", i)
	}
}

func TestAsync_ContextCancellation(t *testing.T) {
	rt := newTestKernel(t)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](rt, ctx, messages.ToolListMsg{})
	assert.Error(t, err, "should fail with cancelled context")
}

func TestAsync_SubscribeCancellation(t *testing.T) {
	rt := newTestKernel(t)

	ctx := context.Background()

	count := 0
	unsub, err := sdk.Subscribe[messages.ToolListResp](rt, ctx, func(resp messages.ToolListResp, msg messages.Message) {
		count++
	})
	require.NoError(t, err)

	// Cancel the subscription
	unsub()

	// Publish should still work (it goes to the router), but the cancelled
	// subscriber shouldn't receive anything
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, count, "cancelled subscriber should not receive messages")
}
