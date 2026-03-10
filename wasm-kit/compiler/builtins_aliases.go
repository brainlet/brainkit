// Ported from: assemblyscript/src/builtins.ts (lines 7130-10828)
// Inline assembler alias builtins: type-specific wrappers that delegate to generic builtins.
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

func registerAliasBuiltins() {
	// === Scalar integer/float aliases ===

	// i32/i64 clz, ctz, popcnt, rotl, rotr
	builtinFunctions[common.BuiltinNameI32Clz] = builtinI32Clz
	builtinFunctions[common.BuiltinNameI64Clz] = builtinI64Clz
	builtinFunctions[common.BuiltinNameI32Ctz] = builtinI32Ctz
	builtinFunctions[common.BuiltinNameI64Ctz] = builtinI64Ctz
	builtinFunctions[common.BuiltinNameI32Popcnt] = builtinI32Popcnt
	builtinFunctions[common.BuiltinNameI64Popcnt] = builtinI64Popcnt
	builtinFunctions[common.BuiltinNameI32Rotl] = builtinI32Rotl
	builtinFunctions[common.BuiltinNameI64Rotl] = builtinI64Rotl
	builtinFunctions[common.BuiltinNameI32Rotr] = builtinI32Rotr
	builtinFunctions[common.BuiltinNameI64Rotr] = builtinI64Rotr

	// f32/f64 abs, max, min, ceil, floor, copysign, nearest, sqrt, trunc
	builtinFunctions[common.BuiltinNameF32Abs] = builtinF32Abs
	builtinFunctions[common.BuiltinNameF64Abs] = builtinF64Abs
	builtinFunctions[common.BuiltinNameF32Max] = builtinF32Max
	builtinFunctions[common.BuiltinNameF64Max] = builtinF64Max
	builtinFunctions[common.BuiltinNameF32Min] = builtinF32Min
	builtinFunctions[common.BuiltinNameF64Min] = builtinF64Min
	builtinFunctions[common.BuiltinNameF32Ceil] = builtinF32Ceil
	builtinFunctions[common.BuiltinNameF64Ceil] = builtinF64Ceil
	builtinFunctions[common.BuiltinNameF32Floor] = builtinF32Floor
	builtinFunctions[common.BuiltinNameF64Floor] = builtinF64Floor
	builtinFunctions[common.BuiltinNameF32Copysign] = builtinF32Copysign
	builtinFunctions[common.BuiltinNameF64Copysign] = builtinF64Copysign
	builtinFunctions[common.BuiltinNameF32Nearest] = builtinF32Nearest
	builtinFunctions[common.BuiltinNameF64Nearest] = builtinF64Nearest
	builtinFunctions[common.BuiltinNameF32Sqrt] = builtinF32Sqrt
	builtinFunctions[common.BuiltinNameF64Sqrt] = builtinF64Sqrt
	builtinFunctions[common.BuiltinNameF32Trunc] = builtinF32Trunc
	builtinFunctions[common.BuiltinNameF64Trunc] = builtinF64Trunc

	// reinterpret
	builtinFunctions[common.BuiltinNameI32ReinterpretF32] = builtinI32ReinterpretF32
	builtinFunctions[common.BuiltinNameI64ReinterpretF64] = builtinI64ReinterpretF64
	builtinFunctions[common.BuiltinNameF32ReinterpretI32] = builtinF32ReinterpretI32
	builtinFunctions[common.BuiltinNameF64ReinterpretI64] = builtinF64ReinterpretI64

	// rem
	builtinFunctions[common.BuiltinNameI32RemS] = builtinI32RemS
	builtinFunctions[common.BuiltinNameI32RemU] = builtinI32RemU
	builtinFunctions[common.BuiltinNameI64RemS] = builtinI64RemS
	builtinFunctions[common.BuiltinNameI64RemU] = builtinI64RemU

	// add
	builtinFunctions[common.BuiltinNameI32Add] = builtinI32Add
	builtinFunctions[common.BuiltinNameI64Add] = builtinI64Add
	builtinFunctions[common.BuiltinNameF32Add] = builtinF32Add
	builtinFunctions[common.BuiltinNameF64Add] = builtinF64Add

	// sub
	builtinFunctions[common.BuiltinNameI32Sub] = builtinI32Sub
	builtinFunctions[common.BuiltinNameI64Sub] = builtinI64Sub
	builtinFunctions[common.BuiltinNameF32Sub] = builtinF32Sub
	builtinFunctions[common.BuiltinNameF64Sub] = builtinF64Sub

	// mul
	builtinFunctions[common.BuiltinNameI32Mul] = builtinI32Mul
	builtinFunctions[common.BuiltinNameI64Mul] = builtinI64Mul
	builtinFunctions[common.BuiltinNameF32Mul] = builtinF32Mul
	builtinFunctions[common.BuiltinNameF64Mul] = builtinF64Mul

	// div
	builtinFunctions[common.BuiltinNameI32DivS] = builtinI32DivS
	builtinFunctions[common.BuiltinNameI32DivU] = builtinI32DivU
	builtinFunctions[common.BuiltinNameI64DivS] = builtinI64DivS
	builtinFunctions[common.BuiltinNameI64DivU] = builtinI64DivU
	builtinFunctions[common.BuiltinNameF32Div] = builtinF32Div
	builtinFunctions[common.BuiltinNameF64Div] = builtinF64Div

	// eq (contextualType always i32)
	builtinFunctions[common.BuiltinNameI32Eq] = builtinI32Eq
	builtinFunctions[common.BuiltinNameI64Eq] = builtinI64Eq
	builtinFunctions[common.BuiltinNameF32Eq] = builtinF32Eq
	builtinFunctions[common.BuiltinNameF64Eq] = builtinF64Eq

	// ne (contextualType always i32)
	builtinFunctions[common.BuiltinNameI32Ne] = builtinI32Ne
	builtinFunctions[common.BuiltinNameI64Ne] = builtinI64Ne
	builtinFunctions[common.BuiltinNameF32Ne] = builtinF32Ne
	builtinFunctions[common.BuiltinNameF64Ne] = builtinF64Ne

	// === Load aliases ===
	builtinFunctions[common.BuiltinNameI32Load8S] = builtinI32Load8S
	builtinFunctions[common.BuiltinNameI32Load8U] = builtinI32Load8U
	builtinFunctions[common.BuiltinNameI32Load16S] = builtinI32Load16S
	builtinFunctions[common.BuiltinNameI32Load16U] = builtinI32Load16U
	builtinFunctions[common.BuiltinNameI32Load] = builtinI32Load
	builtinFunctions[common.BuiltinNameI64Load8S] = builtinI64Load8S
	builtinFunctions[common.BuiltinNameI64Load8U] = builtinI64Load8U
	builtinFunctions[common.BuiltinNameI64Load16S] = builtinI64Load16S
	builtinFunctions[common.BuiltinNameI64Load16U] = builtinI64Load16U
	builtinFunctions[common.BuiltinNameI64Load32S] = builtinI64Load32S
	builtinFunctions[common.BuiltinNameI64Load32U] = builtinI64Load32U
	builtinFunctions[common.BuiltinNameI64Load] = builtinI64Load
	builtinFunctions[common.BuiltinNameF32Load] = builtinF32Load
	builtinFunctions[common.BuiltinNameF64Load] = builtinF64Load

	// === Store aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32Store8] = builtinI32Store8
	builtinFunctions[common.BuiltinNameI32Store16] = builtinI32Store16
	builtinFunctions[common.BuiltinNameI32Store] = builtinI32Store
	builtinFunctions[common.BuiltinNameI64Store8] = builtinI64Store8
	builtinFunctions[common.BuiltinNameI64Store16] = builtinI64Store16
	builtinFunctions[common.BuiltinNameI64Store32] = builtinI64Store32
	builtinFunctions[common.BuiltinNameI64Store] = builtinI64Store
	builtinFunctions[common.BuiltinNameF32Store] = builtinF32Store
	builtinFunctions[common.BuiltinNameF64Store] = builtinF64Store

	// === Atomic load aliases ===
	builtinFunctions[common.BuiltinNameI32AtomicLoad8U] = builtinI32AtomicLoad8U
	builtinFunctions[common.BuiltinNameI32AtomicLoad16U] = builtinI32AtomicLoad16U
	builtinFunctions[common.BuiltinNameI32AtomicLoad] = builtinI32AtomicLoad
	builtinFunctions[common.BuiltinNameI64AtomicLoad8U] = builtinI64AtomicLoad8U
	builtinFunctions[common.BuiltinNameI64AtomicLoad16U] = builtinI64AtomicLoad16U
	builtinFunctions[common.BuiltinNameI64AtomicLoad32U] = builtinI64AtomicLoad32U
	builtinFunctions[common.BuiltinNameI64AtomicLoad] = builtinI64AtomicLoad

	// === Atomic store aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicStore8] = builtinI32AtomicStore8
	builtinFunctions[common.BuiltinNameI32AtomicStore16] = builtinI32AtomicStore16
	builtinFunctions[common.BuiltinNameI32AtomicStore] = builtinI32AtomicStore
	builtinFunctions[common.BuiltinNameI64AtomicStore8] = builtinI64AtomicStore8
	builtinFunctions[common.BuiltinNameI64AtomicStore16] = builtinI64AtomicStore16
	builtinFunctions[common.BuiltinNameI64AtomicStore32] = builtinI64AtomicStore32
	builtinFunctions[common.BuiltinNameI64AtomicStore] = builtinI64AtomicStore

	// === Atomic RMW add aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicRmw8AddU] = builtinI32AtomicRmw8AddU
	builtinFunctions[common.BuiltinNameI32AtomicRmw16AddU] = builtinI32AtomicRmw16AddU
	builtinFunctions[common.BuiltinNameI32AtomicRmwAdd] = builtinI32AtomicRmwAdd
	builtinFunctions[common.BuiltinNameI64AtomicRmw8AddU] = builtinI64AtomicRmw8AddU
	builtinFunctions[common.BuiltinNameI64AtomicRmw16AddU] = builtinI64AtomicRmw16AddU
	builtinFunctions[common.BuiltinNameI64AtomicRmw32AddU] = builtinI64AtomicRmw32AddU
	builtinFunctions[common.BuiltinNameI64AtomicRmwAdd] = builtinI64AtomicRmwAdd

	// === Atomic RMW sub aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicRmw8SubU] = builtinI32AtomicRmw8SubU
	builtinFunctions[common.BuiltinNameI32AtomicRmw16SubU] = builtinI32AtomicRmw16SubU
	builtinFunctions[common.BuiltinNameI32AtomicRmwSub] = builtinI32AtomicRmwSub
	builtinFunctions[common.BuiltinNameI64AtomicRmw8SubU] = builtinI64AtomicRmw8SubU
	builtinFunctions[common.BuiltinNameI64AtomicRmw16SubU] = builtinI64AtomicRmw16SubU
	builtinFunctions[common.BuiltinNameI64AtomicRmw32SubU] = builtinI64AtomicRmw32SubU
	builtinFunctions[common.BuiltinNameI64AtomicRmwSub] = builtinI64AtomicRmwSub

	// === Atomic RMW and aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicRmw8AndU] = builtinI32AtomicRmw8AndU
	builtinFunctions[common.BuiltinNameI32AtomicRmw16AndU] = builtinI32AtomicRmw16AndU
	builtinFunctions[common.BuiltinNameI32AtomicRmwAnd] = builtinI32AtomicRmwAnd
	builtinFunctions[common.BuiltinNameI64AtomicRmw8AndU] = builtinI64AtomicRmw8AndU
	builtinFunctions[common.BuiltinNameI64AtomicRmw16AndU] = builtinI64AtomicRmw16AndU
	builtinFunctions[common.BuiltinNameI64AtomicRmw32AndU] = builtinI64AtomicRmw32AndU
	builtinFunctions[common.BuiltinNameI64AtomicRmwAnd] = builtinI64AtomicRmwAnd

	// === Atomic RMW or aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicRmw8OrU] = builtinI32AtomicRmw8OrU
	builtinFunctions[common.BuiltinNameI32AtomicRmw16OrU] = builtinI32AtomicRmw16OrU
	builtinFunctions[common.BuiltinNameI32AtomicRmwOr] = builtinI32AtomicRmwOr
	builtinFunctions[common.BuiltinNameI64AtomicRmw8OrU] = builtinI64AtomicRmw8OrU
	builtinFunctions[common.BuiltinNameI64AtomicRmw16OrU] = builtinI64AtomicRmw16OrU
	builtinFunctions[common.BuiltinNameI64AtomicRmw32OrU] = builtinI64AtomicRmw32OrU
	builtinFunctions[common.BuiltinNameI64AtomicRmwOr] = builtinI64AtomicRmwOr

	// === Atomic RMW xor aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicRmw8XorU] = builtinI32AtomicRmw8XorU
	builtinFunctions[common.BuiltinNameI32AtomicRmw16XorU] = builtinI32AtomicRmw16XorU
	builtinFunctions[common.BuiltinNameI32AtomicRmwXor] = builtinI32AtomicRmwXor
	builtinFunctions[common.BuiltinNameI64AtomicRmw8XorU] = builtinI64AtomicRmw8XorU
	builtinFunctions[common.BuiltinNameI64AtomicRmw16XorU] = builtinI64AtomicRmw16XorU
	builtinFunctions[common.BuiltinNameI64AtomicRmw32XorU] = builtinI64AtomicRmw32XorU
	builtinFunctions[common.BuiltinNameI64AtomicRmwXor] = builtinI64AtomicRmwXor

	// === Atomic RMW xchg aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicRmw8XchgU] = builtinI32AtomicRmw8XchgU
	builtinFunctions[common.BuiltinNameI32AtomicRmw16XchgU] = builtinI32AtomicRmw16XchgU
	builtinFunctions[common.BuiltinNameI32AtomicRmwXchg] = builtinI32AtomicRmwXchg
	builtinFunctions[common.BuiltinNameI64AtomicRmw8XchgU] = builtinI64AtomicRmw8XchgU
	builtinFunctions[common.BuiltinNameI64AtomicRmw16XchgU] = builtinI64AtomicRmw16XchgU
	builtinFunctions[common.BuiltinNameI64AtomicRmw32XchgU] = builtinI64AtomicRmw32XchgU
	builtinFunctions[common.BuiltinNameI64AtomicRmwXchg] = builtinI64AtomicRmwXchg

	// === Atomic RMW cmpxchg aliases (contextIsExact = true) ===
	builtinFunctions[common.BuiltinNameI32AtomicRmw8CmpxchgU] = builtinI32AtomicRmw8CmpxchgU
	builtinFunctions[common.BuiltinNameI32AtomicRmw16CmpxchgU] = builtinI32AtomicRmw16CmpxchgU
	builtinFunctions[common.BuiltinNameI32AtomicRmwCmpxchg] = builtinI32AtomicRmwCmpxchg
	builtinFunctions[common.BuiltinNameI64AtomicRmw8CmpxchgU] = builtinI64AtomicRmw8CmpxchgU
	builtinFunctions[common.BuiltinNameI64AtomicRmw16CmpxchgU] = builtinI64AtomicRmw16CmpxchgU
	builtinFunctions[common.BuiltinNameI64AtomicRmw32CmpxchgU] = builtinI64AtomicRmw32CmpxchgU
	builtinFunctions[common.BuiltinNameI64AtomicRmwCmpxchg] = builtinI64AtomicRmwCmpxchg

	// NOTE: memory.atomic.wait32 and memory.atomic.wait64 are already registered
	// in builtins_atomics.go as builtinMemoryAtomicWait32/builtinMemoryAtomicWait64.

	// NOTE: v128/SIMD aliases (v128.*, i8x16.*, i16x8.*, i32x4.*, i64x2.*, f32x4.*, f64x2.*)
	// are registered in registerSIMDBuiltins() in builtins_simd.go.
}

// ========================================================================================
// Scalar integer/float alias functions
// ========================================================================================

// i32.clz -> clz<i32>
func builtinI32Clz(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinClz(ctx)
}

// i64.clz -> clz<i64>
func builtinI64Clz(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinClz(ctx)
}

// i32.ctz -> ctz<i32>
func builtinI32Ctz(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinCtz(ctx)
}

// i64.ctz -> ctz<i64>
func builtinI64Ctz(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinCtz(ctx)
}

// i32.popcnt -> popcnt<i32>
func builtinI32Popcnt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinPopcnt(ctx)
}

// i64.popcnt -> popcnt<i64>
func builtinI64Popcnt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinPopcnt(ctx)
}

// i32.rotl -> rotl<i32>
func builtinI32Rotl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinRotl(ctx)
}

// i64.rotl -> rotl<i64>
func builtinI64Rotl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinRotl(ctx)
}

// i32.rotr -> rotr<i32>
func builtinI32Rotr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinRotr(ctx)
}

// i64.rotr -> rotr<i64>
func builtinI64Rotr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinRotr(ctx)
}

// f32.abs -> abs<f32>
func builtinF32Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinAbs(ctx)
}

// f64.abs -> abs<f64>
func builtinF64Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinAbs(ctx)
}

// f32.max -> max<f32>
func builtinF32Max(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinMax(ctx)
}

// f64.max -> max<f64>
func builtinF64Max(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinMax(ctx)
}

// f32.min -> min<f32>
func builtinF32Min(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinMin(ctx)
}

// f64.min -> min<f64>
func builtinF64Min(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinMin(ctx)
}

// f32.ceil -> ceil<f32>
func builtinF32Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinCeil(ctx)
}

// f64.ceil -> ceil<f64>
func builtinF64Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinCeil(ctx)
}

// f32.floor -> floor<f32>
func builtinF32Floor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinFloor(ctx)
}

// f64.floor -> floor<f64>
func builtinF64Floor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinFloor(ctx)
}

// f32.copysign -> copysign<f32>
func builtinF32Copysign(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinCopysign(ctx)
}

// f64.copysign -> copysign<f64>
func builtinF64Copysign(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinCopysign(ctx)
}

// f32.nearest -> nearest<f32>
func builtinF32Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinNearest(ctx)
}

// f64.nearest -> nearest<f64>
func builtinF64Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinNearest(ctx)
}

// i32.reinterpret_f32 -> reinterpret<i32>
func builtinI32ReinterpretF32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeF32
	return builtinReinterpret(ctx)
}

// i64.reinterpret_f64 -> reinterpret<i64>
func builtinI64ReinterpretF64(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeF64
	return builtinReinterpret(ctx)
}

// f32.reinterpret_i32 -> reinterpret<f32>
func builtinF32ReinterpretI32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeI32
	return builtinReinterpret(ctx)
}

// f64.reinterpret_i64 -> reinterpret<f64>
func builtinF64ReinterpretI64(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeI64
	return builtinReinterpret(ctx)
}

// f32.sqrt -> sqrt<f32>
func builtinF32Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinSqrt(ctx)
}

// f64.sqrt -> sqrt<f64>
func builtinF64Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinSqrt(ctx)
}

// f32.trunc -> trunc<f32>
func builtinF32Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinTrunc(ctx)
}

// f64.trunc -> trunc<f64>
func builtinF64Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinTrunc(ctx)
}

// ========================================================================================
// Operator alias functions (rem, add, sub, mul, div, eq, ne)
// ========================================================================================

// i32.rem_s -> rem<i32>
func builtinI32RemS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinRem(ctx)
}

// i32.rem_u -> rem<u32>
func builtinI32RemU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeU32
	return builtinRem(ctx)
}

// i64.rem_s -> rem<i64>
func builtinI64RemS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinRem(ctx)
}

// i64.rem_u -> rem<u64>
func builtinI64RemU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU64}
	ctx.ContextualType = types.TypeU64
	return builtinRem(ctx)
}

// i32.add -> add<i32>
func builtinI32Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinAdd(ctx)
}

// i64.add -> add<i64>
func builtinI64Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinAdd(ctx)
}

// f32.add -> add<f32>
func builtinF32Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinAdd(ctx)
}

// f64.add -> add<f64>
func builtinF64Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinAdd(ctx)
}

// i32.sub -> sub<i32>
func builtinI32Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinSub(ctx)
}

// i64.sub -> sub<i64>
func builtinI64Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinSub(ctx)
}

// f32.sub -> sub<f32>
func builtinF32Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinSub(ctx)
}

// f64.sub -> sub<f64>
func builtinF64Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinSub(ctx)
}

// i32.mul -> mul<i32>
func builtinI32Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinMul(ctx)
}

// i64.mul -> mul<i64>
func builtinI64Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinMul(ctx)
}

// f32.mul -> mul<f32>
func builtinF32Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinMul(ctx)
}

// f64.mul -> mul<f64>
func builtinF64Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinMul(ctx)
}

// i32.div_s -> div<i32>
func builtinI32DivS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinDiv(ctx)
}

// i32.div_u -> div<u32>
func builtinI32DivU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeU32
	return builtinDiv(ctx)
}

// i64.div_s -> div<i64>
func builtinI64DivS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinDiv(ctx)
}

// i64.div_u -> div<u64>
func builtinI64DivU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU64}
	ctx.ContextualType = types.TypeU64
	return builtinDiv(ctx)
}

// f32.div -> div<f32>
func builtinF32Div(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinDiv(ctx)
}

// f64.div -> div<f64>
func builtinF64Div(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinDiv(ctx)
}

// i32.eq -> eq<i32> (contextualType=i32)
func builtinI32Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinEq(ctx)
}

// i64.eq -> eq<i64> (contextualType=i32)
func builtinI64Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI32
	return builtinEq(ctx)
}

// f32.eq -> eq<f32> (contextualType=i32)
func builtinF32Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeI32
	return builtinEq(ctx)
}

// f64.eq -> eq<f64> (contextualType=i32)
func builtinF64Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeI32
	return builtinEq(ctx)
}

// i32.ne -> ne<i32> (contextualType=i32)
func builtinI32Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinNe(ctx)
}

// i64.ne -> ne<i64> (contextualType=i32)
func builtinI64Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI32
	return builtinNe(ctx)
}

// f32.ne -> ne<f32> (contextualType=i32)
func builtinF32Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeI32
	return builtinNe(ctx)
}

// f64.ne -> ne<f64> (contextualType=i32)
func builtinF64Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeI32
	return builtinNe(ctx)
}

// ========================================================================================
// Load alias functions
// ========================================================================================

func builtinI32Load8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI32
	return builtinLoad(ctx)
}

func builtinI32Load8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	return builtinLoad(ctx)
}

func builtinI32Load16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI32
	return builtinLoad(ctx)
}

func builtinI32Load16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	return builtinLoad(ctx)
}

func builtinI32Load(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinLoad(ctx)
}

func builtinI64Load8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI64
	return builtinLoad(ctx)
}

func builtinI64Load8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	return builtinLoad(ctx)
}

func builtinI64Load16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI64
	return builtinLoad(ctx)
}

func builtinI64Load16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	return builtinLoad(ctx)
}

func builtinI64Load32S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI64
	return builtinLoad(ctx)
}

func builtinI64Load32U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	return builtinLoad(ctx)
}

func builtinI64Load(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinLoad(ctx)
}

func builtinF32Load(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinLoad(ctx)
}

func builtinF64Load(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinLoad(ctx)
}

// ========================================================================================
// Store alias functions (contextIsExact = true)
// ========================================================================================

func builtinI32Store8(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinI32Store16(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinI32Store(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinI64Store8(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinI64Store16(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinI64Store32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinI64Store(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinF32Store(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

func builtinF64Store(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

// ========================================================================================
// Atomic load alias functions
// ========================================================================================

func builtinI32AtomicLoad8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	return builtinAtomicLoad(ctx)
}

func builtinI32AtomicLoad16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	return builtinAtomicLoad(ctx)
}

func builtinI32AtomicLoad(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinAtomicLoad(ctx)
}

func builtinI64AtomicLoad8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	return builtinAtomicLoad(ctx)
}

func builtinI64AtomicLoad16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	return builtinAtomicLoad(ctx)
}

func builtinI64AtomicLoad32U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	return builtinAtomicLoad(ctx)
}

func builtinI64AtomicLoad(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinAtomicLoad(ctx)
}

// ========================================================================================
// Atomic store alias functions (contextIsExact = true)
// ========================================================================================

func builtinI32AtomicStore8(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicStore(ctx)
}

func builtinI32AtomicStore16(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicStore(ctx)
}

func builtinI32AtomicStore(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicStore(ctx)
}

func builtinI64AtomicStore8(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicStore(ctx)
}

// Note: TS source uses Type.u16 for i64.atomic.store16 (line 8017)
func builtinI64AtomicStore16(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicStore(ctx)
}

func builtinI64AtomicStore32(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicStore(ctx)
}

func builtinI64AtomicStore(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicStore(ctx)
}

// ========================================================================================
// Atomic RMW alias functions (all contextIsExact = true)
// ========================================================================================

// --- atomic add ---

func builtinI32AtomicRmw8AddU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicAdd(ctx)
}

func builtinI32AtomicRmw16AddU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicAdd(ctx)
}

func builtinI32AtomicRmwAdd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicAdd(ctx)
}

func builtinI64AtomicRmw8AddU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAdd(ctx)
}

func builtinI64AtomicRmw16AddU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAdd(ctx)
}

func builtinI64AtomicRmw32AddU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAdd(ctx)
}

func builtinI64AtomicRmwAdd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAdd(ctx)
}

// --- atomic sub ---

func builtinI32AtomicRmw8SubU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicSub(ctx)
}

func builtinI32AtomicRmw16SubU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicSub(ctx)
}

func builtinI32AtomicRmwSub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicSub(ctx)
}

func builtinI64AtomicRmw8SubU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicSub(ctx)
}

func builtinI64AtomicRmw16SubU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicSub(ctx)
}

func builtinI64AtomicRmw32SubU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicSub(ctx)
}

func builtinI64AtomicRmwSub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicSub(ctx)
}

// --- atomic and ---

func builtinI32AtomicRmw8AndU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicAnd(ctx)
}

func builtinI32AtomicRmw16AndU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicAnd(ctx)
}

func builtinI32AtomicRmwAnd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicAnd(ctx)
}

func builtinI64AtomicRmw8AndU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAnd(ctx)
}

func builtinI64AtomicRmw16AndU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAnd(ctx)
}

func builtinI64AtomicRmw32AndU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAnd(ctx)
}

func builtinI64AtomicRmwAnd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicAnd(ctx)
}

// --- atomic or ---

func builtinI32AtomicRmw8OrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicOr(ctx)
}

func builtinI32AtomicRmw16OrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicOr(ctx)
}

func builtinI32AtomicRmwOr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicOr(ctx)
}

func builtinI64AtomicRmw8OrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicOr(ctx)
}

func builtinI64AtomicRmw16OrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicOr(ctx)
}

func builtinI64AtomicRmw32OrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicOr(ctx)
}

func builtinI64AtomicRmwOr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicOr(ctx)
}

// --- atomic xor ---

func builtinI32AtomicRmw8XorU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicXor(ctx)
}

func builtinI32AtomicRmw16XorU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicXor(ctx)
}

func builtinI32AtomicRmwXor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicXor(ctx)
}

func builtinI64AtomicRmw8XorU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXor(ctx)
}

func builtinI64AtomicRmw16XorU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXor(ctx)
}

func builtinI64AtomicRmw32XorU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXor(ctx)
}

func builtinI64AtomicRmwXor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXor(ctx)
}

// --- atomic xchg ---

func builtinI32AtomicRmw8XchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicXchg(ctx)
}

func builtinI32AtomicRmw16XchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicXchg(ctx)
}

func builtinI32AtomicRmwXchg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicXchg(ctx)
}

func builtinI64AtomicRmw8XchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXchg(ctx)
}

func builtinI64AtomicRmw16XchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXchg(ctx)
}

func builtinI64AtomicRmw32XchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXchg(ctx)
}

func builtinI64AtomicRmwXchg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicXchg(ctx)
}

// --- atomic cmpxchg ---

func builtinI32AtomicRmw8CmpxchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicCmpxchg(ctx)
}

func builtinI32AtomicRmw16CmpxchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicCmpxchg(ctx)
}

func builtinI32AtomicRmwCmpxchg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	ctx.ContextIsExact = true
	return builtinAtomicCmpxchg(ctx)
}

func builtinI64AtomicRmw8CmpxchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicCmpxchg(ctx)
}

func builtinI64AtomicRmw16CmpxchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicCmpxchg(ctx)
}

func builtinI64AtomicRmw32CmpxchgU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicCmpxchg(ctx)
}

func builtinI64AtomicRmwCmpxchg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	ctx.ContextIsExact = true
	return builtinAtomicCmpxchg(ctx)
}
