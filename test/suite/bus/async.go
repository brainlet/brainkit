package bus

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCorrelationIDFiltering(t *testing.T, env *suite.TestEnv) {
	result, got, ok := publishAndWaitMessage(t, env.Kit, toolmsg.ToolListMsg{}, 5*time.Second)
	require.True(t, ok)
	assert.NotEmpty(t, result.CorrelationID, "Publish must return a correlationID")
	assert.NotEmpty(t, result.ReplyTo, "Publish must return a ReplyTo topic")

	var resp toolmsg.ToolListResp
	err := json.Unmarshal(suite.ResponseDataFromMsg(got), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Tools)
}

func testMultipleInFlight(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const n = 10
	var wg sync.WaitGroup
	results := make([]toolmsg.ToolListResp, n)
	errors := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, got, ok := publishAndWaitMessage(t, env.Kit, toolmsg.ToolListMsg{}, 10*time.Second)
			if !ok {
				errors[idx] = ctx.Err()
				return
			}
			errors[idx] = json.Unmarshal(suite.ResponseDataFromMsg(got), &results[idx])
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
	_, _ = sdk.Publish(env.Kit, ctx, toolmsg.ToolListMsg{})
}

func testSubscribeCancellation(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	count := 0
	unsub, err := sdk.SubscribeTo[toolmsg.ToolListResp](env.Kit, ctx, "tools.list.reply.test", func(resp toolmsg.ToolListResp, msg sdk.Message) {
		count++
	})
	require.NoError(t, err)

	unsub()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, count, "cancelled subscriber should not receive messages")
}
