// Package bench provides the BenchEnv abstraction for brainkit benchmarks.
// Each domain (bus, deploy, eval) exports a Run(b, env) function.
// Standalone _bench_test.go files create the env on the memory backend.
package bench

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
)

// BenchEnv is the shared benchmark environment.
// Mirrors suite.TestEnv but takes *testing.B instead of *testing.T.
type BenchEnv struct {
	Kernel *brainkit.Kernel
}

// NewEnv creates a BenchEnv with a fully configured kernel for benchmarks.
// The kernel is created once and reused across all sub-benchmarks.
// Caller must call b.Cleanup or defer env.Close().
func NewEnv(b *testing.B) *BenchEnv {
	b.Helper()
	tmpDir := b.TempDir()

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "bench",
		CallerID:  "bench",
		FSRoot:    tmpDir,
	})
	if err != nil {
		b.Fatalf("bench.NewEnv: NewKernel: %v", err)
	}

	// Register echo tool (used by bus benchmarks).
	if err := brainkit.RegisterTool(k, "echo", registry.TypedTool[echoInput]{
		Description: "echo",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	}); err != nil {
		b.Fatalf("bench.NewEnv: register echo: %v", err)
	}

	b.Cleanup(func() { k.Close() })

	return &BenchEnv{Kernel: k}
}

type echoInput struct {
	Message string `json:"message"`
}
