// Package bench provides the BenchEnv abstraction for brainkit benchmarks.
// Each domain (bus, deploy, eval) exports a Run(b, env) function.
// Standalone _bench_test.go files create the env on the memory backend.
package bench

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
)

// BenchEnv is the shared benchmark environment.
// Mirrors suite.TestEnv but takes *testing.B instead of *testing.T.
type BenchEnv struct {
	Kit *brainkit.Kit
}

// NewEnv creates a BenchEnv with a fully configured kit for benchmarks.
// The kit is created once and reused across all sub-benchmarks.
// Caller must call b.Cleanup or defer env.Close().
func NewEnv(b *testing.B) *BenchEnv {
	b.Helper()
	tmpDir := b.TempDir()

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "bench",
		CallerID:  "bench",
		FSRoot:    tmpDir,
		Modules:   []brainkit.Module{toolsmod.New()},
	})
	if err != nil {
		b.Fatalf("bench.NewEnv: New: %v", err)
	}

	// Register echo tool (used by bus benchmarks).
	if err := k.Mount(context.Background(), toolsmod.GoTool("echo", toolsmod.TypedTool[echoInput]{
		Description: "echo",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})); err != nil {
		b.Fatalf("bench.NewEnv: mount echo tool: %v", err)
	}

	b.Cleanup(func() { k.Close() })

	return &BenchEnv{Kit: k}
}

type echoInput struct {
	Message string `json:"message"`
}
