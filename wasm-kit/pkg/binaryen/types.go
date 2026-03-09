// Ported from: src/glue/binaryen.d.ts (type system section)
package binaryen

/*
#include "binaryen-c.h"
*/
import "C"
import "unsafe"

// ---------------------------------------------------------------------------
// Core value types
// ---------------------------------------------------------------------------

func TypeNone() Type        { return Type(C.BinaryenTypeNone()) }
func TypeInt32() Type       { return Type(C.BinaryenTypeInt32()) }
func TypeInt64() Type       { return Type(C.BinaryenTypeInt64()) }
func TypeFloat32() Type     { return Type(C.BinaryenTypeFloat32()) }
func TypeFloat64() Type     { return Type(C.BinaryenTypeFloat64()) }
func TypeVec128() Type      { return Type(C.BinaryenTypeVec128()) }
func TypeFuncref() Type     { return Type(C.BinaryenTypeFuncref()) }
func TypeExternref() Type   { return Type(C.BinaryenTypeExternref()) }
func TypeAnyref() Type      { return Type(C.BinaryenTypeAnyref()) }
func TypeEqref() Type       { return Type(C.BinaryenTypeEqref()) }
func TypeI31ref() Type      { return Type(C.BinaryenTypeI31ref()) }
func TypeStructref() Type   { return Type(C.BinaryenTypeStructref()) }
func TypeArrayref() Type    { return Type(C.BinaryenTypeArrayref()) }
func TypeStringref() Type   { return Type(C.BinaryenTypeStringref()) }
func TypeNullref() Type     { return Type(C.BinaryenTypeNullref()) }
func TypeNullExternref() Type { return Type(C.BinaryenTypeNullExternref()) }
func TypeNullFuncref() Type { return Type(C.BinaryenTypeNullFuncref()) }
func TypeUnreachable() Type { return Type(C.BinaryenTypeUnreachable()) }
func TypeAuto() Type        { return Type(C.BinaryenTypeAuto()) }

// TypeCreate creates a tuple type from multiple value types.
func TypeCreate(types []Type) Type {
	if len(types) == 0 {
		return TypeNone()
	}
	return Type(C.BinaryenTypeCreate(
		(*C.BinaryenType)(unsafe.Pointer(&types[0])),
		C.BinaryenIndex(len(types)),
	))
}

// TypeArity returns the number of elements in a tuple type (1 for non-tuples).
func TypeArity(t Type) uint32 {
	return uint32(C.BinaryenTypeArity(C.BinaryenType(t)))
}

// TypeExpand expands a tuple type into its component types.
func TypeExpand(t Type) []Type {
	n := TypeArity(t)
	if n == 0 {
		return nil
	}
	buf := make([]Type, n)
	C.BinaryenTypeExpand(C.BinaryenType(t), (*C.BinaryenType)(unsafe.Pointer(&buf[0])))
	return buf
}

// TypeGetHeapType extracts the heap type from a reference type.
func TypeGetHeapType(t Type) HeapType {
	return HeapType(C.BinaryenTypeGetHeapType(C.BinaryenType(t)))
}

// TypeIsNullable returns whether a reference type is nullable.
func TypeIsNullable(t Type) bool {
	return goBool(C.BinaryenTypeIsNullable(C.BinaryenType(t)))
}

// TypeFromHeapType creates a reference type from a heap type.
func TypeFromHeapType(ht HeapType, nullable bool) Type {
	return Type(C.BinaryenTypeFromHeapType(C.BinaryenHeapType(ht), cBool(nullable)))
}

// ---------------------------------------------------------------------------
// Packed types
// ---------------------------------------------------------------------------

func PackedTypeNotPacked() PackedType { return PackedType(C.BinaryenPackedTypeNotPacked()) }
func PackedTypeInt8() PackedType      { return PackedType(C.BinaryenPackedTypeInt8()) }
func PackedTypeInt16() PackedType     { return PackedType(C.BinaryenPackedTypeInt16()) }

// ---------------------------------------------------------------------------
// Heap types
// ---------------------------------------------------------------------------

func HeapTypeExt() HeapType    { return HeapType(C.BinaryenHeapTypeExt()) }
func HeapTypeFunc() HeapType   { return HeapType(C.BinaryenHeapTypeFunc()) }
func HeapTypeAny() HeapType    { return HeapType(C.BinaryenHeapTypeAny()) }
func HeapTypeEq() HeapType     { return HeapType(C.BinaryenHeapTypeEq()) }
func HeapTypeI31() HeapType    { return HeapType(C.BinaryenHeapTypeI31()) }
func HeapTypeStruct() HeapType { return HeapType(C.BinaryenHeapTypeStruct()) }
func HeapTypeArray() HeapType  { return HeapType(C.BinaryenHeapTypeArray()) }
func HeapTypeString() HeapType { return HeapType(C.BinaryenHeapTypeString()) }
func HeapTypeNone() HeapType   { return HeapType(C.BinaryenHeapTypeNone()) }
func HeapTypeNoext() HeapType  { return HeapType(C.BinaryenHeapTypeNoext()) }
func HeapTypeNofunc() HeapType { return HeapType(C.BinaryenHeapTypeNofunc()) }

// HeapType queries
func HeapTypeIsBasic(ht HeapType) bool     { return goBool(C.BinaryenHeapTypeIsBasic(C.BinaryenHeapType(ht))) }
func HeapTypeIsSignature(ht HeapType) bool { return goBool(C.BinaryenHeapTypeIsSignature(C.BinaryenHeapType(ht))) }
func HeapTypeIsStruct(ht HeapType) bool    { return goBool(C.BinaryenHeapTypeIsStruct(C.BinaryenHeapType(ht))) }
func HeapTypeIsArray(ht HeapType) bool     { return goBool(C.BinaryenHeapTypeIsArray(C.BinaryenHeapType(ht))) }
func HeapTypeIsBottom(ht HeapType) bool    { return goBool(C.BinaryenHeapTypeIsBottom(C.BinaryenHeapType(ht))) }
func HeapTypeGetBottom(ht HeapType) HeapType {
	return HeapType(C.BinaryenHeapTypeGetBottom(C.BinaryenHeapType(ht)))
}
func HeapTypeIsSubType(left, right HeapType) bool {
	return goBool(C.BinaryenHeapTypeIsSubType(C.BinaryenHeapType(left), C.BinaryenHeapType(right)))
}

// Struct type queries
func StructTypeGetNumFields(ht HeapType) Index {
	return Index(C.BinaryenStructTypeGetNumFields(C.BinaryenHeapType(ht)))
}
func StructTypeGetFieldType(ht HeapType, index Index) Type {
	return Type(C.BinaryenStructTypeGetFieldType(C.BinaryenHeapType(ht), C.BinaryenIndex(index)))
}
func StructTypeGetFieldPackedType(ht HeapType, index Index) PackedType {
	return PackedType(C.BinaryenStructTypeGetFieldPackedType(C.BinaryenHeapType(ht), C.BinaryenIndex(index)))
}
func StructTypeIsFieldMutable(ht HeapType, index Index) bool {
	return goBool(C.BinaryenStructTypeIsFieldMutable(C.BinaryenHeapType(ht), C.BinaryenIndex(index)))
}

// Array type queries
func ArrayTypeGetElementType(ht HeapType) Type {
	return Type(C.BinaryenArrayTypeGetElementType(C.BinaryenHeapType(ht)))
}
func ArrayTypeGetElementPackedType(ht HeapType) PackedType {
	return PackedType(C.BinaryenArrayTypeGetElementPackedType(C.BinaryenHeapType(ht)))
}
func ArrayTypeIsElementMutable(ht HeapType) bool {
	return goBool(C.BinaryenArrayTypeIsElementMutable(C.BinaryenHeapType(ht)))
}

// Signature type queries
func SignatureTypeGetParams(ht HeapType) Type {
	return Type(C.BinaryenSignatureTypeGetParams(C.BinaryenHeapType(ht)))
}
func SignatureTypeGetResults(ht HeapType) Type {
	return Type(C.BinaryenSignatureTypeGetResults(C.BinaryenHeapType(ht)))
}

// ---------------------------------------------------------------------------
// Expression IDs
// ---------------------------------------------------------------------------

func InvalidId() ExpressionID { return ExpressionID(C.BinaryenInvalidId()) }
func BlockId() ExpressionID   { return ExpressionID(C.BinaryenBlockId()) }
func IfId() ExpressionID      { return ExpressionID(C.BinaryenIfId()) }
func LoopId() ExpressionID    { return ExpressionID(C.BinaryenLoopId()) }
func BreakId() ExpressionID   { return ExpressionID(C.BinaryenBreakId()) }
func SwitchId() ExpressionID  { return ExpressionID(C.BinaryenSwitchId()) }
func CallId() ExpressionID    { return ExpressionID(C.BinaryenCallId()) }
func CallIndirectId() ExpressionID { return ExpressionID(C.BinaryenCallIndirectId()) }
func LocalGetId() ExpressionID { return ExpressionID(C.BinaryenLocalGetId()) }
func LocalSetId() ExpressionID { return ExpressionID(C.BinaryenLocalSetId()) }
func GlobalGetId() ExpressionID { return ExpressionID(C.BinaryenGlobalGetId()) }
func GlobalSetId() ExpressionID { return ExpressionID(C.BinaryenGlobalSetId()) }
func LoadId() ExpressionID    { return ExpressionID(C.BinaryenLoadId()) }
func StoreId() ExpressionID   { return ExpressionID(C.BinaryenStoreId()) }
func ConstId() ExpressionID   { return ExpressionID(C.BinaryenConstId()) }
func UnaryId() ExpressionID   { return ExpressionID(C.BinaryenUnaryId()) }
func BinaryId() ExpressionID  { return ExpressionID(C.BinaryenBinaryId()) }
func SelectId() ExpressionID  { return ExpressionID(C.BinaryenSelectId()) }
func DropId() ExpressionID    { return ExpressionID(C.BinaryenDropId()) }
func ReturnId() ExpressionID  { return ExpressionID(C.BinaryenReturnId()) }
func MemorySizeId() ExpressionID { return ExpressionID(C.BinaryenMemorySizeId()) }
func MemoryGrowId() ExpressionID { return ExpressionID(C.BinaryenMemoryGrowId()) }
func NopId() ExpressionID     { return ExpressionID(C.BinaryenNopId()) }
func UnreachableId() ExpressionID { return ExpressionID(C.BinaryenUnreachableId()) }
func AtomicRMWId() ExpressionID { return ExpressionID(C.BinaryenAtomicRMWId()) }
func AtomicCmpxchgId() ExpressionID { return ExpressionID(C.BinaryenAtomicCmpxchgId()) }
func AtomicWaitId() ExpressionID { return ExpressionID(C.BinaryenAtomicWaitId()) }
func AtomicNotifyId() ExpressionID { return ExpressionID(C.BinaryenAtomicNotifyId()) }
func AtomicFenceId() ExpressionID { return ExpressionID(C.BinaryenAtomicFenceId()) }
func SIMDExtractId() ExpressionID { return ExpressionID(C.BinaryenSIMDExtractId()) }
func SIMDReplaceId() ExpressionID { return ExpressionID(C.BinaryenSIMDReplaceId()) }
func SIMDShuffleId() ExpressionID { return ExpressionID(C.BinaryenSIMDShuffleId()) }
func SIMDTernaryId() ExpressionID { return ExpressionID(C.BinaryenSIMDTernaryId()) }
func SIMDShiftId() ExpressionID { return ExpressionID(C.BinaryenSIMDShiftId()) }
func SIMDLoadId() ExpressionID { return ExpressionID(C.BinaryenSIMDLoadId()) }
func SIMDLoadStoreLaneId() ExpressionID { return ExpressionID(C.BinaryenSIMDLoadStoreLaneId()) }
func MemoryInitId() ExpressionID { return ExpressionID(C.BinaryenMemoryInitId()) }
func DataDropId() ExpressionID { return ExpressionID(C.BinaryenDataDropId()) }
func MemoryCopyId() ExpressionID { return ExpressionID(C.BinaryenMemoryCopyId()) }
func MemoryFillId() ExpressionID { return ExpressionID(C.BinaryenMemoryFillId()) }
func TryId() ExpressionID     { return ExpressionID(C.BinaryenTryId()) }
func ThrowId() ExpressionID   { return ExpressionID(C.BinaryenThrowId()) }
func RethrowId() ExpressionID { return ExpressionID(C.BinaryenRethrowId()) }
func TupleMakeId() ExpressionID { return ExpressionID(C.BinaryenTupleMakeId()) }
func TupleExtractId() ExpressionID { return ExpressionID(C.BinaryenTupleExtractId()) }
func PopId() ExpressionID     { return ExpressionID(C.BinaryenPopId()) }
func RefNullId() ExpressionID { return ExpressionID(C.BinaryenRefNullId()) }
func RefIsNullId() ExpressionID { return ExpressionID(C.BinaryenRefIsNullId()) }
func RefFuncId() ExpressionID { return ExpressionID(C.BinaryenRefFuncId()) }
func RefEqId() ExpressionID   { return ExpressionID(C.BinaryenRefEqId()) }
func TableGetId() ExpressionID { return ExpressionID(C.BinaryenTableGetId()) }
func TableSetId() ExpressionID { return ExpressionID(C.BinaryenTableSetId()) }
func TableSizeId() ExpressionID { return ExpressionID(C.BinaryenTableSizeId()) }
func TableGrowId() ExpressionID { return ExpressionID(C.BinaryenTableGrowId()) }
func TableFillId() ExpressionID { return ExpressionID(C.BinaryenTableFillId()) }
func TableCopyId() ExpressionID { return ExpressionID(C.BinaryenTableCopyId()) }
func TableInitId() ExpressionID { return ExpressionID(C.BinaryenTableInitId()) }

// Exception handling expression IDs
func TryTableId() ExpressionID  { return ExpressionID(C.BinaryenTryTableId()) }
func ThrowRefId() ExpressionID  { return ExpressionID(C.BinaryenThrowRefId()) }

// GC expression IDs
func StructNewId() ExpressionID { return ExpressionID(C.BinaryenStructNewId()) }
func StructGetId() ExpressionID { return ExpressionID(C.BinaryenStructGetId()) }
func StructSetId() ExpressionID { return ExpressionID(C.BinaryenStructSetId()) }
func StructRMWId() ExpressionID { return ExpressionID(C.BinaryenStructRMWId()) }
func StructCmpxchgId() ExpressionID { return ExpressionID(C.BinaryenStructCmpxchgId()) }
func ArrayNewId() ExpressionID { return ExpressionID(C.BinaryenArrayNewId()) }
func ArrayNewDataId() ExpressionID { return ExpressionID(C.BinaryenArrayNewDataId()) }
func ArrayNewElemId() ExpressionID { return ExpressionID(C.BinaryenArrayNewElemId()) }
func ArrayNewFixedId() ExpressionID { return ExpressionID(C.BinaryenArrayNewFixedId()) }
func ArrayGetId() ExpressionID { return ExpressionID(C.BinaryenArrayGetId()) }
func ArraySetId() ExpressionID { return ExpressionID(C.BinaryenArraySetId()) }
func ArrayLenId() ExpressionID { return ExpressionID(C.BinaryenArrayLenId()) }
func ArrayCopyId() ExpressionID { return ExpressionID(C.BinaryenArrayCopyId()) }
func ArrayFillId() ExpressionID { return ExpressionID(C.BinaryenArrayFillId()) }
func ArrayInitDataId() ExpressionID { return ExpressionID(C.BinaryenArrayInitDataId()) }
func ArrayInitElemId() ExpressionID { return ExpressionID(C.BinaryenArrayInitElemId()) }
func RefAsId() ExpressionID   { return ExpressionID(C.BinaryenRefAsId()) }
func RefCastId() ExpressionID { return ExpressionID(C.BinaryenRefCastId()) }
func BrOnId() ExpressionID    { return ExpressionID(C.BinaryenBrOnId()) }
func RefTestId() ExpressionID { return ExpressionID(C.BinaryenRefTestId()) }
func StringNewId() ExpressionID { return ExpressionID(C.BinaryenStringNewId()) }
func StringConstId() ExpressionID { return ExpressionID(C.BinaryenStringConstId()) }
func StringMeasureId() ExpressionID { return ExpressionID(C.BinaryenStringMeasureId()) }
func StringEncodeId() ExpressionID { return ExpressionID(C.BinaryenStringEncodeId()) }
func StringConcatId() ExpressionID { return ExpressionID(C.BinaryenStringConcatId()) }
func StringEqId() ExpressionID { return ExpressionID(C.BinaryenStringEqId()) }
func StringWTF16GetId() ExpressionID { return ExpressionID(C.BinaryenStringWTF16GetId()) }
func StringSliceWTFId() ExpressionID { return ExpressionID(C.BinaryenStringSliceWTFId()) }
func RefI31Id() ExpressionID  { return ExpressionID(C.BinaryenRefI31Id()) }
func I31GetId() ExpressionID  { return ExpressionID(C.BinaryenI31GetId()) }
func CallRefId() ExpressionID { return ExpressionID(C.BinaryenCallRefId()) }

// Continuation expression IDs (stack switching proposal)
func ContNewId() ExpressionID     { return ExpressionID(C.BinaryenContNewId()) }
func ContBindId() ExpressionID    { return ExpressionID(C.BinaryenContBindId()) }
func SuspendId() ExpressionID     { return ExpressionID(C.BinaryenSuspendId()) }
func ResumeId() ExpressionID      { return ExpressionID(C.BinaryenResumeId()) }
func ResumeThrowId() ExpressionID { return ExpressionID(C.BinaryenResumeThrowId()) }
func StackSwitchId() ExpressionID { return ExpressionID(C.BinaryenStackSwitchId()) }

// ---------------------------------------------------------------------------
// External kinds
// ---------------------------------------------------------------------------

func ExternalFunction() ExternalKind { return ExternalKind(C.BinaryenExternalFunction()) }
func ExternalTable() ExternalKind    { return ExternalKind(C.BinaryenExternalTable()) }
func ExternalMemory() ExternalKind   { return ExternalKind(C.BinaryenExternalMemory()) }
func ExternalGlobal() ExternalKind   { return ExternalKind(C.BinaryenExternalGlobal()) }
func ExternalTag() ExternalKind      { return ExternalKind(C.BinaryenExternalTag()) }

// ---------------------------------------------------------------------------
// Feature flags
// ---------------------------------------------------------------------------

func FeatureMVP() Features              { return Features(C.BinaryenFeatureMVP()) }
func FeatureAtomics() Features           { return Features(C.BinaryenFeatureAtomics()) }
func FeatureMutableGlobals() Features    { return Features(C.BinaryenFeatureMutableGlobals()) }
func FeatureNontrappingFPToInt() Features { return Features(C.BinaryenFeatureNontrappingFPToInt()) }
func FeatureSIMD128() Features           { return Features(C.BinaryenFeatureSIMD128()) }
func FeatureBulkMemory() Features        { return Features(C.BinaryenFeatureBulkMemory()) }
func FeatureSignExt() Features           { return Features(C.BinaryenFeatureSignExt()) }
func FeatureExceptionHandling() Features { return Features(C.BinaryenFeatureExceptionHandling()) }
func FeatureTailCall() Features          { return Features(C.BinaryenFeatureTailCall()) }
func FeatureReferenceTypes() Features    { return Features(C.BinaryenFeatureReferenceTypes()) }
func FeatureMultivalue() Features        { return Features(C.BinaryenFeatureMultivalue()) }
func FeatureGC() Features                { return Features(C.BinaryenFeatureGC()) }
func FeatureMemory64() Features          { return Features(C.BinaryenFeatureMemory64()) }
func FeatureRelaxedSIMD() Features       { return Features(C.BinaryenFeatureRelaxedSIMD()) }
func FeatureExtendedConst() Features     { return Features(C.BinaryenFeatureExtendedConst()) }
func FeatureStrings() Features           { return Features(C.BinaryenFeatureStrings()) }
func FeatureMultiMemory() Features       { return Features(C.BinaryenFeatureMultiMemory()) }
func FeatureAll() Features               { return Features(C.BinaryenFeatureAll()) }
