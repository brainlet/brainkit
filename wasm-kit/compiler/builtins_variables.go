// Ported from: assemblyscript/src/builtins.ts builtin variable sections.
package compiler

import (
	"math"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

func init() {
	registerBuiltinVariables()

	// Bridge program-side builtin validation to the live compiler registries.
	program.BuiltinFunctionLookup = func(name string) bool {
		_, ok := builtinFunctions[name]
		return ok
	}
	program.BuiltinVariableOnAccessLookup = func(name string) bool {
		_, ok := BuiltinVariablesOnAccess[name]
		return ok
	}
}

func registerBuiltinVariables() {
	BuiltinVariablesOnCompile[common.BuiltinNameNaN] = builtinNaNCompile
	BuiltinVariablesOnAccess[common.BuiltinNameNaN] = builtinNaNAccess

	BuiltinVariablesOnCompile[common.BuiltinNameInfinity] = builtinInfinityCompile
	BuiltinVariablesOnAccess[common.BuiltinNameInfinity] = builtinInfinityAccess

	BuiltinVariablesOnCompile[common.BuiltinNameHeapBase] = builtinHeapBaseCompile
	BuiltinVariablesOnAccess[common.BuiltinNameHeapBase] = builtinHeapBaseAccess

	BuiltinVariablesOnCompile[common.BuiltinNameDataEnd] = builtinDataEndCompile
	BuiltinVariablesOnAccess[common.BuiltinNameDataEnd] = builtinDataEndAccess

	BuiltinVariablesOnCompile[common.BuiltinNameStackPointer] = builtinStackPointerCompile
	BuiltinVariablesOnAccess[common.BuiltinNameStackPointer] = builtinStackPointerAccess

	BuiltinVariablesOnCompile[common.BuiltinNameRttiBase] = builtinRttiBaseCompile
	BuiltinVariablesOnAccess[common.BuiltinNameRttiBase] = builtinRttiBaseAccess
}

// NaN
func builtinNaNCompile(ctx *BuiltinVariableContext) {
	element := ctx.Element
	if element.Is(common.CommonFlagsModuleExport) {
		mod := ctx.Compiler.Module()
		mod.AddGlobal(element.GetInternalName(), module.TypeRefF64, false, mod.F64(math.NaN()))
	}
}

func builtinNaNAccess(ctx *BuiltinVariableContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if ctx.ContextualType == types.TypeF32 {
		compiler.CurrentType = types.TypeF32
		return mod.F32(float32(math.NaN()))
	}
	compiler.CurrentType = types.TypeF64
	return mod.F64(math.NaN())
}

// Infinity
func builtinInfinityCompile(ctx *BuiltinVariableContext) {
	element := ctx.Element
	if element.Is(common.CommonFlagsModuleExport) {
		mod := ctx.Compiler.Module()
		mod.AddGlobal(element.GetInternalName(), module.TypeRefF64, false, mod.F64(math.Inf(1)))
	}
}

func builtinInfinityAccess(ctx *BuiltinVariableContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if ctx.ContextualType == types.TypeF32 {
		compiler.CurrentType = types.TypeF32
		return mod.F32(float32(math.Inf(1)))
	}
	compiler.CurrentType = types.TypeF64
	return mod.F64(math.Inf(1))
}

// Runtime globals
func builtinHeapBaseCompile(ctx *BuiltinVariableContext) {
	compileRuntimeBuiltinGlobal(ctx, RuntimeFeaturesHeap)
}
func builtinHeapBaseAccess(ctx *BuiltinVariableContext) module.ExpressionRef {
	return accessRuntimeBuiltinGlobal(ctx, RuntimeFeaturesHeap)
}

func builtinDataEndCompile(ctx *BuiltinVariableContext) {
	compileRuntimeBuiltinGlobal(ctx, RuntimeFeaturesData)
}
func builtinDataEndAccess(ctx *BuiltinVariableContext) module.ExpressionRef {
	return accessRuntimeBuiltinGlobal(ctx, RuntimeFeaturesData)
}

func builtinStackPointerCompile(ctx *BuiltinVariableContext) {
	compileRuntimeBuiltinGlobal(ctx, RuntimeFeaturesStack)
}
func builtinStackPointerAccess(ctx *BuiltinVariableContext) module.ExpressionRef {
	return accessRuntimeBuiltinGlobal(ctx, RuntimeFeaturesStack)
}

func builtinRttiBaseCompile(ctx *BuiltinVariableContext) {
	compileRuntimeBuiltinGlobal(ctx, RuntimeFeaturesRtti)
}
func builtinRttiBaseAccess(ctx *BuiltinVariableContext) module.ExpressionRef {
	return accessRuntimeBuiltinGlobal(ctx, RuntimeFeaturesRtti)
}

func compileRuntimeBuiltinGlobal(ctx *BuiltinVariableContext, feature RuntimeFeatures) {
	compiler := ctx.Compiler
	mod := compiler.Module()
	element := ctx.Element
	typ := element.GetResolvedType()
	compiler.RuntimeFeatures |= feature
	mod.AddGlobal(element.GetInternalName(), typ.ToRef(), true, compiler.makeZeroOfType(typ))
}

func accessRuntimeBuiltinGlobal(ctx *BuiltinVariableContext, feature RuntimeFeatures) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	element := ctx.Element
	typ := element.GetResolvedType()
	compiler.RuntimeFeatures |= feature
	compiler.CurrentType = typ
	return mod.GlobalGet(element.GetInternalName(), typ.ToRef())
}
