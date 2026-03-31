package bench_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

func BenchmarkDeploy_1KB(b *testing.B) {
	k := benchKernel(b)
	ctx := context.Background()
	code := `bus.on("x", (msg) => msg.reply({}));`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source := fmt.Sprintf("bench-%d.ts", i)
		k.Deploy(ctx, source, code)
		k.Teardown(ctx, source)
	}
}

func BenchmarkDeploy_10KB(b *testing.B) {
	k := benchKernel(b)
	ctx := context.Background()
	code := `bus.on("x", (msg) => msg.reply({})); ` + strings.Repeat("// padding line\n", 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source := fmt.Sprintf("bench10k-%d.ts", i)
		k.Deploy(ctx, source, code)
		k.Teardown(ctx, source)
	}
}

func BenchmarkEvalTS_Trivial(b *testing.B) {
	k := benchKernel(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k.EvalTS(ctx, "bench.ts", `return "ok"`)
	}
}

func BenchmarkEvalTS_JSONParse(b *testing.B) {
	k := benchKernel(b)
	ctx := context.Background()
	payload := `{"key":"` + strings.Repeat("x", 1000) + `"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k.EvalTS(ctx, "bench.ts", fmt.Sprintf(`return JSON.stringify(JSON.parse('%s'))`, payload))
	}
}

func BenchmarkBusRoundtrip(b *testing.B) {
	k := benchKernel(b)
	ctx := context.Background()

	k.Deploy(ctx, "bench-handler.ts", `bus.on("bench", (msg) => msg.reply({ ok: true }));`)
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr, _ := sdk.SendToService(k, ctx, "bench-handler.ts", "bench", map[string]bool{"x": true})
		ch := make(chan struct{}, 1)
		unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(_ messages.Message) { ch <- struct{}{} })
		<-ch
		unsub()
	}
}

func BenchmarkToolCall(b *testing.B) {
	k := benchKernel(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr, _ := sdk.Publish(k, ctx, messages.ToolCallMsg{
			Name:  "echo",
			Input: json.RawMessage(`{"message":"bench"}`),
		})
		ch := make(chan struct{}, 1)
		unsub, _ := sdk.SubscribeTo[messages.ToolCallResp](k, ctx, pr.ReplyTo, func(_ messages.ToolCallResp, _ messages.Message) {
			ch <- struct{}{}
		})
		<-ch
		unsub()
	}
}

func BenchmarkPumpThroughput(b *testing.B) {
	k := benchKernel(b)
	ctx := context.Background()

	k.Deploy(ctx, "pump-bench.ts", `bus.on("pump", (msg) => msg.reply({ ok: true }));`)
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr, _ := sdk.SendToService(k, ctx, "pump-bench.ts", "pump", map[string]bool{"x": true})
		ch := make(chan struct{}, 1)
		unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(_ messages.Message) { ch <- struct{}{} })
		<-ch
		unsub()
	}
}

func BenchmarkRestartRecovery(b *testing.B) {
	for _, n := range []int{10, 50} {
		b.Run(fmt.Sprintf("deployments=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				storePath := filepath.Join(b.TempDir(), "restart-bench.db")
				store, _ := kit.NewSQLiteStore(storePath)
				k, _ := kit.NewKernel(kit.KernelConfig{Store: store, Namespace: "bench", CallerID: "bench"})
				for j := 0; j < n; j++ {
					k.Deploy(context.Background(), fmt.Sprintf("svc-%d.ts", j),
						`bus.on("x", (msg) => msg.reply({}));`)
				}
				k.Close()

				b.StartTimer()
				store2, _ := kit.NewSQLiteStore(storePath)
				k2, _ := kit.NewKernel(kit.KernelConfig{Store: store2, Namespace: "bench", CallerID: "bench"})
				k2.Close()
			}
		})
	}
}

// benchKernel creates a minimal Kernel for benchmarks.
// Does NOT use testutil.NewTestKernelFull because it takes *testing.T.
func benchKernel(b *testing.B) *kit.Kernel {
	b.Helper()
	tmpDir := b.TempDir()
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "bench",
		CallerID:  "bench",
		FSRoot:    tmpDir,
	})
	if err != nil {
		b.Fatalf("benchKernel: %v", err)
	}

	// Register echo tool for BenchmarkToolCall
	kit.RegisterTool(k, "echo", struct {
		Description string
		Execute     func(ctx context.Context, input struct{ Message string }) (any, error)
	}{
		Description: "echo",
		Execute: func(ctx context.Context, input struct{ Message string }) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})

	b.Cleanup(func() { k.Close() })
	return k
}
