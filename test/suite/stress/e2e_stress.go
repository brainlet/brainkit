package stress

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"

	"github.com/brainlet/brainkit"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
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
			Transport: brainkit.Memory(),
			Namespace: fmt.Sprintf("multi-stress-%d", i),
			CallerID:  fmt.Sprintf("multi-stress-%d", i),
			FSRoot:    tmpDir,
			Modules:   []bkmodule.Module{toolsmod.New()},
		})
		require.NoError(t, err)
		t.Cleanup(func() { k.Close() })

		type echoIn struct {
			Message string `json:"message"`
		}
		err = k.Mount(context.Background(), toolsmod.GoTool(fmt.Sprintf("echo-stress-%d", i), toolsmod.TypedTool[echoIn]{
			Description: "echoes",
			Execute: func(ctx context.Context, in echoIn) (any, error) {
				return map[string]string{"echoed": in.Message}, nil
			},
		}))
		require.NoError(t, err)

		kits[i] = k
	}

	for i, k := range kits {
		payload, ok := sendAndReceive(t, k,
			toolmsg.ToolCallMsg{Name: fmt.Sprintf("echo-stress-%d", i), Input: map[string]any{"message": fmt.Sprintf("kernel-%d", i)}},
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
			resp, err := toolmsg.CallToolCall(env.Kit, ctx, toolmsg.ToolCallMsg{
				Name:  "add",
				Input: map[string]any{"a": val, "b": val},
			}, sdk.WithCallTimeout(10*time.Second))
			if err != nil {
				errors <- err
				return
			}
			var result map[string]int
			json.Unmarshal(resp.Result, &result)
			results <- result["sum"]
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
