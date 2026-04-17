package bus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/bus/caller"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCallStreamAllDelivered — handler sends N chunks then reply; all
// chunks are forwarded through onChunk and the final reply is returned.
// Note: the memory transport does not serialize publishes across rapid
// successive calls — each chunk carries a `seq` field for consumers that
// need strict ordering.
func testCallStreamAllDelivered(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "stream-all.ts", `
		bus.on("tick", async (msg) => {
			for (var i = 0; i < 5; i++) msg.stream.text({ i: i });
			msg.reply({ total: 5 });
		});
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var got []int
	resp, err := brainkit.CallStream[sdk.CustomMsg, map[string]any, map[string]int](
		env.Kit, ctx, sdk.CustomMsg{Topic: "ts.stream-all.tick", Payload: []byte(`{}`)},
		func(chunk map[string]any) error {
			if d, ok := chunk["data"].(map[string]any); ok {
				if i, ok := d["i"].(float64); ok {
					got = append(got, int(i))
				}
			}
			return nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 5, resp["total"])
	assert.ElementsMatch(t, []int{0, 1, 2, 3, 4}, got, "all chunks must be delivered; got=%v", got)
}

// testCallStreamRequiresCallback — nil onChunk errors.
func testCallStreamRequiresCallback(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := brainkit.CallStream[sdk.CustomMsg, any, any](
		env.Kit, ctx, sdk.CustomMsg{Topic: "ts.none.x", Payload: []byte(`{}`)}, nil,
	)
	require.Error(t, err)
}

// testCallStreamBufferErrorPolicy — slow consumer + small buffer +
// BufferError policy → BufferOverflowError finalizes the call.
func testCallStreamBufferErrorPolicy(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "stream-berr.ts", `
		bus.on("flood", async (msg) => {
			for (var i = 0; i < 200; i++) msg.stream.text({ i: i });
			msg.reply({ done: true });
		});
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count int
	var mu sync.Mutex
	_, err := brainkit.CallStream[sdk.CustomMsg, map[string]any, map[string]bool](
		env.Kit, ctx, sdk.CustomMsg{Topic: "ts.stream-berr.flood", Payload: []byte(`{}`)},
		func(chunk map[string]any) error {
			// Slow consumer: sleep after each chunk so buffer fills.
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		},
		brainkit.WithCallBuffer(2),
		brainkit.WithCallBufferPolicy(brainkit.BufferError),
	)

	require.Error(t, err)
	var bo *caller.BufferOverflowError
	assert.True(t, errors.As(err, &bo), "want BufferOverflowError, got %T: %v", err, err)
	mu.Lock()
	t.Logf("chunks delivered before overflow: %d", count)
	mu.Unlock()
}

// testCallStreamDropNewest — slow consumer + DropNewest → some chunks
// drop but terminal still arrives and function returns the final reply.
func testCallStreamDropNewest(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "stream-drop.ts", `
		bus.on("flood", async (msg) => {
			for (var i = 0; i < 50; i++) msg.stream.text({ i: i });
			msg.reply({ ok: true });
		});
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var count int
	var mu sync.Mutex
	resp, err := brainkit.CallStream[sdk.CustomMsg, map[string]any, map[string]bool](
		env.Kit, ctx, sdk.CustomMsg{Topic: "ts.stream-drop.flood", Payload: []byte(`{}`)},
		func(chunk map[string]any) error {
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		},
		brainkit.WithCallBuffer(2),
		brainkit.WithCallBufferPolicy(brainkit.BufferDropNewest),
	)
	require.NoError(t, err)
	assert.True(t, resp["ok"])
	mu.Lock()
	delivered := count
	mu.Unlock()
	assert.Less(t, delivered, 50, "some chunks must have been dropped")
	assert.Greater(t, delivered, 0, "at least some chunks delivered")
	snap := env.Kit.Caller().Snapshot()
	t.Logf("delivered=%d dropped=%d total_metric_delivered=%d", delivered, 50-delivered, snap.ChunksDelivered)
}

// testCallStreamHandlerErrorAborts — onChunk returns non-nil error, call
// finalizes with that error and doesn't wait for terminal.
func testCallStreamHandlerErrorAborts(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "stream-abort.ts", `
		bus.on("flood", async (msg) => {
			for (var i = 0; i < 20; i++) {
				msg.stream.text({ i: i });
			}
			msg.reply({ ok: true });
		});
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sentinel := fmt.Errorf("abort after 3")
	seen := 0
	_, err := brainkit.CallStream[sdk.CustomMsg, map[string]any, map[string]bool](
		env.Kit, ctx, sdk.CustomMsg{Topic: "ts.stream-abort.flood", Payload: []byte(`{}`)},
		func(chunk map[string]any) error {
			seen++
			if seen == 3 {
				return sentinel
			}
			return nil
		},
		brainkit.WithCallBuffer(16),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, sentinel) || err.Error() == sentinel.Error(),
		"want handler sentinel, got %v", err)
}
