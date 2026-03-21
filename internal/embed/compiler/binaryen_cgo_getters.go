package asembed

/*
#cgo CFLAGS: -I${SRCDIR}/deps/binaryen/include
#include "binaryen-c.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// CGo getter/setter wrappers for Binaryen expression properties.
// These thin wrappers convert between Go and C types.

// ===== Block =====

func cgoBlockGetName(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenBlockGetName(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoBlockGetNumChildren(expr uintptr) int {
	return int(C.BinaryenBlockGetNumChildren(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoBlockGetChildAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenBlockGetChildAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

// ===== If =====

func cgoIfGetCondition(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenIfGetCondition(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoIfGetIfTrue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenIfGetIfTrue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoIfGetIfFalse(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenIfGetIfFalse(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Loop =====

func cgoLoopGetName(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenLoopGetName(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoLoopGetBody(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenLoopGetBody(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Break =====

func cgoBreakGetName(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenBreakGetName(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoBreakGetCondition(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenBreakGetCondition(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoBreakGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenBreakGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Switch =====

func cgoSwitchGetNumNames(expr uintptr) int {
	return int(C.BinaryenSwitchGetNumNames(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSwitchGetNameAt(expr uintptr, index int) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenSwitchGetNameAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))
}

func cgoSwitchGetDefaultName(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenSwitchGetDefaultName(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSwitchGetCondition(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSwitchGetCondition(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSwitchGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSwitchGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Call =====

func cgoCallGetTarget(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenCallGetTarget(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoCallGetNumOperands(expr uintptr) int {
	return int(C.BinaryenCallGetNumOperands(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoCallGetOperandAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenCallGetOperandAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

func cgoCallIsReturn(expr uintptr) bool {
	return bool(C.BinaryenCallIsReturn(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== CallIndirect =====

func cgoCallIndirectGetTarget(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenCallIndirectGetTarget(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoCallIndirectGetTable(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenCallIndirectGetTable(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoCallIndirectGetNumOperands(expr uintptr) int {
	return int(C.BinaryenCallIndirectGetNumOperands(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoCallIndirectGetOperandAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenCallIndirectGetOperandAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

func cgoCallIndirectIsReturn(expr uintptr) bool {
	return bool(C.BinaryenCallIndirectIsReturn(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== LocalGet =====

func cgoLocalGetGetIndex(expr uintptr) int {
	return int(C.BinaryenLocalGetGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// LocalGetSetIndex, LocalSetIsTee, LocalSetGetIndex, LocalSetGetValue already in binaryen_cgo.go

// ===== LocalSet =====

// ===== GlobalGet (expression) =====

func cgoGlobalGetGetName(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenGlobalGetGetName(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== GlobalSet (expression) =====
// cgoGlobalSetGetName, cgoGlobalSetGetValue already in binaryen_cgo.go

// ===== TableGet (expression) =====

func cgoTableGetGetTable(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTableGetGetTable(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoTableGetGetIndex(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenTableGetGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== TableSet (expression) =====
// cgoTableSetGetTable, cgoTableSetGetIndex, cgoTableSetGetValue already in binaryen_cgo.go

// ===== TableSize (expression) =====

func cgoTableSizeGetTable(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTableSizeGetTable(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== TableGrow (expression) =====

func cgoTableGrowGetTable(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTableGrowGetTable(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoTableGrowGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenTableGrowGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoTableGrowGetDelta(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenTableGrowGetDelta(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== MemoryGrow =====

func cgoMemoryGrowGetDelta(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryGrowGetDelta(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Load =====

func cgoLoadIsAtomic(expr uintptr) bool {
	return bool(C.BinaryenLoadIsAtomic(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoLoadIsSigned(expr uintptr) bool {
	return bool(C.BinaryenLoadIsSigned(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoLoadGetOffset(expr uintptr) uint32 {
	return uint32(C.BinaryenLoadGetOffset(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoLoadGetBytes(expr uintptr) uint32 {
	return uint32(C.BinaryenLoadGetBytes(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoLoadGetAlign(expr uintptr) uint32 {
	return uint32(C.BinaryenLoadGetAlign(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoLoadGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenLoadGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Store =====

func cgoStoreIsAtomic(expr uintptr) bool {
	return bool(C.BinaryenStoreIsAtomic(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStoreGetBytes(expr uintptr) uint32 {
	return uint32(C.BinaryenStoreGetBytes(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStoreGetOffset(expr uintptr) uint32 {
	return uint32(C.BinaryenStoreGetOffset(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStoreGetAlign(expr uintptr) uint32 {
	return uint32(C.BinaryenStoreGetAlign(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStoreGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStoreGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStoreGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStoreGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStoreGetValueType(expr uintptr) uintptr {
	return uintptr(C.BinaryenStoreGetValueType(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== Const =====

func cgoConstGetValueI32(expr uintptr) int32 {
	return int32(C.BinaryenConstGetValueI32(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoConstGetValueI64Low(expr uintptr) int32 {
	return int32(C.BinaryenConstGetValueI64Low(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoConstGetValueI64High(expr uintptr) int32 {
	return int32(C.BinaryenConstGetValueI64High(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoConstGetValueF32(expr uintptr) float32 {
	return float32(C.BinaryenConstGetValueF32(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoConstGetValueF64(expr uintptr) float64 {
	return float64(C.BinaryenConstGetValueF64(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoConstGetValueV128(expr uintptr, out *[16]byte) {
	C.BinaryenConstGetValueV128(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		(*C.uint8_t)(unsafe.Pointer(&out[0])),
	)
}

// ===== Unary =====

func cgoUnaryGetOp(expr uintptr) int32 {
	return int32(C.BinaryenUnaryGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoUnaryGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenUnaryGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Binary =====

func cgoBinaryGetOp(expr uintptr) int32 {
	return int32(C.BinaryenBinaryGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoBinaryGetLeft(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenBinaryGetLeft(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoBinaryGetRight(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenBinaryGetRight(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Select =====

func cgoSelectGetIfTrue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSelectGetIfTrue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSelectGetIfFalse(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSelectGetIfFalse(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSelectGetCondition(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSelectGetCondition(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Drop =====

func cgoDropGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenDropGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Return =====

func cgoReturnGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenReturnGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== AtomicRMW =====

func cgoAtomicRMWGetOp(expr uintptr) int32 {
	return int32(C.BinaryenAtomicRMWGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoAtomicRMWGetBytes(expr uintptr) uint32 {
	return uint32(C.BinaryenAtomicRMWGetBytes(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoAtomicRMWGetOffset(expr uintptr) uint32 {
	return uint32(C.BinaryenAtomicRMWGetOffset(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoAtomicRMWGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicRMWGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoAtomicRMWGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicRMWGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== AtomicCmpxchg =====

func cgoAtomicCmpxchgGetBytes(expr uintptr) uint32 {
	return uint32(C.BinaryenAtomicCmpxchgGetBytes(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoAtomicCmpxchgGetOffset(expr uintptr) uint32 {
	return uint32(C.BinaryenAtomicCmpxchgGetOffset(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoAtomicCmpxchgGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicCmpxchgGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoAtomicCmpxchgGetExpected(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicCmpxchgGetExpected(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoAtomicCmpxchgGetReplacement(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicCmpxchgGetReplacement(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== AtomicWait =====

func cgoAtomicWaitGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicWaitGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoAtomicWaitGetExpected(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicWaitGetExpected(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoAtomicWaitGetTimeout(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicWaitGetTimeout(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoAtomicWaitGetExpectedType(expr uintptr) uintptr {
	return uintptr(C.BinaryenAtomicWaitGetExpectedType(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== AtomicNotify =====

func cgoAtomicNotifyGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicNotifyGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoAtomicNotifyGetNotifyCount(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenAtomicNotifyGetNotifyCount(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== AtomicFence =====

func cgoAtomicFenceGetOrder(expr uintptr) int {
	return int(C.BinaryenAtomicFenceGetOrder(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== SIMDExtract =====

func cgoSIMDExtractGetOp(expr uintptr) int32 {
	return int32(C.BinaryenSIMDExtractGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDExtractGetVec(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDExtractGetVec(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDExtractGetIndex(expr uintptr) int {
	return int(C.BinaryenSIMDExtractGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== SIMDReplace =====

func cgoSIMDReplaceGetOp(expr uintptr) int32 {
	return int32(C.BinaryenSIMDReplaceGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDReplaceGetVec(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDReplaceGetVec(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDReplaceGetIndex(expr uintptr) int {
	return int(C.BinaryenSIMDReplaceGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDReplaceGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDReplaceGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== SIMDShuffle =====

func cgoSIMDShuffleGetLeft(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDShuffleGetLeft(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDShuffleGetRight(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDShuffleGetRight(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDShuffleGetMask(expr uintptr, mask *[16]byte) {
	C.BinaryenSIMDShuffleGetMask(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		(*C.uint8_t)(unsafe.Pointer(&mask[0])),
	)
}

// ===== SIMDTernary =====

func cgoSIMDTernaryGetOp(expr uintptr) int32 {
	return int32(C.BinaryenSIMDTernaryGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDTernaryGetA(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDTernaryGetA(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDTernaryGetB(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDTernaryGetB(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDTernaryGetC(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDTernaryGetC(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== SIMDShift =====

func cgoSIMDShiftGetOp(expr uintptr) int32 {
	return int32(C.BinaryenSIMDShiftGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDShiftGetVec(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDShiftGetVec(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDShiftGetShift(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDShiftGetShift(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== SIMDLoad =====

func cgoSIMDLoadGetOp(expr uintptr) int32 {
	return int32(C.BinaryenSIMDLoadGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDLoadGetOffset(expr uintptr) uint32 {
	return uint32(C.BinaryenSIMDLoadGetOffset(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDLoadGetAlign(expr uintptr) uint32 {
	return uint32(C.BinaryenSIMDLoadGetAlign(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDLoadGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDLoadGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== SIMDLoadStoreLane =====

func cgoSIMDLoadStoreLaneGetOp(expr uintptr) int32 {
	return int32(C.BinaryenSIMDLoadStoreLaneGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDLoadStoreLaneGetOffset(expr uintptr) uint32 {
	return uint32(C.BinaryenSIMDLoadStoreLaneGetOffset(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDLoadStoreLaneGetAlign(expr uintptr) uint32 {
	return uint32(C.BinaryenSIMDLoadStoreLaneGetAlign(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDLoadStoreLaneGetIndex(expr uintptr) int {
	return int(C.BinaryenSIMDLoadStoreLaneGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoSIMDLoadStoreLaneGetPtr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDLoadStoreLaneGetPtr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDLoadStoreLaneGetVec(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenSIMDLoadStoreLaneGetVec(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoSIMDLoadStoreLaneIsStore(expr uintptr) bool {
	return bool(C.BinaryenSIMDLoadStoreLaneIsStore(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== MemoryInit =====

func cgoMemoryInitGetSegment(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenMemoryInitGetSegment(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoMemoryInitGetDest(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryInitGetDest(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoMemoryInitGetOffset(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryInitGetOffset(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoMemoryInitGetSize(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryInitGetSize(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== DataDrop =====

func cgoDataDropGetSegment(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenDataDropGetSegment(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== MemoryCopy =====

func cgoMemoryCopyGetDest(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryCopyGetDest(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoMemoryCopyGetSource(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryCopyGetSource(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoMemoryCopyGetSize(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryCopyGetSize(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== MemoryFill =====

func cgoMemoryFillGetDest(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryFillGetDest(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoMemoryFillGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryFillGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoMemoryFillGetSize(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenMemoryFillGetSize(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== RefIsNull =====

func cgoRefIsNullGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenRefIsNullGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// cgoRefIsNullSetValue already in binaryen_cgo.go

// ===== RefAs =====

func cgoRefAsGetOp(expr uintptr) int32 {
	return int32(C.BinaryenRefAsGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoRefAsGetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenRefAsGetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== RefFunc =====

func cgoRefFuncGetFunc(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenRefFuncGetFunc(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== RefEq =====

func cgoRefEqGetLeft(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenRefEqGetLeft(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoRefEqGetRight(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenRefEqGetRight(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Try =====

func cgoTryGetName(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTryGetName(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoTryGetBody(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenTryGetBody(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoTryGetNumCatchTags(expr uintptr) int {
	return int(C.BinaryenTryGetNumCatchTags(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoTryGetNumCatchBodies(expr uintptr) int {
	return int(C.BinaryenTryGetNumCatchBodies(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoTryGetCatchTagAt(expr uintptr, index int) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTryGetCatchTagAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))
}

func cgoTryGetCatchBodyAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenTryGetCatchBodyAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

func cgoTryGetDelegateTarget(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTryGetDelegateTarget(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoTryIsDelegate(expr uintptr) bool {
	return bool(C.BinaryenTryIsDelegate(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== Throw =====

func cgoThrowGetTag(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenThrowGetTag(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoThrowGetNumOperands(expr uintptr) int {
	return int(C.BinaryenThrowGetNumOperands(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoThrowGetOperandAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenThrowGetOperandAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

// ===== Rethrow =====

func cgoRethrowGetTarget(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenRethrowGetTarget(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== TupleMake =====

func cgoTupleMakeGetNumOperands(expr uintptr) int {
	return int(C.BinaryenTupleMakeGetNumOperands(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoTupleMakeGetOperandAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenTupleMakeGetOperandAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

// ===== TupleExtract =====

func cgoTupleExtractGetTuple(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenTupleExtractGetTuple(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoTupleExtractGetIndex(expr uintptr) int {
	return int(C.BinaryenTupleExtractGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== RefI31 =====

func cgoRefI31GetValue(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenRefI31GetValue(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== I31Get =====

func cgoI31GetGetI31(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenI31GetGetI31(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoI31GetIsSigned(expr uintptr) bool {
	return bool(C.BinaryenI31GetIsSigned(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// cgoI31GetSetI31, cgoI31GetSetSigned already in binaryen_cgo.go

// ===== CallRef =====

func cgoCallRefGetNumOperands(expr uintptr) int {
	return int(C.BinaryenCallRefGetNumOperands(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoCallRefGetOperandAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenCallRefGetOperandAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

func cgoCallRefGetTarget(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenCallRefGetTarget(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoCallRefIsReturn(expr uintptr) bool {
	return bool(C.BinaryenCallRefIsReturn(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== RefTest =====

func cgoRefTestGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenRefTestGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoRefTestGetCastType(expr uintptr) uintptr {
	return uintptr(C.BinaryenRefTestGetCastType(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== RefCast =====

func cgoRefCastGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenRefCastGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== BrOn =====

func cgoBrOnGetOp(expr uintptr) int32 {
	return int32(C.BinaryenBrOnGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoBrOnGetName(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenBrOnGetName(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoBrOnGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenBrOnGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoBrOnGetCastType(expr uintptr) uintptr {
	return uintptr(C.BinaryenBrOnGetCastType(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== StructNew =====

func cgoStructNewGetNumOperands(expr uintptr) int {
	return int(C.BinaryenStructNewGetNumOperands(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStructNewGetOperandAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStructNewGetOperandAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

// ===== StructGet =====

func cgoStructGetGetIndex(expr uintptr) int {
	return int(C.BinaryenStructGetGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStructGetGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStructGetGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStructGetIsSigned(expr uintptr) bool {
	return bool(C.BinaryenStructGetIsSigned(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// cgoStructGetSetIndex, cgoStructGetSetRef, cgoStructGetSetSigned already in binaryen_cgo.go

// ===== StructSet =====
// cgoStructSetGetIndex, cgoStructSetGetRef, cgoStructSetGetValue already in binaryen_cgo.go

// ===== ArrayNew =====

func cgoArrayNewGetInit(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayNewGetInit(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoArrayNewGetSize(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayNewGetSize(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== ArrayNewFixed =====

func cgoArrayNewFixedGetNumValues(expr uintptr) int {
	return int(C.BinaryenArrayNewFixedGetNumValues(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoArrayNewFixedGetValueAt(expr uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayNewFixedGetValueAt(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	)))
}

// ===== ArrayGet =====

func cgoArrayGetGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayGetGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoArrayGetGetIndex(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayGetGetIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoArrayGetIsSigned(expr uintptr) bool {
	return bool(C.BinaryenArrayGetIsSigned(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// cgoArrayGetSetRef, cgoArrayGetSetIndex, cgoArrayGetSetSigned already in binaryen_cgo.go

// ===== ArraySet =====
// cgoArraySetGetRef, cgoArraySetGetIndex, cgoArraySetGetValue already in binaryen_cgo.go

// ===== ArrayLen =====

func cgoArrayLenGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayLenGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== ArrayCopy =====

func cgoArrayCopyGetDestRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayCopyGetDestRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoArrayCopyGetDestIndex(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayCopyGetDestIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoArrayCopyGetSrcRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayCopyGetSrcRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoArrayCopyGetSrcIndex(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayCopyGetSrcIndex(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoArrayCopyGetLength(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenArrayCopyGetLength(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== StringNew =====

func cgoStringNewGetOp(expr uintptr) int32 {
	return int32(C.BinaryenStringNewGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStringNewGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringNewGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringNewGetStart(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringNewGetStart(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringNewGetEnd(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringNewGetEnd(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== StringConst =====

func cgoStringConstGetString(expr uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenStringConstGetString(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

// ===== StringMeasure =====

func cgoStringMeasureGetOp(expr uintptr) int32 {
	return int32(C.BinaryenStringMeasureGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStringMeasureGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringMeasureGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== StringEncode =====

func cgoStringEncodeGetOp(expr uintptr) int32 {
	return int32(C.BinaryenStringEncodeGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStringEncodeGetStr(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringEncodeGetStr(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringEncodeGetArray(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringEncodeGetArray(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringEncodeGetStart(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringEncodeGetStart(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== StringConcat =====

func cgoStringConcatGetLeft(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringConcatGetLeft(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringConcatGetRight(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringConcatGetRight(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== StringEq =====

func cgoStringEqGetOp(expr uintptr) int32 {
	return int32(C.BinaryenStringEqGetOp(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoStringEqGetLeft(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringEqGetLeft(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringEqGetRight(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringEqGetRight(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== StringWTF16Get =====

func cgoStringWTF16GetGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringWTF16GetGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringWTF16GetGetPos(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringWTF16GetGetPos(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// cgoStringWTF16GetSetRef, cgoStringWTF16GetSetPos already in binaryen_cgo.go

// ===== StringSliceWTF =====

func cgoStringSliceWTFGetRef(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringSliceWTFGetRef(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringSliceWTFGetStart(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringSliceWTFGetStart(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

func cgoStringSliceWTFGetEnd(expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenStringSliceWTFGetEnd(C.BinaryenExpressionRef(unsafe.Pointer(expr)))))
}

// ===== Non-expression getters (module-level) =====

// Global object (not expression)
func cgoGlobalObjGetName(global uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenGlobalGetName(C.BinaryenGlobalRef(unsafe.Pointer(global))))
}

func cgoGlobalObjGetType(global uintptr) uintptr {
	return uintptr(C.BinaryenGlobalGetType(C.BinaryenGlobalRef(unsafe.Pointer(global))))
}

func cgoGlobalObjIsMutable(global uintptr) bool {
	return bool(C.BinaryenGlobalIsMutable(C.BinaryenGlobalRef(unsafe.Pointer(global))))
}

func cgoGlobalObjGetInitExpr(global uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGlobalGetInitExpr(C.BinaryenGlobalRef(unsafe.Pointer(global)))))
}

// Export object
func cgoExportGetKind(export uintptr) int {
	return int(C.BinaryenExportGetKind(C.BinaryenExportRef(unsafe.Pointer(export))))
}

func cgoExportGetName(export uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenExportGetName(C.BinaryenExportRef(unsafe.Pointer(export))))
}

func cgoExportGetValue(export uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenExportGetValue(C.BinaryenExportRef(unsafe.Pointer(export))))
}

// Function object
func cgoFunctionObjGetType(fn uintptr) uintptr {
	return uintptr(C.BinaryenFunctionGetType(C.BinaryenFunctionRef(unsafe.Pointer(fn))))
}

// Module-level indexed getters
func cgoGetExportByIndex(module uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetExportByIndex(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
	)))
}

func cgoGetGlobalByIndex(module uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetGlobalByIndex(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
	)))
}

func cgoGetTableByIndex(module uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetTableByIndex(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
	)))
}

// Element segments
func cgoGetNumElementSegments(module uintptr) int {
	return int(C.BinaryenGetNumElementSegments(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

func cgoGetElementSegment(module uintptr, name unsafe.Pointer) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetElementSegment(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)))
}

func cgoGetElementSegmentByIndex(module uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetElementSegmentByIndex(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
	)))
}

// Memory segments
func cgoGetNumMemorySegments(module uintptr) uint32 {
	return uint32(C.BinaryenGetNumMemorySegments(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

func cgoGetMemorySegmentByteOffset(module uintptr, name unsafe.Pointer) uint32 {
	return uint32(C.BinaryenGetMemorySegmentByteOffset(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	))
}

// Module debug info
func cgoModuleGetDebugInfoFileName(module uintptr, index int) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenModuleGetDebugInfoFileName(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
	))
}

// Tag getters
func cgoTagGetName(tag uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTagGetName(C.BinaryenTagRef(unsafe.Pointer(tag))))
}

func cgoTagGetParams(tag uintptr) uintptr {
	return uintptr(C.BinaryenTagGetParams(C.BinaryenTagRef(unsafe.Pointer(tag))))
}

func cgoTagGetResults(tag uintptr) uintptr {
	return uintptr(C.BinaryenTagGetResults(C.BinaryenTagRef(unsafe.Pointer(tag))))
}

// Table object getters
func cgoTableObjGetName(table uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenTableGetName(C.BinaryenTableRef(unsafe.Pointer(table))))
}

func cgoTableObjGetInitial(table uintptr) int {
	return int(C.BinaryenTableGetInitial(C.BinaryenTableRef(unsafe.Pointer(table))))
}

func cgoTableObjGetMax(table uintptr) int {
	return int(C.BinaryenTableGetMax(C.BinaryenTableRef(unsafe.Pointer(table))))
}

func cgoTableObjGetType(table uintptr) uintptr {
	return uintptr(C.BinaryenTableGetType(C.BinaryenTableRef(unsafe.Pointer(table))))
}
