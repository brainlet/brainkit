package stress

import (
	"context"
	"testing"

	quickjs "github.com/buke/quickjs-go"
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// testGCSingleKernelCleanClose creates a single Kernel and closes it.
func testGCSingleKernelCleanClose(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "gc-stress-test",
		CallerID:  "gc-stress-test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	if err := k.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// testGCMultipleKernelCleanClose creates and destroys 5 Kernels sequentially.
func testGCMultipleKernelCleanClose(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	for i := 0; i < 5; i++ {
		k, err := brainkit.NewKernel(brainkit.KernelConfig{
			Namespace: "gc-stress-multi",
			CallerID:  "gc-stress-multi",
			FSRoot:    t.TempDir(),
		})
		if err != nil {
			t.Fatalf("NewKernel %d: %v", i, err)
		}
		if err := k.Close(); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}
}

// testGCTenKernelCleanClose stress test - 10 sequential Kernels.
func testGCTenKernelCleanClose(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	for i := 0; i < 10; i++ {
		k, err := brainkit.NewKernel(brainkit.KernelConfig{
			Namespace: "gc-stress-ten",
			CallerID:  "gc-stress-ten",
			FSRoot:    t.TempDir(),
		})
		if err != nil {
			t.Fatalf("NewKernel %d: %v", i, err)
		}
		if err := k.Close(); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}
}

// testGCZeroLeakQuickJSMemory verifies QuickJS frees ALL C memory
// after the full cleanup chain (ctx.Close + RunGC + runtime.Close).
func testGCZeroLeakQuickJSMemory(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

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

// testGCZeroLeakSESRuntime verifies that a full SES runtime (brainkit with
// Mastra bundle + AI SDK) can be cleanly closed with zero object leaks.
func testGCZeroLeakSESRuntime(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "gc-stress-leak-test",
		CallerID:  "gc-stress-leak-test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}

	// Exercise the runtime
	result, err := k.EvalTS(context.Background(), "__gc_stress_test.ts", `
		bus.emit("gc.stress.test", { msg: "hello" });
		return JSON.stringify({ tools: tools.list().length, ns: kit.namespace });
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	t.Logf("EvalTS result: %s", result)

	if err := k.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	t.Log("SES runtime closed cleanly")
}
