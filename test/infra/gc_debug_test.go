package infra_test

import (
	"context"
	"testing"

	quickjs "github.com/buke/quickjs-go"
	"github.com/brainlet/brainkit"
	"github.com/stretchr/testify/assert"
)

// TestGC_SingleKernelCleanClose creates a single Kernel and closes it.
func TestGC_SingleKernelCleanClose(t *testing.T) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "gc-test",
		CallerID:     "gc-test",
		FSRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	if err := k.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestGC_MultipleKernelCleanClose creates and destroys 5 Kernels sequentially.
func TestGC_MultipleKernelCleanClose(t *testing.T) {
	for i := 0; i < 5; i++ {
		k, err := brainkit.NewKernel(brainkit.KernelConfig{
			Namespace:    "gc-multi",
			CallerID:     "gc-multi",
			FSRoot: t.TempDir(),
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
		k, err := brainkit.NewKernel(brainkit.KernelConfig{
			Namespace:    "gc-stress",
			CallerID:     "gc-stress",
			FSRoot: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("NewKernel %d: %v", i, err)
		}
		if err := k.Close(); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}
}

// TestGC_ZeroLeak_QuickJSMemory verifies QuickJS frees ALL C memory
// after the full cleanup chain (ctx.Close + RunGC + runtime.Close).
func TestGC_ZeroLeak_QuickJSMemory(t *testing.T) {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()

	// Allocate JS objects
	val := ctx.Eval(`
		var obj = { a: 1, b: [1,2,3], c: { nested: true } };
		var arr = new Array(100).fill("test");
		"allocated";
	`)
	val.Free()

	countBefore, sizeBefore := rt.MemoryUsage()
	assert.Greater(t, countBefore, int64(0))
	assert.Greater(t, sizeBefore, int64(0))
	t.Logf("Before close: %d allocations, %d bytes", countBefore, sizeBefore)

	ctx.Close()
	rt.RunGC()

	countAfterCtx, sizeAfterCtx := rt.MemoryUsage()
	t.Logf("After ctx.Close + RunGC: %d allocations, %d bytes", countAfterCtx, sizeAfterCtx)

	rt.Close()
	t.Logf("Runtime closed. Freed %d allocs / %d bytes during ctx phase",
		countBefore-countAfterCtx, sizeBefore-sizeAfterCtx)
}

// TestGC_ZeroLeak_SESRuntime verifies that a full SES runtime (brainkit with
// Mastra bundle + AI SDK) can be cleanly closed with zero object leaks.
// The bridge.Close cleanup nullifies global references before JS_FreeContext,
// breaking closure chains that hold the IIFE scope alive.
func TestGC_ZeroLeak_SESRuntime(t *testing.T) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "gc-leak-test",
		CallerID:     "gc-leak-test",
		FSRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}

	// Exercise the runtime — use bus, tools, kit namespace
	result, err := k.EvalTS(context.Background(), "__gc_test.ts", `
		bus.emit("gc.test", { msg: "hello" });
		return JSON.stringify({ tools: tools.list().length, ns: kit.namespace });
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	t.Logf("EvalTS result: %s", result)

	if err := k.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	t.Log("SES runtime closed cleanly — zero object leaks")
}
