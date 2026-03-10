// Ported from: assemblyscript/src/builtins.ts (lines 8551-10828)
// SIMD/v128 inline assembler alias builtins + stub forward declarations for unported core builtins.
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// registerSIMDBuiltins registers all SIMD/v128 builtin alias functions.
func registerSIMDBuiltins() {
	// v128 load/store aliases
	builtinFunctions[common.BuiltinNameV128Load] = builtinV128Load
	builtinFunctions[common.BuiltinNameV128Load8x8S] = builtinV128Load8x8S
	builtinFunctions[common.BuiltinNameV128Load8x8U] = builtinV128Load8x8U
	builtinFunctions[common.BuiltinNameV128Load16x4S] = builtinV128Load16x4S
	builtinFunctions[common.BuiltinNameV128Load16x4U] = builtinV128Load16x4U
	builtinFunctions[common.BuiltinNameV128Load32x2S] = builtinV128Load32x2S
	builtinFunctions[common.BuiltinNameV128Load32x2U] = builtinV128Load32x2U
	builtinFunctions[common.BuiltinNameV128Load8Splat] = builtinV128Load8Splat
	builtinFunctions[common.BuiltinNameV128Load16Splat] = builtinV128Load16Splat
	builtinFunctions[common.BuiltinNameV128Load32Splat] = builtinV128Load32Splat
	builtinFunctions[common.BuiltinNameV128Load64Splat] = builtinV128Load64Splat
	builtinFunctions[common.BuiltinNameV128Load32Zero] = builtinV128Load32Zero
	builtinFunctions[common.BuiltinNameV128Load64Zero] = builtinV128Load64Zero
	builtinFunctions[common.BuiltinNameV128Load8Lane] = builtinV128Load8Lane
	builtinFunctions[common.BuiltinNameV128Load16Lane] = builtinV128Load16Lane
	builtinFunctions[common.BuiltinNameV128Load32Lane] = builtinV128Load32Lane
	builtinFunctions[common.BuiltinNameV128Load64Lane] = builtinV128Load64Lane
	builtinFunctions[common.BuiltinNameV128Store8Lane] = builtinV128Store8Lane
	builtinFunctions[common.BuiltinNameV128Store16Lane] = builtinV128Store16Lane
	builtinFunctions[common.BuiltinNameV128Store32Lane] = builtinV128Store32Lane
	builtinFunctions[common.BuiltinNameV128Store64Lane] = builtinV128Store64Lane
	builtinFunctions[common.BuiltinNameV128Store] = builtinV128Store

	// v128 core operations (generic, type-parameterized)
	builtinFunctions[common.BuiltinNameV128Splat] = builtinV128Splat
	builtinFunctions[common.BuiltinNameV128ExtractLane] = builtinV128ExtractLane
	builtinFunctions[common.BuiltinNameV128ReplaceLane] = builtinV128ReplaceLane
	builtinFunctions[common.BuiltinNameV128Shuffle] = builtinV128Shuffle
	builtinFunctions[common.BuiltinNameV128Swizzle] = builtinV128Swizzle
	builtinFunctions[common.BuiltinNameV128Add] = builtinV128Add
	builtinFunctions[common.BuiltinNameV128Sub] = builtinV128Sub
	builtinFunctions[common.BuiltinNameV128Mul] = builtinV128Mul
	builtinFunctions[common.BuiltinNameV128DivOp] = builtinV128Div
	builtinFunctions[common.BuiltinNameV128Neg] = builtinV128Neg
	builtinFunctions[common.BuiltinNameV128AddSat] = builtinV128AddSat
	builtinFunctions[common.BuiltinNameV128SubSat] = builtinV128SubSat
	builtinFunctions[common.BuiltinNameV128Shl] = builtinV128Shl
	builtinFunctions[common.BuiltinNameV128Shr] = builtinV128Shr
	builtinFunctions[common.BuiltinNameV128And] = builtinV128And
	builtinFunctions[common.BuiltinNameV128Or] = builtinV128Or
	builtinFunctions[common.BuiltinNameV128Xor] = builtinV128Xor
	builtinFunctions[common.BuiltinNameV128Andnot] = builtinV128Andnot
	builtinFunctions[common.BuiltinNameV128Not] = builtinV128Not
	builtinFunctions[common.BuiltinNameV128Bitselect] = builtinV128Bitselect
	builtinFunctions[common.BuiltinNameV128AnyTrue] = builtinV128AnyTrue
	builtinFunctions[common.BuiltinNameV128AllTrue] = builtinV128AllTrue
	builtinFunctions[common.BuiltinNameV128Bitmask] = builtinV128Bitmask
	builtinFunctions[common.BuiltinNameV128PopcntOp] = builtinV128Popcnt
	builtinFunctions[common.BuiltinNameV128MinOp] = builtinV128Min
	builtinFunctions[common.BuiltinNameV128MaxOp] = builtinV128Max
	builtinFunctions[common.BuiltinNameV128Pmin] = builtinV128Pmin
	builtinFunctions[common.BuiltinNameV128Pmax] = builtinV128Pmax
	builtinFunctions[common.BuiltinNameV128Dot] = builtinV128Dot
	builtinFunctions[common.BuiltinNameV128Avgr] = builtinV128Avgr
	builtinFunctions[common.BuiltinNameV128AbsOp] = builtinV128Abs
	builtinFunctions[common.BuiltinNameV128SqrtOp] = builtinV128Sqrt
	builtinFunctions[common.BuiltinNameV128CeilOp] = builtinV128Ceil
	builtinFunctions[common.BuiltinNameV128FloorOp] = builtinV128Floor
	builtinFunctions[common.BuiltinNameV128TruncOp] = builtinV128Trunc
	builtinFunctions[common.BuiltinNameV128NearestOp] = builtinV128Nearest
	builtinFunctions[common.BuiltinNameV128EqOp] = builtinV128Eq
	builtinFunctions[common.BuiltinNameV128NeOp] = builtinV128Ne
	builtinFunctions[common.BuiltinNameV128Lt] = builtinV128Lt
	builtinFunctions[common.BuiltinNameV128Le] = builtinV128Le
	builtinFunctions[common.BuiltinNameV128Gt] = builtinV128Gt
	builtinFunctions[common.BuiltinNameV128Ge] = builtinV128Ge
	builtinFunctions[common.BuiltinNameV128Convert] = builtinV128Convert
	builtinFunctions[common.BuiltinNameV128ConvertLow] = builtinV128ConvertLow
	builtinFunctions[common.BuiltinNameV128TruncSat] = builtinV128TruncSat
	builtinFunctions[common.BuiltinNameV128TruncSatZero] = builtinV128TruncSatZero
	builtinFunctions[common.BuiltinNameV128Narrow] = builtinV128Narrow
	builtinFunctions[common.BuiltinNameV128ExtendLow] = builtinV128ExtendLow
	builtinFunctions[common.BuiltinNameV128ExtendHigh] = builtinV128ExtendHigh
	builtinFunctions[common.BuiltinNameV128ExtaddPairwise] = builtinV128ExtaddPairwise
	builtinFunctions[common.BuiltinNameV128DemoteZero] = builtinV128DemoteZero
	builtinFunctions[common.BuiltinNameV128PromoteLow] = builtinV128PromoteLow
	builtinFunctions[common.BuiltinNameV128Q15mulrSat] = builtinV128Q15mulrSat
	builtinFunctions[common.BuiltinNameV128ExtmulLow] = builtinV128ExtmulLow
	builtinFunctions[common.BuiltinNameV128ExtmulHigh] = builtinV128ExtmulHigh
	builtinFunctions[common.BuiltinNameV128RelaxedSwizzle] = builtinV128RelaxedSwizzle
	builtinFunctions[common.BuiltinNameV128RelaxedTrunc] = builtinV128RelaxedTrunc
	builtinFunctions[common.BuiltinNameV128RelaxedTruncZero] = builtinV128RelaxedTruncZero
	builtinFunctions[common.BuiltinNameV128RelaxedMadd] = builtinV128RelaxedMadd
	builtinFunctions[common.BuiltinNameV128RelaxedNmadd] = builtinV128RelaxedNmadd
	builtinFunctions[common.BuiltinNameV128RelaxedLaneselect] = builtinV128RelaxedLaneselect
	builtinFunctions[common.BuiltinNameV128RelaxedMin] = builtinV128RelaxedMin
	builtinFunctions[common.BuiltinNameV128RelaxedMax] = builtinV128RelaxedMax
	builtinFunctions[common.BuiltinNameV128RelaxedQ15mulr] = builtinV128RelaxedQ15mulr
	builtinFunctions[common.BuiltinNameV128RelaxedDot] = builtinV128RelaxedDot
	builtinFunctions[common.BuiltinNameV128RelaxedDotAdd] = builtinV128RelaxedDotAdd

	// i8x16 aliases
	builtinFunctions[common.BuiltinNameI8x16Splat] = builtinI8x16Splat
	builtinFunctions[common.BuiltinNameI8x16ExtractLaneS] = builtinI8x16ExtractLaneS
	builtinFunctions[common.BuiltinNameI8x16ExtractLaneU] = builtinI8x16ExtractLaneU
	builtinFunctions[common.BuiltinNameI8x16ReplaceLane] = builtinI8x16ReplaceLane
	builtinFunctions[common.BuiltinNameI8x16Add] = builtinI8x16Add
	builtinFunctions[common.BuiltinNameI8x16Sub] = builtinI8x16Sub
	builtinFunctions[common.BuiltinNameI8x16MinS] = builtinI8x16MinS
	builtinFunctions[common.BuiltinNameI8x16MinU] = builtinI8x16MinU
	builtinFunctions[common.BuiltinNameI8x16MaxS] = builtinI8x16MaxS
	builtinFunctions[common.BuiltinNameI8x16MaxU] = builtinI8x16MaxU
	builtinFunctions[common.BuiltinNameI8x16AvgrU] = builtinI8x16AvgrU
	builtinFunctions[common.BuiltinNameI8x16Abs] = builtinI8x16Abs
	builtinFunctions[common.BuiltinNameI8x16Neg] = builtinI8x16Neg
	builtinFunctions[common.BuiltinNameI8x16AddSatS] = builtinI8x16AddSatS
	builtinFunctions[common.BuiltinNameI8x16AddSatU] = builtinI8x16AddSatU
	builtinFunctions[common.BuiltinNameI8x16SubSatS] = builtinI8x16SubSatS
	builtinFunctions[common.BuiltinNameI8x16SubSatU] = builtinI8x16SubSatU
	builtinFunctions[common.BuiltinNameI8x16Shl] = builtinI8x16Shl
	builtinFunctions[common.BuiltinNameI8x16ShrS] = builtinI8x16ShrS
	builtinFunctions[common.BuiltinNameI8x16ShrU] = builtinI8x16ShrU
	builtinFunctions[common.BuiltinNameI8x16AllTrue] = builtinI8x16AllTrue
	builtinFunctions[common.BuiltinNameI8x16Bitmask] = builtinI8x16Bitmask
	builtinFunctions[common.BuiltinNameI8x16Popcnt] = builtinI8x16Popcnt
	builtinFunctions[common.BuiltinNameI8x16Eq] = builtinI8x16Eq
	builtinFunctions[common.BuiltinNameI8x16Ne] = builtinI8x16Ne
	builtinFunctions[common.BuiltinNameI8x16LtS] = builtinI8x16LtS
	builtinFunctions[common.BuiltinNameI8x16LtU] = builtinI8x16LtU
	builtinFunctions[common.BuiltinNameI8x16LeS] = builtinI8x16LeS
	builtinFunctions[common.BuiltinNameI8x16LeU] = builtinI8x16LeU
	builtinFunctions[common.BuiltinNameI8x16GtS] = builtinI8x16GtS
	builtinFunctions[common.BuiltinNameI8x16GtU] = builtinI8x16GtU
	builtinFunctions[common.BuiltinNameI8x16GeS] = builtinI8x16GeS
	builtinFunctions[common.BuiltinNameI8x16GeU] = builtinI8x16GeU
	builtinFunctions[common.BuiltinNameI8x16NarrowI16x8S] = builtinI8x16NarrowI16x8S
	builtinFunctions[common.BuiltinNameI8x16NarrowI16x8U] = builtinI8x16NarrowI16x8U
	builtinFunctions[common.BuiltinNameI8x16Shuffle] = builtinI8x16Shuffle
	builtinFunctions[common.BuiltinNameI8x16SwizzleOp] = builtinI8x16Swizzle

	// i16x8 aliases
	builtinFunctions[common.BuiltinNameI16x8Splat] = builtinI16x8Splat
	builtinFunctions[common.BuiltinNameI16x8ExtractLaneS] = builtinI16x8ExtractLaneS
	builtinFunctions[common.BuiltinNameI16x8ExtractLaneU] = builtinI16x8ExtractLaneU
	builtinFunctions[common.BuiltinNameI16x8ReplaceLane] = builtinI16x8ReplaceLane
	builtinFunctions[common.BuiltinNameI16x8Add] = builtinI16x8Add
	builtinFunctions[common.BuiltinNameI16x8Sub] = builtinI16x8Sub
	builtinFunctions[common.BuiltinNameI16x8Mul] = builtinI16x8Mul
	builtinFunctions[common.BuiltinNameI16x8MinS] = builtinI16x8MinS
	builtinFunctions[common.BuiltinNameI16x8MinU] = builtinI16x8MinU
	builtinFunctions[common.BuiltinNameI16x8MaxS] = builtinI16x8MaxS
	builtinFunctions[common.BuiltinNameI16x8MaxU] = builtinI16x8MaxU
	builtinFunctions[common.BuiltinNameI16x8AvgrU] = builtinI16x8AvgrU
	builtinFunctions[common.BuiltinNameI16x8Abs] = builtinI16x8Abs
	builtinFunctions[common.BuiltinNameI16x8Neg] = builtinI16x8Neg
	builtinFunctions[common.BuiltinNameI16x8AddSatS] = builtinI16x8AddSatS
	builtinFunctions[common.BuiltinNameI16x8AddSatU] = builtinI16x8AddSatU
	builtinFunctions[common.BuiltinNameI16x8SubSatS] = builtinI16x8SubSatS
	builtinFunctions[common.BuiltinNameI16x8SubSatU] = builtinI16x8SubSatU
	builtinFunctions[common.BuiltinNameI16x8Shl] = builtinI16x8Shl
	builtinFunctions[common.BuiltinNameI16x8ShrS] = builtinI16x8ShrS
	builtinFunctions[common.BuiltinNameI16x8ShrU] = builtinI16x8ShrU
	builtinFunctions[common.BuiltinNameI16x8AllTrue] = builtinI16x8AllTrue
	builtinFunctions[common.BuiltinNameI16x8Bitmask] = builtinI16x8Bitmask
	builtinFunctions[common.BuiltinNameI16x8Eq] = builtinI16x8Eq
	builtinFunctions[common.BuiltinNameI16x8Ne] = builtinI16x8Ne
	builtinFunctions[common.BuiltinNameI16x8LtS] = builtinI16x8LtS
	builtinFunctions[common.BuiltinNameI16x8LtU] = builtinI16x8LtU
	builtinFunctions[common.BuiltinNameI16x8LeS] = builtinI16x8LeS
	builtinFunctions[common.BuiltinNameI16x8LeU] = builtinI16x8LeU
	builtinFunctions[common.BuiltinNameI16x8GtS] = builtinI16x8GtS
	builtinFunctions[common.BuiltinNameI16x8GtU] = builtinI16x8GtU
	builtinFunctions[common.BuiltinNameI16x8GeS] = builtinI16x8GeS
	builtinFunctions[common.BuiltinNameI16x8GeU] = builtinI16x8GeU
	builtinFunctions[common.BuiltinNameI16x8NarrowI32x4S] = builtinI16x8NarrowI32x4S
	builtinFunctions[common.BuiltinNameI16x8NarrowI32x4U] = builtinI16x8NarrowI32x4U
	builtinFunctions[common.BuiltinNameI16x8ExtendLowI8x16S] = builtinI16x8ExtendLowI8x16S
	builtinFunctions[common.BuiltinNameI16x8ExtendLowI8x16U] = builtinI16x8ExtendLowI8x16U
	builtinFunctions[common.BuiltinNameI16x8ExtendHighI8x16S] = builtinI16x8ExtendHighI8x16S
	builtinFunctions[common.BuiltinNameI16x8ExtendHighI8x16U] = builtinI16x8ExtendHighI8x16U
	builtinFunctions[common.BuiltinNameI16x8ExtaddPairwiseI8x16S] = builtinI16x8ExtaddPairwiseI8x16S
	builtinFunctions[common.BuiltinNameI16x8ExtaddPairwiseI8x16U] = builtinI16x8ExtaddPairwiseI8x16U
	builtinFunctions[common.BuiltinNameI16x8Q15mulrSatS] = builtinI16x8Q15mulrSatS
	builtinFunctions[common.BuiltinNameI16x8ExtmulLowI8x16S] = builtinI16x8ExtmulLowI8x16S
	builtinFunctions[common.BuiltinNameI16x8ExtmulLowI8x16U] = builtinI16x8ExtmulLowI8x16U
	builtinFunctions[common.BuiltinNameI16x8ExtmulHighI8x16S] = builtinI16x8ExtmulHighI8x16S
	builtinFunctions[common.BuiltinNameI16x8ExtmulHighI8x16U] = builtinI16x8ExtmulHighI8x16U
	builtinFunctions[common.BuiltinNameI16x8Shuffle] = builtinI16x8Shuffle

	// i32x4 aliases
	builtinFunctions[common.BuiltinNameI32x4Splat] = builtinI32x4Splat
	builtinFunctions[common.BuiltinNameI32x4ExtractLane] = builtinI32x4ExtractLane
	builtinFunctions[common.BuiltinNameI32x4ReplaceLane] = builtinI32x4ReplaceLane
	builtinFunctions[common.BuiltinNameI32x4Add] = builtinI32x4Add
	builtinFunctions[common.BuiltinNameI32x4Sub] = builtinI32x4Sub
	builtinFunctions[common.BuiltinNameI32x4Mul] = builtinI32x4Mul
	builtinFunctions[common.BuiltinNameI32x4MinS] = builtinI32x4MinS
	builtinFunctions[common.BuiltinNameI32x4MinU] = builtinI32x4MinU
	builtinFunctions[common.BuiltinNameI32x4MaxS] = builtinI32x4MaxS
	builtinFunctions[common.BuiltinNameI32x4MaxU] = builtinI32x4MaxU
	builtinFunctions[common.BuiltinNameI32x4DotI16x8S] = builtinI32x4DotI16x8S
	builtinFunctions[common.BuiltinNameI32x4Abs] = builtinI32x4Abs
	builtinFunctions[common.BuiltinNameI32x4Neg] = builtinI32x4Neg
	builtinFunctions[common.BuiltinNameI32x4Shl] = builtinI32x4Shl
	builtinFunctions[common.BuiltinNameI32x4ShrS] = builtinI32x4ShrS
	builtinFunctions[common.BuiltinNameI32x4ShrU] = builtinI32x4ShrU
	builtinFunctions[common.BuiltinNameI32x4AllTrue] = builtinI32x4AllTrue
	builtinFunctions[common.BuiltinNameI32x4Bitmask] = builtinI32x4Bitmask
	builtinFunctions[common.BuiltinNameI32x4Eq] = builtinI32x4Eq
	builtinFunctions[common.BuiltinNameI32x4Ne] = builtinI32x4Ne
	builtinFunctions[common.BuiltinNameI32x4LtS] = builtinI32x4LtS
	builtinFunctions[common.BuiltinNameI32x4LtU] = builtinI32x4LtU
	builtinFunctions[common.BuiltinNameI32x4LeS] = builtinI32x4LeS
	builtinFunctions[common.BuiltinNameI32x4LeU] = builtinI32x4LeU
	builtinFunctions[common.BuiltinNameI32x4GtS] = builtinI32x4GtS
	builtinFunctions[common.BuiltinNameI32x4GtU] = builtinI32x4GtU
	builtinFunctions[common.BuiltinNameI32x4GeS] = builtinI32x4GeS
	builtinFunctions[common.BuiltinNameI32x4GeU] = builtinI32x4GeU
	builtinFunctions[common.BuiltinNameI32x4TruncSatF32x4S] = builtinI32x4TruncSatF32x4S
	builtinFunctions[common.BuiltinNameI32x4TruncSatF32x4U] = builtinI32x4TruncSatF32x4U
	builtinFunctions[common.BuiltinNameI32x4TruncSatF64x2SZero] = builtinI32x4TruncSatF64x2SZero
	builtinFunctions[common.BuiltinNameI32x4TruncSatF64x2UZero] = builtinI32x4TruncSatF64x2UZero
	builtinFunctions[common.BuiltinNameI32x4ExtendLowI16x8S] = builtinI32x4ExtendLowI16x8S
	builtinFunctions[common.BuiltinNameI32x4ExtendLowI16x8U] = builtinI32x4ExtendLowI16x8U
	builtinFunctions[common.BuiltinNameI32x4ExtendHighI16x8S] = builtinI32x4ExtendHighI16x8S
	builtinFunctions[common.BuiltinNameI32x4ExtendHighI16x8U] = builtinI32x4ExtendHighI16x8U
	builtinFunctions[common.BuiltinNameI32x4ExtaddPairwiseI16x8S] = builtinI32x4ExtaddPairwiseI16x8S
	builtinFunctions[common.BuiltinNameI32x4ExtaddPairwiseI16x8U] = builtinI32x4ExtaddPairwiseI16x8U
	builtinFunctions[common.BuiltinNameI32x4ExtmulLowI16x8S] = builtinI32x4ExtmulLowI16x8S
	builtinFunctions[common.BuiltinNameI32x4ExtmulLowI16x8U] = builtinI32x4ExtmulLowI16x8U
	builtinFunctions[common.BuiltinNameI32x4ExtmulHighI16x8S] = builtinI32x4ExtmulHighI16x8S
	builtinFunctions[common.BuiltinNameI32x4ExtmulHighI16x8U] = builtinI32x4ExtmulHighI16x8U
	builtinFunctions[common.BuiltinNameI32x4Shuffle] = builtinI32x4Shuffle

	// i64x2 aliases
	builtinFunctions[common.BuiltinNameI64x2Splat] = builtinI64x2Splat
	builtinFunctions[common.BuiltinNameI64x2ExtractLane] = builtinI64x2ExtractLane
	builtinFunctions[common.BuiltinNameI64x2ReplaceLane] = builtinI64x2ReplaceLane
	builtinFunctions[common.BuiltinNameI64x2Add] = builtinI64x2Add
	builtinFunctions[common.BuiltinNameI64x2Sub] = builtinI64x2Sub
	builtinFunctions[common.BuiltinNameI64x2Mul] = builtinI64x2Mul
	builtinFunctions[common.BuiltinNameI64x2Abs] = builtinI64x2Abs
	builtinFunctions[common.BuiltinNameI64x2Neg] = builtinI64x2Neg
	builtinFunctions[common.BuiltinNameI64x2Shl] = builtinI64x2Shl
	builtinFunctions[common.BuiltinNameI64x2ShrS] = builtinI64x2ShrS
	builtinFunctions[common.BuiltinNameI64x2ShrU] = builtinI64x2ShrU
	builtinFunctions[common.BuiltinNameI64x2AllTrue] = builtinI64x2AllTrue
	builtinFunctions[common.BuiltinNameI64x2Bitmask] = builtinI64x2Bitmask
	builtinFunctions[common.BuiltinNameI64x2Eq] = builtinI64x2Eq
	builtinFunctions[common.BuiltinNameI64x2Ne] = builtinI64x2Ne
	builtinFunctions[common.BuiltinNameI64x2LtS] = builtinI64x2LtS
	builtinFunctions[common.BuiltinNameI64x2LeS] = builtinI64x2LeS
	builtinFunctions[common.BuiltinNameI64x2GtS] = builtinI64x2GtS
	builtinFunctions[common.BuiltinNameI64x2GeS] = builtinI64x2GeS
	builtinFunctions[common.BuiltinNameI64x2ExtendLowI32x4S] = builtinI64x2ExtendLowI32x4S
	builtinFunctions[common.BuiltinNameI64x2ExtendLowI32x4U] = builtinI64x2ExtendLowI32x4U
	builtinFunctions[common.BuiltinNameI64x2ExtendHighI32x4S] = builtinI64x2ExtendHighI32x4S
	builtinFunctions[common.BuiltinNameI64x2ExtendHighI32x4U] = builtinI64x2ExtendHighI32x4U
	builtinFunctions[common.BuiltinNameI64x2ExtmulLowI32x4S] = builtinI64x2ExtmulLowI32x4S
	builtinFunctions[common.BuiltinNameI64x2ExtmulLowI32x4U] = builtinI64x2ExtmulLowI32x4U
	builtinFunctions[common.BuiltinNameI64x2ExtmulHighI32x4S] = builtinI64x2ExtmulHighI32x4S
	builtinFunctions[common.BuiltinNameI64x2ExtmulHighI32x4U] = builtinI64x2ExtmulHighI32x4U
	builtinFunctions[common.BuiltinNameI64x2Shuffle] = builtinI64x2Shuffle

	// f32x4 aliases
	builtinFunctions[common.BuiltinNameF32x4Splat] = builtinF32x4Splat
	builtinFunctions[common.BuiltinNameF32x4ExtractLane] = builtinF32x4ExtractLane
	builtinFunctions[common.BuiltinNameF32x4ReplaceLane] = builtinF32x4ReplaceLane
	builtinFunctions[common.BuiltinNameF32x4Add] = builtinF32x4Add
	builtinFunctions[common.BuiltinNameF32x4Sub] = builtinF32x4Sub
	builtinFunctions[common.BuiltinNameF32x4Mul] = builtinF32x4Mul
	builtinFunctions[common.BuiltinNameF32x4Div] = builtinF32x4Div
	builtinFunctions[common.BuiltinNameF32x4Neg] = builtinF32x4Neg
	builtinFunctions[common.BuiltinNameF32x4Min] = builtinF32x4Min
	builtinFunctions[common.BuiltinNameF32x4Max] = builtinF32x4Max
	builtinFunctions[common.BuiltinNameF32x4Pmin] = builtinF32x4Pmin
	builtinFunctions[common.BuiltinNameF32x4Pmax] = builtinF32x4Pmax
	builtinFunctions[common.BuiltinNameF32x4Abs] = builtinF32x4Abs
	builtinFunctions[common.BuiltinNameF32x4Sqrt] = builtinF32x4Sqrt
	builtinFunctions[common.BuiltinNameF32x4Ceil] = builtinF32x4Ceil
	builtinFunctions[common.BuiltinNameF32x4Floor] = builtinF32x4Floor
	builtinFunctions[common.BuiltinNameF32x4Trunc] = builtinF32x4Trunc
	builtinFunctions[common.BuiltinNameF32x4Nearest] = builtinF32x4Nearest
	builtinFunctions[common.BuiltinNameF32x4Eq] = builtinF32x4Eq
	builtinFunctions[common.BuiltinNameF32x4Ne] = builtinF32x4Ne
	builtinFunctions[common.BuiltinNameF32x4Lt] = builtinF32x4Lt
	builtinFunctions[common.BuiltinNameF32x4Le] = builtinF32x4Le
	builtinFunctions[common.BuiltinNameF32x4Gt] = builtinF32x4Gt
	builtinFunctions[common.BuiltinNameF32x4Ge] = builtinF32x4Ge
	builtinFunctions[common.BuiltinNameF32x4ConvertI32x4S] = builtinF32x4ConvertI32x4S
	builtinFunctions[common.BuiltinNameF32x4ConvertI32x4U] = builtinF32x4ConvertI32x4U
	builtinFunctions[common.BuiltinNameF32x4DemoteF64x2Zero] = builtinF32x4DemoteF64x2Zero
	builtinFunctions[common.BuiltinNameF32x4Shuffle] = builtinF32x4Shuffle

	// f64x2 aliases
	builtinFunctions[common.BuiltinNameF64x2Splat] = builtinF64x2Splat
	builtinFunctions[common.BuiltinNameF64x2ExtractLane] = builtinF64x2ExtractLane
	builtinFunctions[common.BuiltinNameF64x2ReplaceLane] = builtinF64x2ReplaceLane
	builtinFunctions[common.BuiltinNameF64x2Add] = builtinF64x2Add
	builtinFunctions[common.BuiltinNameF64x2Sub] = builtinF64x2Sub
	builtinFunctions[common.BuiltinNameF64x2Mul] = builtinF64x2Mul
	builtinFunctions[common.BuiltinNameF64x2Div] = builtinF64x2Div
	builtinFunctions[common.BuiltinNameF64x2Neg] = builtinF64x2Neg
	builtinFunctions[common.BuiltinNameF64x2Min] = builtinF64x2Min
	builtinFunctions[common.BuiltinNameF64x2Max] = builtinF64x2Max
	builtinFunctions[common.BuiltinNameF64x2Pmin] = builtinF64x2Pmin
	builtinFunctions[common.BuiltinNameF64x2Pmax] = builtinF64x2Pmax
	builtinFunctions[common.BuiltinNameF64x2Abs] = builtinF64x2Abs
	builtinFunctions[common.BuiltinNameF64x2Sqrt] = builtinF64x2Sqrt
	builtinFunctions[common.BuiltinNameF64x2Ceil] = builtinF64x2Ceil
	builtinFunctions[common.BuiltinNameF64x2Floor] = builtinF64x2Floor
	builtinFunctions[common.BuiltinNameF64x2Trunc] = builtinF64x2Trunc
	builtinFunctions[common.BuiltinNameF64x2Nearest] = builtinF64x2Nearest
	builtinFunctions[common.BuiltinNameF64x2Eq] = builtinF64x2Eq
	builtinFunctions[common.BuiltinNameF64x2Ne] = builtinF64x2Ne
	builtinFunctions[common.BuiltinNameF64x2Lt] = builtinF64x2Lt
	builtinFunctions[common.BuiltinNameF64x2Le] = builtinF64x2Le
	builtinFunctions[common.BuiltinNameF64x2Gt] = builtinF64x2Gt
	builtinFunctions[common.BuiltinNameF64x2Ge] = builtinF64x2Ge
	builtinFunctions[common.BuiltinNameF64x2ConvertLowI32x4S] = builtinF64x2ConvertLowI32x4S
	builtinFunctions[common.BuiltinNameF64x2ConvertLowI32x4U] = builtinF64x2ConvertLowI32x4U
	builtinFunctions[common.BuiltinNameF64x2PromoteLowF32x4] = builtinF64x2PromoteLowF32x4
	builtinFunctions[common.BuiltinNameF64x2Shuffle] = builtinF64x2Shuffle

	// Relaxed SIMD aliases
	builtinFunctions[common.BuiltinNameI8x16RelaxedSwizzle] = builtinI8x16RelaxedSwizzle
	builtinFunctions[common.BuiltinNameI32x4RelaxedTruncF32x4S] = builtinI32x4RelaxedTruncF32x4S
	builtinFunctions[common.BuiltinNameI32x4RelaxedTruncF32x4U] = builtinI32x4RelaxedTruncF32x4U
	builtinFunctions[common.BuiltinNameI32x4RelaxedTruncF64x2SZero] = builtinI32x4RelaxedTruncF64x2SZero
	builtinFunctions[common.BuiltinNameI32x4RelaxedTruncF64x2UZero] = builtinI32x4RelaxedTruncF64x2UZero
	builtinFunctions[common.BuiltinNameF32x4RelaxedMadd] = builtinF32x4RelaxedMadd
	builtinFunctions[common.BuiltinNameF32x4RelaxedNmadd] = builtinF32x4RelaxedNmadd
	builtinFunctions[common.BuiltinNameF64x2RelaxedMadd] = builtinF64x2RelaxedMadd
	builtinFunctions[common.BuiltinNameF64x2RelaxedNmadd] = builtinF64x2RelaxedNmadd
	builtinFunctions[common.BuiltinNameI8x16RelaxedLaneselect] = builtinI8x16RelaxedLaneselect
	builtinFunctions[common.BuiltinNameI16x8RelaxedLaneselect] = builtinI16x8RelaxedLaneselect
	builtinFunctions[common.BuiltinNameI32x4RelaxedLaneselect] = builtinI32x4RelaxedLaneselect
	builtinFunctions[common.BuiltinNameI64x2RelaxedLaneselect] = builtinI64x2RelaxedLaneselect
	builtinFunctions[common.BuiltinNameF32x4RelaxedMin] = builtinF32x4RelaxedMin
	builtinFunctions[common.BuiltinNameF32x4RelaxedMax] = builtinF32x4RelaxedMax
	builtinFunctions[common.BuiltinNameF64x2RelaxedMin] = builtinF64x2RelaxedMin
	builtinFunctions[common.BuiltinNameF64x2RelaxedMax] = builtinF64x2RelaxedMax
	builtinFunctions[common.BuiltinNameI16x8RelaxedQ15mulrS] = builtinI16x8RelaxedQ15mulrS
	builtinFunctions[common.BuiltinNameI16x8RelaxedDotI8x16I7x16S] = builtinI16x8RelaxedDotI8x16I7x16S
	builtinFunctions[common.BuiltinNameI32x4RelaxedDotI8x16I7x16AddS] = builtinI32x4RelaxedDotI8x16I7x16AddS
}

// ========================================================================================
// Stub forward declarations for operator builtins not yet ported.
// TODO: Port these from assemblyscript/src/builtins.ts when the operator builtins section is done.
// ========================================================================================

func builtinRem(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinRem not yet ported")
}

func builtinAdd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinAdd not yet ported")
}

func builtinSub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinSub not yet ported")
}

func builtinMul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinMul not yet ported")
}

func builtinDiv(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinDiv not yet ported")
}

func builtinEq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinEq not yet ported")
}

func builtinNe(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinNe not yet ported")
}

// ========================================================================================
// Stub forward declarations for v128/SIMD core builtins not yet ported.
// TODO: Port these from assemblyscript/src/builtins.ts when the SIMD builtins section is done.
// ========================================================================================

func builtinV128Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Splat not yet ported")
}

func builtinV128ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ExtractLane not yet ported")
}

func builtinV128ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ReplaceLane not yet ported")
}

func builtinV128Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Add not yet ported")
}

func builtinV128Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Sub not yet ported")
}

func builtinV128Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Mul not yet ported")
}

func builtinV128Div(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Div not yet ported")
}

func builtinV128Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Neg not yet ported")
}

func builtinV128Min(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Min not yet ported")
}

func builtinV128Max(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Max not yet ported")
}

func builtinV128Pmin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Pmin not yet ported")
}

func builtinV128Pmax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Pmax not yet ported")
}

func builtinV128Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Abs not yet ported")
}

func builtinV128Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Sqrt not yet ported")
}

func builtinV128Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Ceil not yet ported")
}

func builtinV128Floor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Floor not yet ported")
}

func builtinV128Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Trunc not yet ported")
}

func builtinV128Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Nearest not yet ported")
}

func builtinV128Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Eq not yet ported")
}

func builtinV128Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Ne not yet ported")
}

func builtinV128Lt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Lt not yet ported")
}

func builtinV128Le(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Le not yet ported")
}

func builtinV128Gt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Gt not yet ported")
}

func builtinV128Ge(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Ge not yet ported")
}

func builtinV128Shl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Shl not yet ported")
}

func builtinV128Shr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Shr not yet ported")
}

func builtinV128AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128AllTrue not yet ported")
}

func builtinV128Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Bitmask not yet ported")
}

func builtinV128Popcnt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Popcnt not yet ported")
}

func builtinV128AddSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128AddSat not yet ported")
}

func builtinV128SubSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128SubSat not yet ported")
}

func builtinV128Avgr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Avgr not yet ported")
}

func builtinV128Narrow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Narrow not yet ported")
}

func builtinV128Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Shuffle not yet ported")
}

func builtinV128Swizzle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Swizzle not yet ported")
}

func builtinV128LoadExt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128LoadExt not yet ported")
}

func builtinV128LoadSplat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128LoadSplat not yet ported")
}

func builtinV128LoadZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128LoadZero not yet ported")
}

func builtinV128LoadLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128LoadLane not yet ported")
}

func builtinV128StoreLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128StoreLane not yet ported")
}

func builtinV128Convert(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Convert not yet ported")
}

func builtinV128ConvertLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ConvertLow not yet ported")
}

func builtinV128TruncSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128TruncSat not yet ported")
}

func builtinV128TruncSatZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128TruncSatZero not yet ported")
}

func builtinV128ExtendLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ExtendLow not yet ported")
}

func builtinV128ExtendHigh(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ExtendHigh not yet ported")
}

func builtinV128ExtaddPairwise(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ExtaddPairwise not yet ported")
}

func builtinV128DemoteZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128DemoteZero not yet ported")
}

func builtinV128PromoteLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128PromoteLow not yet ported")
}

func builtinV128Q15mulrSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Q15mulrSat not yet ported")
}

func builtinV128ExtmulLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ExtmulLow not yet ported")
}

func builtinV128ExtmulHigh(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128ExtmulHigh not yet ported")
}

func builtinV128Dot(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Dot not yet ported")
}

func builtinV128RelaxedSwizzle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedSwizzle not yet ported")
}

func builtinV128RelaxedTrunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedTrunc not yet ported")
}

func builtinV128RelaxedTruncZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedTruncZero not yet ported")
}

func builtinV128RelaxedMadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedMadd not yet ported")
}

func builtinV128RelaxedNmadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedNmadd not yet ported")
}

func builtinV128RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedLaneselect not yet ported")
}

func builtinV128RelaxedMin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedMin not yet ported")
}

func builtinV128RelaxedMax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedMax not yet ported")
}

func builtinV128RelaxedQ15mulr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedQ15mulr not yet ported")
}

func builtinV128RelaxedDot(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedDot not yet ported")
}

func builtinV128RelaxedDotAdd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128RelaxedDotAdd not yet ported")
}

func builtinV128And(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128And not yet ported")
}

func builtinV128Or(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Or not yet ported")
}

func builtinV128Xor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Xor not yet ported")
}

func builtinV128Andnot(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Andnot not yet ported")
}

func builtinV128Not(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Not not yet ported")
}

func builtinV128Bitselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128Bitselect not yet ported")
}

func builtinV128AnyTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	panic("builtinV128AnyTrue not yet ported")
}

// ========================================================================================
// v128 load/store alias functions
// ========================================================================================

// v128.load -> load<v128>
func builtinV128Load(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeV128}
	ctx.ContextualType = types.TypeV128
	return builtinLoad(ctx)
}

// v128.load8x8_s -> v128.load_ext<i8>
func builtinV128Load8x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadExt(ctx)
}

// v128.load8x8_u -> v128.load_ext<u8>
func builtinV128Load8x8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadExt(ctx)
}

// v128.load16x4_s -> v128.load_ext<i16>
func builtinV128Load16x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadExt(ctx)
}

// v128.load16x4_u -> v128.load_ext<u16>
func builtinV128Load16x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadExt(ctx)
}

// v128.load32x2_s -> v128.load_ext<i32>
func builtinV128Load32x2S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadExt(ctx)
}

// v128.load32x2_u -> v128.load_ext<u32>
func builtinV128Load32x2U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadExt(ctx)
}

// v128.load8_splat -> v128.load_splat<u8>
func builtinV128Load8Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadSplat(ctx)
}

// v128.load16_splat -> v128.load_splat<u16>
func builtinV128Load16Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadSplat(ctx)
}

// v128.load32_splat -> v128.load_splat<u32>
func builtinV128Load32Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadSplat(ctx)
}

// v128.load64_splat -> v128.load_splat<u64>
func builtinV128Load64Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU64}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadSplat(ctx)
}

// v128.load32_zero -> v128.load_zero<u32>
func builtinV128Load32Zero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadZero(ctx)
}

// v128.load64_zero -> v128.load_zero<u64>
func builtinV128Load64Zero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU64}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadZero(ctx)
}

// v128.load8_lane -> v128.load_lane<u8>
func builtinV128Load8Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadLane(ctx)
}

// v128.load16_lane -> v128.load_lane<u16>
func builtinV128Load16Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadLane(ctx)
}

// v128.load32_lane -> v128.load_lane<u32>
func builtinV128Load32Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadLane(ctx)
}

// v128.load64_lane -> v128.load_lane<u64>
func builtinV128Load64Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU64}
	ctx.ContextualType = types.TypeV128
	return builtinV128LoadLane(ctx)
}

// v128.store8_lane -> v128.store_lane<u8>
func builtinV128Store8Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128StoreLane(ctx)
}

// v128.store16_lane -> v128.store_lane<u16>
func builtinV128Store16Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128StoreLane(ctx)
}

// v128.store32_lane -> v128.store_lane<u32>
func builtinV128Store32Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128StoreLane(ctx)
}

// v128.store64_lane -> v128.store_lane<u64>
func builtinV128Store64Lane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU64}
	ctx.ContextualType = types.TypeV128
	return builtinV128StoreLane(ctx)
}

// v128.store -> store<v128> (contextIsExact = true)
func builtinV128Store(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeV128}
	ctx.ContextualType = types.TypeV128
	ctx.ContextIsExact = true
	return builtinStore(ctx)
}

// ========================================================================================
// i8x16 SIMD alias functions
// ========================================================================================

func builtinI8x16Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Splat(ctx)
}

func builtinI8x16ExtractLaneS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI32
	return builtinV128ExtractLane(ctx)
}

func builtinI8x16ExtractLaneU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeI32
	return builtinV128ExtractLane(ctx)
}

func builtinI8x16ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ReplaceLane(ctx)
}

func builtinI8x16Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Add(ctx)
}

func builtinI8x16Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sub(ctx)
}

func builtinI8x16MinS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}

func builtinI8x16MinU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}

func builtinI8x16MaxS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}

func builtinI8x16MaxU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}

func builtinI8x16AvgrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Avgr(ctx)
}

func builtinI8x16Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Abs(ctx)
}

func builtinI8x16Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Neg(ctx)
}

func builtinI8x16AddSatS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128AddSat(ctx)
}

func builtinI8x16AddSatU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128AddSat(ctx)
}

func builtinI8x16SubSatS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128SubSat(ctx)
}

func builtinI8x16SubSatU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128SubSat(ctx)
}

func builtinI8x16Shl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shl(ctx)
}

func builtinI8x16ShrS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}

func builtinI8x16ShrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}

func builtinI8x16AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI32
	return builtinV128AllTrue(ctx)
}

func builtinI8x16Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeI32
	return builtinV128Bitmask(ctx)
}

func builtinI8x16Popcnt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Popcnt(ctx)
}

func builtinI8x16Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Eq(ctx)
}

func builtinI8x16Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ne(ctx)
}

func builtinI8x16LtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}

func builtinI8x16LtU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}

func builtinI8x16LeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}

func builtinI8x16LeU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}

func builtinI8x16GtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}

func builtinI8x16GtU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}

func builtinI8x16GeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}

func builtinI8x16GeU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}

func builtinI8x16NarrowI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Narrow(ctx)
}

func builtinI8x16NarrowI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Narrow(ctx)
}

func builtinI8x16Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shuffle(ctx)
}

// i8x16.swizzle -> v128.swizzle (typeArguments = nil)
func builtinI8x16Swizzle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = nil
	ctx.ContextualType = types.TypeV128
	return builtinV128Swizzle(ctx)
}

// ========================================================================================
// i16x8 SIMD alias functions
// ========================================================================================

func builtinI16x8Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Splat(ctx)
}

func builtinI16x8ExtractLaneS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI32
	return builtinV128ExtractLane(ctx)
}

func builtinI16x8ExtractLaneU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeI32
	return builtinV128ExtractLane(ctx)
}

func builtinI16x8ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ReplaceLane(ctx)
}

func builtinI16x8Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Add(ctx)
}

func builtinI16x8Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sub(ctx)
}

func builtinI16x8Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Mul(ctx)
}

func builtinI16x8MinS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}

func builtinI16x8MinU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}

func builtinI16x8MaxS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}

func builtinI16x8MaxU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}

func builtinI16x8AvgrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Avgr(ctx)
}

func builtinI16x8Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Abs(ctx)
}

func builtinI16x8Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Neg(ctx)
}

func builtinI16x8AddSatS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128AddSat(ctx)
}

func builtinI16x8AddSatU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128AddSat(ctx)
}

func builtinI16x8SubSatS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128SubSat(ctx)
}

func builtinI16x8SubSatU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128SubSat(ctx)
}

func builtinI16x8Shl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shl(ctx)
}

func builtinI16x8ShrS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}

func builtinI16x8ShrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}

func builtinI16x8AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI32
	return builtinV128AllTrue(ctx)
}

func builtinI16x8Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeI32
	return builtinV128Bitmask(ctx)
}

func builtinI16x8Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Eq(ctx)
}

func builtinI16x8Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ne(ctx)
}

func builtinI16x8LtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}

func builtinI16x8LtU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}

func builtinI16x8LeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}

func builtinI16x8LeU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}

func builtinI16x8GtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}

func builtinI16x8GtU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}

func builtinI16x8GeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}

func builtinI16x8GeU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}

func builtinI16x8NarrowI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Narrow(ctx)
}

func builtinI16x8NarrowI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Narrow(ctx)
}

func builtinI16x8ExtendLowI8x16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendLow(ctx)
}

func builtinI16x8ExtendLowI8x16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendLow(ctx)
}

func builtinI16x8ExtendHighI8x16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendHigh(ctx)
}

func builtinI16x8ExtendHighI8x16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendHigh(ctx)
}

func builtinI16x8ExtaddPairwiseI8x16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtaddPairwise(ctx)
}

func builtinI16x8ExtaddPairwiseI8x16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtaddPairwise(ctx)
}

func builtinI16x8Q15mulrSatS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Q15mulrSat(ctx)
}

func builtinI16x8ExtmulLowI8x16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulLow(ctx)
}

func builtinI16x8ExtmulLowI8x16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulLow(ctx)
}

func builtinI16x8ExtmulHighI8x16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulHigh(ctx)
}

func builtinI16x8ExtmulHighI8x16U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU8}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulHigh(ctx)
}

func builtinI16x8Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shuffle(ctx)
}

// ========================================================================================
// i32x4 SIMD alias functions
// ========================================================================================

func builtinI32x4Splat(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Splat(ctx) }
func builtinI32x4ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeI32; return builtinV128ExtractLane(ctx) }
func builtinI32x4ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128ReplaceLane(ctx) }
func builtinI32x4Add(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Add(ctx) }
func builtinI32x4Sub(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Sub(ctx) }
func builtinI32x4Mul(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Mul(ctx) }
func builtinI32x4MinS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Min(ctx) }
func builtinI32x4MinU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Min(ctx) }
func builtinI32x4MaxS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Max(ctx) }
func builtinI32x4MaxU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Max(ctx) }
func builtinI32x4DotI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128Dot(ctx) }
func builtinI32x4Abs(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Abs(ctx) }
func builtinI32x4Neg(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Neg(ctx) }
func builtinI32x4Shl(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Shl(ctx) }
func builtinI32x4ShrS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Shr(ctx) }
func builtinI32x4ShrU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Shr(ctx) }
func builtinI32x4AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeI32; return builtinV128AllTrue(ctx) }
func builtinI32x4Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeI32; return builtinV128Bitmask(ctx) }
func builtinI32x4Eq(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Eq(ctx) }
func builtinI32x4Ne(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Ne(ctx) }
func builtinI32x4LtS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Lt(ctx) }
func builtinI32x4LtU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Lt(ctx) }
func builtinI32x4LeS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Le(ctx) }
func builtinI32x4LeU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Le(ctx) }
func builtinI32x4GtS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Gt(ctx) }
func builtinI32x4GtU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Gt(ctx) }
func builtinI32x4GeS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Ge(ctx) }
func builtinI32x4GeU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Ge(ctx) }
func builtinI32x4TruncSatF32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128TruncSat(ctx) }
func builtinI32x4TruncSatF32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128TruncSat(ctx) }
func builtinI32x4TruncSatF64x2SZero(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128TruncSatZero(ctx) }
func builtinI32x4TruncSatF64x2UZero(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128TruncSatZero(ctx) }
func builtinI32x4ExtendLowI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendLow(ctx) }
func builtinI32x4ExtendLowI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendLow(ctx) }
func builtinI32x4ExtendHighI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendHigh(ctx) }
func builtinI32x4ExtendHighI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendHigh(ctx) }
func builtinI32x4ExtaddPairwiseI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtaddPairwise(ctx) }
func builtinI32x4ExtaddPairwiseI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtaddPairwise(ctx) }
func builtinI32x4ExtmulLowI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulLow(ctx) }
func builtinI32x4ExtmulLowI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulLow(ctx) }
func builtinI32x4ExtmulHighI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulHigh(ctx) }
func builtinI32x4ExtmulHighI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU16}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulHigh(ctx) }
func builtinI32x4Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Shuffle(ctx) }

// ========================================================================================
// i64x2 SIMD alias functions
// ========================================================================================

func builtinI64x2Splat(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Splat(ctx) }
func builtinI64x2ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeI64; return builtinV128ExtractLane(ctx) }
func builtinI64x2ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128ReplaceLane(ctx) }
func builtinI64x2Add(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Add(ctx) }
func builtinI64x2Sub(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Sub(ctx) }
func builtinI64x2Mul(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Mul(ctx) }
func builtinI64x2Abs(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Abs(ctx) }
func builtinI64x2Neg(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Neg(ctx) }
func builtinI64x2Shl(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Shl(ctx) }
func builtinI64x2ShrS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Shr(ctx) }
func builtinI64x2ShrU(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU64}; ctx.ContextualType = types.TypeV128; return builtinV128Shr(ctx) }
func builtinI64x2AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeI32; return builtinV128AllTrue(ctx) }
func builtinI64x2Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeI32; return builtinV128Bitmask(ctx) }
func builtinI64x2Eq(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Eq(ctx) }
func builtinI64x2Ne(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Ne(ctx) }
func builtinI64x2LtS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Lt(ctx) }
func builtinI64x2LeS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Le(ctx) }
func builtinI64x2GtS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Gt(ctx) }
func builtinI64x2GeS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Ge(ctx) }
func builtinI64x2ExtendLowI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendLow(ctx) }
func builtinI64x2ExtendLowI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendLow(ctx) }
func builtinI64x2ExtendHighI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendHigh(ctx) }
func builtinI64x2ExtendHighI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtendHigh(ctx) }
func builtinI64x2ExtmulLowI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulLow(ctx) }
func builtinI64x2ExtmulLowI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulLow(ctx) }
func builtinI64x2ExtmulHighI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulHigh(ctx) }
func builtinI64x2ExtmulHighI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128ExtmulHigh(ctx) }
func builtinI64x2Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128Shuffle(ctx) }

// ========================================================================================
// f32x4 SIMD alias functions
// ========================================================================================

func builtinF32x4Splat(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Splat(ctx) }
func builtinF32x4ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeF32; return builtinV128ExtractLane(ctx) }
func builtinF32x4ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128ReplaceLane(ctx) }
func builtinF32x4Add(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Add(ctx) }
func builtinF32x4Sub(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Sub(ctx) }
func builtinF32x4Mul(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Mul(ctx) }
func builtinF32x4Div(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Div(ctx) }
func builtinF32x4Neg(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Neg(ctx) }
func builtinF32x4Min(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Min(ctx) }
func builtinF32x4Max(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Max(ctx) }
func builtinF32x4Pmin(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Pmin(ctx) }
func builtinF32x4Pmax(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Pmax(ctx) }
func builtinF32x4Abs(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Abs(ctx) }
func builtinF32x4Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Sqrt(ctx) }
func builtinF32x4Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Ceil(ctx) }
func builtinF32x4Floor(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Floor(ctx) }
func builtinF32x4Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Trunc(ctx) }
func builtinF32x4Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Nearest(ctx) }
func builtinF32x4Eq(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Eq(ctx) }
func builtinF32x4Ne(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Ne(ctx) }
func builtinF32x4Lt(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Lt(ctx) }
func builtinF32x4Le(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Le(ctx) }
func builtinF32x4Gt(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Gt(ctx) }
func builtinF32x4Ge(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Ge(ctx) }
func builtinF32x4ConvertI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128Convert(ctx) }
func builtinF32x4ConvertI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128Convert(ctx) }
func builtinF32x4DemoteF64x2Zero(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128DemoteZero(ctx) }
func builtinF32x4Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128Shuffle(ctx) }

// ========================================================================================
// f64x2 SIMD alias functions
// ========================================================================================

func builtinF64x2Splat(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Splat(ctx) }
func builtinF64x2ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeF64; return builtinV128ExtractLane(ctx) }
func builtinF64x2ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128ReplaceLane(ctx) }
func builtinF64x2Add(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Add(ctx) }
func builtinF64x2Sub(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Sub(ctx) }
func builtinF64x2Mul(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Mul(ctx) }
func builtinF64x2Div(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Div(ctx) }
func builtinF64x2Neg(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Neg(ctx) }
func builtinF64x2Min(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Min(ctx) }
func builtinF64x2Max(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Max(ctx) }
func builtinF64x2Pmin(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Pmin(ctx) }
func builtinF64x2Pmax(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Pmax(ctx) }
func builtinF64x2Abs(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Abs(ctx) }
func builtinF64x2Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Sqrt(ctx) }
func builtinF64x2Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Ceil(ctx) }
func builtinF64x2Floor(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Floor(ctx) }
func builtinF64x2Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Trunc(ctx) }
func builtinF64x2Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Nearest(ctx) }
func builtinF64x2Eq(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Eq(ctx) }
func builtinF64x2Ne(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Ne(ctx) }
func builtinF64x2Lt(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Lt(ctx) }
func builtinF64x2Le(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Le(ctx) }
func builtinF64x2Gt(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Gt(ctx) }
func builtinF64x2Ge(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Ge(ctx) }
func builtinF64x2ConvertLowI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128ConvertLow(ctx) }
func builtinF64x2ConvertLowI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128ConvertLow(ctx) }
// Note: TS source names this function builtin_f64x4_promote_low_f32x4 (typo in TS, registered as f64x2_promote_low_f32x4)
func builtinF64x2PromoteLowF32x4(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128PromoteLow(ctx) }
func builtinF64x2Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128Shuffle(ctx) }

// ========================================================================================
// Relaxed SIMD alias functions
// ========================================================================================

// i8x16.relaxed_swizzle -> v128.relaxed_swizzle (typeArguments = nil)
func builtinI8x16RelaxedSwizzle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = nil
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedSwizzle(ctx)
}

func builtinI32x4RelaxedTruncF32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedTrunc(ctx) }
func builtinI32x4RelaxedTruncF32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedTrunc(ctx) }
func builtinI32x4RelaxedTruncF64x2SZero(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedTruncZero(ctx) }
func builtinI32x4RelaxedTruncF64x2UZero(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeU32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedTruncZero(ctx) }
func builtinF32x4RelaxedMadd(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedMadd(ctx) }
func builtinF32x4RelaxedNmadd(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedNmadd(ctx) }
func builtinF64x2RelaxedMadd(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedMadd(ctx) }
func builtinF64x2RelaxedNmadd(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedNmadd(ctx) }
func builtinI8x16RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI8}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedLaneselect(ctx) }
func builtinI16x8RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedLaneselect(ctx) }
func builtinI32x4RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedLaneselect(ctx) }
func builtinI64x2RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI64}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedLaneselect(ctx) }
func builtinF32x4RelaxedMin(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedMin(ctx) }
func builtinF32x4RelaxedMax(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedMax(ctx) }
func builtinF64x2RelaxedMin(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedMin(ctx) }
func builtinF64x2RelaxedMax(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeF64}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedMax(ctx) }
func builtinI16x8RelaxedQ15mulrS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedQ15mulr(ctx) }
func builtinI16x8RelaxedDotI8x16I7x16S(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI16}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedDot(ctx) }
func builtinI32x4RelaxedDotI8x16I7x16AddS(ctx *BuiltinFunctionContext) module.ExpressionRef { checkTypeAbsent(ctx); ctx.TypeArguments = []*types.Type{types.TypeI32}; ctx.ContextualType = types.TypeV128; return builtinV128RelaxedDotAdd(ctx) }
