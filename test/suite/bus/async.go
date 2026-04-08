package bus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCorrelationIDFiltering(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := sdk.Publish(env.Kit, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)
	assert.NotEmpty(t, result.CorrelationID, "Publish must return a correlationID")
	assert.NotEmpty(t, result.ReplyTo, "Publish must return a ReplyTo topic")

	received := make(chan sdk.ToolListResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.ToolListResp](env.Kit, ctx, result.ReplyTo, func(resp sdk.ToolListResp, msg sdk.Message) {
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

func testMultipleInFlight(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const n = 10
	var wg sync.WaitGroup
	results := make([]sdk.ToolListResp, n)
	errors := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pubResult, err := sdk.Publish(env.Kit, ctx, sdk.ToolListMsg{})
			if err != nil {
				errors[idx] = err
				return
			}
			done := make(chan sdk.ToolListResp, 1)
			unsub, err := sdk.SubscribeTo[sdk.ToolListResp](env.Kit, ctx, pubResult.ReplyTo, func(r sdk.ToolListResp, m sdk.Message) {
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

func testContextCancellation(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = sdk.Publish(env.Kit, ctx, sdk.ToolListMsg{})
}

func testSubscribeCancellation(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	count := 0
	unsub, err := sdk.SubscribeTo[sdk.ToolListResp](env.Kit, ctx, "tools.list.reply.test", func(resp sdk.ToolListResp, msg sdk.Message) {
		count++
	})
	require.NoError(t, err)

	unsub()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, count, "cancelled subscriber should not receive messages")
}
