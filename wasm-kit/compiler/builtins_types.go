// Ported from: assemblyscript/src/builtins.ts (lines 969-1348)
// Static type evaluation builtins and type query functions.
package compiler

import (
	"math/bits"
	"strconv"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// BuiltinFunctionContext holds context for compiling a builtin function call.
// Ported from: assemblyscript/src/builtins.ts BuiltinFunctionContext class (lines 801-821).
type BuiltinFunctionContext struct {
	Compiler       *Compiler
	Prototype      *program.FunctionPrototype
	TypeArguments  []*types.Type
	Operands       []ast.Node
	ThisOperand    ast.Node
	ContextualType *types.Type
	ReportNode     *ast.CallExpression
	ContextIsExact bool
}

// BuiltinFunction is the type for builtin function handlers.
type BuiltinFunction func(ctx *BuiltinFunctionContext) module.ExpressionRef

// builtinFunctions is the global map of builtin function handlers.
// Ported from: assemblyscript/src/builtins.ts builtinFunctions (line 827).
var builtinFunctions = map[string]BuiltinFunction{}

// checkConstantTypeExpr is a helper global used by checkConstantType.
// Ported from: assemblyscript/src/builtins.ts checkConstantType_expr (line 972).
var checkConstantTypeExpr module.ExpressionRef

func init() {
	// Register type query builtins
	builtinFunctions[common.BuiltinNameIsBoolean] = builtinIsBoolean
	builtinFunctions[common.BuiltinNameIsInteger] = builtinIsInteger
	builtinFunctions[common.BuiltinNameIsSigned] = builtinIsSigned
	builtinFunctions[common.BuiltinNameIsFloat] = builtinIsFloat
	builtinFunctions[common.BuiltinNameIsVector] = builtinIsVector
	builtinFunctions[common.BuiltinNameIsReference] = builtinIsReference
	builtinFunctions[common.BuiltinNameIsString] = builtinIsString
	builtinFunctions[common.BuiltinNameIsArray] = builtinIsArray
	builtinFunctions[common.BuiltinNameIsArrayLike] = builtinIsArrayLike
	builtinFunctions[common.BuiltinNameIsFunction] = builtinIsFunction
	builtinFunctions[common.BuiltinNameIsNullable] = builtinIsNullable
	builtinFunctions[common.BuiltinNameIsDefined] = builtinIsDefined
	builtinFunctions[common.BuiltinNameIsConstant] = builtinIsConstant
	builtinFunctions[common.BuiltinNameIsManaged] = builtinIsManaged
	builtinFunctions[common.BuiltinNameIsVoid] = builtinIsVoid
	builtinFunctions[common.BuiltinNameLengthof] = builtinLengthof
	builtinFunctions[common.BuiltinNameSizeof] = builtinSizeof
	builtinFunctions[common.BuiltinNameAlignof] = builtinAlignof
	builtinFunctions[common.BuiltinNameOffsetof] = builtinOffsetof
	builtinFunctions[common.BuiltinNameNameof] = builtinNameof
	builtinFunctions[common.BuiltinNameIdof] = builtinIdof
	builtinFunctions[common.BuiltinNameChangetype] = builtinChangetype
	builtinFunctions[common.BuiltinNameAssert] = builtinAssert
}

// === Helper functions ========================================================================

// checkConstantType checks and resolves the type for a constant type check builtin.
// Ported from: assemblyscript/src/builtins.ts checkConstantType (lines 11147-11188).
func checkConstantType(ctx *BuiltinFunctionContext) *types.Type {
	compiler := ctx.Compiler
	operands := ctx.Operands
	typeArguments := ctx.TypeArguments
	checkConstantTypeExpr = 0
	if len(operands) == 0 { // requires type argument
		if typeArguments == nil || len(typeArguments) != 1 {
			numStr := "0"
			if typeArguments != nil {
				numStr = strconv.Itoa(len(typeArguments))
			}
			compiler.Error(
				diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
				typeArgsRange(ctx),
				"1", numStr, "",
			)
			return nil
		}
		return typeArguments[0]
	}
	if len(operands) == 1 { // optional type argument
		if typeArguments != nil && len(typeArguments) > 0 {
			if len(typeArguments) > 1 {
				compiler.Error(
					diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
					typeArgsRange(ctx),
					"1", strconv.Itoa(len(typeArguments)), "",
				)
				return nil
			}
			checkConstantTypeExpr = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit)
		} else {
			checkConstantTypeExpr = compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
		}
		return compiler.CurrentType
	}
	if typeArguments != nil && len(typeArguments) > 1 {
		compiler.Error(
			diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
			typeArgsRange(ctx),
			"1", strconv.Itoa(len(typeArguments)), "",
		)
	}
	compiler.Error(
		diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
		argsRange(ctx),
		"1", strconv.Itoa(len(operands)), "",
	)
	return nil
}

// reifyConstantType reifies a constant type check potentially involving an expression.
// Ported from: assemblyscript/src/builtins.ts reifyConstantType (lines 11191-11200).
func reifyConstantType(ctx *BuiltinFunctionContext, expr module.ExpressionRef) module.ExpressionRef {
	mod := ctx.Compiler.Module()
	if checkConstantTypeExpr != 0 && module.MustPreserveSideEffects(checkConstantTypeExpr, mod.BinaryenModule()) {
		expr = mod.Block("", []module.ExpressionRef{
			mod.MaybeDrop(checkConstantTypeExpr),
			expr,
		}, module.GetExpressionType(expr))
	}
	return expr
}

// checkTypeRequired checks a call with a single required type argument. Returns true on error.
// Ported from: assemblyscript/src/builtins.ts checkTypeRequired (lines 11270-11289).
func checkTypeRequired(ctx *BuiltinFunctionContext, setCurrentTypeOnError bool) bool {
	compiler := ctx.Compiler
	typeArguments := ctx.TypeArguments
	if typeArguments != nil {
		numTypeArguments := len(typeArguments)
		if numTypeArguments == 1 {
			return false
		}
		// assert(numTypeArguments > 0) // invalid if 0, must not be set at all instead
		if setCurrentTypeOnError {
			compiler.CurrentType = typeArguments[0]
		}
		compiler.Error(
			diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
			typeArgsRange(ctx),
			"1", strconv.Itoa(numTypeArguments), "",
		)
	} else {
		compiler.Error(
			diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
			ctx.ReportNode.GetRange(),
			"1", "0", "",
		)
	}
	return true
}

// checkTypeOptional checks a call with a single optional type argument. Returns true on error.
// Ported from: assemblyscript/src/builtins.ts checkTypeOptional (lines 11292-11307).
func checkTypeOptional(ctx *BuiltinFunctionContext, setCurrentTypeOnError bool) bool {
	typeArguments := ctx.TypeArguments
	if typeArguments != nil {
		compiler := ctx.Compiler
		numTypeArguments := len(typeArguments)
		if numTypeArguments == 1 {
			return false
		}
		// assert(numTypeArguments > 0) // invalid if 0, must not be set at all instead
		if setCurrentTypeOnError {
			compiler.CurrentType = typeArguments[0]
		}
		compiler.Error(
			diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
			typeArgsRange(ctx),
			"1", strconv.Itoa(numTypeArguments), "",
		)
		return true
	}
	return false
}

// checkTypeAbsent checks a call that is not generic. Returns true on error.
// Ported from: assemblyscript/src/builtins.ts checkTypeAbsent (lines 11310-11321).
func checkTypeAbsent(ctx *BuiltinFunctionContext) bool {
	typeArguments := ctx.TypeArguments
	if typeArguments != nil {
		prototype := ctx.Prototype
		prototype.GetProgram().Error(
			diagnostics.DiagnosticCodeType0IsNotGeneric,
			typeArgsRange(ctx),
			prototype.GetInternalName(), "", "",
		)
		return true
	}
	return false
}

// checkArgsRequired checks a call that requires a fixed number of arguments. Returns true on error.
// Ported from: assemblyscript/src/builtins.ts checkArgsRequired (lines 11324-11334).
func checkArgsRequired(ctx *BuiltinFunctionContext, expected int) bool {
	operands := ctx.Operands
	if len(operands) != expected {
		ctx.Compiler.Error(
			diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
			ctx.ReportNode.GetRange(),
			strconv.Itoa(expected), strconv.Itoa(len(operands)), "",
		)
		return true
	}
	return false
}

// checkArgsOptional checks a call that requires a variable number of arguments. Returns true on error.
// Ported from: assemblyscript/src/builtins.ts checkArgsOptional (lines 11337-11354).
func checkArgsOptional(ctx *BuiltinFunctionContext, expectedMinimum, expectedMaximum int) bool {
	operands := ctx.Operands
	numOperands := len(operands)
	if numOperands < expectedMinimum {
		ctx.Compiler.Error(
			diagnostics.DiagnosticCodeExpectedAtLeast0ArgumentsButGot1,
			ctx.ReportNode.GetRange(),
			strconv.Itoa(expectedMinimum), strconv.Itoa(numOperands), "",
		)
		return true
	} else if numOperands > expectedMaximum {
		ctx.Compiler.Error(
			diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
			ctx.ReportNode.GetRange(),
			strconv.Itoa(expectedMaximum), strconv.Itoa(numOperands), "",
		)
		return true
	}
	return false
}

// contextualUsize makes a usize constant matching contextual type if reasonable.
// Ported from: assemblyscript/src/builtins.ts contextualUsize (lines 11357-11394).
func contextualUsize(compiler *Compiler, value int64, contextualType *types.Type) module.ExpressionRef {
	mod := compiler.Module()
	// Check if contextual type fits
	if contextualType != types.TypeAuto && contextualType.IsIntegerValue() {
		switch contextualType.Kind {
		case types.TypeKindI32:
			if value >= -2147483648 && value <= 2147483647 {
				compiler.CurrentType = types.TypeI32
				return mod.I32(int32(value))
			}
		case types.TypeKindU32:
			if value >= 0 && value <= 4294967295 {
				compiler.CurrentType = types.TypeU32
				return mod.I32(int32(value))
			}
		case types.TypeKindI64, types.TypeKindU64:
			compiler.CurrentType = contextualType
			return mod.I64(value)
			// isize/usize falls through
			// small int is probably not intended
		}
	}
	// Default to usize
	if compiler.Options().IsWasm64() {
		compiler.CurrentType = types.TypeUsize64
		return mod.I64(value)
	} else {
		compiler.CurrentType = types.TypeUsize32
		return mod.I32(int32(value))
	}
}

// boolToI32 converts a bool to an i32 constant (1 or 0).
func boolToI32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// === Static type evaluation builtins =========================================================

// builtinIsBoolean implements isBoolean<T!>() / isBoolean<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isBoolean (lines 975-983).
func builtinIsBoolean(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsBooleanValue())))
}

// builtinIsInteger implements isInteger<T!>() / isInteger<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isInteger (lines 986-994).
func builtinIsInteger(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsIntegerValue())))
}

// builtinIsSigned implements isSigned<T!>() / isSigned<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isSigned (lines 997-1005).
func builtinIsSigned(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsSignedIntegerValue())))
}

// builtinIsFloat implements isFloat<T!>() / isFloat<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isFloat (lines 1008-1016).
func builtinIsFloat(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsFloatValue())))
}

// builtinIsVector implements isVector<T!>() / isVector<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isVector (lines 1019-1027).
func builtinIsVector(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsVectorValue())))
}

// builtinIsReference implements isReference<T!>() / isReference<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isReference (lines 1030-1038).
func builtinIsReference(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsReference())))
}

// builtinIsString implements isString<T!>() / isString<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isString (lines 1041-1056).
func builtinIsString(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	classReference := typ.GetClass()
	isStr := false
	if classReference != nil {
		stringInstance := compiler.Program.StringInstance()
		if stringInstance != nil {
			isStr = classReference.IsAssignableTo(stringInstance)
		}
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(isStr)))
}

// builtinIsArray implements isArray<T!>() / isArray<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isArray (lines 1059-1074).
func builtinIsArray(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	classRef := typ.GetClass()
	isArr := false
	if classRef != nil {
		if classInstance, ok := classRef.(*program.Class); ok {
			arrayPrototype := compiler.Program.ArrayPrototype()
			if arrayPrototype != nil {
				isArr = classInstance.ExtendsPrototype(arrayPrototype)
			}
		}
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(isArr)))
}

// builtinIsArrayLike implements isArrayLike<T!>() / isArrayLike<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isArrayLike (lines 1077-1092).
func builtinIsArrayLike(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	classRef := typ.GetClass()
	isArrLike := false
	if classRef != nil {
		if classInstance, ok := classRef.(*program.Class); ok {
			isArrLike = classInstance.IsArrayLike()
		}
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(isArrLike)))
}

// builtinIsFunction implements isFunction<T!> / isFunction<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isFunction (lines 1095-1103).
func builtinIsFunction(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsFunction())))
}

// builtinIsNullable implements isNullable<T!> / isNullable<T?>(value: T) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isNullable (lines 1106-1114).
func builtinIsNullable(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsNullableReference())))
}

// builtinIsDefined implements isDefined(expression) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isDefined (lines 1117-1137).
func builtinIsDefined(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeBool
	if checkTypeAbsent(ctx) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	element := compiler.Resolver().LookupExpression(
		ctx.Operands[0],
		compiler.CurrentFlow,
		types.TypeAuto,
		program.ReportModeSwallow,
	)
	return mod.I32(boolToI32(element != nil))
}

// builtinIsConstant implements isConstant(expression) -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isConstant (lines 1140-1158).
func builtinIsConstant(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = types.TypeBool
	if checkTypeAbsent(ctx) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	expr := compiler.CompileExpression(ctx.Operands[0], types.TypeAuto, ConstraintsNone)
	compiler.CurrentType = types.TypeBool
	if !module.MustPreserveSideEffects(expr, mod.BinaryenModule()) {
		isConst := mod.IsConstExpression(expr)
		return mod.I32(boolToI32(isConst))
	}
	return mod.Block("", []module.ExpressionRef{
		mod.MaybeDrop(expr),
		mod.I32(0),
	}, module.GetExpressionType(expr))
}

// builtinIsManaged implements isManaged<T!>() -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isManaged (lines 1161-1169).
func builtinIsManaged(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.IsManaged())))
}

// builtinIsVoid implements isVoid<T!>() -> bool.
// Ported from: assemblyscript/src/builtins.ts builtin_isVoid (lines 1172-1180).
func builtinIsVoid(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeBool
	if typ == nil {
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(boolToI32(typ.Kind == types.TypeKindVoid)))
}

// builtinLengthof implements lengthof<T!>() -> i32.
// Ported from: assemblyscript/src/builtins.ts builtin_lengthof (lines 1183-1199).
func builtinLengthof(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeI32
	if typ == nil {
		return mod.Unreachable()
	}
	signatureReference := typ.GetSignature()
	if signatureReference == nil {
		compiler.Error(
			diagnostics.DiagnosticCodeType0HasNoCallSignatures,
			ctx.ReportNode.GetRange(),
			typ.String(), "", "",
		)
		return mod.Unreachable()
	}
	return reifyConstantType(ctx, mod.I32(int32(len(signatureReference.ParameterTypes))))
}

// builtinSizeof implements sizeof<T!>() -> usize*.
// Ported from: assemblyscript/src/builtins.ts builtin_sizeof (lines 1202-1221).
func builtinSizeof(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = compiler.Options().UsizeType()
	if checkTypeRequired(ctx, false) || checkArgsRequired(ctx, 0) {
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	byteSize := typ.ByteSize()
	if byteSize == 0 {
		typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			&typeArgsRange,
			"sizeof", typ.String(), "",
		)
		return mod.Unreachable()
	}
	return contextualUsize(compiler, int64(byteSize), ctx.ContextualType)
}

// builtinAlignof implements alignof<T!>() -> usize*.
// Ported from: assemblyscript/src/builtins.ts builtin_alignof (lines 1224-1243).
func builtinAlignof(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = compiler.Options().UsizeType()
	if checkTypeRequired(ctx, false) || checkArgsRequired(ctx, 0) {
		return mod.Unreachable()
	}
	typ := ctx.TypeArguments[0]
	byteSize := typ.ByteSize()
	if !util.IsPowerOf2(byteSize) { // implies == 0
		typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			&typeArgsRange,
			"alignof", typ.String(), "",
		)
		return mod.Unreachable()
	}
	return contextualUsize(compiler, int64(bits.TrailingZeros32(uint32(byteSize))), ctx.ContextualType)
}

// builtinOffsetof implements offsetof<T!>(fieldName?: string) -> usize*.
// Ported from: assemblyscript/src/builtins.ts builtin_offsetof (lines 1246-1300).
func builtinOffsetof(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	compiler.CurrentType = compiler.Options().UsizeType()
	if checkTypeRequired(ctx, false) || checkArgsOptional(ctx, 0, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	contextualType := ctx.ContextualType
	typ := ctx.TypeArguments[0]
	classRef := typ.GetClassOrWrapper(compiler.Program)
	if classRef == nil {
		typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
		compiler.Error(
			diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
			&typeArgsRange,
			"offsetof", typ.String(), "",
		)
		if compiler.Options().IsWasm64() {
			if contextualType.IsIntegerValue() && contextualType.Size <= 32 {
				compiler.CurrentType = types.TypeU32
			}
		} else {
			if contextualType.IsIntegerValue() && contextualType.Size == 64 {
				compiler.CurrentType = types.TypeU64
			}
		}
		return mod.Unreachable()
	}
	classInstance := classRef.(*program.Class)
	if len(operands) > 0 {
		firstOperand := operands[0]
		if !ast.IsLiteralKind(firstOperand, ast.LiteralKindString) {
			compiler.Error(
				diagnostics.DiagnosticCodeStringLiteralExpected,
				operands[0].GetRange(),
				"", "", "",
			)
			return mod.Unreachable()
		}
		fieldName := firstOperand.(*ast.StringLiteralExpression).Value
		fieldMember := classInstance.GetMember(fieldName)
		if fieldMember != nil && fieldMember.GetElementKind() == program.ElementKindPropertyPrototype {
			propProto := fieldMember.(*program.PropertyPrototype)
			property := propProto.PropertyInstance
			if property != nil && property.IsField() {
				// assert(property.MemoryOffset >= 0)
				return contextualUsize(compiler, int64(property.MemoryOffset), contextualType)
			}
		}
		compiler.Error(
			diagnostics.DiagnosticCodeType0HasNoProperty1,
			firstOperand.GetRange(),
			classInstance.GetInternalName(), fieldName, "",
		)
		return mod.Unreachable()
	}
	return contextualUsize(compiler, int64(classInstance.NextMemoryOffset), contextualType)
}

// builtinNameof implements nameof<T> -> string.
// Ported from: assemblyscript/src/builtins.ts builtin_nameof (lines 1303-1325).
func builtinNameof(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	resultType := checkConstantType(ctx)
	if resultType == nil {
		stringInstance := compiler.Program.StringInstance()
		if stringInstance != nil {
			compiler.CurrentType = stringInstance.GetType()
		}
		return mod.Unreachable()
	}
	var value string
	if resultType.IsInternalReference() {
		classRef := resultType.GetClass()
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				value = classInstance.GetName()
			}
		} else {
			// assert(resultType.GetSignature() != nil)
			value = "Function"
		}
	} else {
		value = resultType.String()
	}
	return reifyConstantType(ctx, compiler.EnsureStaticString(value))
}

// builtinIdof implements idof<T> -> u32.
// Ported from: assemblyscript/src/builtins.ts builtin_idof (lines 1328-1348).
func builtinIdof(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typ := checkConstantType(ctx)
	compiler.CurrentType = types.TypeU32
	if typ == nil {
		return mod.Unreachable()
	}
	signatureReference := typ.GetSignature()
	if signatureReference != nil {
		return reifyConstantType(ctx, mod.I32(int32(signatureReference.ID)))
	}
	classRef := typ.GetClassOrWrapper(compiler.Program)
	if classRef != nil && !classRef.HasDecorator(program.DecoratorFlagsUnmanaged) {
		classInstance := classRef.(*program.Class)
		return reifyConstantType(ctx, mod.I32(int32(classInstance.Id())))
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"idof", typ.String(), "",
	)
	return mod.Unreachable()
}

// builtinChangetype implements changetype<T!>(value: *) -> T.
// Ported from: assemblyscript/src/builtins.ts builtin_changetype (lines 3532-3554).
func builtinChangetype(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	if checkTypeRequired(ctx, true) || checkArgsRequired(ctx, 1) {
		return mod.Unreachable()
	}
	operands := ctx.Operands
	toType := ctx.TypeArguments[0]
	arg0 := compiler.CompileExpression(operands[0], types.TypeAuto, ConstraintsNone)
	fromType := compiler.CurrentType
	compiler.CurrentType = toType
	if !fromType.IsChangeableTo(toType) {
		compiler.Error(
			diagnostics.DiagnosticCodeType0CannotBeChangedToType1,
			ctx.ReportNode.GetRange(),
			fromType.String(), toType.String(), "",
		)
		return mod.Unreachable()
	}
	return arg0
}

// builtinAssert implements assert<T?>(isTrueish: T, message?: string) -> T{!= null}.
// Ported from: assemblyscript/src/builtins.ts builtin_assert (lines 3557-3746).
func builtinAssert(ctx *BuiltinFunctionContext) module.ExpressionRef {
	compiler := ctx.Compiler
	mod := compiler.Module()
	typeArguments := ctx.TypeArguments
	if checkTypeOptional(ctx, true) || checkArgsOptional(ctx, 1, 2) {
		if typeArguments != nil && len(typeArguments) > 0 {
			compiler.CurrentType = typeArguments[0].NonNullableType()
		}
		return mod.Unreachable()
	}
	operands := ctx.Operands
	contextualType := ctx.ContextualType
	var arg0 module.ExpressionRef
	if typeArguments != nil && len(typeArguments) > 0 {
		arg0 = compiler.CompileExpression(operands[0], typeArguments[0], ConstraintsConvImplicit|ConstraintsMustWrap)
	} else {
		arg0 = compiler.CompileExpression(operands[0], types.TypeBool, ConstraintsMustWrap)
	}
	typ := compiler.CurrentType
	compiler.CurrentType = typ.NonNullableType()

	// omit if assertions are disabled
	if compiler.Options().NoAssert {
		return arg0
	}

	// omit if the assertion can be proven statically
	evaled := mod.RunExpression(arg0, module.ExpressionRunnerFlagsDefault, 8, 1)
	if evaled != 0 {
		evaledType := module.GetExpressionType(evaled)
		switch evaledType {
		case module.TypeRefI32:
			if module.GetConstValueI32(evaled) != 0 {
				return arg0
			}
		case module.TypeRefI64:
			if module.GetConstValueI64Low(evaled)|module.GetConstValueI64High(evaled) != 0 {
				return arg0
			}
		case module.TypeRefF32:
			if module.GetConstValueF32(evaled) != 0 {
				return arg0
			}
		case module.TypeRefF64:
			if module.GetConstValueF64(evaled) != 0 {
				return arg0
			}
		}
	}

	// otherwise call abort if the assertion is false-ish
	var abortMessage ast.Node
	if len(operands) == 2 {
		abortMessage = operands[1]
	}
	abort := compiler.makeAbort(abortMessage, ctx.ReportNode)
	compiler.CurrentType = typ.NonNullableType()
	if contextualType == types.TypeVoid { // simplify if dropped anyway
		compiler.CurrentType = types.TypeVoid
		switch typ.Kind {
		case types.TypeKindBool,
			types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			return mod.If(mod.Unary(module.UnaryOpEqzI32, arg0), abort, 0)
		case types.TypeKindI64, types.TypeKindU64:
			return mod.If(mod.Unary(module.UnaryOpEqzI64, arg0), abort, 0)
		case types.TypeKindIsize, types.TypeKindUsize:
			return mod.If(mod.Unary(module.UnaryOpEqzSize, arg0), abort, 0)
		case types.TypeKindF32:
			return mod.If(mod.Binary(module.BinaryOpEqF32, arg0, mod.F32(0)), abort, 0)
		case types.TypeKindF64:
			return mod.If(mod.Binary(module.BinaryOpEqF64, arg0, mod.F64(0)), abort, 0)
		case types.TypeKindFunc, types.TypeKindExtern,
			types.TypeKindAny, types.TypeKindEq,
			types.TypeKindStruct, types.TypeKindArray,
			types.TypeKindI31, types.TypeKindString,
			types.TypeKindStringviewWTF8, types.TypeKindStringviewWTF16,
			types.TypeKindStringviewIter:
			return mod.If(mod.RefIsNull(arg0), abort, 0)
		}
	} else {
		compiler.CurrentType = typ.NonNullableType()
		fl := compiler.CurrentFlow
		switch compiler.CurrentType.Kind {
		case types.TypeKindBool,
			types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			temp := fl.GetTempLocal(typ)
			fl.SetLocalFlag(temp.FlowIndex(), flow.LocalFlagWrapped) // arg0 is wrapped
			ret := mod.If(
				mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefI32),
				mod.LocalGet(temp.FlowIndex(), module.TypeRefI32),
				abort,
			)
			return ret
		case types.TypeKindI64, types.TypeKindU64:
			temp := fl.GetTempLocal(types.TypeI64)
			ret := mod.If(
				mod.Unary(module.UnaryOpEqzI64,
					mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefI64),
				),
				abort,
				mod.LocalGet(temp.FlowIndex(), module.TypeRefI64),
			)
			return ret
		case types.TypeKindIsize, types.TypeKindUsize:
			temp := fl.GetTempLocal(compiler.Options().UsizeType())
			ret := mod.If(
				mod.Unary(module.UnaryOpEqzSize,
					mod.LocalTee(temp.FlowIndex(), arg0, typ.IsManaged(), compiler.Options().SizeTypeRef()),
				),
				abort,
				mod.LocalGet(temp.FlowIndex(), compiler.Options().SizeTypeRef()),
			)
			return ret
		case types.TypeKindF32:
			temp := fl.GetTempLocal(types.TypeF32)
			ret := mod.If(
				mod.Binary(module.BinaryOpEqF32,
					mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefF32),
					mod.F32(0),
				),
				abort,
				mod.LocalGet(temp.FlowIndex(), module.TypeRefF32),
			)
			return ret
		case types.TypeKindF64:
			temp := fl.GetTempLocal(types.TypeF64)
			ret := mod.If(
				mod.Binary(module.BinaryOpEqF64,
					mod.LocalTee(temp.FlowIndex(), arg0, false, module.TypeRefF64),
					mod.F64(0),
				),
				abort,
				mod.LocalGet(temp.FlowIndex(), module.TypeRefF64),
			)
			return ret
		case types.TypeKindFunc, types.TypeKindExtern,
			types.TypeKindAny, types.TypeKindEq,
			types.TypeKindStruct, types.TypeKindArray,
			types.TypeKindI31, types.TypeKindString,
			types.TypeKindStringviewWTF8, types.TypeKindStringviewWTF16,
			types.TypeKindStringviewIter:
			temp := fl.GetTempLocal(typ)
			ret := mod.If(
				mod.RefIsNull(
					mod.LocalTee(temp.FlowIndex(), arg0, false, typ.ToRef()),
				),
				abort,
				mod.LocalGet(temp.FlowIndex(), typ.ToRef()),
			)
			return ret
		}
	}
	typeArgsRange := ctx.ReportNode.TypeArgumentsRange()
	compiler.Error(
		diagnostics.DiagnosticCodeOperation0CannotBeAppliedToType1,
		&typeArgsRange,
		"assert", compiler.CurrentType.String(), "",
	)
	return abort
}
