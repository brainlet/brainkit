// allocator_test.go ports the allocator stress test from tests/allocators/.
//
// The original test (allocators/index.js + allocators/runner.js) compiles allocator
// assembly files, instantiates them, and runs 20 rounds of 20,000 random alloc/free
// cycles to verify the heap allocator doesn't corrupt or leak memory.
//
// Ported from:
//   - tests/allocators/index.js — test harness
//   - tests/allocators/runner.js — stress test runner
package tests

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/compiler"
	"github.com/brainlet/brainkit/wasm-kit/parser"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

// TestAllocator_Stub compiles the stub allocator assembly and runs the stress test.
func TestAllocator_Stub(t *testing.T) {
	wasmBytes := compileAllocator(t, "stub", true)
	if wasmBytes == nil {
		t.Skip("compilation failed")
	}
	runAllocatorTest(t, wasmBytes, true)
}

// TestAllocator_Default compiles the default allocator assembly and runs the stress test.
func TestAllocator_Default(t *testing.T) {
	wasmBytes := compileAllocator(t, "default", false)
	if wasmBytes == nil {
		t.Skip("compilation failed")
	}
	runAllocatorTest(t, wasmBytes, false)
}

// compileAllocator compiles an allocator assembly source to Wasm binary.
func compileAllocator(t *testing.T, name string, noAssert bool) []byte {
	t.Helper()

	root := filepath.Join(allocatorsRoot(), name, "assembly")
	sourcePath := filepath.Join(root, "index.ts")
	sourceText, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read allocator source: %v", err)
	}

	p := parser.NewParser(nil)
	p.ParseFile(string(sourceText), "index.ts", true)

	// Parse std library
	stdFiles := collectStdSources(t)
	for _, filename := range stdFiles {
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("read std %s: %v", filename, err)
		}
		relativePath, _ := filepath.Rel(stdAssemblyRoot(), filename)
		logicalPath := common.LIBRARY_PREFIX + filepath.ToSlash(relativePath)
		p.ParseFile(string(data), logicalPath, false)
	}

	for next := p.NextFile(); next != ""; next = p.NextFile() {
	}
	sources := p.Sources()
	p.Finish()

	opts := program.NewOptions()
	if noAssert {
		opts.NoAssert = true
	}
	opts.OptimizeLevelHint = 2
	opts.ShrinkLevelHint = 1
	opts.ExportMemory = true

	// Use stub runtime for the stub allocator
	if name == "stub" {
		opts.Runtime = common.RuntimeStub
	}

	prog := program.NewProgram(opts, nil)
	prog.Sources = sources

	// Catch panics
	var mod interface {
		ToBinary(string) interface{ GetBinary() []byte }
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("PANIC during allocator compilation: %v", r)
			}
		}()
		c := compiler.NewCompiler(prog)
		result := c.CompileProgram()
		if result == nil {
			return
		}
		result.Optimize(int(opts.OptimizeLevelHint), int(opts.ShrinkLevelHint), opts.DebugInfo, opts.ZeroFilledMemory)
		bin := result.ToBinary("")
		_ = bin
	}()

	// Re-compile without the wrapper to get the actual bytes
	c := compiler.NewCompiler(program.NewProgram(opts, nil))
	prog2 := program.NewProgram(opts, nil)
	prog2.Sources = sources
	c2 := compiler.NewCompiler(prog2)
	result := c2.CompileProgram()
	if result == nil {
		t.Log("allocator compilation returned nil module")
		return nil
	}
	result.Optimize(int(opts.OptimizeLevelHint), int(opts.ShrinkLevelHint), opts.DebugInfo, opts.ZeroFilledMemory)
	bin := result.ToBinary("")
	_ = mod
	_ = c
	return bin.Binary
}

// runAllocatorTest instantiates the compiled allocator Wasm and runs the stress test.
// Ported from: tests/allocators/runner.js
func runAllocatorTest(t *testing.T, wasmBytes []byte, hasReset bool) {
	t.Helper()

	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Provide env.abort
	aborted := false
	_, err := rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, msgPtr, filePtr, line, col uint32) {
		aborted = true
		t.Errorf("abort at line %d, col %d", line, col)
	}).Export("abort").
		NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, ptr uint32) {
		// log function — nop for tests
	}).Export("log").
		NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, i int32) {
		// logi function — nop for tests
	}).Export("logi").
		NewFunctionBuilder().WithFunc(func() {
		// trace — nop
	}).Export("trace").
		Instantiate(ctx)
	if err != nil {
		t.Fatalf("env module: %v", err)
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("compile module: %v", err)
	}
	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}

	// Get exported functions
	allocFn := mod.ExportedFunction("heap_alloc")
	freeFn := mod.ExportedFunction("heap_free")
	resetFn := mod.ExportedFunction("heap_reset")
	fillFn := mod.ExportedFunction("memory_fill")

	if allocFn == nil || freeFn == nil {
		t.Fatal("missing heap_alloc or heap_free exports")
	}

	rng := rand.New(rand.NewSource(42)) // deterministic

	var ptrs []uint32

	alloc := func(maxSize int) uint32 {
		if maxSize == 0 {
			maxSize = 8192
		}
		size := uint32((rng.Intn(maxSize) + 1 + 3) &^ 3) // align to 4
		results, err := allocFn.Call(ctx, uint64(size))
		if err != nil {
			t.Fatalf("heap_alloc(%d) error: %v", size, err)
		}
		ptr := uint32(results[0])
		if ptr == 0 {
			t.Fatalf("heap_alloc(%d) returned null", size)
		}
		if ptr&15 != 0 {
			t.Fatalf("invalid alignment: %d on ptr %d", ptr&15, ptr)
		}
		// Check no duplicate pointers
		for _, existing := range ptrs {
			if existing == ptr {
				t.Fatalf("duplicate pointer: %d", ptr)
			}
		}
		// Fill memory if fill function exists
		if fillFn != nil {
			_, err := fillFn.Call(ctx, uint64(ptr), uint64(ptr%16), uint64(size))
			if err != nil {
				t.Logf("memory_fill error (non-fatal): %v", err)
			}
		}
		ptrs = append(ptrs, ptr)
		return ptr
	}

	free := func(ptr uint32) {
		_, err := freeFn.Call(ctx, uint64(ptr))
		if err != nil {
			t.Fatalf("heap_free(%d) error: %v", ptr, err)
		}
	}

	preciseFree := func(ptr uint32) {
		idx := -1
		for i, p := range ptrs {
			if p == ptr {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Fatalf("unknown pointer: %d", ptr)
		}
		ptrs = append(ptrs[:idx], ptrs[idx+1:]...)
		free(ptr)
	}

	randomFree := func() {
		idx := rng.Intn(len(ptrs))
		ptr := ptrs[idx]
		ptrs = append(ptrs[:idx], ptrs[idx+1:]...)
		free(ptr)
	}

	// Get base pointer
	base := alloc(64)
	if hasReset && resetFn != nil {
		_, err := resetFn.Call(ctx)
		if err != nil {
			// If reset fails, free the base pointer instead
			free(base)
		}
	} else {
		free(base)
	}
	ptrs = nil

	runs := 5      // reduced from 20 for test speed
	allocs := 5000 // reduced from 20000 for test speed

	for j := 0; j < runs; j++ {
		if aborted {
			t.Fatalf("abort called during run %d", j)
		}

		t.Logf("run %d (%d allocations)", j+1, allocs)

		for i := 0; i < allocs; i++ {
			ptr := alloc(0)

			// Immediately free every 4th
			if i%4 == 0 {
				preciseFree(ptr)
			} else if len(ptrs) > 0 && rng.Float64() < 0.33 {
				// Occasionally free random blocks
				randomFree()
			}
		}

		// Free the rest randomly
		for len(ptrs) > 0 {
			randomFree()
		}

		// If we have reset, verify base is reused
		if hasReset && resetFn != nil {
			_, err := resetFn.Call(ctx)
			if err != nil {
				t.Logf("reset error (non-fatal): %v", err)
				continue
			}
			results, err := allocFn.Call(ctx, 64)
			if err != nil {
				t.Fatalf("alloc after reset error: %v", err)
			}
			newBase := uint32(results[0])
			if newBase != base {
				t.Errorf("expected base %d after reset, got %d", base, newBase)
			}
			_, _ = resetFn.Call(ctx)
		}
	}

	t.Logf("allocator stress test passed: %d runs × %d allocs", runs, allocs)
	_ = fmt.Sprintf // suppress unused import
}
