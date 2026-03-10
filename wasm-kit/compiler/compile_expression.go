// Ported from: assemblyscript/src/compiler.ts compileExpression (lines 3431-3459),
// compileExpressionBuiltin (lines 3461-3475), compileExpressionRetainType (lines 3477-3510),
// and all individual expression compilation methods (lines 3512-4792).
package compiler

import (
	"fmt"
	"math"
	"strconv"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
	"github.com/brainlet/brainkit/wasm-kit/types"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// CompileExpression compiles an expression and optionally converts to the contextual type.
// Ported from: assemblyscript/src/compiler.ts compileExpression (lines 3431-3459).
func (c *Compiler) CompileExpression(expression ast.Node, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	// Compile the inner expression based on its kind
	expr := c.compileExpressionInner(expression, contextualType, constraints)

	// Perform type conversion if needed
	ct := c.CurrentType
	if ct != contextualType && contextualType != types.TypeVoid {
		if constraints&(ConstraintsConvImplicit|ConstraintsConvExplicit) != 0 {
			expr = c.convertExpression(expr, ct, contextualType, constraints&ConstraintsConvExplicit != 0, expression)
			c.CurrentType = contextualType
		}
	}

	return expr
}

// compileExpressionInner compiles an expression without type conversion.
// Ported from: assemblyscript/src/compiler.ts compileExpression switch (lines 3431-3459).
func (c *Compiler) compileExpressionInner(expression ast.Node, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()

	switch expression.GetKind() {
	case ast.NodeKindAssertion:
		return c.compileAssertionExpression(expression.(*ast.AssertionExpression), contextualType, constraints)
	case ast.NodeKindBinary:
		return c.compileBinaryExpression(expression.(*ast.BinaryExpression), contextualType, constraints)
	case ast.NodeKindCall:
		return c.compileCallExpression(expression.(*ast.CallExpression), contextualType, constraints)
	case ast.NodeKindComma:
		return c.compileCommaExpression(expression.(*ast.CommaExpression), contextualType, constraints)
	case ast.NodeKindElementAccess:
		return c.compileElementAccessExpression(expression.(*ast.ElementAccessExpression), contextualType, constraints)
	case ast.NodeKindFunction:
		return c.compileFunctionExpression(expression.(*ast.FunctionExpression), contextualType, constraints)
	case ast.NodeKindIdentifier, ast.NodeKindFalse, ast.NodeKindNull, ast.NodeKindThis, ast.NodeKindTrue, ast.NodeKindSuper, ast.NodeKindConstructor:
		return c.compileIdentifierExpression(expression.(*ast.IdentifierExpression), contextualType, constraints)
	case ast.NodeKindInstanceOf:
		return c.compileInstanceOfExpression(expression.(*ast.InstanceOfExpression), contextualType, constraints)
	case ast.NodeKindLiteral:
		return c.compileLiteralExpression(expression, contextualType, constraints)
	case ast.NodeKindNew:
		return c.compileNewExpression(expression.(*ast.NewExpression), contextualType, constraints)
	case ast.NodeKindParenthesized:
		return c.compileParenthesizedExpression(expression.(*ast.ParenthesizedExpression), contextualType, constraints)
	case ast.NodeKindPropertyAccess:
		return c.compilePropertyAccessExpression(expression.(*ast.PropertyAccessExpression), contextualType, constraints)
	case ast.NodeKindTernary:
		return c.compileTernaryExpression(expression.(*ast.TernaryExpression), contextualType, constraints)
	case ast.NodeKindUnaryPostfix:
		return c.compileUnaryPostfixExpression(expression.(*ast.UnaryPostfixExpression), contextualType, constraints)
	case ast.NodeKindUnaryPrefix:
		return c.compileUnaryPrefixExpression(expression.(*ast.UnaryPrefixExpression), contextualType, constraints)
	case ast.NodeKindCompiled:
		compiled := expression.(*ast.CompiledExpression)
		c.CurrentType = compiled.Type.(*types.Type)
		return compiled.Expr.(module.ExpressionRef)
	default:
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"Expression kind "+strconv.Itoa(int(expression.GetKind())), "", "",
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}
}

// compileAssertionExpression compiles an assertion expression (as, prefix cast, non-null).
// Ported from: assemblyscript/src/compiler.ts compileAssertionExpression (lines 3795-3851).
func (c *Compiler) compileAssertionExpression(expression *ast.AssertionExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	inheritedConstraints := constraints &^ (ConstraintsConvImplicit | ConstraintsConvExplicit)
	switch expression.AssertionKind {
	case ast.AssertionKindPrefix, ast.AssertionKindAs:
		// Type assertion: resolve target type, compile with explicit conversion constraint
		fl := c.CurrentFlow
		toType := c.Resolver().ResolveType(
			expression.ToType,
			fl,
			fl.TargetFunction.(program.Element),
			util.CloneMap(fl.ContextualTypeArguments()),
			program.ReportModeReport,
		)
		if toType == nil {
			return c.Module().Unreachable()
		}
		return c.CompileExpression(expression.Expression, toType, inheritedConstraints|ConstraintsConvExplicit)

	case ast.AssertionKindNonNull:
		// Non-null assertion: compile expression, optionally insert runtime check
		expr := c.CompileExpression(expression.Expression, contextualType.ExceptVoid(), inheritedConstraints)
		typ := c.CurrentType
		if c.CurrentFlow.IsNonnull(expr, typ) {
			c.Info(
				diagnostics.DiagnosticCodeExpressionIsNeverNull,
				expression.Expression.GetRange(),
				"", "", "",
			)
		} else if !c.Options().NoAssert {
			expr = c.makeRuntimeNonNullCheck(expr, typ, expression)
		}
		c.CurrentType = typ.NonNullableType()
		return expr

	case ast.AssertionKindConst:
		// TODO: decide on the layout of ReadonlyArray first
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"Const assertion", "", "",
		)
		return c.Module().Unreachable()

	default:
		panic("unexpected assertion kind")
	}
}

// compileBinaryExpression compiles a binary expression.
// Ported from: assemblyscript/src/compiler.ts compileBinaryExpression (lines 3642-4045).
// compileBinaryExpression compiles a binary expression.
// Ported from: assemblyscript/src/compiler.ts compileBinaryExpression (lines 4045-4791).
func (c *Compiler) compileBinaryExpression(expression *ast.BinaryExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	left := expression.Left
	right := expression.Right

	var leftExpr, rightExpr, expr module.ExpressionRef
	var leftType, rightType, commonType *types.Type
	compound := false

	operator := expression.Operator
	switch operator {
	// Comparison operators — delegated to separate methods
	case tokenizer.TokenLessThan,
		tokenizer.TokenGreaterThan,
		tokenizer.TokenLessThanEquals,
		tokenizer.TokenGreaterThanEquals:
		return c.compileNonCommutativeCompareBinaryExpression(expression, contextualType)

	case tokenizer.TokenEqualsEquals,
		tokenizer.TokenEqualsEqualsEquals,
		tokenizer.TokenExclamationEquals,
		tokenizer.TokenExclamationEqualsEquals:
		return c.compileCommutativeCompareBinaryExpression(expression, contextualType)

	case tokenizer.TokenEquals:
		return c.compileAssignment(left, right, contextualType)

	case tokenizer.TokenPlusEquals:
		compound = true
		fallthrough
	case tokenizer.TokenPlus:
		leftExpr = c.CompileExpression(left, contextualType, ConstraintsNone)
		leftType = c.CurrentType

		// check operator overload
		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindAdd); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "+", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "+", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinaryAdd(leftExpr, rightExpr, commonType)

	case tokenizer.TokenMinusEquals:
		compound = true
		fallthrough
	case tokenizer.TokenMinus:
		leftExpr = c.CompileExpression(left, contextualType, ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindSub); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "-", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "-", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinarySub(leftExpr, rightExpr, commonType)

	case tokenizer.TokenAsteriskEquals:
		compound = true
		fallthrough
	case tokenizer.TokenAsterisk:
		leftExpr = c.CompileExpression(left, contextualType, ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindMul); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "*", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "*", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinaryMul(leftExpr, rightExpr, commonType)

	case tokenizer.TokenAsteriskAsteriskEquals:
		compound = true
		fallthrough
	case tokenizer.TokenAsteriskAsterisk:
		leftExpr = c.CompileExpression(left, contextualType, ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindPow); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "**", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "**", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.compilePow(leftExpr, rightExpr, commonType, expression)

	case tokenizer.TokenSlashEquals:
		compound = true
		fallthrough
	case tokenizer.TokenSlash:
		leftExpr = c.CompileExpression(left, contextualType, ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindDiv); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "/", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "/", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinaryDiv(leftExpr, rightExpr, commonType)

	case tokenizer.TokenPercentEquals:
		compound = true
		fallthrough
	case tokenizer.TokenPercent:
		leftExpr = c.CompileExpression(left, contextualType, ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindRem); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "%", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsNumericValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "%", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinaryRem(leftExpr, rightExpr, commonType)

	case tokenizer.TokenLessThanLessThanEquals:
		compound = true
		fallthrough
	case tokenizer.TokenLessThanLessThan:
		leftExpr = c.CompileExpression(left, contextualType.IntType(), ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindBitwiseShl); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}
		if !leftType.IsIntegerValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "<<", leftType.ToString(false), "")
			return mod.Unreachable()
		}
		rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
		rightType = c.CurrentType
		expr = c.makeBinaryShl(leftExpr, rightExpr, rightType)

	case tokenizer.TokenGreaterThanGreaterThanEquals:
		compound = true
		fallthrough
	case tokenizer.TokenGreaterThanGreaterThan:
		leftExpr = c.CompileExpression(left, contextualType.IntType(), ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindBitwiseShr); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}
		if !leftType.IsIntegerValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), ">>", leftType.ToString(false), "")
			return mod.Unreachable()
		}
		rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
		rightType = c.CurrentType
		expr = c.makeBinaryShr(leftExpr, rightExpr, rightType, true)

	case tokenizer.TokenGreaterThanGreaterThanGreaterThanEquals:
		compound = true
		fallthrough
	case tokenizer.TokenGreaterThanGreaterThanGreaterThan:
		leftExpr = c.CompileExpression(left, contextualType.IntType(), ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindBitwiseShrU); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}
		if !leftType.IsIntegerValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), ">>>", leftType.ToString(false), "")
			return mod.Unreachable()
		}
		rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
		rightType = c.CurrentType
		expr = c.makeBinaryShr(leftExpr, rightExpr, rightType, false)

	case tokenizer.TokenAmpersandEquals:
		compound = true
		fallthrough
	case tokenizer.TokenAmpersand:
		leftExpr = c.CompileExpression(left, contextualType.IntType(), ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindBitwiseAnd); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsIntegerValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "&", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsIntegerValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "&", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinaryAnd(leftExpr, rightExpr, commonType)

	case tokenizer.TokenBarEquals:
		compound = true
		fallthrough
	case tokenizer.TokenBar:
		leftExpr = c.CompileExpression(left, contextualType.IntType(), ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindBitwiseOr); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsIntegerValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "|", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsIntegerValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "|", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinaryOr(leftExpr, rightExpr, commonType)

	case tokenizer.TokenCaretEquals:
		compound = true
		fallthrough
	case tokenizer.TokenCaret:
		leftExpr = c.CompileExpression(left, contextualType.IntType(), ConstraintsNone)
		leftType = c.CurrentType

		if overload := c.lookupBinaryOverload(leftType, program.OperatorKindBitwiseXor); overload != nil {
			expr = c.compileBinaryOverload(overload, left, leftExpr, leftType, right, expression)
			break
		}

		if compound {
			if !leftType.IsIntegerValue() {
				c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
					expression.GetRange(), "^", leftType.ToString(false), "")
				return mod.Unreachable()
			}
			rightExpr = c.CompileExpression(right, leftType, ConstraintsConvImplicit)
			rightType = c.CurrentType
			commonType = rightType
		} else {
			rightExpr = c.CompileExpression(right, leftType, ConstraintsNone)
			rightType = c.CurrentType
			commonType = types.CommonType(leftType, rightType, contextualType, true)
			if commonType == nil || !commonType.IsIntegerValue() {
				c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(), "^", leftType.ToString(false), rightType.ToString(false))
				c.CurrentType = contextualType
				return mod.Unreachable()
			}
			leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
			leftType = commonType
			rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)
			rightType = commonType
		}
		expr = c.makeBinaryXor(leftExpr, rightExpr, commonType)

	// logical (no overloading)
	case tokenizer.TokenAmpersandAmpersand:
		return c.compileLogicalAnd(left, right, expression, contextualType, constraints)
	case tokenizer.TokenBarBar:
		return c.compileLogicalOr(left, right, expression, contextualType, constraints)

	case tokenizer.TokenIn:
		c.Error(diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(), "'in' operator", "", "")
		c.CurrentType = types.TypeBool
		return mod.Unreachable()

	default:
		c.Error(diagnostics.DiagnosticCodeNotImplemented0, expression.GetRange(), "Binary operator", "", "")
		c.CurrentType = contextualType
		return mod.Unreachable()
	}

	// Compound assignment: store the result back to the target
	if !compound {
		return expr
	}
	resolver := c.Resolver()
	target := resolver.LookupExpression(left, c.CurrentFlow, nil, program.ReportModeReport)
	if target == nil {
		return mod.Unreachable()
	}
	targetType := resolver.GetTypeOfElement(target)
	if targetType == nil {
		targetType = types.TypeVoid
	}
	if !c.CurrentType.IsStrictlyAssignableTo(targetType, true) {
		c.Error(
			diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
			expression.GetRange(),
			c.CurrentType.ToString(false), targetType.ToString(false), "",
		)
		return mod.Unreachable()
	}
	return c.makeAssignment(
		target,
		expr,
		c.CurrentType,
		right,
		resolver.CurrentThisExpression,
		resolver.CurrentElementExpression,
		contextualType != types.TypeVoid,
	)
}

// compileNonCommutativeCompareBinaryExpression compiles >, >=, <, <= binary expressions.
// Ported from: assemblyscript/src/compiler.ts compileNonCommutativeCompareBinaryExpression (lines 3984-4043).
func (c *Compiler) compileNonCommutativeCompareBinaryExpression(expression *ast.BinaryExpression, contextualType *types.Type) module.ExpressionRef {
	mod := c.Module()
	left := expression.Left
	right := expression.Right
	operator := expression.Operator
	operatorString := tokenizer.OperatorTokenToString(operator)

	leftExpr := c.CompileExpression(left, contextualType, ConstraintsNone)
	leftType := c.CurrentType

	// check operator overload
	operatorKind := program.OperatorKindFromBinaryToken(operator)
	leftOverload := leftType.LookupOverload(operatorKind, c.Program)
	if leftOverload != nil {
		return c.compileBinaryOverload(leftOverload.(*program.Function), left, leftExpr, leftType, right, expression)
	}

	rightExpr := c.CompileExpression(right, leftType, ConstraintsNone)
	rightType := c.CurrentType

	signednessIsRelevant := true
	commonType := types.CommonType(leftType, rightType, contextualType, signednessIsRelevant)
	if commonType == nil || !commonType.IsNumericValue() {
		c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
			expression.GetRange(), operatorString, leftType.ToString(false), rightType.ToString(false))
		c.CurrentType = contextualType
		return mod.Unreachable()
	}

	leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
	rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)

	c.CurrentType = types.TypeBool
	switch operator {
	case tokenizer.TokenLessThan:
		return c.makeBinaryLt(leftExpr, rightExpr, commonType)
	case tokenizer.TokenGreaterThan:
		return c.makeBinaryGt(leftExpr, rightExpr, commonType)
	case tokenizer.TokenLessThanEquals:
		return c.makeBinaryLe(leftExpr, rightExpr, commonType)
	case tokenizer.TokenGreaterThanEquals:
		return c.makeBinaryGe(leftExpr, rightExpr, commonType)
	default:
		return mod.Unreachable()
	}
}

// compileCommutativeCompareBinaryExpression compiles ==, ===, !=, !== binary expressions.
// Ported from: assemblyscript/src/compiler.ts compileCommutativeCompareBinaryExpression (lines 3861-3881).
func (c *Compiler) compileCommutativeCompareBinaryExpression(expression *ast.BinaryExpression, contextualType *types.Type) module.ExpressionRef {
	left := expression.Left
	leftExpr := c.CompileExpression(left, contextualType, ConstraintsNone)
	leftType := c.CurrentType

	right := expression.Right
	rightExpr := c.CompileExpression(right, leftType, ConstraintsNone)
	rightType := c.CurrentType

	return c.compileCommutativeCompareBinaryExpressionFromParts(
		expression.Operator,
		left, leftExpr, leftType,
		right, rightExpr, rightType,
		contextualType, expression,
	)
}

// compileCommutativeCompareBinaryExpressionFromParts compiles ==, ===, !=, !== from pre-compiled parts.
// Ported from: assemblyscript/src/compiler.ts compileCommutativeCompareBinaryExpressionFromParts (lines 3889-3981).
func (c *Compiler) compileCommutativeCompareBinaryExpressionFromParts(
	operator tokenizer.Token,
	left ast.Node, leftExpr module.ExpressionRef, leftType *types.Type,
	right ast.Node, rightExpr module.ExpressionRef, rightType *types.Type,
	contextualType *types.Type,
	reportNode ast.Node,
) module.ExpressionRef {
	mod := c.Module()
	operatorString := tokenizer.OperatorTokenToString(operator)

	// check operator overload
	operatorKind := program.OperatorKindFromBinaryToken(operator)
	leftOverload := leftType.LookupOverload(operatorKind, c.Program)
	rightOverload := rightType.LookupOverload(operatorKind, c.Program)

	var leftOverloadFn, rightOverloadFn *program.Function
	if leftOverload != nil {
		leftOverloadFn = leftOverload.(*program.Function)
	}
	if rightOverload != nil {
		rightOverloadFn = rightOverload.(*program.Function)
	}

	if leftOverloadFn != nil && rightOverloadFn != nil && leftOverloadFn != rightOverloadFn {
		c.Error(
			diagnostics.DiagnosticCodeAmbiguousOperatorOverload0ConflictingOverloads1And2,
			reportNode.GetRange(),
			operatorString,
			leftOverloadFn.GetInternalName(),
			rightOverloadFn.GetInternalName(),
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}
	if leftOverloadFn != nil {
		return c.compileCommutativeBinaryOverload(
			leftOverloadFn,
			left, leftExpr, leftType,
			right, rightExpr, rightType,
			reportNode,
		)
	}
	if rightOverloadFn != nil {
		return c.compileCommutativeBinaryOverload(
			rightOverloadFn,
			right, rightExpr, rightType,
			left, leftExpr, leftType,
			reportNode,
		)
	}

	signednessIsRelevant := false
	commonType := types.CommonType(leftType, rightType, contextualType, signednessIsRelevant)
	if commonType == nil {
		c.Error(diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
			reportNode.GetRange(), operatorString, leftType.ToString(false), rightType.ToString(false))
		c.CurrentType = contextualType
		return mod.Unreachable()
	}

	leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
	rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)

	c.CurrentType = types.TypeBool
	switch operator {
	case tokenizer.TokenEqualsEquals, tokenizer.TokenEqualsEqualsEquals:
		return c.makeBinaryEq(leftExpr, rightExpr, commonType)
	case tokenizer.TokenExclamationEquals, tokenizer.TokenExclamationEqualsEquals:
		return c.makeBinaryNe(leftExpr, rightExpr, commonType)
	default:
		return mod.Unreachable()
	}
}

// compileBinaryOverload compiles a non-commutative binary operator overload.
// Ported from: assemblyscript/src/compiler.ts compileBinaryOverload (lines 5611-5631).
// compileUnaryOverload compiles a unary operator overload call.
// Ported from: assemblyscript/src/compiler.ts compileUnaryOverload (lines 5600-5609).
func (c *Compiler) compileUnaryOverload(
	operatorInstance *program.Function,
	value ast.Node, valueExpr module.ExpressionRef,
	reportNode ast.Node,
) module.ExpressionRef {
	return c.makeCallDirect(operatorInstance, []module.ExpressionRef{valueExpr}, reportNode, false)
}

func (c *Compiler) compileBinaryOverload(
	operatorInstance *program.Function,
	left ast.Node, leftExpr module.ExpressionRef, leftType *types.Type,
	right ast.Node,
	reportNode ast.Node,
) module.ExpressionRef {
	var rightType *types.Type
	signature := operatorInstance.Signature
	parameterTypes := signature.ParameterTypes
	if operatorInstance.Is(common.CommonFlagsInstance) {
		leftExpr = c.convertExpression(leftExpr, leftType, signature.ThisType, false, left)
		rightType = parameterTypes[0]
	} else {
		leftExpr = c.convertExpression(leftExpr, leftType, parameterTypes[0], false, left)
		rightType = parameterTypes[1]
	}
	rightExpr := c.CompileExpression(right, rightType, ConstraintsConvImplicit)
	return c.makeCallDirect(operatorInstance, []module.ExpressionRef{leftExpr, rightExpr}, reportNode, false)
}

// compileCommutativeBinaryOverload compiles a commutative binary operator overload (== != === !==).
// Ported from: assemblyscript/src/compiler.ts compileCommutativeBinaryOverload (lines 5634-5654).
func (c *Compiler) compileCommutativeBinaryOverload(
	operatorInstance *program.Function,
	first ast.Node, firstExpr module.ExpressionRef, firstType *types.Type,
	second ast.Node, secondExpr module.ExpressionRef, secondType *types.Type,
	reportNode ast.Node,
) module.ExpressionRef {
	signature := operatorInstance.Signature
	parameterTypes := signature.ParameterTypes
	if operatorInstance.Is(common.CommonFlagsInstance) {
		firstExpr = c.convertExpression(firstExpr, firstType, signature.ThisType, false, first)
		secondExpr = c.convertExpression(secondExpr, secondType, parameterTypes[0], false, second)
	} else {
		firstExpr = c.convertExpression(firstExpr, firstType, parameterTypes[0], false, first)
		secondExpr = c.convertExpression(secondExpr, secondType, parameterTypes[1], false, second)
	}
	return c.makeCallDirect(operatorInstance, []module.ExpressionRef{firstExpr, secondExpr}, reportNode, false)
}

// makeAssignment stores a compiled value to a resolved target element.
// Ported from: assemblyscript/src/compiler.ts makeAssignment (lines 5792-5936).
func (c *Compiler) makeAssignment(
	target program.Element,
	valueExpr module.ExpressionRef,
	valueType *types.Type,
	valueExpression ast.Node,
	thisExpression ast.Node,
	indexExpression ast.Node,
	tee bool,
) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	switch target.GetElementKind() {
	case program.ElementKindLocal:
		local := target.(*program.Local)
		localIndex := local.FlowIndex()
		localType := local.GetResolvedType()
		if fl.IsLocalFlag(localIndex, flow.LocalFlagConstant) {
			c.Error(
				diagnostics.DiagnosticCodeCannotAssignTo0BecauseItIsAConstantOrAReadOnlyProperty,
				valueExpression.GetRange(),
				target.GetInternalName(), "", "",
			)
			if tee {
				c.CurrentType = localType
			} else {
				c.CurrentType = types.TypeVoid
			}
			return mod.Unreachable()
		}
		return c.makeLocalAssignment(local, valueExpr, valueType, tee)

	case program.ElementKindGlobal:
		global := target.(*program.Global)
		if !c.CompileGlobalLazy(global, valueExpression) {
			return mod.Unreachable()
		}
		globalType := global.GetResolvedType()
		if target.IsAny(common.CommonFlagsConst | common.CommonFlagsReadonly) {
			c.Error(
				diagnostics.DiagnosticCodeCannotAssignTo0BecauseItIsAConstantOrAReadOnlyProperty,
				valueExpression.GetRange(),
				target.GetInternalName(), "", "",
			)
			if tee {
				c.CurrentType = globalType
			} else {
				c.CurrentType = types.TypeVoid
			}
			return mod.Unreachable()
		}
		return c.makeGlobalAssignment(global, valueExpr, valueType, tee)

	case program.ElementKindPropertyPrototype:
		propertyPrototype := target.(*program.PropertyPrototype)
		resolvedProp := c.Resolver().ResolveProperty(propertyPrototype, program.ReportModeReport)
		if resolvedProp == nil {
			return mod.Unreachable()
		}
		target = resolvedProp
		fallthrough

	case program.ElementKindProperty:
		propertyInstance := target.(*program.Property)
		if propertyInstance.IsField() {
			// Cannot assign to readonly fields except in constructors if there's no initializer
			isConstructor := fl.SourceFunction().Is(common.CommonFlagsConstructor)
			if propertyInstance.Is(common.CommonFlagsReadonly) {
				initializerNode := propertyInstance.InitializerNode()
				if !isConstructor || initializerNode != nil {
					c.Error(
						diagnostics.DiagnosticCodeCannotAssignTo0BecauseItIsAConstantOrAReadOnlyProperty,
						valueExpression.GetRange(),
						propertyInstance.GetInternalName(), "", "",
					)
					return mod.Unreachable()
				}
			}
			// Mark initialized fields in constructors
			if isConstructor && thisExpression != nil && thisExpression.GetKind() == ast.NodeKindThis {
				fl.SetThisFieldFlag(propertyInstance, flow.FieldFlagInitialized)
			}
		}
		setterInstance := propertyInstance.SetterInstance
		if setterInstance == nil {
			c.Error(
				diagnostics.DiagnosticCodeCannotAssignTo0BecauseItIsAConstantOrAReadOnlyProperty,
				valueExpression.GetRange(),
				target.GetInternalName(), "", "",
			)
			return mod.Unreachable()
		}
		if propertyInstance.Is(common.CommonFlagsInstance) {
			thisType := setterInstance.Signature.ThisType
			thisExpr := c.CompileExpression(thisExpression, thisType, ConstraintsConvImplicit|ConstraintsIsThis)
			if !tee {
				return c.makeCallDirect(setterInstance, []module.ExpressionRef{thisExpr, valueExpr}, valueExpression, false)
			}
			tempLocal := fl.GetTempLocal(valueType)
			tempIndex := tempLocal.FlowIndex()
			valueTypeRef := valueType.ToRef()
			ret := mod.Block("", []module.ExpressionRef{
				c.makeCallDirect(setterInstance, []module.ExpressionRef{
					thisExpr,
					mod.LocalTee(tempIndex, valueExpr, valueType.IsManaged(), valueTypeRef),
				}, valueExpression, false),
				mod.LocalGet(tempIndex, valueTypeRef),
			}, valueTypeRef)
			c.CurrentType = valueType
			return ret
		} else {
			if !tee {
				return c.makeCallDirect(setterInstance, []module.ExpressionRef{valueExpr}, valueExpression, false)
			}
			tempLocal := fl.GetTempLocal(valueType)
			tempIndex := tempLocal.FlowIndex()
			valueTypeRef := valueType.ToRef()
			ret := mod.Block("", []module.ExpressionRef{
				c.makeCallDirect(setterInstance, []module.ExpressionRef{
					mod.LocalTee(tempIndex, valueExpr, valueType.IsManaged(), valueTypeRef),
				}, valueExpression, false),
				mod.LocalGet(tempIndex, valueTypeRef),
			}, valueTypeRef)
			c.CurrentType = valueType
			return ret
		}

	case program.ElementKindIndexSignature:
		indexSignature := target.(*program.IndexSignature)
		parent := indexSignature.GetParent()
		classInstance := parent.(*program.Class)
		isUnchecked := fl.Is(flow.FlowFlagUncheckedContext)
		getterInstance := classInstance.FindOverload(program.OperatorKindIndexedGet, isUnchecked)
		if getterInstance == nil {
			c.Error(
				diagnostics.DiagnosticCodeIndexSignatureIsMissingInType0,
				valueExpression.GetRange(),
				classInstance.GetInternalName(), "", "",
			)
			return mod.Unreachable()
		}
		setterInstance := classInstance.FindOverload(program.OperatorKindIndexedSet, isUnchecked)
		if setterInstance == nil {
			c.Error(
				diagnostics.DiagnosticCodeIndexSignatureInType0OnlyPermitsReading,
				valueExpression.GetRange(),
				classInstance.GetInternalName(), "", "",
			)
			if tee {
				c.CurrentType = getterInstance.Signature.ReturnType
			} else {
				c.CurrentType = types.TypeVoid
			}
			return mod.Unreachable()
		}
		thisType := classInstance.GetResolvedType()
		thisExpr := c.CompileExpression(thisExpression, thisType, ConstraintsConvImplicit|ConstraintsIsThis)
		setterIndexType := setterInstance.Signature.ParameterTypes[0]
		getterIndexType := getterInstance.Signature.ParameterTypes[0]
		if !setterIndexType.Equals(getterIndexType) {
			getterRange := getterInstance.IdentifierAndSignatureRange()
			setterRange := setterInstance.IdentifierAndSignatureRange()
			c.ErrorRelated(
				diagnostics.DiagnosticCodeIndexSignatureAccessorsInType0DifferInTypes,
				&getterRange,
				&setterRange,
				classInstance.GetInternalName(), "", "",
			)
			if tee {
				c.CurrentType = getterInstance.Signature.ReturnType
			} else {
				c.CurrentType = types.TypeVoid
			}
			return mod.Unreachable()
		}
		elementExpr := c.CompileExpression(indexExpression, setterIndexType, ConstraintsConvImplicit)
		elementType := c.CurrentType
		if tee {
			tempTarget := fl.GetTempLocal(thisType)
			tempElement := fl.GetTempLocal(elementType)
			returnType := getterInstance.Signature.ReturnType
			ret := mod.Block("", []module.ExpressionRef{
				c.makeCallDirect(setterInstance, []module.ExpressionRef{
					mod.LocalTee(tempTarget.FlowIndex(), thisExpr, thisType.IsManaged(), thisType.ToRef()),
					mod.LocalTee(tempElement.FlowIndex(), elementExpr, elementType.IsManaged(), elementType.ToRef()),
					valueExpr,
				}, valueExpression, false),
				c.makeCallDirect(getterInstance, []module.ExpressionRef{
					mod.LocalGet(tempTarget.FlowIndex(), thisType.ToRef()),
					mod.LocalGet(tempElement.FlowIndex(), elementType.ToRef()),
				}, valueExpression, false),
			}, returnType.ToRef())
			return ret
		}
		return c.makeCallDirect(setterInstance, []module.ExpressionRef{
			thisExpr,
			elementExpr,
			valueExpr,
		}, valueExpression, false)

	default:
		c.Error(diagnostics.DiagnosticCodeNotImplemented0, valueExpression.GetRange(), "Assignment target kind", "", "")
		c.CurrentType = types.TypeVoid
		return mod.Unreachable()
	}
}

// makeLocalAssignment makes an assignment to a local variable, updating flow flags.
// Ported from: assemblyscript/src/compiler.ts makeLocalAssignment (lines 5990-6022).
func (c *Compiler) makeLocalAssignment(local *program.Local, valueExpr module.ExpressionRef, valueType *types.Type, tee bool) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	localType := local.GetResolvedType()
	localIndex := local.FlowIndex()

	if localType.IsNullableReference() {
		if !valueType.IsNullableReference() || fl.IsNonnull(valueExpr, localType) {
			fl.SetLocalFlag(localIndex, flow.LocalFlagNonNull)
		} else {
			fl.UnsetLocalFlag(localIndex, flow.LocalFlagNonNull)
		}
	}
	fl.SetLocalFlag(localIndex, flow.LocalFlagInitialized)
	if localType.IsShortIntegerValue() {
		if !fl.CanOverflow(valueExpr, localType) {
			fl.SetLocalFlag(localIndex, flow.LocalFlagWrapped)
		} else {
			fl.UnsetLocalFlag(localIndex, flow.LocalFlagWrapped)
		}
	}
	if tee {
		c.CurrentType = localType
		return mod.LocalTee(localIndex, valueExpr, localType.IsManaged(), localType.ToRef())
	}
	c.CurrentType = types.TypeVoid
	return mod.LocalSet(localIndex, valueExpr, localType.IsManaged())
}

// makeGlobalAssignment makes an assignment to a global variable.
// Ported from: assemblyscript/src/compiler.ts makeGlobalAssignment (lines 6024-6053).
func (c *Compiler) makeGlobalAssignment(global program.VariableLikeElement, valueExpr module.ExpressionRef, valueType *types.Type, tee bool) module.ExpressionRef {
	mod := c.Module()
	globalType := global.GetResolvedType()
	typeRef := globalType.ToRef()

	valueExpr = c.ensureSmallIntegerWrap(valueExpr, globalType) // globals must be wrapped
	if tee {
		c.CurrentType = globalType
		return mod.Block("", []module.ExpressionRef{
			mod.GlobalSet(global.GetInternalName(), valueExpr),
			mod.GlobalGet(global.GetInternalName(), typeRef),
		}, typeRef)
	}
	c.CurrentType = types.TypeVoid
	return mod.GlobalSet(global.GetInternalName(), valueExpr)
}

// compileCallExpression compiles a call expression.
// Ported from: assemblyscript/src/compiler.ts compileCallExpression (lines 4047-4340).
func (c *Compiler) compileCallExpression(expression *ast.CallExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	// handle call to super
	if expression.Expression.GetKind() == ast.NodeKindSuper {
		sourceFunction := fl.SourceFunction()
		if !sourceFunction.Is(common.CommonFlagsConstructor) {
			c.Error(
				diagnostics.DiagnosticCodeSuperCallsAreNotPermittedOutsideConstructorsOrInNestedFunctionsInsideConstructors,
				expression.GetRange(),
				"", "", "",
			)
			return mod.Unreachable()
		}
		parent := sourceFunction.(program.Element).GetParent()
		classInstance := parent.(*program.Class)
		baseClassInstance := classInstance.Base
		if baseClassInstance == nil || classInstance.Prototype.ImplicitlyExtendsObject {
			c.Error(
				diagnostics.DiagnosticCodeSuperCanOnlyBeReferencedInADerivedClass,
				expression.Expression.GetRange(),
				"", "", "",
			)
			return mod.Unreachable()
		}
		thisLocal := fl.LookupLocal(common.CommonNameThis)
		sizeTypeRef := c.Options().UsizeType().ToRef()

		baseCtorInstance := c.ensureConstructor(baseClassInstance, expression)
		if baseCtorInstance == nil {
			return mod.Unreachable()
		}
		c.checkFieldInitialization(baseClassInstance, expression)
		superCall := c.compileCallDirect(baseCtorInstance, expression.Args, expression,
			mod.LocalGet(thisLocal.(*program.Local).Index, sizeTypeRef), ConstraintsNone)

		// check that super had been called before accessing `this`
		if fl.IsAny(flow.FlowFlagAccessesThis | flow.FlowFlagConditionallyAccessesThis) {
			c.Error(
				diagnostics.DiagnosticCodeSuperMustBeCalledBeforeAccessingThisInTheConstructorOfADerivedClass,
				expression.GetRange(),
				"", "", "",
			)
			return mod.Unreachable()
		}
		fl.SetFlag(flow.FlowFlagAccessesThis | flow.FlowFlagCallsSuper)
		c.CurrentType = types.TypeVoid
		return mod.LocalSet(thisLocal.(*program.Local).Index, superCall, classInstance.GetResolvedType().IsManaged())
	}

	// otherwise resolve normally
	resolver := c.Resolver()
	target := resolver.LookupExpression(expression.Expression, fl, nil, program.ReportModeReport)
	if target == nil {
		return mod.Unreachable()
	}
	thisExpression := resolver.CurrentThisExpression

	// handle direct call
	switch target.GetElementKind() {
	case program.ElementKindFunctionPrototype:
		functionPrototype := target.(*program.FunctionPrototype)
		// builtins handle present respectively omitted type arguments on their own
		if functionPrototype.HasDecorator(program.DecoratorFlagsBuiltin) {
			return c.compileCallExpressionBuiltin(functionPrototype, expression, contextualType)
		}
		functionInstance := resolver.MaybeInferCall(expression, functionPrototype, fl, program.ReportModeReport)
		if functionInstance == nil {
			return mod.Unreachable()
		}
		target = functionInstance
		// fall through to Function case
		fallthrough

	case program.ElementKindFunction:
		functionInstance := target.(*program.Function)
		var thisArg module.ExpressionRef
		if functionInstance.Is(common.CommonFlagsInstance) {
			if thisExpression != nil {
				thisArg = c.CompileExpression(
					thisExpression,
					functionInstance.Signature.ThisType,
					ConstraintsConvImplicit|ConstraintsIsThis,
				)
			}
		}
		return c.compileCallDirect(functionInstance, expression.Args, expression, thisArg, constraints)
	}

	// handle indirect call
	functionArg := c.CompileExpression(expression.Expression, types.TypeAuto, 0)
	sig := c.CurrentType.GetSignature()
	if sig != nil {
		return c.compileCallIndirect(sig, functionArg, expression.Args, expression, 0, contextualType == types.TypeVoid)
	}
	c.Error(
		diagnostics.DiagnosticCodeCannotInvokeAnExpressionWhoseTypeLacksACallSignatureType0HasNoCompatibleCallSignatures,
		expression.GetRange(),
		c.CurrentType.ToString(false), "", "",
	)
	// additional diagnostic hint for getter properties
	if target != nil && target.GetElementKind() == program.ElementKindPropertyPrototype {
		pp := target.(*program.PropertyPrototype)
		if pp.GetterPrototype != nil {
			c.InfoRelated(
				diagnostics.DiagnosticCodeThisExpressionIsNotCallableBecauseItIsAGetAccessorDidYouMeanToUseItWithout,
				expression.GetRange(),
				pp.GetterPrototype.IdentifierNode().GetRange(),
				"", "", "",
			)
		}
	}
	return mod.Unreachable()
}

// checkCallSignature checks the number of arguments against a call signature.
// Returns true if the signature is compatible, otherwise reports a diagnostic and returns false.
// Ported from: assemblyscript/src/compiler.ts checkCallSignature (lines 6274-6316).
func (c *Compiler) checkCallSignature(
	signature *types.Signature,
	numArguments int32,
	hasThis bool,
	reportNode ast.Node,
) bool {
	// cannot call an instance method without a `this` argument (TODO: .call?)
	thisType := signature.ThisType
	if hasThis != (thisType != nil) {
		c.Error(
			diagnostics.DiagnosticCodeTheThisTypesOfEachSignatureAreIncompatible,
			reportNode.GetRange(),
			"", "", "",
		)
		return false
	}

	hasRest := signature.HasRest
	minimum := signature.RequiredParameters
	maximum := int32(len(signature.ParameterTypes))

	// must at least be called with required arguments
	if numArguments < minimum {
		code := diagnostics.DiagnosticCodeExpected0ArgumentsButGot1
		if minimum < maximum {
			code = diagnostics.DiagnosticCodeExpectedAtLeast0ArgumentsButGot1
		}
		c.Error(
			code,
			reportNode.GetRange(),
			strconv.Itoa(int(minimum)), strconv.Itoa(int(numArguments)), "",
		)
		return false
	}

	// must not be called with more than the maximum arguments
	if numArguments > maximum && !hasRest {
		c.Error(
			diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
			reportNode.GetRange(),
			strconv.Itoa(int(maximum)), strconv.Itoa(int(numArguments)), "",
		)
		return false
	}

	return true
}

// compileCallDirect compiles a direct function call.
// Ported from: assemblyscript/src/compiler.ts compileCallDirect (lines 6368-6442).
func (c *Compiler) compileCallDirect(
	instance *program.Function,
	argumentExpressions []ast.Node,
	reportNode ast.Node,
	thisArg module.ExpressionRef,
	constraints Constraints,
) module.ExpressionRef {
	numArguments := len(argumentExpressions)
	signature := instance.Signature

	if !c.checkCallSignature(
		signature,
		int32(numArguments),
		thisArg != 0,
		reportNode,
	) {
		c.CurrentType = signature.ReturnType
		return c.Module().Unreachable()
	}
	if instance.HasDecorator(program.DecoratorFlagsUnsafe) {
		c.checkUnsafe(reportNode, nil)
	}

	argumentExpressions = c.adjustArgumentsForRestParams(argumentExpressions, signature, reportNode)
	numArguments = len(argumentExpressions)

	// handle call on `this` in constructors
	sourceFunction := c.CurrentFlow.SourceFunction()
	if sourceFunction.Is(common.CommonFlagsConstructor) && ast.IsAccessOnThis(reportNode) {
		parentRef := sourceFunction.FlowParent()
		if parentRef != nil && parentRef.GetElementKind() == program.ElementKindClass {
			c.checkFieldInitialization(parentRef.(*program.Class), reportNode)
		}
	}

	// Inline if explicitly requested
	inlineRequested := instance.HasDecorator(program.DecoratorFlagsInline) || c.CurrentFlow.Is(flow.FlowFlagInlineContext)
	if inlineRequested && (!instance.Is(common.CommonFlagsOverridden) || ast.IsAccessOnSuper(reportNode)) {
		if instance.Is(common.CommonFlagsStub) {
			panic("@inline on stub doesn't make sense")
		}
		inlineStack := c.InlineStack
		if inlineStackContains(inlineStack, instance) {
			c.Warning(
				diagnostics.DiagnosticCodeFunction0CannotBeInlinedIntoItself,
				reportNode.GetRange(),
				instance.GetInternalName(), "", "",
			)
		} else {
			parameterTypes := signature.ParameterTypes
			if numArguments > len(parameterTypes) {
				panic("numArguments > parameterTypes.length for inline")
			}
			// compile argument expressions *before* pushing to the inline stack
			// otherwise, the arguments may not be inlined, e.g. `abc(abc(123))`
			args := make([]module.ExpressionRef, numArguments)
			for i := 0; i < numArguments; i++ {
				args[i] = c.CompileExpression(argumentExpressions[i], parameterTypes[i], ConstraintsConvImplicit)
			}
			// make the inlined call
			c.InlineStack = append(inlineStack, instance)
			expr := c.makeCallInline(instance, args, thisArg, (constraints&ConstraintsWillDrop) != 0)
			c.InlineStack = c.InlineStack[:len(c.InlineStack)-1]
			return expr
		}
	}

	// Otherwise compile to just a call
	numArgumentsInclThis := numArguments
	if thisArg != 0 {
		numArgumentsInclThis++
	}
	operands := make([]module.ExpressionRef, numArgumentsInclThis)
	index := 0
	if thisArg != 0 {
		operands[0] = thisArg
		index = 1
	}
	parameterTypes := signature.ParameterTypes
	for i := 0; i < numArguments; i++ {
		paramType := parameterTypes[i]
		paramExpr := c.CompileExpression(argumentExpressions[i], paramType, ConstraintsConvImplicit)
		operands[index] = paramExpr
		index++
	}
	if index != numArgumentsInclThis {
		panic("index != numArgumentsInclThis")
	}
	return c.makeCallDirect(instance, operands, reportNode, (constraints&ConstraintsWillDrop) != 0)
}

// compileCallIndirect compiles an indirect call to a first-class function.
// Checks the call signature, adjusts for rest params, compiles argument
// expressions to operands, then delegates to makeCallIndirect.
// Ported from: assemblyscript/src/compiler.ts compileCallIndirect (lines 7007-7044).
func (c *Compiler) compileCallIndirect(
	signature *types.Signature,
	functionArg module.ExpressionRef,
	argumentExpressions []ast.Node,
	reportNode ast.Node,
	thisArg module.ExpressionRef,
	immediatelyDropped bool,
) module.ExpressionRef {
	numArguments := len(argumentExpressions)

	if !c.checkCallSignature( // reports
		signature,
		int32(numArguments),
		thisArg != 0,
		reportNode,
	) {
		return c.Module().Unreachable()
	}

	argumentExpressions = c.adjustArgumentsForRestParams(argumentExpressions, signature, reportNode)
	numArguments = len(argumentExpressions)

	numArgumentsInclThis := numArguments
	if thisArg != 0 {
		numArgumentsInclThis++
	}
	operands := make([]module.ExpressionRef, numArgumentsInclThis)
	index := 0
	if thisArg != 0 {
		operands[0] = thisArg
		index = 1
	}
	parameterTypes := signature.ParameterTypes
	for i := 0; i < numArguments; i++ {
		operands[index] = c.CompileExpression(argumentExpressions[i], parameterTypes[i], ConstraintsConvImplicit)
		index++
	}
	if index != numArgumentsInclThis {
		panic("compileCallIndirect: index != numArgumentsInclThis")
	}
	return c.makeCallIndirect(signature, functionArg, reportNode, operands, immediatelyDropped)
}

// compileCommaExpression compiles a comma expression.
// Ported from: assemblyscript/src/compiler.ts compileCommaExpression (lines 7114-7129).
func (c *Compiler) compileCommaExpression(expression *ast.CommaExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	exprs := expression.Expressions
	numExpressions := len(exprs)
	compiledExprs := make([]module.ExpressionRef, numExpressions)
	numExpressions--
	for i := 0; i < numExpressions; i++ {
		compiledExprs[i] = c.CompileExpression(exprs[i], types.TypeVoid, // drop all except last
			ConstraintsConvImplicit|ConstraintsWillDrop,
		)
	}
	compiledExprs[numExpressions] = c.CompileExpression(exprs[numExpressions], contextualType, constraints)
	return c.Module().Flatten(compiledExprs, c.CurrentType.ToRef())
}

// compileElementAccessExpression compiles an element access expression (array indexing).
// Ported from: assemblyscript/src/compiler.ts compileElementAccessExpression (lines 7131-7176).
func (c *Compiler) compileElementAccessExpression(expression *ast.ElementAccessExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	targetExpression := expression.Expression
	resolver := c.Resolver()
	fl := c.CurrentFlow

	// Check if the target is an enum (enum-to-string conversion path)
	targetElement := resolver.LookupExpression(targetExpression, fl, types.TypeAuto, program.ReportModeSwallow)
	if targetElement != nil && targetElement.GetElementKind() == program.ElementKindEnum {
		elementExpr := c.CompileExpression(expression.ElementExpression, types.TypeI32, ConstraintsConvImplicit)
		toStringFunctionName := c.ensureEnumToString(targetElement.(*program.Enum), expression)
		c.CurrentType = c.Program.StringInstance().GetResolvedType()
		if toStringFunctionName == "" {
			return mod.Unreachable()
		}
		return mod.Call(toStringFunctionName, []module.ExpressionRef{elementExpr}, module.TypeRefI32)
	}

	// Resolve the target type
	targetType := resolver.ResolveExpression(targetExpression, fl, types.TypeAuto, program.ReportModeReport)
	if targetType != nil {
		classReference := targetType.GetClassOrWrapper(c.Program)
		if classReference != nil {
			if classInstance, ok := classReference.(*program.Class); ok {
				isUnchecked := fl.Is(flow.FlowFlagUncheckedContext)
				indexedGet := classInstance.FindOverload(program.OperatorKindIndexedGet, isUnchecked)
				if indexedGet != nil {
					thisType := indexedGet.Signature.ThisType
					thisArg := c.CompileExpression(targetExpression, thisType, ConstraintsConvImplicit)
					if !isUnchecked && c.Options().Pedantic {
						c.Pedantic(
							diagnostics.DiagnosticCodeIndexedAccessMayInvolveBoundsChecking,
							expression.GetRange(),
							"", "", "",
						)
					}
					return c.compileCallDirect(indexedGet, []ast.Node{
						expression.ElementExpression,
					}, expression, thisArg, constraints)
				}
			}
		}
		c.Error(
			diagnostics.DiagnosticCodeIndexSignatureIsMissingInType0,
			targetExpression.GetRange(),
			targetType.String(), "", "",
		)
	}
	return mod.Unreachable()
}

// compileFunctionExpression compiles a function expression (arrow function or anonymous function).
// Ported from: assemblyscript/src/compiler.ts compileFunctionExpression (lines 7178-7344).
func (c *Compiler) compileFunctionExpression(expression *ast.FunctionExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	resolver := c.Resolver()
	sourceFunction := fl.SourceFunction()

	declaration := expression.Declaration
	if declaration == nil {
		c.CurrentType = contextualType
		return mod.Unreachable()
	}
	declaration = declaration.Clone() // generic contexts can have multiple

	isNamed := len(declaration.Name.Text) > 0
	isSemanticallyAnonymous := !isNamed || contextualType != types.TypeVoid

	// Build prototype name
	var protoName string
	if isSemanticallyAnonymous {
		nameBase := "anonymous"
		if isNamed {
			nameBase = declaration.Name.Text
		}
		protoName = fmt.Sprintf("%s|%d", nameBase, sourceFunction.(*program.Function).NextAnonymousId)
		sourceFunction.(*program.Function).NextAnonymousId++
	} else {
		protoName = declaration.Name.Text
	}

	prototype := program.NewFunctionPrototype(
		protoName,
		sourceFunction.(program.Element),
		declaration,
		program.DecoratorFlagsNone,
	)
	contextualTypeArguments := cloneTypeArgMap(fl.ContextualTypeArguments())

	var instance *program.Function

	// Compile according to context: omitted parameter/return types can be inferred
	contextualSignature := contextualType.SignatureReference
	if contextualSignature != nil {
		signatureNode := prototype.FunctionTypeNode()
		parameterNodes := signatureNode.Parameters
		numPresentParameters := int32(len(parameterNodes))

		// must not require more than the maximum number of parameters
		parameterTypes := contextualSignature.ParameterTypes
		numParameters := int32(len(parameterTypes))
		if numPresentParameters > numParameters {
			c.Error(
				diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
				expression.GetRange(),
				fmt.Sprintf("%d", numParameters), fmt.Sprintf("%d", numPresentParameters), "",
			)
			return mod.Unreachable()
		}

		// check non-omitted parameter types
		for i := int32(0); i < numPresentParameters; i++ {
			parameterNode := parameterNodes[i]
			if !ast.IsTypeOmitted(parameterNode.Type) {
				resolvedType := resolver.ResolveType(
					parameterNode.Type, fl,
					sourceFunction.(program.Element).GetParent(),
					contextualTypeArguments,
					program.ReportModeReport,
				)
				if resolvedType == nil {
					return mod.Unreachable()
				}
				if !parameterTypes[i].IsStrictlyAssignableTo(resolvedType, false) {
					c.Error(
						diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
						parameterNode.GetRange(),
						parameterTypes[i].String(), resolvedType.String(), "",
					)
					return mod.Unreachable()
				}
			}
		}

		// check non-omitted return type
		returnType := contextualSignature.ReturnType
		if !ast.IsTypeOmitted(signatureNode.ReturnType) {
			resolvedType := resolver.ResolveType(
				signatureNode.ReturnType, fl,
				sourceFunction.(program.Element).GetParent(),
				contextualTypeArguments,
				program.ReportModeReport,
			)
			if resolvedType == nil {
				return mod.Unreachable()
			}
			if returnType == types.TypeVoid {
				if resolvedType != types.TypeVoid {
					c.Error(
						diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
						signatureNode.ReturnType.GetRange(),
						resolvedType.String(), returnType.String(), "",
					)
					return mod.Unreachable()
				}
			} else if !resolvedType.IsStrictlyAssignableTo(returnType, false) {
				c.Error(
					diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
					signatureNode.ReturnType.GetRange(),
					resolvedType.String(), returnType.String(), "",
				)
				return mod.Unreachable()
			}
		}

		// check explicit this type
		thisType := contextualSignature.ThisType
		thisTypeNode := signatureNode.ExplicitThisType
		if thisTypeNode != nil {
			if thisType == nil {
				c.Error(
					diagnostics.DiagnosticCodeThisCannotBeReferencedInCurrentLocation,
					thisTypeNode.GetRange(),
					"", "", "",
				)
				return mod.Unreachable()
			}
			resolvedType := resolver.ResolveType(
				thisTypeNode, fl,
				sourceFunction.(program.Element).GetParent(),
				contextualTypeArguments,
				program.ReportModeReport,
			)
			if resolvedType == nil {
				return mod.Unreachable()
			}
			if !thisType.IsStrictlyAssignableTo(resolvedType, false) {
				c.Error(
					diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
					thisTypeNode.GetRange(),
					thisType.String(), resolvedType.String(), "",
				)
				return mod.Unreachable()
			}
		}

		signature := types.CreateSignature(c.Program, parameterTypes, returnType, thisType, numParameters, false)
		instance = program.NewFunction(
			prototype.GetName(),
			prototype,
			nil,
			signature,
			contextualTypeArguments,
		)
		instance.Flow.Outer = fl
		worked := c.CompileFunction(instance)
		c.CurrentType = contextualSignature.Type
		if !worked {
			return mod.Unreachable()
		}
	} else {
		// otherwise compile like a normal function
		instance = resolver.ResolveFunction(prototype, nil, contextualTypeArguments, program.ReportModeReport)
		if instance == nil {
			return mod.Unreachable()
		}
		instance.Flow.Outer = fl
		worked := c.CompileFunction(instance)
		c.CurrentType = instance.Signature.Type
		if !worked {
			return mod.Unreachable()
		}
	}

	offset := c.ensureRuntimeFunction(instance)
	var expr module.ExpressionRef
	if c.Options().IsWasm64() {
		expr = mod.I64(int64(offset))
	} else {
		expr = mod.I32(int32(offset))
	}

	// add a constant local referring to the function if applicable
	if !isSemanticallyAnonymous {
		fname := instance.GetName()
		existingLocal := fl.GetScopedLocal(fname)
		if existingLocal != nil {
			if existingLocal.DeclarationIsNative() {
				// scoped locals are shared temps that don't track declarations
				c.Error(
					diagnostics.DiagnosticCodeDuplicateIdentifier0,
					declaration.Name.GetRange(),
					fname, "", "",
				)
			} else {
				c.ErrorRelated(
					diagnostics.DiagnosticCodeDuplicateIdentifier0,
					declaration.Name.GetRange(),
					existingLocal.DeclarationNameRange().(*diagnostics.Range),
					fname, "", "",
				)
			}
		} else {
			ftype := instance.GetResolvedType()
			local := fl.AddScopedLocal(instance.GetName(), ftype)
			fl.SetLocalFlag(local.FlowIndex(), flow.LocalFlagConstant|flow.LocalFlagInitialized)
			expr = mod.LocalTee(local.FlowIndex(), expr, ftype.IsManaged(), ftype.ToRef())
		}
	}

	return expr
}

// compileIdentifierExpression compiles an identifier expression (variable, constant, true, false, null, etc.).
// Ported from: assemblyscript/src/compiler.ts compileIdentifierExpression (lines 4547-4792).
func (c *Compiler) compileIdentifierExpression(expression *ast.IdentifierExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	switch expression.GetKind() {
	case ast.NodeKindTrue:
		c.CurrentType = types.TypeBool
		return mod.I32(1)
	case ast.NodeKindFalse:
		c.CurrentType = types.TypeBool
		return mod.I32(0)
	case ast.NodeKindNull:
		options := c.Options()
		if contextualType.IsReference() {
			classRef := contextualType.GetClass()
			if classRef != nil {
				c.CurrentType = classRef.GetType().AsNullable()
				if options.IsWasm64() {
					return mod.I64(0)
				}
				return mod.I32(0)
			}
			sigRef := contextualType.GetSignature()
			if sigRef != nil {
				c.CurrentType = sigRef.Type.AsNullable()
				if options.IsWasm64() {
					return mod.I64(0)
				}
				return mod.I32(0)
			}
			return c.makeZeroOfType(contextualType)
		}
		c.CurrentType = options.UsizeType()
		c.Warning(
			diagnostics.DiagnosticCodeExpressionResolvesToUnusualType0,
			expression.GetRange(),
			c.CurrentType.String(), "", "",
		)
		if options.IsWasm64() {
			return mod.I64(0)
		}
		return mod.I32(0)
	case ast.NodeKindThis:
		sourceFunction := fl.SourceFunction()
		thisType := sourceFunction.FlowSignature().ThisType
		if thisType == nil {
			c.Error(
				diagnostics.DiagnosticCodeThisCannotBeReferencedInCurrentLocation,
				expression.GetRange(),
				"", "", "",
			)
			c.CurrentType = c.Options().UsizeType()
			return mod.Unreachable()
		}
		if sourceFunction.Is(uint32(common.CommonFlagsConstructor)) {
			if fl.Is(flow.FlowFlagCtorParamContext) {
				c.Error(
					diagnostics.DiagnosticCodeThisCannotBeReferencedInConstructorArguments,
					expression.GetRange(),
					"", "", "",
				)
			}
			if constraints&ConstraintsIsThis == 0 {
				parent := sourceFunction.(program.Element).GetParent()
				if parent != nil && parent.GetElementKind() == program.ElementKindClass {
					c.checkFieldInitialization(parent.(*program.Class), expression)
				}
			}
		}
		thisLocal := fl.LookupLocal("this")
		fl.SetFlag(flow.FlowFlagAccessesThis)
		c.CurrentType = thisType
		if thisLocal != nil {
			return mod.LocalGet(thisLocal.FlowIndex(), thisType.ToRef())
		}
		return mod.LocalGet(0, thisType.ToRef())
	case ast.NodeKindSuper:
		sourceFunction := fl.SourceFunction()
		if sourceFunction.Is(uint32(common.CommonFlagsConstructor)) {
			if fl.Is(flow.FlowFlagCtorParamContext) {
				c.Error(
					diagnostics.DiagnosticCodeSuperCannotBeReferencedInConstructorArguments,
					expression.GetRange(),
					"", "", "",
				)
			} else if !fl.Is(flow.FlowFlagCallsSuper) {
				c.Error(
					diagnostics.DiagnosticCodeSuperMustBeCalledBeforeAccessingAPropertyOfSuperInTheConstructorOfADerivedClass,
					expression.GetRange(),
					"", "", "",
				)
			}
		}
		if fl.IsInline() {
			scopedThis := fl.LookupLocal("this")
			if scopedThis != nil {
				scopedThisClassRef := scopedThis.GetType().GetClass()
				if scopedThisClassRef != nil {
					if scopedThisClass, ok := scopedThisClassRef.(*program.Class); ok {
						base := scopedThisClass.Base
						if base != nil {
							superType := base.GetResolvedType()
							c.CurrentType = superType
							return mod.LocalGet(scopedThis.FlowIndex(), superType.ToRef())
						}
					}
				}
			}
		}
		if sourceFunction.Is(uint32(common.CommonFlagsInstance)) {
			parent := sourceFunction.(program.Element).GetParent()
			if parent != nil && parent.GetElementKind() == program.ElementKindClass {
				classInstance := parent.(*program.Class)
				baseClassInstance := classInstance.Base
				if baseClassInstance != nil {
					superType := baseClassInstance.GetResolvedType()
					c.CurrentType = superType
					return mod.LocalGet(0, superType.ToRef())
				}
			}
		}
		c.Error(
			diagnostics.DiagnosticCodeSuperCanOnlyBeReferencedInADerivedClass,
			expression.GetRange(),
			"", "", "",
		)
		c.CurrentType = c.Options().UsizeType()
		return mod.Unreachable()
	}

	// Maybe compile the enclosing source file
	c.maybeCompileEnclosingSource(expression)

	// Resolve identifier through the resolver
	resolver := c.Resolver()
	target := resolver.LookupExpression(expression, fl, contextualType, program.ReportModeReport)
	if target == nil {
		// make a guess to avoid assertions in calling code
		if c.CurrentType == types.TypeVoid {
			c.CurrentType = types.TypeI32
		}
		return mod.Unreachable()
	}

	switch target.GetElementKind() {
	case program.ElementKindLocal:
		local := target.(*program.Local)
		localType := local.GetResolvedType()
		if localType == types.TypeVoid {
			panic("assertion failed: local type != void")
		}
		if _, pending := c.PendingElements[local]; pending {
			c.Error(
				diagnostics.DiagnosticCodeVariable0UsedBeforeItsDeclaration,
				expression.GetRange(),
				local.GetInternalName(), "", "",
			)
			c.CurrentType = localType
			return mod.Unreachable()
		}
		if local.Is(common.CommonFlagsInlined) {
			return c.compileInlineConstant(local, contextualType, constraints)
		}
		localIndex := local.Index
		if !fl.IsLocalFlag(localIndex, flow.LocalFlagInitialized) {
			c.Error(
				diagnostics.DiagnosticCodeVariable0IsUsedBeforeBeingAssigned,
				expression.GetRange(),
				local.GetName(), "", "",
			)
		}
		if localIndex < 0 {
			panic("assertion failed: local index >= 0")
		}
		isNonNull := fl.IsLocalFlagDefault(localIndex, flow.LocalFlagNonNull, false)
		if localType.IsNullableReference() && isNonNull && (!localType.IsExternalReference() || c.Options().HasFeature(common.FeatureGC)) {
			c.CurrentType = localType.NonNullableType()
		} else {
			c.CurrentType = localType
		}

		if !local.DeclaredByFlow(fl) {
			// TODO: closures
			c.Error(
				diagnostics.DiagnosticCodeNotImplemented0,
				expression.GetRange(),
				"Closures", "", "",
			)
			return mod.Unreachable()
		}
		expr := mod.LocalGet(localIndex, localType.ToRef())
		// TODO: ref_as_nonnull for GC-enabled nullable external references
		return expr

	case program.ElementKindGlobal:
		global := target.(*program.Global)
		if !c.CompileGlobalLazy(global, expression) {
			return mod.Unreachable()
		}
		globalType := global.GetResolvedType()
		if _, pending := c.PendingElements[global]; pending {
			c.Error(
				diagnostics.DiagnosticCodeVariable0UsedBeforeItsDeclaration,
				expression.GetRange(),
				global.GetInternalName(), "", "",
			)
			c.CurrentType = globalType
			return mod.Unreachable()
		}
		if globalType == types.TypeVoid {
			panic("assertion failed: global type != void")
		}
		if global.HasDecorator(program.DecoratorFlagsBuiltin) {
			return c.compileIdentifierExpressionBuiltin(global, expression, contextualType)
		}
		if global.Is(common.CommonFlagsInlined) {
			return c.compileInlineConstant(global, contextualType, constraints)
		}
		expr := mod.GlobalGet(global.GetInternalName(), globalType.ToRef())
		if global.Is(common.CommonFlagsDefinitelyAssigned) && globalType.IsReference() && !globalType.IsNullableReference() {
			expr = c.makeRuntimeNonNullCheck(expr, globalType, expression)
		}
		c.CurrentType = globalType
		return expr

	case program.ElementKindEnumValue:
		// here: if referenced from within the same enum
		enumValue := target.(*program.EnumValue)
		if !target.Is(common.CommonFlagsCompiled) {
			c.Error(
				diagnostics.DiagnosticCodeAMemberInitializerInAEnumDeclarationCannotReferenceMembersDeclaredAfterItIncludingMembersDefinedInOtherEnums,
				expression.GetRange(),
				"", "", "",
			)
			c.CurrentType = types.TypeI32
			return mod.Unreachable()
		}
		c.CurrentType = types.TypeI32
		if enumValue.Is(common.CommonFlagsInlined) {
			return mod.I32(int32(enumValue.GetConstantIntegerValue()))
		}
		return mod.GlobalGet(enumValue.GetInternalName(), module.TypeRefI32)

	case program.ElementKindFunctionPrototype:
		functionPrototype := target.(*program.FunctionPrototype)
		typeParameterNodes := functionPrototype.TypeParameterNodes()

		if typeParameterNodes != nil && len(typeParameterNodes) != 0 {
			c.Error(
				diagnostics.DiagnosticCodeTypeArgumentExpected,
				expression.GetRange(),
				"", "", "",
			)
			break // also diagnose 'not a value at runtime'
		}

		functionInstance := resolver.ResolveFunction(
			functionPrototype,
			nil,
			cloneTypeArgMap(fl.ContextualTypeArguments()),
			program.ReportModeReport,
		)
		if functionInstance == nil || !c.CompileFunction(functionInstance) {
			return mod.Unreachable()
		}
		if functionInstance.HasDecorator(program.DecoratorFlagsBuiltin) {
			c.Error(
				diagnostics.DiagnosticCodeNotImplemented0,
				expression.GetRange(),
				"First-class built-ins", "", "",
			)
			c.CurrentType = functionInstance.GetResolvedType()
			return mod.Unreachable()
		}
		if contextualType.IsExternalReference() {
			// TODO: Concrete function types currently map to first class functions implemented in
			// linear memory (on top of `usize`), leaving only generic `funcref` for use here.
			c.CurrentType = types.TypeFunc
			return mod.RefFunc(functionInstance.GetInternalName(), types.TypeFunc.ToRef())
		}
		offset := c.ensureRuntimeFunction(functionInstance)
		c.CurrentType = functionInstance.Signature.Type
		if c.Options().IsWasm64() {
			return mod.I64(int64(offset))
		}
		return mod.I32(int32(offset))
	}

	c.Error(
		diagnostics.DiagnosticCodeExpressionDoesNotCompileToAValueAtRuntime,
		expression.GetRange(),
		"", "", "",
	)
	return mod.Unreachable()
}

// maybeCompileEnclosingSource makes sure the enclosing source file of the specified expression
// has been compiled. Ported from: assemblyscript/src/compiler.ts maybeCompileEnclosingSource (lines 7346-7355).
func (c *Compiler) maybeCompileEnclosingSource(expression ast.Node) {
	rng := expression.GetRange()
	if rng == nil || rng.Source == nil {
		return
	}
	src, ok := rng.Source.(*ast.Source)
	if !ok {
		return
	}
	internalPath := src.InternalPath
	filesByName := c.Program.FilesByName
	if enclosingFile, exists := filesByName[internalPath]; exists {
		if !enclosingFile.Is(common.CommonFlagsCompiled) {
			c.CompileFileByPath(internalPath, expression)
		}
	}
}

// compileIdentifierExpressionBuiltin compiles a builtin identifier expression.
// Ported from: assemblyscript/src/compiler.ts compileIdentifierExpressionBuiltin (lines 7633-7648).
func (c *Compiler) compileIdentifierExpressionBuiltin(global *program.Global, expression *ast.IdentifierExpression, contextualType *types.Type) module.ExpressionRef {
	if global.HasDecorator(program.DecoratorFlagsUnsafe) {
		c.checkUnsafe(expression, global.IdentifierNode())
	}
	// TODO: implement builtinVariables_onAccess dispatch
	c.Error(
		diagnostics.DiagnosticCodeNotImplemented0,
		expression.GetRange(),
		"Built-in variable access", "", "",
	)
	c.CurrentType = contextualType
	return c.Module().Unreachable()
}

// compileInstanceOfExpression compiles an instanceof expression.
// Ported from: assemblyscript/src/compiler.ts compileInstanceOfExpression (lines 7650-7683).
func (c *Compiler) compileInstanceOfExpression(expression *ast.InstanceOfExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	fl := c.CurrentFlow
	isTypeNode := expression.IsType

	// Mimic `instanceof CLASS` (generic prototype)
	if namedType, ok := isTypeNode.(*ast.NamedTypeNode); ok {
		if !namedType.IsNullable && !namedType.HasTypeArguments() {
			element := c.Resolver().ResolveTypeName(
				namedType.Name,
				fl,
				fl.SourceFunction().(program.Element),
				program.ReportModeSwallow,
			)
			if element != nil && element.GetElementKind() == program.ElementKindClassPrototype {
				prototype := element.(*program.ClassPrototype)
				if prototype.Is(common.CommonFlagsGeneric) {
					return c.makeInstanceofClass(expression, prototype)
				}
			}
		}
	}

	// Fall back to `instanceof TYPE`
	ctxTypes := cloneTypeArgMap(fl.ContextualTypeArguments())
	expectedType := c.Resolver().ResolveType(
		expression.IsType,
		fl,
		fl.SourceFunction().(program.Element),
		ctxTypes,
		program.ReportModeReport,
	)
	if expectedType == nil {
		c.CurrentType = types.TypeBool
		return c.Module().Unreachable()
	}
	return c.makeInstanceofType(expression, expectedType)
}

// makeInstanceofType compiles an instanceof check against a resolved type.
// Ported from: assemblyscript/src/compiler.ts makeInstanceofType (lines 7685-7786).
func (c *Compiler) makeInstanceofType(expression *ast.InstanceOfExpression, expectedType *types.Type) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	expr := c.CompileExpression(expression.Expression, expectedType, ConstraintsNone)
	actualType := c.CurrentType
	c.CurrentType = types.TypeBool

	// instanceof <value> - must be exact
	if expectedType.IsValue() {
		if actualType == expectedType {
			return mod.MaybeDropCondition(expr, mod.I32(1))
		}
		return mod.MaybeDropCondition(expr, mod.I32(0))
	}

	// <value> instanceof <nonValue> - always false
	if actualType.IsValue() {
		return mod.MaybeDropCondition(expr, mod.I32(0))
	}

	// both LHS and RHS are references now
	sizeTypeRef := actualType.ToRef()

	// <nullable> instanceof <nonNullable> - LHS must be != 0
	if actualType.IsNullableReference() && !expectedType.IsNullableReference() {

		// same or upcast - check statically
		if actualType.NonNullableType().IsAssignableTo(expectedType, false) {
			var neOp module.Op
			if sizeTypeRef == module.TypeRefI64 {
				neOp = module.BinaryOpNeI64
			} else {
				neOp = module.BinaryOpNeI32
			}
			return mod.Binary(neOp, expr, c.makeZeroOfType(actualType))
		}

		// potential downcast - check dynamically
		if actualType.NonNullableType().HasSubtypeAssignableTo(expectedType) {
			if !actualType.IsUnmanaged() && !expectedType.IsUnmanaged() {
				if c.Options().Pedantic {
					c.Pedantic(
						diagnostics.DiagnosticCodeExpressionCompilesToADynamicCheckAtRuntime,
						expression.GetRange(),
						"", "", "",
					)
				}
				temp := fl.GetTempLocal(actualType)
				tempIndex := temp.FlowIndex()
				var eqzOp module.Op
				if sizeTypeRef == module.TypeRefI64 {
					eqzOp = module.UnaryOpEqzI64
				} else {
					eqzOp = module.UnaryOpEqzI32
				}
				classRef := expectedType.ClassRef.(*program.Class)
				return mod.If(
					mod.Unary(eqzOp,
						mod.LocalTee(tempIndex, expr, actualType.IsManaged(), sizeTypeRef),
					),
					mod.I32(0),
					mod.Call(c.prepareInstanceOf(classRef), []module.ExpressionRef{
						mod.LocalGet(tempIndex, sizeTypeRef),
					}, module.TypeRefI32),
				)
			} else {
				c.Error(
					diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(),
					"instanceof", actualType.String(), expectedType.String(),
				)
			}
		}

	// either none or both nullable
	} else {

		// same or upcast - check statically
		if actualType.IsAssignableTo(expectedType, false) {
			return mod.MaybeDropCondition(expr, mod.I32(1))
		}

		// potential downcast - check dynamically
		if actualType.HasSubtypeAssignableTo(expectedType) {
			if !actualType.IsUnmanaged() && !expectedType.IsUnmanaged() {
				temp := fl.GetTempLocal(actualType)
				tempIndex := temp.FlowIndex()
				var eqzOp module.Op
				if sizeTypeRef == module.TypeRefI64 {
					eqzOp = module.UnaryOpEqzI64
				} else {
					eqzOp = module.UnaryOpEqzI32
				}
				classRef := expectedType.ClassRef.(*program.Class)
				return mod.If(
					mod.Unary(eqzOp,
						mod.LocalTee(tempIndex, expr, actualType.IsManaged(), sizeTypeRef),
					),
					mod.I32(0),
					mod.Call(c.prepareInstanceOf(classRef), []module.ExpressionRef{
						mod.LocalGet(tempIndex, sizeTypeRef),
					}, module.TypeRefI32),
				)
			} else {
				c.Error(
					diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
					expression.GetRange(),
					"instanceof", actualType.String(), expectedType.String(),
				)
			}
		}
	}

	// false
	return mod.MaybeDropCondition(expr, mod.I32(0))
}

// makeInstanceofClass compiles an instanceof check against a generic class prototype.
// Ported from: assemblyscript/src/compiler.ts makeInstanceofClass (lines 7876-7929).
func (c *Compiler) makeInstanceofClass(expression *ast.InstanceOfExpression, prototype *program.ClassPrototype) module.ExpressionRef {
	mod := c.Module()
	expr := c.CompileExpression(expression.Expression, types.TypeAuto, ConstraintsNone)
	actualType := c.CurrentType
	sizeTypeRef := actualType.ToRef()

	c.CurrentType = types.TypeBool

	// exclusively interested in class references here
	classRef := actualType.GetClass()
	if classRef != nil {
		classInstance, ok := classRef.(*program.Class)
		if ok {
			// static check
			if classInstance.ExtendsPrototype(prototype) {
				// <nullable> instanceof <PROTOTYPE> - LHS must be != 0
				if actualType.IsNullableReference() {
					var neOp module.Op
					if sizeTypeRef == module.TypeRefI64 {
						neOp = module.BinaryOpNeI64
					} else {
						neOp = module.BinaryOpNeI32
					}
					return mod.Binary(neOp, expr, c.makeZeroOfType(actualType))
				}
				// <nonNullable> is just `true`
				return mod.MaybeDropCondition(expr, mod.I32(1))

			// dynamic check against all possible concrete ids
			} else if prototype.Extends(classInstance.Prototype) {
				fl := c.CurrentFlow
				temp := fl.GetTempLocal(actualType)
				tempIndex := temp.FlowIndex()
				var eqzOp module.Op
				if sizeTypeRef == module.TypeRefI64 {
					eqzOp = module.UnaryOpEqzI64
				} else {
					eqzOp = module.UnaryOpEqzI32
				}
				// !(t = expr) ? 0 : anyinstanceof(t)
				return mod.If(
					mod.Unary(eqzOp,
						mod.LocalTee(tempIndex, expr, actualType.IsManaged(), sizeTypeRef),
					),
					mod.I32(0),
					mod.Call(c.prepareAnyInstanceOf(prototype), []module.ExpressionRef{
						mod.LocalGet(tempIndex, sizeTypeRef),
					}, module.TypeRefI32),
				)
			}
		}
	}

	// false
	return mod.MaybeDropCondition(expr, mod.I32(0))
}

// compileLiteralExpression compiles a literal expression (number, string, array, etc.).
// Ported from: assemblyscript/src/compiler.ts compileLiteralExpression (lines 4795-4900).
func (c *Compiler) compileLiteralExpression(expression ast.Node, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()

	switch lit := expression.(type) {
	case *ast.IntegerLiteralExpression:
		return c.compileIntegerLiteral(lit, contextualType)
	case *ast.FloatLiteralExpression:
		return c.compileFloatLiteral(lit, contextualType)
	case *ast.StringLiteralExpression:
		return c.compileStringLiteral(lit, contextualType, constraints)
	case *ast.ArrayLiteralExpression:
		return c.compileArrayLiteral(lit, contextualType, constraints)
	case *ast.ObjectLiteralExpression:
		return c.compileObjectLiteral(lit, contextualType, constraints)
	case *ast.RegexpLiteralExpression:
		c.Error(diagnostics.DiagnosticCodeNotImplemented0, expression.GetRange(), "RegExp literals", "", "")
		c.CurrentType = contextualType
		return mod.Unreachable()
	case *ast.TemplateLiteralExpression:
		c.Error(diagnostics.DiagnosticCodeNotImplemented0, expression.GetRange(), "Template literals", "", "")
		c.CurrentType = contextualType
		return mod.Unreachable()
	default:
		c.CurrentType = contextualType
		return mod.Unreachable()
	}
}

// compileIntegerLiteral compiles an integer literal.
// Ported from: assemblyscript/src/compiler.ts compileIntegerLiteralExpression (lines 4902-4970).
func (c *Compiler) compileIntegerLiteral(expression *ast.IntegerLiteralExpression, contextualType *types.Type) module.ExpressionRef {
	mod := c.Module()
	value := expression.Value

	// If the contextual type is f64 or f32, produce a float
	if contextualType == types.TypeF64 {
		c.CurrentType = types.TypeF64
		return mod.F64(float64(value))
	}
	if contextualType == types.TypeF32 {
		c.CurrentType = types.TypeF32
		return mod.F32(float32(value))
	}

	// If the contextual type is i64 or u64, produce i64
	if contextualType == types.TypeI64 || contextualType == types.TypeU64 {
		c.CurrentType = contextualType
		return mod.I64(value)
	}

	// isize/usize: depends on wasm32 vs wasm64
	if contextualType.Kind == types.TypeKindIsize || contextualType.Kind == types.TypeKindUsize {
		c.CurrentType = contextualType
		if c.Options().IsWasm64() {
			return mod.I64(value)
		}
		return mod.I32(int32(value))
	}

	// Otherwise, produce i32
	if contextualType == types.TypeI32 || contextualType == types.TypeU32 ||
		contextualType == types.TypeI16 || contextualType == types.TypeU16 ||
		contextualType == types.TypeI8 || contextualType == types.TypeU8 ||
		contextualType == types.TypeBool {
		c.CurrentType = contextualType
		return mod.I32(int32(value))
	}

	// Default: i32
	c.CurrentType = types.TypeI32
	return mod.I32(int32(value))
}

// compileFloatLiteral compiles a float literal.
// Ported from: assemblyscript/src/compiler.ts compileFloatLiteralExpression (lines 4972-5010).
func (c *Compiler) compileFloatLiteral(expression *ast.FloatLiteralExpression, contextualType *types.Type) module.ExpressionRef {
	mod := c.Module()
	value := expression.Value

	if contextualType == types.TypeF32 {
		c.CurrentType = types.TypeF32
		return mod.F32(float32(value))
	}

	// Default: f64
	c.CurrentType = types.TypeF64
	return mod.F64(value)
}

// compileStringLiteral compiles a string literal.
// Ported from: assemblyscript/src/compiler.ts compileStaticString (lines 9515-9598).
func (c *Compiler) compileStringLiteral(expression *ast.StringLiteralExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	value := expression.Value
	stringInstance := c.Program.StringInstance()
	if stringInstance == nil {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"String type not available", "", "",
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}

	c.CurrentType = stringInstance.GetResolvedType()
	return c.EnsureStaticString(value)
}

// EnsureStaticString ensures a static string is in the data segment and returns a
// pointer expression to it.
// Ported from: assemblyscript/src/compiler.ts ensureStaticString (lines 9515-9598).
func (c *Compiler) EnsureStaticString(value string) module.ExpressionRef {
	mod := c.Module()
	isWasm64 := c.Options().IsWasm64()
	totalOverhead := c.Program.TotalOverhead()

	// Check if already cached
	if _, ok := c.StringSegments[value]; ok {
		// The segment already exists. Recalculate the pointer offset.
		// We store the base offset in StringSegmentOffsets for retrieval.
		ptrOffset := c.StringSegmentOffsets[value]
		if isWasm64 {
			return mod.I64(ptrOffset)
		}
		return mod.I32(int32(ptrOffset))
	}

	// Encode string as UTF-16LE (AssemblyScript string format)
	runes := []rune(value)
	byteLen := len(runes) * 2 // UTF-16 code units
	stringInstance := c.Program.StringInstance()

	// Build the data: [BLOCK header][OBJECT header][string content]
	// BLOCK: mmInfo(4) + gcInfo(4) + rtId(4) + rtSize(4) = 16 bytes
	// OBJECT: gcInfo2(4) = 4 bytes (wasm32)

	headerBytes := int(totalOverhead)
	totalBytes := headerBytes + byteLen
	buf := make([]byte, totalBytes)

	// Write BLOCK header
	blockContentSize := int32(byteLen) + c.Program.ObjectOverhead()
	writeI32(buf, 0, blockContentSize)           // mmInfo
	writeI32(buf, 4, 0)                          // gcInfo
	writeI32(buf, 8, int32(stringInstance.Id())) // rtId
	writeI32(buf, 12, int32(byteLen))            // rtSize

	// Write OBJECT overhead (gcInfo2 = 0)
	writeI32(buf, 16, 0)

	// Write string content as UTF-16LE
	contentOffset := headerBytes
	for i, r := range runes {
		code := uint16(r)
		buf[contentOffset+i*2] = byte(code)
		buf[contentOffset+i*2+1] = byte(code >> 8)
	}

	// Allocate in data segment
	currentOffset := c.MemoryOffset
	alignedOffset := (currentOffset + 15) & ^int64(15) // align to 16

	var offsetExpr module.ExpressionRef
	if isWasm64 {
		offsetExpr = mod.I64(alignedOffset)
	} else {
		offsetExpr = mod.I32(int32(alignedOffset))
	}

	segment := &module.MemorySegment{
		Buffer: buf,
		Offset: offsetExpr,
	}
	c.MemorySegments = append(c.MemorySegments, segment)
	c.StringSegments[value] = segment
	c.MemoryOffset = alignedOffset + int64(totalBytes)

	// Return pointer to the string data (after the header)
	ptrOffset := alignedOffset + int64(totalOverhead)
	c.StringSegmentOffsets[value] = ptrOffset
	if isWasm64 {
		return mod.I64(ptrOffset)
	}
	return mod.I32(int32(ptrOffset))
}

// compileStaticString compiles a static string value (e.g., for typeof results).
func (c *Compiler) compileStaticString(value string) module.ExpressionRef {
	stringInstance := c.Program.StringInstance()
	if stringInstance == nil {
		c.CurrentType = c.Options().UsizeType()
		return c.makeZeroOfType(c.CurrentType)
	}
	c.CurrentType = stringInstance.GetResolvedType()
	return c.EnsureStaticString(value)
}

// compileArrayLiteral compiles an array literal expression.
// Ported from: assemblyscript/src/compiler.ts compileArrayLiteral (lines 7267-7383).
func (c *Compiler) compileArrayLiteral(expression *ast.ArrayLiteralExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()

	// Determine the array class and element type from contextual type
	classRef := contextualType.GetClassOrWrapper(c.Program)
	if classRef == nil {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"Array literal without contextual array type", "", "",
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}
	classInstance, ok := classRef.(*program.Class)
	if !ok {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"Array literal with non-class type", "", "",
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}

	// Get element type from type arguments (Array<T> has T as first type arg)
	typeArgs := classInstance.TypeArguments
	if len(typeArgs) == 0 {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"Array literal without element type", "", "",
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}
	elementType := typeArgs[0]

	elements := expression.ElementExpressions
	numElements := int32(len(elements))
	classId := int32(classInstance.Id())
	sizeTypeRef := c.Options().UsizeType().ToRef()
	alignLog2 := int32(elementType.AlignLog2())

	// Compile element expressions
	elementExprs := make([]module.ExpressionRef, len(elements))
	for i, elem := range elements {
		if elem != nil {
			elementExprs[i] = c.CompileExpression(elem, elementType, ConstraintsConvImplicit)
		} else {
			elementExprs[i] = c.makeZeroOfType(elementType)
		}
	}

	// If all elements are constant and small, create a static buffer
	// Otherwise, call __newArray and store elements dynamically
	// For now, always use the dynamic path:
	// __newArray(length, alignLog2, classId, staticBuffer?)
	// Returns the array reference

	// Allocate: call __newArray(numElements, alignLog2, classId, 0 /* no static buffer */)
	allocExpr := mod.Call(
		common.CommonNameNewArray,
		[]module.ExpressionRef{
			mod.I32(numElements),
			mod.I32(alignLog2),
			mod.I32(classId),
			c.makeZeroOfType(c.Options().UsizeType()), // no static data buffer
		},
		sizeTypeRef,
	)

	if numElements == 0 {
		c.CurrentType = contextualType
		return allocExpr
	}

	// Store elements: get the data pointer from the array, then store each element.
	// Ported from: assemblyscript/src/compiler.ts compileArrayLiteral (lines 7330-7375).
	fl := c.CurrentFlow
	resolver := c.Resolver()
	tempLocal := fl.GetTempLocal(c.Options().UsizeType())
	tempIdx := tempLocal.FlowIndex()

	stmts := make([]module.ExpressionRef, 0, 3+len(elementExprs))
	stmts = append(stmts, mod.LocalSet(tempIdx, allocExpr, false))

	// Look up the dataStart field offset from ArrayBufferView
	abvInstance := c.Program.ArrayBufferViewInstance()
	if abvInstance == nil {
		// Runtime not available — return empty array
		stmts = append(stmts, mod.LocalGet(tempIdx, sizeTypeRef))
		c.CurrentType = contextualType
		return mod.Block("", stmts, sizeTypeRef)
	}

	dataStartMember, hasDataStart := abvInstance.Prototype.InstanceMembers["dataStart"]
	if !hasDataStart {
		stmts = append(stmts, mod.LocalGet(tempIdx, sizeTypeRef))
		c.CurrentType = contextualType
		return mod.Block("", stmts, sizeTypeRef)
	}

	dataStartProp, isProp := dataStartMember.(*program.PropertyPrototype)
	if !isProp {
		stmts = append(stmts, mod.LocalGet(tempIdx, sizeTypeRef))
		c.CurrentType = contextualType
		return mod.Block("", stmts, sizeTypeRef)
	}

	dataStartProperty := resolver.ResolveProperty(dataStartProp, program.ReportModeReport)
	if dataStartProperty == nil || dataStartProperty.MemoryOffset < 0 {
		stmts = append(stmts, mod.LocalGet(tempIdx, sizeTypeRef))
		c.CurrentType = contextualType
		return mod.Block("", stmts, sizeTypeRef)
	}
	dataStartFieldOffset := uint32(dataStartProperty.MemoryOffset)

	// Load the dataStart pointer from the array
	usizeType := c.Options().UsizeType()
	usizeBytes := uint32(usizeType.ByteSize())
	dataStartExpr := mod.Load(
		usizeBytes, false,
		mod.LocalGet(tempIdx, sizeTypeRef),
		sizeTypeRef,
		dataStartFieldOffset, usizeBytes,
		"",
	)

	// Store dataStart in a temp local
	dataTemp := fl.GetTempLocal(usizeType)
	dataTempIdx := dataTemp.FlowIndex()
	stmts = append(stmts, mod.LocalSet(dataTempIdx, dataStartExpr, false))

	// Store each compiled element expression into the buffer
	elementBytes := uint32(elementType.ByteSize())
	elementTypeRef := elementType.ToRef()
	for i, elemExpr := range elementExprs {
		stmts = append(stmts, mod.Store(
			elementBytes,
			mod.LocalGet(dataTempIdx, sizeTypeRef),
			elemExpr,
			elementTypeRef,
			uint32(i)*elementBytes,
			uint32(elementType.AlignLog2()),
			"",
		))
	}

	// Return the array reference
	stmts = append(stmts, mod.LocalGet(tempIdx, sizeTypeRef))
	c.CurrentType = contextualType
	return mod.Block("", stmts, sizeTypeRef)
}

// compileObjectLiteral compiles an object literal expression.
// Ported from: assemblyscript/src/compiler.ts compileObjectLiteral (lines 7384-7487).
func (c *Compiler) compileObjectLiteral(expression *ast.ObjectLiteralExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()

	// Object literals require a contextual type (the class to instantiate)
	classRef := contextualType.GetClassOrWrapper(c.Program)
	if classRef == nil {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"Object literal without contextual class type", "", "",
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}
	classInstance, ok := classRef.(*program.Class)
	if !ok {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			expression.GetRange(),
			"Object literal with non-class type", "", "",
		)
		c.CurrentType = contextualType
		return mod.Unreachable()
	}

	// Allocate the object via __new
	sizeTypeRef := c.Options().UsizeType().ToRef()
	size := int32(classInstance.NextMemoryOffset)
	classId := int32(classInstance.Id())
	allocExpr := mod.Call("~lib/rt/__new",
		[]module.ExpressionRef{mod.I32(size), mod.I32(classId)},
		sizeTypeRef,
	)

	names := expression.Names
	values := expression.Values
	if len(names) == 0 {
		c.CurrentType = contextualType
		return allocExpr
	}

	// Store the allocation in a temp, then set each field
	fl := c.CurrentFlow
	tempLocal := fl.GetTempLocal(c.Options().UsizeType())
	tempIdx := tempLocal.FlowIndex()

	stmts := make([]module.ExpressionRef, 0, 2+len(names))
	stmts = append(stmts, mod.LocalSet(tempIdx, allocExpr, false))

	// For each name/value pair, look up the member on the class and store
	for i := 0; i < len(names) && i < len(values); i++ {
		fieldName := names[i].Text
		member, hasMember := classInstance.Prototype.InstanceMembers[fieldName]
		if !hasMember {
			c.Error(
				diagnostics.DiagnosticCodeProperty0DoesNotExistOnType1,
				names[i].GetRange(),
				fieldName, classInstance.GetInternalName(), "",
			)
			continue
		}

		// Check if it's a property with a setter
		switch member.GetElementKind() {
		case program.ElementKindPropertyPrototype:
			propertyPrototype := member.(*program.PropertyPrototype)
			propertyInstance := c.Resolver().ResolveProperty(propertyPrototype, program.ReportModeReport)
			if propertyInstance == nil {
				continue
			}
			setterInstance := propertyInstance.SetterInstance
			if setterInstance == nil {
				// Direct field — store via memory
				fieldType := propertyInstance.GetterInstance.Signature.ReturnType
				if fieldType == nil {
					continue
				}
				valueExpr := c.CompileExpression(values[i], fieldType, ConstraintsConvImplicit)
				thisRef := mod.LocalGet(tempIdx, sizeTypeRef)
				stmts = append(stmts, mod.Store(
					uint32(fieldType.ByteSize()),
					thisRef, valueExpr, fieldType.ToRef(),
					uint32(propertyInstance.MemoryOffset), uint32(fieldType.AlignLog2()),
					"",
				))
			} else {
				// Has setter — call it
				valueExpr := c.CompileExpression(values[i], setterInstance.Signature.ParameterTypes[0], ConstraintsConvImplicit)
				stmts = append(stmts, c.makeCallDirect(setterInstance,
					[]module.ExpressionRef{mod.LocalGet(tempIdx, sizeTypeRef), valueExpr},
					expression, false,
				))
			}
		}
	}

	stmts = append(stmts, mod.LocalGet(tempIdx, sizeTypeRef))
	c.CurrentType = contextualType
	return mod.Block("", stmts, sizeTypeRef)
}

// compileNewExpression compiles a new expression.
// Ported from: assemblyscript/src/compiler.ts compileNewExpression (lines 8805-8865).
func (c *Compiler) compileNewExpression(expression *ast.NewExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	resolver := c.Resolver()

	// Obtain the class being instantiated
	sourceFunc := fl.SourceFunction()
	target := resolver.ResolveTypeName(
		expression.TypeName,
		fl,
		sourceFunc.(program.Element),
		program.ReportModeReport,
	)
	if target == nil {
		return mod.Unreachable()
	}
	if target.GetElementKind() != program.ElementKindClassPrototype {
		c.Error(
			diagnostics.DiagnosticCodeThisExpressionIsNotConstructable,
			expression.TypeName.GetRange(),
			"", "", "",
		)
		return mod.Unreachable()
	}
	if target.Is(common.CommonFlagsAbstract) {
		c.Error(
			diagnostics.DiagnosticCodeCannotCreateAnInstanceOfAnAbstractClass,
			expression.TypeName.GetRange(),
			"", "", "",
		)
		return mod.Unreachable()
	}

	classPrototype := target.(*program.ClassPrototype)

	// Resolve type arguments to get a concrete class instance.
	var classInstance *program.Class
	typeArguments := expression.TypeArguments
	if len(typeArguments) == 0 {
		// Check if we can infer generic type arguments from contextual type
		// e.g. `arr: Array<T> = new Array()`
		classReference := contextualType.ClassRef
		if classReference != nil {
			if classRef, ok := classReference.(*program.Class); ok {
				if classRef.Prototype == classPrototype && classRef.Is(common.CommonFlagsGeneric) {
					classInstance = resolver.ResolveClass(
						classPrototype,
						classRef.TypeArguments,
						cloneTypeArgMap(fl.ContextualTypeArguments()),
						program.ReportModeReport,
					)
				}
			}
		}
	}
	if classInstance == nil {
		classInstance = resolver.ResolveClassInclTypeArguments(
			classPrototype,
			typeArguments,
			fl,
			sourceFunc.(program.Element).GetParent(),
			cloneTypeArgMap(fl.ContextualTypeArguments()),
			expression,
			program.ReportModeReport,
		)
	}
	if classInstance == nil {
		return mod.Unreachable()
	}
	if contextualType == types.TypeVoid {
		constraints |= ConstraintsWillDrop
	}

	// Ensure the constructor exists and is compiled
	ctorInstance := c.ensureConstructor(classInstance, expression)
	if ctorInstance == nil {
		return mod.Unreachable()
	}

	// Check field initialization (unless inline decorator)
	if !ctorInstance.HasDecorator(program.DecoratorFlagsInline) {
		c.checkFieldInitialization(classInstance, expression)
	}

	return c.compileInstantiate(ctorInstance, expression.Args, constraints, expression)
}

// compileInstantiate compiles a class instantiation via constructor call.
// Ported from: assemblyscript/src/compiler.ts compileInstantiate (lines 9050-9076).
func (c *Compiler) compileInstantiate(
	ctorInstance *program.Function,
	argumentExpressions []ast.Node,
	constraints Constraints,
	reportNode ast.Node,
) module.ExpressionRef {
	parent := ctorInstance.GetParent()
	classInstance := parent.(*program.Class)
	// TODO: checkUnsafe if classInstance.type.isUnmanaged or ctorInstance.hasDecorator(Unsafe)
	expr := c.compileCallDirect(ctorInstance, argumentExpressions, reportNode,
		c.makeZeroOfType(c.Options().UsizeType()), constraints)
	if module.GetExpressionType(expr) != module.TypeRefNone {
		// Important because a super ctor could be called
		c.CurrentType = classInstance.GetResolvedType()
	}
	return expr
}

// compileParenthesizedExpression compiles a parenthesized expression.
// Ported from: assemblyscript/src/compiler.ts compileParenthesizedExpression.
func (c *Compiler) compileParenthesizedExpression(expression *ast.ParenthesizedExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	// Just compile the inner expression
	return c.CompileExpression(expression.Expression, contextualType, constraints)
}

// compilePropertyAccessExpression compiles a property access expression.
// Ported from: assemblyscript/src/compiler.ts compilePropertyAccessExpression (lines 5080-5340).
func (c *Compiler) compilePropertyAccessExpression(expression *ast.PropertyAccessExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	// Ported from: assemblyscript/src/compiler.ts compilePropertyAccessExpression (lines 9078-9210).
	c.maybeCompileEnclosingSource(expression)

	resolver := c.Resolver()
	target := resolver.LookupExpression(expression, fl, contextualType, program.ReportModeReport)
	if target == nil {
		return mod.Unreachable()
	}
	thisExpression := resolver.CurrentThisExpression
	if target.HasDecorator(program.DecoratorFlagsUnsafe) {
		c.checkUnsafe(expression, nil)
	}

	switch target.GetElementKind() {
	case program.ElementKindGlobal:
		// static field
		global := target.(*program.Global)
		if !c.CompileGlobalLazy(global, expression) {
			return mod.Unreachable()
		}
		globalType := global.GetResolvedType()
		if globalType == types.TypeVoid {
			return mod.Unreachable()
		}
		if _, pending := c.PendingElements[global]; pending {
			c.Error(
				diagnostics.DiagnosticCodeVariable0UsedBeforeItsDeclaration,
				expression.GetRange(),
				global.GetInternalName(), "", "",
			)
			c.CurrentType = globalType
			return mod.Unreachable()
		}
		if global.Is(common.CommonFlagsInlined) {
			return c.compileInlineConstant(global, contextualType, constraints)
		}
		expr := mod.GlobalGet(global.GetInternalName(), globalType.ToRef())
		if global.Is(common.CommonFlagsDefinitelyAssigned) && globalType.IsReference() && !globalType.IsNullableReference() {
			expr = c.makeRuntimeNonNullCheck(expr, globalType, expression)
		}
		c.CurrentType = globalType
		return expr

	case program.ElementKindEnumValue:
		// enum value
		enumValue := target.(*program.EnumValue)
		parent := enumValue.GetParent()
		enum := parent.(*program.Enum)
		if !c.CompileEnum(enum) {
			c.CurrentType = types.TypeI32
			return mod.Unreachable()
		}
		c.CurrentType = types.TypeI32
		if enumValue.Is(common.CommonFlagsInlined) {
			return c.compileInlineConstant(enumValue, contextualType, constraints)
		}
		return mod.GlobalGet(enumValue.GetInternalName(), module.TypeRefI32)

	case program.ElementKindPropertyPrototype:
		propertyPrototype := target.(*program.PropertyPrototype)
		resolvedProperty := resolver.ResolveProperty(propertyPrototype, program.ReportModeReport)
		if resolvedProperty == nil {
			return mod.Unreachable()
		}
		return c.compilePropertyAccessProperty(resolvedProperty, fl, thisExpression, expression)

	case program.ElementKindProperty:
		propertyInstance := target.(*program.Property)
		return c.compilePropertyAccessProperty(propertyInstance, fl, thisExpression, expression)
	}

	c.Error(
		diagnostics.DiagnosticCodeExpressionDoesNotCompileToAValueAtRuntime,
		expression.GetRange(),
		"", "", "",
	)
	return mod.Unreachable()
}

// compilePropertyAccessProperty handles the Property case for compilePropertyAccessExpression.
// Ported from: assemblyscript/src/compiler.ts compilePropertyAccessExpression Property case (lines 9145-9173).
func (c *Compiler) compilePropertyAccessProperty(propertyInstance *program.Property, fl *flow.Flow, thisExpression ast.Node, expression ast.Node) module.ExpressionRef {
	mod := c.Module()
	if propertyInstance.IsField() {
		if fl.SourceFunction().Is(uint32(common.CommonFlagsConstructor)) &&
			thisExpression != nil && thisExpression.GetKind() == ast.NodeKindThis &&
			!fl.IsThisFieldFlag(propertyInstance, flow.FieldFlagInitialized) &&
			!propertyInstance.Is(common.CommonFlagsDefinitelyAssigned) {
			c.ErrorRelated(
				diagnostics.DiagnosticCodeProperty0IsUsedBeforeBeingAssigned,
				expression.GetRange(),
				propertyInstance.IdentifierNode().GetRange(),
				propertyInstance.GetInternalName(), "", "",
			)
		}
	}
	getterInstance := propertyInstance.GetterInstance
	if getterInstance == nil {
		return mod.Unreachable() // failed earlier
	}
	var thisArg module.ExpressionRef
	if getterInstance.Is(common.CommonFlagsInstance) {
		thisArg = c.CompileExpression(
			thisExpression,
			getterInstance.Signature.ThisType,
			ConstraintsConvImplicit|ConstraintsIsThis,
		)
	}
	return c.compileCallDirect(getterInstance, nil, expression, thisArg, ConstraintsNone)
}

// compileTernaryExpression compiles a ternary expression (condition ? ifThen : ifElse).
// Ported from: assemblyscript/src/compiler.ts compileTernaryExpression (lines 5342-5435).
func (c *Compiler) compileTernaryExpression(expression *ast.TernaryExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	ifThen := expression.IfThen
	ifElse := expression.IfElse

	condExpr := c.CompileExpression(expression.Condition, types.TypeBool, ConstraintsNone)
	condExprTrueish := c.makeIsTrueish(condExpr, c.CurrentType, expression.Condition)

	// Try to eliminate unnecessary branches if the condition is constant
	// FIXME: skips common denominator, inconsistently picking branch type
	condKind := c.evaluateCondition(condExprTrueish)
	if condKind == flow.ConditionKindTrue {
		return mod.MaybeDropCondition(condExprTrueish, c.CompileExpression(ifThen, contextualType, ConstraintsNone))
	}
	if condKind == flow.ConditionKindFalse {
		return mod.MaybeDropCondition(condExprTrueish, c.CompileExpression(ifElse, contextualType, ConstraintsNone))
	}

	outerFlow := c.CurrentFlow
	ifThenFlow := outerFlow.ForkThen(condExpr, false, false)
	c.CurrentFlow = ifThenFlow
	ifThenExpr := c.CompileExpression(ifThen, contextualType, ConstraintsNone)
	ifThenType := c.CurrentType

	ifElseCtx := contextualType
	if ifElseCtx == types.TypeAuto {
		ifElseCtx = ifThenType
	}
	ifElseFlow := outerFlow.ForkElse(condExpr)
	c.CurrentFlow = ifElseFlow
	ifElseExpr := c.CompileExpression(ifElse, ifElseCtx, ConstraintsNone)
	ifElseType := c.CurrentType

	if contextualType == types.TypeVoid {
		// Values, including type mismatch, are irrelevant
		if ifThenType != types.TypeVoid {
			ifThenExpr = mod.Drop(ifThenExpr)
			ifThenType = types.TypeVoid
		}
		if ifElseType != types.TypeVoid {
			ifElseExpr = mod.Drop(ifElseExpr)
			ifElseType = types.TypeVoid
		}
		c.CurrentType = types.TypeVoid
	} else {
		commonType := types.CommonType(ifThenType, ifElseType, contextualType, false)
		if commonType == nil {
			c.Error(
				diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
				ifElse.GetRange(), ifElseType.String(), ifThenType.String(), "",
			)
			c.CurrentType = contextualType
			return mod.Unreachable()
		}
		ifThenExpr = c.convertExpression(ifThenExpr, ifThenType, commonType, false, ifThen)
		_ = ifThenType
		ifElseExpr = c.convertExpression(ifElseExpr, ifElseType, commonType, false, ifElse)
		_ = ifElseType
		c.CurrentType = commonType
	}

	outerFlow.InheritAlternatives(ifThenFlow, ifElseFlow)
	c.CurrentFlow = outerFlow

	return mod.If(condExprTrueish, ifThenExpr, ifElseExpr)
}

// compileUnaryPostfixExpression compiles a unary postfix expression (x++, x--).
// Ported from: assemblyscript/src/compiler.ts compileUnaryPostfixExpression (lines 9277-9518).
func (c *Compiler) compileUnaryPostfixExpression(expression *ast.UnaryPostfixExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	resolver := c.Resolver()

	// Make a getter for the expression (also obtains the type)
	getValueOriginal := c.CompileExpression(expression.Operand, contextualType.ExceptVoid(), ConstraintsNone)
	var getValue module.ExpressionRef

	// If the value isn't dropped, a temp local is required to remember the original value,
	// except if a static overload is found, which reverses the use of a temp (see below)
	var tempLocal flow.FlowLocalRef
	if contextualType != types.TypeVoid {
		tempLocal = fl.GetTempLocal(c.CurrentType)
		getValue = mod.LocalTee(tempLocal.FlowIndex(), getValueOriginal, c.CurrentType.IsManaged(), c.CurrentType.ToRef())
	} else {
		getValue = getValueOriginal
	}

	var expr module.ExpressionRef

	switch expression.Operator {
	case tokenizer.TokenPlusPlus:
		// Check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.FindOverload(program.OperatorKindPostfixInc, false)
				if overload != nil {
					isInstance := overload.Is(common.CommonFlagsInstance)
					if tempLocal != nil && !isInstance {
						// Revert: static overload simply returns
						getValue = getValueOriginal
						tempLocal = nil
					}
					expr = c.compileUnaryOverload(overload, expression.Operand, getValue, expression)
					if isInstance {
						break
					}
					return expr
				}
			}
		}
		if !c.CurrentType.IsValue() {
			c.Error(
				diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(),
				"++", c.CurrentType.String(), "",
			)
			return mod.Unreachable()
		}
		switch c.CurrentType.Kind {
		case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
			types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			expr = mod.Binary(module.BinaryOpAddI32, getValue, mod.I32(1))
		case types.TypeKindI64, types.TypeKindU64:
			expr = mod.Binary(module.BinaryOpAddI64, getValue, mod.I64(1))
		case types.TypeKindIsize, types.TypeKindUsize:
			expr = mod.Binary(module.BinaryOpAddSize, getValue, c.makeOneOfType(c.CurrentType))
		case types.TypeKindF32:
			expr = mod.Binary(module.BinaryOpAddF32, getValue, mod.F32(1))
		case types.TypeKindF64:
			expr = mod.Binary(module.BinaryOpAddF64, getValue, mod.F64(1))
		default:
			c.Error(
				diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(),
				"++", c.CurrentType.String(), "",
			)
			return mod.Unreachable()
		}

	case tokenizer.TokenMinusMinus:
		// Check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.FindOverload(program.OperatorKindPostfixDec, false)
				if overload != nil {
					isInstance := overload.Is(common.CommonFlagsInstance)
					if tempLocal != nil && !isInstance {
						// Revert: static overload simply returns
						getValue = getValueOriginal
						tempLocal = nil
					}
					expr = c.compileUnaryOverload(overload, expression.Operand, getValue, expression)
					if isInstance {
						break
					}
					return expr
				}
			}
		}
		if !c.CurrentType.IsValue() {
			c.Error(
				diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(),
				"--", c.CurrentType.String(), "",
			)
			return mod.Unreachable()
		}
		switch c.CurrentType.Kind {
		case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
			types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			expr = mod.Binary(module.BinaryOpSubI32, getValue, mod.I32(1))
		case types.TypeKindI64, types.TypeKindU64:
			expr = mod.Binary(module.BinaryOpSubI64, getValue, mod.I64(1))
		case types.TypeKindIsize, types.TypeKindUsize:
			expr = mod.Binary(module.BinaryOpSubSize, getValue, c.makeOneOfType(c.CurrentType))
		case types.TypeKindF32:
			expr = mod.Binary(module.BinaryOpSubF32, getValue, mod.F32(1))
		case types.TypeKindF64:
			expr = mod.Binary(module.BinaryOpSubF64, getValue, mod.F64(1))
		default:
			c.Error(
				diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(),
				"--", c.CurrentType.String(), "",
			)
			return mod.Unreachable()
		}

	default:
		return mod.Unreachable()
	}

	// Look up the assignment target
	target := resolver.LookupExpression(expression.Operand, fl, nil, program.ReportModeReport)
	if target == nil {
		return mod.Unreachable()
	}

	// Simplify if dropped anyway (contextualType == void)
	if tempLocal == nil {
		return c.makeAssignment(
			target,
			expr,
			c.CurrentType,
			expression.Operand,
			resolver.CurrentThisExpression,
			resolver.CurrentElementExpression,
			false,
		)
	}

	// Otherwise use the temp local for the intermediate value (always possibly overflows)
	setValue := c.makeAssignment(
		target,
		expr, // includes a tee of getValue to tempLocal
		c.CurrentType,
		expression.Operand,
		resolver.CurrentThisExpression,
		resolver.CurrentElementExpression,
		false,
	)

	c.CurrentType = tempLocal.GetType()
	typeRef := tempLocal.GetType().ToRef()

	return mod.Block("", []module.ExpressionRef{
		setValue,
		mod.LocalGet(tempLocal.FlowIndex(), typeRef),
	}, typeRef) // result of 'x++' / 'x--' might overflow
}

// compileUnaryPrefixExpression compiles a unary prefix expression (++x, --x, -x, !x, ~x, etc.).
// Ported from: assemblyscript/src/compiler.ts compileUnaryPrefixExpression (lines 9521-9875).
func (c *Compiler) compileUnaryPrefixExpression(expression *ast.UnaryPrefixExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	compound := false
	var expr module.ExpressionRef

	switch expression.Operator {
	case tokenizer.TokenPlus:
		// Unary plus: nop, but check for operator overload.
		// Ported from: assemblyscript/src/compiler.ts Token.Plus (lines 9531-9554).
		expr = c.CompileExpression(expression.Operand, contextualType.ExceptVoid(), ConstraintsNone)

		// check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.LookupOverload(program.OperatorKindPlus)
				if overload != nil {
					return c.compileUnaryOverload(overload.(*program.Function), expression.Operand, expr, expression)
				}
			}
		}
		if !c.CurrentType.IsValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "+", c.CurrentType.String(), "")
			return mod.Unreachable()
		}
		// nop

	case tokenizer.TokenMinus:
		// Unary minus: negate with overload check.
		// Ported from: assemblyscript/src/compiler.ts Token.Minus (lines 9555-9626).
		operand := expression.Operand
		// TODO: isNumericLiteral → compileLiteralExpression with negate=true
		expr = c.CompileExpression(operand, contextualType.ExceptVoid(), ConstraintsNone)

		// check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.LookupOverload(program.OperatorKindMinus)
				if overload != nil {
					return c.compileUnaryOverload(overload.(*program.Function), expression.Operand, expr, expression)
				}
			}
		}
		if !c.CurrentType.IsValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "-", c.CurrentType.String(), "")
			return mod.Unreachable()
		}

		switch c.CurrentType.Kind {
		case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
			types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			expr = mod.Binary(module.BinaryOpSubI32, mod.I32(0), expr)
		case types.TypeKindI64, types.TypeKindU64:
			expr = mod.Binary(module.BinaryOpSubI64, mod.I64(0), expr)
		case types.TypeKindIsize, types.TypeKindUsize:
			expr = mod.Binary(module.BinaryOpSubSize, c.makeZeroOfType(c.CurrentType), expr)
		case types.TypeKindF32:
			expr = mod.Unary(module.UnaryOpNegF32, expr)
		case types.TypeKindF64:
			expr = mod.Unary(module.UnaryOpNegF64, expr)
		default:
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "-", c.CurrentType.String(), "")
			expr = mod.Unreachable()
		}

	case tokenizer.TokenPlusPlus:
		// Prefix increment
		// Ported from: assemblyscript/src/compiler.ts Token.Plus_Plus (lines 9628-9695).
		compound = true
		expr = c.CompileExpression(expression.Operand, contextualType.ExceptVoid(), ConstraintsNone)

		// check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.FindOverload(program.OperatorKindPrefixInc, false)
				if overload != nil {
					expr = c.compileUnaryOverload(overload, expression.Operand, expr, expression)
					if overload.Is(common.CommonFlagsInstance) {
						break // re-assign
					}
					return expr // skip re-assign
				}
			}
		}
		if !c.CurrentType.IsValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "++", c.CurrentType.String(), "")
			return mod.Unreachable()
		}

		switch c.CurrentType.Kind {
		case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
			types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			expr = mod.Binary(module.BinaryOpAddI32, expr, mod.I32(1))
		case types.TypeKindI64, types.TypeKindU64:
			expr = mod.Binary(module.BinaryOpAddI64, expr, mod.I64(1))
		case types.TypeKindIsize, types.TypeKindUsize:
			expr = mod.Binary(module.BinaryOpAddSize, expr, c.makeOneOfType(c.CurrentType))
		case types.TypeKindF32:
			expr = mod.Binary(module.BinaryOpAddF32, expr, mod.F32(1))
		case types.TypeKindF64:
			expr = mod.Binary(module.BinaryOpAddF64, expr, mod.F64(1))
		default:
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "++", c.CurrentType.String(), "")
			expr = mod.Unreachable()
		}

	case tokenizer.TokenMinusMinus:
		// Prefix decrement
		// Ported from: assemblyscript/src/compiler.ts Token.Minus_Minus (lines 9697-9765).
		compound = true
		expr = c.CompileExpression(expression.Operand, contextualType.ExceptVoid(), ConstraintsNone)

		// check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.FindOverload(program.OperatorKindPrefixDec, false)
				if overload != nil {
					expr = c.compileUnaryOverload(overload, expression.Operand, expr, expression)
					if overload.Is(common.CommonFlagsInstance) {
						break // re-assign
					}
					return expr // skip re-assign
				}
			}
		}
		if !c.CurrentType.IsValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "--", c.CurrentType.String(), "")
			return mod.Unreachable()
		}

		switch c.CurrentType.Kind {
		case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
			types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			expr = mod.Binary(module.BinaryOpSubI32, expr, mod.I32(1))
		case types.TypeKindI64, types.TypeKindU64:
			expr = mod.Binary(module.BinaryOpSubI64, expr, mod.I64(1))
		case types.TypeKindIsize, types.TypeKindUsize:
			expr = mod.Binary(module.BinaryOpSubSize, expr, c.makeOneOfType(c.CurrentType))
		case types.TypeKindF32:
			expr = mod.Binary(module.BinaryOpSubF32, expr, mod.F32(1))
		case types.TypeKindF64:
			expr = mod.Binary(module.BinaryOpSubF64, expr, mod.F64(1))
		default:
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "--", c.CurrentType.String(), "")
			expr = mod.Unreachable()
		}

	case tokenizer.TokenExclamation:
		// Logical NOT with operator overload check.
		// Ported from: assemblyscript/src/compiler.ts Token.Exclamation (lines 9766-9784).
		expr = c.CompileExpression(expression.Operand, contextualType.ExceptVoid(), ConstraintsNone)

		// check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.LookupOverload(program.OperatorKindNot)
				if overload != nil {
					return c.compileUnaryOverload(overload.(*program.Function), expression.Operand, expr, expression)
				}
			}
			// fall back to compare by value
		}

		expr = mod.Unary(module.UnaryOpEqzI32, c.makeIsTrueish(expr, c.CurrentType, expression.Operand))
		c.CurrentType = types.TypeBool

	case tokenizer.TokenTilde:
		// Bitwise NOT: coerce contextual type (void→i32, float→i64)
		// Ported from: assemblyscript/src/compiler.ts Token.Tilde (lines 9785-9846).
		tildeCtx := contextualType
		if tildeCtx == types.TypeVoid {
			tildeCtx = types.TypeI32
		} else if tildeCtx.IsFloatValue() {
			tildeCtx = types.TypeI64
		}
		expr = c.CompileExpression(expression.Operand, tildeCtx, ConstraintsNone)

		// check operator overload
		classRef := c.CurrentType.GetClassOrWrapper(c.Program)
		if classRef != nil {
			if classInstance, ok := classRef.(*program.Class); ok {
				overload := classInstance.LookupOverload(program.OperatorKindBitwiseNot)
				if overload != nil {
					return c.compileUnaryOverload(overload.(*program.Function), expression.Operand, expr, expression)
				}
			}
		}
		if !c.CurrentType.IsValue() {
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "~", c.CurrentType.String(), "")
			return mod.Unreachable()
		}

		// convert to integer type (e.g. f32→i32, f64→i64)
		expr = c.convertExpression(expr, c.CurrentType, c.CurrentType.IntType(), false, expression.Operand)

		switch c.CurrentType.Kind {
		case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
			types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			expr = mod.Binary(module.BinaryOpXorI32, expr, mod.I32(-1))
		case types.TypeKindI64, types.TypeKindU64:
			expr = mod.Binary(module.BinaryOpXorI64, expr, mod.I64(-1))
		case types.TypeKindIsize, types.TypeKindUsize:
			expr = mod.Binary(module.BinaryOpXorSize, expr, c.makeNegOneOfType(c.CurrentType))
		default:
			c.Error(diagnostics.DiagnosticCodeThe0OperatorCannotBeAppliedToType1,
				expression.GetRange(), "~", c.CurrentType.String(), "")
			expr = mod.Unreachable()
		}

	case tokenizer.TokenTypeOf:
		// Ported from: assemblyscript/src/compiler.ts compileTypeof (lines 9877-9959).
		return c.compileTypeof(expression, contextualType, constraints)

	case tokenizer.TokenDotDotDot:
		c.Error(diagnostics.DiagnosticCodeNotImplemented0, expression.GetRange(), "Spread operator", "", "")
		return mod.Unreachable()

	default:
		return mod.Unreachable()
	}

	if !compound {
		return expr
	}

	// Compound assignment for prefix ++/--: look up target and assign
	resolver := c.Resolver()
	target := resolver.LookupExpression(expression.Operand, c.CurrentFlow, nil, program.ReportModeReport)
	if target == nil {
		return mod.Unreachable()
	}
	return c.makeAssignment(
		target,
		expr,
		c.CurrentType,
		expression.Operand,
		resolver.CurrentThisExpression,
		resolver.CurrentElementExpression,
		contextualType != types.TypeVoid,
	)
}

// compileTypeof compiles a typeof expression. Returns a string constant for value types,
// and resolves reference types to their typeof string.
// Ported from: assemblyscript/src/compiler.ts compileTypeof (lines 9877-9959).
func (c *Compiler) compileTypeof(
	expression *ast.UnaryPrefixExpression,
	contextualType *types.Type,
	constraints Constraints,
) module.ExpressionRef {
	mod := c.Module()
	operand := expression.Operand
	var expr module.ExpressionRef
	stringInstance := c.Program.StringInstance()
	var typeString string

	if operand.GetKind() == ast.NodeKindNull {
		// special since `null` without type context is usize
		typeString = "object"
	} else {
		resolver := c.Resolver()
		element := resolver.LookupExpression(operand, c.CurrentFlow, types.TypeAuto, program.ReportModeSwallow)
		if element == nil {
			switch operand.GetKind() {
			case ast.NodeKindIdentifier:
				// ignore error: typeof doesntExist -> undefined
			case ast.NodeKindPropertyAccess, ast.NodeKindElementAccess:
				var targetOperand ast.Node
				if operand.GetKind() == ast.NodeKindPropertyAccess {
					targetOperand = operand.(*ast.PropertyAccessExpression).Expression
				} else {
					targetOperand = operand.(*ast.ElementAccessExpression).Expression
				}
				targetType := resolver.ResolveExpression(targetOperand, c.CurrentFlow, types.TypeAuto, program.ReportModeReport)
				if targetType == nil {
					// access on non-object
					c.CurrentType = stringInstance.GetResolvedType()
					return mod.Unreachable()
				}
				// fall-through to default
				fallthrough
			default:
				expr = c.CompileExpression(operand, types.TypeAuto, ConstraintsNone) // may trigger an error
				expr = c.convertExpression(expr, c.CurrentType, types.TypeVoid, true, operand)
			}
			typeString = "undefined"
		} else {
			switch element.GetElementKind() {
			case program.ElementKindClassPrototype,
				program.ElementKindNamespace,
				program.ElementKindEnum:
				typeString = "object"

			case program.ElementKindFunctionPrototype:
				typeString = "function"

			default:
				expr = c.CompileExpression(operand, types.TypeAuto, ConstraintsNone)
				typ := c.CurrentType
				expr = c.convertExpression(expr, typ, types.TypeVoid, true, operand)
				if typ.IsReference() {
					signatureReference := typ.GetSignature()
					if signatureReference != nil {
						typeString = "function"
					} else {
						classReference := typ.GetClass()
						if classReference != nil {
							classInst, ok := classReference.(*program.Class)
							if ok && classInst.Prototype == stringInstance.Prototype {
								typeString = "string"
							} else {
								typeString = "object"
							}
						} else {
							typeString = "externref" // TODO?
						}
					}
				} else if typ == types.TypeBool {
					typeString = "boolean"
				} else if typ.IsNumericValue() {
					typeString = "number"
				} else {
					typeString = "undefined" // failed to compile?
				}
			}
		}
	}

	c.CurrentType = stringInstance.GetResolvedType()
	if expr != 0 {
		return mod.Block("", []module.ExpressionRef{expr, c.EnsureStaticString(typeString)}, c.Options().SizeTypeRef())
	}
	return c.EnsureStaticString(typeString)
}

// compilePropertyGet compiles reading a property value, either via a getter call or direct memory load.
// Ported from: assemblyscript/src/compiler.ts compilePropertyAccessExpression property handling.
func (c *Compiler) compilePropertyGet(propertyInstance *program.Property, thisExpression ast.Node, reportNode ast.Node, constraints Constraints) module.ExpressionRef {
	mod := c.Module()

	// If the property has a getter, call it
	getterInstance := propertyInstance.GetterInstance
	if getterInstance != nil {
		var thisArg module.ExpressionRef
		if getterInstance.Is(common.CommonFlagsInstance) && thisExpression != nil {
			thisArg = c.CompileExpression(
				thisExpression,
				getterInstance.Signature.ThisType,
				ConstraintsConvImplicit|ConstraintsIsThis,
			)
		}
		return c.compileCallDirect(getterInstance, nil, reportNode, thisArg, constraints)
	}

	// Direct field access — load from memory
	if propertyInstance.IsField() && propertyInstance.MemoryOffset >= 0 {
		fieldType := propertyInstance.GetResolvedType()
		if fieldType == nil {
			return mod.Unreachable()
		}
		var thisArg module.ExpressionRef
		if thisExpression != nil {
			thisArg = c.CompileExpression(thisExpression, c.Options().UsizeType(), ConstraintsConvImplicit|ConstraintsIsThis)
		} else {
			thisArg = mod.I32(0) // shouldn't happen for instance fields
		}
		c.CurrentType = fieldType
		return mod.Load(
			uint32(fieldType.ByteSize()), fieldType.IsSignedIntegerValue(),
			thisArg, fieldType.ToRef(),
			uint32(propertyInstance.MemoryOffset), uint32(fieldType.AlignLog2()),
			"",
		)
	}

	return mod.Unreachable()
}

// compilePropertySet compiles writing a property value, either via a setter call or direct memory store.
// Ported from: assemblyscript/src/compiler.ts compilePropertyAccessExpression assignment handling.
func (c *Compiler) compilePropertySet(propertyInstance *program.Property, thisExpression ast.Node, valueExpr module.ExpressionRef, valueType *types.Type, reportNode ast.Node) module.ExpressionRef {
	mod := c.Module()

	// If the property has a setter, call it
	setterInstance := propertyInstance.SetterInstance
	if setterInstance != nil {
		operands := make([]module.ExpressionRef, 0, 2)
		if setterInstance.Is(common.CommonFlagsInstance) && thisExpression != nil {
			operands = append(operands, c.CompileExpression(
				thisExpression,
				setterInstance.Signature.ThisType,
				ConstraintsConvImplicit|ConstraintsIsThis,
			))
		}
		operands = append(operands, valueExpr)
		c.CurrentType = valueType
		return c.makeCallDirect(setterInstance, operands, reportNode, false)
	}

	// Direct field access — store to memory
	if propertyInstance.IsField() && propertyInstance.MemoryOffset >= 0 {
		fieldType := propertyInstance.GetResolvedType()
		if fieldType == nil {
			fieldType = valueType
		}
		var thisArg module.ExpressionRef
		if thisExpression != nil {
			thisArg = c.CompileExpression(thisExpression, c.Options().UsizeType(), ConstraintsConvImplicit|ConstraintsIsThis)
		} else {
			thisArg = mod.I32(0)
		}
		c.CurrentType = valueType
		return mod.Store(
			uint32(fieldType.ByteSize()),
			thisArg, valueExpr, fieldType.ToRef(),
			uint32(propertyInstance.MemoryOffset), uint32(fieldType.AlignLog2()),
			"",
		)
	}

	c.Error(
		diagnostics.DiagnosticCodeNotImplemented0,
		reportNode.GetRange(),
		"Property has no setter", "", "",
	)
	c.CurrentType = valueType
	return mod.Unreachable()
}

// lookupBinaryOverload is a helper that looks up a binary operator overload on a type
// and returns it as *program.Function, or nil if not found.
func (c *Compiler) lookupBinaryOverload(typ *types.Type, kind program.OperatorKind) *program.Function {
	classRef := typ.GetClassOrWrapper(c.Program)
	if classRef != nil {
		overload := classRef.LookupOverload(kind)
		if overload != nil {
			return overload.(*program.Function)
		}
	}
	return nil
}

// --- Binary operation helpers ---
// Ported from: assemblyscript/src/compiler.ts (lines 4793-5791).

// makeBinaryAdd compiles an addition operation.
// Does not care about garbage bits or signedness.
// Ported from: assemblyscript/src/compiler.ts makeAdd (lines 5021-5041).
func (c *Compiler) makeBinaryAdd(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindU8, types.TypeKindU16, types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpAddI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpAddI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpAddSize, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpAddF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpAddF64, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeBinarySub compiles a subtraction operation.
// Does not care about garbage bits or signedness.
// Ported from: assemblyscript/src/compiler.ts makeSub (lines 5043-5063).
func (c *Compiler) makeBinarySub(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindU8, types.TypeKindU16, types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpSubI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpSubI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpSubSize, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpSubF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpSubF64, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeBinaryMul compiles a multiplication operation.
// Does not care about garbage bits or signedness.
// Ported from: assemblyscript/src/compiler.ts makeMul (lines 5065-5085).
func (c *Compiler) makeBinaryMul(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindU8, types.TypeKindU16, types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpMulI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpMulI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpMulSize, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpMulF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpMulF64, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryDiv(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpDivI32, left, right)
	case types.TypeKindI32:
		return mod.Binary(module.BinaryOpDivI32, left, right)
	case types.TypeKindI64:
		return mod.Binary(module.BinaryOpDivI64, left, right)
	case types.TypeKindIsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpDivI64, left, right)
		}
		return mod.Binary(module.BinaryOpDivI32, left, right)
	case types.TypeKindBool, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpDivU32, left, right)
	case types.TypeKindU32:
		return mod.Binary(module.BinaryOpDivU32, left, right)
	case types.TypeKindU64:
		return mod.Binary(module.BinaryOpDivU64, left, right)
	case types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpDivU64, left, right)
		}
		return mod.Binary(module.BinaryOpDivU32, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpDivF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpDivF64, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryRem(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpRemI32, left, right)
	case types.TypeKindI32:
		return mod.Binary(module.BinaryOpRemI32, left, right)
	case types.TypeKindI64:
		return mod.Binary(module.BinaryOpRemI64, left, right)
	case types.TypeKindIsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpRemI64, left, right)
		}
		return mod.Binary(module.BinaryOpRemI32, left, right)
	case types.TypeKindBool, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpRemU32, left, right)
	case types.TypeKindU32:
		return mod.Binary(module.BinaryOpRemU32, left, right)
	case types.TypeKindU64:
		return mod.Binary(module.BinaryOpRemU64, left, right)
	case types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpRemU64, left, right)
		}
		return mod.Binary(module.BinaryOpRemU32, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeBinaryShl compiles a shift left operation.
// Cares about garbage bits on the RHS, but only for types smaller than 5 bits.
// Ported from: assemblyscript/src/compiler.ts makeShl (lines 5435-5464).
func (c *Compiler) makeBinaryShl(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool:
		return left
	case types.TypeKindI8, types.TypeKindI16, types.TypeKindU8, types.TypeKindU16:
		// leftExpr << (rightExpr & (size-1))
		return mod.Binary(module.BinaryOpShlI32, left,
			mod.Binary(module.BinaryOpAndI32, right, mod.I32(typ.Size-1)))
	case types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpShlI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpShlI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpShlSize, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeBinaryShr compiles a shift right operation (signed or unsigned).
// Cares about garbage bits on the LHS, but on the RHS only for types smaller than 5 bits.
// When signed=true, ports makeShr (lines 5466-5507). When signed=false, ports makeShru (lines 5509-5538).
func (c *Compiler) makeBinaryShr(left, right module.ExpressionRef, typ *types.Type, signed bool) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	if signed {
		// makeShr: signedness matters for the shift op
		switch typ.Kind {
		case types.TypeKindBool:
			return left
		case types.TypeKindI8, types.TypeKindI16:
			// Signed small: ShrI32 with ensureSmallIntegerWrap on LHS, mask RHS
			return mod.Binary(module.BinaryOpShrI32,
				c.ensureSmallIntegerWrap(left, typ),
				mod.Binary(module.BinaryOpAndI32, right, mod.I32(typ.Size-1)))
		case types.TypeKindU8, types.TypeKindU16:
			// Unsigned small: ShrU32 with ensureSmallIntegerWrap on LHS, mask RHS
			return mod.Binary(module.BinaryOpShrU32,
				c.ensureSmallIntegerWrap(left, typ),
				mod.Binary(module.BinaryOpAndI32, right, mod.I32(typ.Size-1)))
		case types.TypeKindI32:
			return mod.Binary(module.BinaryOpShrI32, left, right)
		case types.TypeKindI64:
			return mod.Binary(module.BinaryOpShrI64, left, right)
		case types.TypeKindIsize:
			return mod.Binary(module.BinaryOpShrISize, left, right)
		case types.TypeKindU32:
			return mod.Binary(module.BinaryOpShrU32, left, right)
		case types.TypeKindU64:
			return mod.Binary(module.BinaryOpShrU64, left, right)
		case types.TypeKindUsize:
			return mod.Binary(module.BinaryOpShrUSize, left, right)
		default:
			return mod.Unreachable()
		}
	}
	// makeShru: always unsigned shift, cares about garbage bits on LHS
	switch typ.Kind {
	case types.TypeKindBool:
		return left
	case types.TypeKindI8, types.TypeKindI16, types.TypeKindU8, types.TypeKindU16:
		// leftExpr >>> (rightExpr & (size-1))
		return mod.Binary(module.BinaryOpShrU32,
			c.ensureSmallIntegerWrap(left, typ),
			mod.Binary(module.BinaryOpAndI32, right, mod.I32(typ.Size-1)))
	case types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpShrU32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpShrU64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpShrUSize, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeBinaryAnd compiles a bitwise AND operation.
// Does not care about garbage bits or signedness.
// Ported from: assemblyscript/src/compiler.ts makeAnd (lines 5540-5558).
func (c *Compiler) makeBinaryAnd(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
		return mod.Binary(module.BinaryOpAndI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpAndI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpAndSize, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeBinaryOr compiles a bitwise OR operation.
// Does not care about garbage bits or signedness.
// Ported from: assemblyscript/src/compiler.ts makeOr (lines 5560-5578).
func (c *Compiler) makeBinaryOr(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindU8, types.TypeKindU16, types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpOrI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpOrI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpOrSize, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeBinaryXor compiles a bitwise XOR operation.
// Does not care about garbage bits or signedness.
// Ported from: assemblyscript/src/compiler.ts makeXor (lines 5580-5598).
func (c *Compiler) makeBinaryXor(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindU8, types.TypeKindU16, types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpXorI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpXorI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpXorSize, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryEq(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpEqI32, left, right)
	case types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpEqI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpEqI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpEqI64, left, right)
		}
		return mod.Binary(module.BinaryOpEqI32, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpEqF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpEqF64, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryNe(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpNeI32, left, right)
	case types.TypeKindI32, types.TypeKindU32:
		return mod.Binary(module.BinaryOpNeI32, left, right)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpNeI64, left, right)
	case types.TypeKindIsize, types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpNeI64, left, right)
		}
		return mod.Binary(module.BinaryOpNeI32, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpNeF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpNeF64, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryLt(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpLtI32, left, right)
	case types.TypeKindI32:
		return mod.Binary(module.BinaryOpLtI32, left, right)
	case types.TypeKindI64:
		return mod.Binary(module.BinaryOpLtI64, left, right)
	case types.TypeKindIsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpLtI64, left, right)
		}
		return mod.Binary(module.BinaryOpLtI32, left, right)
	case types.TypeKindBool, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpLtU32, left, right)
	case types.TypeKindU32:
		return mod.Binary(module.BinaryOpLtU32, left, right)
	case types.TypeKindU64:
		return mod.Binary(module.BinaryOpLtU64, left, right)
	case types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpLtU64, left, right)
		}
		return mod.Binary(module.BinaryOpLtU32, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpLtF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpLtF64, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryGt(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpGtI32, left, right)
	case types.TypeKindI32:
		return mod.Binary(module.BinaryOpGtI32, left, right)
	case types.TypeKindI64:
		return mod.Binary(module.BinaryOpGtI64, left, right)
	case types.TypeKindIsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpGtI64, left, right)
		}
		return mod.Binary(module.BinaryOpGtI32, left, right)
	case types.TypeKindBool, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpGtU32, left, right)
	case types.TypeKindU32:
		return mod.Binary(module.BinaryOpGtU32, left, right)
	case types.TypeKindU64:
		return mod.Binary(module.BinaryOpGtU64, left, right)
	case types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpGtU64, left, right)
		}
		return mod.Binary(module.BinaryOpGtU32, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpGtF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpGtF64, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryLe(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpLeI32, left, right)
	case types.TypeKindI32:
		return mod.Binary(module.BinaryOpLeI32, left, right)
	case types.TypeKindI64:
		return mod.Binary(module.BinaryOpLeI64, left, right)
	case types.TypeKindIsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpLeI64, left, right)
		}
		return mod.Binary(module.BinaryOpLeI32, left, right)
	case types.TypeKindBool, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpLeU32, left, right)
	case types.TypeKindU32:
		return mod.Binary(module.BinaryOpLeU32, left, right)
	case types.TypeKindU64:
		return mod.Binary(module.BinaryOpLeU64, left, right)
	case types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpLeU64, left, right)
		}
		return mod.Binary(module.BinaryOpLeU32, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpLeF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpLeF64, left, right)
	default:
		return mod.Unreachable()
	}
}

func (c *Compiler) makeBinaryGe(left, right module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpGeI32, left, right)
	case types.TypeKindI32:
		return mod.Binary(module.BinaryOpGeI32, left, right)
	case types.TypeKindI64:
		return mod.Binary(module.BinaryOpGeI64, left, right)
	case types.TypeKindIsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpGeI64, left, right)
		}
		return mod.Binary(module.BinaryOpGeI32, left, right)
	case types.TypeKindBool, types.TypeKindU8, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		return mod.Binary(module.BinaryOpGeU32, left, right)
	case types.TypeKindU32:
		return mod.Binary(module.BinaryOpGeU32, left, right)
	case types.TypeKindU64:
		return mod.Binary(module.BinaryOpGeU64, left, right)
	case types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Binary(module.BinaryOpGeU64, left, right)
		}
		return mod.Binary(module.BinaryOpGeU32, left, right)
	case types.TypeKindF32:
		return mod.Binary(module.BinaryOpGeF32, left, right)
	case types.TypeKindF64:
		return mod.Binary(module.BinaryOpGeF64, left, right)
	default:
		return mod.Unreachable()
	}
}

// makeUnaryNeg compiles a unary negation operation.
// Ported from: assemblyscript/src/compiler.ts compileUnaryPrefixExpression Token.Minus (lines 9585-9624).
func (c *Compiler) makeUnaryNeg(expr module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
		return mod.Binary(module.BinaryOpSubI32, mod.I32(0), expr)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpSubI64, mod.I64(0), expr)
	case types.TypeKindIsize, types.TypeKindUsize:
		return mod.Binary(module.BinaryOpSubSize, c.makeZeroOfType(typ), expr)
	case types.TypeKindF32:
		return mod.Unary(module.UnaryOpNegF32, expr)
	case types.TypeKindF64:
		return mod.Unary(module.UnaryOpNegF64, expr)
	default:
		return mod.Unreachable()
	}
}

// compilePow compiles a power expression (x ** y).
// Ported from: assemblyscript/src/compiler.ts makePow (lines 5087-5230).
// TODO: Full implementation needs runtime function resolution (ipow32/ipow64 instances).
func (c *Compiler) compilePow(left, right module.ExpressionRef, typ *types.Type, reportNode ast.Node) module.ExpressionRef {
	mod := c.Module()
	c.CurrentType = typ
	switch typ.Kind {
	case types.TypeKindBool:
		// select(1, right == 0, left)
		return mod.Select(mod.I32(1),
			mod.Binary(module.BinaryOpEqI32, right, mod.I32(0)),
			left)
	case types.TypeKindI8, types.TypeKindU8, types.TypeKindI16, types.TypeKindU16:
		left = c.ensureSmallIntegerWrap(left, typ)
		right = c.ensureSmallIntegerWrap(right, typ)
		// fall through to I32 case — call ipow32, then wrap result
		expr := mod.Call(common.CommonNameIpow32, []module.ExpressionRef{left, right}, module.TypeRefI32)
		return c.ensureSmallIntegerWrap(expr, typ)
	case types.TypeKindI32, types.TypeKindU32:
		return mod.Call(common.CommonNameIpow32, []module.ExpressionRef{left, right}, module.TypeRefI32)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Call(common.CommonNameIpow64, []module.ExpressionRef{left, right}, module.TypeRefI64)
	case types.TypeKindIsize, types.TypeKindUsize:
		if c.Options().IsWasm64() {
			return mod.Call(common.CommonNameIpow64, []module.ExpressionRef{left, right}, module.TypeRefI64)
		}
		return mod.Call(common.CommonNameIpow32, []module.ExpressionRef{left, right}, module.TypeRefI32)
	case types.TypeKindF32:
		return mod.Call("~lib/math/NativeMathf.pow", []module.ExpressionRef{left, right}, module.TypeRefF32)
	case types.TypeKindF64:
		return mod.Call("~lib/math/NativeMath.pow", []module.ExpressionRef{left, right}, module.TypeRefF64)
	default:
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			reportNode.GetRange(),
			"Power operator type", "", "",
		)
		return mod.Unreachable()
	}
}

// --- Assignment helpers ---

// compileAssignment compiles an assignment expression.
// Ported from: assemblyscript/src/compiler.ts compileAssignment (lines 5656-5789).
func (c *Compiler) compileAssignment(expression ast.Node, valueExpression ast.Node, contextualType *types.Type) module.ExpressionRef {
	resolver := c.Resolver()
	fl := c.CurrentFlow
	target := resolver.LookupExpression(expression, fl, contextualType, program.ReportModeReport) // reports
	if target == nil {
		return c.Module().Unreachable()
	}
	thisExpression := resolver.CurrentThisExpression
	elementExpression := resolver.CurrentElementExpression

	// to compile just the value, we need to know the target's type
	var targetType *types.Type
	switch target.GetElementKind() {
	case program.ElementKindGlobal, program.ElementKindLocal:
		if target.GetElementKind() == program.ElementKindGlobal {
			if !c.CompileGlobalLazy(target.(*program.Global), expression) {
				return c.Module().Unreachable()
			}
		} else {
			local := target.(*program.Local)
			if !local.DeclaredByFlow(fl) {
				// TODO: closures
				c.Error(
					diagnostics.DiagnosticCodeNotImplemented0,
					expression.GetRange(),
					"Closures", "", "",
				)
				return c.Module().Unreachable()
			}
		}
		if _, pending := c.PendingElements[target]; pending {
			c.Error(
				diagnostics.DiagnosticCodeVariable0UsedBeforeItsDeclaration,
				expression.GetRange(),
				target.GetInternalName(), "", "",
			)
			return c.Module().Unreachable()
		}
		targetType = target.(program.VariableLikeElement).GetResolvedType()
		if target.HasDecorator(program.DecoratorFlagsUnsafe) {
			c.checkUnsafe(expression, nil)
		}

	case program.ElementKindPropertyPrototype:
		propertyPrototype := target.(*program.PropertyPrototype)
		propertyInstance := resolver.ResolveProperty(propertyPrototype, program.ReportModeReport)
		if propertyInstance == nil {
			return c.Module().Unreachable()
		}
		target = propertyInstance
		// fall-through to Property
		fallthrough

	case program.ElementKindProperty:
		// Handle both direct and fall-through from PropertyPrototype
		propertyInstance, ok := target.(*program.Property)
		if !ok {
			return c.Module().Unreachable()
		}
		if propertyInstance.IsField() {
			if _, pending := c.PendingElements[target]; pending {
				c.Error(
					diagnostics.DiagnosticCodeVariable0UsedBeforeItsDeclaration,
					expression.GetRange(),
					target.GetInternalName(), "", "",
				)
				return c.Module().Unreachable()
			}
		}
		setterInstance := propertyInstance.SetterInstance
		if setterInstance == nil {
			c.Error(
				diagnostics.DiagnosticCodeCannotAssignTo0BecauseItIsAConstantOrAReadOnlyProperty,
				expression.GetRange(),
				propertyInstance.GetInternalName(), "", "",
			)
			return c.Module().Unreachable()
		}
		targetType = setterInstance.Signature.ParameterTypes[0]
		if setterInstance.HasDecorator(program.DecoratorFlagsUnsafe) {
			c.checkUnsafe(expression, nil)
		}

	case program.ElementKindIndexSignature:
		indexSig := target.(*program.IndexSignature)
		parent := indexSig.GetParent()
		classInstance := parent.(*program.Class)
		isUnchecked := fl.Is(flow.FlowFlagUncheckedContext)
		indexedSet := classInstance.FindOverload(program.OperatorKindIndexedSet, isUnchecked)
		if indexedSet == nil {
			indexedGet := classInstance.FindOverload(program.OperatorKindIndexedGet, isUnchecked)
			if indexedGet == nil {
				c.Error(
					diagnostics.DiagnosticCodeIndexSignatureIsMissingInType0,
					expression.GetRange(),
					classInstance.GetInternalName(), "", "",
				)
			} else {
				c.Error(
					diagnostics.DiagnosticCodeIndexSignatureInType0OnlyPermitsReading,
					expression.GetRange(),
					classInstance.GetInternalName(), "", "",
				)
			}
			return c.Module().Unreachable()
		}
		parameterTypes := indexedSet.Signature.ParameterTypes
		targetType = parameterTypes[1] // 2nd parameter is the element
		if indexedSet.HasDecorator(program.DecoratorFlagsUnsafe) {
			c.checkUnsafe(expression, nil)
		}
		if !isUnchecked && c.Options().Pedantic {
			c.Pedantic(
				diagnostics.DiagnosticCodeIndexedAccessMayInvolveBoundsChecking,
				expression.GetRange(),
				"", "", "",
			)
		}

	default:
		c.Error(
			diagnostics.DiagnosticCodeCannotAssignTo0BecauseItIsAConstantOrAReadOnlyProperty,
			expression.GetRange(),
			target.GetInternalName(), "", "",
		)
		return c.Module().Unreachable()
	}

	// compile the value and do the assignment
	valueExpr := c.CompileExpression(valueExpression, targetType, ConstraintsNone)
	valueType := c.CurrentType
	if targetType.IsNullableReference() && fl.IsNonnull(valueExpr, valueType) {
		targetType = targetType.NonNullableType()
	}
	return c.makeAssignment(
		target,
		c.convertExpression(valueExpr, valueType, targetType, false, valueExpression),
		targetType,
		valueExpression,
		thisExpression,
		elementExpression,
		contextualType != types.TypeVoid,
	)
}

// --- Logical operators (short-circuit) ---

// compileLogicalAnd compiles a logical AND expression.
// Ported from: assemblyscript/src/compiler.ts case Token.Ampersand_Ampersand (lines 4569-4658).
func (c *Compiler) compileLogicalAnd(left, right ast.Node, expression *ast.BinaryExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	inheritedConstraints := constraints & ConstraintsMustWrap

	leftExpr := c.CompileExpression(left, contextualType.ExceptVoid(), inheritedConstraints)
	leftType := c.CurrentType

	rightFlow := fl.ForkThen(leftExpr, false, false)
	c.CurrentFlow = rightFlow

	var expr module.ExpressionRef

	// simplify if only interested in true or false
	if contextualType == types.TypeBool || contextualType == types.TypeVoid {
		leftExpr = c.makeIsTrueish(leftExpr, leftType, left)

		// shortcut if lhs is always false
		condKind := c.evaluateCondition(leftExpr)
		if condKind == flow.ConditionKindFalse {
			expr = leftExpr
			// RHS is not compiled
		} else {
			rightExpr := c.CompileExpression(right, leftType, inheritedConstraints)
			rightType := c.CurrentType
			rightExpr = c.makeIsTrueish(rightExpr, rightType, right)

			// simplify if lhs is always true
			if condKind == flow.ConditionKindTrue {
				expr = rightExpr
				fl.Inherit(rightFlow) // true && RHS -> RHS always executes
			} else {
				expr = mod.If(leftExpr, rightExpr, mod.I32(0))
				fl.MergeBranch(rightFlow) // LHS && RHS -> RHS conditionally executes
				fl.NoteThen(expr, rightFlow) // LHS && RHS == true -> RHS always executes
			}
		}
		c.CurrentFlow = fl
		c.CurrentType = types.TypeBool

	} else {
		rightExpr := c.CompileExpression(right, leftType, inheritedConstraints)
		rightType := c.CurrentType
		commonType := types.CommonType(leftType, rightType, contextualType, false)
		if commonType == nil {
			c.Error(
				diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
				expression.GetRange(),
				"&&", leftType.ToString(false), rightType.ToString(false),
			)
			c.CurrentType = contextualType
			return mod.Unreachable()
		}
		leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
		leftType = commonType

		// This is sometimes needed to make the left trivial
		leftPrecompExpr := mod.RunExpression(leftExpr, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
		if leftPrecompExpr != 0 {
			leftExpr = leftPrecompExpr
		}

		rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)

		condExpr := c.makeIsTrueish(leftExpr, commonType, left)
		condKind := c.evaluateCondition(condExpr)

		if condKind != flow.ConditionKindUnknown {
			// simplify if left is a constant
			if condKind == flow.ConditionKindTrue {
				expr = rightExpr
			} else {
				expr = leftExpr
			}
		} else if trivialCopy := mod.TryCopyTrivialExpression(leftExpr); trivialCopy != 0 {
			// simplify if copying left is trivial
			expr = mod.If(condExpr, rightExpr, trivialCopy)
		} else {
			// if not possible, tee left to a temp
			tempLocal := fl.GetTempLocal(leftType)
			tempIndex := tempLocal.FlowIndex()
			if !fl.CanOverflow(leftExpr, leftType) {
				fl.SetLocalFlag(tempIndex, flow.LocalFlagWrapped)
			}
			if fl.IsNonnull(leftExpr, leftType) {
				fl.SetLocalFlag(tempIndex, flow.LocalFlagNonNull)
			}
			expr = mod.If(
				c.makeIsTrueish(mod.LocalTee(tempIndex, leftExpr, leftType.IsManaged(), leftType.ToRef()), leftType, left),
				rightExpr,
				mod.LocalGet(tempIndex, leftType.ToRef()),
			)
		}
		fl.MergeBranch(rightFlow)     // LHS && RHS -> RHS conditionally executes
		fl.NoteThen(expr, rightFlow)  // LHS && RHS == true -> RHS always executes
		c.CurrentFlow = fl
		c.CurrentType = commonType
	}
	return expr
}

// compileLogicalOr compiles a logical OR expression.
// Ported from: assemblyscript/src/compiler.ts case Token.Bar_Bar (lines 4660-4754).
func (c *Compiler) compileLogicalOr(left, right ast.Node, expression *ast.BinaryExpression, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	inheritedConstraints := constraints & ConstraintsMustWrap

	leftExpr := c.CompileExpression(left, contextualType.ExceptVoid(), inheritedConstraints)
	leftType := c.CurrentType

	rightFlow := fl.ForkElse(leftExpr)
	c.CurrentFlow = rightFlow

	var expr module.ExpressionRef

	// simplify if only interested in true or false
	if contextualType == types.TypeBool || contextualType == types.TypeVoid {
		leftExpr = c.makeIsTrueish(leftExpr, leftType, left)

		// shortcut if lhs is always true
		condKind := c.evaluateCondition(leftExpr)
		if condKind == flow.ConditionKindTrue {
			expr = leftExpr
			// RHS is not compiled
		} else {
			rightExpr := c.CompileExpression(right, leftType, inheritedConstraints)
			rightType := c.CurrentType
			rightExpr = c.makeIsTrueish(rightExpr, rightType, right)

			// simplify if lhs is always false
			if condKind == flow.ConditionKindFalse {
				expr = rightExpr
				fl.Inherit(rightFlow) // false || RHS -> RHS always executes
			} else {
				expr = mod.If(leftExpr, mod.I32(1), rightExpr)
				fl.MergeBranch(rightFlow) // LHS || RHS -> RHS conditionally executes
				fl.NoteElse(expr, rightFlow) // LHS || RHS == false -> RHS always executes
			}
		}
		c.CurrentFlow = fl
		c.CurrentType = types.TypeBool

	} else {
		rightExpr := c.CompileExpression(right, leftType, inheritedConstraints)
		rightType := c.CurrentType
		commonType := types.CommonType(leftType, rightType, contextualType, false)
		if commonType == nil {
			c.Error(
				diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
				expression.GetRange(),
				"||", leftType.ToString(false), rightType.ToString(false),
			)
			c.CurrentType = contextualType
			return mod.Unreachable()
		}
		possiblyNull := leftType.Is(types.TypeFlagNullable) && rightType.Is(types.TypeFlagNullable)
		leftExpr = c.convertExpression(leftExpr, leftType, commonType, false, left)
		leftType = commonType

		// This is sometimes needed to make the left trivial
		leftPrecompExpr := mod.RunExpression(leftExpr, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
		if leftPrecompExpr != 0 {
			leftExpr = leftPrecompExpr
		}

		rightExpr = c.convertExpression(rightExpr, rightType, commonType, false, right)

		condExpr := c.makeIsTrueish(leftExpr, commonType, left)
		condKind := c.evaluateCondition(condExpr)

		if condKind != flow.ConditionKindUnknown {
			// simplify if left is a constant
			if condKind == flow.ConditionKindTrue {
				expr = leftExpr
			} else {
				expr = rightExpr
			}
		} else if trivialCopy := mod.TryCopyTrivialExpression(leftExpr); trivialCopy != 0 {
			// otherwise, simplify if copying left is trivial
			expr = mod.If(condExpr, trivialCopy, rightExpr)
		} else {
			// if not possible, tee left to a temp. local
			temp := fl.GetTempLocal(leftType)
			tempIndex := temp.FlowIndex()
			if !fl.CanOverflow(leftExpr, leftType) {
				fl.SetLocalFlag(tempIndex, flow.LocalFlagWrapped)
			}
			if fl.IsNonnull(leftExpr, leftType) {
				fl.SetLocalFlag(tempIndex, flow.LocalFlagNonNull)
			}
			expr = mod.If(
				c.makeIsTrueish(mod.LocalTee(tempIndex, leftExpr, leftType.IsManaged(), leftType.ToRef()), leftType, left),
				mod.LocalGet(tempIndex, leftType.ToRef()),
				rightExpr,
			)
		}
		fl.MergeBranch(rightFlow)     // LHS || RHS -> RHS conditionally executes
		fl.NoteElse(expr, rightFlow)  // LHS || RHS == false -> RHS always executes
		c.CurrentFlow = fl
		if possiblyNull {
			c.CurrentType = commonType
		} else {
			c.CurrentType = commonType.NonNullableType()
		}
	}
	return expr
}

// --- Type conversion ---

// convertExpression converts an expression from one type to another.
// Ported from: assemblyscript/src/compiler.ts convertExpression (lines 3477-3510).
// convertExpression converts an expression from one type to another.
// Ported from: assemblyscript/src/compiler.ts convertExpression (lines 3545-3793).
func (c *Compiler) convertExpression(expr module.ExpressionRef, fromType, toType *types.Type, explicit bool, reportNode ast.Node) module.ExpressionRef {
	mod := c.Module()

	// void handling
	if fromType.Kind == types.TypeKindVoid {
		if toType.Kind == types.TypeKindVoid {
			return expr
		}
		c.Error(
			diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
			reportNode.GetRange(),
			fromType.String(), toType.String(), "",
		)
		return mod.Unreachable()
	}
	if toType.Kind == types.TypeKindVoid {
		return mod.Drop(expr)
	}

	// reference involved
	if fromType.IsReference() || toType.IsReference() {
		if c.CurrentFlow.IsNonnull(expr, fromType) {
			fromType = fromType.NonNullableType()
		} else if explicit && fromType.IsNullableReference() && !toType.IsNullableReference() {
			if !c.Options().NoAssert {
				expr = c.makeRuntimeNonNullCheck(expr, fromType, reportNode)
			}
			fromType = fromType.NonNullableType()
		}
		if fromType.IsAssignableTo(toType, false) {
			c.CurrentType = toType
			return expr
		}
		if explicit && toType.NonNullableType().IsAssignableTo(fromType, false) {
			// downcast
			if toType.IsExternalReference() {
				c.Error(
					diagnostics.DiagnosticCodeNotImplemented0,
					reportNode.GetRange(),
					"ref.cast", "", "",
				)
				c.CurrentType = toType
				return mod.Unreachable()
			}
			if !c.Options().NoAssert {
				expr = c.makeRuntimeDowncastCheck(expr, fromType, toType, reportNode)
			}
			c.CurrentType = toType
			return expr
		}
		c.Error(
			diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
			reportNode.GetRange(),
			fromType.String(), toType.String(), "",
		)
		c.CurrentType = toType
		return mod.Unreachable()
	}

	// early return for same types
	if toType.Kind == fromType.Kind {
		c.CurrentType = toType
		return expr
	}

	// v128 to any / any to v128 (except v128 to bool)
	if !toType.IsBooleanValue() && (toType.IsVectorValue() || fromType.IsVectorValue()) {
		c.Error(
			diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
			reportNode.GetRange(),
			fromType.String(), toType.String(), "",
		)
		return mod.Unreachable()
	}

	if !fromType.IsAssignableTo(toType, false) {
		if !explicit {
			c.Error(
				diagnostics.DiagnosticCodeConversionFromType0To1RequiresAnExplicitCast,
				reportNode.GetRange(),
				fromType.String(), toType.String(), "",
			)
		}
	}

	if fromType.IsFloatValue() {
		// float to float
		if toType.IsFloatValue() {
			if fromType.Kind == types.TypeKindF32 {
				if toType.Kind == types.TypeKindF64 {
					expr = mod.Unary(module.UnaryOpPromoteF32ToF64, expr)
				}
			} else if toType.Kind == types.TypeKindF32 {
				expr = mod.Unary(module.UnaryOpDemoteF64ToF32, expr)
			}
		} else if toType.IsIntegerValue() {
			// float to int
			saturating := c.Options().HasFeature(common.FeatureNontrappingF2I)
			if fromType.Kind == types.TypeKindF32 {
				if toType.IsBooleanValue() {
					expr = c.makeIsTrueish(expr, types.TypeF32, reportNode)
				} else if toType.IsSignedIntegerValue() {
					if toType.IsLongIntegerValue() {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF32ToI64, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF32ToI64, expr)
						}
					} else {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF32ToI32, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF32ToI32, expr)
						}
					}
				} else {
					if toType.IsLongIntegerValue() {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF32ToU64, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF32ToU64, expr)
						}
					} else {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF32ToU32, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF32ToU32, expr)
						}
					}
				}
			} else {
				// f64 to int
				if toType.IsBooleanValue() {
					expr = c.makeIsTrueish(expr, types.TypeF64, reportNode)
				} else if toType.IsSignedIntegerValue() {
					if toType.IsLongIntegerValue() {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF64ToI64, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF64ToI64, expr)
						}
					} else {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF64ToI32, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF64ToI32, expr)
						}
					}
				} else {
					if toType.IsLongIntegerValue() {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF64ToU64, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF64ToU64, expr)
						}
					} else {
						if saturating {
							expr = mod.Unary(module.UnaryOpTruncSatF64ToU32, expr)
						} else {
							expr = mod.Unary(module.UnaryOpTruncF64ToU32, expr)
						}
					}
				}
			}
		} else {
			// float to void
			expr = mod.Drop(expr)
		}
	} else if fromType.IsIntegerValue() && toType.IsFloatValue() {
		// int to float: clear extra bits first
		expr = c.ensureSmallIntegerWrap(expr, fromType)
		var op module.Op
		if toType.Kind == types.TypeKindF32 {
			if fromType.IsLongIntegerValue() {
				if fromType.IsSignedIntegerValue() {
					op = module.UnaryOpConvertI64ToF32
				} else {
					op = module.UnaryOpConvertU64ToF32
				}
			} else {
				if fromType.IsSignedIntegerValue() {
					op = module.UnaryOpConvertI32ToF32
				} else {
					op = module.UnaryOpConvertU32ToF32
				}
			}
		} else {
			if fromType.IsLongIntegerValue() {
				if fromType.IsSignedIntegerValue() {
					op = module.UnaryOpConvertI64ToF64
				} else {
					op = module.UnaryOpConvertU64ToF64
				}
			} else {
				if fromType.IsSignedIntegerValue() {
					op = module.UnaryOpConvertI32ToF64
				} else {
					op = module.UnaryOpConvertU32ToF64
				}
			}
		}
		expr = mod.Unary(op, expr)
	} else if fromType == types.TypeV128 && toType.IsBooleanValue() {
		// v128 to bool
		expr = c.makeIsTrueish(expr, types.TypeV128, reportNode)
	} else {
		// int to int
		if fromType.IsLongIntegerValue() {
			// i64 to ...
			if toType.IsBooleanValue() {
				expr = mod.Binary(module.BinaryOpNeI64, expr, mod.I64(0))
			} else if !toType.IsLongIntegerValue() {
				expr = mod.Unary(module.UnaryOpWrapI64ToI32, expr)
			}
		} else if toType.IsLongIntegerValue() {
			// i32 or smaller to i64
			if fromType.IsSignedIntegerValue() {
				expr = mod.Unary(module.UnaryOpExtendI32ToI64, c.ensureSmallIntegerWrap(expr, fromType))
			} else {
				expr = mod.Unary(module.UnaryOpExtendU32ToU64, c.ensureSmallIntegerWrap(expr, fromType))
			}
		} else {
			// i32 to i32
			if fromType.IsShortIntegerValue() {
				// small i32 to larger i32
				if fromType.Size < toType.Size {
					expr = c.ensureSmallIntegerWrap(expr, fromType)
				}
			} else {
				// same size
				if !explicit && !c.Options().IsWasm64() && fromType.IsVaryingIntegerValue() && !toType.IsVaryingIntegerValue() {
					c.Warning(
						diagnostics.DiagnosticCodeConversionFromType0To1WillRequireAnExplicitCastWhenSwitchingBetween3264Bit,
						reportNode.GetRange(),
						fromType.String(), toType.String(), "",
					)
				}
			}
		}
	}

	c.CurrentType = toType
	return expr
}

// --- Utility functions ---

// writeI32 writes a little-endian 32-bit integer to a byte slice at the given offset.
func writeI32(buf []byte, offset int, value int32) {
	buf[offset] = byte(value)
	buf[offset+1] = byte(value >> 8)
	buf[offset+2] = byte(value >> 16)
	buf[offset+3] = byte(value >> 24)
}

// Ensure unused imports are referenced
var _ = math.MaxFloat64
