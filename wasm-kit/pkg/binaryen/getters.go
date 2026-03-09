package binaryen

/*
#include "binaryen-c.h"
*/
import "C"
import "unsafe"

// ---------------------------------------------------------------------------
// Expression (generic)
// ---------------------------------------------------------------------------

// ExpressionGetId returns the kind (expression ID) of the given expression.
func ExpressionGetId(expr ExpressionRef) ExpressionID {
	return ExpressionID(C.BinaryenExpressionGetId((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ExpressionGetType returns the Binaryen type of the given expression.
func ExpressionGetType(expr ExpressionRef) Type {
	return Type(C.BinaryenExpressionGetType((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// Block
// ---------------------------------------------------------------------------

func BlockGetName(expr ExpressionRef) string {
	return goString(C.BinaryenBlockGetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func BlockGetNumChildren(expr ExpressionRef) Index {
	return Index(C.BinaryenBlockGetNumChildren((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func BlockGetChildAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBlockGetChildAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

// ---------------------------------------------------------------------------
// If
// ---------------------------------------------------------------------------

func IfGetCondition(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenIfGetCondition((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func IfGetIfTrue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenIfGetIfTrue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func IfGetIfFalse(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenIfGetIfFalse((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Loop
// ---------------------------------------------------------------------------

func LoopGetName(expr ExpressionRef) string {
	return goString(C.BinaryenLoopGetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LoopGetBody(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLoopGetBody((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Break
// ---------------------------------------------------------------------------

func BreakGetName(expr ExpressionRef) string {
	return goString(C.BinaryenBreakGetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func BreakGetCondition(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBreakGetCondition((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func BreakGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBreakGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Switch
// ---------------------------------------------------------------------------

func SwitchGetNumNames(expr ExpressionRef) Index {
	return Index(C.BinaryenSwitchGetNumNames((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SwitchGetNameAt(expr ExpressionRef, index Index) string {
	return goString(C.BinaryenSwitchGetNameAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index)))
}

func SwitchGetDefaultName(expr ExpressionRef) string {
	return goString(C.BinaryenSwitchGetDefaultName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SwitchGetCondition(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSwitchGetCondition((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SwitchGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSwitchGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Call
// ---------------------------------------------------------------------------

func CallGetTarget(expr ExpressionRef) string {
	return goString(C.BinaryenCallGetTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func CallGetNumOperands(expr ExpressionRef) Index {
	return Index(C.BinaryenCallGetNumOperands((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func CallGetOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenCallGetOperandAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

func CallIsReturn(expr ExpressionRef) bool {
	return goBool(C.BinaryenCallIsReturn((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// CallIndirect
// ---------------------------------------------------------------------------

func CallIndirectGetTarget(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenCallIndirectGetTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func CallIndirectGetTable(expr ExpressionRef) string {
	return goString(C.BinaryenCallIndirectGetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func CallIndirectGetNumOperands(expr ExpressionRef) Index {
	return Index(C.BinaryenCallIndirectGetNumOperands((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func CallIndirectGetOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenCallIndirectGetOperandAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

func CallIndirectIsReturn(expr ExpressionRef) bool {
	return goBool(C.BinaryenCallIndirectIsReturn((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func CallIndirectGetParams(expr ExpressionRef) Type {
	return Type(C.BinaryenCallIndirectGetParams((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func CallIndirectGetResults(expr ExpressionRef) Type {
	return Type(C.BinaryenCallIndirectGetResults((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// LocalGet
// ---------------------------------------------------------------------------

func LocalGetGetIndex(expr ExpressionRef) Index {
	return Index(C.BinaryenLocalGetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// LocalSet
// ---------------------------------------------------------------------------

func LocalSetIsTee(expr ExpressionRef) bool {
	return goBool(C.BinaryenLocalSetIsTee((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LocalSetGetIndex(expr ExpressionRef) Index {
	return Index(C.BinaryenLocalSetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LocalSetGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLocalSetGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// GlobalGet
// ---------------------------------------------------------------------------

func GlobalGetGetName(expr ExpressionRef) string {
	return goString(C.BinaryenGlobalGetGetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// GlobalSet
// ---------------------------------------------------------------------------

func GlobalSetGetName(expr ExpressionRef) string {
	return goString(C.BinaryenGlobalSetGetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func GlobalSetGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenGlobalSetGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// TableGet
// ---------------------------------------------------------------------------

func TableGetGetTable(expr ExpressionRef) string {
	return goString(C.BinaryenTableGetGetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TableGetGetIndex(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableGetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// TableSet
// ---------------------------------------------------------------------------

func TableSetGetTable(expr ExpressionRef) string {
	return goString(C.BinaryenTableSetGetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TableSetGetIndex(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableSetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func TableSetGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableSetGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// TableSize
// ---------------------------------------------------------------------------

func TableSizeGetTable(expr ExpressionRef) string {
	return goString(C.BinaryenTableSizeGetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// TableGrow
// ---------------------------------------------------------------------------

func TableGrowGetTable(expr ExpressionRef) string {
	return goString(C.BinaryenTableGrowGetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TableGrowGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableGrowGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func TableGrowGetDelta(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableGrowGetDelta((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// MemoryGrow
// ---------------------------------------------------------------------------

func MemoryGrowGetDelta(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryGrowGetDelta((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func LoadIsAtomic(expr ExpressionRef) bool {
	return goBool(C.BinaryenLoadIsAtomic((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LoadIsSigned(expr ExpressionRef) bool {
	return goBool(C.BinaryenLoadIsSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LoadGetOffset(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenLoadGetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LoadGetBytes(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenLoadGetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LoadGetAlign(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenLoadGetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func LoadGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLoadGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

func StoreIsAtomic(expr ExpressionRef) bool {
	return goBool(C.BinaryenStoreIsAtomic((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StoreGetBytes(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenStoreGetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StoreGetOffset(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenStoreGetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StoreGetAlign(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenStoreGetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StoreGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStoreGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StoreGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStoreGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StoreGetValueType(expr ExpressionRef) Type {
	return Type(C.BinaryenStoreGetValueType((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// Const
// ---------------------------------------------------------------------------

func ConstGetValueI32(expr ExpressionRef) int32 {
	return int32(C.BinaryenConstGetValueI32((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ConstGetValueI64(expr ExpressionRef) int64 {
	return int64(C.BinaryenConstGetValueI64((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ConstGetValueI64Low(expr ExpressionRef) int32 {
	return int32(C.BinaryenConstGetValueI64Low((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ConstGetValueI64High(expr ExpressionRef) int32 {
	return int32(C.BinaryenConstGetValueI64High((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ConstGetValueF32(expr ExpressionRef) float32 {
	return float32(C.BinaryenConstGetValueF32((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ConstGetValueF64(expr ExpressionRef) float64 {
	return float64(C.BinaryenConstGetValueF64((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ConstGetValueV128 reads the 128-bit vector value into the provided 16-byte buffer.
func ConstGetValueV128(expr ExpressionRef, out *[16]byte) {
	C.BinaryenConstGetValueV128((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), (*C.uint8_t)(unsafe.Pointer(&out[0])))
}

// ---------------------------------------------------------------------------
// Unary
// ---------------------------------------------------------------------------

func UnaryGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenUnaryGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func UnaryGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenUnaryGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Binary
// ---------------------------------------------------------------------------

func BinaryGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenBinaryGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func BinaryGetLeft(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBinaryGetLeft((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func BinaryGetRight(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBinaryGetRight((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Select
// ---------------------------------------------------------------------------

func SelectGetIfTrue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSelectGetIfTrue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SelectGetIfFalse(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSelectGetIfFalse((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SelectGetCondition(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSelectGetCondition((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Drop
// ---------------------------------------------------------------------------

func DropGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenDropGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Return
// ---------------------------------------------------------------------------

func ReturnGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenReturnGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// AtomicRMW
// ---------------------------------------------------------------------------

func AtomicRMWGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenAtomicRMWGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func AtomicRMWGetBytes(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenAtomicRMWGetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func AtomicRMWGetOffset(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenAtomicRMWGetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func AtomicRMWGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicRMWGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func AtomicRMWGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicRMWGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// AtomicCmpxchg
// ---------------------------------------------------------------------------

func AtomicCmpxchgGetBytes(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenAtomicCmpxchgGetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func AtomicCmpxchgGetOffset(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenAtomicCmpxchgGetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func AtomicCmpxchgGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicCmpxchgGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func AtomicCmpxchgGetExpected(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicCmpxchgGetExpected((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func AtomicCmpxchgGetReplacement(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicCmpxchgGetReplacement((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// AtomicWait
// ---------------------------------------------------------------------------

func AtomicWaitGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicWaitGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func AtomicWaitGetExpected(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicWaitGetExpected((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func AtomicWaitGetTimeout(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicWaitGetTimeout((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func AtomicWaitGetExpectedType(expr ExpressionRef) Type {
	return Type(C.BinaryenAtomicWaitGetExpectedType((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// AtomicNotify
// ---------------------------------------------------------------------------

func AtomicNotifyGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicNotifyGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func AtomicNotifyGetNotifyCount(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicNotifyGetNotifyCount((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// AtomicFence
// ---------------------------------------------------------------------------

func AtomicFenceGetOrder(expr ExpressionRef) uint8 {
	return uint8(C.BinaryenAtomicFenceGetOrder((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// SIMDExtract
// ---------------------------------------------------------------------------

func SIMDExtractGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenSIMDExtractGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDExtractGetVec(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDExtractGetVec((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDExtractGetIndex(expr ExpressionRef) uint8 {
	return uint8(C.BinaryenSIMDExtractGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// SIMDReplace
// ---------------------------------------------------------------------------

func SIMDReplaceGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenSIMDReplaceGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDReplaceGetVec(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDReplaceGetVec((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDReplaceGetIndex(expr ExpressionRef) uint8 {
	return uint8(C.BinaryenSIMDReplaceGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDReplaceGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDReplaceGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// SIMDShuffle
// ---------------------------------------------------------------------------

func SIMDShuffleGetLeft(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDShuffleGetLeft((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDShuffleGetRight(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDShuffleGetRight((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// SIMDShuffleGetMask reads the 128-bit shuffle mask into the provided 16-byte buffer.
func SIMDShuffleGetMask(expr ExpressionRef, mask *[16]byte) {
	C.BinaryenSIMDShuffleGetMask((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), (*C.uint8_t)(unsafe.Pointer(&mask[0])))
}

// ---------------------------------------------------------------------------
// SIMDTernary
// ---------------------------------------------------------------------------

func SIMDTernaryGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenSIMDTernaryGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDTernaryGetA(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDTernaryGetA((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDTernaryGetB(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDTernaryGetB((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDTernaryGetC(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDTernaryGetC((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// SIMDShift
// ---------------------------------------------------------------------------

func SIMDShiftGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenSIMDShiftGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDShiftGetVec(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDShiftGetVec((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDShiftGetShift(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDShiftGetShift((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// SIMDLoad
// ---------------------------------------------------------------------------

func SIMDLoadGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenSIMDLoadGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDLoadGetOffset(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenSIMDLoadGetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDLoadGetAlign(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenSIMDLoadGetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDLoadGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDLoadGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// SIMDLoadStoreLane
// ---------------------------------------------------------------------------

func SIMDLoadStoreLaneGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenSIMDLoadStoreLaneGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDLoadStoreLaneGetOffset(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenSIMDLoadStoreLaneGetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDLoadStoreLaneGetAlign(expr ExpressionRef) uint32 {
	return uint32(C.BinaryenSIMDLoadStoreLaneGetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDLoadStoreLaneGetIndex(expr ExpressionRef) uint8 {
	return uint8(C.BinaryenSIMDLoadStoreLaneGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func SIMDLoadStoreLaneGetPtr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDLoadStoreLaneGetPtr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDLoadStoreLaneGetVec(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDLoadStoreLaneGetVec((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func SIMDLoadStoreLaneIsStore(expr ExpressionRef) bool {
	return goBool(C.BinaryenSIMDLoadStoreLaneIsStore((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// MemoryInit
// ---------------------------------------------------------------------------

func MemoryInitGetSegment(expr ExpressionRef) string {
	return goString(C.BinaryenMemoryInitGetSegment((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func MemoryInitGetDest(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryInitGetDest((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func MemoryInitGetOffset(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryInitGetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func MemoryInitGetSize(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryInitGetSize((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// DataDrop
// ---------------------------------------------------------------------------

func DataDropGetSegment(expr ExpressionRef) string {
	return goString(C.BinaryenDataDropGetSegment((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// MemoryCopy
// ---------------------------------------------------------------------------

func MemoryCopyGetDest(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryCopyGetDest((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func MemoryCopyGetSource(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryCopyGetSource((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func MemoryCopyGetSize(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryCopyGetSize((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// MemoryFill
// ---------------------------------------------------------------------------

func MemoryFillGetDest(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryFillGetDest((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func MemoryFillGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryFillGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func MemoryFillGetSize(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryFillGetSize((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// RefIsNull
// ---------------------------------------------------------------------------

func RefIsNullGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefIsNullGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// RefAs
// ---------------------------------------------------------------------------

func RefAsGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenRefAsGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func RefAsGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefAsGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// RefFunc
// ---------------------------------------------------------------------------

func RefFuncGetFunc(expr ExpressionRef) string {
	return goString(C.BinaryenRefFuncGetFunc((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// RefEq
// ---------------------------------------------------------------------------

func RefEqGetLeft(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefEqGetLeft((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func RefEqGetRight(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefEqGetRight((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// Try
// ---------------------------------------------------------------------------

func TryGetName(expr ExpressionRef) string {
	return goString(C.BinaryenTryGetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TryGetBody(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTryGetBody((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func TryGetNumCatchTags(expr ExpressionRef) Index {
	return Index(C.BinaryenTryGetNumCatchTags((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TryGetNumCatchBodies(expr ExpressionRef) Index {
	return Index(C.BinaryenTryGetNumCatchBodies((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TryGetCatchTagAt(expr ExpressionRef, index Index) string {
	return goString(C.BinaryenTryGetCatchTagAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index)))
}

func TryGetCatchBodyAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTryGetCatchBodyAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

func TryHasCatchAll(expr ExpressionRef) bool {
	return goBool(C.BinaryenTryHasCatchAll((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TryGetDelegateTarget(expr ExpressionRef) string {
	return goString(C.BinaryenTryGetDelegateTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TryIsDelegate(expr ExpressionRef) bool {
	return goBool(C.BinaryenTryIsDelegate((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// Throw
// ---------------------------------------------------------------------------

func ThrowGetTag(expr ExpressionRef) string {
	return goString(C.BinaryenThrowGetTag((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ThrowGetNumOperands(expr ExpressionRef) Index {
	return Index(C.BinaryenThrowGetNumOperands((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ThrowGetOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenThrowGetOperandAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

// ---------------------------------------------------------------------------
// Rethrow
// ---------------------------------------------------------------------------

func RethrowGetTarget(expr ExpressionRef) string {
	return goString(C.BinaryenRethrowGetTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// TupleMake
// ---------------------------------------------------------------------------

func TupleMakeGetNumOperands(expr ExpressionRef) Index {
	return Index(C.BinaryenTupleMakeGetNumOperands((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func TupleMakeGetOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTupleMakeGetOperandAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

// ---------------------------------------------------------------------------
// TupleExtract
// ---------------------------------------------------------------------------

func TupleExtractGetTuple(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTupleExtractGetTuple((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func TupleExtractGetIndex(expr ExpressionRef) Index {
	return Index(C.BinaryenTupleExtractGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// RefI31
// ---------------------------------------------------------------------------

func RefI31GetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefI31GetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// I31Get
// ---------------------------------------------------------------------------

func I31GetGetI31(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenI31GetGetI31((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func I31GetIsSigned(expr ExpressionRef) bool {
	return goBool(C.BinaryenI31GetIsSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// CallRef
// ---------------------------------------------------------------------------

func CallRefGetNumOperands(expr ExpressionRef) Index {
	return Index(C.BinaryenCallRefGetNumOperands((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func CallRefGetOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenCallRefGetOperandAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

func CallRefGetTarget(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenCallRefGetTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func CallRefIsReturn(expr ExpressionRef) bool {
	return goBool(C.BinaryenCallRefIsReturn((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// RefTest
// ---------------------------------------------------------------------------

func RefTestGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefTestGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func RefTestGetCastType(expr ExpressionRef) Type {
	return Type(C.BinaryenRefTestGetCastType((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// RefCast
// ---------------------------------------------------------------------------

func RefCastGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefCastGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// BrOn
// ---------------------------------------------------------------------------

func BrOnGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenBrOnGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func BrOnGetName(expr ExpressionRef) string {
	return goString(C.BinaryenBrOnGetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func BrOnGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBrOnGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func BrOnGetCastType(expr ExpressionRef) Type {
	return Type(C.BinaryenBrOnGetCastType((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// StructNew
// ---------------------------------------------------------------------------

func StructNewGetNumOperands(expr ExpressionRef) Index {
	return Index(C.BinaryenStructNewGetNumOperands((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StructNewGetOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStructNewGetOperandAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

// ---------------------------------------------------------------------------
// StructGet
// ---------------------------------------------------------------------------

func StructGetGetIndex(expr ExpressionRef) Index {
	return Index(C.BinaryenStructGetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StructGetGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStructGetGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StructGetIsSigned(expr ExpressionRef) bool {
	return goBool(C.BinaryenStructGetIsSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// StructSet
// ---------------------------------------------------------------------------

func StructSetGetIndex(expr ExpressionRef) Index {
	return Index(C.BinaryenStructSetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StructSetGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStructSetGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StructSetGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStructSetGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// ArrayNew
// ---------------------------------------------------------------------------

func ArrayNewGetInit(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayNewGetInit((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArrayNewGetSize(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayNewGetSize((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// ArrayNewFixed
// ---------------------------------------------------------------------------

func ArrayNewFixedGetNumValues(expr ExpressionRef) Index {
	return Index(C.BinaryenArrayNewFixedGetNumValues((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func ArrayNewFixedGetValueAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayNewFixedGetValueAt((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))))
}

// ---------------------------------------------------------------------------
// ArrayGet
// ---------------------------------------------------------------------------

func ArrayGetGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayGetGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArrayGetGetIndex(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayGetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArrayGetIsSigned(expr ExpressionRef) bool {
	return goBool(C.BinaryenArrayGetIsSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// ArraySet
// ---------------------------------------------------------------------------

func ArraySetGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArraySetGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArraySetGetIndex(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArraySetGetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArraySetGetValue(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArraySetGetValue((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// ArrayLen
// ---------------------------------------------------------------------------

func ArrayLenGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayLenGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// ArrayCopy
// ---------------------------------------------------------------------------

func ArrayCopyGetDestRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayCopyGetDestRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArrayCopyGetDestIndex(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayCopyGetDestIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArrayCopyGetSrcRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayCopyGetSrcRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArrayCopyGetSrcIndex(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayCopyGetSrcIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func ArrayCopyGetLength(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayCopyGetLength((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// StringNew
// ---------------------------------------------------------------------------

func StringNewGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenStringNewGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StringNewGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringNewGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringNewGetStart(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringNewGetStart((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringNewGetEnd(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringNewGetEnd((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// StringConst
// ---------------------------------------------------------------------------

func StringConstGetString(expr ExpressionRef) string {
	return goString(C.BinaryenStringConstGetString((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

// ---------------------------------------------------------------------------
// StringMeasure
// ---------------------------------------------------------------------------

func StringMeasureGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenStringMeasureGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StringMeasureGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringMeasureGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// StringEncode
// ---------------------------------------------------------------------------

func StringEncodeGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenStringEncodeGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StringEncodeGetStr(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringEncodeGetStr((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringEncodeGetArray(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringEncodeGetArray((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringEncodeGetStart(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringEncodeGetStart((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// StringConcat
// ---------------------------------------------------------------------------

func StringConcatGetLeft(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringConcatGetLeft((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringConcatGetRight(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringConcatGetRight((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// StringEq
// ---------------------------------------------------------------------------

func StringEqGetOp(expr ExpressionRef) Op {
	return Op(C.BinaryenStringEqGetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr))))
}

func StringEqGetLeft(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringEqGetLeft((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringEqGetRight(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringEqGetRight((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// StringWTF16Get
// ---------------------------------------------------------------------------

func StringWTF16GetGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringWTF16GetGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringWTF16GetGetPos(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringWTF16GetGetPos((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

// ---------------------------------------------------------------------------
// StringSliceWTF
// ---------------------------------------------------------------------------

func StringSliceWTFGetRef(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringSliceWTFGetRef((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringSliceWTFGetStart(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringSliceWTFGetStart((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}

func StringSliceWTFGetEnd(expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringSliceWTFGetEnd((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))))
}
