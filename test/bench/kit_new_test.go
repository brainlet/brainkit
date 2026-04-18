package bench_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
)

// BenchmarkKitNew measures the cost of constructing and closing a
// fresh Kit on the memory transport. Baseline for "how expensive is
// a Kit" — bounded by QuickJS runtime init + bus router startup.
func BenchmarkKitNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tmp := b.TempDir()
		k, err := brainkit.New(brainkit.Config{
			Transport: brainkit.Memory(),
			Namespace: "bench-new",
			CallerID:  "bench",
			FSRoot:    tmp,
		})
		if err != nil {
			b.Fatalf("new: %v", err)
		}
		k.Close()
	}
}

// BenchmarkKitNewGoroutineDelta reports the net goroutine delta
// across New + Close. Surfaces goroutine leaks from subsystem
// init that don't clean up on Close. Reports a single iteration
// only — the metric is the leak delta, not throughput.
func BenchmarkKitNewGoroutineDelta(b *testing.B) {
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	for i := 0; i < b.N; i++ {
		tmp := b.TempDir()
		k, err := brainkit.New(brainkit.Config{
			Transport: brainkit.Memory(),
			Namespace: "bench-new-gr",
			CallerID:  "bench",
			FSRoot:    tmp,
		})
		if err != nil {
			b.Fatalf("new: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = k.Shutdown(ctx)
		cancel()
	}

	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	after := runtime.NumGoroutine()
	b.ReportMetric(float64(after-before), "goroutine-delta")
}
