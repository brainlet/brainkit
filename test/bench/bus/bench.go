// Package bus provides bus-domain benchmarks for brainkit.
package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/bench"
)

// Run executes all bus domain benchmarks against the given environment.
func Run(b *testing.B, env *bench.BenchEnv) {
	ctx := context.Background()
	k := env.Kit

	// Deploy a handler for roundtrip and pump benchmarks.
	if err := testutil.DeployErr(k, "bench-handler.ts", `bus.on("bench", (msg) => msg.reply({ ok: true }));`); err != nil {
		b.Fatalf("deploy bench-handler: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	b.Run("roundtrip", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pr, err := sdk.SendToService(k, ctx, "bench-handler.ts", "bench", map[string]bool{"x": true})
			if err != nil {
				b.Fatalf("send: %v", err)
			}
			ch := make(chan struct{}, 1)
			unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(_ messages.Message) { ch <- struct{}{} })
			if err != nil {
				b.Fatalf("subscribe: %v", err)
			}
			<-ch
			unsub()
		}
	})

	b.Run("tool_call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pr, err := sdk.Publish(k, ctx, messages.ToolCallMsg{
				Name:  "echo",
				Input: json.RawMessage(`{"message":"bench"}`),
			})
			if err != nil {
				b.Fatalf("publish: %v", err)
			}
			ch := make(chan struct{}, 1)
			unsub, err := sdk.SubscribeTo[messages.ToolCallResp](k, ctx, pr.ReplyTo, func(_ messages.ToolCallResp, _ messages.Message) {
				ch <- struct{}{}
			})
			if err != nil {
				b.Fatalf("subscribe: %v", err)
			}
			<-ch
			unsub()
		}
	})

	b.Run("pump_throughput", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pr, err := sdk.SendToService(k, ctx, "bench-handler.ts", "bench", map[string]bool{"x": true})
			if err != nil {
				b.Fatalf("send: %v", err)
			}
			ch := make(chan struct{}, 1)
			unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(_ messages.Message) { ch <- struct{}{} })
			if err != nil {
				b.Fatalf("subscribe: %v", err)
			}
			<-ch
			unsub()
		}
	})
}
