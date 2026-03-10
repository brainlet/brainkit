// Ported from: assemblyscript/src/builtins.ts
// Control flow and GC builtins: select, unreachable, memory.size/grow/copy/fill,
// i31.new, i31.get, call_indirect, unchecked, inline.always, instantiate
package compiler

import (
	"math"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// registerControlBuiltins registers control flow and GC builtin handlers.
func registerControlBuiltins() {
	builtinFunctions[common.BuiltinNameSelect] = builtinSelect
	builtinFunctions[common.BuiltinNameUnreachable] = builtinUnreachable
	builtinFunctions[common.BuiltinNameMemorySize] = builtinMemorySize
	builtinFunctions[common.BuiltinNameMemoryGrow] = builtinMemoryGrow
	builtinFunctions[common.BuiltinNameMemoryCopy] = builtinMemoryCopy
	builtinFunctions[common.BuiltinNameMemoryFill] = builtinMemoryFill
	builtinFunctions[common.BuiltinNameI31New] = builtinI31New
	builtinFunctions[common.BuiltinNameI31Get] = builtinI31Get
	builtinFunctions[common.BuiltinNameCallIndirect] = builtinCallIndirect
	builtinFunctions[common.BuiltinNameUnchecked] = builtinUnchecked
	builtinFunctions[common.BuiltinNameInlineAlways] = builtinInlineAlways
	builtinFunctions[common.BuiltinNameInstantiate] = builtinInstantiate
}

// select<T?>(ifTrue: T, ifFalse: T, condition: bool) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_select (line 3262).
func builtinSelect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkTypeOptional(ctx, true))|
		boolToInt(checkArgsRequired(ctx, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	}
	typ := compiler.CurrentType
	if !typ.IsAny(types.TypeFlagValue | types.TypeFlagReference) {
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			typeArgsRange(ctx), "select", typ.String(), "",
		)
		return mod.Unreachable()
	}
	arg1 := compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit)
	arg2 := compiler.makeIsTrueish(
		compiler.CompileExpression(operands[2], types.TypeBool, ConstraintsNone),
		compiler.CurrentType,
		operands[2],
	)
	compiler.CurrentType = typ
	return mod.Select(arg0, arg1, arg2)
}

// unreachable() -> *
// Ported from: assemblyscript/src/builtins.ts builtin_unreachable (line 3294).
func builtinUnreachable(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	checkArgsRequired(ctx, 0)
	return ctx.Compiler.Module().Unreachable()
}

// memory.size() -> i32
// Ported from: assemblyscript/src/builtins.ts builtin_memory_size (line 3304).
func builtinMemorySize(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeI32
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 0)) != 0 {
		return mod.Unreachable()
	}
	return mod.MemorySize()
}

// memory.grow(pages: i32) -> i32
// Ported from: assemblyscript/src/builtins.ts builtin_memory_grow (line 3317).
func builtinMemoryGrow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeI32
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		return mod.Unreachable()
	}
	return mod.MemoryGrow(compiler.CompileExpression(ctx.Operands[0], types.TypeI32, ConstraintsConvImplicit))
}

// memory.copy(dest: usize, src: usize, n: usize) -> void
// Ported from: assemblyscript/src/builtins.ts builtin_memory_copy (line 3330).
func builtinMemoryCopy(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeVoid
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	if !compiler.Options().HasFeature(common.FeatureBulkMemory) {
		// use stdlib alternative if not supported
		instance := compiler.Resolver().ResolveFunction(ctx.Prototype, nil, nil, program.ReportModeReport)
		compiler.CurrentType = types.TypeVoid
		if instance == nil || !compiler.CompileFunction(instance) {
			return mod.Unreachable()
		}
		return compiler.compileCallDirect(instance, operands, ctx.ReportNode, 0, ConstraintsNone)
	}
	usizeType := compiler.Options().UsizeType()
	arg0 := compiler.CompileExpression(operands[0], usizeType, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(operands[1], usizeType, ConstraintsConvImplicit)
	arg2 := compiler.CompileExpression(operands[2], usizeType, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeVoid
	return mod.MemoryCopy(arg0, arg1, arg2, "", "")
}

// memory.fill(dest: usize, value: u8, n: usize) -> void
// Ported from: assemblyscript/src/builtins.ts builtin_memory_fill (line 3356).
func builtinMemoryFill(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeVoid
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	if !compiler.Options().HasFeature(common.FeatureBulkMemory) {
		// use stdlib alternative if not supported
		instance := compiler.Resolver().ResolveFunction(ctx.Prototype, nil, nil, program.ReportModeReport)
		compiler.CurrentType = types.TypeVoid
		if instance == nil || !compiler.CompileFunction(instance) {
			return mod.Unreachable()
		}
		return compiler.compileCallDirect(instance, operands, ctx.ReportNode, 0, ConstraintsNone)
	}
	usizeType := compiler.Options().UsizeType()
	arg0 := compiler.CompileExpression(operands[0], usizeType, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(operands[1], types.TypeU8, ConstraintsConvImplicit)
	arg2 := compiler.CompileExpression(operands[2], usizeType, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeVoid
	return mod.MemoryFill(arg0, arg1, arg2, "")
}

// i31.new(value: i32) -> i31ref
// Ported from: assemblyscript/src/builtins.ts builtin_i31_new (line 3496).
func builtinI31New(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	arg0 := compiler.CompileExpression(operands[0], types.TypeI32, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeI31
	return mod.RefI31(arg0)
}

// i31.get(value: i31ref) -> i32
// Ported from: assemblyscript/src/builtins.ts builtin_i31_get (line 3510).
func builtinI31Get(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	arg0 := compiler.CompileExpression(operands[0], types.TypeI31.AsNullable(), ConstraintsConvImplicit)
	if ctx.ContextualType.Is(types.TypeFlagUnsigned) {
		compiler.CurrentType = types.TypeU32
		return mod.I31Get(arg0, false)
	}
	compiler.CurrentType = types.TypeI32
	return mod.I31Get(arg0, true)
}

// unchecked(expr: *) -> *
// Ported from: assemblyscript/src/builtins.ts builtin_unchecked (line 3749).
func builtinUnchecked(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		return mod.Unreachable()
	}
	fl := compiler.CurrentFlow
	ignoreUnchecked := compiler.Options().UncheckedBehavior == int32(UncheckedBehaviorNever)
	alreadyUnchecked := fl.Is(flow.FlowFlagUncheckedContext)
	if ignoreUnchecked {
		if alreadyUnchecked {
			panic("builtins: unchecked context already set when UncheckedBehavior is Never")
		}
	} else {
		fl.SetFlag(flow.FlowFlagUncheckedContext)
	}
	// eliminate unnecessary tees by preferring contextualType(=void)
	expr := compiler.CompileExpression(ctx.Operands[0], ctx.ContextualType, ConstraintsNone)
	if !alreadyUnchecked {
		fl.UnsetFlag(flow.FlowFlagUncheckedContext)
	}
	return expr
}

// inline.always(expr: *) -> *
// Ported from: assemblyscript/src/builtins.ts builtin_inline_always (line 3769).
func builtinInlineAlways(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		return mod.Unreachable()
	}
	fl := compiler.CurrentFlow
	alreadyInline := fl.Is(flow.FlowFlagInlineContext)
	if !alreadyInline {
		fl.SetFlag(flow.FlowFlagInlineContext)
	}
	// eliminate unnecessary tees by preferring contextualType(=void)
	expr := compiler.CompileExpression(ctx.Operands[0], ctx.ContextualType, ConstraintsNone)
	if !alreadyInline {
		fl.UnsetFlag(flow.FlowFlagInlineContext)
	}
	return expr
}

// call_indirect<T?>(index: u32, ...args: *[]) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_call_indirect (line 3787).
func builtinCallIndirect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkTypeOptional(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 1, math.MaxInt32)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var returnType *types.Type
	if typeArguments != nil && len(typeArguments) > 0 {
		returnType = typeArguments[0]
	} else {
		returnType = ctx.ContextualType
	}
	indexArg := compiler.CompileExpression(operands[0], types.TypeU32, ConstraintsConvImplicit)
	numOperands := len(operands) - 1
	operandExprs := make([]module.ExpressionRef, numOperands)
	paramTypeRefs := make([]module.TypeRef, numOperands)
	for i := 0; i < numOperands; i++ {
		operandExprs[i] = compiler.CompileExpression(operands[1+i], types.TypeAuto, ConstraintsNone)
		if compiler.CurrentType.IsManaged() {
			operandExprs[i] = mod.Tostack(operandExprs[i])
		}
		paramTypeRefs[i] = compiler.CurrentType.ToRef()
	}
	compiler.CurrentType = returnType
	return mod.CallIndirect("" /* TODO */, indexArg, operandExprs, module.CreateType(paramTypeRefs), returnType.ToRef())
}

// instantiate<T!>(...args: *[]) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_instantiate (line 3820).
func builtinInstantiate(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeRequired(ctx, true) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	typeArgument := typeArguments[0]
	classInstance := typeArgument.GetClass()
	if classInstance == nil {
		compiler.Error(
			diagnostics.DiagnosticCodeThisExpressionIsNotConstructable,
			ctx.ReportNode.Expression.GetRange(),
			"", "", "",
		)
		return mod.Unreachable()
	}
	compiler.CurrentType = classInstance.GetType()
	ctor := compiler.ensureConstructor(classInstance.(*program.Class), ctx.ReportNode)
	compiler.checkFieldInitialization(classInstance.(*program.Class), ctx.ReportNode)
	return compiler.compileInstantiate(ctor, operands, ConstraintsNone, ctx.ReportNode)
}
