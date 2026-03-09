// Ported from: assemblyscript/src/module.ts (constants section, lines 1-1315)
//
// These constants are fixed per Binaryen version and match the C API values.
// They must NOT be obtained at runtime via C function calls (except for
// reference/GC types whose numeric encodings are Binaryen-internal).
package module

import (
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
)

// ---------------------------------------------------------------------------
// Type aliases -- re-exported from the binaryen package so that consumers of
// the module package never need to import binaryen directly.
// ---------------------------------------------------------------------------

type HeapTypeRef = binaryen.HeapType
type PackedType = binaryen.PackedType
type ExpressionID = binaryen.ExpressionID
type ExternalKind = binaryen.ExternalKind
type Features = binaryen.Features
type Op = binaryen.Op
type ImportRef = binaryen.ImportRef
type ExportRef = binaryen.ExportRef
type RelooperRef = binaryen.RelooperRef
type RelooperBlockRef = binaryen.RelooperBlockRef
type Index = binaryen.Index
type ExpressionRunnerFlags = binaryen.ExpressionRunnerFlags
type SideEffects = binaryen.SideEffects
type TableRef = binaryen.TableRef

// TypeRef, ExpressionRef, FunctionRef, GlobalRef, TagRef are already
// declared as type aliases in helpers.go.

// ---------------------------------------------------------------------------
// TypeRef constants
// ---------------------------------------------------------------------------

// Core value types -- fixed numeric values per Binaryen version.
const (
	TypeRefNone        TypeRef = 0 // _BinaryenTypeNone
	TypeRefUnreachable TypeRef = 1 // _BinaryenTypeUnreachable
	TypeRefI32         TypeRef = 2 // _BinaryenTypeInt32
	TypeRefI64         TypeRef = 3 // _BinaryenTypeInt64
	TypeRefF32         TypeRef = 4 // _BinaryenTypeFloat32
	TypeRefF64         TypeRef = 5 // _BinaryenTypeFloat64
	TypeRefV128        TypeRef = 6 // _BinaryenTypeVec128
)

// Reference/GC types -- these are runtime-determined because the numeric
// encoding is internal to Binaryen and may change across versions.
var (
	TypeRefFuncref     TypeRef
	TypeRefExternref   TypeRef
	TypeRefAnyref      TypeRef
	TypeRefEqref       TypeRef
	TypeRefStructref   TypeRef
	TypeRefArrayref    TypeRef
	TypeRefI31ref      TypeRef
	TypeRefStringref   TypeRef
	TypeRefNoneref     TypeRef
	TypeRefNofuncref   TypeRef
	TypeRefNoexternref TypeRef
)

func init() {
	TypeRefFuncref = binaryen.TypeFuncref()
	TypeRefExternref = binaryen.TypeExternref()
	TypeRefAnyref = binaryen.TypeAnyref()
	TypeRefEqref = binaryen.TypeEqref()
	TypeRefStructref = binaryen.TypeStructref()
	TypeRefArrayref = binaryen.TypeArrayref()
	TypeRefI31ref = binaryen.TypeI31ref()
	TypeRefStringref = binaryen.TypeStringref()
	TypeRefNoneref = binaryen.TypeNullref()
	TypeRefNofuncref = binaryen.TypeNullFuncref()
	TypeRefNoexternref = binaryen.TypeNullExternref()
}

// ---------------------------------------------------------------------------
// HeapTypeRef constants
// ---------------------------------------------------------------------------

//        any                  extern      func
//         |                      |          |
//     __ eq __          ?     noextern    (...)
//    /    |   \         |                   |
// i31  struct  array  string              nofunc
//  |      |      |      |
// none  (...)  (...)    ?
//         |      |
//        none   none
//
// where (...) represents the concrete subtypes

const (
	HeapTypeRefExtern   HeapTypeRef = 8   // _BinaryenHeapTypeExt
	HeapTypeRefFunc     HeapTypeRef = 16  // _BinaryenHeapTypeFunc
	HeapTypeRefAny      HeapTypeRef = 32  // _BinaryenHeapTypeAny
	HeapTypeRefEq       HeapTypeRef = 40  // _BinaryenHeapTypeEq
	HeapTypeRefI31      HeapTypeRef = 48  // _BinaryenHeapTypeI31
	HeapTypeRefStruct   HeapTypeRef = 56  // _BinaryenHeapTypeStruct
	HeapTypeRefArray    HeapTypeRef = 64  // _BinaryenHeapTypeArray
	HeapTypeRefExn      HeapTypeRef = 7   // BinaryenHeapTypeExn
	HeapTypeRefString   HeapTypeRef = 80  // _BinaryenHeapTypeString
	HeapTypeRefNone     HeapTypeRef = 88  // _BinaryenHeapTypeNone
	HeapTypeRefNoextern HeapTypeRef = 96  // _BinaryenHeapTypeNoext
	HeapTypeRefNofunc   HeapTypeRef = 104 // _BinaryenHeapTypeNofunc
)

// HeapTypeIsBottom returns whether the given heap type is a bottom type.
func HeapTypeIsBottom(ht HeapTypeRef) bool {
	return binaryen.HeapTypeIsBottom(ht)
}

// HeapTypeGetBottom returns the bottom type for the given heap type's hierarchy.
func HeapTypeGetBottom(ht HeapTypeRef) HeapTypeRef {
	return binaryen.HeapTypeGetBottom(ht)
}

// HeapTypeIsSubtype returns whether ht is a subtype of superHt.
func HeapTypeIsSubtype(ht, superHt HeapTypeRef) bool {
	return binaryen.HeapTypeIsSubType(ht, superHt)
}

// HeapTypeLeastUpperBound computes the least upper bound of two heap types.
// Returns ^uintptr(0) (all bits set, i.e. -1 as signed) if they are
// incomparable.
func HeapTypeLeastUpperBound(a, b HeapTypeRef) HeapTypeRef {
	// Mirrors binaryen/src/wasm/wasm-type.cpp
	const invalid = ^HeapTypeRef(0) // -1 as unsigned

	if a == b {
		return a
	}
	if HeapTypeGetBottom(a) != HeapTypeGetBottom(b) {
		return invalid
	}
	if HeapTypeIsBottom(a) {
		return b
	}
	if HeapTypeIsBottom(b) {
		return a
	}
	if a > b {
		a, b = b, a
	}
	switch a {
	case HeapTypeRefExtern:
		if b == HeapTypeRefString {
			return a
		}
		return invalid
	case HeapTypeRefFunc:
		return invalid
	case HeapTypeRefAny:
		return a
	case HeapTypeRefEq:
		if b == HeapTypeRefI31 || b == HeapTypeRefStruct || b == HeapTypeRefArray {
			return HeapTypeRefEq
		}
		return HeapTypeRefAny
	case HeapTypeRefI31:
		if b == HeapTypeRefStruct || b == HeapTypeRefArray {
			return HeapTypeRefEq
		}
		return HeapTypeRefAny
	case HeapTypeRefStruct:
		if b == HeapTypeRefArray {
			return HeapTypeRefEq
		}
		return HeapTypeRefAny
	case HeapTypeRefArray:
		return HeapTypeRefAny
	}
	return invalid
}

// ---------------------------------------------------------------------------
// PackedType constants
// ---------------------------------------------------------------------------

const (
	PackedTypeNotPacked PackedType = 0 // _BinaryenPackedTypeNotPacked
	PackedTypeI8        PackedType = 1 // _BinaryenPackedTypeInt8
	PackedTypeI16       PackedType = 2 // _BinaryenPackedTypeInt16
)

// ---------------------------------------------------------------------------
// TypeBuilderErrorReason
// ---------------------------------------------------------------------------

// TypeBuilderErrorReason describes why a type builder operation failed.
type TypeBuilderErrorReason = uint32

const (
	TypeBuilderErrorReasonSelfSupertype            TypeBuilderErrorReason = 0 // _TypeBuilderErrorReasonSelfSupertype
	TypeBuilderErrorReasonInvalidSupertype          TypeBuilderErrorReason = 1 // _TypeBuilderErrorReasonInvalidSupertype
	TypeBuilderErrorReasonForwardSupertypeReference TypeBuilderErrorReason = 2 // _TypeBuilderErrorReasonForwardSupertypeReference
	TypeBuilderErrorReasonForwardChildReference     TypeBuilderErrorReason = 3 // _TypeBuilderErrorReasonForwardChildReference
)

// TypeBuilderErrorReasonToString converts a type builder error reason to a
// human-readable string.
func TypeBuilderErrorReasonToString(reason TypeBuilderErrorReason) string {
	switch reason {
	case TypeBuilderErrorReasonSelfSupertype:
		return "SelfSupertype"
	case TypeBuilderErrorReasonInvalidSupertype:
		return "InvalidSupertype"
	case TypeBuilderErrorReasonForwardSupertypeReference:
		return "ForwardSupertypeReference"
	case TypeBuilderErrorReasonForwardChildReference:
		return "ForwardChildReference"
	}
	return ""
}

// ---------------------------------------------------------------------------
// FeatureFlags
// ---------------------------------------------------------------------------

const (
	FeatureFlagMVP                Features = 0       // _BinaryenFeatureMVP
	FeatureFlagAtomics            Features = 1       // _BinaryenFeatureAtomics
	FeatureFlagMutableGlobals     Features = 2       // _BinaryenFeatureMutableGlobals
	FeatureFlagTruncSat           Features = 4       // _BinaryenFeatureNontrappingFPToInt
	FeatureFlagSIMD               Features = 8       // _BinaryenFeatureSIMD128
	FeatureFlagBulkMemory         Features = 16      // _BinaryenFeatureBulkMemory
	FeatureFlagSignExt            Features = 32      // _BinaryenFeatureSignExt
	FeatureFlagExceptionHandling  Features = 64      // _BinaryenFeatureExceptionHandling
	FeatureFlagTailCall           Features = 128     // _BinaryenFeatureTailCall
	FeatureFlagReferenceTypes     Features = 256     // _BinaryenFeatureReferenceTypes
	FeatureFlagMultiValue         Features = 512     // _BinaryenFeatureMultivalue
	FeatureFlagGC                 Features = 1024    // _BinaryenFeatureGC
	FeatureFlagMemory64           Features = 2048    // _BinaryenFeatureMemory64
	FeatureFlagRelaxedSIMD        Features = 4096    // _BinaryenFeatureRelaxedSIMD
	FeatureFlagExtendedConst      Features = 8192    // _BinaryenFeatureExtendedConst
	FeatureFlagStringref          Features = 16384   // _BinaryenFeatureStrings
	FeatureFlagMultiMemory        Features = 32768   // _BinaryenFeatureMultiMemory
	FeatureFlagStackSwitching     Features = 65536   // _BinaryenFeatureStackSwitching
	FeatureFlagSharedEverything    Features = 131072  // _BinaryenFeatureSharedEverything
	FeatureFlagFP16               Features = 262144  // _BinaryenFeatureFP16
	FeatureFlagBulkMemoryOpt      Features = 524288  // _BinaryenFeatureBulkMemoryOpt
	FeatureFlagCallIndirectOverlong Features = 1048576 // _BinaryenFeatureCallIndirectOverlong
	FeatureFlagAll                Features = 4194303 // _BinaryenFeatureAll
)

// ---------------------------------------------------------------------------
// ExpressionId
// ---------------------------------------------------------------------------

const (
	ExpressionIdInvalid          ExpressionID = 0  // _BinaryenInvalidId
	ExpressionIdBlock            ExpressionID = 1  // _BinaryenBlockId
	ExpressionIdIf               ExpressionID = 2  // _BinaryenIfId
	ExpressionIdLoop             ExpressionID = 3  // _BinaryenLoopId
	ExpressionIdBreak            ExpressionID = 4  // _BinaryenBreakId
	ExpressionIdSwitch           ExpressionID = 5  // _BinaryenSwitchId
	ExpressionIdCall             ExpressionID = 6  // _BinaryenCallId
	ExpressionIdCallIndirect     ExpressionID = 7  // _BinaryenCallIndirectId
	ExpressionIdLocalGet         ExpressionID = 8  // _BinaryenLocalGetId
	ExpressionIdLocalSet         ExpressionID = 9  // _BinaryenLocalSetId
	ExpressionIdGlobalGet        ExpressionID = 10 // _BinaryenGlobalGetId
	ExpressionIdGlobalSet        ExpressionID = 11 // _BinaryenGlobalSetId
	ExpressionIdLoad             ExpressionID = 12 // _BinaryenLoadId
	ExpressionIdStore            ExpressionID = 13 // _BinaryenStoreId
	ExpressionIdConst            ExpressionID = 14 // _BinaryenConstId
	ExpressionIdUnary            ExpressionID = 15 // _BinaryenUnaryId
	ExpressionIdBinary           ExpressionID = 16 // _BinaryenBinaryId
	ExpressionIdSelect           ExpressionID = 17 // _BinaryenSelectId
	ExpressionIdDrop             ExpressionID = 18 // _BinaryenDropId
	ExpressionIdReturn           ExpressionID = 19 // _BinaryenReturnId
	ExpressionIdMemorySize       ExpressionID = 20 // _BinaryenMemorySizeId
	ExpressionIdMemoryGrow       ExpressionID = 21 // _BinaryenMemoryGrowId
	ExpressionIdNop              ExpressionID = 22 // _BinaryenNopId
	ExpressionIdUnreachable      ExpressionID = 23 // _BinaryenUnreachableId
	ExpressionIdAtomicRMW        ExpressionID = 24 // _BinaryenAtomicRMWId
	ExpressionIdAtomicCmpxchg    ExpressionID = 25 // _BinaryenAtomicCmpxchgId
	ExpressionIdAtomicWait       ExpressionID = 26 // _BinaryenAtomicWaitId
	ExpressionIdAtomicNotify     ExpressionID = 27 // _BinaryenAtomicNotifyId
	ExpressionIdAtomicFence      ExpressionID = 28 // _BinaryenAtomicFenceId
	ExpressionIdSIMDExtract      ExpressionID = 29 // _BinaryenSIMDExtractId
	ExpressionIdSIMDReplace      ExpressionID = 30 // _BinaryenSIMDReplaceId
	ExpressionIdSIMDShuffle      ExpressionID = 31 // _BinaryenSIMDShuffleId
	ExpressionIdSIMDTernary      ExpressionID = 32 // _BinaryenSIMDTernaryId
	ExpressionIdSIMDShift        ExpressionID = 33 // _BinaryenSIMDShiftId
	ExpressionIdSIMDLoad         ExpressionID = 34 // _BinaryenSIMDLoadId
	ExpressionIdSIMDLoadStoreLane ExpressionID = 35 // _BinaryenSIMDLoadStoreLaneId
	ExpressionIdMemoryInit       ExpressionID = 36 // _BinaryenMemoryInitId
	ExpressionIdDataDrop         ExpressionID = 37 // _BinaryenDataDropId
	ExpressionIdMemoryCopy       ExpressionID = 38 // _BinaryenMemoryCopyId
	ExpressionIdMemoryFill       ExpressionID = 39 // _BinaryenMemoryFillId
	ExpressionIdPop              ExpressionID = 40 // _BinaryenPopId
	ExpressionIdRefNull          ExpressionID = 41 // _BinaryenRefNullId
	ExpressionIdRefIsNull        ExpressionID = 42 // _BinaryenRefIsNullId
	ExpressionIdRefFunc          ExpressionID = 43 // _BinaryenRefFuncId
	ExpressionIdRefEq            ExpressionID = 44 // _BinaryenRefEqId
	ExpressionIdTableGet         ExpressionID = 45 // _BinaryenTableGetId
	ExpressionIdTableSet         ExpressionID = 46 // _BinaryenTableSetId
	ExpressionIdTableSize        ExpressionID = 47 // _BinaryenTableSizeId
	ExpressionIdTableGrow        ExpressionID = 48 // _BinaryenTableGrowId
	ExpressionIdTableFill        ExpressionID = 49 // _BinaryenTableFillId
	ExpressionIdTableCopy        ExpressionID = 50 // _BinaryenTableCopyId
	ExpressionIdTableInit        ExpressionID = 51 // _BinaryenTableInitId
	ExpressionIdTry              ExpressionID = 52 // _BinaryenTryId
	ExpressionIdTryTable         ExpressionID = 53 // _BinaryenTryTableId
	ExpressionIdThrow            ExpressionID = 54 // _BinaryenThrowId
	ExpressionIdRethrow          ExpressionID = 55 // _BinaryenRethrowId
	ExpressionIdThrowRef         ExpressionID = 56 // _BinaryenThrowRefId
	ExpressionIdTupleMake        ExpressionID = 57 // _BinaryenTupleMakeId
	ExpressionIdTupleExtract     ExpressionID = 58 // _BinaryenTupleExtractId
	ExpressionIdRefI31           ExpressionID = 59 // _BinaryenRefI31Id
	ExpressionIdI31Get           ExpressionID = 60 // _BinaryenI31GetId
	ExpressionIdCallRef          ExpressionID = 61 // _BinaryenCallRefId
	ExpressionIdRefTest          ExpressionID = 62 // _BinaryenRefTestId
	ExpressionIdRefCast          ExpressionID = 63 // _BinaryenRefCastId
	ExpressionIdRefGetDesc       ExpressionID = 64 // _BinaryenRefGetDescId
	ExpressionIdBrOn             ExpressionID = 65 // _BinaryenBrOnId
	ExpressionIdStructNew        ExpressionID = 66 // _BinaryenStructNewId
	ExpressionIdStructGet        ExpressionID = 67 // _BinaryenStructGetId
	ExpressionIdStructSet        ExpressionID = 68 // _BinaryenStructSetId
	ExpressionIdStructRMW        ExpressionID = 69 // _BinaryenStructRMWId
	ExpressionIdStructCmpxchg    ExpressionID = 70 // _BinaryenStructCmpxchgId
	ExpressionIdArrayNew         ExpressionID = 71 // _BinaryenArrayNewId
	ExpressionIdArrayNewData     ExpressionID = 72 // _BinaryenArrayNewDataId
	ExpressionIdArrayNewElem     ExpressionID = 73 // _BinaryenArrayNewElemId
	ExpressionIdArrayNewFixed    ExpressionID = 74 // _BinaryenArrayNewFixedId
	ExpressionIdArrayGet         ExpressionID = 75 // _BinaryenArrayGetId
	ExpressionIdArraySet         ExpressionID = 76 // _BinaryenArraySetId
	ExpressionIdArrayLen         ExpressionID = 77 // _BinaryenArrayLenId
	ExpressionIdArrayCopy        ExpressionID = 78 // _BinaryenArrayCopyId
	ExpressionIdArrayFill        ExpressionID = 79 // _BinaryenArrayFillId
	ExpressionIdArrayInitData    ExpressionID = 80 // _BinaryenArrayInitDataId
	ExpressionIdArrayInitElem    ExpressionID = 81 // _BinaryenArrayInitElemId
	ExpressionIdRefAs            ExpressionID = 82 // _BinaryenRefAsId
	ExpressionIdStringNew        ExpressionID = 83 // _BinaryenStringNewId
	ExpressionIdStringConst      ExpressionID = 84 // _BinaryenStringConstId
	ExpressionIdStringMeasure    ExpressionID = 85 // _BinaryenStringMeasureId
	ExpressionIdStringEncode     ExpressionID = 86 // _BinaryenStringEncodeId
	ExpressionIdStringConcat     ExpressionID = 87 // _BinaryenStringConcatId
	ExpressionIdStringEq         ExpressionID = 88 // _BinaryenStringEqId
	ExpressionIdStringWTF16Get   ExpressionID = 89 // _BinaryenStringWTF16GetId
	ExpressionIdStringSliceWTF   ExpressionID = 90 // _BinaryenStringSliceWTFId
	ExpressionIdContNew          ExpressionID = 91 // _BinaryenContNewId
	ExpressionIdContBind         ExpressionID = 92 // _BinaryenContBindId
	ExpressionIdSuspend          ExpressionID = 93 // _BinaryenSuspendId
	ExpressionIdResume           ExpressionID = 94 // _BinaryenResumeId
	ExpressionIdResumeThrow      ExpressionID = 95 // _BinaryenResumeThrowId
	ExpressionIdStackSwitch      ExpressionID = 96 // _BinaryenStackSwitchId
)

// ---------------------------------------------------------------------------
// ExternalKind
// ---------------------------------------------------------------------------

const (
	ExternalKindFunction ExternalKind = 0 // _BinaryenExternalFunction
	ExternalKindTable    ExternalKind = 1 // _BinaryenExternalTable
	ExternalKindMemory   ExternalKind = 2 // _BinaryenExternalMemory
	ExternalKindGlobal   ExternalKind = 3 // _BinaryenExternalGlobal
	ExternalKindTag      ExternalKind = 4 // _BinaryenExternalTag
)

// ---------------------------------------------------------------------------
// UnaryOp
// ---------------------------------------------------------------------------

const (
	UnaryOpClzI32       Op = 0  // i32.clz
	UnaryOpClzI64       Op = 1  // i64.clz
	UnaryOpCtzI32       Op = 2  // i32.ctz
	UnaryOpCtzI64       Op = 3  // i64.ctz
	UnaryOpPopcntI32    Op = 4  // i32.popcnt
	UnaryOpPopcntI64    Op = 5  // i64.popcnt
	UnaryOpNegF32       Op = 6  // f32.neg
	UnaryOpNegF64       Op = 7  // f64.neg
	UnaryOpAbsF32       Op = 8  // f32.abs
	UnaryOpAbsF64       Op = 9  // f64.abs
	UnaryOpCeilF32      Op = 10 // f32.ceil
	UnaryOpCeilF64      Op = 11 // f64.ceil
	UnaryOpFloorF32     Op = 12 // f32.floor
	UnaryOpFloorF64     Op = 13 // f64.floor
	UnaryOpTruncF32     Op = 14 // f32.trunc
	UnaryOpTruncF64     Op = 15 // f64.trunc
	UnaryOpNearestF32   Op = 16 // f32.nearest
	UnaryOpNearestF64   Op = 17 // f64.nearest
	UnaryOpSqrtF32      Op = 18 // f32.sqrt
	UnaryOpSqrtF64      Op = 19 // f64.sqrt
	UnaryOpEqzI32       Op = 20 // i32.eqz
	UnaryOpEqzI64       Op = 21 // i64.eqz
	UnaryOpExtendI32ToI64   Op = 22 // i64.extend_i32_s
	UnaryOpExtendU32ToU64   Op = 23 // i64.extend_i32_u
	UnaryOpWrapI64ToI32     Op = 24 // i32.wrap_i64
	UnaryOpTruncF32ToI32    Op = 25 // i32.trunc_f32_s
	UnaryOpTruncF32ToI64    Op = 26 // i64.trunc_f32_s
	UnaryOpTruncF32ToU32    Op = 27 // i32.trunc_f32_u
	UnaryOpTruncF32ToU64    Op = 28 // i64.trunc_f32_u
	UnaryOpTruncF64ToI32    Op = 29 // i32.trunc_f64_s
	UnaryOpTruncF64ToI64    Op = 30 // i64.trunc_f64_s
	UnaryOpTruncF64ToU32    Op = 31 // i32.trunc_f64_u
	UnaryOpTruncF64ToU64    Op = 32 // i64.trunc_f64_u
	UnaryOpReinterpretF32ToI32 Op = 33 // i32.reinterpret_f32
	UnaryOpReinterpretF64ToI64 Op = 34 // i64.reinterpret_f64
	UnaryOpConvertI32ToF32  Op = 35 // f32.convert_i32_s
	UnaryOpConvertI32ToF64  Op = 36 // f64.convert_i32_s
	UnaryOpConvertU32ToF32  Op = 37 // f32.convert_i32_u
	UnaryOpConvertU32ToF64  Op = 38 // f64.convert_i32_u
	UnaryOpConvertI64ToF32  Op = 39 // f32.convert_i64_s
	UnaryOpConvertI64ToF64  Op = 40 // f64.convert_i64_s
	UnaryOpConvertU64ToF32  Op = 41 // f32.convert_i64_u
	UnaryOpConvertU64ToF64  Op = 42 // f64.convert_i64_u
	UnaryOpPromoteF32ToF64  Op = 43 // f64.promote_f32
	UnaryOpDemoteF64ToF32   Op = 44 // f32.demote_f64
	UnaryOpReinterpretI32ToF32 Op = 45 // f32.reinterpret_i32
	UnaryOpReinterpretI64ToF64 Op = 46 // f64.reinterpret_i64

	// Sign extension ops
	UnaryOpExtend8I32  Op = 47 // i32.extend8_s
	UnaryOpExtend16I32 Op = 48 // i32.extend16_s
	UnaryOpExtend8I64  Op = 49 // i64.extend8_s
	UnaryOpExtend16I64 Op = 50 // i64.extend16_s
	UnaryOpExtend32I64 Op = 51 // i64.extend32_s (i64 in, i64 out)

	// Saturating truncation ops
	UnaryOpTruncSatF32ToI32 Op = 52 // i32.trunc_sat_f32_s
	UnaryOpTruncSatF32ToU32 Op = 53 // i32.trunc_sat_f32_u
	UnaryOpTruncSatF64ToI32 Op = 54 // i32.trunc_sat_f64_s
	UnaryOpTruncSatF64ToU32 Op = 55 // i32.trunc_sat_f64_u
	UnaryOpTruncSatF32ToI64 Op = 56 // i64.trunc_sat_f32_s
	UnaryOpTruncSatF32ToU64 Op = 57 // i64.trunc_sat_f32_u
	UnaryOpTruncSatF64ToI64 Op = 58 // i64.trunc_sat_f64_s
	UnaryOpTruncSatF64ToU64 Op = 59 // i64.trunc_sat_f64_u

	// SIMD splat
	UnaryOpSplatI8x16 Op = 60 // i8x16.splat
	UnaryOpSplatI16x8 Op = 61 // i16x8.splat
	UnaryOpSplatI32x4 Op = 62 // i32x4.splat
	UnaryOpSplatI64x2 Op = 63 // i64x2.splat
	UnaryOpSplatF32x4 Op = 64 // f32x4.splat
	UnaryOpSplatF64x2 Op = 65 // f64x2.splat

	// SIMD bitwise
	UnaryOpNotV128     Op = 66 // v128.not
	UnaryOpAnyTrueV128 Op = 67 // v128.any_true

	// SIMD i8x16 unary
	UnaryOpAbsI8x16     Op = 68 // i8x16.abs
	UnaryOpNegI8x16     Op = 69 // i8x16.neg
	UnaryOpAllTrueI8x16 Op = 70 // i8x16.all_true
	UnaryOpBitmaskI8x16 Op = 71 // i8x16.bitmask
	UnaryOpPopcntI8x16  Op = 72 // i8x16.popcnt

	// SIMD i16x8 unary
	UnaryOpAbsI16x8     Op = 73 // i16x8.abs
	UnaryOpNegI16x8     Op = 74 // i16x8.neg
	UnaryOpAllTrueI16x8 Op = 75 // i16x8.all_true
	UnaryOpBitmaskI16x8 Op = 76 // i16x8.bitmask

	// SIMD i32x4 unary
	UnaryOpAbsI32x4     Op = 77 // i32x4.abs
	UnaryOpNegI32x4     Op = 78 // i32x4.neg
	UnaryOpAllTrueI32x4 Op = 79 // i32x4.all_true
	UnaryOpBitmaskI32x4 Op = 80 // i32x4.bitmask

	// SIMD i64x2 unary
	UnaryOpAbsI64x2     Op = 81 // i64x2.abs
	UnaryOpNegI64x2     Op = 82 // i64x2.neg
	UnaryOpAllTrueI64x2 Op = 83 // i64x2.all_true
	UnaryOpBitmaskI64x2 Op = 84 // i64x2.bitmask

	// NOTE: values 85-91 are reserved for F16 ops (not yet in C API)

	// SIMD f32x4 unary
	UnaryOpAbsF32x4     Op = 92  // f32x4.abs
	UnaryOpNegF32x4     Op = 93  // f32x4.neg
	UnaryOpSqrtF32x4    Op = 94  // f32x4.sqrt
	UnaryOpCeilF32x4    Op = 95  // f32x4.ceil
	UnaryOpFloorF32x4   Op = 96  // f32x4.floor
	UnaryOpTruncF32x4   Op = 97  // f32x4.trunc
	UnaryOpNearestF32x4 Op = 98  // f32x4.nearest

	// SIMD f64x2 unary
	UnaryOpAbsF64x2     Op = 99  // f64x2.abs
	UnaryOpNegF64x2     Op = 100 // f64x2.neg
	UnaryOpSqrtF64x2    Op = 101 // f64x2.sqrt
	UnaryOpCeilF64x2    Op = 102 // f64x2.ceil
	UnaryOpFloorF64x2   Op = 103 // f64x2.floor
	UnaryOpTruncF64x2   Op = 104 // f64x2.trunc
	UnaryOpNearestF64x2 Op = 105 // f64x2.nearest

	// SIMD extended pairwise addition
	UnaryOpExtaddPairwiseI8x16ToI16x8 Op = 106 // i16x8.extadd_pairwise_i8x16_s
	UnaryOpExtaddPairwiseU8x16ToU16x8 Op = 107 // i16x8.extadd_pairwise_i8x16_u
	UnaryOpExtaddPairwiseI16x8ToI32x4 Op = 108 // i32x4.extadd_pairwise_i16x8_s
	UnaryOpExtaddPairwiseU16x8ToU32x4 Op = 109 // i32x4.extadd_pairwise_i16x8_u

	// SIMD truncation/conversion
	UnaryOpTruncSatF32x4ToI32x4  Op = 110 // i32x4.trunc_sat_f32x4_s
	UnaryOpTruncSatF32x4ToU32x4  Op = 111 // i32x4.trunc_sat_f32x4_u
	UnaryOpConvertI32x4ToF32x4   Op = 112 // f32x4.convert_i32x4_s
	UnaryOpConvertU32x4ToF32x4   Op = 113 // f32x4.convert_i32x4_u

	// SIMD extend (widen)
	UnaryOpExtendLowI8x16ToI16x8   Op = 114 // i16x8.extend_low_i8x16_s
	UnaryOpExtendHighI8x16ToI16x8  Op = 115 // i16x8.extend_high_i8x16_s
	UnaryOpExtendLowU8x16ToU16x8   Op = 116 // i16x8.extend_low_i8x16_u
	UnaryOpExtendHighU8x16ToU16x8  Op = 117 // i16x8.extend_high_i8x16_u
	UnaryOpExtendLowI16x8ToI32x4   Op = 118 // i32x4.extend_low_i16x8_s
	UnaryOpExtendHighI16x8ToI32x4  Op = 119 // i32x4.extend_high_i16x8_s
	UnaryOpExtendLowU16x8ToU32x4   Op = 120 // i32x4.extend_low_i16x8_u
	UnaryOpExtendHighU16x8ToU32x4  Op = 121 // i32x4.extend_high_i16x8_u
	UnaryOpExtendLowI32x4ToI64x2   Op = 122 // i64x2.extend_low_i32x4_s
	UnaryOpExtendHighI32x4ToI64x2  Op = 123 // i64x2.extend_high_i32x4_s
	UnaryOpExtendLowU32x4ToU64x2   Op = 124 // i64x2.extend_low_i32x4_u
	UnaryOpExtendHighU32x4ToU64x2  Op = 125 // i64x2.extend_high_i32x4_u

	// SIMD F64x2 <-> I32x4/F32x4 conversions
	UnaryOpConvertLowI32x4ToF64x2      Op = 126 // f64x2.convert_low_i32x4_s
	UnaryOpConvertLowU32x4ToF64x2      Op = 127 // f64x2.convert_low_i32x4_u
	UnaryOpTruncSatF64x2ToI32x4Zero    Op = 128 // i32x4.trunc_sat_f64x2_s_zero
	UnaryOpTruncSatF64x2ToU32x4Zero    Op = 129 // i32x4.trunc_sat_f64x2_u_zero
	UnaryOpDemoteZeroF64x2ToF32x4      Op = 130 // f32x4.demote_f64x2_zero
	UnaryOpPromoteLowF32x4ToF64x2      Op = 131 // f64x2.promote_low_f32x4

	// Relaxed SIMD truncation
	UnaryOpRelaxedTruncF32x4ToI32x4      Op = 132 // i32x4.relaxed_trunc_f32x4_s
	UnaryOpRelaxedTruncF32x4ToU32x4      Op = 133 // i32x4.relaxed_trunc_f32x4_u
	UnaryOpRelaxedTruncF64x2ToI32x4Zero  Op = 134 // i32x4.relaxed_trunc_f64x2_s_zero
	UnaryOpRelaxedTruncF64x2ToU32x4Zero  Op = 135 // i32x4.relaxed_trunc_f64x2_u_zero

	unaryOpLast = UnaryOpRelaxedTruncF64x2ToU32x4Zero

	// Target-dependent size variants (placed above _last as consecutive constants)
	UnaryOpClzSize    Op = 136 // i32.clz or i64.clz depending on target word size
	UnaryOpCtzSize    Op = 137 // i32.ctz or i64.ctz depending on target word size
	UnaryOpPopcntSize Op = 138 // i32.popcnt or i64.popcnt depending on target word size
	UnaryOpEqzSize    Op = 139 // i32.eqz or i64.eqz depending on target word size
)

// ---------------------------------------------------------------------------
// BinaryOp
// ---------------------------------------------------------------------------

const (
	// i32 arithmetic and comparison
	BinaryOpAddI32  Op = 0  // i32.add
	BinaryOpSubI32  Op = 1  // i32.sub
	BinaryOpMulI32  Op = 2  // i32.mul
	BinaryOpDivI32  Op = 3  // i32.div_s
	BinaryOpDivU32  Op = 4  // i32.div_u
	BinaryOpRemI32  Op = 5  // i32.rem_s
	BinaryOpRemU32  Op = 6  // i32.rem_u
	BinaryOpAndI32  Op = 7  // i32.and
	BinaryOpOrI32   Op = 8  // i32.or
	BinaryOpXorI32  Op = 9  // i32.xor
	BinaryOpShlI32  Op = 10 // i32.shl
	BinaryOpShrI32  Op = 11 // i32.shr_s
	BinaryOpShrU32  Op = 12 // i32.shr_u
	BinaryOpRotlI32 Op = 13 // i32.rotl
	BinaryOpRotrI32 Op = 14 // i32.rotr
	BinaryOpEqI32   Op = 15 // i32.eq
	BinaryOpNeI32   Op = 16 // i32.ne
	BinaryOpLtI32   Op = 17 // i32.lt_s
	BinaryOpLtU32   Op = 18 // i32.lt_u
	BinaryOpLeI32   Op = 19 // i32.le_s
	BinaryOpLeU32   Op = 20 // i32.le_u
	BinaryOpGtI32   Op = 21 // i32.gt_s
	BinaryOpGtU32   Op = 22 // i32.gt_u
	BinaryOpGeI32   Op = 23 // i32.ge_s
	BinaryOpGeU32   Op = 24 // i32.ge_u

	// i64 arithmetic and comparison
	BinaryOpAddI64  Op = 25 // i64.add
	BinaryOpSubI64  Op = 26 // i64.sub
	BinaryOpMulI64  Op = 27 // i64.mul
	BinaryOpDivI64  Op = 28 // i64.div_s
	BinaryOpDivU64  Op = 29 // i64.div_u
	BinaryOpRemI64  Op = 30 // i64.rem_s
	BinaryOpRemU64  Op = 31 // i64.rem_u
	BinaryOpAndI64  Op = 32 // i64.and
	BinaryOpOrI64   Op = 33 // i64.or
	BinaryOpXorI64  Op = 34 // i64.xor
	BinaryOpShlI64  Op = 35 // i64.shl
	BinaryOpShrI64  Op = 36 // i64.shr_s
	BinaryOpShrU64  Op = 37 // i64.shr_u
	BinaryOpRotlI64 Op = 38 // i64.rotl
	BinaryOpRotrI64 Op = 39 // i64.rotr
	BinaryOpEqI64   Op = 40 // i64.eq
	BinaryOpNeI64   Op = 41 // i64.ne
	BinaryOpLtI64   Op = 42 // i64.lt_s
	BinaryOpLtU64   Op = 43 // i64.lt_u
	BinaryOpLeI64   Op = 44 // i64.le_s
	BinaryOpLeU64   Op = 45 // i64.le_u
	BinaryOpGtI64   Op = 46 // i64.gt_s
	BinaryOpGtU64   Op = 47 // i64.gt_u
	BinaryOpGeI64   Op = 48 // i64.ge_s
	BinaryOpGeU64   Op = 49 // i64.ge_u

	// f32 arithmetic and comparison
	BinaryOpAddF32      Op = 50 // f32.add
	BinaryOpSubF32      Op = 51 // f32.sub
	BinaryOpMulF32      Op = 52 // f32.mul
	BinaryOpDivF32      Op = 53 // f32.div
	BinaryOpCopysignF32 Op = 54 // f32.copysign
	BinaryOpMinF32      Op = 55 // f32.min
	BinaryOpMaxF32      Op = 56 // f32.max
	BinaryOpEqF32       Op = 57 // f32.eq
	BinaryOpNeF32       Op = 58 // f32.ne
	BinaryOpLtF32       Op = 59 // f32.lt
	BinaryOpLeF32       Op = 60 // f32.le
	BinaryOpGtF32       Op = 61 // f32.gt
	BinaryOpGeF32       Op = 62 // f32.ge

	// f64 arithmetic and comparison
	BinaryOpAddF64      Op = 63 // f64.add
	BinaryOpSubF64      Op = 64 // f64.sub
	BinaryOpMulF64      Op = 65 // f64.mul
	BinaryOpDivF64      Op = 66 // f64.div
	BinaryOpCopysignF64 Op = 67 // f64.copysign
	BinaryOpMinF64      Op = 68 // f64.min
	BinaryOpMaxF64      Op = 69 // f64.max
	BinaryOpEqF64       Op = 70 // f64.eq
	BinaryOpNeF64       Op = 71 // f64.ne
	BinaryOpLtF64       Op = 72 // f64.lt
	BinaryOpLeF64       Op = 73 // f64.le
	BinaryOpGtF64       Op = 74 // f64.gt
	BinaryOpGeF64       Op = 75 // f64.ge

	// SIMD i8x16 comparison
	BinaryOpEqI8x16  Op = 76 // i8x16.eq
	BinaryOpNeI8x16  Op = 77 // i8x16.ne
	BinaryOpLtI8x16  Op = 78 // i8x16.lt_s
	BinaryOpLtU8x16  Op = 79 // i8x16.lt_u
	BinaryOpGtI8x16  Op = 80 // i8x16.gt_s
	BinaryOpGtU8x16  Op = 81 // i8x16.gt_u
	BinaryOpLeI8x16  Op = 82 // i8x16.le_s
	BinaryOpLeU8x16  Op = 83 // i8x16.le_u
	BinaryOpGeI8x16  Op = 84 // i8x16.ge_s
	BinaryOpGeU8x16  Op = 85 // i8x16.ge_u

	// SIMD i16x8 comparison
	BinaryOpEqI16x8  Op = 86 // i16x8.eq
	BinaryOpNeI16x8  Op = 87 // i16x8.ne
	BinaryOpLtI16x8  Op = 88 // i16x8.lt_s
	BinaryOpLtU16x8  Op = 89 // i16x8.lt_u
	BinaryOpGtI16x8  Op = 90 // i16x8.gt_s
	BinaryOpGtU16x8  Op = 91 // i16x8.gt_u
	BinaryOpLeI16x8  Op = 92 // i16x8.le_s
	BinaryOpLeU16x8  Op = 93 // i16x8.le_u
	BinaryOpGeI16x8  Op = 94 // i16x8.ge_s
	BinaryOpGeU16x8  Op = 95 // i16x8.ge_u

	// SIMD i32x4 comparison
	BinaryOpEqI32x4  Op = 96  // i32x4.eq
	BinaryOpNeI32x4  Op = 97  // i32x4.ne
	BinaryOpLtI32x4  Op = 98  // i32x4.lt_s
	BinaryOpLtU32x4  Op = 99  // i32x4.lt_u
	BinaryOpGtI32x4  Op = 100 // i32x4.gt_s
	BinaryOpGtU32x4  Op = 101 // i32x4.gt_u
	BinaryOpLeI32x4  Op = 102 // i32x4.le_s
	BinaryOpLeU32x4  Op = 103 // i32x4.le_u
	BinaryOpGeI32x4  Op = 104 // i32x4.ge_s
	BinaryOpGeU32x4  Op = 105 // i32x4.ge_u

	// SIMD i64x2 comparison
	BinaryOpEqI64x2  Op = 106 // i64x2.eq
	BinaryOpNeI64x2  Op = 107 // i64x2.ne
	BinaryOpLtI64x2  Op = 108 // i64x2.lt_s
	BinaryOpGtI64x2  Op = 109 // i64x2.gt_s
	BinaryOpLeI64x2  Op = 110 // i64x2.le_s
	BinaryOpGeI64x2  Op = 111 // i64x2.ge_s

	// NOTE: values 112-117 are reserved for F16 ops (not yet in C API)

	// SIMD f32x4 comparison
	BinaryOpEqF32x4  Op = 118 // f32x4.eq
	BinaryOpNeF32x4  Op = 119 // f32x4.ne
	BinaryOpLtF32x4  Op = 120 // f32x4.lt
	BinaryOpGtF32x4  Op = 121 // f32x4.gt
	BinaryOpLeF32x4  Op = 122 // f32x4.le
	BinaryOpGeF32x4  Op = 123 // f32x4.ge

	// SIMD f64x2 comparison
	BinaryOpEqF64x2  Op = 124 // f64x2.eq
	BinaryOpNeF64x2  Op = 125 // f64x2.ne
	BinaryOpLtF64x2  Op = 126 // f64x2.lt
	BinaryOpGtF64x2  Op = 127 // f64x2.gt
	BinaryOpLeF64x2  Op = 128 // f64x2.le
	BinaryOpGeF64x2  Op = 129 // f64x2.ge

	// SIMD v128 bitwise
	BinaryOpAndV128    Op = 130 // v128.and
	BinaryOpOrV128     Op = 131 // v128.or
	BinaryOpXorV128    Op = 132 // v128.xor
	BinaryOpAndnotV128 Op = 133 // v128.andnot

	// SIMD i8x16 arithmetic
	BinaryOpAddI8x16    Op = 134 // i8x16.add
	BinaryOpAddSatI8x16 Op = 135 // i8x16.add_sat_s
	BinaryOpAddSatU8x16 Op = 136 // i8x16.add_sat_u
	BinaryOpSubI8x16    Op = 137 // i8x16.sub
	BinaryOpSubSatI8x16 Op = 138 // i8x16.sub_sat_s
	BinaryOpSubSatU8x16 Op = 139 // i8x16.sub_sat_u
	BinaryOpMinI8x16    Op = 140 // i8x16.min_s
	BinaryOpMinU8x16    Op = 141 // i8x16.min_u
	BinaryOpMaxI8x16    Op = 142 // i8x16.max_s
	BinaryOpMaxU8x16    Op = 143 // i8x16.max_u
	BinaryOpAvgrU8x16   Op = 144 // i8x16.avgr_u

	// SIMD i16x8 arithmetic
	BinaryOpAddI16x8      Op = 145 // i16x8.add
	BinaryOpAddSatI16x8   Op = 146 // i16x8.add_sat_s
	BinaryOpAddSatU16x8   Op = 147 // i16x8.add_sat_u
	BinaryOpSubI16x8      Op = 148 // i16x8.sub
	BinaryOpSubSatI16x8   Op = 149 // i16x8.sub_sat_s
	BinaryOpSubSatU16x8   Op = 150 // i16x8.sub_sat_u
	BinaryOpMulI16x8      Op = 151 // i16x8.mul
	BinaryOpMinI16x8      Op = 152 // i16x8.min_s
	BinaryOpMinU16x8      Op = 153 // i16x8.min_u
	BinaryOpMaxI16x8      Op = 154 // i16x8.max_s
	BinaryOpMaxU16x8      Op = 155 // i16x8.max_u
	BinaryOpAvgrU16x8     Op = 156 // i16x8.avgr_u
	BinaryOpQ15mulrSatI16x8 Op = 157 // i16x8.q15mulr_sat_s
	BinaryOpExtmulLowI16x8  Op = 158 // i16x8.extmul_low_i8x16_s
	BinaryOpExtmulHighI16x8 Op = 159 // i16x8.extmul_high_i8x16_s
	BinaryOpExtmulLowU16x8  Op = 160 // i16x8.extmul_low_i8x16_u
	BinaryOpExtmulHighU16x8 Op = 161 // i16x8.extmul_high_i8x16_u

	// SIMD i32x4 arithmetic
	BinaryOpAddI32x4        Op = 162 // i32x4.add
	BinaryOpSubI32x4        Op = 163 // i32x4.sub
	BinaryOpMulI32x4        Op = 164 // i32x4.mul
	BinaryOpMinI32x4        Op = 165 // i32x4.min_s
	BinaryOpMinU32x4        Op = 166 // i32x4.min_u
	BinaryOpMaxI32x4        Op = 167 // i32x4.max_s
	BinaryOpMaxU32x4        Op = 168 // i32x4.max_u
	BinaryOpDotI16x8        Op = 169 // i32x4.dot_i16x8_s
	BinaryOpExtmulLowI32x4  Op = 170 // i32x4.extmul_low_i16x8_s
	BinaryOpExtmulHighI32x4 Op = 171 // i32x4.extmul_high_i16x8_s
	BinaryOpExtmulLowU32x4  Op = 172 // i32x4.extmul_low_i16x8_u
	BinaryOpExtmulHighU32x4 Op = 173 // i32x4.extmul_high_i16x8_u

	// SIMD i64x2 arithmetic
	BinaryOpAddI64x2        Op = 174 // i64x2.add
	BinaryOpSubI64x2        Op = 175 // i64x2.sub
	BinaryOpMulI64x2        Op = 176 // i64x2.mul
	BinaryOpExtmulLowI64x2  Op = 177 // i64x2.extmul_low_i32x4_s
	BinaryOpExtmulHighI64x2 Op = 178 // i64x2.extmul_high_i32x4_s
	BinaryOpExtmulLowU64x2  Op = 179 // i64x2.extmul_low_i32x4_u
	BinaryOpExtmulHighU64x2 Op = 180 // i64x2.extmul_high_i32x4_u

	// NOTE: values 181-188 are reserved for F16 ops (not yet in C API)

	// SIMD f32x4 arithmetic
	BinaryOpAddF32x4  Op = 189 // f32x4.add
	BinaryOpSubF32x4  Op = 190 // f32x4.sub
	BinaryOpMulF32x4  Op = 191 // f32x4.mul
	BinaryOpDivF32x4  Op = 192 // f32x4.div
	BinaryOpMinF32x4  Op = 193 // f32x4.min
	BinaryOpMaxF32x4  Op = 194 // f32x4.max
	BinaryOpPminF32x4 Op = 195 // f32x4.pmin
	BinaryOpPmaxF32x4 Op = 196 // f32x4.pmax

	// SIMD f64x2 arithmetic
	BinaryOpAddF64x2  Op = 197 // f64x2.add
	BinaryOpSubF64x2  Op = 198 // f64x2.sub
	BinaryOpMulF64x2  Op = 199 // f64x2.mul
	BinaryOpDivF64x2  Op = 200 // f64x2.div
	BinaryOpMinF64x2  Op = 201 // f64x2.min
	BinaryOpMaxF64x2  Op = 202 // f64x2.max
	BinaryOpPminF64x2 Op = 203 // f64x2.pmin
	BinaryOpPmaxF64x2 Op = 204 // f64x2.pmax

	// SIMD narrowing
	BinaryOpNarrowI16x8ToI8x16 Op = 205 // i8x16.narrow_i16x8_s
	BinaryOpNarrowU16x8ToU8x16 Op = 206 // i8x16.narrow_i16x8_u
	BinaryOpNarrowI32x4ToI16x8 Op = 207 // i16x8.narrow_i32x4_s
	BinaryOpNarrowU32x4ToU16x8 Op = 208 // i16x8.narrow_i32x4_u

	// SIMD swizzle
	BinaryOpSwizzleI8x16 Op = 209 // i8x16.swizzle

	// Relaxed SIMD
	BinaryOpRelaxedSwizzleI8x16        Op = 210 // i8x16.relaxed_swizzle
	BinaryOpRelaxedMinF32x4            Op = 211 // f32x4.relaxed_min
	BinaryOpRelaxedMaxF32x4            Op = 212 // f32x4.relaxed_max
	BinaryOpRelaxedMinF64x2            Op = 213 // f64x2.relaxed_min
	BinaryOpRelaxedMaxF64x2            Op = 214 // f64x2.relaxed_max
	BinaryOpRelaxedQ15MulrI16x8        Op = 215 // i16x8.relaxed_q15mulr_s
	BinaryOpRelaxedDotI8x16I7x16ToI16x8 Op = 216 // i16x8.relaxed_dot_i8x16_i7x16_s

	binaryOpLast = BinaryOpRelaxedDotI8x16I7x16ToI16x8

	// Target-dependent size variants (placed above _last as consecutive constants)
	BinaryOpAddSize  Op = 217 // i32.add or i64.add depending on target word size
	BinaryOpSubSize  Op = 218 // i32.sub or i64.sub depending on target word size
	BinaryOpMulSize  Op = 219 // i32.mul or i64.mul depending on target word size
	BinaryOpDivISize Op = 220 // i32.div_s or i64.div_s depending on target word size
	BinaryOpDivUSize Op = 221 // i32.div_u or i64.div_u depending on target word size
	BinaryOpRemISize Op = 222 // i32.rem_s or i64.rem_s depending on target word size
	BinaryOpRemUSize Op = 223 // i32.rem_u or i64.rem_u depending on target word size
	BinaryOpAndSize  Op = 224 // i32.and or i64.and depending on target word size
	BinaryOpOrSize   Op = 225 // i32.or or i64.or depending on target word size
	BinaryOpXorSize  Op = 226 // i32.xor or i64.xor depending on target word size
	BinaryOpShlSize  Op = 227 // i32.shl or i64.shl depending on target word size
	BinaryOpShrISize Op = 228 // i32.shr_s or i64.shr_s depending on target word size
	BinaryOpShrUSize Op = 229 // i32.shr_u or i64.shr_u depending on target word size
	BinaryOpRotlSize Op = 230 // i32.rotl or i64.rotl depending on target word size
	BinaryOpRotrSize Op = 231 // i32.rotr or i64.rotr depending on target word size
	BinaryOpEqSize   Op = 232 // i32.eq or i64.eq depending on target word size
	BinaryOpNeSize   Op = 233 // i32.ne or i64.ne depending on target word size
	BinaryOpLtISize  Op = 234 // i32.lt_s or i64.lt_s depending on target word size
	BinaryOpLtUSize  Op = 235 // i32.lt_u or i64.lt_u depending on target word size
	BinaryOpLeISize  Op = 236 // i32.le_s or i64.le_s depending on target word size
	BinaryOpLeUSize  Op = 237 // i32.le_u or i64.le_u depending on target word size
	BinaryOpGtISize  Op = 238 // i32.gt_s or i64.gt_s depending on target word size
	BinaryOpGtUSize  Op = 239 // i32.gt_u or i64.gt_u depending on target word size
	BinaryOpGeISize  Op = 240 // i32.ge_s or i64.ge_s depending on target word size
	BinaryOpGeUSize  Op = 241 // i32.ge_u or i64.ge_u depending on target word size
)

// ---------------------------------------------------------------------------
// AtomicRMWOp
// ---------------------------------------------------------------------------

const (
	AtomicRMWOpAdd  Op = 0 // _BinaryenAtomicRMWAdd
	AtomicRMWOpSub  Op = 1 // _BinaryenAtomicRMWSub
	AtomicRMWOpAnd  Op = 2 // _BinaryenAtomicRMWAnd
	AtomicRMWOpOr   Op = 3 // _BinaryenAtomicRMWOr
	AtomicRMWOpXor  Op = 4 // _BinaryenAtomicRMWXor
	AtomicRMWOpXchg Op = 5 // _BinaryenAtomicRMWXchg
)

// ---------------------------------------------------------------------------
// SIMDExtractOp
// ---------------------------------------------------------------------------

const (
	SIMDExtractOpExtractLaneI8x16 Op = 0 // i8x16.extract_lane_s
	SIMDExtractOpExtractLaneU8x16 Op = 1 // i8x16.extract_lane_u
	SIMDExtractOpExtractLaneI16x8 Op = 2 // i16x8.extract_lane_s
	SIMDExtractOpExtractLaneU16x8 Op = 3 // i16x8.extract_lane_u
	SIMDExtractOpExtractLaneI32x4 Op = 4 // i32x4.extract_lane
	SIMDExtractOpExtractLaneI64x2 Op = 5 // i64x2.extract_lane
	// NOTE: value 6 is reserved for F16 (not yet in C API)
	SIMDExtractOpExtractLaneF32x4 Op = 7 // f32x4.extract_lane
	SIMDExtractOpExtractLaneF64x2 Op = 8 // f64x2.extract_lane
)

// ---------------------------------------------------------------------------
// SIMDReplaceOp
// ---------------------------------------------------------------------------

const (
	SIMDReplaceOpReplaceLaneI8x16 Op = 0 // i8x16.replace_lane
	SIMDReplaceOpReplaceLaneI16x8 Op = 1 // i16x8.replace_lane
	SIMDReplaceOpReplaceLaneI32x4 Op = 2 // i32x4.replace_lane
	SIMDReplaceOpReplaceLaneI64x2 Op = 3 // i64x2.replace_lane
	// NOTE: value 4 is reserved for F16 (not yet in C API)
	SIMDReplaceOpReplaceLaneF32x4 Op = 5 // f32x4.replace_lane
	SIMDReplaceOpReplaceLaneF64x2 Op = 6 // f64x2.replace_lane
)

// ---------------------------------------------------------------------------
// SIMDShiftOp
// ---------------------------------------------------------------------------

const (
	SIMDShiftOpShlI8x16  Op = 0  // i8x16.shl
	SIMDShiftOpShrI8x16  Op = 1  // i8x16.shr_s
	SIMDShiftOpShrU8x16  Op = 2  // i8x16.shr_u
	SIMDShiftOpShlI16x8  Op = 3  // i16x8.shl
	SIMDShiftOpShrI16x8  Op = 4  // i16x8.shr_s
	SIMDShiftOpShrU16x8  Op = 5  // i16x8.shr_u
	SIMDShiftOpShlI32x4  Op = 6  // i32x4.shl
	SIMDShiftOpShrI32x4  Op = 7  // i32x4.shr_s
	SIMDShiftOpShrU32x4  Op = 8  // i32x4.shr_u
	SIMDShiftOpShlI64x2  Op = 9  // i64x2.shl
	SIMDShiftOpShrI64x2  Op = 10 // i64x2.shr_s
	SIMDShiftOpShrU64x2  Op = 11 // i64x2.shr_u
)

// ---------------------------------------------------------------------------
// SIMDLoadOp
// ---------------------------------------------------------------------------

const (
	SIMDLoadOpLoad8Splat  Op = 0  // v128.load8_splat
	SIMDLoadOpLoad16Splat Op = 1  // v128.load16_splat
	SIMDLoadOpLoad32Splat Op = 2  // v128.load32_splat
	SIMDLoadOpLoad64Splat Op = 3  // v128.load64_splat
	SIMDLoadOpLoad8x8S    Op = 4  // v128.load8x8_s
	SIMDLoadOpLoad8x8U    Op = 5  // v128.load8x8_u
	SIMDLoadOpLoad16x4S   Op = 6  // v128.load16x4_s
	SIMDLoadOpLoad16x4U   Op = 7  // v128.load16x4_u
	SIMDLoadOpLoad32x2S   Op = 8  // v128.load32x2_s
	SIMDLoadOpLoad32x2U   Op = 9  // v128.load32x2_u
	SIMDLoadOpLoad32Zero  Op = 10 // v128.load32_zero
	SIMDLoadOpLoad64Zero  Op = 11 // v128.load64_zero
)

// ---------------------------------------------------------------------------
// SIMDLoadStoreLaneOp
// ---------------------------------------------------------------------------

const (
	SIMDLoadStoreLaneOpLoad8Lane   Op = 0 // v128.load8_lane
	SIMDLoadStoreLaneOpLoad16Lane  Op = 1 // v128.load16_lane
	SIMDLoadStoreLaneOpLoad32Lane  Op = 2 // v128.load32_lane
	SIMDLoadStoreLaneOpLoad64Lane  Op = 3 // v128.load64_lane
	SIMDLoadStoreLaneOpStore8Lane  Op = 4 // v128.store8_lane
	SIMDLoadStoreLaneOpStore16Lane Op = 5 // v128.store16_lane
	SIMDLoadStoreLaneOpStore32Lane Op = 6 // v128.store32_lane
	SIMDLoadStoreLaneOpStore64Lane Op = 7 // v128.store64_lane
)

// ---------------------------------------------------------------------------
// SIMDTernaryOp
// ---------------------------------------------------------------------------

const (
	SIMDTernaryOpBitselect                    Op = 0  // v128.bitselect
	SIMDTernaryOpRelaxedMaddVecF16x8          Op = 1  // f16x8.relaxed_madd
	SIMDTernaryOpRelaxedNmaddVecF16x8         Op = 2  // f16x8.relaxed_nmadd
	SIMDTernaryOpRelaxedMaddF32x4             Op = 3  // f32x4.relaxed_madd
	SIMDTernaryOpRelaxedNmaddF32x4            Op = 4  // f32x4.relaxed_nmadd
	SIMDTernaryOpRelaxedMaddF64x2             Op = 5  // f64x2.relaxed_madd
	SIMDTernaryOpRelaxedNmaddF64x2            Op = 6  // f64x2.relaxed_nmadd
	SIMDTernaryOpRelaxedLaneselectI8x16       Op = 7  // i8x16.relaxed_laneselect
	SIMDTernaryOpRelaxedLaneselectI16x8       Op = 8  // i16x8.relaxed_laneselect
	SIMDTernaryOpRelaxedLaneselectI32x4       Op = 9  // i32x4.relaxed_laneselect
	SIMDTernaryOpRelaxedLaneselectI64x2       Op = 10 // i64x2.relaxed_laneselect
	SIMDTernaryOpRelaxedDotI8x16I7x16AddToI32x4 Op = 11 // i32x4.relaxed_dot_i8x16_i7x16_add_s
)

// ---------------------------------------------------------------------------
// RefAsOp
// ---------------------------------------------------------------------------

const (
	RefAsOpNonNull          Op = 0 // ref.as_non_null
	RefAsOpExternInternalize Op = 1 // any.convert_extern
	RefAsOpExternExternalize Op = 2 // extern.convert_any
)

// ---------------------------------------------------------------------------
// BrOnOp
// ---------------------------------------------------------------------------

const (
	BrOnOpNull     Op = 0 // br_on_null
	BrOnOpNonNull  Op = 1 // br_on_non_null
	BrOnOpCast     Op = 2 // br_on_cast
	BrOnOpCastFail Op = 3 // br_on_cast_fail
)

// ---------------------------------------------------------------------------
// StringNewOp
// ---------------------------------------------------------------------------

const (
	StringNewOpLossyUTF8Array Op = 0 // string.new_wtf8_array replace
	StringNewOpWTF16Array     Op = 1 // string.new_wtf16_array
	StringNewOpFromCodePoint  Op = 2 // string.from_code_point
)

// ---------------------------------------------------------------------------
// StringMeasureOp
// ---------------------------------------------------------------------------

const (
	StringMeasureOpUTF8  Op = 0 // string.measure_wtf8 utf8
	StringMeasureOpWTF16 Op = 1 // string.measure_wtf16
)

// ---------------------------------------------------------------------------
// StringEncodeOp
// ---------------------------------------------------------------------------

const (
	StringEncodeOpLossyUTF8Array Op = 0 // string.encode_lossy_utf8_array utf8
	StringEncodeOpWTF16Array     Op = 1 // string.encode_wtf16_array
)

// ---------------------------------------------------------------------------
// StringEqOp
// ---------------------------------------------------------------------------

const (
	StringEqOpEqual   Op = 0 // string.eq
	StringEqOpCompare Op = 1 // string.compare
)

// ---------------------------------------------------------------------------
// ExpressionRunnerFlags constants
// ---------------------------------------------------------------------------

const (
	ExpressionRunnerFlagsDefault             ExpressionRunnerFlags = 0 // _ExpressionRunnerFlagsDefault
	ExpressionRunnerFlagsPreserveSideeffects ExpressionRunnerFlags = 1 // _ExpressionRunnerFlagsPreserveSideeffects
)

// ---------------------------------------------------------------------------
// SideEffects constants
// ---------------------------------------------------------------------------

const (
	SideEffectsNone              SideEffects = 0     // _BinaryenSideEffectNone
	SideEffectsBranches          SideEffects = 1     // _BinaryenSideEffectBranches
	SideEffectsCalls             SideEffects = 2     // _BinaryenSideEffectCalls
	SideEffectsReadsLocal        SideEffects = 4     // _BinaryenSideEffectReadsLocal
	SideEffectsWritesLocal       SideEffects = 8     // _BinaryenSideEffectWritesLocal
	SideEffectsReadsGlobal       SideEffects = 16    // _BinaryenSideEffectReadsGlobal
	SideEffectsWritesGlobal      SideEffects = 32    // _BinaryenSideEffectWritesGlobal
	SideEffectsReadsMemory       SideEffects = 64    // _BinaryenSideEffectReadsMemory
	SideEffectsWritesMemory      SideEffects = 128   // _BinaryenSideEffectWritesMemory
	SideEffectsReadsTable        SideEffects = 256   // _BinaryenSideEffectReadsTable
	SideEffectsWritesTable       SideEffects = 512   // _BinaryenSideEffectWritesTable
	SideEffectsImplicitTrap      SideEffects = 1024  // _BinaryenSideEffectImplicitTrap
	SideEffectsIsAtomic          SideEffects = 2048  // _BinaryenSideEffectIsAtomic
	SideEffectsThrows            SideEffects = 4096  // _BinaryenSideEffectThrows
	SideEffectsDanglingPop       SideEffects = 8192  // _BinaryenSideEffectDanglingPop
	SideEffectsTrapsNeverHappen  SideEffects = 16384 // _BinaryenSideEffectTrapsNeverHappen
	SideEffectsAny               SideEffects = 32767 // _BinaryenSideEffectAny
)

// ---------------------------------------------------------------------------
// MemorySegment is declared in module.go.
