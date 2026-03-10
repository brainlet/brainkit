// Ported from: assemblyscript/src/builtins.ts (lines 3935-4025)
// Portable type conversion builtins: i8, i16, i32, i64, isize, u8, u16, u32, u64, usize, bool, f32, f64.
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

func init() {
	builtinFunctions[common.BuiltinNameI8] = builtinI8
	builtinFunctions[common.BuiltinNameI16] = builtinI16
	builtinFunctions[common.BuiltinNameI32] = builtinI32
	builtinFunctions[common.BuiltinNameI64] = builtinI64
	builtinFunctions[common.BuiltinNameIsize] = builtinIsize
	builtinFunctions[common.BuiltinNameU8] = builtinU8
	builtinFunctions[common.BuiltinNameU16] = builtinU16
	builtinFunctions[common.BuiltinNameU32] = builtinU32
	builtinFunctions[common.BuiltinNameU64] = builtinU64
	builtinFunctions[common.BuiltinNameUsize] = builtinUsize
	builtinFunctions[common.BuiltinNameBool] = builtinBool
	builtinFunctions[common.BuiltinNameF32] = builtinF32
	builtinFunctions[common.BuiltinNameF64] = builtinF64
}

// builtinConversion is the shared helper for all portable type conversions.
// Ported from: assemblyscript/src/builtins.ts builtin_conversion (lines 3937-3947).
func builtinConversion(ctx *BuiltinFunctionContext, toType *types.Type) module.ExpressionRef {
	compiler := ctx.Compiler
	// Note: TS uses bitwise OR `|` to ensure both checks run and both errors are reported.
	hasError := boolToInt(checkTypeAbsent(ctx)) | boolToInt(checkArgsRequired(ctx, 1))
	if hasError != 0 {
		compiler.CurrentType = toType
		return compiler.Module().Unreachable()
	}
	return compiler.CompileExpression(ctx.Operands[0], toType, ConstraintsConvExplicit)
}

// builtinI8 implements i8(*) -> i8.
// Ported from: assemblyscript/src/builtins.ts builtin_i8 (lines 3950-3952).
func builtinI8(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeI8)
}

// builtinI16 implements i16(*) -> i16.
// Ported from: assemblyscript/src/builtins.ts builtin_i16 (lines 3956-3958).
func builtinI16(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeI16)
}

// builtinI32 implements i32(*) -> i32.
// Ported from: assemblyscript/src/builtins.ts builtin_i32 (lines 3962-3964).
func builtinI32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeI32)
}

// builtinI64 implements i64(*) -> i64.
// Ported from: assemblyscript/src/builtins.ts builtin_i64 (lines 3968-3970).
func builtinI64(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeI64)
}

// builtinIsize implements isize(*) -> isize.
// Ported from: assemblyscript/src/builtins.ts builtin_isize (lines 3974-3976).
func builtinIsize(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, ctx.Compiler.Options().IsizeType())
}

// builtinU8 implements u8(*) -> u8.
// Ported from: assemblyscript/src/builtins.ts builtin_u8 (lines 3980-3982).
func builtinU8(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeU8)
}

// builtinU16 implements u16(*) -> u16.
// Ported from: assemblyscript/src/builtins.ts builtin_u16 (lines 3986-3988).
func builtinU16(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeU16)
}

// builtinU32 implements u32(*) -> u32.
// Ported from: assemblyscript/src/builtins.ts builtin_u32 (lines 3992-3994).
func builtinU32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeU32)
}

// builtinU64 implements u64(*) -> u64.
// Ported from: assemblyscript/src/builtins.ts builtin_u64 (lines 3998-4000).
func builtinU64(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeU64)
}

// builtinUsize implements usize(*) -> usize.
// Ported from: assemblyscript/src/builtins.ts builtin_usize (lines 4004-4006).
func builtinUsize(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, ctx.Compiler.Options().UsizeType())
}

// builtinBool implements bool(*) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_bool (lines 4010-4012).
func builtinBool(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeBool)
}

// builtinF32 implements f32(*) -> f32.
// Ported from: assemblyscript/src/builtins.ts builtin_f32 (lines 4016-4018).
func builtinF32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeF32)
}

// builtinF64 implements f64(*) -> f64.
// Ported from: assemblyscript/src/builtins.ts builtin_f64 (lines 4022-4024).
func builtinF64(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinConversion(ctx, types.TypeF64)
}
