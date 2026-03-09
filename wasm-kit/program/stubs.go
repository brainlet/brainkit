package program

// This file contains interface stubs and function variables for packages
// that are not yet ported (module, compiler, builtins).
// These will be replaced with real imports when those packages are ported.

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// --- compiler package stubs ---

// Options represents compiler options. Stub until compiler/ is ported.
type Options struct {
	// Size type for the target (e.g. usize).
	UsizeType *types.Type
	// Whether target is wasm64.
	IsWasm64 bool
	// Stack size, >0 enables stack pointer.
	StackSize int32
	// SizeTypeRef is the Binaryen type ref for usize.
	SizeTypeRef uintptr
	// Whether to generate source maps.
	SourceMap bool
	// Whether to generate debug info.
	DebugInfo bool
}

// --- module package stubs ---

// ExpressionRef is a Binaryen expression reference.
type ExpressionRef = uintptr

// FunctionRef is a Binaryen function reference.
type FunctionRef = uintptr

// MemorySegment represents a memory segment in the module.
type MemorySegment struct {
	Buffer []byte
	Offset ExpressionRef
}

// Module represents a Binaryen module. Stub until module/ is ported.
type Module struct{}

// ModuleCreate creates a new module. Stub.
var ModuleCreate func(hasStack bool, sizeTypeRef uintptr) *Module

// GetFunctionName gets the name of a function by its ref.
var GetFunctionName func(ref FunctionRef) string

// ModuleSetDebugLocation sets a debug location in the module.
var ModuleSetDebugLocation func(m *Module, funcRef FunctionRef, exprRef ExpressionRef, fileIndex int32, line int32, col int32)

// ModuleSetLocalName sets a local variable name for debug info.
var ModuleSetLocalName func(m *Module, funcRef FunctionRef, index int32, name string)

// --- resolver helpers ---

// ResolveFunction is a function variable that resolves a function prototype.
// It delegates to Resolver.ResolveFunction with default contextual types and ReportModeReport.
// This exists so that program.go RequireFunction can call it without knowing Resolver internals.
var ResolveFunction func(r *Resolver, prototype *FunctionPrototype, typeArguments []*types.Type) *Function

func init() {
	// Wire the real resolver implementation.
	ResolveFunction = func(r *Resolver, prototype *FunctionPrototype, typeArguments []*types.Type) *Function {
		return r.ResolveFunction(prototype, typeArguments, nil, ReportModeReport)
	}
}

// --- parser package stubs ---
// Parser is already ported, but to avoid circular imports we define
// the narrow interface needed here.

// ParserRef represents a reference to the parser. Stub to break circular dependency.
type ParserRef struct{}

// NewParserRef creates a new parser reference.
var NewParserRef func(diagnostics []*diagnostics.DiagnosticMessage, sources []*ast.Source) *ParserRef

// --- builtins package stubs ---

// BuiltinFunctions is the set of builtin function handlers. Stub.
var BuiltinFunctions map[string]interface{}

// BuiltinVariablesOnAccess is the set of builtin variables with on-access callbacks. Stub.
var BuiltinVariablesOnAccess map[string]interface{}
