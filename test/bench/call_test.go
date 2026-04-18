package bench_test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

type callBenchEnv struct {
	Kit *brainkit.Kit
}

func callBenchKit(b *testing.B) *callBenchEnv {
	b.Helper()
	return &callBenchEnv{Kit: benchKit(b)}
}

// BenchmarkCall measures the shared-inbox Caller round trip for a
// tools.call request answered by an in-process echo tool. Covers:
// serialization + publish + inbox subscribe + correlation demux +
// reply decode.
func BenchmarkCall(b *testing.B) {
	env := callBenchKit(b)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := sdk.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "ping"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := brainkit.Call[sdk.ToolCallMsg, sdk.ToolCallResp](
			env.Kit, ctx, msg,
			brainkit.WithCallTimeout(5*time.Second),
		)
		if err != nil {
			b.Fatalf("call: %v", err)
		}
	}
}

// BenchmarkCallParallel measures the same path under concurrent
// load — exercises pending-map contention on the shared Caller.
func BenchmarkCallParallel(b *testing.B) {
	env := callBenchKit(b)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := sdk.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "ping"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := brainkit.Call[sdk.ToolCallMsg, sdk.ToolCallResp](
				env.Kit, ctx, msg,
				brainkit.WithCallTimeout(5*time.Second),
			)
			if err != nil {
				b.Errorf("call: %v", err)
				return
			}
		}
	})
}
