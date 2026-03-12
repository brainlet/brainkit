// Experiment: QuickJS-Binaryen Memory Bridge
//
// Goal: Prove that the Binaryen memory bridge works — the emulated linear
// memory approach where JS (in QuickJS) reads/writes a Go []byte, and Go
// reads from that []byte when calling actual Binaryen C functions via CGo.
//
// This is the last HIGH risk item for the QuickJS embedding architecture.
//
// Architecture:
//   JS (in QuickJS)                 Go                          C (libbinaryen)
//     |                              |                            |
//     | _malloc(size) ──────────────>| allocate in Go []byte      |
//     | __i32_store(ptr, val) ──────>| write to Go []byte         |
//     | _BinaryenBlock(...) ────────>| read from Go []byte ──────>| BinaryenBlock()
//     |<────────── return handle ────| (handle is just an int)    |
//
// Tests:
//   1. Module Create/Dispose
//   2. Simple Const Expression
//   3. String Passing
//   4. Array Passing (Block with children)
//   5. Full Wasm Module (function returning 42)
//   6. Add/Return Expression (function adding two params)
//   7. Memory Operations Performance

package main

/*
#cgo CFLAGS: -I/Users/davidroman/Documents/code/clones/binaryen/src
#cgo LDFLAGS: -L/Users/davidroman/Documents/code/clones/binaryen/build/lib -lbinaryen -lstdc++ -lm
#include "binaryen-c.h"
#include <string.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/fastschema/qjs"
)

// lm is the global linear memory shared between all bridge functions.
var lm *LinearMemory

func main() {
	fmt.Println("=== QuickJS-Binaryen Memory Bridge Experiment ===")
	fmt.Println()
	fmt.Printf("BinaryenLiteral size: %d bytes\n", binaryenSizeofLiteral())
	fmt.Printf("Linear memory size: 64MB\n")
	fmt.Println()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Module Create/Dispose", testModuleCreateDispose},
		{"Simple Const Expression", testSimpleConst},
		{"String Passing", testStringPassing},
		{"Array Passing (Block with children)", testArrayPassing},
		{"Full Wasm Module (return 42)", testFullWasmModule},
		{"Add/Return Expression", testAddReturn},
		{"Memory Operations Performance", testMemoryPerformance},
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

// createBridgeContext creates a QuickJS runtime and context with all the
// Emscripten-style memory bridge functions and Binaryen wrappers registered.
func createBridgeContext() (*qjs.Runtime, *qjs.Context) {
	// Reset linear memory for each test
	lm = NewLinearMemory()

	rt, err := qjs.New(qjs.Option{
		MemoryLimit:  256 * 1024 * 1024,
		MaxStackSize: 4 * 1024 * 1024,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create QuickJS runtime: %v", err))
	}

	ctx := rt.Context()

	// ---------------------------------------------------------------
	// Register Emscripten-style memory functions
	// ---------------------------------------------------------------

	ctx.SetFunc("_malloc", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		size := int(args[0].Int32())
		ptr := lm.Malloc(size)
		return this.Context().NewInt32(int32(ptr)), nil
	})

	ctx.SetFunc("_free", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		ptr := int(args[0].Int32())
		lm.Free(ptr)
		return this.Context().NewInt32(0), nil
	})

	// --- i32 store/load ---
	ctx.SetFunc("__i32_store", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := int(args[1].Int32())
		lm.I32Store(addr, value)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__i32_store8", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := byte(args[1].Int32())
		lm.I32Store8(addr, value)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__i32_store16", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := uint16(args[1].Int32())
		lm.I32Store16(addr, value)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__i32_load", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := lm.I32Load(addr)
		return this.Context().NewInt32(int32(value)), nil
	})

	ctx.SetFunc("__i32_load8_u", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := lm.I32Load8U(addr)
		return this.Context().NewInt32(int32(value)), nil
	})

	ctx.SetFunc("__i32_load8_s", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := lm.I32Load8S(addr)
		return this.Context().NewInt32(int32(value)), nil
	})

	ctx.SetFunc("__i32_load16_u", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := lm.I32Load16U(addr)
		return this.Context().NewInt32(int32(value)), nil
	})

	ctx.SetFunc("__i32_load16_s", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := lm.I32Load16S(addr)
		return this.Context().NewInt32(int32(value)), nil
	})

	// --- float store/load ---
	ctx.SetFunc("__f32_store", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := float32(args[1].Float64())
		lm.F32Store(addr, value)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__f64_store", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := args[1].Float64()
		lm.F64Store(addr, value)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__f32_load", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := lm.F32Load(addr)
		return this.Context().NewFloat64(float64(value)), nil
	})

	ctx.SetFunc("__f64_load", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		value := lm.F64Load(addr)
		return this.Context().NewFloat64(value), nil
	})

	// ---------------------------------------------------------------
	// Register Binaryen functions with memory bridge logic
	// ---------------------------------------------------------------

	// --- Module lifecycle ---
	ctx.SetFunc("_BinaryenModuleCreate", func(this *qjs.This) (*qjs.Value, error) {
		module := binaryenModuleCreate()
		// Module handles are pointers — can exceed int32 range.
		// We store as float64 (which has 53 bits of mantissa, enough for pointers).
		return this.Context().NewFloat64(float64(module)), nil
	})

	ctx.SetFunc("_BinaryenModuleDispose", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		binaryenModuleDispose(module)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("_BinaryenModuleValidate", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		valid := binaryenModuleValidate(module)
		if valid {
			return this.Context().NewInt32(1), nil
		}
		return this.Context().NewInt32(0), nil
	})

	// --- Type operations ---
	ctx.SetFunc("_BinaryenTypeNone", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewFloat64(float64(binaryenTypeNone())), nil
	})

	ctx.SetFunc("_BinaryenTypeInt32", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewFloat64(float64(binaryenTypeInt32())), nil
	})

	ctx.SetFunc("_BinaryenTypeCreate", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		typesPtr := int(args[0].Int32())
		numTypes := int(args[1].Int32())
		types := make([]uintptr, numTypes)
		for i := 0; i < numTypes; i++ {
			// Each type is stored as a 32-bit value in linear memory.
			// BinaryenType is uintptr_t, but on the JS side we store the
			// lower 32 bits since type values are small.
			types[i] = uintptr(uint32(lm.I32Load(typesPtr + i*4)))
		}
		result := binaryenTypeCreate(types)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// --- Literal operations ---

	// _BinaryenSizeofLiteral returns the size of the C BinaryenLiteral struct.
	ctx.SetFunc("_BinaryenSizeofLiteral", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewInt32(int32(binaryenSizeofLiteral())), nil
	})

	// _BinaryenLiteralInt32 fills a BinaryenLiteral in linear memory with an i32 value.
	// JS calls: _BinaryenLiteralInt32(litPtr, value)
	// Go: creates a real C literal, copies its bytes into linear memory at litPtr.
	ctx.SetFunc("_BinaryenLiteralInt32", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		litPtr := int(args[0].Int32())
		value := int32(args[1].Int32())

		// Create the real C literal
		lit := binaryenLiteralInt32(value)

		// Copy the literal struct bytes into linear memory
		litSize := binaryenSizeofLiteral()
		litBytes := C.GoBytes(unsafe.Pointer(&lit), C.int(litSize))
		lm.WriteBytes(litPtr, litBytes)

		return this.Context().NewInt32(0), nil
	})

	// --- Expression operations ---

	// _BinaryenConst reads a literal from linear memory, creates a const expression.
	// JS calls: _BinaryenConst(module, litPtr)
	ctx.SetFunc("_BinaryenConst", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		litPtr := int(args[1].Int32())

		// Read the literal bytes from linear memory
		litSize := binaryenSizeofLiteral()
		litBytes := lm.ReadBytes(litPtr, litSize)

		// Create a C literal struct from those bytes
		var lit C.struct_BinaryenLiteral
		C.memcpy(unsafe.Pointer(&lit), unsafe.Pointer(&litBytes[0]), C.size_t(litSize))

		// Call BinaryenConst with the struct by value
		result := binaryenConst(module, lit)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// _BinaryenBlock creates a block expression.
	// JS calls: _BinaryenBlock(module, namePtr, childrenPtr, numChildren, type)
	ctx.SetFunc("_BinaryenBlock", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		namePtr := int(args[1].Int32())
		childrenPtr := int(args[2].Int32())
		numChildren := int(args[3].Int32())
		typ := uintptr(args[4].Float64())

		// Read name from linear memory
		var name string
		if namePtr != 0 {
			name = lm.ReadString(namePtr)
		}

		// Read children array from linear memory
		// Each child is an ExpressionRef (pointer) stored as i32 in linear memory.
		// We load as uint32 to avoid sign extension issues with pointers.
		children := make([]uintptr, numChildren)
		for i := 0; i < numChildren; i++ {
			// Read raw 32-bit value from linear memory without sign extension
			raw := uint32(lm.I32Load(childrenPtr + i*4))
			children[i] = uintptr(raw)
		}

		result := binaryenBlock(module, name, children, typ)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// _BinaryenLocalGet creates a local.get expression.
	ctx.SetFunc("_BinaryenLocalGet", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		index := int(args[1].Int32())
		typ := uintptr(args[2].Float64())
		result := binaryenLocalGet(module, index, typ)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// _BinaryenLocalSet creates a local.set expression.
	ctx.SetFunc("_BinaryenLocalSet", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		index := int(args[1].Int32())
		value := uintptr(args[2].Float64())
		result := binaryenLocalSet(module, index, value)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// _BinaryenBinary creates a binary operation expression.
	ctx.SetFunc("_BinaryenBinary", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		op := int32(args[1].Int32())
		left := uintptr(args[2].Float64())
		right := uintptr(args[3].Float64())
		result := binaryenBinaryOp(module, op, left, right)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// _BinaryenAddInt32 returns the add-i32 opcode.
	ctx.SetFunc("_BinaryenAddInt32", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewInt32(binaryenAddInt32()), nil
	})

	// _BinaryenReturn creates a return expression.
	ctx.SetFunc("_BinaryenReturn", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		value := uintptr(args[1].Float64())
		result := binaryenReturn(module, value)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// --- Function operations ---

	// _BinaryenAddFunction adds a function to a module.
	// JS calls: _BinaryenAddFunction(module, namePtr, params, results, varTypesPtr, numVarTypes, body)
	ctx.SetFunc("_BinaryenAddFunction", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		namePtr := int(args[1].Int32())
		params := uintptr(args[2].Float64())
		results := uintptr(args[3].Float64())
		varTypesPtr := int(args[4].Int32())
		numVarTypes := int(args[5].Int32())
		body := uintptr(args[6].Float64())

		// Read function name from linear memory
		name := lm.ReadString(namePtr)

		// Read var types from linear memory
		var varTypes []uintptr
		if numVarTypes > 0 && varTypesPtr != 0 {
			varTypes = make([]uintptr, numVarTypes)
			for i := 0; i < numVarTypes; i++ {
				varTypes[i] = uintptr(uint32(lm.I32Load(varTypesPtr + i*4)))
			}
		}

		result := binaryenAddFunction(module, name, params, results, varTypes, body)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// --- Export operations ---

	// _BinaryenAddFunctionExport adds a function export.
	// JS calls: _BinaryenAddFunctionExport(module, internalNamePtr, externalNamePtr)
	ctx.SetFunc("_BinaryenAddFunctionExport", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		internalNamePtr := int(args[1].Int32())
		externalNamePtr := int(args[2].Int32())

		internalName := lm.ReadString(internalNamePtr)
		externalName := lm.ReadString(externalNamePtr)

		result := binaryenAddFunctionExport(module, internalName, externalName)
		return this.Context().NewFloat64(float64(result)), nil
	})

	// _BinaryenModuleAllocateAndWrite serializes the module and writes
	// the result struct into linear memory.
	// JS calls: _BinaryenModuleAllocateAndWrite(module, outPtr, sourceMapUrlPtr)
	// outPtr points to a 12-byte struct in linear memory:
	//   [0:4]  = pointer to binary data (in linear memory)
	//   [4:8]  = binary length
	//   [8:12] = pointer to source map (in linear memory, or 0)
	ctx.SetFunc("_BinaryenModuleAllocateAndWrite", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		module := uintptr(args[0].Float64())
		outPtr := int(args[1].Int32())
		sourceMapURLPtr := int(args[2].Int32())

		var sourceMapURL string
		if sourceMapURLPtr != 0 {
			sourceMapURL = lm.ReadString(sourceMapURLPtr)
		}

		// Call the real C function
		result := binaryenModuleAllocateAndWrite(module, sourceMapURL)

		// Copy binary data INTO linear memory
		binaryPtr := lm.Malloc(len(result.Binary))
		lm.WriteBytes(binaryPtr, result.Binary)

		// Write the result struct into linear memory at outPtr
		lm.I32Store(outPtr, binaryPtr)
		lm.I32Store(outPtr+4, len(result.Binary))

		// Handle source map
		if result.SourceMap != "" {
			smPtr := lm.Malloc(len(result.SourceMap) + 1)
			lm.WriteString(smPtr, result.SourceMap)
			lm.I32Store(outPtr+8, smPtr)
		} else {
			lm.I32Store(outPtr+8, 0)
		}

		return this.Context().NewInt32(0), nil
	})

	return rt, ctx
}

// ---------------------------------------------------------------------------
// Test 1: Module Create/Dispose
// ---------------------------------------------------------------------------

func testModuleCreateDispose() error {
	rt, ctx := createBridgeContext()
	defer rt.Close()

	result, err := ctx.Eval("test1.js", qjs.Code(`
		var module = _BinaryenModuleCreate();
		var ok = (module !== 0 && module !== undefined);
		_BinaryenModuleDispose(module);
		ok ? "created_and_disposed" : "failed";
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	val := result.String()
	if val != "created_and_disposed" {
		return fmt.Errorf("expected 'created_and_disposed', got '%s'", val)
	}
	fmt.Println("  Module created and disposed without crash")
	return nil
}

// ---------------------------------------------------------------------------
// Test 2: Simple Const Expression
// ---------------------------------------------------------------------------

func testSimpleConst() error {
	rt, ctx := createBridgeContext()
	defer rt.Close()

	result, err := ctx.Eval("test2.js", qjs.Code(`
		var module = _BinaryenModuleCreate();

		// Allocate literal struct in linear memory
		var litSize = _BinaryenSizeofLiteral();
		var litPtr = _malloc(litSize);

		// Fill the literal: i32.const 42
		_BinaryenLiteralInt32(litPtr, 42);

		// Create const expression
		var expr = _BinaryenConst(module, litPtr);

		// expr should be a non-zero ExpressionRef
		var ok = (expr !== 0 && expr !== undefined);

		_BinaryenModuleDispose(module);
		JSON.stringify({ ok: ok, litSize: litSize, expr: expr });
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	var res struct {
		OK      bool    `json:"ok"`
		LitSize int     `json:"litSize"`
		Expr    float64 `json:"expr"`
	}
	if err := json.Unmarshal([]byte(result.String()), &res); err != nil {
		return fmt.Errorf("JSON parse failed: %w", err)
	}
	if !res.OK {
		return fmt.Errorf("expression ref was zero or undefined")
	}
	fmt.Printf("  Literal size: %d bytes, ExpressionRef: %v\n", res.LitSize, res.Expr)
	return nil
}

// ---------------------------------------------------------------------------
// Test 3: String Passing
// ---------------------------------------------------------------------------

func testStringPassing() error {
	rt, ctx := createBridgeContext()
	defer rt.Close()

	result, err := ctx.Eval("test3.js", qjs.Code(`
		var module = _BinaryenModuleCreate();

		// Helper: allocate string in linear memory
		function allocateString(str) {
			var len = str.length;
			var ptr = _malloc(len + 1);
			for (var i = 0; i < len; i++) {
				__i32_store8(ptr + i, str.charCodeAt(i));
			}
			__i32_store8(ptr + len, 0);
			return ptr;
		}

		// Create a named block with no children
		var namePtr = allocateString("test_block");
		var noneType = _BinaryenTypeNone();
		var block = _BinaryenBlock(module, namePtr, 0, 0, noneType);
		var ok = (block !== 0 && block !== undefined);

		_BinaryenModuleDispose(module);
		JSON.stringify({ ok: ok, block: block });
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	var res struct {
		OK    bool    `json:"ok"`
		Block float64 `json:"block"`
	}
	if err := json.Unmarshal([]byte(result.String()), &res); err != nil {
		return fmt.Errorf("JSON parse failed: %w", err)
	}
	if !res.OK {
		return fmt.Errorf("block was zero or undefined")
	}
	fmt.Printf("  Named block created successfully, ref: %v\n", res.Block)
	return nil
}

// ---------------------------------------------------------------------------
// Test 4: Array Passing (Block with children)
// ---------------------------------------------------------------------------

func testArrayPassing() error {
	rt, ctx := createBridgeContext()
	defer rt.Close()

	result, err := ctx.Eval("test4.js", qjs.Code(`
		var module = _BinaryenModuleCreate();

		function allocateString(str) {
			var len = str.length;
			var ptr = _malloc(len + 1);
			for (var i = 0; i < len; i++) {
				__i32_store8(ptr + i, str.charCodeAt(i));
			}
			__i32_store8(ptr + len, 0);
			return ptr;
		}

		// Create two const expressions
		var litSize = _BinaryenSizeofLiteral();

		var lit1Ptr = _malloc(litSize);
		_BinaryenLiteralInt32(lit1Ptr, 10);
		var const1 = _BinaryenConst(module, lit1Ptr);

		var lit2Ptr = _malloc(litSize);
		_BinaryenLiteralInt32(lit2Ptr, 20);
		var const2 = _BinaryenConst(module, lit2Ptr);

		// Allocate array of ExpressionRefs in linear memory
		var childrenPtr = _malloc(2 * 4); // 2 pointers, 4 bytes each
		__i32_store(childrenPtr, const1);      // truncates to i32
		__i32_store(childrenPtr + 4, const2);  // truncates to i32

		// Create block with children
		var namePtr = allocateString("two_consts");
		var i32Type = _BinaryenTypeInt32();
		var block = _BinaryenBlock(module, namePtr, childrenPtr, 2, i32Type);

		var ok = (block !== 0 && block !== undefined);
		var c1ok = (const1 !== 0);
		var c2ok = (const2 !== 0);

		_BinaryenModuleDispose(module);
		JSON.stringify({
			ok: ok,
			const1: const1,
			const2: const2,
			block: block,
			c1ok: c1ok,
			c2ok: c2ok
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	var res struct {
		OK    bool    `json:"ok"`
		Block float64 `json:"block"`
		C1OK  bool    `json:"c1ok"`
		C2OK  bool    `json:"c2ok"`
	}
	if err := json.Unmarshal([]byte(result.String()), &res); err != nil {
		return fmt.Errorf("JSON parse failed: %w", err)
	}
	if !res.OK || !res.C1OK || !res.C2OK {
		return fmt.Errorf("block or children were zero (ok=%v, c1ok=%v, c2ok=%v)", res.OK, res.C1OK, res.C2OK)
	}
	fmt.Printf("  Block with 2 children created successfully, ref: %v\n", res.Block)
	return nil
}

// ---------------------------------------------------------------------------
// Test 5: Full Wasm Module (function returning 42)
// ---------------------------------------------------------------------------

func testFullWasmModule() error {
	rt, ctx := createBridgeContext()
	defer rt.Close()

	result, err := ctx.Eval("test5.js", qjs.Code(`
		// This follows the EXACT patterns from AssemblyScript's module.ts

		// Helper: allocate string in linear memory
		function allocateString(str) {
			var len = str.length;
			var ptr = _malloc(len + 1);
			for (var i = 0; i < len; i++) {
				__i32_store8(ptr + i, str.charCodeAt(i));
			}
			__i32_store8(ptr + len, 0);
			return ptr;
		}

		// Create module
		var module = _BinaryenModuleCreate();

		// Create literal: i32.const 42
		var litPtr = _malloc(_BinaryenSizeofLiteral());
		_BinaryenLiteralInt32(litPtr, 42);
		var body = _BinaryenConst(module, litPtr);

		// Add function "answer" with no params, returns i32
		var namePtr = allocateString("answer");
		var params = _BinaryenTypeNone();
		var results = _BinaryenTypeInt32();
		_BinaryenAddFunction(module, namePtr, params, results, 0, 0, body);

		// Export it
		var exportNamePtr = allocateString("answer");
		_BinaryenAddFunctionExport(module, namePtr, exportNamePtr);

		// Validate
		var valid = _BinaryenModuleValidate(module);

		// Emit binary
		var resPtr = _malloc(12);
		_BinaryenModuleAllocateAndWrite(module, resPtr, 0);
		var binaryPtr = __i32_load(resPtr);
		var binaryLen = __i32_load(resPtr + 4);

		// Read first 4 bytes (Wasm magic number) from linear memory
		var magic0 = __i32_load8_u(binaryPtr);
		var magic1 = __i32_load8_u(binaryPtr + 1);
		var magic2 = __i32_load8_u(binaryPtr + 2);
		var magic3 = __i32_load8_u(binaryPtr + 3);

		_BinaryenModuleDispose(module);

		JSON.stringify({
			valid: valid,
			binaryLen: binaryLen,
			binaryPtr: binaryPtr,
			magic: [magic0, magic1, magic2, magic3]
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	var res struct {
		Valid     int     `json:"valid"`
		BinaryLen int     `json:"binaryLen"`
		BinaryPtr int     `json:"binaryPtr"`
		Magic     [4]int  `json:"magic"`
	}
	if err := json.Unmarshal([]byte(result.String()), &res); err != nil {
		return fmt.Errorf("JSON parse failed: %w", err)
	}

	if res.Valid != 1 {
		return fmt.Errorf("module validation failed (valid=%d)", res.Valid)
	}

	// Check Wasm magic number: \0asm = [0x00, 0x61, 0x73, 0x6d]
	expectedMagic := [4]int{0x00, 0x61, 0x73, 0x6d}
	if res.Magic != expectedMagic {
		return fmt.Errorf("invalid Wasm magic: got %v, expected %v", res.Magic, expectedMagic)
	}

	// Also read the binary from linear memory in Go to verify
	wasmBytes := lm.ReadBytes(res.BinaryPtr, res.BinaryLen)
	if len(wasmBytes) != res.BinaryLen {
		return fmt.Errorf("binary length mismatch: got %d, expected %d", len(wasmBytes), res.BinaryLen)
	}
	if wasmBytes[0] != 0x00 || wasmBytes[1] != 0x61 || wasmBytes[2] != 0x73 || wasmBytes[3] != 0x6d {
		return fmt.Errorf("Wasm magic bytes mismatch in Go-side read")
	}

	fmt.Printf("  Module valid: true\n")
	fmt.Printf("  Binary size: %d bytes\n", res.BinaryLen)
	fmt.Printf("  Wasm magic: %v (\\0asm)\n", res.Magic)
	fmt.Printf("  Go-side binary verification: passed\n")

	return nil
}

// ---------------------------------------------------------------------------
// Test 6: Add/Return Expression
// ---------------------------------------------------------------------------

func testAddReturn() error {
	rt, ctx := createBridgeContext()
	defer rt.Close()

	result, err := ctx.Eval("test6.js", qjs.Code(`
		function allocateString(str) {
			var len = str.length;
			var ptr = _malloc(len + 1);
			for (var i = 0; i < len; i++) {
				__i32_store8(ptr + i, str.charCodeAt(i));
			}
			__i32_store8(ptr + len, 0);
			return ptr;
		}

		var module = _BinaryenModuleCreate();
		var i32Type = _BinaryenTypeInt32();
		var noneType = _BinaryenTypeNone();

		// Create params type: (i32, i32) — use BinaryenTypeCreate
		var typesPtr = _malloc(8); // 2 types * 4 bytes
		__i32_store(typesPtr, i32Type);      // truncates to i32
		__i32_store(typesPtr + 4, i32Type);  // truncates to i32
		var paramsType = _BinaryenTypeCreate(typesPtr, 2);

		// local.get 0 (first param)
		var a = _BinaryenLocalGet(module, 0, i32Type);
		// local.get 1 (second param)
		var b = _BinaryenLocalGet(module, 1, i32Type);

		// a + b
		var addOp = _BinaryenAddInt32();
		var add = _BinaryenBinary(module, addOp, a, b);

		// return (a + b)
		var ret = _BinaryenReturn(module, add);

		// Add function "add"
		var namePtr = allocateString("add");
		_BinaryenAddFunction(module, namePtr, paramsType, i32Type, 0, 0, ret);

		// Export it
		var exportNamePtr = allocateString("add");
		_BinaryenAddFunctionExport(module, namePtr, exportNamePtr);

		// Validate
		var valid = _BinaryenModuleValidate(module);

		// Emit binary
		var resPtr = _malloc(12);
		_BinaryenModuleAllocateAndWrite(module, resPtr, 0);
		var binaryPtr = __i32_load(resPtr);
		var binaryLen = __i32_load(resPtr + 4);

		// Read magic
		var magic0 = __i32_load8_u(binaryPtr);
		var magic1 = __i32_load8_u(binaryPtr + 1);
		var magic2 = __i32_load8_u(binaryPtr + 2);
		var magic3 = __i32_load8_u(binaryPtr + 3);

		_BinaryenModuleDispose(module);

		JSON.stringify({
			valid: valid,
			binaryLen: binaryLen,
			magic: [magic0, magic1, magic2, magic3]
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	var res struct {
		Valid     int    `json:"valid"`
		BinaryLen int    `json:"binaryLen"`
		Magic     [4]int `json:"magic"`
	}
	if err := json.Unmarshal([]byte(result.String()), &res); err != nil {
		return fmt.Errorf("JSON parse failed: %w", err)
	}

	if res.Valid != 1 {
		return fmt.Errorf("module validation failed (valid=%d)", res.Valid)
	}

	expectedMagic := [4]int{0x00, 0x61, 0x73, 0x6d}
	if res.Magic != expectedMagic {
		return fmt.Errorf("invalid Wasm magic: got %v, expected %v", res.Magic, expectedMagic)
	}

	fmt.Printf("  Add function: (i32, i32) -> i32\n")
	fmt.Printf("  Module valid: true\n")
	fmt.Printf("  Binary size: %d bytes\n", res.BinaryLen)
	fmt.Printf("  Wasm magic: %v (\\0asm)\n", res.Magic)

	return nil
}

// ---------------------------------------------------------------------------
// Test 7: Memory Operations Performance
// ---------------------------------------------------------------------------

func testMemoryPerformance() error {
	const iterations = 10000

	lm = NewLinearMemory()

	// Time _malloc
	start := time.Now()
	for i := 0; i < iterations; i++ {
		lm.Malloc(64) // allocate 64 bytes each time
	}
	mallocTime := time.Since(start)
	mallocPerOp := mallocTime / time.Duration(iterations)

	// Reset for store/load tests
	lm.Reset()

	// Pre-allocate a region for store/load
	base := lm.Malloc(iterations * 4)

	// Time __i32_store
	start = time.Now()
	for i := 0; i < iterations; i++ {
		lm.I32Store(base+i*4, i)
	}
	storeTime := time.Since(start)
	storePerOp := storeTime / time.Duration(iterations)

	// Time __i32_load
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = lm.I32Load(base + i*4)
	}
	loadTime := time.Since(start)
	loadPerOp := loadTime / time.Duration(iterations)

	// Now test through the full JS bridge
	rt, ctx := createBridgeContext()
	defer rt.Close()

	jsCode := fmt.Sprintf(`
		var count = %d;

		// Measure malloc
		var mallocStart = Date.now();
		for (var i = 0; i < count; i++) {
			_malloc(64);
		}
		var mallocMs = Date.now() - mallocStart;

		// Measure i32_store
		var base = _malloc(count * 4);
		var storeStart = Date.now();
		for (var i = 0; i < count; i++) {
			__i32_store(base + i * 4, i);
		}
		var storeMs = Date.now() - storeStart;

		// Measure i32_load
		var loadStart = Date.now();
		var sum = 0;
		for (var i = 0; i < count; i++) {
			sum += __i32_load(base + i * 4);
		}
		var loadMs = Date.now() - loadStart;

		JSON.stringify({
			count: count,
			mallocMs: mallocMs,
			storeMs: storeMs,
			loadMs: loadMs,
			sum: sum
		});
	`, iterations)

	result, err := ctx.Eval("perf.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	var res struct {
		Count    int `json:"count"`
		MallocMs int `json:"mallocMs"`
		StoreMs  int `json:"storeMs"`
		LoadMs   int `json:"loadMs"`
		Sum      int `json:"sum"`
	}
	if err := json.Unmarshal([]byte(result.String()), &res); err != nil {
		return fmt.Errorf("JSON parse failed: %w", err)
	}

	// Verify sum
	expectedSum := (iterations - 1) * iterations / 2
	if res.Sum != expectedSum {
		return fmt.Errorf("sum mismatch: got %d, expected %d", res.Sum, expectedSum)
	}

	fmt.Printf("  Go-only (no JS bridge):\n")
	fmt.Printf("    malloc:     %d ops in %s (%s/op)\n", iterations, mallocTime.Round(time.Microsecond), mallocPerOp)
	fmt.Printf("    i32_store:  %d ops in %s (%s/op)\n", iterations, storeTime.Round(time.Microsecond), storePerOp)
	fmt.Printf("    i32_load:   %d ops in %s (%s/op)\n", iterations, loadTime.Round(time.Microsecond), loadPerOp)
	fmt.Printf("  Through JS bridge (QuickJS -> Go -> memory):\n")
	fmt.Printf("    malloc:     %d ops in %dms\n", res.Count, res.MallocMs)
	fmt.Printf("    i32_store:  %d ops in %dms\n", res.Count, res.StoreMs)
	fmt.Printf("    i32_load:   %d ops in %dms\n", res.Count, res.LoadMs)
	fmt.Printf("  Data integrity: sum=%d (verified)\n", res.Sum)

	return nil
}
