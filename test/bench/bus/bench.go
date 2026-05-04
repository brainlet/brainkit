// Package bus provides bus-domain benchmarks for brainkit.
package bus

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
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
			_, err := sdk.Call[sdk.CustomMsg, json.RawMessage](k, ctx, sdk.CustomMsg{
				Topic:   benchServiceTopic("bench-handler.ts", "bench"),
				Payload: json.RawMessage(`{"x":true}`),
			})
			if err != nil {
				b.Fatalf("call: %v", err)
			}
		}
	})

	b.Run("tool_call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := toolmsg.CallToolCall(k, ctx, toolmsg.ToolCallMsg{
				Name:  "echo",
				Input: json.RawMessage(`{"message":"bench"}`),
			})
			if err != nil {
				b.Fatalf("call: %v", err)
			}
		}
	})

	b.Run("pump_throughput", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sdk.Call[sdk.CustomMsg, json.RawMessage](k, ctx, sdk.CustomMsg{
				Topic:   benchServiceTopic("bench-handler.ts", "bench"),
				Payload: json.RawMessage(`{"x":true}`),
			})
			if err != nil {
				b.Fatalf("call: %v", err)
			}
		}
	})
}

func benchServiceTopic(source, topic string) string {
	name := strings.TrimSuffix(source, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
