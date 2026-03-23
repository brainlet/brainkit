package test

import (
	"context"
	"sync"
	"encoding/json"
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

	// Publish a command — get the ReplyTo topic
	result, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
	require.NoError(t, err)
	assert.NotEmpty(t, result.CorrelationID, "Publish must return a correlationID")
	assert.NotEmpty(t, result.ReplyTo, "Publish must return a ReplyTo topic")

	// Subscribe to the reply topic
	received := make(chan messages.ToolListResp, 1)
	unsub, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, result.ReplyTo, func(resp messages.ToolListResp, msg messages.Message) {
		received <- resp
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case resp := <-received:
		assert.NotNil(t, resp.Tools)
	case <-ctx.Done():
		t.Fatal("timeout waiting for correlated response")
	}
}

func TestAsync_MultipleInFlight(t *testing.T) {
	rt := newTestKernel(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Fire 10 concurrent Publish calls — each gets its own ReplyTo topic.
	// No correlationID filtering needed — each subscriber listens on its own topic.
	const n = 10
	var wg sync.WaitGroup
	results := make([]messages.ToolListResp, n)
	errors := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pubResult, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
			if err != nil {
				errors[idx] = err
				return
			}
			done := make(chan messages.ToolListResp, 1)
			unsub, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, pubResult.ReplyTo, func(r messages.ToolListResp, m messages.Message) {
				done <- r
			})
			if err != nil {
				errors[idx] = err
				return
			}
			defer unsub()
			select {
			case results[idx] = <-done:
			case <-ctx.Done():
				errors[idx] = ctx.Err()
			}
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

	pr, _ := sdk.Publish(rt, ctx, messages.ToolListMsg{})
	errCh := make(chan string, 1)
	un, _ := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		var r struct { Error string `json:"error"` }
		json.Unmarshal(msg.Payload, &r)
		errCh <- r.Error
	})
	defer un()
	select {
	case errMsg := <-errCh:
		assert.NotEmpty(t, errMsg, "should fail with cancelled context")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestAsync_SubscribeCancellation(t *testing.T) {
	rt := newTestKernel(t)

	ctx := context.Background()

	count := 0
	unsub, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, "tools.list.reply.test", func(resp messages.ToolListResp, msg messages.Message) {
		count++
	})
	require.NoError(t, err)

	// Cancel the subscription
	unsub()

	// After cancellation, nothing should be received
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, count, "cancelled subscriber should not receive messages")
}
