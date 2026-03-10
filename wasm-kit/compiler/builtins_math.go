// Ported from: assemblyscript/src/builtins.ts (lines 1350-2326)
// Math builtins: bit operations, float math, arithmetic helpers.
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

func init() {
	// Bit operations
	builtinFunctions[common.BuiltinNameClz] = builtinClz
	builtinFunctions[common.BuiltinNameCtz] = builtinCtz
	builtinFunctions[common.BuiltinNamePopcnt] = builtinPopcnt
	builtinFunctions[common.BuiltinNameRotl] = builtinRotl
	builtinFunctions[common.BuiltinNameRotr] = builtinRotr

	// Float math
	builtinFunctions[common.BuiltinNameAbs] = builtinAbs
	builtinFunctions[common.BuiltinNameMax] = builtinMax
	builtinFunctions[common.BuiltinNameMin] = builtinMin
	builtinFunctions[common.BuiltinNameCeil] = builtinCeil
	builtinFunctions[common.BuiltinNameFloor] = builtinFloor
	builtinFunctions[common.BuiltinNameCopysign] = builtinCopysign
	builtinFunctions[common.BuiltinNameNearest] = builtinNearest
	builtinFunctions[common.BuiltinNameReinterpret] = builtinReinterpret
	builtinFunctions[common.BuiltinNameSqrt] = builtinSqrt
	builtinFunctions[common.BuiltinNameTrunc] = builtinTrunc

	// isNaN / isFinite
	builtinFunctions[common.BuiltinNameIsNaN] = builtinIsNaN
	builtinFunctions[common.BuiltinNameIsFinite] = builtinIsFinite
}

// builtinClz implements clz<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_clz (lines 1399-1432).
func builtinClz(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(ctx.Operands[0], typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(ctx.Operands[0], types.TypeI32, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindBool, // not wrapped
			types.TypeKindI8, types.TypeKindU8,
			types.TypeKindI16, types.TypeKindU16,
			types.TypeKindI32, types.TypeKindU32:
			return mod.Unary(module.UnaryOpClzI32, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			return mod.Unary(module.UnaryOpClzSize, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Unary(module.UnaryOpClzI64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"clz", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinCtz implements ctz<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_ctz (lines 1435-1469).
func builtinCtz(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeI32, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindBool, // not wrapped
			types.TypeKindI8, types.TypeKindU8,
			types.TypeKindI16, types.TypeKindU16,
			types.TypeKindI32, types.TypeKindU32:
			return mod.Unary(module.UnaryOpCtzI32, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			return mod.Unary(module.UnaryOpCtzSize, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Unary(module.UnaryOpCtzI64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"ctz", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinPopcnt implements popcnt<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_popcnt (lines 1472-1506).
func builtinPopcnt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeI32, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch compiler.CurrentType.Kind {
		case types.TypeKindBool:
			return arg0
		case types.TypeKindI8, // not wrapped
			types.TypeKindU8,
			types.TypeKindI16, types.TypeKindU16,
			types.TypeKindI32, types.TypeKindU32:
			return mod.Unary(module.UnaryOpPopcntI32, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Unary(module.UnaryOpPopcntI64, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			return mod.Unary(module.UnaryOpPopcntSize, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"popcnt", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinRotl implements rotl<T?>(value: T, shift: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_rotl (lines 1509-1578).
func builtinRotl(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 2) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeI32, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		arg1 := compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit)
		switch typ.Kind {
		case types.TypeKindBool:
			return arg0
		case types.TypeKindI8, types.TypeKindI16,
			types.TypeKindU8, types.TypeKindU16:
			// (value << (shift & mask)) | (value >>> ((0 - shift) & mask))
			fl := compiler.CurrentFlow
			temp1 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp1.FlowIndex(), flow.LocalFlagWrapped)
			temp2 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp2.FlowIndex(), flow.LocalFlagWrapped)
			ret := mod.Binary(module.BinaryOpOrI32,
				mod.Binary(module.BinaryOpShlI32,
					mod.LocalTee(temp1.FlowIndex(), arg0, false, module.TypeRefI32),
					mod.Binary(module.BinaryOpAndI32,
						mod.LocalTee(temp2.FlowIndex(), arg1, false, module.TypeRefI32),
						mod.I32(typ.Size-1),
					),
				),
				mod.Binary(module.BinaryOpShrU32,
					mod.LocalGet(temp1.FlowIndex(), module.TypeRefI32),
					mod.Binary(module.BinaryOpAndI32,
						mod.Binary(module.BinaryOpSubI32,
							mod.I32(0),
							mod.LocalGet(temp2.FlowIndex(), module.TypeRefI32),
						),
						mod.I32(typ.Size-1),
					),
				),
			)
			return ret
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Binary(module.BinaryOpRotlI32, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Binary(module.BinaryOpRotlI64, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			return mod.Binary(module.BinaryOpRotlSize, arg0, arg1)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"rotl", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinRotr implements rotr<T?>(value: T, shift: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_rotr (lines 1581-1650).
func builtinRotr(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 2) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeI32, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		arg1 := compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit)
		switch typ.Kind {
		case types.TypeKindBool:
			return arg0
		case types.TypeKindI8, types.TypeKindI16,
			types.TypeKindU8, types.TypeKindU16:
			// (value >>> (shift & mask)) | (value << ((0 - shift) & mask))
			fl := compiler.CurrentFlow
			temp1 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp1.FlowIndex(), flow.LocalFlagWrapped)
			temp2 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp2.FlowIndex(), flow.LocalFlagWrapped)
			ret := mod.Binary(module.BinaryOpOrI32,
				mod.Binary(module.BinaryOpShrU32,
					mod.LocalTee(temp1.FlowIndex(), arg0, false, module.TypeRefI32),
					mod.Binary(module.BinaryOpAndI32,
						mod.LocalTee(temp2.FlowIndex(), arg1, false, module.TypeRefI32),
						mod.I32(typ.Size-1),
					),
				),
				mod.Binary(module.BinaryOpShlI32,
					mod.LocalGet(temp1.FlowIndex(), module.TypeRefI32),
					mod.Binary(module.BinaryOpAndI32,
						mod.Binary(module.BinaryOpSubI32,
							mod.I32(0),
							mod.LocalGet(temp2.FlowIndex(), module.TypeRefI32),
						),
						mod.I32(typ.Size-1),
					),
				),
			)
			return ret
		case types.TypeKindI32, types.TypeKindU32:
			return mod.Binary(module.BinaryOpRotrI32, arg0, arg1)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.Binary(module.BinaryOpRotrI64, arg0, arg1)
		case types.TypeKindIsize, types.TypeKindUsize:
			return mod.Binary(module.BinaryOpRotrSize, arg0, arg1)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"rotr", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinAbs implements abs<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_abs (lines 1653-1755).
func builtinAbs(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindBool,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32,
			types.TypeKindU64, types.TypeKindUsize:
			return arg0
		case types.TypeKindI8, types.TypeKindI16, types.TypeKindI32:
			fl := compiler.CurrentFlow
			// possibly overflows, e.g. abs<i8>(-128) == 128
			temp1 := fl.GetTempLocal(types.TypeI32)
			temp2 := fl.GetTempLocal(types.TypeI32)
			// (x + (x >> 31)) ^ (x >> 31)
			ret := mod.Binary(module.BinaryOpXorI32,
				mod.Binary(module.BinaryOpAddI32,
					mod.LocalTee(temp2.FlowIndex(),
						mod.Binary(module.BinaryOpShrI32,
							mod.LocalTee(temp1.FlowIndex(), arg0, false, module.TypeRefI32),
							mod.I32(31),
						),
						false, module.TypeRefI32,
					),
					mod.LocalGet(temp1.FlowIndex(), module.TypeRefI32),
				),
				mod.LocalGet(temp2.FlowIndex(), module.TypeRefI32),
			)
			return ret
		case types.TypeKindIsize:
			options := compiler.Options()
			fl := compiler.CurrentFlow
			temp1 := fl.GetTempLocal(options.UsizeType())
			temp2 := fl.GetTempLocal(options.UsizeType())
			var shiftVal module.ExpressionRef
			if options.IsWasm64() {
				shiftVal = mod.I64(63)
			} else {
				shiftVal = mod.I32(31)
			}
			ret := mod.Binary(module.BinaryOpXorSize,
				mod.Binary(module.BinaryOpAddSize,
					mod.LocalTee(temp2.FlowIndex(),
						mod.Binary(module.BinaryOpShrISize,
							mod.LocalTee(temp1.FlowIndex(), arg0, false, options.SizeTypeRef()),
							shiftVal,
						),
						false, options.SizeTypeRef(),
					),
					mod.LocalGet(temp1.FlowIndex(), options.SizeTypeRef()),
				),
				mod.LocalGet(temp2.FlowIndex(), options.SizeTypeRef()),
			)
			return ret
		case types.TypeKindI64:
			fl := compiler.CurrentFlow
			temp1 := fl.GetTempLocal(types.TypeI64)
			temp2 := fl.GetTempLocal(types.TypeI64)
			// (x + (x >> 63)) ^ (x >> 63)
			ret := mod.Binary(module.BinaryOpXorI64,
				mod.Binary(module.BinaryOpAddI64,
					mod.LocalTee(temp2.FlowIndex(),
						mod.Binary(module.BinaryOpShrI64,
							mod.LocalTee(temp1.FlowIndex(), arg0, false, module.TypeRefI64),
							mod.I64(63),
						),
						false, module.TypeRefI64,
					),
					mod.LocalGet(temp1.FlowIndex(), module.TypeRefI64),
				),
				mod.LocalGet(temp2.FlowIndex(), module.TypeRefI64),
			)
			return ret
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpAbsF32, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpAbsF64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"abs", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinMax implements max<T?>(left: T, right: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_max (lines 1758-1823).
func builtinMax(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 2) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	left := operands[0]
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(left, typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		var arg1 module.ExpressionRef
		if (typeArguments == nil || len(typeArguments) == 0) && ast.IsNumericLiteral(left) { // prefer right type
			arg1 = compiler.CompileExpression(operands[1], typ, ConstraintsMustWrap)
			if compiler.CurrentType != typ {
				typ = compiler.CurrentType
				arg0 = compiler.CompileExpression(left, typ, ConstraintsConvImplicit|ConstraintsMustWrap)
			}
		} else {
			arg1 = compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit|ConstraintsMustWrap)
		}
		op := module.Op(-1)
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindI16, types.TypeKindI32:
			op = module.BinaryOpGtI32
		case types.TypeKindBool,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			op = module.BinaryOpGtU32
		case types.TypeKindI64:
			op = module.BinaryOpGtI64
		case types.TypeKindU64:
			op = module.BinaryOpGtU64
		case types.TypeKindIsize:
			op = module.BinaryOpGtISize
		case types.TypeKindUsize:
			op = module.BinaryOpGtUSize
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpMaxF32, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpMaxF64, arg0, arg1)
		}
		if int32(op) != -1 {
			fl := compiler.CurrentFlow
			typeRef := typ.ToRef()
			temp1 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp1.FlowIndex(), flow.LocalFlagWrapped)
			temp2 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp2.FlowIndex(), flow.LocalFlagWrapped)
			ret := mod.Select(
				mod.LocalTee(temp1.FlowIndex(), arg0, false, typeRef),
				mod.LocalTee(temp2.FlowIndex(), arg1, false, typeRef),
				mod.Binary(op,
					mod.LocalGet(temp1.FlowIndex(), typeRef),
					mod.LocalGet(temp2.FlowIndex(), typeRef),
				),
			)
			return ret
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"max", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinMin implements min<T?>(left: T, right: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_min (lines 1826-1891).
func builtinMin(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 2) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	left := operands[0]
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(left, typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		var arg1 module.ExpressionRef
		if (typeArguments == nil || len(typeArguments) == 0) && ast.IsNumericLiteral(left) { // prefer right type
			arg1 = compiler.CompileExpression(operands[1], typ, ConstraintsMustWrap)
			if compiler.CurrentType != typ {
				typ = compiler.CurrentType
				arg0 = compiler.CompileExpression(left, typ, ConstraintsConvImplicit|ConstraintsMustWrap)
			}
		} else {
			arg1 = compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit|ConstraintsMustWrap)
		}
		op := module.Op(-1)
		switch typ.Kind {
		case types.TypeKindI8, types.TypeKindI16, types.TypeKindI32:
			op = module.BinaryOpLtI32
		case types.TypeKindBool,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			op = module.BinaryOpLtU32
		case types.TypeKindI64:
			op = module.BinaryOpLtI64
		case types.TypeKindU64:
			op = module.BinaryOpLtU64
		case types.TypeKindIsize:
			op = module.BinaryOpLtISize
		case types.TypeKindUsize:
			op = module.BinaryOpLtUSize
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpMinF32, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpMinF64, arg0, arg1)
		}
		if int32(op) != -1 {
			fl := compiler.CurrentFlow
			typeRef := typ.ToRef()
			temp1 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp1.FlowIndex(), flow.LocalFlagWrapped)
			temp2 := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp2.FlowIndex(), flow.LocalFlagWrapped)
			ret := mod.Select(
				mod.LocalTee(temp1.FlowIndex(), arg0, false, typeRef),
				mod.LocalTee(temp2.FlowIndex(), arg1, false, typeRef),
				mod.Binary(op,
					mod.LocalGet(temp1.FlowIndex(), typeRef),
					mod.LocalGet(temp2.FlowIndex(), typeRef),
				),
			)
			return ret
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"min", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinCeil implements ceil<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_ceil (lines 1894-1930).
func builtinCeil(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindBool,
			types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindI64, types.TypeKindIsize,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32,
			types.TypeKindU64, types.TypeKindUsize:
			return arg0 // considered rounded
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpCeilF32, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpCeilF64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"ceil", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinFloor implements floor<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_floor (lines 1933-1969).
func builtinFloor(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindBool,
			types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindI64, types.TypeKindIsize,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32,
			types.TypeKindU64, types.TypeKindUsize:
			return arg0 // considered rounded
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpFloorF32, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpFloorF64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"floor", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinCopysign implements copysign<T?>(left: T, right: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_copysign (lines 1972-1999).
func builtinCopysign(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 2) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeF64, ConstraintsNone)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		arg1 := compiler.CompileExpression(operands[1], typ, ConstraintsConvImplicit)
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Binary(module.BinaryOpCopysignF32, arg0, arg1)
		case types.TypeKindF64:
			return mod.Binary(module.BinaryOpCopysignF64, arg0, arg1)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"copysign", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinNearest implements nearest<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_nearest (lines 2002-2038).
func builtinNearest(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindBool,
			types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindI64, types.TypeKindIsize,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32,
			types.TypeKindU64, types.TypeKindUsize:
			return arg0
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpNearestF32, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpNearestF64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"nearest", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinReinterpret implements reinterpret<T!>(value: *) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_reinterpret (lines 2041-2098).
func builtinReinterpret(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeRequired(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typ := ctx.TypeArguments[0]
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindI32, types.TypeKindU32:
			arg0 := compiler.CompileExpression(operands[0], types.TypeF32, ConstraintsConvImplicit)
			compiler.CurrentType = typ
			return mod.Unary(module.UnaryOpReinterpretF32ToI32, arg0)
		case types.TypeKindI64, types.TypeKindU64:
			arg0 := compiler.CompileExpression(operands[0], types.TypeF64, ConstraintsConvImplicit)
			compiler.CurrentType = typ
			return mod.Unary(module.UnaryOpReinterpretF64ToI64, arg0)
		case types.TypeKindIsize, types.TypeKindUsize:
			isWasm64 := compiler.Options().IsWasm64()
			var sourceType *types.Type
			if isWasm64 {
				sourceType = types.TypeF64
			} else {
				sourceType = types.TypeF32
			}
			arg0 := compiler.CompileExpression(operands[0], sourceType, ConstraintsConvImplicit)
			compiler.CurrentType = typ
			if isWasm64 {
				return mod.Unary(module.UnaryOpReinterpretF64ToI64, arg0)
			}
			return mod.Unary(module.UnaryOpReinterpretF32ToI32, arg0)
		case types.TypeKindF32:
			arg0 := compiler.CompileExpression(operands[0], types.TypeI32, ConstraintsConvImplicit)
			compiler.CurrentType = types.TypeF32
			return mod.Unary(module.UnaryOpReinterpretI32ToF32, arg0)
		case types.TypeKindF64:
			arg0 := compiler.CompileExpression(operands[0], types.TypeI64, ConstraintsConvImplicit)
			compiler.CurrentType = types.TypeF64
			return mod.Unary(module.UnaryOpReinterpretI64ToF64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"reinterpret", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinSqrt implements sqrt<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_sqrt (lines 2101-2127).
func builtinSqrt(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeF64, ConstraintsNone)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpSqrtF32, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpSqrtF64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"sqrt", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinTrunc implements trunc<T?>(value: T) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_trunc (lines 2130-2166).
func builtinTrunc(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	}
	typ := compiler.CurrentType
	if typ.IsValue() {
		switch typ.Kind {
		case types.TypeKindBool,
			types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindI64, types.TypeKindIsize,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32,
			types.TypeKindU64, types.TypeKindUsize:
			return arg0 // considered truncated
		case types.TypeKindF32:
			return mod.Unary(module.UnaryOpTruncF32, arg0)
		case types.TypeKindF64:
			return mod.Unary(module.UnaryOpTruncF64, arg0)
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"trunc", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinIsNaN implements isNaN<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isNaN (lines 2169-2240).
func builtinIsNaN(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, false) || checkArgsRequired(ctx, 1) {
		compiler.CurrentType = types.TypeBool
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	}
	typ := compiler.CurrentType
	compiler.CurrentType = types.TypeBool
	if typ.IsValue() {
		switch typ.Kind {
		// never NaN
		case types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindI64, types.TypeKindIsize,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32,
			types.TypeKindU64, types.TypeKindUsize:
			return mod.MaybeDropCondition(arg0, mod.I32(0))
		// (t = arg0) != t
		case types.TypeKindF32:
			if module.GetExpressionId(arg0) == module.ExpressionIdLocalGet {
				return mod.Binary(module.BinaryOpNeF32,
					arg0,
					mod.LocalGet(int32(module.GetLocalGetIndex(arg0)), module.TypeRefF32),
				)
			}
			fl := compiler.CurrentFlow
			temp := fl.GetTempLocal(types.TypeF32)
			ret := mod.Binary(module.BinaryOpNeF32,
				mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefF32),
				mod.LocalGet(temp.FlowIndex(), module.TypeRefF32),
			)
			return ret
		case types.TypeKindF64:
			if module.GetExpressionId(arg0) == module.ExpressionIdLocalGet {
				return mod.Binary(module.BinaryOpNeF64,
					arg0,
					mod.LocalGet(int32(module.GetLocalGetIndex(arg0)), module.TypeRefF64),
				)
			}
			fl := compiler.CurrentFlow
			temp := fl.GetTempLocal(types.TypeF64)
			ret := mod.Binary(module.BinaryOpNeF64,
				mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefF64),
				mod.LocalGet(temp.FlowIndex(), module.TypeRefF64),
			)
			return ret
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"isNaN", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinIsFinite implements isFinite<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isFinite (lines 2243-2326).
func builtinIsFinite(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeOptional(ctx, false) || checkArgsRequired(ctx, 1) {
		compiler.CurrentType = types.TypeBool
		return mod.Unreachable()
	}
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	}
	typ := compiler.CurrentType
	compiler.CurrentType = types.TypeBool
	if typ.IsValue() {
		switch typ.Kind {
		// always finite
		case types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindI64, types.TypeKindIsize,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32,
			types.TypeKindU64, types.TypeKindUsize:
			return mod.MaybeDropCondition(arg0, mod.I32(1))
		// (t = arg0) - t == 0
		case types.TypeKindF32:
			if module.GetExpressionId(arg0) == module.ExpressionIdLocalGet {
				return mod.Binary(module.BinaryOpEqF32,
					mod.Binary(module.BinaryOpSubF32,
						arg0,
						mod.LocalGet(int32(module.GetLocalGetIndex(arg0)), module.TypeRefF32),
					),
					mod.F32(0),
				)
			}
			fl := compiler.CurrentFlow
			temp := fl.GetTempLocal(types.TypeF32)
			ret := mod.Binary(module.BinaryOpEqF32,
				mod.Binary(module.BinaryOpSubF32,
					mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefF32),
					mod.LocalGet(temp.FlowIndex(), module.TypeRefF32),
				),
				mod.F32(0),
			)
			return ret
		case types.TypeKindF64:
			if module.GetExpressionId(arg0) == module.ExpressionIdLocalGet {
				return mod.Binary(module.BinaryOpEqF64,
					mod.Binary(module.BinaryOpSubF64,
						arg0,
						mod.LocalGet(int32(module.GetLocalGetIndex(arg0)), module.TypeRefF64),
					),
					mod.F64(0),
				)
			}
			fl := compiler.CurrentFlow
			temp := fl.GetTempLocal(types.TypeF64)
			ret := mod.Binary(module.BinaryOpEqF64,
				mod.Binary(module.BinaryOpSubF64,
					mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefF64),
					mod.LocalGet(temp.FlowIndex(), module.TypeRefF64),
				),
				mod.F64(0),
			)
			return ret
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"isFinite", typ.String(), "",
	)
	return mod.Unreachable()
}
