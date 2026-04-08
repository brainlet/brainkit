// Package eval provides eval-domain benchmarks for brainkit.
package eval

import (
	"fmt"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/bench"
)

// Run executes all eval domain benchmarks against the given environment.
func Run(b *testing.B, env *bench.BenchEnv) {
	k := env.Kit

	b.Run("trivial", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := testutil.EvalTSErr(k, "bench.ts", `return "ok"`); err != nil {
				b.Fatalf("eval: %v", err)
			}
		}
	})

	b.Run("json_parse", func(b *testing.B) {
		payload := `{"key":"` + strings.Repeat("x", 1000) + `"}`
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := testutil.EvalTSErr(k, "bench.ts", fmt.Sprintf(`return JSON.stringify(JSON.parse('%s'))`, payload)); err != nil {
				b.Fatalf("eval: %v", err)
			}
		}
	})
}
