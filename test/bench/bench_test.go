package bench_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk/protocol"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/stores"
)

func benchTeardownName(source string) string {
	return strings.TrimSuffix(source, ".ts")
}

func BenchmarkDeploy_1KB(b *testing.B) {
	k := benchKit(b)
	code := `bus.on("x", (msg) => msg.reply({}));`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source := fmt.Sprintf("bench-%d.ts", i)
		testutil.DeployErr(k, source, code)
		// Teardown via bus — fire and forget for bench
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		protocol.Publish(k, ctx, packagemsg.PackageTeardownMsg{Name: benchTeardownName(source)})
		cancel()
	}
}

func BenchmarkDeploy_10KB(b *testing.B) {
	k := benchKit(b)
	code := `bus.on("x", (msg) => msg.reply({})); ` + strings.Repeat("// padding line\n", 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source := fmt.Sprintf("bench10k-%d.ts", i)
		testutil.DeployErr(k, source, code)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		protocol.Publish(k, ctx, packagemsg.PackageTeardownMsg{Name: benchTeardownName(source)})
		cancel()
	}
}

func BenchmarkEvalTS_Trivial(b *testing.B) {
	k := benchKit(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testutil.EvalTSErr(k, "bench.ts", `return "ok"`)
	}
}

func BenchmarkEvalTS_JSONParse(b *testing.B) {
	k := benchKit(b)
	payload := `{"key":"` + strings.Repeat("x", 1000) + `"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testutil.EvalTSErr(k, "bench.ts", fmt.Sprintf(`return JSON.stringify(JSON.parse('%s'))`, payload))
	}
}

func BenchmarkBusRoundtrip(b *testing.B) {
	k := benchKit(b)
	ctx := context.Background()

	testutil.DeployErr(k, "bench-handler.ts", `bus.on("bench", (msg) => msg.reply({ ok: true }));`)
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr, _ := protocol.SendToService(k, ctx, "bench-handler.ts", "bench", map[string]bool{"x": true})
		ch := make(chan struct{}, 1)
		unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(_ sdk.Message) { ch <- struct{}{} })
		<-ch
		unsub()
	}
}

func BenchmarkToolCall(b *testing.B) {
	k := benchKit(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr, _ := protocol.Publish(k, ctx, toolmsg.ToolCallMsg{
			Name:  "echo",
			Input: json.RawMessage(`{"message":"bench"}`),
		})
		ch := make(chan struct{}, 1)
		unsub, _ := sdk.SubscribeTo[toolmsg.ToolCallResp](k, ctx, pr.ReplyTo, func(_ toolmsg.ToolCallResp, _ sdk.Message) {
			ch <- struct{}{}
		})
		<-ch
		unsub()
	}
}

func BenchmarkPumpThroughput(b *testing.B) {
	k := benchKit(b)
	ctx := context.Background()

	testutil.DeployErr(k, "pump-bench.ts", `bus.on("pump", (msg) => msg.reply({ ok: true }));`)
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr, _ := protocol.SendToService(k, ctx, "pump-bench.ts", "pump", map[string]bool{"x": true})
		ch := make(chan struct{}, 1)
		unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(_ sdk.Message) { ch <- struct{}{} })
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
				store, _ := stores.NewSQLite(storePath)
				k, _ := brainkit.New(brainkit.Config{Transport: brainkit.Memory(), Store: store, Namespace: "bench", CallerID: "bench"})
				for j := 0; j < n; j++ {
					testutil.DeployErr(k, fmt.Sprintf("svc-%d.ts", j),
						`bus.on("x", (msg) => msg.reply({}));`)
				}
				k.Close()

				b.StartTimer()
				store2, _ := stores.NewSQLite(storePath)
				k2, _ := brainkit.New(brainkit.Config{Transport: brainkit.Memory(), Store: store2, Namespace: "bench", CallerID: "bench"})
				k2.Close()
			}
		})
	}
}

// benchKit creates a minimal Kit for benchmarks.
func benchKit(b *testing.B) *brainkit.Kit {
	b.Helper()
	tmpDir := b.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "bench",
		CallerID:  "bench",
		FSRoot:    tmpDir,
		Modules:   []bkmodule.Module{toolsmod.New()},
	})
	if err != nil {
		b.Fatalf("benchKit: %v", err)
	}

	type echoIn struct {
		Message string `json:"message"`
	}
	if err := k.Mount(context.Background(), toolsmod.GoTool("echo", toolsmod.TypedTool[echoIn]{
		Description: "echo",
		Execute: func(ctx context.Context, input echoIn) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})); err != nil {
		b.Fatalf("benchKit mount echo tool: %v", err)
	}

	b.Cleanup(func() { k.Close() })
	return k
}
