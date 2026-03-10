// Ported from: assemblyscript/src/builtins.ts
// Memory access builtins: load, store, memory.data
package compiler

import (
	"encoding/binary"
	"math"
	"strconv"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// registerMemoryBuiltins registers memory access builtin handlers.
func registerMemoryBuiltins() {
	builtinFunctions["load"] = builtinLoad
	builtinFunctions["store"] = builtinStore
	builtinFunctions["memory.data"] = builtinMemoryData
}

// load<T!>(ptr: usize, immOffset?: usize, immAlign?: usize) -> T*
// Ported from: assemblyscript/src/builtins.ts builtin_load (line 2423).
func builtinLoad(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if boolToInt(checkTypeRequired(ctx, true))|
		boolToInt(checkArgsOptional(ctx, 1, 3)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	contextualType := ctx.ContextualType
	typ := typeArguments[0]

	outType := typ
	if contextualType != types.TypeAuto &&
		typ.IsIntegerValue() &&
		contextualType.IsIntegerValue() &&
		contextualType.Size > typ.Size {
		outType = contextualType
	}

	if !outType.IsMemory() {
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			typeArgsRange(ctx), "load", outType.String(), "",
		)
		compiler.CurrentType = types.TypeVoid
		return mod.Unreachable()
	}

	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	numOperands := len(operands)
	var immOffset int32
	immAlign := typ.ByteSize()
	if numOperands >= 2 {
		immOffset = evaluateImmediateOffset(operands[1], compiler) // reports
		if immOffset < 0 {
			compiler.CurrentType = outType
			return mod.Unreachable()
		}
		if numOperands == 3 {
			immAlign = evaluateImmediateAlign(operands[2], immAlign, compiler) // reports
			if immAlign < 0 {
				compiler.CurrentType = outType
				return mod.Unreachable()
			}
		}
	}
	compiler.CurrentType = outType
	return mod.Load(
		uint32(typ.ByteSize()),
		typ.IsSignedIntegerValue(),
		arg0,
		outType.ToRef(),
		uint32(immOffset),
		uint32(immAlign),
		"",
	)
}

// store<T!>(ptr: usize, value: T*, immOffset?: usize, immAlign?: usize) -> void
// Ported from: assemblyscript/src/builtins.ts builtin_store (line 2482).
func builtinStore(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeVoid
	if boolToInt(checkTypeRequired(ctx, false))|
		boolToInt(checkArgsOptional(ctx, 2, 4)) != 0 {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	numOperands := len(operands)
	typeArguments := ctx.TypeArguments
	contextualType := ctx.ContextualType
	typ := typeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], compiler.Options().UsizeType(), ConstraintsConvImplicit)
	var arg1 module.ExpressionRef
	if ctx.ContextIsExact {
		arg1 = compiler.CompileExpression(operands[1],
			contextualType,
			ConstraintsConvImplicit,
		)
	} else {
		constraints := ConstraintsConvImplicit
		if typ.IsIntegerValue() {
			constraints = ConstraintsNone // no need to convert to small int (but now might result in a float)
		}
		arg1 = compiler.CompileExpression(
			operands[1],
			typ,
			constraints,
		)
	}
	inType := compiler.CurrentType
	if !inType.IsMemory() {
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			typeArgsRange(ctx), "store", inType.String(), "",
		)
		compiler.CurrentType = types.TypeVoid
		return mod.Unreachable()
	}
	if typ.IsIntegerValue() &&
		(!inType.IsIntegerValue() || // float to int
			inType.Size < typ.Size) { // int to larger int (clear garbage bits)
		// either conversion or memory operation clears garbage bits
		arg1 = compiler.convertExpression(arg1, inType, typ, false, operands[1])
		inType = typ
	}
	var immOffset int32
	immAlign := typ.ByteSize()
	if numOperands >= 3 {
		immOffset = evaluateImmediateOffset(operands[2], compiler) // reports
		if immOffset < 0 {
			compiler.CurrentType = types.TypeVoid
			return mod.Unreachable()
		}
		if numOperands == 4 {
			immAlign = evaluateImmediateAlign(operands[3], immAlign, compiler) // reports
			if immAlign < 0 {
				compiler.CurrentType = types.TypeVoid
				return mod.Unreachable()
			}
		}
	}
	compiler.CurrentType = types.TypeVoid
	return mod.Store(uint32(typ.ByteSize()), arg0, arg1, inType.ToRef(), uint32(immOffset), uint32(immAlign), "")
}

// memory.data(size[, align]) -> usize
// memory.data<T>(values[, align]) -> usize
// Ported from: assemblyscript/src/builtins.ts builtin_memory_data (line 3383).
func builtinMemoryData(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeI32
	if boolToInt(checkTypeOptional(ctx, false))|
		boolToInt(checkArgsOptional(ctx, 1, 2)) != 0 {
		return mod.Unreachable()
	}
	typeArguments := ctx.TypeArguments
	operands := ctx.Operands
	numOperands := len(operands)
	usizeType := compiler.Options().UsizeType()
	var offset int64

	if typeArguments != nil && len(typeArguments) > 0 {
		// data<T>(values[, align])
		elementType := typeArguments[0]
		if !elementType.IsValue() {
			compiler.Error(
				diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
				typeArgsRange(ctx), "memory.data", elementType.String(), "",
			)
			compiler.CurrentType = usizeType
			return mod.Unreachable()
		}
		valuesOperand := operands[0]
		arrayLiteral, ok := valuesOperand.(*ast.ArrayLiteralExpression)
		if !ok {
			compiler.Error(
				diagnostics.DiagnosticCodeArrayLiteralExpected,
				operands[0].GetRange(),
				"", "", "",
			)
			compiler.CurrentType = usizeType
			return mod.Unreachable()
		}
		expressions := arrayLiteral.ElementExpressions
		numElements := len(expressions)
		exprs := make([]module.ExpressionRef, numElements)
		isStatic := true
		for i := 0; i < numElements; i++ {
			elementExpression := expressions[i]
			if elementExpression != nil && elementExpression.GetKind() != ast.NodeKindOmitted {
				expr := compiler.CompileExpression(elementExpression, elementType, ConstraintsConvImplicit)
				precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
				if precomp != 0 {
					expr = precomp
				} else {
					isStatic = false
				}
				exprs[i] = expr
			} else {
				exprs[i] = compiler.makeZeroOfType(elementType)
			}
		}
		if !isStatic {
			compiler.Error(
				diagnostics.DiagnosticCodeExpressionMustBeACompileTimeConstant,
				valuesOperand.GetRange(),
				"", "", "",
			)
			compiler.CurrentType = usizeType
			return mod.Unreachable()
		}
		align := elementType.ByteSize()
		if numOperands == 2 {
			align = evaluateImmediateAlign(operands[1], align, compiler) // reports
			if align < 0 {
				compiler.CurrentType = usizeType
				return mod.Unreachable()
			}
		}
		buf := make([]byte, numElements*int(elementType.ByteSize()))
		written := writeStaticBuffer(buf, 0, elementType, exprs)
		if written != len(buf) {
			panic("builtins: writeStaticBuffer size mismatch")
		}
		segment := compiler.addAlignedMemorySegment(buf, align)
		offset = module.GetConstValueInteger(segment.Offset, compiler.Options().IsWasm64())
	} else {
		// data(size[, align])
		arg0 := compiler.CompileExpression(operands[0], types.TypeI32, ConstraintsConvImplicit)
		precomp := mod.RunExpression(arg0, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
		if precomp == 0 {
			compiler.Error(
				diagnostics.DiagnosticCodeExpressionMustBeACompileTimeConstant,
				operands[0].GetRange(),
				"", "", "",
			)
			compiler.CurrentType = usizeType
			return mod.Unreachable()
		}
		size := module.GetConstValueI32(precomp)
		if size < 1 {
			compiler.Error(
				diagnostics.DiagnosticCode0MustBeAValueBetween1And2Inclusive,
				operands[0].GetRange(),
				"1", strconv.Itoa(math.MaxInt32), "",
			)
			compiler.CurrentType = usizeType
			return mod.Unreachable()
		}
		var align int32 = 16
		if numOperands == 2 {
			align = evaluateImmediateAlign(operands[1], align, compiler) // reports
			if align < 0 {
				compiler.CurrentType = usizeType
				return mod.Unreachable()
			}
		}
		segment := compiler.addAlignedMemorySegment(make([]byte, size), align)
		offset = module.GetConstValueInteger(segment.Offset, compiler.Options().IsWasm64())
	}
	// FIXME: what if recompiles happen? recompiles are bad.
	compiler.CurrentType = usizeType
	if usizeType == types.TypeUsize32 {
		return mod.I32(int32(offset))
	}
	return mod.I64(offset)
}

// writeStaticBuffer writes constant expression values into a byte buffer.
// Ported from: assemblyscript/src/compiler.ts writeStaticBuffer (line 1999).
func writeStaticBuffer(buf []byte, pos int, elementType *types.Type, values []module.ExpressionRef) int {
	length := len(values)
	byteSize := int(elementType.ByteSize())
	elementTypeRef := elementType.ToRef()

	switch elementTypeRef {
	case module.TypeRefI32:
		switch byteSize {
		case 1:
			for i := 0; i < length; i++ {
				buf[pos] = byte(module.GetConstValueI32(values[i]))
				pos += 1
			}
		case 2:
			for i := 0; i < length; i++ {
				v := uint16(module.GetConstValueI32(values[i]))
				binary.LittleEndian.PutUint16(buf[pos:], v)
				pos += 2
			}
		case 4:
			for i := 0; i < length; i++ {
				v := uint32(module.GetConstValueI32(values[i]))
				binary.LittleEndian.PutUint32(buf[pos:], v)
				pos += 4
			}
		default:
			panic("writeStaticBuffer: unexpected i32 byte size")
		}
	case module.TypeRefI64:
		for i := 0; i < length; i++ {
			v := uint64(module.GetConstValueI64(values[i]))
			binary.LittleEndian.PutUint64(buf[pos:], v)
			pos += 8
		}
	case module.TypeRefF32:
		for i := 0; i < length; i++ {
			v := math.Float32bits(module.GetConstValueF32(values[i]))
			binary.LittleEndian.PutUint32(buf[pos:], v)
			pos += 4
		}
	case module.TypeRefF64:
		for i := 0; i < length; i++ {
			v := math.Float64bits(module.GetConstValueF64(values[i]))
			binary.LittleEndian.PutUint64(buf[pos:], v)
			pos += 8
		}
	default:
		// v128 case: write 16 bytes per element
		if byteSize == 16 {
			for i := 0; i < length; i++ {
				v128 := module.GetConstValueV128(values[i])
				copy(buf[pos:pos+16], v128[:])
				pos += 16
			}
		} else {
			panic("writeStaticBuffer: unsupported element type")
		}
	}
	return pos
}
