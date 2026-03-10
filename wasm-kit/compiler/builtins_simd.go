// Ported from: assemblyscript/src/builtins.ts (lines 8551-10828)
// SIMD/v128 inline assembler alias builtins + stub forward declarations for unported core builtins.
package compiler

import (
	"encoding/binary"
	"math"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
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

	// Generic operator builtins (TS builtins.ts:2601-2927)
	builtinFunctions[common.BuiltinNameAdd] = builtinAdd
	builtinFunctions[common.BuiltinNameSub] = builtinSub
	builtinFunctions[common.BuiltinNameMul] = builtinMul
	builtinFunctions[common.BuiltinNameDiv] = builtinDiv
	builtinFunctions[common.BuiltinNameRem] = builtinRem
	builtinFunctions[common.BuiltinNameEq] = builtinEq
	builtinFunctions[common.BuiltinNameNe] = builtinNe

	// v128 constructor builtins (TS builtins.ts:4031-4335)
	builtinFunctions[common.BuiltinNameV128] = builtinV128Ctor
	builtinFunctions[common.BuiltinNameI8x16] = builtinI8x16Ctor
	builtinFunctions[common.BuiltinNameI16x8] = builtinI16x8Ctor
	builtinFunctions[common.BuiltinNameI32x4] = builtinI32x4Ctor
	builtinFunctions[common.BuiltinNameI64x2] = builtinI64x2Ctor
	builtinFunctions[common.BuiltinNameF32x4] = builtinF32x4Ctor
	builtinFunctions[common.BuiltinNameF64x2] = builtinF64x2Ctor
}

func reportBuiltinOperationTypeError(ctx *BuiltinFunctionContext, op string, typ *types.Type) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	typeName := "<nil>"
	if typ != nil {
		typeName = typ.String()
	}
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		op,
		typeName,
		"",
	)
	return mod.Unreachable()
}

func prepareBuiltinValueBinaryOperands(ctx *BuiltinFunctionContext) (module.ExpressionRef, module.ExpressionRef, *types.Type, bool) {
	compiler := ctx.Compiler
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 2) {
		return 0, 0, compiler.CurrentType, false
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	left := operands[0]

	var arg0 module.ExpressionRef
	if len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(left, typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(left, types.TypeAuto, 0)
	}
	typ := compiler.CurrentType
	if typ == nil || !typ.IsValue() {
		return arg0, 0, typ, false
	}

	var arg1 module.ExpressionRef
	if len(typeArguments) == 0 && ast.IsNumericLiteral(left) {
		arg1 = compiler.CompileExpression(operands[1], typ, 0)
		if compiler.CurrentType != typ {
			typ = compiler.CurrentType
			arg0 = compiler.CompileExpression(left, typ, ConstraintsConvImplicit)
		}
	} else {
		arg1 = compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit)
	}
	return arg0, arg1, typ, true
}

func prepareRequiredV128UnaryBuiltin(
	ctx *BuiltinFunctionContext,
	resultType *types.Type,
) (*Compiler, *module.Module, *types.Type, module.ExpressionRef, bool) {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		compiler.CurrentType = resultType
		return compiler, mod, nil, 0, false
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = resultType
	return compiler, mod, typ, arg0, true
}

func prepareOptionalV128UnaryBuiltin(
	ctx *BuiltinFunctionContext,
	defaultType *types.Type,
	resultType *types.Type,
) (*Compiler, *module.Module, *types.Type, module.ExpressionRef, bool) {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeOptional(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		compiler.CurrentType = resultType
		return compiler, mod, nil, 0, false
	}
	typ := defaultType
	if len(ctx.TypeArguments) > 0 {
		typ = ctx.TypeArguments[0]
	}
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = resultType
	return compiler, mod, typ, arg0, true
}

func prepareRequiredV128BinaryBuiltin(
	ctx *BuiltinFunctionContext,
	arg1Type *types.Type,
	resultType *types.Type,
) (*Compiler, *module.Module, *types.Type, module.ExpressionRef, module.ExpressionRef, bool) {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = resultType
		return compiler, mod, nil, 0, 0, false
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], arg1Type, ConstraintsConvImplicit)
	compiler.CurrentType = resultType
	return compiler, mod, typ, arg0, arg1, true
}

func prepareRequiredV128TernaryBuiltin(
	ctx *BuiltinFunctionContext,
	feature common.Feature,
	resultType *types.Type,
) (*Compiler, *module.Module, *types.Type, module.ExpressionRef, module.ExpressionRef, module.ExpressionRef, bool) {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, feature))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 3)) != 0 {
		compiler.CurrentType = resultType
		return compiler, mod, nil, 0, 0, 0, false
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	arg2 := compiler.CompileExpression(ctx.Operands[2], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = resultType
	return compiler, mod, typ, arg0, arg1, arg2, true
}

func evaluateSIMDConstantIndex(expr module.ExpressionRef, reportNode ast.Node, compiler *Compiler) int32 {
	mod := compiler.Module()
	precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
	if precomp != 0 {
		return module.GetConstValueI32(precomp)
	}
	compiler.Error(
		diagnostics.DiagnosticCodeExpressionMustBeACompileTimeConstant,
		reportNode.GetRange(),
		"", "", "",
	)
	return 0
}

func validateSIMDLaneIndex(idx int32, laneType *types.Type, reportNode ast.Node, compiler *Compiler) uint8 {
	if laneType == nil {
		return 0
	}
	laneWidth := laneType.ByteSize()
	if laneWidth <= 0 {
		return 0
	}
	maxIdx := (16 / laneWidth) - 1
	if idx < 0 || idx > maxIdx {
		compiler.Error(
			diagnostics.DiagnosticCode0MustBeAValueBetween1And2Inclusive,
			reportNode.GetRange(),
			"Lane index",
			"0",
			intToString(int(maxIdx)),
		)
		idx = 0
	}
	return uint8(idx)
}

func evaluateSIMDMemoryImmediateOperands(
	operands []ast.Node,
	immediateStart int,
	naturalAlign int32,
	compiler *Compiler,
) (uint32, uint32, bool) {
	immOffset := int32(0)
	immAlign := naturalAlign
	numOperands := len(operands)
	if numOperands >= immediateStart+1 {
		immOffset = evaluateImmediateOffset(operands[immediateStart], compiler)
		if immOffset < 0 {
			return 0, 0, false
		}
		if numOperands == immediateStart+2 {
			immAlign = evaluateImmediateAlign(operands[immediateStart+1], immAlign, compiler)
			if immAlign < 0 {
				return 0, 0, false
			}
		}
	}
	return uint32(immOffset), uint32(immAlign), true
}

func builtinV128BitwiseBinaryOp(ctx *BuiltinFunctionContext, op module.Op) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	return mod.Binary(op, arg0, arg1)
}

func builtinV128BitwiseUnaryOp(ctx *BuiltinFunctionContext, op module.Op) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	return mod.Unary(op, arg0)
}

func builtinRem(ctx *BuiltinFunctionContext) module.ExpressionRef {
	arg0, arg1, typ, ok := prepareBuiltinValueBinaryOperands(ctx)
	if ok && typ.IsIntegerValue() {
		return ctx.Compiler.makeBinaryRem(arg0, arg1, typ, ctx.ReportNode)
	}
	return reportBuiltinOperationTypeError(ctx, "rem", typ)
}

func builtinAdd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	arg0, arg1, typ, ok := prepareBuiltinValueBinaryOperands(ctx)
	if ok && typ.IsNumericValue() {
		return ctx.Compiler.makeBinaryAdd(arg0, arg1, typ)
	}
	return reportBuiltinOperationTypeError(ctx, "add", typ)
}

func builtinSub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	arg0, arg1, typ, ok := prepareBuiltinValueBinaryOperands(ctx)
	if ok && typ.IsNumericValue() {
		return ctx.Compiler.makeBinarySub(arg0, arg1, typ)
	}
	return reportBuiltinOperationTypeError(ctx, "sub", typ)
}

func builtinMul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	arg0, arg1, typ, ok := prepareBuiltinValueBinaryOperands(ctx)
	if ok && typ.IsNumericValue() {
		return ctx.Compiler.makeBinaryMul(arg0, arg1, typ)
	}
	return reportBuiltinOperationTypeError(ctx, "mul", typ)
}

func builtinDiv(ctx *BuiltinFunctionContext) module.ExpressionRef {
	arg0, arg1, typ, ok := prepareBuiltinValueBinaryOperands(ctx)
	if ok && typ.IsNumericValue() {
		return ctx.Compiler.makeBinaryDiv(arg0, arg1, typ)
	}
	return reportBuiltinOperationTypeError(ctx, "div", typ)
}

func builtinEq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	arg0, arg1, typ, ok := prepareBuiltinValueBinaryOperands(ctx)
	if ok && typ.IsNumericValue() {
		ctx.Compiler.CurrentType = types.TypeI32
		return ctx.Compiler.makeBinaryEq(arg0, arg1, typ, ctx.ReportNode)
	}
	return reportBuiltinOperationTypeError(ctx, "eq", typ)
}

func builtinNe(ctx *BuiltinFunctionContext) module.ExpressionRef {
	arg0, arg1, typ, ok := prepareBuiltinValueBinaryOperands(ctx)
	if ok && typ.IsNumericValue() {
		ctx.Compiler.CurrentType = types.TypeI32
		return ctx.Compiler.makeBinaryNe(arg0, arg1, typ, ctx.ReportNode)
	}
	return reportBuiltinOperationTypeError(ctx, "ne", typ)
}

func builtinV128Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], typ, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Unary(module.UnaryOpSplatI8x16, arg0)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Unary(module.UnaryOpSplatI16x8, arg0)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Unary(module.UnaryOpSplatI32x4, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Unary(module.UnaryOpSplatI64x2, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Unary(module.UnaryOpSplatI64x2, arg0)
			}
			return mod.Unary(module.UnaryOpSplatI32x4, arg0)
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpSplatF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpSplatF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.splat", typ)
}

func builtinV128ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(operands[1], types.TypeU8, ConstraintsConvImplicit)
	compiler.CurrentType = typ
	idx := validateSIMDLaneIndex(evaluateSIMDConstantIndex(arg1, operands[1], compiler), typ, operands[1], compiler)
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneI8x16, arg0, idx)
		case types.TypeKindU8:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneU8x16, arg0, idx)
		case types.TypeKindI16:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneI16x8, arg0, idx)
		case types.TypeKindU16:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneU16x8, arg0, idx)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneI32x4, arg0, idx)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneI64x2, arg0, idx)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDExtract(module.SIMDExtractOpExtractLaneI64x2, arg0, idx)
			}
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneI32x4, arg0, idx)
		case types.TypeKindF32:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneF32x4, arg0, idx)
		case types.TypeKindF64:
			return mod.SIMDExtract(module.SIMDExtractOpExtractLaneF64x2, arg0, idx)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.extract_lane", typ)
}

func builtinV128ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 3)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(operands[1], types.TypeU8, ConstraintsConvImplicit)
	arg2 := compiler.CompileExpression(operands[2], typ, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	idx := validateSIMDLaneIndex(evaluateSIMDConstantIndex(arg1, operands[1], compiler), typ, operands[1], compiler)
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI8x16, arg0, idx, arg2)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI16x8, arg0, idx, arg2)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI32x4, arg0, idx, arg2)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI64x2, arg0, idx, arg2)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI64x2, arg0, idx, arg2)
			}
			return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI32x4, arg0, idx, arg2)
		case types.TypeKindF32:
			return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneF32x4, arg0, idx, arg2)
		case types.TypeKindF64:
			return mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneF64x2, arg0, idx, arg2)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.replace_lane", typ)
}

func builtinV128Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Binary(module.BinaryOpAddI8x16, arg0, arg1)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Binary(module.BinaryOpAddI16x8, arg0, arg1)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Binary(module.BinaryOpAddI32x4, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Binary(module.BinaryOpAddI64x2, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpAddI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpAddI32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpAddF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpAddF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.add", typ)
}

func builtinV128Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Binary(module.BinaryOpSubI8x16, arg0, arg1)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Binary(module.BinaryOpSubI16x8, arg0, arg1)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Binary(module.BinaryOpSubI32x4, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Binary(module.BinaryOpSubI64x2, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpSubI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpSubI32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpSubF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpSubF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.sub", typ)
}

func builtinV128Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Binary(module.BinaryOpMulI16x8, arg0, arg1)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Binary(module.BinaryOpMulI32x4, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Binary(module.BinaryOpMulI64x2, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpMulI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpMulI32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpMulF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpMulF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.mul", typ)
}

func builtinV128Div(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpDivF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpDivF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.div", typ)
}

func builtinV128Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Unary(module.UnaryOpNegI8x16, arg0)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Unary(module.UnaryOpNegI16x8, arg0)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Unary(module.UnaryOpNegI32x4, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Unary(module.UnaryOpNegI64x2, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Unary(module.UnaryOpNegI64x2, arg0)
			}
			return mod.Unary(module.UnaryOpNegI32x4, arg0)
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpNegF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpNegF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.neg", typ)
}

func builtinV128Min(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpMinI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpMinU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpMinI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpMinU16x8, arg0, arg1)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpMinI32x4, arg0, arg1)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpMinU32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpMinF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpMinF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.min", typ)
}

func builtinV128Max(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpMaxI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpMaxU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpMaxI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpMaxU16x8, arg0, arg1)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpMaxI32x4, arg0, arg1)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpMaxU32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpMaxF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpMaxF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.max", typ)
}

func builtinV128Pmin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpPminF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpPminF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.pmin", typ)
}

func builtinV128Pmax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpPmaxF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpPmaxF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.pmax", typ)
}

func builtinV128Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Unary(module.UnaryOpAbsI8x16, arg0)
		case types.TypeKindI16:
			return mod.Unary(module.UnaryOpAbsI16x8, arg0)
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpAbsI32x4, arg0)
		case types.TypeKindI64:
			return mod.Unary(module.UnaryOpAbsI64x2, arg0)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				return mod.Unary(module.UnaryOpAbsI64x2, arg0)
			}
			return mod.Unary(module.UnaryOpAbsI32x4, arg0)
		case types.TypeKindU8, types.TypeKindU16, types.TypeKindU32, types.TypeKindU64, types.TypeKindUsize:
			return arg0
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpAbsF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpAbsF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.abs", typ)
}

func builtinV128Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpSqrtF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpSqrtF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.sqrt", typ)
}

func builtinV128Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpCeilF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpCeilF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.ceil", typ)
}

func builtinV128Floor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpFloorF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpFloorF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.floor", typ)
}

func builtinV128Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpTruncF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpTruncF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.trunc", typ)
}

func builtinV128Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpNearestF32x4, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpNearestF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.nearest", typ)
}

func builtinV128Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Binary(module.BinaryOpEqI8x16, arg0, arg1)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Binary(module.BinaryOpEqI16x8, arg0, arg1)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Binary(module.BinaryOpEqI32x4, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Binary(module.BinaryOpEqI64x2, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpEqI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpEqI32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpEqF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpEqF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.eq", typ)
}

func builtinV128Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Binary(module.BinaryOpNeI8x16, arg0, arg1)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Binary(module.BinaryOpNeI16x8, arg0, arg1)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Binary(module.BinaryOpNeI32x4, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Binary(module.BinaryOpNeI64x2, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpNeI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpNeI32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpNeF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpNeF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.ne", typ)
}

func builtinV128Lt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpLtI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpLtU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpLtI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpLtU16x8, arg0, arg1)
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpLtI32x4, arg0, arg1)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpLtU32x4, arg0, arg1)
		case types.TypeKindI64:
			return mod.Binary(module.BinaryOpLtI64x2, arg0, arg1)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpLtI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpLtI32x4, arg0, arg1)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			return mod.Binary(module.BinaryOpLtU32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpLtF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpLtF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.lt", typ)
}

func builtinV128Le(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpLeI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpLeU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpLeI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpLeU16x8, arg0, arg1)
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpLeI32x4, arg0, arg1)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpLeU32x4, arg0, arg1)
		case types.TypeKindI64:
			return mod.Binary(module.BinaryOpLeI64x2, arg0, arg1)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpLeI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpLeI32x4, arg0, arg1)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			return mod.Binary(module.BinaryOpLeU32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpLeF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpLeF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.le", typ)
}

func builtinV128Gt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpGtI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpGtU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpGtI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpGtU16x8, arg0, arg1)
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpGtI32x4, arg0, arg1)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpGtU32x4, arg0, arg1)
		case types.TypeKindI64:
			return mod.Binary(module.BinaryOpGtI64x2, arg0, arg1)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpGtI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpGtI32x4, arg0, arg1)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			return mod.Binary(module.BinaryOpGtU32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpGtF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpGtF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.gt", typ)
}

func builtinV128Ge(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpGeI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpGeU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpGeI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpGeU16x8, arg0, arg1)
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpGeI32x4, arg0, arg1)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpGeU32x4, arg0, arg1)
		case types.TypeKindI64:
			return mod.Binary(module.BinaryOpGeI64x2, arg0, arg1)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				return mod.Binary(module.BinaryOpGeI64x2, arg0, arg1)
			}
			return mod.Binary(module.BinaryOpGeI32x4, arg0, arg1)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			return mod.Binary(module.BinaryOpGeU32x4, arg0, arg1)
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpGeF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpGeF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.ge", typ)
}

func builtinV128Shl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeI32, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.SIMDShift(module.SIMDShiftOpShlI8x16, arg0, arg1)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.SIMDShift(module.SIMDShiftOpShlI16x8, arg0, arg1)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.SIMDShift(module.SIMDShiftOpShlI32x4, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.SIMDShift(module.SIMDShiftOpShlI64x2, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDShift(module.SIMDShiftOpShlI64x2, arg0, arg1)
			}
			return mod.SIMDShift(module.SIMDShiftOpShlI32x4, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.shl", typ)
}

func builtinV128Shr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeI32, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.SIMDShift(module.SIMDShiftOpShrI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.SIMDShift(module.SIMDShiftOpShrU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.SIMDShift(module.SIMDShiftOpShrI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.SIMDShift(module.SIMDShiftOpShrU16x8, arg0, arg1)
		case types.TypeKindI32:
			return mod.SIMDShift(module.SIMDShiftOpShrI32x4, arg0, arg1)
		case types.TypeKindU32:
			return mod.SIMDShift(module.SIMDShiftOpShrU32x4, arg0, arg1)
		case types.TypeKindI64:
			return mod.SIMDShift(module.SIMDShiftOpShrI64x2, arg0, arg1)
		case types.TypeKindU64:
			return mod.SIMDShift(module.SIMDShiftOpShrU64x2, arg0, arg1)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDShift(module.SIMDShiftOpShrI64x2, arg0, arg1)
			}
			return mod.SIMDShift(module.SIMDShiftOpShrI32x4, arg0, arg1)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDShift(module.SIMDShiftOpShrU64x2, arg0, arg1)
			}
			return mod.SIMDShift(module.SIMDShiftOpShrU32x4, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.shr", typ)
}

func builtinV128AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeBool)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Unary(module.UnaryOpAllTrueI8x16, arg0)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Unary(module.UnaryOpAllTrueI16x8, arg0)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Unary(module.UnaryOpAllTrueI32x4, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Unary(module.UnaryOpAllTrueI64x2, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Unary(module.UnaryOpAllTrueI64x2, arg0)
			}
			return mod.Unary(module.UnaryOpAllTrueI32x4, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.all_true", typ)
}

func builtinV128Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeI32)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Unary(module.UnaryOpBitmaskI8x16, arg0)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.Unary(module.UnaryOpBitmaskI16x8, arg0)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Unary(module.UnaryOpBitmaskI32x4, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Unary(module.UnaryOpBitmaskI64x2, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.Unary(module.UnaryOpBitmaskI64x2, arg0)
			}
			return mod.Unary(module.UnaryOpBitmaskI32x4, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.bitmask", typ)
}

func builtinV128Popcnt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.Unary(module.UnaryOpPopcntI8x16, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.popcnt", typ)
}

func builtinV128AddSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpAddSatI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpAddSatU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpAddSatI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpAddSatU16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.add_sat", typ)
}

func builtinV128SubSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpSubSatI8x16, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpSubSatU8x16, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpSubSatI16x8, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpSubSatU16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.sub_sat", typ)
}

func builtinV128Avgr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpAvgrU8x16, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpAvgrU16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.avgr", typ)
}

func builtinV128Narrow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpNarrowI16x8ToI8x16, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpNarrowU16x8ToU8x16, arg0, arg1)
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpNarrowI32x4ToI16x8, arg0, arg1)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpNarrowU32x4ToU16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.narrow", typ)
}

func builtinV128Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, false)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	if typ.IsValue() {
		laneWidth := typ.ByteSize()
		laneCount := int32(16 / laneWidth)
		if checkArgsRequired(ctx, int(2+laneCount)) {
			compiler.CurrentType = types.TypeV128
			return mod.Unreachable()
		}
		arg0 := compiler.CompileExpression(operands[0], types.TypeV128, ConstraintsConvImplicit)
		arg1 := compiler.CompileExpression(operands[1], types.TypeV128, ConstraintsConvImplicit)
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindI16, types.TypeKindI32, types.TypeKindI64, types.TypeKindIsize,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32, types.TypeKindU64, types.TypeKindUsize,
			types.TypeKindF32, types.TypeKindF64:
			var mask [16]byte
			maxIdx := (laneCount << 1) - 1
			for i := int32(0); i < laneCount; i++ {
				operand := operands[2+i]
				argN := compiler.CompileExpression(operand, types.TypeU8, ConstraintsConvImplicit)
				idx := evaluateSIMDConstantIndex(argN, operand, compiler)
				if idx < 0 || idx > maxIdx {
					compiler.Error(
						diagnostics.DiagnosticCode0MustBeAValueBetween1And2Inclusive,
						operand.GetRange(),
						"Lane index",
						"0",
						intToString(int(maxIdx)),
					)
					idx = 0
				}
				offset := i * laneWidth
				idx8 := idx * laneWidth
				for j := int32(0); j < laneWidth; j++ {
					mask[offset+j] = byte(idx8 + j)
				}
			}
			compiler.CurrentType = types.TypeV128
			return mod.SIMDShuffle(arg0, arg1, mask)
		}
	}
	compiler.CurrentType = types.TypeV128
	return reportBuiltinOperationTypeError(ctx, "v128.shuffle", typ)
}

func builtinV128Swizzle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinV128BitwiseBinaryOp(ctx, module.BinaryOpSwizzleI8x16)
}

func builtinV128LoadExt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 1, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	offset, align, ok := evaluateSIMDMemoryImmediateOperands(operands, 1, typ.ByteSize(), compiler)
	if !ok {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.SIMDLoad(module.SIMDLoadOpLoad8x8S, arg0, offset, align, "")
		case types.TypeKindU8:
			return mod.SIMDLoad(module.SIMDLoadOpLoad8x8U, arg0, offset, align, "")
		case types.TypeKindI16:
			return mod.SIMDLoad(module.SIMDLoadOpLoad16x4S, arg0, offset, align, "")
		case types.TypeKindU16:
			return mod.SIMDLoad(module.SIMDLoadOpLoad16x4U, arg0, offset, align, "")
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.SIMDLoad(module.SIMDLoadOpLoad32x2S, arg0, offset, align, "")
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.SIMDLoad(module.SIMDLoadOpLoad32x2U, arg0, offset, align, "")
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.load_ext", typ)
}

func builtinV128LoadSplat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 1, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	offset, align, ok := evaluateSIMDMemoryImmediateOperands(operands, 1, typ.ByteSize(), compiler)
	if !ok {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.SIMDLoad(module.SIMDLoadOpLoad8Splat, arg0, offset, align, "")
		case types.TypeKindI16, types.TypeKindU16:
			return mod.SIMDLoad(module.SIMDLoadOpLoad16Splat, arg0, offset, align, "")
		case types.TypeKindI32, types.TypeKindU32, types.TypeKindF32:
			return mod.SIMDLoad(module.SIMDLoadOpLoad32Splat, arg0, offset, align, "")
		case types.TypeKindIsize, types.TypeKindUsize:
			if !compiler.Options().IsWasm64() {
				return mod.SIMDLoad(module.SIMDLoadOpLoad32Splat, arg0, offset, align, "")
			}
			fallthrough
		case types.TypeKindI64, types.TypeKindU64, types.TypeKindF64:
			return mod.SIMDLoad(module.SIMDLoadOpLoad64Splat, arg0, offset, align, "")
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.load_splat", typ)
}

func builtinV128LoadZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 1, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	offset, align, ok := evaluateSIMDMemoryImmediateOperands(operands, 1, typ.ByteSize(), compiler)
	if !ok {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI32, types.TypeKindU32, types.TypeKindF32:
			return mod.SIMDLoad(module.SIMDLoadOpLoad32Zero, arg0, offset, align, "")
		case types.TypeKindI64, types.TypeKindU64, types.TypeKindF64:
			return mod.SIMDLoad(module.SIMDLoadOpLoad64Zero, arg0, offset, align, "")
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDLoad(module.SIMDLoadOpLoad64Zero, arg0, offset, align, "")
			}
			return mod.SIMDLoad(module.SIMDLoadOpLoad32Zero, arg0, offset, align, "")
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.load_zero", typ)
}

func builtinV128LoadLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 3, 5)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(operands[1], types.TypeV128, ConstraintsConvImplicit)
	arg2 := compiler.CompileExpression(operands[2], types.TypeU8, ConstraintsConvImplicit)
	idx := validateSIMDLaneIndex(evaluateSIMDConstantIndex(arg2, operands[2], compiler), typ, operands[1], compiler)
	offset, align, ok := evaluateSIMDMemoryImmediateOperands(operands, 3, typ.ByteSize(), compiler)
	if !ok {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpLoad8Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindI16, types.TypeKindU16:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpLoad16Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindI32, types.TypeKindU32, types.TypeKindF32:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpLoad32Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindI64, types.TypeKindU64, types.TypeKindF64:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpLoad64Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpLoad64Lane, arg0, offset, align, idx, arg1, "")
			}
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpLoad32Lane, arg0, offset, align, idx, arg1, "")
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.load_lane", typ)
}

func builtinV128StoreLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 3, 5)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(operands[1], types.TypeV128, ConstraintsConvImplicit)
	arg2 := compiler.CompileExpression(operands[2], types.TypeU8, ConstraintsConvImplicit)
	idx := validateSIMDLaneIndex(evaluateSIMDConstantIndex(arg2, operands[2], compiler), typ, operands[1], compiler)
	offset, align, ok := evaluateSIMDMemoryImmediateOperands(operands, 3, typ.ByteSize(), compiler)
	if !ok {
		compiler.CurrentType = types.TypeVoid
		return mod.Unreachable()
	}
	compiler.CurrentType = types.TypeVoid
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpStore8Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindI16, types.TypeKindU16:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpStore16Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindI32, types.TypeKindU32, types.TypeKindF32:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpStore32Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindI64, types.TypeKindU64, types.TypeKindF64:
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpStore64Lane, arg0, offset, align, idx, arg1, "")
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpStore64Lane, arg0, offset, align, idx, arg1, "")
			}
			return mod.SIMDLoadStoreLane(module.SIMDLoadStoreLaneOpStore32Lane, arg0, offset, align, idx, arg1, "")
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.store_lane", typ)
}

func builtinV128Convert(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpConvertI32x4ToF32x4, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpConvertU32x4ToF32x4, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.convert", typ)
}

func builtinV128ConvertLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpConvertLowI32x4ToF64x2, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpConvertLowU32x4ToF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.convert_low", typ)
}

func builtinV128TruncSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpTruncSatF32x4ToI32x4, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpTruncSatF32x4ToU32x4, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.trunc_sat", typ)
}

func builtinV128TruncSatZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpTruncSatF64x2ToI32x4Zero, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpTruncSatF64x2ToU32x4Zero, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.trunc_sat_zero", typ)
}

func builtinV128ExtendLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Unary(module.UnaryOpExtendLowI8x16ToI16x8, arg0)
		case types.TypeKindU8:
			return mod.Unary(module.UnaryOpExtendLowU8x16ToU16x8, arg0)
		case types.TypeKindI16:
			return mod.Unary(module.UnaryOpExtendLowI16x8ToI32x4, arg0)
		case types.TypeKindU16:
			return mod.Unary(module.UnaryOpExtendLowU16x8ToU32x4, arg0)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpExtendLowI32x4ToI64x2, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpExtendLowU32x4ToU64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.extend_low", typ)
}

func builtinV128ExtendHigh(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Unary(module.UnaryOpExtendHighI8x16ToI16x8, arg0)
		case types.TypeKindU8:
			return mod.Unary(module.UnaryOpExtendHighU8x16ToU16x8, arg0)
		case types.TypeKindI16:
			return mod.Unary(module.UnaryOpExtendHighI16x8ToI32x4, arg0)
		case types.TypeKindU16:
			return mod.Unary(module.UnaryOpExtendHighU16x8ToU32x4, arg0)
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpExtendHighI32x4ToI64x2, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpExtendHighU32x4ToU64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.extend_high", typ)
}

func builtinV128ExtaddPairwise(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareRequiredV128UnaryBuiltin(ctx, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Unary(module.UnaryOpExtaddPairwiseI8x16ToI16x8, arg0)
		case types.TypeKindU8:
			return mod.Unary(module.UnaryOpExtaddPairwiseU8x16ToU16x8, arg0)
		case types.TypeKindI16:
			return mod.Unary(module.UnaryOpExtaddPairwiseI16x8ToI32x4, arg0)
		case types.TypeKindU16:
			return mod.Unary(module.UnaryOpExtaddPairwiseU16x8ToU32x4, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.extadd_pairwise", typ)
}

func builtinV128DemoteZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareOptionalV128UnaryBuiltin(ctx, types.TypeF64, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpDemoteZeroF64x2ToF32x4, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.demote_zero", typ)
}

func builtinV128PromoteLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, ok := prepareOptionalV128UnaryBuiltin(ctx, types.TypeF32, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpPromoteLowF32x4ToF64x2, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.promote_low", typ)
}

func builtinV128Q15mulrSat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpQ15mulrSatI16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.q15mulr_sat", typ)
}

func builtinV128ExtmulLow(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpExtmulLowI16x8, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpExtmulLowU16x8, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpExtmulLowI32x4, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpExtmulLowU32x4, arg0, arg1)
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpExtmulLowI64x2, arg0, arg1)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpExtmulLowU64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.extmul_low", typ)
}

func builtinV128ExtmulHigh(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8:
			return mod.Binary(module.BinaryOpExtmulHighI16x8, arg0, arg1)
		case types.TypeKindU8:
			return mod.Binary(module.BinaryOpExtmulHighU16x8, arg0, arg1)
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpExtmulHighI32x4, arg0, arg1)
		case types.TypeKindU16:
			return mod.Binary(module.BinaryOpExtmulHighU32x4, arg0, arg1)
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpExtmulHighI64x2, arg0, arg1)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpExtmulHighU64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.extmul_high", typ)
}

func builtinV128Dot(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, ok := prepareRequiredV128BinaryBuiltin(ctx, types.TypeV128, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpDotI16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.dot", typ)
}

func builtinV128RelaxedSwizzle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureRelaxedSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	return mod.Binary(module.BinaryOpRelaxedSwizzleI8x16, arg0, arg1)
}

func builtinV128RelaxedTrunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureRelaxedSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpRelaxedTruncF32x4ToI32x4, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpRelaxedTruncF32x4ToU32x4, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_trunc", typ)
}

func builtinV128RelaxedTruncZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureRelaxedSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindIsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindI32:
			return mod.Unary(module.UnaryOpRelaxedTruncF64x2ToI32x4Zero, arg0)
		case types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				break
			}
			fallthrough
		case types.TypeKindU32:
			return mod.Unary(module.UnaryOpRelaxedTruncF64x2ToU32x4Zero, arg0)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_trunc_zero", typ)
}

func builtinV128RelaxedMadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, arg2, ok := prepareRequiredV128TernaryBuiltin(ctx, common.FeatureRelaxedSimd, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedMaddF32x4, arg0, arg1, arg2)
		case types.TypeKindF64:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedMaddF64x2, arg0, arg1, arg2)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_madd", typ)
}

func builtinV128RelaxedNmadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	_, mod, typ, arg0, arg1, arg2, ok := prepareRequiredV128TernaryBuiltin(ctx, common.FeatureRelaxedSimd, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedNmaddF32x4, arg0, arg1, arg2)
		case types.TypeKindF64:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedNmaddF64x2, arg0, arg1, arg2)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_nmadd", typ)
}

func builtinV128RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, arg2, ok := prepareRequiredV128TernaryBuiltin(ctx, common.FeatureRelaxedSimd, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindU8:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedLaneselectI8x16, arg0, arg1, arg2)
		case types.TypeKindI16, types.TypeKindU16:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedLaneselectI16x8, arg0, arg1, arg2)
		case types.TypeKindI32, types.TypeKindU32:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedLaneselectI32x4, arg0, arg1, arg2)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedLaneselectI64x2, arg0, arg1, arg2)
		case types.TypeKindIsize, types.TypeKindUsize:
			if compiler.Options().IsWasm64() {
				return mod.SIMDTernary(module.SIMDTernaryOpRelaxedLaneselectI64x2, arg0, arg1, arg2)
			}
			return mod.SIMDTernary(module.SIMDTernaryOpRelaxedLaneselectI32x4, arg0, arg1, arg2)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_laneselect", typ)
}

func builtinV128RelaxedMin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureRelaxedSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpRelaxedMinF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpRelaxedMinF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_min", typ)
}

func builtinV128RelaxedMax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureRelaxedSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpRelaxedMaxF32x4, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpRelaxedMaxF64x2, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_max", typ)
}

func builtinV128RelaxedQ15mulr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureRelaxedSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpRelaxedQ15MulrI16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_q15mulr", typ)
}

func builtinV128RelaxedDot(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureRelaxedSimd))|
		boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI16:
			return mod.Binary(module.BinaryOpRelaxedDotI8x16I7x16ToI16x8, arg0, arg1)
		}
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_dot", typ)
}

func builtinV128RelaxedDotAdd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler, mod, typ, arg0, arg1, arg2, ok := prepareRequiredV128TernaryBuiltin(ctx, common.FeatureRelaxedSimd, types.TypeV128)
	if !ok {
		return mod.Unreachable()
	}
	switch typ.Kind {
	case types.TypeKindIsize:
		if compiler.Options().IsWasm64() {
			break
		}
		fallthrough
	case types.TypeKindI32:
		return mod.SIMDTernary(module.SIMDTernaryOpRelaxedDotI8x16I7x16AddToI32x4, arg0, arg1, arg2)
	}
	return reportBuiltinOperationTypeError(ctx, "v128.relaxed_dot_add", typ)
}

func builtinV128And(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinV128BitwiseBinaryOp(ctx, module.BinaryOpAndV128)
}

func builtinV128Or(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinV128BitwiseBinaryOp(ctx, module.BinaryOpOrV128)
}

func builtinV128Xor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinV128BitwiseBinaryOp(ctx, module.BinaryOpXorV128)
}

func builtinV128Andnot(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinV128BitwiseBinaryOp(ctx, module.BinaryOpAndnotV128)
}

func builtinV128Not(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinV128BitwiseUnaryOp(ctx, module.UnaryOpNotV128)
}

func builtinV128Bitselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 3)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	arg1 := compiler.CompileExpression(ctx.Operands[1], types.TypeV128, ConstraintsConvImplicit)
	arg2 := compiler.CompileExpression(ctx.Operands[2], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeV128
	return mod.SIMDTernary(module.SIMDTernaryOpBitselect, arg0, arg1, arg2)
}

func builtinV128AnyTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 1)) != 0 {
		compiler.CurrentType = types.TypeBool
		return mod.Unreachable()
	}
	arg0 := compiler.CompileExpression(ctx.Operands[0], types.TypeV128, ConstraintsConvImplicit)
	compiler.CurrentType = types.TypeBool
	return mod.Unary(module.UnaryOpAnyTrueV128, arg0)
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

func builtinI32x4Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Splat(ctx)
}
func builtinI32x4ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinV128ExtractLane(ctx)
}
func builtinI32x4ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ReplaceLane(ctx)
}
func builtinI32x4Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Add(ctx)
}
func builtinI32x4Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sub(ctx)
}
func builtinI32x4Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Mul(ctx)
}
func builtinI32x4MinS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}
func builtinI32x4MinU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}
func builtinI32x4MaxS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}
func builtinI32x4MaxU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}
func builtinI32x4DotI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128Dot(ctx)
}
func builtinI32x4Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Abs(ctx)
}
func builtinI32x4Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Neg(ctx)
}
func builtinI32x4Shl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shl(ctx)
}
func builtinI32x4ShrS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}
func builtinI32x4ShrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}
func builtinI32x4AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinV128AllTrue(ctx)
}
func builtinI32x4Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeI32
	return builtinV128Bitmask(ctx)
}
func builtinI32x4Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Eq(ctx)
}
func builtinI32x4Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ne(ctx)
}
func builtinI32x4LtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}
func builtinI32x4LtU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}
func builtinI32x4LeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}
func builtinI32x4LeU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}
func builtinI32x4GtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}
func builtinI32x4GtU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}
func builtinI32x4GeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}
func builtinI32x4GeU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}
func builtinI32x4TruncSatF32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128TruncSat(ctx)
}
func builtinI32x4TruncSatF32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128TruncSat(ctx)
}
func builtinI32x4TruncSatF64x2SZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128TruncSatZero(ctx)
}
func builtinI32x4TruncSatF64x2UZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128TruncSatZero(ctx)
}
func builtinI32x4ExtendLowI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendLow(ctx)
}
func builtinI32x4ExtendLowI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendLow(ctx)
}
func builtinI32x4ExtendHighI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendHigh(ctx)
}
func builtinI32x4ExtendHighI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendHigh(ctx)
}
func builtinI32x4ExtaddPairwiseI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtaddPairwise(ctx)
}
func builtinI32x4ExtaddPairwiseI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtaddPairwise(ctx)
}
func builtinI32x4ExtmulLowI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulLow(ctx)
}
func builtinI32x4ExtmulLowI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulLow(ctx)
}
func builtinI32x4ExtmulHighI16x8S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulHigh(ctx)
}
func builtinI32x4ExtmulHighI16x8U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU16}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulHigh(ctx)
}
func builtinI32x4Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shuffle(ctx)
}

// ========================================================================================
// i64x2 SIMD alias functions
// ========================================================================================

func builtinI64x2Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Splat(ctx)
}
func builtinI64x2ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI64
	return builtinV128ExtractLane(ctx)
}
func builtinI64x2ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128ReplaceLane(ctx)
}
func builtinI64x2Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Add(ctx)
}
func builtinI64x2Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sub(ctx)
}
func builtinI64x2Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Mul(ctx)
}
func builtinI64x2Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Abs(ctx)
}
func builtinI64x2Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Neg(ctx)
}
func builtinI64x2Shl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shl(ctx)
}
func builtinI64x2ShrS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}
func builtinI64x2ShrU(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shr(ctx)
}
func builtinI64x2AllTrue(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI32
	return builtinV128AllTrue(ctx)
}
func builtinI64x2Bitmask(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeI32
	return builtinV128Bitmask(ctx)
}
func builtinI64x2Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Eq(ctx)
}
func builtinI64x2Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ne(ctx)
}
func builtinI64x2LtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}
func builtinI64x2LeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}
func builtinI64x2GtS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}
func builtinI64x2GeS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}
func builtinI64x2ExtendLowI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendLow(ctx)
}
func builtinI64x2ExtendLowI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendLow(ctx)
}
func builtinI64x2ExtendHighI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendHigh(ctx)
}
func builtinI64x2ExtendHighI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtendHigh(ctx)
}
func builtinI64x2ExtmulLowI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulLow(ctx)
}
func builtinI64x2ExtmulLowI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulLow(ctx)
}
func builtinI64x2ExtmulHighI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulHigh(ctx)
}
func builtinI64x2ExtmulHighI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ExtmulHigh(ctx)
}
func builtinI64x2Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shuffle(ctx)
}

// ========================================================================================
// f32x4 SIMD alias functions
// ========================================================================================

func builtinF32x4Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Splat(ctx)
}
func builtinF32x4ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeF32
	return builtinV128ExtractLane(ctx)
}
func builtinF32x4ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ReplaceLane(ctx)
}
func builtinF32x4Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Add(ctx)
}
func builtinF32x4Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sub(ctx)
}
func builtinF32x4Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Mul(ctx)
}
func builtinF32x4Div(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Div(ctx)
}
func builtinF32x4Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Neg(ctx)
}
func builtinF32x4Min(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}
func builtinF32x4Max(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}
func builtinF32x4Pmin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Pmin(ctx)
}
func builtinF32x4Pmax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Pmax(ctx)
}
func builtinF32x4Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Abs(ctx)
}
func builtinF32x4Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sqrt(ctx)
}
func builtinF32x4Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ceil(ctx)
}
func builtinF32x4Floor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Floor(ctx)
}
func builtinF32x4Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Trunc(ctx)
}
func builtinF32x4Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Nearest(ctx)
}
func builtinF32x4Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Eq(ctx)
}
func builtinF32x4Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ne(ctx)
}
func builtinF32x4Lt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}
func builtinF32x4Le(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}
func builtinF32x4Gt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}
func builtinF32x4Ge(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}
func builtinF32x4ConvertI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Convert(ctx)
}
func builtinF32x4ConvertI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Convert(ctx)
}
func builtinF32x4DemoteF64x2Zero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128DemoteZero(ctx)
}
func builtinF32x4Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shuffle(ctx)
}

// ========================================================================================
// f64x2 SIMD alias functions
// ========================================================================================

func builtinF64x2Splat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Splat(ctx)
}
func builtinF64x2ExtractLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeF64
	return builtinV128ExtractLane(ctx)
}
func builtinF64x2ReplaceLane(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128ReplaceLane(ctx)
}
func builtinF64x2Add(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Add(ctx)
}
func builtinF64x2Sub(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sub(ctx)
}
func builtinF64x2Mul(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Mul(ctx)
}
func builtinF64x2Div(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Div(ctx)
}
func builtinF64x2Neg(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Neg(ctx)
}
func builtinF64x2Min(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Min(ctx)
}
func builtinF64x2Max(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Max(ctx)
}
func builtinF64x2Pmin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Pmin(ctx)
}
func builtinF64x2Pmax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Pmax(ctx)
}
func builtinF64x2Abs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Abs(ctx)
}
func builtinF64x2Sqrt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Sqrt(ctx)
}
func builtinF64x2Ceil(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ceil(ctx)
}
func builtinF64x2Floor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Floor(ctx)
}
func builtinF64x2Trunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Trunc(ctx)
}
func builtinF64x2Nearest(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Nearest(ctx)
}
func builtinF64x2Eq(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Eq(ctx)
}
func builtinF64x2Ne(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ne(ctx)
}
func builtinF64x2Lt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Lt(ctx)
}
func builtinF64x2Le(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Le(ctx)
}
func builtinF64x2Gt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Gt(ctx)
}
func builtinF64x2Ge(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Ge(ctx)
}
func builtinF64x2ConvertLowI32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ConvertLow(ctx)
}
func builtinF64x2ConvertLowI32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128ConvertLow(ctx)
}

// Note: TS source names this function builtin_f64x4_promote_low_f32x4 (typo in TS, registered as f64x2_promote_low_f32x4)
func builtinF64x2PromoteLowF32x4(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128PromoteLow(ctx)
}
func builtinF64x2Shuffle(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128Shuffle(ctx)
}

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

func builtinI32x4RelaxedTruncF32x4S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedTrunc(ctx)
}
func builtinI32x4RelaxedTruncF32x4U(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedTrunc(ctx)
}
func builtinI32x4RelaxedTruncF64x2SZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedTruncZero(ctx)
}
func builtinI32x4RelaxedTruncF64x2UZero(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeU32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedTruncZero(ctx)
}
func builtinF32x4RelaxedMadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedMadd(ctx)
}
func builtinF32x4RelaxedNmadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedNmadd(ctx)
}
func builtinF64x2RelaxedMadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedMadd(ctx)
}
func builtinF64x2RelaxedNmadd(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedNmadd(ctx)
}
func builtinI8x16RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI8}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedLaneselect(ctx)
}
func builtinI16x8RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedLaneselect(ctx)
}
func builtinI32x4RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedLaneselect(ctx)
}
func builtinI64x2RelaxedLaneselect(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI64}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedLaneselect(ctx)
}
func builtinF32x4RelaxedMin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedMin(ctx)
}
func builtinF32x4RelaxedMax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedMax(ctx)
}
func builtinF64x2RelaxedMin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedMin(ctx)
}
func builtinF64x2RelaxedMax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeF64}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedMax(ctx)
}
func builtinI16x8RelaxedQ15mulrS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedQ15mulr(ctx)
}
func builtinI16x8RelaxedDotI8x16I7x16S(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI16}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedDot(ctx)
}
func builtinI32x4RelaxedDotI8x16I7x16AddS(ctx *BuiltinFunctionContext) module.ExpressionRef {
	checkTypeAbsent(ctx)
	ctx.TypeArguments = []*types.Type{types.TypeI32}
	ctx.ContextualType = types.TypeV128
	return builtinV128RelaxedDotAdd(ctx)
}

// === v128 constructor builtins ===
// Ported from: builtins.ts:4031-4335

// v128(...values: i8[16]) -> v128
// TS line 4031: builtin_v128 = builtin_i8x16
func builtinV128Ctor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	return builtinI8x16Ctor(ctx)
}

// i8x16(...values: i8[16]) -> v128
// Ported from: builtins.ts:4036-4082
func builtinI8x16Ctor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 16)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	var bytes [16]byte
	vars := make([]module.ExpressionRef, 16)
	numVars := 0

	for i := 0; i < 16; i++ {
		expr := compiler.CompileExpression(operands[i], types.TypeI8, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
		if precomp != 0 {
			bytes[i] = byte(module.GetConstValueI32(precomp))
		} else {
			vars[i] = expr
			numVars++
		}
	}
	compiler.CurrentType = types.TypeV128
	if numVars == 0 {
		return mod.V128(bytes)
	}
	var vec module.ExpressionRef
	fullVars := numVars == 16
	if fullVars {
		vec = mod.Unary(module.UnaryOpSplatI8x16, vars[0])
	} else {
		vec = mod.V128(bytes)
	}
	startIdx := 0
	if fullVars {
		startIdx = 1
	}
	for i := startIdx; i < 16; i++ {
		if vars[i] != 0 {
			vec = mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI8x16, vec, uint8(i), vars[i])
		}
	}
	return vec
}

// i16x8(...values: i16[8]) -> v128
// Ported from: builtins.ts:4086-4132
func builtinI16x8Ctor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 8)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	var bytes [16]byte
	vars := make([]module.ExpressionRef, 8)
	numVars := 0

	for i := 0; i < 8; i++ {
		expr := compiler.CompileExpression(operands[i], types.TypeI16, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
		if precomp != 0 {
			binary.LittleEndian.PutUint16(bytes[i<<1:], uint16(module.GetConstValueI32(precomp)))
		} else {
			vars[i] = expr
			numVars++
		}
	}
	compiler.CurrentType = types.TypeV128
	if numVars == 0 {
		return mod.V128(bytes)
	}
	var vec module.ExpressionRef
	fullVars := numVars == 8
	if fullVars {
		vec = mod.Unary(module.UnaryOpSplatI16x8, vars[0])
	} else {
		vec = mod.V128(bytes)
	}
	startIdx := 0
	if fullVars {
		startIdx = 1
	}
	for i := startIdx; i < 8; i++ {
		if vars[i] != 0 {
			vec = mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI16x8, vec, uint8(i), vars[i])
		}
	}
	return vec
}

// i32x4(...values: i32[4]) -> v128
// Ported from: builtins.ts:4136-4182
func builtinI32x4Ctor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 4)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	var bytes [16]byte
	vars := make([]module.ExpressionRef, 4)
	numVars := 0

	for i := 0; i < 4; i++ {
		expr := compiler.CompileExpression(operands[i], types.TypeI32, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
		if precomp != 0 {
			binary.LittleEndian.PutUint32(bytes[i<<2:], uint32(module.GetConstValueI32(precomp)))
		} else {
			vars[i] = expr
			numVars++
		}
	}
	compiler.CurrentType = types.TypeV128
	if numVars == 0 {
		return mod.V128(bytes)
	}
	var vec module.ExpressionRef
	fullVars := numVars == 4
	if fullVars {
		vec = mod.Unary(module.UnaryOpSplatI32x4, vars[0])
	} else {
		vec = mod.V128(bytes)
	}
	startIdx := 0
	if fullVars {
		startIdx = 1
	}
	for i := startIdx; i < 4; i++ {
		if vars[i] != 0 {
			vec = mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI32x4, vec, uint8(i), vars[i])
		}
	}
	return vec
}

// i64x2(...values: i64[2]) -> v128
// Ported from: builtins.ts:4186-4234
func builtinI64x2Ctor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	var bytes [16]byte
	vars := make([]module.ExpressionRef, 2)
	numVars := 0

	for i := 0; i < 2; i++ {
		expr := compiler.CompileExpression(operands[i], types.TypeI64, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
		if precomp != 0 {
			off := i << 3
			binary.LittleEndian.PutUint32(bytes[off:], uint32(module.GetConstValueI64Low(precomp)))
			binary.LittleEndian.PutUint32(bytes[off+4:], uint32(module.GetConstValueI64High(precomp)))
		} else {
			vars[i] = expr
			numVars++
		}
	}
	compiler.CurrentType = types.TypeV128
	if numVars == 0 {
		return mod.V128(bytes)
	}
	var vec module.ExpressionRef
	fullVars := numVars == 2
	if fullVars {
		vec = mod.Unary(module.UnaryOpSplatI64x2, vars[0])
	} else {
		vec = mod.V128(bytes)
	}
	startIdx := 0
	if fullVars {
		startIdx = 1
	}
	for i := startIdx; i < 2; i++ {
		if vars[i] != 0 {
			vec = mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneI64x2, vec, uint8(i), vars[i])
		}
	}
	return vec
}

// f32x4(...values: f32[4]) -> v128
// Ported from: builtins.ts:4238-4284
func builtinF32x4Ctor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 4)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	var bytes [16]byte
	vars := make([]module.ExpressionRef, 4)
	numVars := 0

	for i := 0; i < 4; i++ {
		expr := compiler.CompileExpression(operands[i], types.TypeF32, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
		if precomp != 0 {
			binary.LittleEndian.PutUint32(bytes[i<<2:], math.Float32bits(module.GetConstValueF32(precomp)))
		} else {
			vars[i] = expr
			numVars++
		}
	}
	compiler.CurrentType = types.TypeV128
	if numVars == 0 {
		return mod.V128(bytes)
	}
	var vec module.ExpressionRef
	fullVars := numVars == 4
	if fullVars {
		vec = mod.Unary(module.UnaryOpSplatF32x4, vars[0])
	} else {
		vec = mod.V128(bytes)
	}
	startIdx := 0
	if fullVars {
		startIdx = 1
	}
	for i := startIdx; i < 4; i++ {
		if vars[i] != 0 {
			vec = mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneF32x4, vec, uint8(i), vars[i])
		}
	}
	return vec
}

// f64x2(...values: f64[2]) -> v128
// Ported from: builtins.ts:4288-4334
func builtinF64x2Ctor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkFeatureEnabled(ctx, common.FeatureSimd))|
		boolToInt(checkTypeAbsent(ctx))|
		boolToInt(checkArgsRequired(ctx, 2)) != 0 {
		compiler.CurrentType = types.TypeV128
		return mod.Unreachable()
	}
	operands := ctx.Operands
	var bytes [16]byte
	vars := make([]module.ExpressionRef, 2)
	numVars := 0

	for i := 0; i < 2; i++ {
		expr := compiler.CompileExpression(operands[i], types.TypeF64, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
		if precomp != 0 {
			binary.LittleEndian.PutUint64(bytes[i<<3:], math.Float64bits(module.GetConstValueF64(precomp)))
		} else {
			vars[i] = expr
			numVars++
		}
	}
	compiler.CurrentType = types.TypeV128
	if numVars == 0 {
		return mod.V128(bytes)
	}
	var vec module.ExpressionRef
	fullVars := numVars == 2
	if fullVars {
		vec = mod.Unary(module.UnaryOpSplatF64x2, vars[0])
	} else {
		vec = mod.V128(bytes)
	}
	startIdx := 0
	if fullVars {
		startIdx = 1
	}
	for i := startIdx; i < 2; i++ {
		if vars[i] != 0 {
			vec = mod.SIMDReplace(module.SIMDReplaceOpReplaceLaneF64x2, vec, uint8(i), vars[i])
		}
	}
	return vec
}

// Ensure imports are used
var _ = binary.LittleEndian
var _ = math.Float32bits
