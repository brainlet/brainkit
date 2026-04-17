// Package deploy provides deploy-domain benchmarks for brainkit.
package deploy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/bench"
)

// Run executes all deploy domain benchmarks against the given environment.
func Run(b *testing.B, env *bench.BenchEnv) {
	k := env.Kit

	b.Run("deploy_1KB", func(b *testing.B) {
		code := `bus.on("x", (msg) => msg.reply({}));`
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			source := fmt.Sprintf("bench-%d.ts", i)
			if err := testutil.DeployErr(k, source, code); err != nil {
				b.Fatalf("deploy: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			sdk.Publish(k, ctx, sdk.PackageTeardownMsg{Name: strings.TrimSuffix(source, ".ts")})
			cancel()
		}
	})

	b.Run("deploy_10KB", func(b *testing.B) {
		code := `bus.on("x", (msg) => msg.reply({})); ` + strings.Repeat("// padding line\n", 500)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			source := fmt.Sprintf("bench10k-%d.ts", i)
			if err := testutil.DeployErr(k, source, code); err != nil {
				b.Fatalf("deploy: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			sdk.Publish(k, ctx, sdk.PackageTeardownMsg{Name: strings.TrimSuffix(source, ".ts")})
			cancel()
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
					k2, err := brainkit.New(brainkit.Config{Transport: brainkit.Memory(), Store: store, Namespace: "bench", CallerID: "bench"})
					if err != nil {
						b.Fatalf("new kit: %v", err)
					}
					for j := 0; j < n; j++ {
						if err := testutil.DeployErr(k2, fmt.Sprintf("svc-%d.ts", j),
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
					k3, err := brainkit.New(brainkit.Config{Transport: brainkit.Memory(), Store: store2, Namespace: "bench", CallerID: "bench"})
					if err != nil {
						b.Fatalf("new kit2: %v", err)
					}
					k3.Close()
				}
			})
		}
	})
}
