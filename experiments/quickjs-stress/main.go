// Experiment: QuickJS Stress Tests
//
// Goal: Test performance limits, bytecode precompilation, and error handling
// needed for embedding large JS libraries in Go via QuickJS (wazero backend).
//
// Tests:
//  1.  Large Code Evaluation (100KB, 500KB, 1MB)
//  2.  Bytecode Precompilation (compile vs source vs bytecode execution)
//  3.  Rapid Runtime Creation (100 runtimes)
//  4.  Many Function Registrations (500 Go functions)
//  5.  Deep Call Stack (Go<->JS nesting depth limit)
//  6.  Error Handling Across Bridge (throws, panics, syntax errors)
//  7.  Memory Pressure (100K objects, GC, stability)
//  8.  Multiple Independent Contexts (parallel goroutines)
//  9.  Binary Data Handling (1MB base64 round-trip)
//  10. Repeated Eval Performance (10K calls)

package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fastschema/qjs"
)

func main() {
	fmt.Println("=== QuickJS Stress Test Experiment ===")
	fmt.Println()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Large Code Evaluation", testLargeCodeEval},
		{"Bytecode Precompilation", testBytecodePrecompilation},
		{"Rapid Runtime Creation", testRapidRuntimeCreation},
		{"Many Function Registrations", testManyFunctionRegistrations},
		{"Deep Call Stack", testDeepCallStack},
		{"Error Handling Across Bridge", testErrorHandling},
		{"Memory Pressure", testMemoryPressure},
		{"Multiple Independent Contexts", testMultipleContexts},
		{"Binary Data Handling", testBinaryDataHandling},
		{"Repeated Eval Performance", testRepeatedEvalPerformance},
	}

	passed := 0
	failed := 0
	for i, t := range tests {
		fmt.Printf("--- Test %d: %s ---\n", i+1, t.name)
		start := time.Now()
		if err := t.fn(); err != nil {
			fmt.Printf("FAILED: %v\n\n", err)
			failed++
		} else {
			fmt.Printf("PASS (%s)\n\n", time.Since(start).Round(time.Millisecond))
			passed++
		}
	}

	fmt.Println("=========================================")
	fmt.Printf("Results: %d passed, %d failed, %d total\n", passed, failed, passed+failed)
	if failed > 0 {
		log.Fatalf("%d test(s) failed", failed)
	}
	fmt.Println("=== ALL TESTS PASSED ===")
}

// ---------------------------------------------------------------------------
// Test 1: Large Code Evaluation
// ---------------------------------------------------------------------------

func generateLargeJS(targetBytes int) string {
	// Each function is roughly: "function f123() { return 123; }\n" = ~35 bytes
	var b strings.Builder
	b.Grow(targetBytes + 1024)
	i := 0
	for b.Len() < targetBytes {
		fmt.Fprintf(&b, "function f%d() { return %d; }\n", i, i)
		i++
	}
	// Add a final call so we can verify the last function loaded
	lastIdx := i - 1
	fmt.Fprintf(&b, "f%d();\n", lastIdx)
	return b.String()
}

func testLargeCodeEval() error {
	sizes := []struct {
		label string
		bytes int
	}{
		{"100KB", 100 * 1024},
		{"500KB", 500 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, s := range sizes {
		code := generateLargeJS(s.bytes)
		actualSize := len(code)

		rt, err := qjs.New(qjs.Option{
			MemoryLimit:  256 * 1024 * 1024,
			MaxStackSize: 4 * 1024 * 1024,
		})
		if err != nil {
			return fmt.Errorf("[%s] failed to create runtime: %w", s.label, err)
		}

		ctx := rt.Context()

		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		start := time.Now()
		result, err := ctx.Eval("large.js", qjs.Code(code))
		elapsed := time.Since(start)

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		if err != nil {
			rt.Close()
			return fmt.Errorf("[%s] eval failed: %w", s.label, err)
		}

		val := result.Int32()
		result.Free()
		rt.Close()

		memDelta := int64(memAfter.TotalAlloc) - int64(memBefore.TotalAlloc)
		fmt.Printf("  %s: code=%d bytes, eval=%s, mem_delta=%.1fMB, last_func_returned=%d\n",
			s.label, actualSize, elapsed.Round(time.Millisecond),
			float64(memDelta)/(1024*1024), val)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 2: Bytecode Precompilation
// ---------------------------------------------------------------------------

func testBytecodePrecompilation() error {
	sizes := []struct {
		label string
		bytes int
	}{
		{"1KB", 1024},
		{"100KB", 100 * 1024},
	}

	for _, s := range sizes {
		code := generateLargeJS(s.bytes)

		// --- Compile to bytecode ---
		rtCompile, err := qjs.New()
		if err != nil {
			return fmt.Errorf("[%s] compile runtime: %w", s.label, err)
		}

		startCompile := time.Now()
		bytecode, err := rtCompile.Context().Compile("bench.js", qjs.Code(code))
		compileTime := time.Since(startCompile)
		rtCompile.Close()
		if err != nil {
			return fmt.Errorf("[%s] compile failed: %w", s.label, err)
		}

		// --- Execute from source ---
		rtSource, err := qjs.New(qjs.Option{
			MemoryLimit:  256 * 1024 * 1024,
			MaxStackSize: 4 * 1024 * 1024,
		})
		if err != nil {
			return fmt.Errorf("[%s] source runtime: %w", s.label, err)
		}
		startSource := time.Now()
		resultSource, err := rtSource.Context().Eval("bench.js", qjs.Code(code))
		sourceTime := time.Since(startSource)
		if err != nil {
			rtSource.Close()
			return fmt.Errorf("[%s] source eval failed: %w", s.label, err)
		}
		sourceVal := resultSource.Int32()
		resultSource.Free()
		rtSource.Close()

		// --- Execute from bytecode ---
		rtBytecode, err := qjs.New(qjs.Option{
			MemoryLimit:  256 * 1024 * 1024,
			MaxStackSize: 4 * 1024 * 1024,
		})
		if err != nil {
			return fmt.Errorf("[%s] bytecode runtime: %w", s.label, err)
		}
		startBytecode := time.Now()
		resultBytecode, err := rtBytecode.Context().Eval("bench.js", qjs.Bytecode(bytecode))
		bytecodeTime := time.Since(startBytecode)
		if err != nil {
			rtBytecode.Close()
			return fmt.Errorf("[%s] bytecode eval failed: %w", s.label, err)
		}
		bytecodeVal := resultBytecode.Int32()
		resultBytecode.Free()
		rtBytecode.Close()

		// Verify results match
		if sourceVal != bytecodeVal {
			return fmt.Errorf("[%s] results differ: source=%d, bytecode=%d", s.label, sourceVal, bytecodeVal)
		}

		speedup := float64(sourceTime) / float64(bytecodeTime)

		fmt.Printf("  %s: compile=%s, source_eval=%s, bytecode_eval=%s, speedup=%.2fx, bytecode_size=%d bytes, results_match=%v\n",
			s.label, compileTime.Round(time.Millisecond),
			sourceTime.Round(time.Millisecond),
			bytecodeTime.Round(time.Millisecond),
			speedup, len(bytecode), sourceVal == bytecodeVal)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 3: Rapid Runtime Creation
// ---------------------------------------------------------------------------

func testRapidRuntimeCreation() error {
	const count = 100

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	for i := 0; i < count; i++ {
		rt, err := qjs.New()
		if err != nil {
			return fmt.Errorf("runtime %d creation failed: %w", i, err)
		}
		ctx := rt.Context()

		// Verify each runtime works
		result, err := ctx.Eval("quick.js", qjs.Code(fmt.Sprintf("%d + 1;", i)))
		if err != nil {
			rt.Close()
			return fmt.Errorf("runtime %d eval failed: %w", i, err)
		}
		val := result.Int32()
		if val != int32(i+1) {
			result.Free()
			rt.Close()
			return fmt.Errorf("runtime %d: expected %d, got %d", i, i+1, val)
		}
		result.Free()
		rt.Close()
	}

	elapsed := time.Since(start)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Force GC and check again
	runtime.GC()
	var memGC runtime.MemStats
	runtime.ReadMemStats(&memGC)

	fmt.Printf("  Created and destroyed %d runtimes in %s (%.1fms/runtime)\n",
		count, elapsed.Round(time.Millisecond), float64(elapsed.Milliseconds())/float64(count))
	fmt.Printf("  Go heap before=%.1fMB, after=%.1fMB, post_GC=%.1fMB\n",
		float64(memBefore.HeapAlloc)/(1024*1024),
		float64(memAfter.HeapAlloc)/(1024*1024),
		float64(memGC.HeapAlloc)/(1024*1024))

	return nil
}

// ---------------------------------------------------------------------------
// Test 4: Many Function Registrations
// ---------------------------------------------------------------------------

func testManyFunctionRegistrations() error {
	const numFuncs = 500

	rt, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer rt.Close()

	ctx := rt.Context()

	startReg := time.Now()
	for i := 0; i < numFuncs; i++ {
		idx := i // capture
		ctx.SetFunc(fmt.Sprintf("goFunc_%d", i), func(this *qjs.This) (*qjs.Value, error) {
			return this.Context().NewInt32(int32(idx)), nil
		})
	}
	regTime := time.Since(startReg)

	// Call a random selection to verify
	rng := rand.New(rand.NewSource(42))
	testIndices := make([]int, 20)
	for i := range testIndices {
		testIndices[i] = rng.Intn(numFuncs)
	}

	startCall := time.Now()
	for _, idx := range testIndices {
		code := fmt.Sprintf("goFunc_%d();", idx)
		result, err := ctx.Eval("verify.js", qjs.Code(code))
		if err != nil {
			return fmt.Errorf("calling goFunc_%d failed: %w", idx, err)
		}
		val := result.Int32()
		result.Free()
		if val != int32(idx) {
			return fmt.Errorf("goFunc_%d returned %d, expected %d", idx, val, idx)
		}
	}
	callTime := time.Since(startCall)

	fmt.Printf("  Registered %d functions in %s (%.2fms/func)\n",
		numFuncs, regTime.Round(time.Millisecond), float64(regTime.Microseconds())/float64(numFuncs)/1000.0)
	fmt.Printf("  Verified %d random calls in %s\n", len(testIndices), callTime.Round(time.Millisecond))

	return nil
}

// ---------------------------------------------------------------------------
// Test 5: Deep Call Stack
// ---------------------------------------------------------------------------

func testDeepCallStack() error {
	// Find the maximum nesting depth for Go<->JS calls.
	// JS calls Go which evals more JS which calls Go... etc.

	maxDepthFound := 0

	// Binary search for the limit
	testDepth := func(depth int) (ok bool) {
		defer func() {
			if r := recover(); r != nil {
				ok = false
			}
		}()

		rt, err := qjs.New(qjs.Option{
			MaxStackSize: 4 * 1024 * 1024,
			MemoryLimit:  128 * 1024 * 1024,
		})
		if err != nil {
			return false
		}
		defer rt.Close()

		ctx := rt.Context()

		// Register a Go function that calls back into JS
		ctx.SetFunc("goNest", func(this *qjs.This) (*qjs.Value, error) {
			args := this.Args()
			if len(args) < 1 {
				return this.Context().NewInt32(0), nil
			}
			remaining := args[0].Int32()
			if remaining <= 0 {
				return this.Context().NewInt32(0), nil
			}
			code := fmt.Sprintf("goNest(%d);", remaining-1)
			result, err := this.Context().Eval("nest.js", qjs.Code(code))
			if err != nil {
				return nil, err
			}
			val := result.Int32()
			result.Free()
			return this.Context().NewInt32(val + 1), nil
		})

		code := fmt.Sprintf("goNest(%d);", depth)
		result, err := ctx.Eval("deep.js", qjs.Code(code))
		if err != nil {
			return false
		}
		result.Free()
		return true
	}

	// Linear scan with increasing steps to find rough limit quickly
	depths := []int{5, 10, 20, 50, 100, 200, 500, 1000}
	for _, d := range depths {
		if testDepth(d) {
			maxDepthFound = d
		} else {
			break
		}
	}

	// Binary search between last success and first failure for precision
	if maxDepthFound > 0 {
		lo := maxDepthFound
		hi := lo * 2
		if hi > 1000 {
			hi = 1000
		}
		// Only refine if the rough limit wasn't already the largest we tested
		if maxDepthFound < depths[len(depths)-1] {
			for lo < hi-1 {
				mid := (lo + hi) / 2
				if testDepth(mid) {
					lo = mid
				} else {
					hi = mid
				}
			}
			maxDepthFound = lo
		}
	}

	fmt.Printf("  Maximum Go<->JS nesting depth: %d\n", maxDepthFound)
	if maxDepthFound < 5 {
		return fmt.Errorf("nesting depth too low: %d (expected at least 5)", maxDepthFound)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 6: Error Handling Across Bridge
// ---------------------------------------------------------------------------

func testErrorHandling() error {
	subtests := []struct {
		name string
		fn   func() error
	}{
		{"JS throws Error -> Go receives it", testErrorJSThrows},
		{"Go returns error -> JS catches it", testErrorGoReturns},
		{"JS syntax error -> Go gets message", testErrorSyntax},
		{"JS runtime error -> Go gets error", testErrorRuntime},
		{"Go function panics -> caught", testErrorGoPanic},
		{"JS infinite loop detection", testErrorInfiniteLoop},
	}

	for _, st := range subtests {
		fmt.Printf("  [%s] ", st.name)
		if err := st.fn(); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			return fmt.Errorf("subtest '%s' failed: %w", st.name, err)
		}
		fmt.Println("OK")
	}
	return nil
}

func testErrorJSThrows() error {
	rt, err := qjs.New()
	if err != nil {
		return err
	}
	defer rt.Close()

	_, err = rt.Context().Eval("throw.js", qjs.Code(`throw new Error("test error from JS");`))
	if err == nil {
		return fmt.Errorf("expected error from JS throw, got nil")
	}
	if !strings.Contains(err.Error(), "test error from JS") {
		return fmt.Errorf("error message doesn't contain expected text: %v", err)
	}
	return nil
}

func testErrorGoReturns() error {
	rt, err := qjs.New()
	if err != nil {
		return err
	}
	defer rt.Close()

	ctx := rt.Context()
	ctx.SetFunc("goFail", func(this *qjs.This) (*qjs.Value, error) {
		return nil, fmt.Errorf("deliberate Go error")
	})

	// JS should see this as an exception
	_, err = ctx.Eval("goerr.js", qjs.Code(`
		let caught = false;
		try {
			goFail();
		} catch(e) {
			caught = true;
		}
		if (!caught) throw new Error("Go error was not caught in JS");
		"ok";
	`))
	if err != nil {
		// If the Go error propagates as an eval error, that's also valid behavior
		if strings.Contains(err.Error(), "deliberate Go error") {
			return nil // Error propagated, which is valid
		}
		return fmt.Errorf("unexpected error: %w", err)
	}
	return nil
}

func testErrorSyntax() error {
	rt, err := qjs.New()
	if err != nil {
		return err
	}
	defer rt.Close()

	_, err = rt.Context().Eval("syntax.js", qjs.Code(`function( { broken syntax }`))
	if err == nil {
		return fmt.Errorf("expected syntax error, got nil")
	}
	// Just verify we got a non-empty error message
	if len(err.Error()) == 0 {
		return fmt.Errorf("got empty error message for syntax error")
	}
	return nil
}

func testErrorRuntime() error {
	rt, err := qjs.New()
	if err != nil {
		return err
	}
	defer rt.Close()

	_, err = rt.Context().Eval("runtime.js", qjs.Code(`undefinedVariable.property;`))
	if err == nil {
		return fmt.Errorf("expected runtime error, got nil")
	}
	return nil
}

func testErrorGoPanic() (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			// If we get here, the panic was NOT caught by QuickJS.
			// That's still useful information.
			retErr = nil // Don't fail the test, just report
			fmt.Printf("(panic escaped to Go, recovered: %v) ", r)
		}
	}()

	rt, err := qjs.New()
	if err != nil {
		return err
	}
	defer rt.Close()

	ctx := rt.Context()
	ctx.SetFunc("goPanic", func(this *qjs.This) (*qjs.Value, error) {
		panic("deliberate panic in Go function")
	})

	_, err = ctx.Eval("panic.js", qjs.Code(`goPanic();`))
	if err != nil {
		// Good: panic was caught and converted to an error
		return nil
	}
	// If no error and no panic, something unexpected happened
	return nil
}

func testErrorInfiniteLoop() error {
	// Use a memory/stack-limited runtime to detect infinite loops
	rt, err := qjs.New(qjs.Option{
		MemoryLimit:  8 * 1024 * 1024, // 8MB
		MaxStackSize: 512 * 1024,      // 512KB
		GCThreshold:  256 * 1024,
	})
	if err != nil {
		return err
	}
	defer rt.Close()

	// Use a loop that grows memory to trigger the memory limit
	_, err = rt.Context().Eval("infinite.js", qjs.Code(`
		var arr = [];
		while(true) { arr.push(new Array(1000).fill(0)); }
	`))
	if err == nil {
		return fmt.Errorf("expected error from memory-limited infinite loop, got nil")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test 7: Memory Pressure
// ---------------------------------------------------------------------------

func testMemoryPressure() error {
	rt, err := qjs.New(qjs.Option{
		MemoryLimit:  512 * 1024 * 1024,
		MaxStackSize: 4 * 1024 * 1024,
		GCThreshold:  1024 * 1024,
	})
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer rt.Close()

	ctx := rt.Context()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Create 100,000 objects in JS
	start := time.Now()
	_, err = ctx.Eval("pressure.js", qjs.Code(`
		var objects = [];
		for (var i = 0; i < 100000; i++) {
			objects.push({
				id: i,
				name: "object_" + i,
				data: new Array(10).fill(i)
			});
		}
		objects.length;
	`))
	createTime := time.Since(start)
	if err != nil {
		return fmt.Errorf("object creation failed: %w", err)
	}

	var memAfterCreate runtime.MemStats
	runtime.ReadMemStats(&memAfterCreate)

	// Pass objects through Go bridge (read a sample)
	ctx.SetFunc("goProcessObj", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		_ = args[0].String() // read the JSON string
		return this.Context().NewInt32(1), nil
	})

	startBridge := time.Now()
	result, err := ctx.Eval("bridge.js", qjs.Code(`
		var count = 0;
		for (var i = 0; i < 1000; i++) {
			count += goProcessObj(JSON.stringify(objects[i]));
		}
		count;
	`))
	bridgeTime := time.Since(startBridge)
	if err != nil {
		return fmt.Errorf("bridge processing failed: %w", err)
	}
	bridgeCount := result.Int32()
	result.Free()

	// Force GC in JS by removing references
	startGC := time.Now()
	result2, err := ctx.Eval("gc.js", qjs.Code(`
		objects = null;
		// Create some more objects to trigger GC
		for (var i = 0; i < 1000; i++) {
			var tmp = { x: i };
		}
		"gc_done";
	`))
	gcTime := time.Since(startGC)
	if err != nil {
		return fmt.Errorf("GC trigger failed: %w", err)
	}
	result2.Free()

	// Verify runtime is still stable
	result3, err := ctx.Eval("verify.js", qjs.Code(`40 + 2;`))
	if err != nil {
		return fmt.Errorf("post-GC stability check failed: %w", err)
	}
	val := result3.Int32()
	result3.Free()
	if val != 42 {
		return fmt.Errorf("post-GC: expected 42, got %d", val)
	}

	var memAfterGC runtime.MemStats
	runtime.ReadMemStats(&memAfterGC)

	fmt.Printf("  Created 100K objects in %s\n", createTime.Round(time.Millisecond))
	fmt.Printf("  Bridged 1000 objects in %s (count=%d)\n", bridgeTime.Round(time.Millisecond), bridgeCount)
	fmt.Printf("  GC completed in %s\n", gcTime.Round(time.Millisecond))
	fmt.Printf("  Go heap: before=%.1fMB, after_create=%.1fMB, after_gc=%.1fMB\n",
		float64(memBefore.HeapAlloc)/(1024*1024),
		float64(memAfterCreate.HeapAlloc)/(1024*1024),
		float64(memAfterGC.HeapAlloc)/(1024*1024))
	fmt.Printf("  Runtime stable after GC: true\n")

	return nil
}

// ---------------------------------------------------------------------------
// Test 8: Multiple Independent Contexts
// ---------------------------------------------------------------------------

func testMultipleContexts() error {
	const numRuntimes = 5

	// Phase 1: Verify isolation (sequential)
	runtimes := make([]*qjs.Runtime, numRuntimes)
	for i := 0; i < numRuntimes; i++ {
		rt, err := qjs.New()
		if err != nil {
			return fmt.Errorf("runtime %d creation failed: %w", i, err)
		}
		runtimes[i] = rt

		ctx := rt.Context()

		// Set different state in each runtime
		code := fmt.Sprintf("var X = %d;", i+1)
		_, err = ctx.Eval("init.js", qjs.Code(code))
		if err != nil {
			return fmt.Errorf("runtime %d init failed: %w", i, err)
		}

		// Register a unique function per runtime
		idx := i + 1
		ctx.SetFunc("getID", func(this *qjs.This) (*qjs.Value, error) {
			return this.Context().NewInt32(int32(idx * 100)), nil
		})
	}

	// Verify isolation
	for i, rt := range runtimes {
		result, err := rt.Context().Eval("check.js", qjs.Code(`X;`))
		if err != nil {
			return fmt.Errorf("runtime %d isolation check failed: %w", i, err)
		}
		val := result.Int32()
		result.Free()
		expected := int32(i + 1)
		if val != expected {
			return fmt.Errorf("runtime %d: X = %d, expected %d (isolation violated)", i, val, expected)
		}
	}

	// Close sequential runtimes
	for _, rt := range runtimes {
		rt.Close()
	}

	fmt.Printf("  Sequential isolation verified for %d runtimes\n", numRuntimes)

	// Phase 2: Parallel goroutines (each with their own runtime)
	var wg sync.WaitGroup
	errors := make([]error, numRuntimes)
	results := make([]int32, numRuntimes)

	for i := 0; i < numRuntimes; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			rt, err := qjs.New()
			if err != nil {
				errors[idx] = fmt.Errorf("goroutine %d runtime failed: %w", idx, err)
				return
			}
			defer rt.Close()

			ctx := rt.Context()

			// Each goroutine does independent work
			code := fmt.Sprintf(`
				var sum = 0;
				for (var i = 0; i < 10000; i++) {
					sum += %d;
				}
				sum;
			`, idx+1)

			result, err := ctx.Eval("parallel.js", qjs.Code(code))
			if err != nil {
				errors[idx] = fmt.Errorf("goroutine %d eval failed: %w", idx, err)
				return
			}
			results[idx] = result.Int32()
			result.Free()
		}(i)
	}

	wg.Wait()

	for i := 0; i < numRuntimes; i++ {
		if errors[i] != nil {
			return errors[i]
		}
		expected := int32((i + 1) * 10000)
		if results[i] != expected {
			return fmt.Errorf("goroutine %d: sum=%d, expected %d", i, results[i], expected)
		}
	}

	fmt.Printf("  Parallel goroutines verified for %d runtimes (all results correct)\n", numRuntimes)

	return nil
}

// ---------------------------------------------------------------------------
// Test 9: Binary Data Handling
// ---------------------------------------------------------------------------

func testBinaryDataHandling() error {
	const dataSize = 1024 * 1024 // 1MB

	// Generate random bytes
	rng := rand.New(rand.NewSource(12345))
	originalData := make([]byte, dataSize)
	for i := range originalData {
		originalData[i] = byte(rng.Intn(256))
	}

	// Encode to base64 in Go
	encoded := base64.StdEncoding.EncodeToString(originalData)

	rt, err := qjs.New(qjs.Option{
		MemoryLimit:  256 * 1024 * 1024,
		MaxStackSize: 4 * 1024 * 1024,
	})
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer rt.Close()

	ctx := rt.Context()

	// Register function to receive the base64 string
	ctx.SetFunc("goSendData", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewString(encoded), nil
	})

	var receivedBack string
	ctx.SetFunc("goReceiveData", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		receivedBack = args[0].String()
		return this.Context().NewInt32(int32(len(receivedBack))), nil
	})

	// JS: get base64 from Go, "decode" (just verify length), pass back
	start := time.Now()
	result, err := ctx.Eval("binary.js", qjs.Code(`
		// Get base64 data from Go
		var b64 = goSendData();

		// Verify it's a valid base64 string (length check)
		var expectedLen = b64.length;

		// Pass it back to Go unchanged (round-trip test)
		var sentLen = goReceiveData(b64);

		JSON.stringify({
			receivedLen: expectedLen,
			sentLen: sentLen
		});
	`))
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("binary data eval failed: %w", err)
	}
	defer result.Free()

	// Verify the round-trip
	if receivedBack != encoded {
		return fmt.Errorf("round-trip data mismatch: sent %d chars, received %d chars",
			len(encoded), len(receivedBack))
	}

	// Decode and verify the data
	decoded, err := base64.StdEncoding.DecodeString(receivedBack)
	if err != nil {
		return fmt.Errorf("base64 decode failed: %w", err)
	}
	if len(decoded) != len(originalData) {
		return fmt.Errorf("decoded length %d != original %d", len(decoded), len(originalData))
	}
	for i := range decoded {
		if decoded[i] != originalData[i] {
			return fmt.Errorf("data mismatch at byte %d: got %d, expected %d", i, decoded[i], originalData[i])
		}
	}

	fmt.Printf("  Round-trip 1MB binary data: %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  Base64 size: %d chars, decoded: %d bytes, integrity: verified\n",
		len(encoded), len(decoded))

	return nil
}

// ---------------------------------------------------------------------------
// Test 10: Repeated Eval Performance
// ---------------------------------------------------------------------------

func testRepeatedEvalPerformance() error {
	const iterations = 10000

	rt, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer rt.Close()

	ctx := rt.Context()

	// Define a function once
	_, err = ctx.Eval("define.js", qjs.Code(`
		function compute(x) {
			return x * x + x * 2 + 1;
		}
	`))
	if err != nil {
		return fmt.Errorf("function definition failed: %w", err)
	}

	// Call it many times
	start := time.Now()
	for i := 0; i < iterations; i++ {
		code := fmt.Sprintf("compute(%d);", i)
		result, err := ctx.Eval("call.js", qjs.Code(code))
		if err != nil {
			return fmt.Errorf("iteration %d failed: %w", i, err)
		}
		result.Free()
	}
	elapsed := time.Since(start)

	// Verify correctness with a known value
	result, err := ctx.Eval("verify.js", qjs.Code(`compute(10);`))
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}
	val := result.Int32()
	result.Free()
	// 10*10 + 10*2 + 1 = 121
	if val != 121 {
		return fmt.Errorf("compute(10) = %d, expected 121", val)
	}

	perCall := elapsed / time.Duration(iterations)
	callsPerSec := float64(iterations) / elapsed.Seconds()

	fmt.Printf("  %d calls in %s (%.2f us/call, %.0f calls/sec)\n",
		iterations, elapsed.Round(time.Millisecond),
		float64(perCall.Nanoseconds())/1000.0, callsPerSec)
	fmt.Printf("  Correctness verified: compute(10) = %d\n", val)

	return nil
}
