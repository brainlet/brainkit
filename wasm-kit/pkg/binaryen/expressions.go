package binaryen

/*
#include "binaryen-c.h"
*/
import "C"
import "unsafe"

// ---------------------------------------------------------------------------
// Helpers for converting Go slices to C arrays
// ---------------------------------------------------------------------------

// cExprSlice converts a Go slice of ExpressionRef to a C array.
// Returns (pointer, length) suitable for passing to C functions.
func cExprSlice(exprs []ExpressionRef) (*C.BinaryenExpressionRef, C.BinaryenIndex) {
	if len(exprs) == 0 {
		return nil, 0
	}
	buf := make([]C.BinaryenExpressionRef, len(exprs))
	for i, e := range exprs {
		buf[i] = (C.BinaryenExpressionRef)(unsafe.Pointer(e))
	}
	return &buf[0], C.BinaryenIndex(len(exprs))
}

// cStringSlice converts a Go slice of strings to a C array of char pointers.
// Uses the module's string pool for caching. Returns (pointer, length).
func (m *Module) cStringSlice(strs []string) (**C.char, C.BinaryenIndex) {
	if len(strs) == 0 {
		return nil, 0
	}
	buf := make([]*C.char, len(strs))
	for i, s := range strs {
		buf[i] = m.str(s)
	}
	return &buf[0], C.BinaryenIndex(len(strs))
}

// ---------------------------------------------------------------------------
// Control flow
// ---------------------------------------------------------------------------

// Block creates a block expression. name can be empty. Pass TypeAuto() for
// type to let Binaryen infer it.
func (m *Module) Block(name string, children []ExpressionRef, typ Type) ExpressionRef {
	ptr, n := cExprSlice(children)
	return ExpressionRef(unsafe.Pointer(C.BinaryenBlock(
		m.ref, m.str(name), ptr, n, C.BinaryenType(typ),
	)))
}

// If creates an if expression. ifFalse can be 0 for no else branch.
func (m *Module) If(condition, ifTrue, ifFalse ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenIf(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(condition)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifTrue)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifFalse)),
	)))
}

// Loop creates a loop expression. name is the label for the loop.
func (m *Module) Loop(name string, body ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLoop(
		m.ref, m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(body)),
	)))
}

// Break creates a break (br) expression. condition and value can be 0.
func (m *Module) Break(name string, condition, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBreak(
		m.ref, m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(condition)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// Switch creates a switch (br_table) expression. value can be 0.
func (m *Module) Switch(names []string, defaultName string, condition, value ExpressionRef) ExpressionRef {
	cNames, numNames := m.cStringSlice(names)
	return ExpressionRef(unsafe.Pointer(C.BinaryenSwitch(
		m.ref, cNames, numNames, m.str(defaultName),
		(C.BinaryenExpressionRef)(unsafe.Pointer(condition)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// ---------------------------------------------------------------------------
// Function calls
// ---------------------------------------------------------------------------

// Call creates a direct function call expression.
func (m *Module) Call(target string, operands []ExpressionRef, returnType Type) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenCall(
		m.ref, m.str(target), ptr, n, C.BinaryenType(returnType),
	)))
}

// CallIndirect creates an indirect function call expression through a table.
func (m *Module) CallIndirect(table string, target ExpressionRef, operands []ExpressionRef, params, results Type) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenCallIndirect(
		m.ref, m.str(table),
		(C.BinaryenExpressionRef)(unsafe.Pointer(target)),
		ptr, n,
		C.BinaryenType(params), C.BinaryenType(results),
	)))
}

// ReturnCall creates a tail-call expression (direct).
func (m *Module) ReturnCall(target string, operands []ExpressionRef, returnType Type) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenReturnCall(
		m.ref, m.str(target), ptr, n, C.BinaryenType(returnType),
	)))
}

// ReturnCallIndirect creates a tail-call expression (indirect, through a table).
func (m *Module) ReturnCallIndirect(table string, target ExpressionRef, operands []ExpressionRef, params, results Type) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenReturnCallIndirect(
		m.ref, m.str(table),
		(C.BinaryenExpressionRef)(unsafe.Pointer(target)),
		ptr, n,
		C.BinaryenType(params), C.BinaryenType(results),
	)))
}

// ---------------------------------------------------------------------------
// Local and global variables
// ---------------------------------------------------------------------------

// LocalGet creates a local.get expression.
func (m *Module) LocalGet(index Index, typ Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLocalGet(
		m.ref, C.BinaryenIndex(index), C.BinaryenType(typ),
	)))
}

// LocalSet creates a local.set expression.
func (m *Module) LocalSet(index Index, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLocalSet(
		m.ref, C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// LocalTee creates a local.tee expression (set and return the value).
func (m *Module) LocalTee(index Index, value ExpressionRef, typ Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLocalTee(
		m.ref, C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
		C.BinaryenType(typ),
	)))
}

// GlobalGet creates a global.get expression.
func (m *Module) GlobalGet(name string, typ Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenGlobalGet(
		m.ref, m.str(name), C.BinaryenType(typ),
	)))
}

// GlobalSet creates a global.set expression.
func (m *Module) GlobalSet(name string, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenGlobalSet(
		m.ref, m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// ---------------------------------------------------------------------------
// Memory access
// ---------------------------------------------------------------------------

// Load creates a memory load expression. align can be 0 for natural alignment.
func (m *Module) Load(bytes uint32, signed bool, offset, align uint32, typ Type, ptr ExpressionRef, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenLoad(
		m.ref, C.uint32_t(bytes), cBool(signed),
		C.uint32_t(offset), C.uint32_t(align),
		C.BinaryenType(typ),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		m.str(memoryName),
	)))
}

// Store creates a memory store expression. align can be 0 for natural alignment.
func (m *Module) Store(bytes, offset, align uint32, ptr, value ExpressionRef, typ Type, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStore(
		m.ref, C.uint32_t(bytes), C.uint32_t(offset), C.uint32_t(align),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
		C.BinaryenType(typ),
		m.str(memoryName),
	)))
}

// MemorySize creates a memory.size expression.
func (m *Module) MemorySize(memoryName string, memoryIs64 bool) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemorySize(
		m.ref, m.str(memoryName), cBool(memoryIs64),
	)))
}

// MemoryGrow creates a memory.grow expression.
func (m *Module) MemoryGrow(delta ExpressionRef, memoryName string, memoryIs64 bool) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryGrow(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(delta)),
		m.str(memoryName), cBool(memoryIs64),
	)))
}

// MemoryInit creates a memory.init expression.
func (m *Module) MemoryInit(segment string, dest, offset, size ExpressionRef, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryInit(
		m.ref, m.str(segment),
		(C.BinaryenExpressionRef)(unsafe.Pointer(dest)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(offset)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
		m.str(memoryName),
	)))
}

// DataDrop creates a data.drop expression.
func (m *Module) DataDrop(segment string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenDataDrop(
		m.ref, m.str(segment),
	)))
}

// MemoryCopy creates a memory.copy expression.
func (m *Module) MemoryCopy(dest, source, size ExpressionRef, destMemory, sourceMemory string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryCopy(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(dest)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(source)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
		m.str(destMemory), m.str(sourceMemory),
	)))
}

// MemoryFill creates a memory.fill expression.
func (m *Module) MemoryFill(dest, value, size ExpressionRef, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenMemoryFill(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(dest)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
		m.str(memoryName),
	)))
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// Const creates a constant expression from a BinaryenLiteral.
func (m *Module) Const(value C.struct_BinaryenLiteral) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(m.ref, value)))
}

// ConstInt32 creates an i32 constant.
func (m *Module) ConstInt32(x int32) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(
		m.ref, C.BinaryenLiteralInt32(C.int32_t(x)),
	)))
}

// ConstInt64 creates an i64 constant.
func (m *Module) ConstInt64(x int64) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(
		m.ref, C.BinaryenLiteralInt64(C.int64_t(x)),
	)))
}

// ConstFloat32 creates an f32 constant.
func (m *Module) ConstFloat32(x float32) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(
		m.ref, C.BinaryenLiteralFloat32(C.float(x)),
	)))
}

// ConstFloat64 creates an f64 constant.
func (m *Module) ConstFloat64(x float64) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(
		m.ref, C.BinaryenLiteralFloat64(C.double(x)),
	)))
}

// ConstVec128 creates a v128 constant from 16 bytes.
func (m *Module) ConstVec128(x [16]byte) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(
		m.ref, C.BinaryenLiteralVec128((*C.uint8_t)(&x[0])),
	)))
}

// ConstFloat32Bits creates an f32 constant from raw integer bits.
func (m *Module) ConstFloat32Bits(x int32) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(
		m.ref, C.BinaryenLiteralFloat32Bits(C.int32_t(x)),
	)))
}

// ConstFloat64Bits creates an f64 constant from raw integer bits.
func (m *Module) ConstFloat64Bits(x int64) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenConst(
		m.ref, C.BinaryenLiteralFloat64Bits(C.int64_t(x)),
	)))
}

// ---------------------------------------------------------------------------
// Arithmetic and logical operations
// ---------------------------------------------------------------------------

// Unary creates a unary operation expression.
func (m *Module) Unary(op Op, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenUnary(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// Binary creates a binary operation expression.
func (m *Module) Binary(op Op, left, right ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBinary(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)))
}

// Select creates a select (ternary) expression.
func (m *Module) Select(condition, ifTrue, ifFalse ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSelect(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(condition)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifTrue)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ifFalse)),
	)))
}

// Drop creates a drop expression (discards a value).
func (m *Module) Drop(value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenDrop(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// Return creates a return expression. value can be 0 for void return.
func (m *Module) Return(value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenReturn(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// Nop creates a nop expression (no operation).
func (m *Module) Nop() ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenNop(m.ref)))
}

// Unreachable creates an unreachable expression (trap).
func (m *Module) Unreachable() ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenUnreachable(m.ref)))
}

// ---------------------------------------------------------------------------
// Atomics
// ---------------------------------------------------------------------------

// AtomicLoad creates an atomic load expression.
func (m *Module) AtomicLoad(bytes, offset uint32, typ Type, ptr ExpressionRef, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicLoad(
		m.ref, C.uint32_t(bytes), C.uint32_t(offset),
		C.BinaryenType(typ),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		m.str(memoryName),
	)))
}

// AtomicStore creates an atomic store expression.
func (m *Module) AtomicStore(bytes, offset uint32, ptr, value ExpressionRef, typ Type, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicStore(
		m.ref, C.uint32_t(bytes), C.uint32_t(offset),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
		C.BinaryenType(typ),
		m.str(memoryName),
	)))
}

// AtomicRMW creates an atomic read-modify-write expression.
func (m *Module) AtomicRMW(op Op, bytes, offset Index, ptr, value ExpressionRef, typ Type, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicRMW(
		m.ref, C.BinaryenOp(op),
		C.BinaryenIndex(bytes), C.BinaryenIndex(offset),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
		C.BinaryenType(typ),
		m.str(memoryName),
	)))
}

// AtomicCmpxchg creates an atomic compare-and-exchange expression.
func (m *Module) AtomicCmpxchg(bytes, offset Index, ptr, expected, replacement ExpressionRef, typ Type, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicCmpxchg(
		m.ref, C.BinaryenIndex(bytes), C.BinaryenIndex(offset),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(expected)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(replacement)),
		C.BinaryenType(typ),
		m.str(memoryName),
	)))
}

// AtomicWait creates an atomic wait expression.
func (m *Module) AtomicWait(ptr, expected, timeout ExpressionRef, typ Type, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicWait(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(expected)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(timeout)),
		C.BinaryenType(typ),
		m.str(memoryName),
	)))
}

// AtomicNotify creates an atomic notify expression.
func (m *Module) AtomicNotify(ptr, notifyCount ExpressionRef, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicNotify(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(notifyCount)),
		m.str(memoryName),
	)))
}

// AtomicFence creates an atomic fence expression.
func (m *Module) AtomicFence() ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenAtomicFence(m.ref)))
}

// ---------------------------------------------------------------------------
// SIMD operations
// ---------------------------------------------------------------------------

// SIMDExtract creates a SIMD extract lane expression.
func (m *Module) SIMDExtract(op Op, vec ExpressionRef, index uint8) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDExtract(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
		C.uint8_t(index),
	)))
}

// SIMDReplace creates a SIMD replace lane expression.
func (m *Module) SIMDReplace(op Op, vec ExpressionRef, index uint8, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDReplace(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
		C.uint8_t(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// SIMDShuffle creates a SIMD shuffle expression with a 16-byte mask.
func (m *Module) SIMDShuffle(left, right ExpressionRef, mask [16]byte) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDShuffle(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
		(*C.uint8_t)(&mask[0]),
	)))
}

// SIMDTernary creates a SIMD ternary expression (e.g. bitselect).
func (m *Module) SIMDTernary(op Op, a, b, c ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDTernary(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(a)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(b)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(c)),
	)))
}

// SIMDShift creates a SIMD shift expression.
func (m *Module) SIMDShift(op Op, vec, shift ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDShift(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(shift)),
	)))
}

// SIMDLoad creates a SIMD load expression (e.g. v128.load8_splat).
func (m *Module) SIMDLoad(op Op, offset, align uint32, ptr ExpressionRef, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDLoad(
		m.ref, C.BinaryenOp(op),
		C.uint32_t(offset), C.uint32_t(align),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		m.str(memoryName),
	)))
}

// SIMDLoadStoreLane creates a SIMD load/store lane expression.
func (m *Module) SIMDLoadStoreLane(op Op, offset, align uint32, index uint8, ptr, vec ExpressionRef, memoryName string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenSIMDLoadStoreLane(
		m.ref, C.BinaryenOp(op),
		C.uint32_t(offset), C.uint32_t(align),
		C.uint8_t(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(vec)),
		m.str(memoryName),
	)))
}

// ---------------------------------------------------------------------------
// Reference types
// ---------------------------------------------------------------------------

// RefNull creates a ref.null expression of the given type.
func (m *Module) RefNull(typ Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefNull(
		m.ref, C.BinaryenType(typ),
	)))
}

// RefIsNull creates a ref.is_null expression.
func (m *Module) RefIsNull(value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefIsNull(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// RefAs creates a ref.as expression (e.g. ref.as_non_null).
func (m *Module) RefAs(op Op, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefAs(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// RefFunc creates a ref.func expression.
func (m *Module) RefFunc(name string, typ HeapType) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefFunc(
		m.ref, m.str(name), C.BinaryenHeapType(typ),
	)))
}

// RefEq creates a ref.eq expression.
func (m *Module) RefEq(left, right ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefEq(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)))
}

// RefTest creates a ref.test expression.
func (m *Module) RefTest(ref ExpressionRef, castType Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefTest(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		C.BinaryenType(castType),
	)))
}

// RefCast creates a ref.cast expression.
func (m *Module) RefCast(ref ExpressionRef, typ Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefCast(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		C.BinaryenType(typ),
	)))
}

// BrOn creates a br_on expression (br_on_cast, br_on_null, etc.).
func (m *Module) BrOn(op Op, name string, ref ExpressionRef, castType Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenBrOn(
		m.ref, C.BinaryenOp(op), m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		C.BinaryenType(castType),
	)))
}

// ---------------------------------------------------------------------------
// Table operations
// ---------------------------------------------------------------------------

// TableGet creates a table.get expression.
func (m *Module) TableGet(name string, index ExpressionRef, typ Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableGet(
		m.ref, m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
		C.BinaryenType(typ),
	)))
}

// TableSet creates a table.set expression.
func (m *Module) TableSet(name string, index, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableSet(
		m.ref, m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// TableSize creates a table.size expression.
func (m *Module) TableSize(name string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableSize(
		m.ref, m.str(name),
	)))
}

// TableGrow creates a table.grow expression.
func (m *Module) TableGrow(name string, value, delta ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTableGrow(
		m.ref, m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(delta)),
	)))
}

// ---------------------------------------------------------------------------
// Exception handling
// ---------------------------------------------------------------------------

// Try creates a try expression. name can be empty. delegateTarget should be
// empty for try-catch (non-delegate).
func (m *Module) Try(name string, body ExpressionRef, catchTags []string, catchBodies []ExpressionRef, delegateTarget string) ExpressionRef {
	cTags, nTags := m.cStringSlice(catchTags)
	cBodies, nBodies := cExprSlice(catchBodies)
	return ExpressionRef(unsafe.Pointer(C.BinaryenTry(
		m.ref, m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(body)),
		cTags, nTags,
		cBodies, nBodies,
		m.str(delegateTarget),
	)))
}

// Throw creates a throw expression.
func (m *Module) Throw(tag string, operands []ExpressionRef) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenThrow(
		m.ref, m.str(tag), ptr, n,
	)))
}

// Rethrow creates a rethrow expression.
func (m *Module) Rethrow(target string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRethrow(
		m.ref, m.str(target),
	)))
}

// ---------------------------------------------------------------------------
// Tuples
// ---------------------------------------------------------------------------

// TupleMake creates a tuple.make expression.
func (m *Module) TupleMake(operands []ExpressionRef) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenTupleMake(
		m.ref, ptr, n,
	)))
}

// TupleExtract creates a tuple.extract expression.
func (m *Module) TupleExtract(tuple ExpressionRef, index Index) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenTupleExtract(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(tuple)),
		C.BinaryenIndex(index),
	)))
}

// Pop creates a pop expression (used in catch blocks).
func (m *Module) Pop(typ Type) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenPop(
		m.ref, C.BinaryenType(typ),
	)))
}

// ---------------------------------------------------------------------------
// GC — i31 references
// ---------------------------------------------------------------------------

// RefI31 creates a ref.i31 expression.
func (m *Module) RefI31(value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenRefI31(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// I31Get creates an i31.get expression.
func (m *Module) I31Get(i31 ExpressionRef, signed bool) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenI31Get(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(i31)),
		cBool(signed),
	)))
}

// ---------------------------------------------------------------------------
// GC — call_ref
// ---------------------------------------------------------------------------

// CallRef creates a call_ref expression.
func (m *Module) CallRef(target ExpressionRef, operands []ExpressionRef, typ Type, isReturn bool) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenCallRef(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(target)),
		ptr, n,
		C.BinaryenType(typ),
		cBool(isReturn),
	)))
}

// ---------------------------------------------------------------------------
// GC — structs
// ---------------------------------------------------------------------------

// StructNew creates a struct.new expression. Pass nil operands for
// struct.new_default.
func (m *Module) StructNew(operands []ExpressionRef, typ HeapType) ExpressionRef {
	ptr, n := cExprSlice(operands)
	return ExpressionRef(unsafe.Pointer(C.BinaryenStructNew(
		m.ref, ptr, n, C.BinaryenHeapType(typ),
	)))
}

// StructGet creates a struct.get expression.
func (m *Module) StructGet(index Index, ref ExpressionRef, typ Type, signed bool) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStructGet(
		m.ref, C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		C.BinaryenType(typ),
		cBool(signed),
	)))
}

// StructSet creates a struct.set expression.
func (m *Module) StructSet(index Index, ref, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStructSet(
		m.ref, C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// ---------------------------------------------------------------------------
// GC — arrays
// ---------------------------------------------------------------------------

// ArrayNew creates an array.new expression. init can be 0 for array.new_default.
func (m *Module) ArrayNew(typ HeapType, size, init ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayNew(
		m.ref, C.BinaryenHeapType(typ),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(init)),
	)))
}

// ArrayNewData creates an array.new_data expression.
func (m *Module) ArrayNewData(typ HeapType, name string, offset, size ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayNewData(
		m.ref, C.BinaryenHeapType(typ), m.str(name),
		(C.BinaryenExpressionRef)(unsafe.Pointer(offset)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(size)),
	)))
}

// ArrayNewFixed creates an array.new_fixed expression.
func (m *Module) ArrayNewFixed(typ HeapType, values []ExpressionRef) ExpressionRef {
	ptr, n := cExprSlice(values)
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayNewFixed(
		m.ref, C.BinaryenHeapType(typ), ptr, n,
	)))
}

// ArrayGet creates an array.get expression.
func (m *Module) ArrayGet(ref, index ExpressionRef, typ Type, signed bool) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayGet(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
		C.BinaryenType(typ),
		cBool(signed),
	)))
}

// ArraySet creates an array.set expression.
func (m *Module) ArraySet(ref, index, value ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArraySet(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(index)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	)))
}

// ArrayLen creates an array.len expression.
func (m *Module) ArrayLen(ref ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayLen(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)))
}

// ArrayCopy creates an array.copy expression.
func (m *Module) ArrayCopy(destRef, destIndex, srcRef, srcIndex, length ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenArrayCopy(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(destRef)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(destIndex)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(srcRef)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(srcIndex)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(length)),
	)))
}

// ---------------------------------------------------------------------------
// Strings
// ---------------------------------------------------------------------------

// StringNew creates a string.new expression.
func (m *Module) StringNew(op Op, ref, start, end ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringNew(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(start)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(end)),
	)))
}

// StringConst creates a string.const expression.
func (m *Module) StringConst(name string) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringConst(
		m.ref, m.str(name),
	)))
}

// StringMeasure creates a string.measure expression.
func (m *Module) StringMeasure(op Op, ref ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringMeasure(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
	)))
}

// StringEncode creates a string.encode expression.
func (m *Module) StringEncode(op Op, ref, ptr, start ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringEncode(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(ptr)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(start)),
	)))
}

// StringConcat creates a string.concat expression.
func (m *Module) StringConcat(left, right ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringConcat(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)))
}

// StringEq creates a string.eq expression.
func (m *Module) StringEq(op Op, left, right ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringEq(
		m.ref, C.BinaryenOp(op),
		(C.BinaryenExpressionRef)(unsafe.Pointer(left)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(right)),
	)))
}

// StringWTF16Get creates a stringview_wtf16.get_codeunit expression.
func (m *Module) StringWTF16Get(ref, pos ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringWTF16Get(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(pos)),
	)))
}

// StringSliceWTF creates a stringview_wtf8.slice or stringview_wtf16.slice expression.
func (m *Module) StringSliceWTF(ref, start, end ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.BinaryenStringSliceWTF(
		m.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(ref)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(start)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(end)),
	)))
}
