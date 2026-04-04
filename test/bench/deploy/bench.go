// Package deploy provides deploy-domain benchmarks for brainkit.
package deploy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/bench"
)

// Run executes all deploy domain benchmarks against the given environment.
func Run(b *testing.B, env *bench.BenchEnv) {
	ctx := context.Background()
	k := env.Kernel

	b.Run("deploy_1KB", func(b *testing.B) {
		code := `bus.on("x", (msg) => msg.reply({}));`
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			source := fmt.Sprintf("bench-%d.ts", i)
			if _, err := k.Deploy(ctx, source, code); err != nil {
				b.Fatalf("deploy: %v", err)
			}
			k.Teardown(ctx, source)
		}
	})

	b.Run("deploy_10KB", func(b *testing.B) {
		code := `bus.on("x", (msg) => msg.reply({})); ` + strings.Repeat("// padding line\n", 500)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			source := fmt.Sprintf("bench10k-%d.ts", i)
			if _, err := k.Deploy(ctx, source, code); err != nil {
				b.Fatalf("deploy: %v", err)
			}
			k.Teardown(ctx, source)
		}
	})

	b.Run("restart_recovery", func(b *testing.B) {
		for _, n := range []int{10, 50} {
			b.Run(fmt.Sprintf("deployments=%d", n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					storePath := filepath.Join(b.TempDir(), "restart-bench.db")
					store, err := brainkit.NewSQLiteStore(storePath)
					if err != nil {
						b.Fatalf("open store: %v", err)
					}
					k2, err := brainkit.NewKernel(brainkit.KernelConfig{Store: store, Namespace: "bench", CallerID: "bench"})
					if err != nil {
						b.Fatalf("new kernel: %v", err)
					}
					for j := 0; j < n; j++ {
						if _, err := k2.Deploy(context.Background(), fmt.Sprintf("svc-%d.ts", j),
							`bus.on("x", (msg) => msg.reply({}));`); err != nil {
							b.Fatalf("deploy svc-%d: %v", j, err)
						}
					}
					k2.Close()

					b.StartTimer()
					store2, err := brainkit.NewSQLiteStore(storePath)
					if err != nil {
						b.Fatalf("open store2: %v", err)
					}
					k3, err := brainkit.NewKernel(brainkit.KernelConfig{Store: store2, Namespace: "bench", CallerID: "bench"})
					if err != nil {
						b.Fatalf("new kernel2: %v", err)
					}
					k3.Close()
				}
			})
		}
	})
}
