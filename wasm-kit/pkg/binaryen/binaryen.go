// Package binaryen provides Go bindings to Binaryen's C API via CGo.
//
// This is the foundation package for wasm-kit. It wraps libbinaryen.a
// (version 123) which must be built from source before this package compiles.
//
// Build libbinaryen.a:
//
//	cd binaryen && git checkout version_123
//	mkdir -p build && cd build
//	cmake .. -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTS=OFF -DBUILD_TOOLS=OFF -DBUILD_STATIC_LIB=ON
//	make -j$(nproc)
//
// All Binaryen types (ExpressionRef, FunctionRef, etc.) are opaque handles.
// Memory is owned by the Module — disposing the module frees everything inside.
package binaryen

/*
#cgo CFLAGS: -I${SRCDIR}/../../../../../clones/binaryen/src
#cgo LDFLAGS: -L${SRCDIR}/../../../../../clones/binaryen/build/lib -lbinaryen -lstdc++ -lm
#include "binaryen-c.h"
#include <stdlib.h>
*/
import "C"
import (
	"sync"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Fundamental types — these map 1:1 to the C typedefs in binaryen-c.h
// ---------------------------------------------------------------------------

// Index is used for internal indexes and list sizes.
type Index = uint32

// Type represents a Binaryen value type (i32, i64, f32, f64, v128, ref types, tuples).
type Type = uintptr

// HeapType represents a Binaryen heap type (func, extern, any, eq, struct, array, etc.).
type HeapType = uintptr

// PackedType represents a Binaryen packed type (i8, i16 for struct fields).
type PackedType = uint32

// ExpressionID identifies the kind of an expression (Block, If, Loop, etc.).
type ExpressionID = uint32

// ExternalKind identifies the kind of an import/export (function, table, memory, global, tag).
type ExternalKind = uint32

// Features represents a set of Wasm feature flags (atomics, SIMD, etc.).
type Features = uint32

// Op represents a unary, binary, SIMD, or atomic operation.
type Op = int32

// ExpressionRef is an opaque handle to a Binaryen expression node.
// Owned by the parent Module — never free these manually.
type ExpressionRef = uintptr

// FunctionRef is an opaque handle to a Binaryen function.
type FunctionRef = uintptr

// GlobalRef is an opaque handle to a Binaryen global.
type GlobalRef = uintptr

// TagRef is an opaque handle to a Binaryen tag.
type TagRef = uintptr

// ImportRef is an opaque handle to a Binaryen import.
type ImportRef = uintptr

// ExportRef is an opaque handle to a Binaryen export.
type ExportRef = uintptr

// TableRef is an opaque handle to a Binaryen table.
type TableRef = uintptr

// ElementSegmentRef is an opaque handle to a Binaryen element segment.
type ElementSegmentRef = uintptr

// RelooperRef is an opaque handle to a Binaryen Relooper.
type RelooperRef = uintptr

// RelooperBlockRef is an opaque handle to a Binaryen Relooper block.
type RelooperBlockRef = uintptr

// ExpressionRunnerRef is an opaque handle to a Binaryen expression runner.
type ExpressionRunnerRef = uintptr

// TypeBuilderRef is an opaque handle to a Binaryen type builder.
type TypeBuilderRef = uintptr

// ExpressionRunnerFlags controls expression runner behavior.
type ExpressionRunnerFlags = uint32

// ExpressionRunnerFlagsDefault returns the default expression runner flags (0).
func ExpressionRunnerFlagsDefault() ExpressionRunnerFlags { return 0 }

// ExpressionRunnerFlagsPreserveSideeffects returns flags that preserve side effects (1).
func ExpressionRunnerFlagsPreserveSideeffects() ExpressionRunnerFlags { return 1 }

// SideEffects represents the side effects of an expression.
type SideEffects = uint32

// ---------------------------------------------------------------------------
// StringPool — caches C strings to avoid repeated CString/free cycles
// ---------------------------------------------------------------------------

// StringPool caches C strings for reuse within a Module's lifetime.
// All strings are freed when the pool is freed.
type StringPool struct {
	mu    sync.Mutex
	cache map[string]*C.char
}

func newStringPool() *StringPool {
	return &StringPool{cache: make(map[string]*C.char)}
}

// CStr returns a cached C string. The returned pointer is valid until the pool is freed.
// Passing an empty string returns nil (which Binaryen interprets as "no name").
func (p *StringPool) CStr(s string) *C.char {
	if s == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if cs, ok := p.cache[s]; ok {
		return cs
	}
	cs := C.CString(s)
	p.cache[s] = cs
	return cs
}

func (p *StringPool) free() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, cs := range p.cache {
		C.free(unsafe.Pointer(cs))
	}
	p.cache = nil
}

// ---------------------------------------------------------------------------
// Module — the central object that owns all expressions, functions, etc.
// ---------------------------------------------------------------------------

// Module wraps a BinaryenModuleRef with Go lifecycle management.
// All expressions, functions, globals, etc. created on a Module are owned by it.
// Call Dispose() when done to free all resources.
type Module struct {
	ref     C.BinaryenModuleRef
	strings *StringPool
}

// NewModule creates a new empty Binaryen module.
func NewModule() *Module {
	return &Module{
		ref:     C.BinaryenModuleCreate(),
		strings: newStringPool(),
	}
}

// Dispose frees the module and all its contents. Must be called when done.
// After Dispose, the Module must not be used.
func (m *Module) Dispose() {
	if m.ref != nil {
		C.BinaryenModuleDispose(m.ref)
		m.ref = nil
	}
	if m.strings != nil {
		m.strings.free()
		m.strings = nil
	}
}

// Ref returns the underlying C module reference.
// Used by code that needs to pass the module to other C functions.
func (m *Module) Ref() C.BinaryenModuleRef {
	return m.ref
}

// str is a shorthand for the module's string pool.
func (m *Module) str(s string) *C.char {
	return m.strings.CStr(s)
}

// ---------------------------------------------------------------------------
// Helper functions for CGo boundary crossing
// ---------------------------------------------------------------------------

// cBool converts a Go bool to a C.bool.
func cBool(b bool) C.bool {
	if b {
		return C.bool(true)
	}
	return C.bool(false)
}

// goBool converts a C.bool to a Go bool.
func goBool(b C.bool) bool {
	return bool(b)
}

// goString converts a C string to a Go string without freeing the C string.
// Returns "" if the pointer is nil.
func goString(cs *C.char) string {
	if cs == nil {
		return ""
	}
	return C.GoString(cs)
}
