package binaryen

/*
#include "binaryen-c.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// ---------------------------------------------------------------------------
// Expression (generic)
// ---------------------------------------------------------------------------

// ExpressionSetType sets the type of the given expression.
func ExpressionSetType(expr ExpressionRef, t Type) {
	C.BinaryenExpressionSetType((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenType(t))
}

// ExpressionFinalize re-finalizes an expression after it has been modified.
func ExpressionFinalize(expr ExpressionRef) {
	C.BinaryenExpressionFinalize((C.BinaryenExpressionRef)(unsafe.Pointer(expr)))
}

// ExpressionCopy makes a deep copy of the given expression.
func ExpressionCopy(expr ExpressionRef, module *Module) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenExpressionCopy(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		module.ref,
	))))
}

// ---------------------------------------------------------------------------
// Block
// ---------------------------------------------------------------------------

// BlockSetName sets the name (label) of a block expression.
func BlockSetName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenBlockSetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// BlockSetChildAt sets (replaces) the child expression at the specified index of a block.
func BlockSetChildAt(expr ExpressionRef, index Index, child ExpressionRef) {
	C.BinaryenBlockSetChildAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(child)),
	)
}

// BlockAppendChild appends a child expression to a block, returning its insertion index.
func BlockAppendChild(expr ExpressionRef, child ExpressionRef) Index {
	return Index(C.BinaryenBlockAppendChild(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(child)),
	))
}

// BlockInsertChildAt inserts a child expression at the specified index of a block.
func BlockInsertChildAt(expr ExpressionRef, index Index, child ExpressionRef) {
	C.BinaryenBlockInsertChildAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(child)),
	)
}

// BlockRemoveChildAt removes the child expression at the specified index and returns it.
func BlockRemoveChildAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenBlockRemoveChildAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// ---------------------------------------------------------------------------
// If
// ---------------------------------------------------------------------------

// IfSetCondition sets the condition expression of an if expression.
func IfSetCondition(expr ExpressionRef, cond ExpressionRef) {
	C.BinaryenIfSetCondition(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(cond)),
	)
}

// IfSetIfTrue sets the ifTrue (then) expression of an if expression.
func IfSetIfTrue(expr ExpressionRef, ifTrue ExpressionRef) {
	C.BinaryenIfSetIfTrue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifTrue)),
	)
}

// IfSetIfFalse sets the ifFalse (else) expression of an if expression.
func IfSetIfFalse(expr ExpressionRef, ifFalse ExpressionRef) {
	C.BinaryenIfSetIfFalse(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifFalse)),
	)
}

// ---------------------------------------------------------------------------
// Loop
// ---------------------------------------------------------------------------

// LoopSetName sets the name (label) of a loop expression.
func LoopSetName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenLoopSetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// LoopSetBody sets the body expression of a loop expression.
func LoopSetBody(expr ExpressionRef, body ExpressionRef) {
	C.BinaryenLoopSetBody(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(body)),
	)
}

// ---------------------------------------------------------------------------
// Break
// ---------------------------------------------------------------------------

// BreakSetName sets the name (target label) of a br or br_if expression.
func BreakSetName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenBreakSetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// BreakSetCondition sets the condition expression of a br_if expression.
func BreakSetCondition(expr ExpressionRef, cond ExpressionRef) {
	C.BinaryenBreakSetCondition(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(cond)),
	)
}

// BreakSetValue sets the value expression of a br or br_if expression.
func BreakSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenBreakSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// Switch
// ---------------------------------------------------------------------------

// SwitchSetNameAt sets the name (target label) at the specified index of a br_table.
func SwitchSetNameAt(expr ExpressionRef, index Index, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenSwitchSetNameAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		cs,
	)
}

// SwitchAppendName appends a name to a br_table, returning its insertion index.
func SwitchAppendName(expr ExpressionRef, name string) Index {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return Index(C.BinaryenSwitchAppendName(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		cs,
	))
}

// SwitchInsertNameAt inserts a name at the specified index of a br_table.
func SwitchInsertNameAt(expr ExpressionRef, index Index, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenSwitchInsertNameAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		cs,
	)
}

// SwitchRemoveNameAt removes the name at the specified index of a br_table.
// Returns the removed name.
func SwitchRemoveNameAt(expr ExpressionRef, index Index) string {
	return goString(C.BinaryenSwitchRemoveNameAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))
}

// SwitchSetDefaultName sets the default name (target label) of a br_table.
func SwitchSetDefaultName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenSwitchSetDefaultName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// SwitchSetCondition sets the condition expression of a br_table.
func SwitchSetCondition(expr ExpressionRef, cond ExpressionRef) {
	C.BinaryenSwitchSetCondition(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(cond)),
	)
}

// SwitchSetValue sets the value expression of a br_table.
func SwitchSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenSwitchSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// Call
// ---------------------------------------------------------------------------

// CallSetTarget sets the target function name of a call expression.
func CallSetTarget(expr ExpressionRef, target string) {
	cs := C.CString(target)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenCallSetTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// CallSetOperandAt sets the operand expression at the specified index of a call.
func CallSetOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenCallSetOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// CallAppendOperand appends an operand expression to a call, returning its insertion index.
func CallAppendOperand(expr ExpressionRef, operand ExpressionRef) Index {
	return Index(C.BinaryenCallAppendOperand(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	))
}

// CallInsertOperandAt inserts an operand expression at the specified index of a call.
func CallInsertOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenCallInsertOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// CallRemoveOperandAt removes the operand at the specified index and returns it.
func CallRemoveOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenCallRemoveOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// CallSetReturn sets whether the specified call expression is a tail call.
func CallSetReturn(expr ExpressionRef, isReturn bool) {
	C.BinaryenCallSetReturn((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(isReturn))
}

// ---------------------------------------------------------------------------
// CallIndirect
// ---------------------------------------------------------------------------

// CallIndirectSetTarget sets the target expression of a call_indirect.
func CallIndirectSetTarget(expr ExpressionRef, target ExpressionRef) {
	C.BinaryenCallIndirectSetTarget(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(target)),
	)
}

// CallIndirectSetTable sets the table name of a call_indirect.
func CallIndirectSetTable(expr ExpressionRef, table string) {
	cs := C.CString(table)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenCallIndirectSetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// CallIndirectSetOperandAt sets the operand at the specified index of a call_indirect.
func CallIndirectSetOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenCallIndirectSetOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// CallIndirectAppendOperand appends an operand to a call_indirect, returning its insertion index.
func CallIndirectAppendOperand(expr ExpressionRef, operand ExpressionRef) Index {
	return Index(C.BinaryenCallIndirectAppendOperand(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	))
}

// CallIndirectInsertOperandAt inserts an operand at the specified index of a call_indirect.
func CallIndirectInsertOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenCallIndirectInsertOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// CallIndirectRemoveOperandAt removes the operand at the specified index and returns it.
func CallIndirectRemoveOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenCallIndirectRemoveOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// CallIndirectSetReturn sets whether the call_indirect is a tail call.
func CallIndirectSetReturn(expr ExpressionRef, isReturn bool) {
	C.BinaryenCallIndirectSetReturn((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(isReturn))
}

// CallIndirectSetParams sets the parameter types of a call_indirect.
func CallIndirectSetParams(expr ExpressionRef, params Type) {
	C.BinaryenCallIndirectSetParams((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenType(params))
}

// CallIndirectSetResults sets the result types of a call_indirect.
func CallIndirectSetResults(expr ExpressionRef, results Type) {
	C.BinaryenCallIndirectSetResults((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenType(results))
}

// ---------------------------------------------------------------------------
// LocalGet
// ---------------------------------------------------------------------------

// LocalGetSetIndex sets the local index of a local.get expression.
func LocalGetSetIndex(expr ExpressionRef, index Index) {
	C.BinaryenLocalGetSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))
}

// ---------------------------------------------------------------------------
// LocalSet
// ---------------------------------------------------------------------------

// LocalSetSetIndex sets the local index of a local.set or local.tee expression.
func LocalSetSetIndex(expr ExpressionRef, index Index) {
	C.BinaryenLocalSetSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))
}

// LocalSetSetValue sets the value expression of a local.set or local.tee expression.
func LocalSetSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenLocalSetSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// GlobalGet
// ---------------------------------------------------------------------------

// GlobalGetSetName sets the name of the global accessed by a global.get expression.
func GlobalGetSetName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenGlobalGetSetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ---------------------------------------------------------------------------
// GlobalSet
// ---------------------------------------------------------------------------

// GlobalSetSetName sets the name of the global accessed by a global.set expression.
func GlobalSetSetName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenGlobalSetSetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// GlobalSetSetValue sets the value expression of a global.set expression.
func GlobalSetSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenGlobalSetSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// TableGet
// ---------------------------------------------------------------------------

// TableGetSetTable sets the name of the table accessed by a table.get expression.
func TableGetSetTable(expr ExpressionRef, table string) {
	cs := C.CString(table)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTableGetSetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// TableGetSetIndex sets the index expression of a table.get expression.
func TableGetSetIndex(expr ExpressionRef, index ExpressionRef) {
	C.BinaryenTableGetSetIndex(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
	)
}

// ---------------------------------------------------------------------------
// TableSet
// ---------------------------------------------------------------------------

// TableSetSetTable sets the name of the table accessed by a table.set expression.
func TableSetSetTable(expr ExpressionRef, table string) {
	cs := C.CString(table)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTableSetSetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// TableSetSetIndex sets the index expression of a table.set expression.
func TableSetSetIndex(expr ExpressionRef, index ExpressionRef) {
	C.BinaryenTableSetSetIndex(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
	)
}

// TableSetSetValue sets the value expression of a table.set expression.
func TableSetSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenTableSetSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// TableSize
// ---------------------------------------------------------------------------

// TableSizeSetTable sets the name of the table accessed by a table.size expression.
func TableSizeSetTable(expr ExpressionRef, table string) {
	cs := C.CString(table)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTableSizeSetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ---------------------------------------------------------------------------
// TableGrow
// ---------------------------------------------------------------------------

// TableGrowSetTable sets the name of the table accessed by a table.grow expression.
func TableGrowSetTable(expr ExpressionRef, table string) {
	cs := C.CString(table)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTableGrowSetTable((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// TableGrowSetValue sets the value expression of a table.grow expression.
func TableGrowSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenTableGrowSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// TableGrowSetDelta sets the delta expression of a table.grow expression.
func TableGrowSetDelta(expr ExpressionRef, delta ExpressionRef) {
	C.BinaryenTableGrowSetDelta(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(delta)),
	)
}

// ---------------------------------------------------------------------------
// MemoryGrow
// ---------------------------------------------------------------------------

// MemoryGrowSetDelta sets the delta expression of a memory.grow expression.
func MemoryGrowSetDelta(expr ExpressionRef, delta ExpressionRef) {
	C.BinaryenMemoryGrowSetDelta(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(delta)),
	)
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

// LoadSetAtomic sets whether a load expression is atomic.
func LoadSetAtomic(expr ExpressionRef, isAtomic bool) {
	C.BinaryenLoadSetAtomic((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(isAtomic))
}

// LoadSetSigned sets whether a load expression operates on a signed value.
func LoadSetSigned(expr ExpressionRef, isSigned bool) {
	C.BinaryenLoadSetSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(isSigned))
}

// LoadSetOffset sets the constant offset of a load expression.
func LoadSetOffset(expr ExpressionRef, offset uint32) {
	C.BinaryenLoadSetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(offset))
}

// LoadSetBytes sets the number of bytes loaded by a load expression.
func LoadSetBytes(expr ExpressionRef, bytes uint32) {
	C.BinaryenLoadSetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(bytes))
}

// LoadSetAlign sets the byte alignment of a load expression.
func LoadSetAlign(expr ExpressionRef, align uint32) {
	C.BinaryenLoadSetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(align))
}

// LoadSetPtr sets the pointer expression of a load expression.
func LoadSetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenLoadSetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

// StoreSetAtomic sets whether a store expression is atomic.
func StoreSetAtomic(expr ExpressionRef, isAtomic bool) {
	C.BinaryenStoreSetAtomic((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(isAtomic))
}

// StoreSetBytes sets the number of bytes stored by a store expression.
func StoreSetBytes(expr ExpressionRef, bytes uint32) {
	C.BinaryenStoreSetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(bytes))
}

// StoreSetOffset sets the constant offset of a store expression.
func StoreSetOffset(expr ExpressionRef, offset uint32) {
	C.BinaryenStoreSetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(offset))
}

// StoreSetAlign sets the byte alignment of a store expression.
func StoreSetAlign(expr ExpressionRef, align uint32) {
	C.BinaryenStoreSetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(align))
}

// StoreSetPtr sets the pointer expression of a store expression.
func StoreSetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenStoreSetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// StoreSetValue sets the value expression of a store expression.
func StoreSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenStoreSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// StoreSetValueType sets the value type of a store expression.
func StoreSetValueType(expr ExpressionRef, valueType Type) {
	C.BinaryenStoreSetValueType((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenType(valueType))
}

// ---------------------------------------------------------------------------
// Const
// ---------------------------------------------------------------------------

// ConstSetValueI32 sets the 32-bit integer value of an i32.const expression.
func ConstSetValueI32(expr ExpressionRef, value int32) {
	C.BinaryenConstSetValueI32((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.int32_t(value))
}

// ConstSetValueI64 sets the 64-bit integer value of an i64.const expression.
func ConstSetValueI64(expr ExpressionRef, value int64) {
	C.BinaryenConstSetValueI64((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.int64_t(value))
}

// ConstSetValueI64Low sets the low 32-bits of the 64-bit integer value of an i64.const.
func ConstSetValueI64Low(expr ExpressionRef, valueLow int32) {
	C.BinaryenConstSetValueI64Low((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.int32_t(valueLow))
}

// ConstSetValueI64High sets the high 32-bits of the 64-bit integer value of an i64.const.
func ConstSetValueI64High(expr ExpressionRef, valueHigh int32) {
	C.BinaryenConstSetValueI64High((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.int32_t(valueHigh))
}

// ConstSetValueF32 sets the 32-bit float value of a f32.const expression.
func ConstSetValueF32(expr ExpressionRef, value float32) {
	C.BinaryenConstSetValueF32((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.float(value))
}

// ConstSetValueF64 sets the 64-bit float (double) value of a f64.const expression.
func ConstSetValueF64(expr ExpressionRef, value float64) {
	C.BinaryenConstSetValueF64((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.double(value))
}

// ConstSetValueV128 sets the 128-bit vector value of a v128.const expression.
func ConstSetValueV128(expr ExpressionRef, value [16]byte) {
	C.BinaryenConstSetValueV128(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(*C.uint8_t)(unsafe.Pointer(&value[0])),
	)
}

// ---------------------------------------------------------------------------
// Unary
// ---------------------------------------------------------------------------

// UnarySetOp sets the operation performed by a unary expression.
func UnarySetOp(expr ExpressionRef, op Op) {
	C.BinaryenUnarySetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// UnarySetValue sets the value expression of a unary expression.
func UnarySetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenUnarySetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// Binary
// ---------------------------------------------------------------------------

// BinarySetOp sets the operation performed by a binary expression.
func BinarySetOp(expr ExpressionRef, op Op) {
	C.BinaryenBinarySetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// BinarySetLeft sets the left expression of a binary expression.
func BinarySetLeft(expr ExpressionRef, left ExpressionRef) {
	C.BinaryenBinarySetLeft(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
	)
}

// BinarySetRight sets the right expression of a binary expression.
func BinarySetRight(expr ExpressionRef, right ExpressionRef) {
	C.BinaryenBinarySetRight(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)
}

// ---------------------------------------------------------------------------
// Select
// ---------------------------------------------------------------------------

// SelectSetIfTrue sets the expression selected if the condition is true.
func SelectSetIfTrue(expr ExpressionRef, ifTrue ExpressionRef) {
	C.BinaryenSelectSetIfTrue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifTrue)),
	)
}

// SelectSetIfFalse sets the expression selected if the condition is false.
func SelectSetIfFalse(expr ExpressionRef, ifFalse ExpressionRef) {
	C.BinaryenSelectSetIfFalse(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifFalse)),
	)
}

// SelectSetCondition sets the condition expression of a select expression.
func SelectSetCondition(expr ExpressionRef, cond ExpressionRef) {
	C.BinaryenSelectSetCondition(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(cond)),
	)
}

// ---------------------------------------------------------------------------
// Drop
// ---------------------------------------------------------------------------

// DropSetValue sets the value expression being dropped.
func DropSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenDropSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// Return
// ---------------------------------------------------------------------------

// ReturnSetValue sets the value expression being returned.
func ReturnSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenReturnSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// AtomicRMW
// ---------------------------------------------------------------------------

// AtomicRMWSetOp sets the operation performed by an atomic read-modify-write expression.
func AtomicRMWSetOp(expr ExpressionRef, op Op) {
	C.BinaryenAtomicRMWSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// AtomicRMWSetBytes sets the number of bytes affected by an atomic RMW expression.
func AtomicRMWSetBytes(expr ExpressionRef, bytes uint32) {
	C.BinaryenAtomicRMWSetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(bytes))
}

// AtomicRMWSetOffset sets the constant offset of an atomic RMW expression.
func AtomicRMWSetOffset(expr ExpressionRef, offset uint32) {
	C.BinaryenAtomicRMWSetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(offset))
}

// AtomicRMWSetPtr sets the pointer expression of an atomic RMW expression.
func AtomicRMWSetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenAtomicRMWSetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// AtomicRMWSetValue sets the value expression of an atomic RMW expression.
func AtomicRMWSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenAtomicRMWSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// AtomicCmpxchg
// ---------------------------------------------------------------------------

// AtomicCmpxchgSetBytes sets the number of bytes affected by an atomic cmpxchg expression.
func AtomicCmpxchgSetBytes(expr ExpressionRef, bytes uint32) {
	C.BinaryenAtomicCmpxchgSetBytes((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(bytes))
}

// AtomicCmpxchgSetOffset sets the constant offset of an atomic cmpxchg expression.
func AtomicCmpxchgSetOffset(expr ExpressionRef, offset uint32) {
	C.BinaryenAtomicCmpxchgSetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(offset))
}

// AtomicCmpxchgSetPtr sets the pointer expression of an atomic cmpxchg expression.
func AtomicCmpxchgSetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenAtomicCmpxchgSetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// AtomicCmpxchgSetExpected sets the expected value expression of an atomic cmpxchg.
func AtomicCmpxchgSetExpected(expr ExpressionRef, expected ExpressionRef) {
	C.BinaryenAtomicCmpxchgSetExpected(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(expected)),
	)
}

// AtomicCmpxchgSetReplacement sets the replacement expression of an atomic cmpxchg.
func AtomicCmpxchgSetReplacement(expr ExpressionRef, replacement ExpressionRef) {
	C.BinaryenAtomicCmpxchgSetReplacement(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(replacement)),
	)
}

// ---------------------------------------------------------------------------
// AtomicWait
// ---------------------------------------------------------------------------

// AtomicWaitSetPtr sets the pointer expression of a memory.atomic.wait expression.
func AtomicWaitSetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenAtomicWaitSetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// AtomicWaitSetExpected sets the expected value expression of a memory.atomic.wait.
func AtomicWaitSetExpected(expr ExpressionRef, expected ExpressionRef) {
	C.BinaryenAtomicWaitSetExpected(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(expected)),
	)
}

// AtomicWaitSetTimeout sets the timeout expression of a memory.atomic.wait.
func AtomicWaitSetTimeout(expr ExpressionRef, timeout ExpressionRef) {
	C.BinaryenAtomicWaitSetTimeout(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(timeout)),
	)
}

// AtomicWaitSetExpectedType sets the expected type of a memory.atomic.wait expression.
func AtomicWaitSetExpectedType(expr ExpressionRef, expectedType Type) {
	C.BinaryenAtomicWaitSetExpectedType((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenType(expectedType))
}

// ---------------------------------------------------------------------------
// AtomicNotify
// ---------------------------------------------------------------------------

// AtomicNotifySetPtr sets the pointer expression of a memory.atomic.notify expression.
func AtomicNotifySetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenAtomicNotifySetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// AtomicNotifySetNotifyCount sets the notify count expression of a memory.atomic.notify.
func AtomicNotifySetNotifyCount(expr ExpressionRef, notifyCount ExpressionRef) {
	C.BinaryenAtomicNotifySetNotifyCount(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(notifyCount)),
	)
}

// ---------------------------------------------------------------------------
// AtomicFence
// ---------------------------------------------------------------------------

// AtomicFenceSetOrder sets the order of an atomic.fence expression.
func AtomicFenceSetOrder(expr ExpressionRef, order uint8) {
	C.BinaryenAtomicFenceSetOrder((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint8_t(order))
}

// ---------------------------------------------------------------------------
// SIMDExtract
// ---------------------------------------------------------------------------

// SIMDExtractSetOp sets the operation performed by a SIMD extract expression.
func SIMDExtractSetOp(expr ExpressionRef, op Op) {
	C.BinaryenSIMDExtractSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// SIMDExtractSetVec sets the vector expression of a SIMD extract expression.
func SIMDExtractSetVec(expr ExpressionRef, vec ExpressionRef) {
	C.BinaryenSIMDExtractSetVec(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
	)
}

// SIMDExtractSetIndex sets the lane index of a SIMD extract expression.
func SIMDExtractSetIndex(expr ExpressionRef, index uint8) {
	C.BinaryenSIMDExtractSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint8_t(index))
}

// ---------------------------------------------------------------------------
// SIMDReplace
// ---------------------------------------------------------------------------

// SIMDReplaceSetOp sets the operation performed by a SIMD replace expression.
func SIMDReplaceSetOp(expr ExpressionRef, op Op) {
	C.BinaryenSIMDReplaceSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// SIMDReplaceSetVec sets the vector expression of a SIMD replace expression.
func SIMDReplaceSetVec(expr ExpressionRef, vec ExpressionRef) {
	C.BinaryenSIMDReplaceSetVec(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
	)
}

// SIMDReplaceSetIndex sets the lane index of a SIMD replace expression.
func SIMDReplaceSetIndex(expr ExpressionRef, index uint8) {
	C.BinaryenSIMDReplaceSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint8_t(index))
}

// SIMDReplaceSetValue sets the value expression of a SIMD replace expression.
func SIMDReplaceSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenSIMDReplaceSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// SIMDShuffle
// ---------------------------------------------------------------------------

// SIMDShuffleSetLeft sets the left expression of a SIMD shuffle expression.
func SIMDShuffleSetLeft(expr ExpressionRef, left ExpressionRef) {
	C.BinaryenSIMDShuffleSetLeft(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
	)
}

// SIMDShuffleSetRight sets the right expression of a SIMD shuffle expression.
func SIMDShuffleSetRight(expr ExpressionRef, right ExpressionRef) {
	C.BinaryenSIMDShuffleSetRight(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)
}

// SIMDShuffleSetMask sets the 128-bit mask of a SIMD shuffle expression.
func SIMDShuffleSetMask(expr ExpressionRef, mask [16]byte) {
	C.BinaryenSIMDShuffleSetMask(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(*C.uint8_t)(unsafe.Pointer(&mask[0])),
	)
}

// ---------------------------------------------------------------------------
// SIMDTernary
// ---------------------------------------------------------------------------

// SIMDTernarySetOp sets the operation performed by a SIMD ternary expression.
func SIMDTernarySetOp(expr ExpressionRef, op Op) {
	C.BinaryenSIMDTernarySetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// SIMDTernarySetA sets the first operand expression of a SIMD ternary expression.
func SIMDTernarySetA(expr ExpressionRef, a ExpressionRef) {
	C.BinaryenSIMDTernarySetA(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(a)),
	)
}

// SIMDTernarySetB sets the second operand expression of a SIMD ternary expression.
func SIMDTernarySetB(expr ExpressionRef, b ExpressionRef) {
	C.BinaryenSIMDTernarySetB(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(b)),
	)
}

// SIMDTernarySetC sets the third operand expression of a SIMD ternary expression.
func SIMDTernarySetC(expr ExpressionRef, c ExpressionRef) {
	C.BinaryenSIMDTernarySetC(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(c)),
	)
}

// ---------------------------------------------------------------------------
// SIMDShift
// ---------------------------------------------------------------------------

// SIMDShiftSetOp sets the operation performed by a SIMD shift expression.
func SIMDShiftSetOp(expr ExpressionRef, op Op) {
	C.BinaryenSIMDShiftSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// SIMDShiftSetVec sets the expression being shifted by a SIMD shift expression.
func SIMDShiftSetVec(expr ExpressionRef, vec ExpressionRef) {
	C.BinaryenSIMDShiftSetVec(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
	)
}

// SIMDShiftSetShift sets the shift amount expression of a SIMD shift expression.
func SIMDShiftSetShift(expr ExpressionRef, shift ExpressionRef) {
	C.BinaryenSIMDShiftSetShift(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(shift)),
	)
}

// ---------------------------------------------------------------------------
// SIMDLoad
// ---------------------------------------------------------------------------

// SIMDLoadSetOp sets the operation performed by a SIMD load expression.
func SIMDLoadSetOp(expr ExpressionRef, op Op) {
	C.BinaryenSIMDLoadSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// SIMDLoadSetOffset sets the constant offset of a SIMD load expression.
func SIMDLoadSetOffset(expr ExpressionRef, offset uint32) {
	C.BinaryenSIMDLoadSetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(offset))
}

// SIMDLoadSetAlign sets the byte alignment of a SIMD load expression.
func SIMDLoadSetAlign(expr ExpressionRef, align uint32) {
	C.BinaryenSIMDLoadSetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(align))
}

// SIMDLoadSetPtr sets the pointer expression of a SIMD load expression.
func SIMDLoadSetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenSIMDLoadSetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// ---------------------------------------------------------------------------
// SIMDLoadStoreLane
// ---------------------------------------------------------------------------

// SIMDLoadStoreLaneSetOp sets the operation performed by a SIMD load/store lane expression.
func SIMDLoadStoreLaneSetOp(expr ExpressionRef, op Op) {
	C.BinaryenSIMDLoadStoreLaneSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// SIMDLoadStoreLaneSetOffset sets the constant offset of a SIMD load/store lane expression.
func SIMDLoadStoreLaneSetOffset(expr ExpressionRef, offset uint32) {
	C.BinaryenSIMDLoadStoreLaneSetOffset((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(offset))
}

// SIMDLoadStoreLaneSetAlign sets the byte alignment of a SIMD load/store lane expression.
func SIMDLoadStoreLaneSetAlign(expr ExpressionRef, align uint32) {
	C.BinaryenSIMDLoadStoreLaneSetAlign((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint32_t(align))
}

// SIMDLoadStoreLaneSetIndex sets the lane index of a SIMD load/store lane expression.
func SIMDLoadStoreLaneSetIndex(expr ExpressionRef, index uint8) {
	C.BinaryenSIMDLoadStoreLaneSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.uint8_t(index))
}

// SIMDLoadStoreLaneSetPtr sets the pointer expression of a SIMD load/store lane expression.
func SIMDLoadStoreLaneSetPtr(expr ExpressionRef, ptr ExpressionRef) {
	C.BinaryenSIMDLoadStoreLaneSetPtr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
	)
}

// SIMDLoadStoreLaneSetVec sets the vector expression of a SIMD load/store lane expression.
func SIMDLoadStoreLaneSetVec(expr ExpressionRef, vec ExpressionRef) {
	C.BinaryenSIMDLoadStoreLaneSetVec(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
	)
}

// ---------------------------------------------------------------------------
// MemoryInit
// ---------------------------------------------------------------------------

// MemoryInitSetSegment sets the segment name of a memory.init expression.
func MemoryInitSetSegment(expr ExpressionRef, segment string) {
	cs := C.CString(segment)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenMemoryInitSetSegment((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// MemoryInitSetDest sets the destination expression of a memory.init expression.
func MemoryInitSetDest(expr ExpressionRef, dest ExpressionRef) {
	C.BinaryenMemoryInitSetDest(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(dest)),
	)
}

// MemoryInitSetOffset sets the offset expression of a memory.init expression.
func MemoryInitSetOffset(expr ExpressionRef, offset ExpressionRef) {
	C.BinaryenMemoryInitSetOffset(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(offset)),
	)
}

// MemoryInitSetSize sets the size expression of a memory.init expression.
func MemoryInitSetSize(expr ExpressionRef, size ExpressionRef) {
	C.BinaryenMemoryInitSetSize(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
	)
}

// ---------------------------------------------------------------------------
// DataDrop
// ---------------------------------------------------------------------------

// DataDropSetSegment sets the segment name of a data.drop expression.
func DataDropSetSegment(expr ExpressionRef, segment string) {
	cs := C.CString(segment)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenDataDropSetSegment((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ---------------------------------------------------------------------------
// MemoryCopy
// ---------------------------------------------------------------------------

// MemoryCopySetDest sets the destination expression of a memory.copy expression.
func MemoryCopySetDest(expr ExpressionRef, dest ExpressionRef) {
	C.BinaryenMemoryCopySetDest(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(dest)),
	)
}

// MemoryCopySetSource sets the source expression of a memory.copy expression.
func MemoryCopySetSource(expr ExpressionRef, source ExpressionRef) {
	C.BinaryenMemoryCopySetSource(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(source)),
	)
}

// MemoryCopySetSize sets the size expression of a memory.copy expression.
func MemoryCopySetSize(expr ExpressionRef, size ExpressionRef) {
	C.BinaryenMemoryCopySetSize(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
	)
}

// ---------------------------------------------------------------------------
// MemoryFill
// ---------------------------------------------------------------------------

// MemoryFillSetDest sets the destination expression of a memory.fill expression.
func MemoryFillSetDest(expr ExpressionRef, dest ExpressionRef) {
	C.BinaryenMemoryFillSetDest(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(dest)),
	)
}

// MemoryFillSetValue sets the value expression of a memory.fill expression.
func MemoryFillSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenMemoryFillSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// MemoryFillSetSize sets the size expression of a memory.fill expression.
func MemoryFillSetSize(expr ExpressionRef, size ExpressionRef) {
	C.BinaryenMemoryFillSetSize(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
	)
}

// ---------------------------------------------------------------------------
// RefIsNull
// ---------------------------------------------------------------------------

// RefIsNullSetValue sets the value expression tested by a ref.is_null expression.
func RefIsNullSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenRefIsNullSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// RefAs
// ---------------------------------------------------------------------------

// RefAsSetOp sets the operation performed by a ref.as_* expression.
func RefAsSetOp(expr ExpressionRef, op Op) {
	C.BinaryenRefAsSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// RefAsSetValue sets the value expression tested by a ref.as_* expression.
func RefAsSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenRefAsSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// RefFunc
// ---------------------------------------------------------------------------

// RefFuncSetFunc sets the name of the function wrapped by a ref.func expression.
func RefFuncSetFunc(expr ExpressionRef, funcName string) {
	cs := C.CString(funcName)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenRefFuncSetFunc((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ---------------------------------------------------------------------------
// RefEq
// ---------------------------------------------------------------------------

// RefEqSetLeft sets the left expression of a ref.eq expression.
func RefEqSetLeft(expr ExpressionRef, left ExpressionRef) {
	C.BinaryenRefEqSetLeft(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
	)
}

// RefEqSetRight sets the right expression of a ref.eq expression.
func RefEqSetRight(expr ExpressionRef, right ExpressionRef) {
	C.BinaryenRefEqSetRight(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)
}

// ---------------------------------------------------------------------------
// Try
// ---------------------------------------------------------------------------

// TrySetName sets the name (label) of a try expression.
func TrySetName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTrySetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// TrySetBody sets the body expression of a try expression.
func TrySetBody(expr ExpressionRef, body ExpressionRef) {
	C.BinaryenTrySetBody(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(body)),
	)
}

// TrySetCatchTagAt sets the catch tag at the specified index of a try expression.
func TrySetCatchTagAt(expr ExpressionRef, index Index, catchTag string) {
	cs := C.CString(catchTag)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTrySetCatchTagAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		cs,
	)
}

// TryAppendCatchTag appends a catch tag to a try expression, returning its insertion index.
func TryAppendCatchTag(expr ExpressionRef, catchTag string) Index {
	cs := C.CString(catchTag)
	defer C.free(unsafe.Pointer(cs))
	return Index(C.BinaryenTryAppendCatchTag(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		cs,
	))
}

// TryInsertCatchTagAt inserts a catch tag at the specified index of a try expression.
func TryInsertCatchTagAt(expr ExpressionRef, index Index, catchTag string) {
	cs := C.CString(catchTag)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTryInsertCatchTagAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		cs,
	)
}

// TryRemoveCatchTagAt removes the catch tag at the specified index and returns it.
func TryRemoveCatchTagAt(expr ExpressionRef, index Index) string {
	return goString(C.BinaryenTryRemoveCatchTagAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))
}

// TrySetCatchBodyAt sets the catch body at the specified index of a try expression.
func TrySetCatchBodyAt(expr ExpressionRef, index Index, catchExpr ExpressionRef) {
	C.BinaryenTrySetCatchBodyAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(catchExpr)),
	)
}

// TryAppendCatchBody appends a catch body to a try expression, returning its insertion index.
func TryAppendCatchBody(expr ExpressionRef, catchExpr ExpressionRef) Index {
	return Index(C.BinaryenTryAppendCatchBody(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(catchExpr)),
	))
}

// TryInsertCatchBodyAt inserts a catch body at the specified index of a try expression.
func TryInsertCatchBodyAt(expr ExpressionRef, index Index, catchExpr ExpressionRef) {
	C.BinaryenTryInsertCatchBodyAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(catchExpr)),
	)
}

// TryRemoveCatchBodyAt removes the catch body at the specified index and returns it.
func TryRemoveCatchBodyAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenTryRemoveCatchBodyAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// TrySetDelegateTarget sets the target label of a delegate expression.
func TrySetDelegateTarget(expr ExpressionRef, delegateTarget string) {
	cs := C.CString(delegateTarget)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenTrySetDelegateTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ---------------------------------------------------------------------------
// Throw
// ---------------------------------------------------------------------------

// ThrowSetTag sets the name of the tag being thrown by a throw expression.
func ThrowSetTag(expr ExpressionRef, tagName string) {
	cs := C.CString(tagName)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenThrowSetTag((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ThrowSetOperandAt sets the operand at the specified index of a throw expression.
func ThrowSetOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenThrowSetOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// ThrowAppendOperand appends an operand to a throw expression, returning its insertion index.
func ThrowAppendOperand(expr ExpressionRef, operand ExpressionRef) Index {
	return Index(C.BinaryenThrowAppendOperand(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	))
}

// ThrowInsertOperandAt inserts an operand at the specified index of a throw expression.
func ThrowInsertOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenThrowInsertOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// ThrowRemoveOperandAt removes the operand at the specified index and returns it.
func ThrowRemoveOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenThrowRemoveOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// ---------------------------------------------------------------------------
// Rethrow
// ---------------------------------------------------------------------------

// RethrowSetTarget sets the target catch's try label of a rethrow expression.
func RethrowSetTarget(expr ExpressionRef, target string) {
	cs := C.CString(target)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenRethrowSetTarget((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ---------------------------------------------------------------------------
// TupleMake
// ---------------------------------------------------------------------------

// TupleMakeSetOperandAt sets the operand at the specified index of a tuple.make expression.
func TupleMakeSetOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenTupleMakeSetOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// TupleMakeAppendOperand appends an operand to a tuple.make expression, returning its insertion index.
func TupleMakeAppendOperand(expr ExpressionRef, operand ExpressionRef) Index {
	return Index(C.BinaryenTupleMakeAppendOperand(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	))
}

// TupleMakeInsertOperandAt inserts an operand at the specified index of a tuple.make expression.
func TupleMakeInsertOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenTupleMakeInsertOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// TupleMakeRemoveOperandAt removes the operand at the specified index and returns it.
func TupleMakeRemoveOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenTupleMakeRemoveOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// ---------------------------------------------------------------------------
// TupleExtract
// ---------------------------------------------------------------------------

// TupleExtractSetTuple sets the tuple expression of a tuple.extract expression.
func TupleExtractSetTuple(expr ExpressionRef, tuple ExpressionRef) {
	C.BinaryenTupleExtractSetTuple(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(tuple)),
	)
}

// TupleExtractSetIndex sets the index extracted at of a tuple.extract expression.
func TupleExtractSetIndex(expr ExpressionRef, index Index) {
	C.BinaryenTupleExtractSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))
}

// ---------------------------------------------------------------------------
// RefI31
// ---------------------------------------------------------------------------

// RefI31SetValue sets the value expression of a ref.i31 expression.
func RefI31SetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenRefI31SetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// I31Get
// ---------------------------------------------------------------------------

// I31GetSetI31 sets the i31 expression of an i31.get expression.
func I31GetSetI31(expr ExpressionRef, i31 ExpressionRef) {
	C.BinaryenI31GetSetI31(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(i31)),
	)
}

// I31GetSetSigned sets whether an i31.get expression returns a signed value.
func I31GetSetSigned(expr ExpressionRef, signed bool) {
	C.BinaryenI31GetSetSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(signed))
}

// ---------------------------------------------------------------------------
// CallRef
// ---------------------------------------------------------------------------

// CallRefSetOperandAt sets the operand at the specified index of a call_ref expression.
func CallRefSetOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenCallRefSetOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// CallRefAppendOperand appends an operand to a call_ref, returning its insertion index.
func CallRefAppendOperand(expr ExpressionRef, operand ExpressionRef) Index {
	return Index(C.BinaryenCallRefAppendOperand(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	))
}

// CallRefInsertOperandAt inserts an operand at the specified index of a call_ref.
func CallRefInsertOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenCallRefInsertOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// CallRefRemoveOperandAt removes the operand at the specified index and returns it.
func CallRefRemoveOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenCallRefRemoveOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// CallRefSetTarget sets the target expression of a call_ref expression.
func CallRefSetTarget(expr ExpressionRef, target ExpressionRef) {
	C.BinaryenCallRefSetTarget(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(target)),
	)
}

// CallRefSetReturn sets whether a call_ref is a tail call.
func CallRefSetReturn(expr ExpressionRef, isReturn bool) {
	C.BinaryenCallRefSetReturn((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(isReturn))
}

// ---------------------------------------------------------------------------
// RefTest
// ---------------------------------------------------------------------------

// RefTestSetRef sets the ref expression of a ref.test expression.
func RefTestSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenRefTestSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// RefTestSetCastType sets the cast type of a ref.test expression.
func RefTestSetCastType(expr ExpressionRef, intendedType Type) {
	C.BinaryenRefTestSetCastType((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenType(intendedType))
}

// ---------------------------------------------------------------------------
// RefCast
// ---------------------------------------------------------------------------

// RefCastSetRef sets the ref expression of a ref.cast expression.
func RefCastSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenRefCastSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// ---------------------------------------------------------------------------
// BrOn
// ---------------------------------------------------------------------------

// BrOnSetOp sets the operation of a br_on_* expression.
func BrOnSetOp(expr ExpressionRef, op Op) {
	C.BinaryenBrOnSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// BrOnSetName sets the name (target label) of a br_on_* expression.
func BrOnSetName(expr ExpressionRef, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenBrOnSetName((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// BrOnSetRef sets the ref expression of a br_on_* expression.
func BrOnSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenBrOnSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// BrOnSetCastType sets the cast type of a br_on_* expression.
func BrOnSetCastType(expr ExpressionRef, castType Type) {
	C.BinaryenBrOnSetCastType((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenType(castType))
}

// ---------------------------------------------------------------------------
// StructNew
// ---------------------------------------------------------------------------

// StructNewSetOperandAt sets the operand at the specified index of a struct.new expression.
func StructNewSetOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenStructNewSetOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// StructNewAppendOperand appends an operand to a struct.new, returning its insertion index.
func StructNewAppendOperand(expr ExpressionRef, operand ExpressionRef) Index {
	return Index(C.BinaryenStructNewAppendOperand(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	))
}

// StructNewInsertOperandAt inserts an operand at the specified index of a struct.new.
func StructNewInsertOperandAt(expr ExpressionRef, index Index, operand ExpressionRef) {
	C.BinaryenStructNewInsertOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(operand)),
	)
}

// StructNewRemoveOperandAt removes the operand at the specified index and returns it.
func StructNewRemoveOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenStructNewRemoveOperandAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// ---------------------------------------------------------------------------
// StructGet
// ---------------------------------------------------------------------------

// StructGetSetIndex sets the field index of a struct.get expression.
func StructGetSetIndex(expr ExpressionRef, index Index) {
	C.BinaryenStructGetSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))
}

// StructGetSetRef sets the ref expression of a struct.get expression.
func StructGetSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenStructGetSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// StructGetSetSigned sets whether a struct.get returns a signed value.
func StructGetSetSigned(expr ExpressionRef, signed bool) {
	C.BinaryenStructGetSetSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(signed))
}

// ---------------------------------------------------------------------------
// StructSet
// ---------------------------------------------------------------------------

// StructSetSetIndex sets the field index of a struct.set expression.
func StructSetSetIndex(expr ExpressionRef, index Index) {
	C.BinaryenStructSetSetIndex((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenIndex(index))
}

// StructSetSetRef sets the ref expression of a struct.set expression.
func StructSetSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenStructSetSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// StructSetSetValue sets the value expression of a struct.set expression.
func StructSetSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenStructSetSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// ArrayNew
// ---------------------------------------------------------------------------

// ArrayNewSetInit sets the init expression of an array.new expression.
func ArrayNewSetInit(expr ExpressionRef, init ExpressionRef) {
	C.BinaryenArrayNewSetInit(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(init)),
	)
}

// ArrayNewSetSize sets the size expression of an array.new expression.
func ArrayNewSetSize(expr ExpressionRef, size ExpressionRef) {
	C.BinaryenArrayNewSetSize(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
	)
}

// ---------------------------------------------------------------------------
// ArrayNewFixed
// ---------------------------------------------------------------------------

// ArrayNewFixedSetValueAt sets the value at the specified index of an array.new_fixed expression.
func ArrayNewFixedSetValueAt(expr ExpressionRef, index Index, value ExpressionRef) {
	C.BinaryenArrayNewFixedSetValueAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ArrayNewFixedAppendValue appends a value to an array.new_fixed, returning its insertion index.
func ArrayNewFixedAppendValue(expr ExpressionRef, value ExpressionRef) Index {
	return Index(C.BinaryenArrayNewFixedAppendValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	))
}

// ArrayNewFixedInsertValueAt inserts a value at the specified index of an array.new_fixed.
func ArrayNewFixedInsertValueAt(expr ExpressionRef, index Index, value ExpressionRef) {
	C.BinaryenArrayNewFixedInsertValueAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ArrayNewFixedRemoveValueAt removes the value at the specified index and returns it.
func ArrayNewFixedRemoveValueAt(expr ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(uintptr(unsafe.Pointer(C.BinaryenArrayNewFixedRemoveValueAt(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(index),
	))))
}

// ---------------------------------------------------------------------------
// ArrayGet
// ---------------------------------------------------------------------------

// ArrayGetSetRef sets the ref expression of an array.get expression.
func ArrayGetSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenArrayGetSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// ArrayGetSetIndex sets the index expression of an array.get expression.
func ArrayGetSetIndex(expr ExpressionRef, index ExpressionRef) {
	C.BinaryenArrayGetSetIndex(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
	)
}

// ArrayGetSetSigned sets whether an array.get returns a signed value.
func ArrayGetSetSigned(expr ExpressionRef, signed bool) {
	C.BinaryenArrayGetSetSigned((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cBool(signed))
}

// ---------------------------------------------------------------------------
// ArraySet
// ---------------------------------------------------------------------------

// ArraySetSetRef sets the ref expression of an array.set expression.
func ArraySetSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenArraySetSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// ArraySetSetIndex sets the index expression of an array.set expression.
func ArraySetSetIndex(expr ExpressionRef, index ExpressionRef) {
	C.BinaryenArraySetSetIndex(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
	)
}

// ArraySetSetValue sets the value expression of an array.set expression.
func ArraySetSetValue(expr ExpressionRef, value ExpressionRef) {
	C.BinaryenArraySetSetValue(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)
}

// ---------------------------------------------------------------------------
// ArrayLen
// ---------------------------------------------------------------------------

// ArrayLenSetRef sets the ref expression of an array.len expression.
func ArrayLenSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenArrayLenSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// ---------------------------------------------------------------------------
// ArrayCopy
// ---------------------------------------------------------------------------

// ArrayCopySetDestRef sets the destination ref expression of an array.copy expression.
func ArrayCopySetDestRef(expr ExpressionRef, destRef ExpressionRef) {
	C.BinaryenArrayCopySetDestRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(destRef)),
	)
}

// ArrayCopySetDestIndex sets the destination index expression of an array.copy expression.
func ArrayCopySetDestIndex(expr ExpressionRef, destIndex ExpressionRef) {
	C.BinaryenArrayCopySetDestIndex(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(destIndex)),
	)
}

// ArrayCopySetSrcRef sets the source ref expression of an array.copy expression.
func ArrayCopySetSrcRef(expr ExpressionRef, srcRef ExpressionRef) {
	C.BinaryenArrayCopySetSrcRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(srcRef)),
	)
}

// ArrayCopySetSrcIndex sets the source index expression of an array.copy expression.
func ArrayCopySetSrcIndex(expr ExpressionRef, srcIndex ExpressionRef) {
	C.BinaryenArrayCopySetSrcIndex(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(srcIndex)),
	)
}

// ArrayCopySetLength sets the length expression of an array.copy expression.
func ArrayCopySetLength(expr ExpressionRef, length ExpressionRef) {
	C.BinaryenArrayCopySetLength(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(length)),
	)
}

// ---------------------------------------------------------------------------
// StringNew
// ---------------------------------------------------------------------------

// StringNewSetOp sets the operation of a string.new expression.
func StringNewSetOp(expr ExpressionRef, op Op) {
	C.BinaryenStringNewSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// StringNewSetRef sets the ref expression of a string.new expression.
func StringNewSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenStringNewSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// StringNewSetStart sets the start expression of a string.new expression.
func StringNewSetStart(expr ExpressionRef, start ExpressionRef) {
	C.BinaryenStringNewSetStart(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(start)),
	)
}

// StringNewSetEnd sets the end expression of a string.new expression.
func StringNewSetEnd(expr ExpressionRef, end ExpressionRef) {
	C.BinaryenStringNewSetEnd(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(end)),
	)
}

// NOTE: BinaryenStringNewSetTry is declared in the header but not implemented
// in the version_123 library. Omitted.

// ---------------------------------------------------------------------------
// StringConst
// ---------------------------------------------------------------------------

// StringConstSetString sets the string value of a string.const expression.
func StringConstSetString(expr ExpressionRef, str string) {
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenStringConstSetString((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), cs)
}

// ---------------------------------------------------------------------------
// StringMeasure
// ---------------------------------------------------------------------------

// StringMeasureSetOp sets the operation of a string.measure expression.
func StringMeasureSetOp(expr ExpressionRef, op Op) {
	C.BinaryenStringMeasureSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// StringMeasureSetRef sets the ref expression of a string.measure expression.
func StringMeasureSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenStringMeasureSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// ---------------------------------------------------------------------------
// StringEncode
// ---------------------------------------------------------------------------

// StringEncodeSetOp sets the operation of a string.encode expression.
func StringEncodeSetOp(expr ExpressionRef, op Op) {
	C.BinaryenStringEncodeSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// StringEncodeSetStr sets the string expression of a string.encode expression.
func StringEncodeSetStr(expr ExpressionRef, str ExpressionRef) {
	C.BinaryenStringEncodeSetStr(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(str)),
	)
}

// StringEncodeSetArray sets the array expression of a string.encode expression.
func StringEncodeSetArray(expr ExpressionRef, array ExpressionRef) {
	C.BinaryenStringEncodeSetArray(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(array)),
	)
}

// StringEncodeSetStart sets the start expression of a string.encode expression.
func StringEncodeSetStart(expr ExpressionRef, start ExpressionRef) {
	C.BinaryenStringEncodeSetStart(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(start)),
	)
}

// ---------------------------------------------------------------------------
// StringConcat
// ---------------------------------------------------------------------------

// StringConcatSetLeft sets the left expression of a string.concat expression.
func StringConcatSetLeft(expr ExpressionRef, left ExpressionRef) {
	C.BinaryenStringConcatSetLeft(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
	)
}

// StringConcatSetRight sets the right expression of a string.concat expression.
func StringConcatSetRight(expr ExpressionRef, right ExpressionRef) {
	C.BinaryenStringConcatSetRight(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)
}

// ---------------------------------------------------------------------------
// StringEq
// ---------------------------------------------------------------------------

// StringEqSetOp sets the operation of a string.eq expression.
func StringEqSetOp(expr ExpressionRef, op Op) {
	C.BinaryenStringEqSetOp((C.BinaryenExpressionRef)(unsafe.Pointer(expr)), C.BinaryenOp(op))
}

// StringEqSetLeft sets the left expression of a string.eq expression.
func StringEqSetLeft(expr ExpressionRef, left ExpressionRef) {
	C.BinaryenStringEqSetLeft(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
	)
}

// StringEqSetRight sets the right expression of a string.eq expression.
func StringEqSetRight(expr ExpressionRef, right ExpressionRef) {
	C.BinaryenStringEqSetRight(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)
}

// ---------------------------------------------------------------------------
// StringWTF16Get
// ---------------------------------------------------------------------------

// StringWTF16GetSetRef sets the ref expression of a string.wtf16.get expression.
func StringWTF16GetSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenStringWTF16GetSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// StringWTF16GetSetPos sets the position expression of a string.wtf16.get expression.
func StringWTF16GetSetPos(expr ExpressionRef, pos ExpressionRef) {
	C.BinaryenStringWTF16GetSetPos(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(pos)),
	)
}

// ---------------------------------------------------------------------------
// StringSliceWTF
// ---------------------------------------------------------------------------

// StringSliceWTFSetRef sets the ref expression of a string.slice expression.
func StringSliceWTFSetRef(expr ExpressionRef, ref ExpressionRef) {
	C.BinaryenStringSliceWTFSetRef(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)
}

// StringSliceWTFSetStart sets the start expression of a string.slice expression.
func StringSliceWTFSetStart(expr ExpressionRef, start ExpressionRef) {
	C.BinaryenStringSliceWTFSetStart(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(start)),
	)
}

// StringSliceWTFSetEnd sets the end expression of a string.slice expression.
func StringSliceWTFSetEnd(expr ExpressionRef, end ExpressionRef) {
	C.BinaryenStringSliceWTFSetEnd(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(end)),
	)
}
