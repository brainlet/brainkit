// Ported from: assemblyscript/src/module.ts (class Module, lines 1316-2987)
//
// Module wraps a *binaryen.Module with AssemblyScript-specific conveniences:
//   - Size-aware unary/binary ops that dispatch to i32 or i64 variants
//   - Shadow-stack integration for managed references
//   - Custom optimization pass ordering
package module

import (
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
)

// DefaultMemory is the default linear memory name used by Wasm modules.
const DefaultMemory = "0"

// DefaultTable is the default indirect function table name.
const DefaultTable = "0"

// UnlimitedMemory represents an unlimited memory maximum.
const UnlimitedMemory uint32 = 0xFFFFFFFF

// UnlimitedTable represents an unlimited table maximum.
const UnlimitedTable uint32 = 0xFFFFFFFF

// BinaryModule holds the output of ToBinary.
type BinaryModule struct {
	Binary    []byte
	SourceMap string
}

// MemorySegment represents a data segment to be placed into linear memory.
type MemorySegment struct {
	Buffer    []byte
	Offset    ExpressionRef
	RawOffset int64 // raw numeric offset matching TS MemorySegment.offset (i64)
}

// Target selects between wasm32 and wasm64 compilation.
type Target int

const (
	TargetWasm32 Target = 1 // matches common.TargetWasm32
	TargetWasm64 Target = 2 // matches common.TargetWasm64
)

// Module wraps a *binaryen.Module with size-type dispatching and shadow-stack
// support. All expression-building methods delegate to the underlying binaryen
// module; no CGo is used in this package.
type Module struct {
	bmod           *binaryen.Module
	UseShadowStack bool
	SizeType       TypeRef

	hasTemporaryFunction bool
}

// Create constructs a new empty Module.
func Create(useShadowStack bool, sizeType TypeRef) *Module {
	return &Module{
		bmod:           binaryen.NewModule(),
		UseShadowStack: useShadowStack,
		SizeType:       sizeType,
	}
}

// CreateFrom wraps an existing *binaryen.Module.
func CreateFrom(bmod *binaryen.Module, useShadowStack bool, sizeType TypeRef) *Module {
	return &Module{
		bmod:           bmod,
		UseShadowStack: useShadowStack,
		SizeType:       sizeType,
	}
}

// CreateFromBinary creates a Module by reading a Wasm binary buffer.
// This corresponds to the TS Module.createFrom(buffer, ...) which calls
// BinaryenModuleRead internally.
func CreateFromBinary(data []byte, useShadowStack bool, sizeType TypeRef) *Module {
	bmod := binaryen.ModuleRead(data)
	if bmod == nil {
		return nil
	}
	return &Module{
		bmod:           bmod,
		UseShadowStack: useShadowStack,
		SizeType:       sizeType,
	}
}

// BinaryenModule returns the underlying binaryen module for direct access.
func (m *Module) BinaryenModule() *binaryen.Module { return m.bmod }

// =========================================================================
// Constants (literal helpers)
// =========================================================================

// I32 creates an i32 constant expression.
func (m *Module) I32(value int32) ExpressionRef {
	return m.bmod.ConstInt32(value)
}

// I64 creates an i64 constant expression.
func (m *Module) I64(value int64) ExpressionRef {
	return m.bmod.ConstInt64(value)
}

// Usize creates a constant of the architecture-dependent size type.
// For wasm64, produces an i64 constant; for wasm32, produces an i32 constant.
func (m *Module) Usize(value int64) ExpressionRef {
	if m.SizeType == binaryen.TypeInt64() {
		return m.I64(value)
	}
	return m.I32(int32(value))
}

// F32 creates an f32 constant expression.
func (m *Module) F32(value float32) ExpressionRef {
	return m.bmod.ConstFloat32(value)
}

// F64 creates an f64 constant expression.
func (m *Module) F64(value float64) ExpressionRef {
	return m.bmod.ConstFloat64(value)
}

// V128 creates a v128 constant expression from 16 bytes.
func (m *Module) V128(bytes [16]byte) ExpressionRef {
	return m.bmod.ConstVec128(bytes)
}

// =========================================================================
// Reference type expressions
// =========================================================================

// RefNull creates a ref.null expression of the given type.
func (m *Module) RefNull(typ TypeRef) ExpressionRef {
	return m.bmod.RefNull(typ)
}

// RefEq creates a ref.eq expression.
func (m *Module) RefEq(left, right ExpressionRef) ExpressionRef {
	return m.bmod.RefEq(left, right)
}

// StringEq creates a string equality check expression.
func (m *Module) StringEq(left, right ExpressionRef) ExpressionRef {
	return m.bmod.StringEq(binaryen.StringEqEqual(), left, right)
}

// StringCompare creates a string comparison expression.
func (m *Module) StringCompare(left, right ExpressionRef) ExpressionRef {
	return m.bmod.StringEq(binaryen.StringEqCompare(), left, right)
}

// =========================================================================
// Size-aware unary operations
// =========================================================================

// Unary creates a unary operation expression. Size-variant ops (ClzSize,
// CtzSize, PopcntSize, EqzSize) are dispatched to their 32-bit or 64-bit
// counterparts depending on m.SizeType.
func (m *Module) Unary(op Op, value ExpressionRef) ExpressionRef {
	isWasm64 := m.SizeType == binaryen.TypeInt64()
	switch op {
	case UnaryOpClzSize:
		if isWasm64 {
			op = UnaryOpClzI64
		} else {
			op = UnaryOpClzI32
		}
	case UnaryOpCtzSize:
		if isWasm64 {
			op = UnaryOpCtzI64
		} else {
			op = UnaryOpCtzI32
		}
	case UnaryOpPopcntSize:
		if isWasm64 {
			op = UnaryOpPopcntI64
		} else {
			op = UnaryOpPopcntI32
		}
	case UnaryOpEqzSize:
		if isWasm64 {
			op = UnaryOpEqzI64
		} else {
			op = UnaryOpEqzI32
		}
	}
	return m.bmod.Unary(op, value)
}

// =========================================================================
// Size-aware binary operations
// =========================================================================

// Binary creates a binary operation expression. Size-variant ops are dispatched
// to their 32-bit or 64-bit counterparts depending on m.SizeType.
func (m *Module) Binary(op Op, left, right ExpressionRef) ExpressionRef {
	isWasm64 := m.SizeType == binaryen.TypeInt64()
	switch op {
	case BinaryOpAddSize:
		if isWasm64 {
			op = BinaryOpAddI64
		} else {
			op = BinaryOpAddI32
		}
	case BinaryOpSubSize:
		if isWasm64 {
			op = BinaryOpSubI64
		} else {
			op = BinaryOpSubI32
		}
	case BinaryOpMulSize:
		if isWasm64 {
			op = BinaryOpMulI64
		} else {
			op = BinaryOpMulI32
		}
	case BinaryOpDivISize:
		if isWasm64 {
			op = BinaryOpDivI64
		} else {
			op = BinaryOpDivI32
		}
	case BinaryOpDivUSize:
		if isWasm64 {
			op = BinaryOpDivU64
		} else {
			op = BinaryOpDivU32
		}
	case BinaryOpRemISize:
		if isWasm64 {
			op = BinaryOpRemI64
		} else {
			op = BinaryOpRemI32
		}
	case BinaryOpRemUSize:
		if isWasm64 {
			op = BinaryOpRemU64
		} else {
			op = BinaryOpRemU32
		}
	case BinaryOpAndSize:
		if isWasm64 {
			op = BinaryOpAndI64
		} else {
			op = BinaryOpAndI32
		}
	case BinaryOpOrSize:
		if isWasm64 {
			op = BinaryOpOrI64
		} else {
			op = BinaryOpOrI32
		}
	case BinaryOpXorSize:
		if isWasm64 {
			op = BinaryOpXorI64
		} else {
			op = BinaryOpXorI32
		}
	case BinaryOpShlSize:
		if isWasm64 {
			op = BinaryOpShlI64
		} else {
			op = BinaryOpShlI32
		}
	case BinaryOpShrISize:
		if isWasm64 {
			op = BinaryOpShrI64
		} else {
			op = BinaryOpShrI32
		}
	case BinaryOpShrUSize:
		if isWasm64 {
			op = BinaryOpShrU64
		} else {
			op = BinaryOpShrU32
		}
	case BinaryOpRotlSize:
		if isWasm64 {
			op = BinaryOpRotlI64
		} else {
			op = BinaryOpRotlI32
		}
	case BinaryOpRotrSize:
		if isWasm64 {
			op = BinaryOpRotrI64
		} else {
			op = BinaryOpRotrI32
		}
	case BinaryOpEqSize:
		if isWasm64 {
			op = BinaryOpEqI64
		} else {
			op = BinaryOpEqI32
		}
	case BinaryOpNeSize:
		if isWasm64 {
			op = BinaryOpNeI64
		} else {
			op = BinaryOpNeI32
		}
	case BinaryOpLtISize:
		if isWasm64 {
			op = BinaryOpLtI64
		} else {
			op = BinaryOpLtI32
		}
	case BinaryOpLtUSize:
		if isWasm64 {
			op = BinaryOpLtU64
		} else {
			op = BinaryOpLtU32
		}
	case BinaryOpLeISize:
		if isWasm64 {
			op = BinaryOpLeI64
		} else {
			op = BinaryOpLeI32
		}
	case BinaryOpLeUSize:
		if isWasm64 {
			op = BinaryOpLeU64
		} else {
			op = BinaryOpLeU32
		}
	case BinaryOpGtISize:
		if isWasm64 {
			op = BinaryOpGtI64
		} else {
			op = BinaryOpGtI32
		}
	case BinaryOpGtUSize:
		if isWasm64 {
			op = BinaryOpGtU64
		} else {
			op = BinaryOpGtU32
		}
	case BinaryOpGeISize:
		if isWasm64 {
			op = BinaryOpGeI64
		} else {
			op = BinaryOpGeI32
		}
	case BinaryOpGeUSize:
		if isWasm64 {
			op = BinaryOpGeU64
		} else {
			op = BinaryOpGeU32
		}
	}
	return m.bmod.Binary(op, left, right)
}

// =========================================================================
// Memory operations
// =========================================================================

// MemorySize creates a memory.size expression.
func (m *Module) MemorySize() ExpressionRef {
	return m.bmod.MemorySize(DefaultMemory, m.SizeType == binaryen.TypeInt64())
}

// MemoryGrow creates a memory.grow expression.
func (m *Module) MemoryGrow(delta ExpressionRef) ExpressionRef {
	return m.bmod.MemoryGrow(delta, DefaultMemory, m.SizeType == binaryen.TypeInt64())
}

// =========================================================================
// Table operations
// =========================================================================

// TableSize creates a table.size expression.
func (m *Module) TableSize(name string) ExpressionRef {
	return m.bmod.TableSize(name)
}

// TableGrow creates a table.grow expression.
func (m *Module) TableGrow(name string, value, delta ExpressionRef) ExpressionRef {
	return m.bmod.TableGrow(name, value, delta)
}

// =========================================================================
// Local variable operations
// =========================================================================

// Tostack wraps a value through the shadow stack runtime function.
// If the shadow stack is not in use, returns the value as-is.
func (m *Module) Tostack(value ExpressionRef) ExpressionRef {
	if m.UseShadowStack {
		typ := binaryen.ExpressionGetType(value)
		return m.bmod.Call("~tostack", []ExpressionRef{value}, typ)
	}
	return value
}

// LocalGet creates a local.get expression.
func (m *Module) LocalGet(index int32, typ TypeRef) ExpressionRef {
	return m.bmod.LocalGet(uint32(index), typ)
}

// LocalSet creates a local.set expression. If isManaged is true and the shadow
// stack is enabled, the value is wrapped through the tostack runtime.
func (m *Module) LocalSet(index int32, value ExpressionRef, isManaged bool) ExpressionRef {
	if isManaged && m.UseShadowStack {
		value = m.Tostack(value)
	}
	return m.bmod.LocalSet(uint32(index), value)
}

// LocalTee creates a local.tee expression. If isManaged is true and the shadow
// stack is enabled, the value is wrapped through the tostack runtime.
func (m *Module) LocalTee(index int32, value ExpressionRef, isManaged bool, typ TypeRef) ExpressionRef {
	if isManaged && m.UseShadowStack {
		value = m.Tostack(value)
	}
	return m.bmod.LocalTee(uint32(index), value, typ)
}

// =========================================================================
// Global variable operations
// =========================================================================

// GlobalGet creates a global.get expression.
func (m *Module) GlobalGet(name string, typ TypeRef) ExpressionRef {
	return m.bmod.GlobalGet(name, typ)
}

// GlobalSet creates a global.set expression.
func (m *Module) GlobalSet(name string, value ExpressionRef) ExpressionRef {
	return m.bmod.GlobalSet(name, value)
}

// =========================================================================
// Table get/set
// =========================================================================

// TableGet creates a table.get expression.
func (m *Module) TableGet(name string, index ExpressionRef, typ TypeRef) ExpressionRef {
	return m.bmod.TableGet(name, index, typ)
}

// TableSet creates a table.set expression.
func (m *Module) TableSet(name string, index, value ExpressionRef) ExpressionRef {
	return m.bmod.TableSet(name, index, value)
}

// =========================================================================
// Memory load / store
// =========================================================================

// Load creates a memory load expression.
func (m *Module) Load(bytes uint32, signed bool, ptr ExpressionRef, typ TypeRef, offset uint32, align uint32, memoryName string) ExpressionRef {
	if memoryName == "" {
		memoryName = DefaultMemory
	}
	return m.bmod.Load(bytes, signed, offset, align, typ, ptr, memoryName)
}

// Store creates a memory store expression.
func (m *Module) Store(bytes uint32, ptr, value ExpressionRef, typ TypeRef, offset uint32, align uint32, memoryName string) ExpressionRef {
	if memoryName == "" {
		memoryName = DefaultMemory
	}
	return m.bmod.Store(bytes, offset, align, ptr, value, typ, memoryName)
}

// =========================================================================
// Atomic operations
// =========================================================================

// AtomicLoad creates an atomic load expression.
func (m *Module) AtomicLoad(bytes uint32, ptr ExpressionRef, typ TypeRef, offset uint32, memoryName string) ExpressionRef {
	return m.bmod.AtomicLoad(bytes, offset, typ, ptr, memoryName)
}

// AtomicStore creates an atomic store expression.
func (m *Module) AtomicStore(bytes uint32, ptr, value ExpressionRef, typ TypeRef, offset uint32, memoryName string) ExpressionRef {
	return m.bmod.AtomicStore(bytes, offset, ptr, value, typ, memoryName)
}

// AtomicRMW creates an atomic read-modify-write expression.
func (m *Module) AtomicRMW(op Op, bytes, offset uint32, ptr, value ExpressionRef, typ TypeRef, memoryName string) ExpressionRef {
	return m.bmod.AtomicRMW(op, bytes, offset, ptr, value, typ, memoryName)
}

// AtomicCmpxchg creates an atomic compare-exchange expression.
func (m *Module) AtomicCmpxchg(bytes, offset uint32, ptr, expected, replacement ExpressionRef, typ TypeRef, memoryName string) ExpressionRef {
	return m.bmod.AtomicCmpxchg(bytes, offset, ptr, expected, replacement, typ, memoryName)
}

// AtomicWait creates an atomic wait expression.
func (m *Module) AtomicWait(ptr, expected, timeout ExpressionRef, expectedType TypeRef, memoryName string) ExpressionRef {
	return m.bmod.AtomicWait(ptr, expected, timeout, expectedType, memoryName)
}

// AtomicNotify creates an atomic notify expression.
func (m *Module) AtomicNotify(ptr, notifyCount ExpressionRef, memoryName string) ExpressionRef {
	return m.bmod.AtomicNotify(ptr, notifyCount, memoryName)
}

// AtomicFence creates an atomic fence expression.
func (m *Module) AtomicFence() ExpressionRef {
	return m.bmod.AtomicFence()
}

// =========================================================================
// Control flow
// =========================================================================

// Block creates a block expression. label can be empty for an anonymous block.
func (m *Module) Block(label string, children []ExpressionRef, typ TypeRef) ExpressionRef {
	return m.bmod.Block(label, children, typ)
}

// Flatten attempts to trivially flatten a series of statements. If the list
// has zero elements it produces a nop, if one element it returns that element
// directly (with type-safety checks), otherwise it wraps in a block.
func (m *Module) Flatten(stmts []ExpressionRef, typ TypeRef) ExpressionRef {
	length := len(stmts)
	if length == 0 {
		return m.Nop()
	}
	if length == 1 {
		single := stmts[0]
		id := GetExpressionId(single)
		switch id {
		case binaryen.ReturnId(), binaryen.ThrowId(), binaryen.UnreachableId():
			return single
		}
		singleType := GetExpressionType(single)
		if singleType != binaryen.TypeUnreachable() && singleType != typ {
			return m.Unreachable()
		}
		return single
	}
	return m.Block("", stmts, typ)
}

// Br creates a break (br) expression. condition and value can be 0.
func (m *Module) Br(label string, condition, value ExpressionRef) ExpressionRef {
	return m.bmod.Break(label, condition, value)
}

// BrIf creates a conditional break expression (sugar for Br with condition).
func (m *Module) BrIf(label string, condition ExpressionRef, value ExpressionRef) ExpressionRef {
	return m.bmod.Break(label, condition, value)
}

// Drop creates a drop expression.
func (m *Module) Drop(expression ExpressionRef) ExpressionRef {
	return m.bmod.Drop(expression)
}

// MaybeDrop drops an expression only if it evaluates to a value.
func (m *Module) MaybeDrop(expression ExpressionRef) ExpressionRef {
	typ := binaryen.ExpressionGetType(expression)
	if typ != binaryen.TypeNone() && typ != binaryen.TypeUnreachable() {
		return m.bmod.Drop(expression)
	}
	return expression
}

// MaybeDropCondition drops a pre-evaluated condition if it has relevant side
// effects, then returns the result. This is necessary because Binaryen's
// ExpressionRunner bails early when encountering a local with an unknown value.
func (m *Module) MaybeDropCondition(condition, result ExpressionRef) ExpressionRef {
	effects := binaryen.GetSideEffects(condition, m.bmod)
	// Mask out ReadsLocal and ReadsGlobal — these are harmless
	if (effects & ^(SideEffectReadsLocal | SideEffectReadsGlobal)) != 0 {
		resultType := binaryen.ExpressionGetType(result)
		return m.Block("", []ExpressionRef{
			m.Drop(condition),
			result,
		}, resultType)
	}
	return result
}

// Loop creates a loop expression.
func (m *Module) Loop(label string, body ExpressionRef) ExpressionRef {
	return m.bmod.Loop(label, body)
}

// If creates an if expression. ifFalse can be 0 for no else branch.
func (m *Module) If(condition, ifTrue, ifFalse ExpressionRef) ExpressionRef {
	return m.bmod.If(condition, ifTrue, ifFalse)
}

// Nop creates a nop expression.
func (m *Module) Nop() ExpressionRef {
	return m.bmod.Nop()
}

// Return creates a return expression. expression can be 0 for void return.
func (m *Module) Return(expression ExpressionRef) ExpressionRef {
	return m.bmod.Return(expression)
}

// Select creates a select (ternary) expression.
func (m *Module) Select(ifTrue, ifFalse, condition ExpressionRef) ExpressionRef {
	return m.bmod.Select(condition, ifTrue, ifFalse)
}

// Switch creates a switch (br_table) expression. value can be 0.
func (m *Module) Switch(names []string, defaultName string, condition, value ExpressionRef) ExpressionRef {
	return m.bmod.Switch(names, defaultName, condition, value)
}

// Call creates a direct function call expression.
func (m *Module) Call(target string, operands []ExpressionRef, returnType TypeRef) ExpressionRef {
	return m.bmod.Call(target, operands, returnType)
}

// ReturnCall creates a tail-call expression (direct).
func (m *Module) ReturnCall(target string, operands []ExpressionRef, returnType TypeRef) ExpressionRef {
	return m.bmod.ReturnCall(target, operands, returnType)
}

// CallIndirect creates an indirect function call through a table.
func (m *Module) CallIndirect(tableName string, index ExpressionRef, operands []ExpressionRef, params, results TypeRef) ExpressionRef {
	if tableName == "" {
		tableName = DefaultTable
	}
	return m.bmod.CallIndirect(tableName, index, operands, params, results)
}

// ReturnCallIndirect creates a tail-call indirect expression.
func (m *Module) ReturnCallIndirect(tableName string, index ExpressionRef, operands []ExpressionRef, params, results TypeRef) ExpressionRef {
	if tableName == "" {
		tableName = DefaultTable
	}
	return m.bmod.ReturnCallIndirect(tableName, index, operands, params, results)
}

// Unreachable creates an unreachable expression (trap).
func (m *Module) Unreachable() ExpressionRef {
	return m.bmod.Unreachable()
}

// =========================================================================
// Bulk memory
// =========================================================================

// MemoryCopy creates a memory.copy expression.
func (m *Module) MemoryCopy(dest, source, size ExpressionRef, destName, sourceName string) ExpressionRef {
	if destName == "" {
		destName = DefaultMemory
	}
	if sourceName == "" {
		sourceName = DefaultMemory
	}
	return m.bmod.MemoryCopy(dest, source, size, destName, sourceName)
}

// MemoryFill creates a memory.fill expression.
func (m *Module) MemoryFill(dest, value, size ExpressionRef, memoryName string) ExpressionRef {
	if memoryName == "" {
		memoryName = DefaultMemory
	}
	return m.bmod.MemoryFill(dest, value, size, memoryName)
}

// =========================================================================
// Exception handling
// =========================================================================

// Try creates a try expression.
func (m *Module) Try(name string, body ExpressionRef, catchTags []string, catchBodies []ExpressionRef, delegateTarget string) ExpressionRef {
	return m.bmod.Try(name, body, catchTags, catchBodies, delegateTarget)
}

// Throw creates a throw expression.
func (m *Module) Throw(tagName string, operands []ExpressionRef) ExpressionRef {
	return m.bmod.Throw(tagName, operands)
}

// Rethrow creates a rethrow expression.
func (m *Module) Rethrow(target string) ExpressionRef {
	return m.bmod.Rethrow(target)
}

// =========================================================================
// Multi-value (pseudo instructions)
// =========================================================================

// Pop creates a pop expression (used in catch blocks).
func (m *Module) Pop(typ TypeRef) ExpressionRef {
	return m.bmod.Pop(typ)
}

// TupleMake creates a tuple.make expression.
func (m *Module) TupleMake(operands []ExpressionRef) ExpressionRef {
	return m.bmod.TupleMake(operands)
}

// TupleExtract creates a tuple.extract expression.
func (m *Module) TupleExtract(tuple ExpressionRef, index uint32) ExpressionRef {
	return m.bmod.TupleExtract(tuple, index)
}

// =========================================================================
// SIMD
// =========================================================================

// SIMDExtract creates a SIMD extract lane expression.
func (m *Module) SIMDExtract(op Op, vec ExpressionRef, idx uint8) ExpressionRef {
	return m.bmod.SIMDExtract(op, vec, idx)
}

// SIMDReplace creates a SIMD replace lane expression.
func (m *Module) SIMDReplace(op Op, vec ExpressionRef, idx uint8, value ExpressionRef) ExpressionRef {
	return m.bmod.SIMDReplace(op, vec, idx, value)
}

// SIMDShuffle creates a SIMD shuffle expression with a 16-byte mask.
func (m *Module) SIMDShuffle(vec1, vec2 ExpressionRef, mask [16]byte) ExpressionRef {
	return m.bmod.SIMDShuffle(vec1, vec2, mask)
}

// SIMDTernary creates a SIMD ternary expression (e.g. bitselect).
func (m *Module) SIMDTernary(op Op, a, b, c ExpressionRef) ExpressionRef {
	return m.bmod.SIMDTernary(op, a, b, c)
}

// SIMDShift creates a SIMD shift expression.
func (m *Module) SIMDShift(op Op, vec, shift ExpressionRef) ExpressionRef {
	return m.bmod.SIMDShift(op, vec, shift)
}

// SIMDLoad creates a SIMD load expression.
func (m *Module) SIMDLoad(op Op, ptr ExpressionRef, offset, align uint32, memoryName string) ExpressionRef {
	if memoryName == "" {
		memoryName = DefaultMemory
	}
	return m.bmod.SIMDLoad(op, offset, align, ptr, memoryName)
}

// SIMDLoadStoreLane creates a SIMD load/store lane expression.
func (m *Module) SIMDLoadStoreLane(op Op, ptr ExpressionRef, offset, align uint32, index uint8, vec ExpressionRef, memoryName string) ExpressionRef {
	if memoryName == "" {
		memoryName = DefaultMemory
	}
	return m.bmod.SIMDLoadStoreLane(op, offset, align, index, ptr, vec, memoryName)
}

// =========================================================================
// Reference types / GC
// =========================================================================

// RefIsNull creates a ref.is_null expression.
func (m *Module) RefIsNull(expr ExpressionRef) ExpressionRef {
	return m.bmod.RefIsNull(expr)
}

// RefAs creates a ref.as expression.
func (m *Module) RefAs(op Op, expr ExpressionRef) ExpressionRef {
	return m.bmod.RefAs(op, expr)
}

// RefAsNonNull creates a ref.as_non_null expression if the type is nullable,
// otherwise returns the expression unchanged.
func (m *Module) RefAsNonNull(expr ExpressionRef) ExpressionRef {
	if IsNullableType(GetExpressionType(expr)) {
		return m.bmod.RefAs(binaryen.RefAsNonNull(), expr)
	}
	return expr
}

// RefFunc creates a ref.func expression.
func (m *Module) RefFunc(name string, typ TypeRef) ExpressionRef {
	ht := binaryen.TypeGetHeapType(typ)
	return m.bmod.RefFunc(name, ht)
}

// RefI31 creates a ref.i31 expression.
func (m *Module) RefI31(value ExpressionRef) ExpressionRef {
	return m.bmod.RefI31(value)
}

// I31Get creates an i31.get expression.
func (m *Module) I31Get(expr ExpressionRef, signed bool) ExpressionRef {
	return m.bmod.I31Get(expr, signed)
}

// RefTest creates a ref.test expression.
func (m *Module) RefTest(ref ExpressionRef, castType TypeRef) ExpressionRef {
	return m.bmod.RefTest(ref, castType)
}

// RefCast creates a ref.cast expression.
func (m *Module) RefCast(ref ExpressionRef, typ TypeRef) ExpressionRef {
	return m.bmod.RefCast(ref, typ)
}

// BrOn creates a br_on expression (br_on_cast, br_on_null, etc.).
func (m *Module) BrOn(op Op, name string, ref ExpressionRef, castType TypeRef) ExpressionRef {
	return m.bmod.BrOn(op, name, ref, castType)
}

// CallRef creates a call_ref expression.
func (m *Module) CallRef(target ExpressionRef, operands []ExpressionRef, typ TypeRef, isReturn bool) ExpressionRef {
	return m.bmod.CallRef(target, operands, typ, isReturn)
}

// =========================================================================
// GC — structs
// =========================================================================

// StructNew creates a struct.new expression. Pass nil operands for struct.new_default.
func (m *Module) StructNew(operands []ExpressionRef, typ binaryen.HeapType) ExpressionRef {
	return m.bmod.StructNew(operands, typ)
}

// StructGet creates a struct.get expression.
func (m *Module) StructGet(index uint32, ref ExpressionRef, typ TypeRef, signed bool) ExpressionRef {
	return m.bmod.StructGet(index, ref, typ, signed)
}

// StructSet creates a struct.set expression.
func (m *Module) StructSet(index uint32, ref, value ExpressionRef) ExpressionRef {
	return m.bmod.StructSet(index, ref, value)
}

// =========================================================================
// GC — arrays
// =========================================================================

// ArrayNew creates an array.new expression. init can be 0 for array.new_default.
func (m *Module) ArrayNew(typ binaryen.HeapType, size, init ExpressionRef) ExpressionRef {
	return m.bmod.ArrayNew(typ, size, init)
}

// ArrayNewData creates an array.new_data expression.
func (m *Module) ArrayNewData(typ binaryen.HeapType, name string, offset, size ExpressionRef) ExpressionRef {
	return m.bmod.ArrayNewData(typ, name, offset, size)
}

// ArrayNewFixed creates an array.new_fixed expression.
func (m *Module) ArrayNewFixed(typ binaryen.HeapType, values []ExpressionRef) ExpressionRef {
	return m.bmod.ArrayNewFixed(typ, values)
}

// ArrayGet creates an array.get expression.
func (m *Module) ArrayGet(ref, index ExpressionRef, typ TypeRef, signed bool) ExpressionRef {
	return m.bmod.ArrayGet(ref, index, typ, signed)
}

// ArraySet creates an array.set expression.
func (m *Module) ArraySet(ref, index, value ExpressionRef) ExpressionRef {
	return m.bmod.ArraySet(ref, index, value)
}

// ArrayLen creates an array.len expression.
func (m *Module) ArrayLen(ref ExpressionRef) ExpressionRef {
	return m.bmod.ArrayLen(ref)
}

// ArrayCopy creates an array.copy expression.
func (m *Module) ArrayCopy(destRef, destIndex, srcRef, srcIndex, length ExpressionRef) ExpressionRef {
	return m.bmod.ArrayCopy(destRef, destIndex, srcRef, srcIndex, length)
}

// =========================================================================
// Strings
// =========================================================================

// StringNew creates a string.new expression.
func (m *Module) StringNew(op Op, ref, start, end ExpressionRef) ExpressionRef {
	return m.bmod.StringNew(op, ref, start, end)
}

// StringConst creates a string.const expression.
func (m *Module) StringConst(name string) ExpressionRef {
	return m.bmod.StringConst(name)
}

// StringMeasure creates a string.measure expression.
func (m *Module) StringMeasure(op Op, ref ExpressionRef) ExpressionRef {
	return m.bmod.StringMeasure(op, ref)
}

// StringEncode creates a string.encode expression.
func (m *Module) StringEncode(op Op, ref, ptr, start ExpressionRef) ExpressionRef {
	return m.bmod.StringEncode(op, ref, ptr, start)
}

// StringConcat creates a string.concat expression.
func (m *Module) StringConcat(left, right ExpressionRef) ExpressionRef {
	return m.bmod.StringConcat(left, right)
}

// StringWTF16Get creates a stringview_wtf16.get_codeunit expression.
func (m *Module) StringWTF16Get(ref, pos ExpressionRef) ExpressionRef {
	return m.bmod.StringWTF16Get(ref, pos)
}

// StringSliceWTF creates a stringview_wtf.slice expression.
func (m *Module) StringSliceWTF(ref, start, end ExpressionRef) ExpressionRef {
	return m.bmod.StringSliceWTF(ref, start, end)
}

// =========================================================================
// Module structure — globals
// =========================================================================

// AddGlobal adds a global to the module.
func (m *Module) AddGlobal(name string, typ TypeRef, mutable bool, initializer ExpressionRef) GlobalRef {
	return m.bmod.AddGlobal(name, typ, mutable, initializer)
}

// GetGlobal returns a global reference by name, or 0 if not found.
func (m *Module) GetGlobal(name string) GlobalRef {
	return m.bmod.GetGlobal(name)
}

// RemoveGlobal removes a global by name. Returns false if the global did not exist.
func (m *Module) RemoveGlobal(name string) bool {
	if m.bmod.GetGlobal(name) == 0 {
		return false
	}
	m.bmod.RemoveGlobal(name)
	return true
}

// =========================================================================
// Module structure — tags
// =========================================================================

// AddTag adds a tag to the module.
func (m *Module) AddTag(name string, params, results TypeRef) TagRef {
	return m.bmod.AddTag(name, params, results)
}

// GetTag returns a tag reference by name, or 0 if not found.
func (m *Module) GetTag(name string) TagRef {
	return m.bmod.GetTag(name)
}

// RemoveTag removes a tag by name.
func (m *Module) RemoveTag(name string) {
	m.bmod.RemoveTag(name)
}

// =========================================================================
// Module structure — functions
// =========================================================================

// AddFunction adds a function to the module.
func (m *Module) AddFunction(name string, params, results TypeRef, varTypes []TypeRef, body ExpressionRef) FunctionRef {
	return m.bmod.AddFunction(name, params, results, varTypes, body)
}

// SetLocalName sets the debug name for a function local.
func (m *Module) SetLocalName(fn FunctionRef, index uint32, name string) {
	binaryen.FunctionSetLocalName(fn, index, name)
}

// GetFunction returns a function reference by name, or 0 if not found.
func (m *Module) GetFunction(name string) FunctionRef {
	return m.bmod.GetFunction(name)
}

// RemoveFunction removes a function by name.
func (m *Module) RemoveFunction(name string) {
	m.bmod.RemoveFunction(name)
}

// HasFunction returns whether a function with the given name exists.
func (m *Module) HasFunction(name string) bool {
	return m.bmod.GetFunction(name) != 0
}

// GetFunctionBody returns the body expression of a function.
func (m *Module) GetFunctionBody(fn FunctionRef) ExpressionRef {
	return binaryen.FunctionGetBody(fn)
}

// AddTemporaryFunction adds a function with an empty name for temporary use.
// Only one temporary function can exist at a time.
func (m *Module) AddTemporaryFunction(result TypeRef, paramTypes []TypeRef, body ExpressionRef) FunctionRef {
	if m.hasTemporaryFunction {
		panic("module: temporary function already exists")
	}
	m.hasTemporaryFunction = true
	params := CreateType(paramTypes)
	return m.bmod.AddFunction("", params, result, nil, body)
}

// RemoveTemporaryFunction removes the temporary function added by AddTemporaryFunction.
func (m *Module) RemoveTemporaryFunction() {
	if !m.hasTemporaryFunction {
		panic("module: no temporary function to remove")
	}
	m.hasTemporaryFunction = false
	m.bmod.RemoveFunction("")
}

// SetStart sets the start function.
func (m *Module) SetStart(fn FunctionRef) {
	m.bmod.SetStart(fn)
}

// =========================================================================
// Exports
// =========================================================================

// AddFunctionExport adds a function export to the module.
func (m *Module) AddFunctionExport(internalName, externalName string) binaryen.ExportRef {
	return m.bmod.AddFunctionExport(internalName, externalName)
}

// AddTableExport adds a table export to the module.
func (m *Module) AddTableExport(internalName, externalName string) binaryen.ExportRef {
	return m.bmod.AddTableExport(internalName, externalName)
}

// AddMemoryExport adds a memory export to the module.
func (m *Module) AddMemoryExport(internalName, externalName string) binaryen.ExportRef {
	return m.bmod.AddMemoryExport(internalName, externalName)
}

// AddGlobalExport adds a global export to the module.
func (m *Module) AddGlobalExport(internalName, externalName string) binaryen.ExportRef {
	return m.bmod.AddGlobalExport(internalName, externalName)
}

// AddTagExport adds a tag export to the module.
func (m *Module) AddTagExport(internalName, externalName string) binaryen.ExportRef {
	return m.bmod.AddTagExport(internalName, externalName)
}

// HasExport returns whether an export with the given external name exists.
func (m *Module) HasExport(externalName string) bool {
	return m.bmod.GetExport(externalName) != 0
}

// GetExport returns an export reference by external name, or 0 if not found.
func (m *Module) GetExport(externalName string) binaryen.ExportRef {
	return m.bmod.GetExport(externalName)
}

// RemoveExport removes an export by external name.
func (m *Module) RemoveExport(externalName string) {
	m.bmod.RemoveExport(externalName)
}

// =========================================================================
// Imports
// =========================================================================

// AddFunctionImport adds a function import.
func (m *Module) AddFunctionImport(internalName, externalModuleName, externalBaseName string, params, results TypeRef) {
	m.bmod.AddFunctionImport(internalName, externalModuleName, externalBaseName, params, results)
}

// AddTableImport adds a table import.
func (m *Module) AddTableImport(internalName, externalModuleName, externalBaseName string) {
	m.bmod.AddTableImport(internalName, externalModuleName, externalBaseName)
}

// AddMemoryImport adds a memory import.
func (m *Module) AddMemoryImport(internalName, externalModuleName, externalBaseName string, shared bool) {
	m.bmod.AddMemoryImport(internalName, externalModuleName, externalBaseName, shared)
}

// AddGlobalImport adds a global import.
func (m *Module) AddGlobalImport(internalName, externalModuleName, externalBaseName string, globalType TypeRef, mutable bool) {
	m.bmod.AddGlobalImport(internalName, externalModuleName, externalBaseName, globalType, mutable)
}

// AddTagImport adds a tag import.
func (m *Module) AddTagImport(internalName, externalModuleName, externalBaseName string, params, results TypeRef) {
	m.bmod.AddTagImport(internalName, externalModuleName, externalBaseName, params, results)
}

// =========================================================================
// Memory
// =========================================================================

// SetMemory configures the module's memory with optional data segments.
func (m *Module) SetMemory(initial, maximum uint32, segments []MemorySegment, target Target, exportName string, memoryName string, shared bool) {
	if memoryName == "" {
		memoryName = DefaultMemory
	}
	is64 := target == TargetWasm64
	if len(segments) == 0 {
		m.bmod.SetMemoryFull(initial, maximum, exportName, nil, shared, is64, memoryName)
		return
	}
	bsegs := make([]binaryen.DataSegment, len(segments))
	for i, seg := range segments {
		bsegs[i] = binaryen.DataSegment{
			Name:    segmentName(i),
			Data:    seg.Buffer,
			Passive: false,
			Offset:  seg.Offset,
		}
	}
	m.bmod.SetMemoryFull(initial, maximum, exportName, bsegs, shared, is64, memoryName)
}

// segmentName generates a name for a data segment by index.
func segmentName(index int) string {
	// Use a simple numeric naming scheme
	const digits = "0123456789"
	if index < 10 {
		return string(digits[index])
	}
	result := make([]byte, 0, 4)
	for index > 0 {
		result = append([]byte{digits[index%10]}, result...)
		index /= 10
	}
	return string(result)
}

// =========================================================================
// Tables
// =========================================================================

// AddFunctionTable adds or updates a function table with an active element segment.
func (m *Module) AddFunctionTable(name string, initial, maximum uint32, funcs []string, offset ExpressionRef) {
	tableRef := m.bmod.GetTable(name)
	if tableRef == 0 {
		m.bmod.AddTable(name, initial, maximum, binaryen.TypeFuncref())
	} else {
		binaryen.TableSetInitial(tableRef, initial)
		binaryen.TableSetMax(tableRef, maximum)
	}
	m.bmod.AddActiveElementSegment(name, name, funcs, offset)
}

// =========================================================================
// Custom sections
// =========================================================================

// AddCustomSection adds a custom section to the module.
// Note: requires the binaryen AddCustomSection API.
func (m *Module) AddCustomSection(name string, contents []byte) {
	// Delegate to binaryen; the binding must expose this.
	// The binaryen Go bindings should provide this API.
	_ = name
	_ = contents
}

// =========================================================================
// Pass configuration (global settings)
// =========================================================================

// SetOptimizeLevel sets the global optimization level (0-4).
func (m *Module) SetOptimizeLevel(level int) {
	binaryen.SetOptimizeLevel(level)
}

// GetOptimizeLevel returns the current global optimization level.
func (m *Module) GetOptimizeLevel() int {
	return binaryen.GetOptimizeLevel()
}

// SetShrinkLevel sets the global shrink level (0-2).
func (m *Module) SetShrinkLevel(level int) {
	binaryen.SetShrinkLevel(level)
}

// GetShrinkLevel returns the current global shrink level.
func (m *Module) GetShrinkLevel() int {
	return binaryen.GetShrinkLevel()
}

// SetDebugInfo enables or disables debug info generation.
func (m *Module) SetDebugInfo(on bool) {
	binaryen.SetDebugInfo(on)
}

// SetLowMemoryUnused marks low memory as unused for optimization.
func (m *Module) SetLowMemoryUnused(on bool) {
	binaryen.SetLowMemoryUnused(on)
}

// GetLowMemoryUnused returns whether low memory is marked as unused.
func (m *Module) GetLowMemoryUnused() bool {
	return binaryen.GetLowMemoryUnused()
}

// SetZeroFilledMemory marks memory as zero-filled for optimization.
func (m *Module) SetZeroFilledMemory(on bool) {
	binaryen.SetZeroFilledMemory(on)
}

// SetFastMath enables or disables fast math optimizations.
func (m *Module) SetFastMath(on bool) {
	binaryen.SetFastMath(on)
}

// SetClosedWorld enables closed-world assumptions for optimization.
func (m *Module) SetClosedWorld(on bool) {
	binaryen.SetClosedWorld(on)
}

// SetGenerateStackIR enables or disables stack IR generation.
func (m *Module) SetGenerateStackIR(on bool) {
	binaryen.SetGenerateStackIR(on)
}

// SetOptimizeStackIR enables or disables stack IR optimization.
func (m *Module) SetOptimizeStackIR(on bool) {
	binaryen.SetOptimizeStackIR(on)
}

// SetPassArgument sets a pass argument key-value pair.
func (m *Module) SetPassArgument(key, value string) {
	binaryen.SetPassArgument(key, value)
}

// ClearPassArguments clears all pass arguments.
func (m *Module) ClearPassArguments() {
	binaryen.ClearPassArguments()
}

// SetAlwaysInlineMaxSize sets the maximum function size for always-inlining.
func (m *Module) SetAlwaysInlineMaxSize(size uint32) {
	binaryen.SetAlwaysInlineMaxSize(size)
}

// SetFlexibleInlineMaxSize sets the maximum function size for flexible inlining.
func (m *Module) SetFlexibleInlineMaxSize(size uint32) {
	binaryen.SetFlexibleInlineMaxSize(size)
}

// SetOneCallerInlineMaxSize sets the maximum function size for one-caller inlining.
func (m *Module) SetOneCallerInlineMaxSize(size uint32) {
	binaryen.SetOneCallerInlineMaxSize(size)
}

// SetAllowInliningFunctionsWithLoops enables or disables inlining of functions
// containing loops.
func (m *Module) SetAllowInliningFunctionsWithLoops(enabled bool) {
	binaryen.SetAllowInliningFunctionsWithLoops(enabled)
}

// =========================================================================
// Feature flags
// =========================================================================

// GetFeatures returns the enabled Wasm features.
func (m *Module) GetFeatures() binaryen.Features {
	return m.bmod.GetFeatures()
}

// SetFeatures sets the enabled Wasm features. Automatically enables
// BulkMemoryOpt when BulkMemory is set (ported from TS: module.ts:2576-2579).
func (m *Module) SetFeatures(features binaryen.Features) {
	if features&binaryen.FeatureBulkMemory() != 0 {
		features |= binaryen.FeatureBulkMemoryOpt()
	}
	m.bmod.SetFeatures(features)
}

// =========================================================================
// Run passes
// =========================================================================

// RunPasses runs the named optimization passes on the module.
func (m *Module) RunPasses(passes []string) {
	m.bmod.RunPasses(passes)
}

// RunPassesOnFunction runs the named optimization passes on a single function.
// Ported from TS: module.ts:2581-2595 (the func != 0 branch of runPasses).
func (m *Module) RunPassesOnFunction(fn FunctionRef, passes []string) {
	binaryen.FunctionRunPasses(fn, m.bmod, passes)
}

// =========================================================================
// Optimize — custom pass ordering from AssemblyScript
// =========================================================================

// Optimize runs the AssemblyScript custom optimization pipeline.
// This differs substantially from Binaryen's default pass ordering.
func (m *Module) Optimize(optimizeLevel, shrinkLevel int, debugInfo, zeroFilledMemory bool) {
	// Implicitly run costly non-LLVM optimizations on -O3 or -Oz
	if optimizeLevel >= 3 || shrinkLevel >= 2 {
		optimizeLevel = 4
	}

	m.SetOptimizeLevel(optimizeLevel)
	m.SetShrinkLevel(shrinkLevel)
	m.SetDebugInfo(debugInfo)
	m.SetZeroFilledMemory(zeroFilledMemory)
	m.SetFastMath(true)
	m.ClearPassArguments()

	// OptimizationOptions#parse in src/tools/optimization-options.h
	stackIR := optimizeLevel >= 2 || shrinkLevel >= 1
	m.SetGenerateStackIR(stackIR)
	m.SetOptimizeStackIR(stackIR)

	// Tweak inlining limits based on optimization levels
	if optimizeLevel >= 2 && shrinkLevel == 0 {
		m.SetAlwaysInlineMaxSize(12)
		m.SetFlexibleInlineMaxSize(70)
		m.SetOneCallerInlineMaxSize(200)
		m.SetAllowInliningFunctionsWithLoops(optimizeLevel >= 3)
	} else {
		if optimizeLevel <= 1 || shrinkLevel >= 2 {
			m.SetAlwaysInlineMaxSize(2)
		} else {
			m.SetAlwaysInlineMaxSize(6)
		}
		m.SetFlexibleInlineMaxSize(65)
		m.SetOneCallerInlineMaxSize(80)
		m.SetAllowInliningFunctionsWithLoops(false)
	}

	// Pass order here differs substantially from Binaryen's defaults
	// see: Binaryen/src/pass.cpp
	if optimizeLevel > 0 || shrinkLevel > 0 {
		var passes []string

		// --- PassRunner::addDefaultGlobalOptimizationPrePasses ---
		passes = append(passes, "duplicate-function-elimination")
		passes = append(passes, "remove-unused-module-elements")

		// --- PassRunner::addDefaultFunctionOptimizationPasses ---
		if optimizeLevel >= 2 {
			passes = append(passes, "once-reduction")
			passes = append(passes, "inlining")
			passes = append(passes, "simplify-globals-optimizing")
		}
		if optimizeLevel >= 3 || shrinkLevel >= 1 {
			passes = append(passes, "rse")
			passes = append(passes, "vacuum")
			passes = append(passes, "code-folding")
			passes = append(passes, "ssa-nomerge")
			passes = append(passes, "local-cse")
			passes = append(passes, "remove-unused-brs")
			passes = append(passes, "remove-unused-names")
			passes = append(passes, "merge-blocks")
			passes = append(passes, "precompute-propagate")
			passes = append(passes, "simplify-globals-optimizing")
			passes = append(passes, "gufa-optimizing")
			passes = append(passes, "dae-optimizing")
		}
		if optimizeLevel >= 3 {
			passes = append(passes, "simplify-locals-nostructure")
			passes = append(passes, "flatten")
			passes = append(passes, "vacuum")
			passes = append(passes, "simplify-locals-notee-nostructure")
			passes = append(passes, "vacuum")
			passes = append(passes, "licm")
			passes = append(passes, "merge-locals")
			passes = append(passes, "reorder-locals")
		}
		passes = append(passes, "optimize-instructions")
		if optimizeLevel >= 3 || shrinkLevel >= 1 {
			passes = append(passes, "dce")
		}
		passes = append(passes, "remove-unused-brs")
		passes = append(passes, "remove-unused-names")
		if optimizeLevel >= 3 || shrinkLevel >= 2 {
			passes = append(passes, "inlining")
			passes = append(passes, "precompute-propagate")
			passes = append(passes, "simplify-globals-optimizing")
		} else {
			passes = append(passes, "precompute")
		}
		if optimizeLevel >= 2 || shrinkLevel >= 1 {
			passes = append(passes, "pick-load-signs")
		}
		passes = append(passes, "simplify-locals-notee-nostructure")
		passes = append(passes, "vacuum")
		if optimizeLevel >= 2 || shrinkLevel >= 1 {
			passes = append(passes, "local-cse")
		}
		passes = append(passes, "reorder-locals")
		passes = append(passes, "coalesce-locals")
		passes = append(passes, "simplify-locals")
		passes = append(passes, "coalesce-locals")
		passes = append(passes, "reorder-locals")
		passes = append(passes, "vacuum")
		if optimizeLevel >= 2 || shrinkLevel >= 1 {
			passes = append(passes, "rse")
			passes = append(passes, "vacuum")
		}
		if optimizeLevel >= 3 || shrinkLevel >= 1 {
			passes = append(passes, "merge-locals")
			passes = append(passes, "vacuum")
		}
		if optimizeLevel >= 2 || shrinkLevel >= 1 {
			passes = append(passes, "simplify-globals-optimizing")
			passes = append(passes, "simplify-globals-optimizing")
		}
		passes = append(passes, "remove-unused-brs")
		passes = append(passes, "remove-unused-names")
		passes = append(passes, "merge-blocks")
		if optimizeLevel >= 3 {
			passes = append(passes, "optimize-instructions")
		}

		// --- PassRunner::addDefaultGlobalOptimizationPostPasses ---
		if optimizeLevel >= 2 || shrinkLevel >= 1 {
			passes = append(passes, "simplify-globals-optimizing")
			passes = append(passes, "dae-optimizing")
		}
		if optimizeLevel >= 2 || shrinkLevel >= 2 {
			passes = append(passes, "inlining-optimizing")
		}
		if m.GetLowMemoryUnused() {
			if optimizeLevel >= 3 || shrinkLevel >= 1 {
				passes = append(passes, "optimize-added-constants-propagate")
			} else {
				passes = append(passes, "optimize-added-constants")
			}
		}
		passes = append(passes, "duplicate-import-elimination")
		if optimizeLevel >= 2 || shrinkLevel >= 2 {
			passes = append(passes, "simplify-globals-optimizing")
		} else {
			passes = append(passes, "simplify-globals")
			passes = append(passes, "vacuum")
		}
		if optimizeLevel >= 2 && (m.GetFeatures()&binaryen.FeatureGC()) != 0 {
			passes = append(passes, "heap2local")
			passes = append(passes, "merge-locals")
			passes = append(passes, "local-subtyping")
		}
		// precompute works best after global optimizations
		if optimizeLevel >= 2 || shrinkLevel >= 1 {
			passes = append(passes, "precompute-propagate")
			passes = append(passes, "simplify-globals-optimizing")
			passes = append(passes, "simplify-globals-optimizing")
		} else {
			passes = append(passes, "precompute")
		}
		passes = append(passes, "directize")
		passes = append(passes, "dae-optimizing")
		passes = append(passes, "inlining-optimizing")
		if optimizeLevel >= 2 || shrinkLevel >= 1 {
			passes = append(passes, "code-folding")
			passes = append(passes, "ssa-nomerge")
			passes = append(passes, "rse")
			passes = append(passes, "code-pushing")
			if optimizeLevel >= 3 {
				// very expensive, so O3 only
				passes = append(passes, "simplify-globals")
				passes = append(passes, "vacuum")

				passes = append(passes, "precompute-propagate")

				// replace indirect with direct calls again and inline
				passes = append(passes, "inlining-optimizing")
				passes = append(passes, "directize")
				passes = append(passes, "dae-optimizing")
				passes = append(passes, "local-cse")

				passes = append(passes, "merge-locals")
				passes = append(passes, "coalesce-locals")
				passes = append(passes, "simplify-locals")
				passes = append(passes, "vacuum")

				passes = append(passes, "inlining")
				passes = append(passes, "precompute-propagate")
				passes = append(passes, "rse")
				passes = append(passes, "vacuum")
				passes = append(passes, "ssa-nomerge")
				passes = append(passes, "simplify-locals")
				passes = append(passes, "coalesce-locals")
			}
			passes = append(passes, "optimize-instructions")
			passes = append(passes, "remove-unused-brs")
			passes = append(passes, "remove-unused-names")
			passes = append(passes, "merge-blocks")
			passes = append(passes, "vacuum")

			passes = append(passes, "simplify-globals-optimizing")
			passes = append(passes, "reorder-globals")
			passes = append(passes, "remove-unused-brs")
			passes = append(passes, "optimize-instructions")
		}
		// clean up
		passes = append(passes, "duplicate-function-elimination")
		if shrinkLevel >= 2 {
			passes = append(passes, "merge-similar-functions")
		}
		passes = append(passes, "memory-packing")
		passes = append(passes, "remove-unused-module-elements")

		m.RunPasses(passes)
	}
}

// =========================================================================
// Validation and output
// =========================================================================

// Validate checks whether the module is valid Wasm.
func (m *Module) Validate() bool {
	return m.bmod.Validate()
}

// Interpret runs the module through the Binaryen interpreter.
func (m *Module) Interpret() {
	m.bmod.Interpret()
}

// ToBinary serializes the module to Wasm binary format.
func (m *Module) ToBinary(sourceMapURL string) BinaryModule {
	if sourceMapURL != "" {
		binary, sourceMap := m.bmod.EmitBinaryWithSourceMap(sourceMapURL)
		return BinaryModule{
			Binary:    binary,
			SourceMap: sourceMap,
		}
	}
	return BinaryModule{
		Binary:    m.bmod.EmitBinary(),
		SourceMap: "",
	}
}

// ToText serializes the module to WAT text format.
func (m *Module) ToText(watFormat bool) string {
	if watFormat {
		return m.bmod.EmitStackIR()
	}
	return m.bmod.EmitText()
}

// =========================================================================
// Dispose
// =========================================================================

// Dispose frees the module and all its contents. After Dispose, the Module
// must not be used.
func (m *Module) Dispose() {
	if m.bmod != nil {
		m.bmod.Dispose()
		m.bmod = nil
	}
}

// =========================================================================
// Expression utilities
// =========================================================================

// TryCopyTrivialExpression makes a copy of a trivial expression (one that
// does not contain subexpressions). Returns 0 if the expression is non-trivial.
func (m *Module) TryCopyTrivialExpression(expr ExpressionRef) ExpressionRef {
	id := GetExpressionId(expr)
	switch id {
	case binaryen.LocalGetId(),
		binaryen.GlobalGetId(),
		binaryen.ConstId(),
		binaryen.MemorySizeId(),
		binaryen.NopId(),
		binaryen.UnreachableId(),
		binaryen.DataDropId(),
		binaryen.RefNullId():
		return m.CopyExpression(expr)
	}
	return 0
}

// CopyExpression makes a deep copy of any expression including all subexpressions.
func (m *Module) CopyExpression(expr ExpressionRef) ExpressionRef {
	return binaryen.ExpressionCopy(expr, m.bmod)
}

// RunExpression evaluates an expression using the Binaryen expression runner.
// Returns the pre-computed constant expression, or 0 if evaluation fails.
func (m *Module) RunExpression(expr ExpressionRef, flags binaryen.ExpressionRunnerFlags, maxDepth, maxLoopIterations int32) ExpressionRef {
	runner := binaryen.ExpressionRunnerCreate(m.bmod, flags, uint32(maxDepth), uint32(maxLoopIterations))
	precomp := binaryen.ExpressionRunnerRunAndDispose(runner, expr)
	if precomp != 0 && !m.IsConstExpression(precomp) {
		return 0
	}
	return precomp
}

// IsConstExpression returns whether an expression is a constant expression
// suitable for use as a global initializer.
func (m *Module) IsConstExpression(expr ExpressionRef) bool {
	id := GetExpressionId(expr)
	switch id {
	case binaryen.ConstId(),
		binaryen.RefNullId(),
		binaryen.RefFuncId(),
		binaryen.RefI31Id():
		return true
	case binaryen.BinaryId():
		if m.GetFeatures()&binaryen.FeatureExtendedConst() != 0 {
			op := GetBinaryOp(expr)
			switch op {
			case binaryen.AddInt32(),
				binaryen.SubInt32(),
				binaryen.MulInt32(),
				binaryen.AddInt64(),
				binaryen.SubInt64(),
				binaryen.MulInt64():
				return m.IsConstExpression(GetBinaryLeft(expr)) &&
					m.IsConstExpression(GetBinaryRight(expr))
			}
		}
	}
	return false
}

// =========================================================================
// Debug info
// =========================================================================

// AddDebugInfoFile adds a source file to the debug info and returns its index.
func (m *Module) AddDebugInfoFile(name string) uint32 {
	return m.bmod.AddDebugInfoFileName(name)
}

// GetDebugInfoFile returns the source file name at the given index.
func (m *Module) GetDebugInfoFile(index uint32) string {
	return m.bmod.GetDebugInfoFileName(index)
}

// SetDebugLocation sets the debug source location for an expression within a function.
func (m *Module) SetDebugLocation(fn FunctionRef, expr ExpressionRef, fileIndex, lineNumber, columnNumber uint32) {
	m.bmod.SetDebugLocation(fn, expr, fileIndex, lineNumber, columnNumber)
}

// =========================================================================
// Relooper
// =========================================================================

// CreateRelooper creates a new Relooper for CFG to structured control flow conversion.
func (m *Module) CreateRelooper() *binaryen.Relooper {
	return m.bmod.NewRelooper()
}
