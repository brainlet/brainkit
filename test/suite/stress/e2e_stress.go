package stress

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testE2EMultipleKernels — create 3 independent Kits, each deploys and works independently.
func testE2EMultipleKernels(t *testing.T, _ *suite.TestEnv) {
	kits := make([]*brainkit.Kit, 3)

	for i := 0; i < 3; i++ {
		tmpDir := t.TempDir()
		k, err := brainkit.New(brainkit.Config{
			Namespace: fmt.Sprintf("multi-stress-%d", i),
			CallerID:  fmt.Sprintf("multi-stress-%d", i),
			FSRoot:    tmpDir,
		})
		require.NoError(t, err)
		t.Cleanup(func() { k.Close() })

		type echoIn struct{ Message string `json:"message"` }
		brainkit.RegisterTool(k, fmt.Sprintf("echo-stress-%d", i), tools.TypedTool[echoIn]{
			Description: "echoes",
			Execute: func(ctx context.Context, in echoIn) (any, error) {
				return map[string]string{"echoed": in.Message}, nil
			},
		})

		kits[i] = k
	}

	for i, k := range kits {
		payload, ok := sendAndReceive(t, k,
			sdk.ToolCallMsg{Name: fmt.Sprintf("echo-stress-%d", i), Input: map[string]any{"message": fmt.Sprintf("kernel-%d", i)}},
			5*time.Second)
		require.True(t, ok, "kit %d didn't respond", i)
		assert.Contains(t, string(payload), fmt.Sprintf("kernel-%d", i))
	}
}

// testE2EConcurrentOperations — fire concurrent tool calls and verify all complete.
func testE2EConcurrentOperations(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const n = 3
	results := make(chan int, n)
	errors := make(chan error, n)

	for i := range n {
		go func(val int) {
			pubResult, err := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{
				Name:  "add",
				Input: map[string]any{"a": val, "b": val},
			})
			if err != nil {
				errors <- err
				return
			}
			done := make(chan sdk.ToolCallResp, 1)
			unsub, err := sdk.SubscribeTo[sdk.ToolCallResp](env.Kit, ctx, pubResult.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) {
				done <- r
			})
			if err != nil {
				errors <- err
				return
			}
			defer unsub()
			select {
			case resp := <-done:
				var result map[string]int
				json.Unmarshal(resp.Result, &result)
				results <- result["sum"]
			case <-ctx.Done():
				errors <- ctx.Err()
			}
		}(i)
	}

	sums := make(map[int]bool)
	for range n {
		select {
		case sum := <-results:
			sums[sum] = true
		case err := <-errors:
			t.Fatalf("concurrent call failed: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	}

	for i := range n {
		assert.True(t, sums[i*2], "should have sum %d", i*2)
	}
}
