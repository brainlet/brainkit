// Ported from: assemblyscript/src/builtins.ts (lines 776-831)
// Additional context types and registration maps for builtin type and variable handlers.
// Note: BuiltinFunctionContext, BuiltinFunction, and builtinFunctions are in builtins_types.go.
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// BuiltinTypesContext is the context for resolving builtin types.
// Ported from: BuiltinTypesContext class (line 777).
type BuiltinTypesContext struct {
	Resolver   *program.Resolver
	Node       *ast.NamedTypeNode
	CtxElement program.Element
	CtxTypes   map[string]*types.Type // nil if absent
	ReportMode program.ReportMode
}

// BuiltinVariableContext is the context for compiling builtin variables.
// Ported from: BuiltinVariableContext class (line 788).
type BuiltinVariableContext struct {
	Compiler       *Compiler
	Element        program.VariableLikeElement
	ContextualType *types.Type
	ReportNode     *ast.IdentifierExpression
}

// BuiltinTypeHandler is the function signature for builtin type resolution handlers.
type BuiltinTypeHandler func(ctx *BuiltinTypesContext) *types.Type

// BuiltinVariableCompileHandler is the function signature for builtin variable compile handlers.
type BuiltinVariableCompileHandler func(ctx *BuiltinVariableContext)

// BuiltinVariableAccessHandler is the function signature for builtin variable access handlers.
type BuiltinVariableAccessHandler func(ctx *BuiltinVariableContext) module.ExpressionRef

// Builtin registration maps for types and variables.
// Ported from: builtinTypes, builtinVariables_onCompile, builtinVariables_onAccess (lines 824, 830-831).
// Note: builtinFunctions is in builtins_types.go.
var (
	// BuiltinTypes maps builtin type names to their resolution handlers.
	BuiltinTypes = make(map[string]BuiltinTypeHandler)

	// BuiltinVariablesOnCompile maps builtin variable names to their compile-time handlers.
	BuiltinVariablesOnCompile = make(map[string]BuiltinVariableCompileHandler)

	// BuiltinVariablesOnAccess maps builtin variable names to their access-time handlers.
	BuiltinVariablesOnAccess = make(map[string]BuiltinVariableAccessHandler)
)
