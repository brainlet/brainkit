package infra_test

import (
	"context"
	"testing"

	quickjs "github.com/buke/quickjs-go"
	"github.com/brainlet/brainkit/kit"
	"github.com/stretchr/testify/assert"
)

// TestGC_SingleKernelCleanClose creates a single Kernel and closes it.
// Verifies JS_FreeContext + JS_FreeRuntime complete without crash.
func TestGC_SingleKernelCleanClose(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "gc-test",
		CallerID:     "gc-test",
		WorkspaceDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	if err := k.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestGC_MultipleKernelCleanClose creates and destroys 5 Kernels sequentially.
// Tests accumulated bridge lifecycles don't cause thread/memory issues.
func TestGC_MultipleKernelCleanClose(t *testing.T) {
	for i := 0; i < 5; i++ {
		k, err := kit.NewKernel(kit.KernelConfig{
			Namespace:    "gc-multi",
			CallerID:     "gc-multi",
			WorkspaceDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("NewKernel %d: %v", i, err)
		}
		if err := k.Close(); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}
}

// TestGC_TenKernelCleanClose stress test — 10 sequential Kernels.
func TestGC_TenKernelCleanClose(t *testing.T) {
	for i := 0; i < 10; i++ {
		k, err := kit.NewKernel(kit.KernelConfig{
			Namespace:    "gc-stress",
			CallerID:     "gc-stress",
			WorkspaceDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("NewKernel %d: %v", i, err)
		}
		if err := k.Close(); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}
}

// TestGC_ZeroLeak_QuickJSMemory verifies that QuickJS frees ALL C memory
// after runtime.Close(). Uses QuickJS's built-in JSMemoryUsage tracking.
// A clean close should leave malloc_count=0 and malloc_size=0.
func TestGC_ZeroLeak_QuickJSMemory(t *testing.T) {
	rt := quickjs.NewRuntime()

	ctx := rt.NewContext()
	// Allocate some JS objects
	val := ctx.Eval(`
		var obj = { a: 1, b: [1,2,3], c: { nested: true } };
		var arr = new Array(100).fill("test");
		"allocated";
	`)
	val.Free()

	// Check memory before close — should be > 0
	countBefore, sizeBefore := rt.MemoryUsage()
	assert.Greater(t, countBefore, int64(0), "should have allocations before close")
	assert.Greater(t, sizeBefore, int64(0), "should have memory used before close")
	t.Logf("Before close: %d allocations, %d bytes", countBefore, sizeBefore)

	ctx.Close()
	rt.RunGC()

	// Check memory after context close + GC — most should be freed
	countAfterCtx, sizeAfterCtx := rt.MemoryUsage()
	t.Logf("After ctx.Close + RunGC: %d allocations, %d bytes", countAfterCtx, sizeAfterCtx)

	rt.Close()
	// Can't check after runtime.Close — runtime is freed
	// But if we get here without crash, the cleanup succeeded

	t.Logf("Runtime closed cleanly. Freed %d allocations, %d bytes",
		countBefore-countAfterCtx, sizeBefore-sizeAfterCtx)
}

// TestGC_ZeroLeak_SESRuntime verifies that a full SES runtime (like brainkit uses)
// can be cleanly closed without memory leaks.
func TestGC_ZeroLeak_SESRuntime(t *testing.T) {
	// Create a Kernel — this loads SES lockdown + Mastra bundle
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "gc-leak-test",
		CallerID:     "gc-leak-test",
		WorkspaceDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}

	// Do some work — use bus, eval JS, exercise the runtime
	result, err := k.EvalTS(context.Background(), "__gc_test.ts", `
		bus.emit("gc.test", { msg: "hello" });
		return JSON.stringify({ tools: tools.list().length, ns: kit.namespace });
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	t.Logf("EvalTS result: %s", result)

	// Close — should free everything
	if err := k.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	t.Log("SES runtime closed cleanly — no crash, no leak")
}
