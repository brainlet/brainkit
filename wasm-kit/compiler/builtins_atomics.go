// Ported from: assemblyscript/src/builtins.ts
// Atomic operation builtins: atomic.load, atomic.store, atomic.add/sub/and/or/xor/xchg,
// atomic.cmpxchg, atomic.wait, atomic.notify, atomic.fence
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// registerAtomicsBuiltins registers atomic operation builtin handlers.
func registerAtomicsBuiltins() {
	builtinFunctions[common.BuiltinNameAtomicLoad] = builtinAtomicLoad
	builtinFunctions[common.BuiltinNameAtomicStore] = builtinAtomicStore
	builtinFunctions[common.BuiltinNameAtomicAdd] = builtinAtomicAdd
	builtinFunctions[common.BuiltinNameAtomicSub] = builtinAtomicSub
	builtinFunctions[common.BuiltinNameAtomicAnd] = builtinAtomicAnd
	builtinFunctions[common.BuiltinNameAtomicOr] = builtinAtomicOr
	builtinFunctions[common.BuiltinNameAtomicXor] = builtinAtomicXor
	builtinFunctions[common.BuiltinNameAtomicXchg] = builtinAtomicXchg
	builtinFunctions[common.BuiltinNameAtomicCmpxchg] = builtinAtomicCmpxchg
	builtinFunctions[common.BuiltinNameAtomicWait] = builtinAtomicWait
	builtinFunctions[common.BuiltinNameAtomicNotify] = builtinAtomicNotify
	builtinFunctions[common.BuiltinNameAtomicFence] = builtinAtomicFence
	// memory.atomic.wait32/wait64 aliases
	builtinFunctions[common.BuiltinNameMemoryAtomicWait32] = builtinMemoryAtomicWait32
	builtinFunctions[common.BuiltinNameMemoryAtomicWait64] = builtinMemoryAtomicWait64
}

// atomic.load<T!>(ptr: usize, immOffset?: usize) -> T*
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_load (line 2932).
func builtinAtomicLoad(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureThreads))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 1, 2)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	contextualType := ctx.ContextualType
	typ := typeArguments[0]
	outType := typ
	if typ.IsIntegerValue() &&
		contextualType.IsIntegerValue() &&
		contextualType.Size > typ.Size {
		outType = contextualType
	}
	if !typ.IsIntegerValue() {
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			typeArgsRange(ctx), "atomic.load", typ.String(), "",
		)
		compiler.CurrentType = outType
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	var immOffset int32
	if len(operands) == 2 {
		immOffset = evaluateImmediateOffset(operands[1], compiler) // reports
	}
	if immOffset < 0 {
		compiler.CurrentType = outType
		return mod.Unreachable()
	}
	compiler.CurrentType = outType
	return mod.AtomicLoad(
		uint32(typ.ByteSize()),
		arg0,
		outType.ToRef(),
		uint32(immOffset),
		"",
	)
}

// atomic.store<T!>(offset: usize, value: T*, immOffset?: usize) -> void
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_store (line 2974).
func builtinAtomicStore(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureThreads))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsOptional(ctx, 2, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	contextualType := ctx.ContextualType
	typ := typeArguments[0]
	if !typ.IsIntegerValue() {
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			typeArgsRange(ctx), "atomic.store", typ.String(), "",
		)
		compiler.CurrentType = types.TypeVoid
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	var arg1 module.ExpressionRef
	if ctx.ContextIsExact {
		arg1 = compiler.CompileExpression(
			operands[1],
			contextualType,
			ConstraintsConvImplicit,
		)
	} else {
		constraints := ConstraintsConvImplicit
		if typ.IsIntegerValue() {
			constraints = ConstraintsNone // no need to convert to small int (but now might result in a float)
		}
		arg1 = compiler.CompileExpression(
			operands[1],
			typ,
			constraints,
		)
	}
	inType := compiler.CurrentType
	if typ.IsIntegerValue() &&
		(!inType.IsIntegerValue() || // float to int
			inType.Size < typ.Size) { // int to larger int (clear garbage bits)
		// either conversion or memory operation clears garbage bits
		arg1 = compiler.convertExpression(arg1, inType, typ, false, operands[1])
		inType = typ
	}
	var immOffset int32
	if len(operands) == 3 {
		immOffset = evaluateImmediateOffset(operands[2], compiler) // reports
	}
	if immOffset < 0 {
		compiler.CurrentType = types.TypeVoid
		return mod.Unreachable()
	}
	compiler.CurrentType = types.TypeVoid
	return mod.AtomicStore(uint32(typ.ByteSize()), arg0, arg1, inType.ToRef(), uint32(immOffset), "")
}

// builtinAtomicBinary handles any_atomic_binary<T!>(ptr, value: T, immOffset?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_binary (line 3031).
func builtinAtomicBinary(ctx *BuiltinFunctionContext, op module.Op, opName string) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureThreads))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 2, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	contextualType := ctx.ContextualType
	typ := typeArguments[0]
	if !typ.IsIntegerValue() || typ.Size < 8 {
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			typeArgsRange(ctx), opName, typ.String(), "",
		)
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(operands[0],
		compiler.Options().UsizeType(),
		ConstraintsConvImplicit,
	)
	var arg1 module.ExpressionRef
	if ctx.ContextIsExact {
		arg1 = compiler.CompileExpression(operands[1],
			contextualType,
			ConstraintsConvImplicit,
		)
	} else {
		constraints := ConstraintsConvImplicit
		if typ.IsIntegerValue() {
			constraints = ConstraintsNone // no need to convert to small int (but now might result in a float)
		}
		arg1 = compiler.CompileExpression(
			operands[1],
			typ,
			constraints,
		)
	}
	inType := compiler.CurrentType
	if typ.IsIntegerValue() &&
		(!inType.IsIntegerValue() || // float to int
			inType.Size < typ.Size) { // int to larger int (clear garbage bits)
		// either conversion or memory operation clears garbage bits
		arg1 = compiler.convertExpression(arg1, inType, typ, false, operands[1])
		inType = typ
	}
	var immOffset int32
	if len(operands) == 3 {
		immOffset = evaluateImmediateOffset(operands[2], compiler) // reports
	}
	if immOffset < 0 {
		compiler.CurrentType = inType
		return mod.Unreachable()
	}
	compiler.CurrentType = inType
	return mod.AtomicRMW(op, uint32(typ.ByteSize()), uint32(immOffset), arg0, arg1, inType.ToRef(), "")
}

// atomic.add<T!>(ptr, value: T, immOffset?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_add (line 3088).
func builtinAtomicAdd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinAtomicBinary(ctx, module.AtomicRMWOpAdd, "atomic.add")
}

// atomic.sub<T!>(ptr, value: T, immOffset?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_sub (line 3094).
func builtinAtomicSub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinAtomicBinary(ctx, module.AtomicRMWOpSub, "atomic.sub")
}

// atomic.and<T!>(ptr, value: T, immOffset?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_and (line 3100).
func builtinAtomicAnd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinAtomicBinary(ctx, module.AtomicRMWOpAnd, "atomic.and")
}

// atomic.or<T!>(ptr, value: T, immOffset?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_or (line 3106).
func builtinAtomicOr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinAtomicBinary(ctx, module.AtomicRMWOpOr, "atomic.or")
}

// atomic.xor<T!>(ptr, value: T, immOffset?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_xor (line 3112).
func builtinAtomicXor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinAtomicBinary(ctx, module.AtomicRMWOpXor, "atomic.xor")
}

// atomic.xchg<T!>(ptr, value: T, immOffset?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_xchg (line 3118).
func builtinAtomicXchg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinAtomicBinary(ctx, module.AtomicRMWOpXchg, "atomic.xchg")
}

// atomic.cmpxchg<T!>(ptr: usize, expected: T, replacement: T, off?: usize) -> T
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_cmpxchg (line 3124).
func builtinAtomicCmpxchg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureThreads))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 3, 4)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	contextualType := ctx.ContextualType
	typ := typeArguments[0]
	if !typ.IsIntegerValue() || typ.Size < 8 {
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			typeArgsRange(ctx), "atomic.cmpxchg", typ.String(), "",
		)
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(operands[0],
		compiler.Options().UsizeType(),
		ConstraintsConvImplicit,
	)
	var arg1 module.ExpressionRef
	if ctx.ContextIsExact {
		arg1 = compiler.CompileExpression(operands[1],
			contextualType,
			ConstraintsConvImplicit,
		)
	} else {
		constraints := ConstraintsConvImplicit
		if typ.IsIntegerValue() {
			constraints = ConstraintsNone // no need to convert to small int (but now might result in a float)
		}
		arg1 = compiler.CompileExpression(
			operands[1],
			typ,
			constraints,
		)
	}
	inType := compiler.CurrentType
	arg2 := compiler.CompileExpression(operands[2],
		inType,
		ConstraintsConvImplicit,
	)
	if typ.IsIntegerValue() &&
		(!inType.IsIntegerValue() || // float to int
			inType.Size < typ.Size) { // int to larger int (clear garbage bits)
		// either conversion or memory operation clears garbage bits
		arg1 = compiler.convertExpression(arg1, inType, typ, false, operands[1])
		arg2 = compiler.convertExpression(arg2, inType, typ, false, operands[2])
		inType = typ
	}
	var immOffset int32
	if len(operands) == 4 {
		immOffset = evaluateImmediateOffset(operands[3], compiler) // reports
	}
	if immOffset < 0 {
		compiler.CurrentType = inType
		return mod.Unreachable()
	}
	compiler.CurrentType = inType
	return mod.AtomicCmpxchg(uint32(typ.ByteSize()), uint32(immOffset), arg0, arg1, arg2, inType.ToRef(), "")
}

// atomic.wait<T!>(ptr: usize, expected: T, timeout?: i64) -> i32
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_wait (line 3187).
func builtinAtomicWait(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureThreads))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsOptional(ctx, 2, 3)) != 0 {
		compiler.CurrentType = types.TypeI32
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	typ := typeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit)
	var arg2 module.ExpressionRef
	if len(operands) == 3 {
		arg2 = compiler.CompileExpression(operands[2], types.TypeI64, ConstraintsConvImplicit)
	} else {
		arg2 = mod.I64(-1) // Infinite timeout: i64(-1, -1) = 0xFFFFFFFFFFFFFFFF
	}
	compiler.CurrentType = types.TypeI32
	switch typ.Kind {
	case types.TypeKindI32,
		types.TypeKindI64,
		types.TypeKindIsize,
		types.TypeKindU32,
		types.TypeKindU64,
		types.TypeKindUsize:
		return mod.AtomicWait(arg0, arg1, arg2, typ.ToRef(), "")
	}
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		typeArgsRange(ctx), "atomic.wait", typ.String(), "",
	)
	return mod.Unreachable()
}

// atomic.notify(ptr: usize, count?: i32) -> i32
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_notify (line 3224).
func builtinAtomicNotify(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureThreads))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsOptional(ctx, 1, 2)) != 0 {
		compiler.CurrentType = types.TypeI32
		return mod.Unreachable()
	}
	operands := ctx.Operands
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	var arg1 module.ExpressionRef
	if len(operands) == 2 {
		arg1 = compiler.CompileExpression(operands[1], types.TypeI32, ConstraintsConvImplicit)
	} else {
		arg1 = mod.I32(-1) // Infinity count of waiters
	}
	compiler.CurrentType = types.TypeI32
	return mod.AtomicNotify(arg0, arg1, "")
}

// atomic.fence() -> void
// Ported from: assemblyscript/src/builtins.ts builtin_atomic_fence (line 3246).
func builtinAtomicFence(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeVoid
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureThreads))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 0)) != 0 {
		return mod.Unreachable()
	}
	return mod.AtomicFence()
}

// memory.atomic.wait32 -> atomic.wait<i32>
// Ported from: assemblyscript/src/builtins.ts builtin_memory_atomic_wait32 (line 8535).
func builtinMemoryAtomicWait32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	return builtinAtomicWait(ctx)
}

// memory.atomic.wait64 -> atomic.wait<i64>
// Ported from: assemblyscript/src/builtins.ts builtin_memory_atomic_wait64 (line 8543).
func builtinMemoryAtomicWait64(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI32
	return builtinAtomicWait(ctx)
}
