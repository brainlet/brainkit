// Standalone CGo wrapper for Binaryen C functions needed by the bridge experiment.
// This file provides thin Go wrappers around the C API, converting between
// Go/uintptr types and C types.

package main

/*
#cgo CFLAGS: -I/Users/davidroman/Documents/code/clones/binaryen/src
#cgo LDFLAGS: -L/Users/davidroman/Documents/code/clones/binaryen/build/lib -lbinaryen -lstdc++ -lm
#include "binaryen-c.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"
import "unsafe"

// --- Module lifecycle ---

// binaryenModuleCreate creates a new Binaryen module.
func binaryenModuleCreate() uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenModuleCreate()))
}

// binaryenModuleDispose disposes a Binaryen module.
func binaryenModuleDispose(module uintptr) {
	C.BinaryenModuleDispose(C.BinaryenModuleRef(unsafe.Pointer(module)))
}

// binaryenModuleValidate validates a Binaryen module. Returns true if valid.
func binaryenModuleValidate(module uintptr) bool {
	return bool(C.BinaryenModuleValidate(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

// BinaryResult holds the result of BinaryenModuleAllocateAndWrite.
type BinaryResult struct {
	Binary    []byte
	SourceMap string
}

// binaryenModuleAllocateAndWrite serializes a module to binary Wasm format.
// Returns the binary bytes and source map string. The caller does not need
// to free anything — the C memory is copied and freed internally.
func binaryenModuleAllocateAndWrite(module uintptr, sourceMapURL string) BinaryResult {
	var cURL *C.char
	if sourceMapURL != "" {
		cURL = C.CString(sourceMapURL)
		defer C.free(unsafe.Pointer(cURL))
	}

	result := C.BinaryenModuleAllocateAndWrite(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cURL,
	)

	// Copy the binary data to Go
	binaryLen := int(result.binaryBytes)
	binary := C.GoBytes(result.binary, C.int(binaryLen))

	// Copy source map if present
	var sourceMap string
	if result.sourceMap != nil {
		sourceMap = C.GoString(result.sourceMap)
		C.free(unsafe.Pointer(result.sourceMap))
	}

	// Free the C-allocated binary
	C.free(result.binary)

	return BinaryResult{
		Binary:    binary,
		SourceMap: sourceMap,
	}
}

// --- Type operations ---

// binaryenTypeNone returns the "none" type (void).
func binaryenTypeNone() uintptr {
	return uintptr(C.BinaryenTypeNone())
}

// binaryenTypeInt32 returns the i32 type.
func binaryenTypeInt32() uintptr {
	return uintptr(C.BinaryenTypeInt32())
}

// binaryenTypeCreate creates a tuple type from multiple types.
func binaryenTypeCreate(types []uintptr) uintptr {
	if len(types) == 0 {
		return binaryenTypeNone()
	}
	cTypes := make([]C.BinaryenType, len(types))
	for i, t := range types {
		cTypes[i] = C.BinaryenType(t)
	}
	return uintptr(C.BinaryenTypeCreate(&cTypes[0], C.BinaryenIndex(len(types))))
}

// --- Literal operations ---

// binaryenLiteralInt32 creates a BinaryenLiteral for an i32 value.
// Returns the literal as raw bytes that can be stored in linear memory.
func binaryenLiteralInt32(value int32) C.struct_BinaryenLiteral {
	return C.BinaryenLiteralInt32(C.int32_t(value))
}

// binaryenSizeofLiteral returns the size of a BinaryenLiteral struct in bytes.
func binaryenSizeofLiteral() int {
	return int(C.sizeof_struct_BinaryenLiteral)
}

// --- Expression operations ---

// binaryenConst creates a constant expression from a literal.
func binaryenConst(module uintptr, literal C.struct_BinaryenLiteral) uintptr {
	ref := C.BinaryenConst(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		literal,
	)
	return uintptr(unsafe.Pointer(ref))
}

// binaryenBlock creates a block expression.
func binaryenBlock(module uintptr, name string, children []uintptr, typ uintptr) uintptr {
	var cName *C.char
	if name != "" {
		cName = C.CString(name)
		defer C.free(unsafe.Pointer(cName))
	}

	var cChildren *C.BinaryenExpressionRef
	if len(children) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(children))
		for i, c := range children {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(c))
		}
		cChildren = &cArr[0]
	}

	ref := C.BinaryenBlock(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cName,
		cChildren,
		C.BinaryenIndex(len(children)),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

// binaryenLocalGet creates a local.get expression.
func binaryenLocalGet(module uintptr, index int, typ uintptr) uintptr {
	ref := C.BinaryenLocalGet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

// binaryenLocalSet creates a local.set expression.
func binaryenLocalSet(module uintptr, index int, value uintptr) uintptr {
	ref := C.BinaryenLocalSet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

// binaryenBinary creates a binary (two-operand) expression.
func binaryenBinaryOp(module uintptr, op int32, left, right uintptr) uintptr {
	ref := C.BinaryenBinary(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(left)),
		C.BinaryenExpressionRef(unsafe.Pointer(right)),
	)
	return uintptr(unsafe.Pointer(ref))
}

// binaryenAddInt32 returns the AddInt32 opcode.
func binaryenAddInt32() int32 {
	return int32(C.BinaryenAddInt32())
}

// binaryenReturn creates a return expression.
func binaryenReturn(module uintptr, value uintptr) uintptr {
	var cValue C.BinaryenExpressionRef
	if value != 0 {
		cValue = C.BinaryenExpressionRef(unsafe.Pointer(value))
	}
	ref := C.BinaryenReturn(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cValue,
	)
	return uintptr(unsafe.Pointer(ref))
}

// --- Function operations ---

// binaryenAddFunction adds a function to a module.
func binaryenAddFunction(module uintptr, name string, params, results uintptr, varTypes []uintptr, body uintptr) uintptr {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var cVarTypes *C.BinaryenType
	if len(varTypes) > 0 {
		cArr := make([]C.BinaryenType, len(varTypes))
		for i, t := range varTypes {
			cArr[i] = C.BinaryenType(t)
		}
		cVarTypes = &cArr[0]
	}

	ref := C.BinaryenAddFunction(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cName,
		C.BinaryenType(params),
		C.BinaryenType(results),
		cVarTypes,
		C.BinaryenIndex(len(varTypes)),
		C.BinaryenExpressionRef(unsafe.Pointer(body)),
	)
	return uintptr(unsafe.Pointer(ref))
}

// --- Export operations ---

// binaryenAddFunctionExport adds a function export to a module.
func binaryenAddFunctionExport(module uintptr, internalName, externalName string) uintptr {
	cInternal := C.CString(internalName)
	defer C.free(unsafe.Pointer(cInternal))
	cExternal := C.CString(externalName)
	defer C.free(unsafe.Pointer(cExternal))

	ref := C.BinaryenAddFunctionExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cInternal,
		cExternal,
	)
	return uintptr(unsafe.Pointer(ref))
}
