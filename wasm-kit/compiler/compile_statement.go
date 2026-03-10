// Ported from: assemblyscript/src/compiler.ts compileStatement (lines 2234-2334),
// and all individual statement compilation methods (lines 2336-3429).
package compiler

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// CompileStatement compiles a single statement.
// Ported from: assemblyscript/src/compiler.ts compileStatement (lines 2234-2334).
func (c *Compiler) CompileStatement(statement ast.Node) module.ExpressionRef {
	mod := c.Module()
	var stmt module.ExpressionRef

	switch statement.GetKind() {
	case ast.NodeKindBlock:
		stmt = c.compileBlockStatement(statement.(*ast.BlockStatement))
	case ast.NodeKindBreak:
		stmt = c.compileBreakStatement(statement.(*ast.BreakStatement))
	case ast.NodeKindContinue:
		stmt = c.compileContinueStatement(statement.(*ast.ContinueStatement))
	case ast.NodeKindDo:
		stmt = c.compileDoStatement(statement.(*ast.DoStatement))
	case ast.NodeKindEmpty:
		stmt = mod.Nop()
	case ast.NodeKindExpression:
		stmt = c.compileExpressionStatement(statement.(*ast.ExpressionStatement))
	case ast.NodeKindFor:
		stmt = c.compileForStatement(statement.(*ast.ForStatement))
	case ast.NodeKindForOf:
		stmt = c.compileForOfStatement(statement.(*ast.ForOfStatement))
	case ast.NodeKindIf:
		stmt = c.compileIfStatement(statement.(*ast.IfStatement))
	case ast.NodeKindReturn:
		stmt = c.compileReturnStatement(statement.(*ast.ReturnStatement))
	case ast.NodeKindSwitch:
		stmt = c.compileSwitchStatement(statement.(*ast.SwitchStatement))
	case ast.NodeKindThrow:
		stmt = c.compileThrowStatement(statement.(*ast.ThrowStatement))
	case ast.NodeKindTry:
		stmt = c.compileTryStatement(statement.(*ast.TryStatement))
	case ast.NodeKindVariable:
		stmt = c.compileVariableStatement(statement.(*ast.VariableStatement))
	case ast.NodeKindVoid:
		stmt = c.compileVoidStatement(statement.(*ast.VoidStatement))
	case ast.NodeKindWhile:
		stmt = c.compileWhileStatement(statement.(*ast.WhileStatement))
	default:
		stmt = mod.Unreachable()
	}

	// Add debug location if source maps are enabled
	if c.Options().SourceMap {
		c.addDebugLocation(stmt, statement.GetRange())
	}
	return stmt
}

// CompileStatements compiles a sequence of statements, appending to the given slice.
// Ported from: assemblyscript/src/compiler.ts compileStatements (lines 2336-2352).
func (c *Compiler) CompileStatements(statements []ast.Node, stmts []module.ExpressionRef) []module.ExpressionRef {
	fl := c.CurrentFlow
	for _, statement := range statements {
		stmt := c.CompileStatement(statement)
		switch module.GetExpressionId(stmt) {
		case module.ExpressionIdBlock:
			if module.GetBlockName(stmt) == "" {
				// Flatten unnamed blocks into parent
				for j := module.Index(0); j < module.GetBlockChildCount(stmt); j++ {
					stmts = append(stmts, module.GetBlockChildAt(stmt, j))
				}
			} else {
				stmts = append(stmts, stmt)
			}
		case module.ExpressionIdNop:
			// skip nops
		default:
			stmts = append(stmts, stmt)
		}
		if fl.IsAny(flow.FlowFlagTerminates | flow.FlowFlagBreaks) {
			break
		}
	}
	return stmts
}

// compileBlockStatement compiles a block statement.
// Ported from: assemblyscript/src/compiler.ts compileBlockStatement (lines 2354-2366).
func (c *Compiler) compileBlockStatement(statement *ast.BlockStatement) module.ExpressionRef {
	outerFlow := c.CurrentFlow
	innerFlow := outerFlow.Fork(false, false)
	c.CurrentFlow = innerFlow

	stmts := make([]module.ExpressionRef, 0, len(statement.Statements))
	stmts = c.CompileStatements(statement.Statements, stmts)

	outerFlow.Inherit(innerFlow)
	c.CurrentFlow = outerFlow
	return c.Module().Flatten(stmts, module.TypeRefNone)
}

// compileBreakStatement compiles a break statement.
// Ported from: assemblyscript/src/compiler.ts compileBreakStatement (lines 2386-2410).
func (c *Compiler) compileBreakStatement(statement *ast.BreakStatement) module.ExpressionRef {
	mod := c.Module()
	labelNode := statement.Label
	if labelNode != nil {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			labelNode.GetRange(),
			"Break label", "", "",
		)
		return mod.Unreachable()
	}
	fl := c.CurrentFlow
	breakLabel := fl.BreakLabel
	if breakLabel == "" {
		c.Error(
			diagnostics.DiagnosticCodeABreakStatementCanOnlyBeUsedWithinAnEnclosingIterationOrSwitchStatement,
			statement.GetRange(),
			"", "", "",
		)
		return mod.Unreachable()
	}
	fl.SetFlag(flow.FlowFlagBreaks)
	return mod.Br(breakLabel, 0, 0)
}

// compileContinueStatement compiles a continue statement.
// Ported from: assemblyscript/src/compiler.ts compileContinueStatement (lines 2412-2437).
func (c *Compiler) compileContinueStatement(statement *ast.ContinueStatement) module.ExpressionRef {
	mod := c.Module()
	label := statement.Label
	if label != nil {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			label.GetRange(),
			"Continue label", "", "",
		)
		return mod.Unreachable()
	}
	// Check if 'continue' is allowed here
	fl := c.CurrentFlow
	continueLabel := fl.ContinueLabel
	if continueLabel == "" {
		c.Error(
			diagnostics.DiagnosticCodeAContinueStatementCanOnlyBeUsedWithinAnEnclosingIterationStatement,
			statement.GetRange(),
			"", "", "",
		)
		return mod.Unreachable()
	}
	fl.SetFlag(flow.FlowFlagContinues | flow.FlowFlagTerminates)
	return mod.Br(continueLabel, 0, 0)
}

// compileDoStatement compiles a do-while statement.
// Ported from: assemblyscript/src/compiler.ts compileDoStatement (lines 2439-2549).
func (c *Compiler) compileDoStatement(statement *ast.DoStatement) module.ExpressionRef {
	return c.doCompileDoStatement(statement)
}

// doCompileDoStatement is the inner implementation of compileDoStatement,
// called recursively when loop convergence requires recompilation.
// Ported from: assemblyscript/src/compiler.ts doCompileDoStatement (lines 2446-2549).
func (c *Compiler) doCompileDoStatement(statement *ast.DoStatement) module.ExpressionRef {
	mod := c.Module()
	outerFlow := c.CurrentFlow
	numLocalsBefore := len(outerFlow.TargetFunction.FlowLocalsByIndex())

	// (block $break
	//  (loop $loop
	//   (?block $continue
	//    (body)
	//   )
	//   (br_if $loop (condition))
	//  )
	// )

	// Cases of interest:
	// * If the body never falls through or continues, the condition never executes
	// * If the condition is always true and body never breaks, overall flow terminates
	// * If the body terminates with a continue, condition is still reached

	// Compile the body (always executes)
	fl := outerFlow.Fork(true, true)
	label := fl.PushControlFlowLabel()
	breakLabel := fmt.Sprintf("do-break|%d", label)
	fl.BreakLabel = breakLabel
	continueLabel := fmt.Sprintf("do-continue|%d", label)
	fl.ContinueLabel = continueLabel
	loopLabel := fmt.Sprintf("do-loop|%d", label)
	c.CurrentFlow = fl
	bodyStmts := make([]module.ExpressionRef, 0)
	body := statement.Body
	if body.GetKind() == ast.NodeKindBlock {
		bodyStmts = c.CompileStatements(body.(*ast.BlockStatement).Statements, bodyStmts)
	} else {
		bodyStmts = append(bodyStmts, c.CompileStatement(body))
	}
	fl.PopControlFlowLabel(label)

	possiblyContinues := fl.IsAny(flow.FlowFlagContinues | flow.FlowFlagConditionallyContinues)
	possiblyBreaks := fl.IsAny(flow.FlowFlagBreaks | flow.FlowFlagConditionallyBreaks)
	possiblyFallsThrough := !fl.IsAny(flow.FlowFlagTerminates | flow.FlowFlagBreaks)

	// Shortcut if the condition is never reached
	if !possiblyFallsThrough && !possiblyContinues {
		bodyStmts = append(bodyStmts, mod.Unreachable())
		outerFlow.Inherit(fl)

		// If the body also never breaks, the overall flow terminates
		if !possiblyBreaks {
			outerFlow.SetFlag(flow.FlowFlagTerminates)
		}

	// Otherwise compile and evaluate the condition (from here on always executes)
	} else {
		condExpr := c.CompileExpression(statement.Condition, types.TypeBool, ConstraintsConvImplicit)
		condExprTrueish := c.makeIsTrueish(condExpr, c.CurrentType, statement.Condition)
		condKind := c.evaluateCondition(condExprTrueish)

		// Detect if local flags are incompatible before and after looping, and
		// if so recompile by unifying local flags between iterations. Note that
		// this may be necessary multiple times where locals depend on each other.
		possiblyLoops := condKind != flow.ConditionKindFalse && (possiblyContinues || possiblyFallsThrough)
		if possiblyLoops && outerFlow.ResetIfNeedsRecompile(fl.ForkThen(condExpr, false, false), numLocalsBefore) {
			c.CurrentFlow = outerFlow
			return c.doCompileDoStatement(statement)
		}

		if possiblyContinues {
			bodyStmts[0] = mod.Block(continueLabel, bodyStmts, module.TypeRefNone)
			bodyStmts = bodyStmts[:1]
			fl.UnsetFlag(flow.FlowFlagTerminates) // Continue breaks to condition
		}
		bodyStmts = append(bodyStmts,
			mod.Br(loopLabel, condExprTrueish, 0),
		)
		outerFlow.Inherit(fl)

		// Terminate if the condition is always true and body never breaks
		if condKind == flow.ConditionKindTrue && !possiblyBreaks {
			outerFlow.SetFlag(flow.FlowFlagTerminates)
		}
	}

	// Finalize and leave everything else to the optimizer
	c.CurrentFlow = outerFlow
	expr := mod.Loop(loopLabel,
		mod.Flatten(bodyStmts, module.TypeRefNone),
	)
	if possiblyBreaks {
		expr = mod.Block(breakLabel, []module.ExpressionRef{expr}, module.TypeRefNone)
	}
	if outerFlow.Is(flow.FlowFlagTerminates) {
		expr = mod.Block("", []module.ExpressionRef{expr, mod.Unreachable()}, module.TypeRefNone)
	}
	return expr
}

// compileExpressionStatement compiles an expression statement.
// Ported from: assemblyscript/src/compiler.ts compileExpressionStatement (lines 2557-2561).
func (c *Compiler) compileExpressionStatement(statement *ast.ExpressionStatement) module.ExpressionRef {
	return c.CompileExpression(statement.Expression, types.TypeVoid, ConstraintsConvImplicit)
}

// compileForStatement compiles a for statement.
// Ported from: assemblyscript/src/compiler.ts compileForStatement (lines 2563-2711).
func (c *Compiler) compileForStatement(statement *ast.ForStatement) module.ExpressionRef {
	return c.doCompileForStatement(statement)
}

// doCompileForStatement is the inner implementation of compileForStatement,
// called recursively when loop convergence requires recompilation.
// Ported from: assemblyscript/src/compiler.ts doCompileForStatement (lines 2570-2711).
func (c *Compiler) doCompileForStatement(statement *ast.ForStatement) module.ExpressionRef {
	mod := c.Module()
	outerFlow := c.CurrentFlow
	numLocalsBefore := len(outerFlow.TargetFunction.FlowLocalsByIndex())

	// Compile initializer if present. The initializer might introduce scoped
	// locals bound to the for statement, so create a new flow early.
	fl := outerFlow.Fork(false, false)
	c.CurrentFlow = fl
	stmts := make([]module.ExpressionRef, 0)
	initializer := statement.Initializer
	if initializer != nil {
		stmts = append(stmts, c.CompileStatement(initializer))
	}

	// Precompute the condition if present, or default to `true`
	var condExpr module.ExpressionRef
	var condExprTrueish module.ExpressionRef
	var condKind flow.ConditionKind
	condition := statement.Condition
	if condition != nil {
		condExpr = c.CompileExpression(condition, types.TypeBool, ConstraintsConvImplicit)
		condExprTrueish = c.makeIsTrueish(condExpr, c.CurrentType, condition)
		condKind = c.evaluateCondition(condExprTrueish)

		// Shortcut if condition is always false (body never executes)
		if condKind == flow.ConditionKindFalse {
			stmts = append(stmts, mod.Drop(condExprTrueish))
			outerFlow.Inherit(fl)
			c.CurrentFlow = outerFlow
			return mod.Flatten(stmts, module.TypeRefNone)
		}
	} else {
		condExpr = mod.I32(1)
		condExprTrueish = condExpr
		condKind = flow.ConditionKindTrue
	}
	// From here on condition is either true or unknown

	// Compile the body assuming the condition turned out true
	bodyFlow := fl.ForkThen(condExpr, true, true)
	label := bodyFlow.PushControlFlowLabel()
	breakLabel := fmt.Sprintf("for-break%d", label)
	bodyFlow.BreakLabel = breakLabel
	continueLabel := fmt.Sprintf("for-continue|%d", label)
	bodyFlow.ContinueLabel = continueLabel
	loopLabel := fmt.Sprintf("for-loop|%d", label)
	c.CurrentFlow = bodyFlow
	bodyStmts := make([]module.ExpressionRef, 0)
	body := statement.Body
	if body.GetKind() == ast.NodeKindBlock {
		bodyStmts = c.CompileStatements(body.(*ast.BlockStatement).Statements, bodyStmts)
	} else {
		bodyStmts = append(bodyStmts, c.CompileStatement(body))
	}
	bodyFlow.PopControlFlowLabel(label)
	bodyFlow.BreakLabel = ""
	bodyFlow.ContinueLabel = ""

	possiblyFallsThrough := !bodyFlow.IsAny(flow.FlowFlagTerminates | flow.FlowFlagBreaks)
	possiblyContinues := bodyFlow.IsAny(flow.FlowFlagContinues | flow.FlowFlagConditionallyContinues)
	possiblyBreaks := bodyFlow.IsAny(flow.FlowFlagBreaks | flow.FlowFlagConditionallyBreaks)

	if possiblyContinues {
		bodyStmts[0] = mod.Block(continueLabel, bodyStmts, module.TypeRefNone)
		bodyStmts = bodyStmts[:1]
	}

	if condKind == flow.ConditionKindTrue {
		// Body executes at least once
		fl.Inherit(bodyFlow)
	} else {
		// Otherwise executes conditionally
		fl.MergeBranch(bodyFlow)
	}

	// Compile the incrementor if it possibly executes
	possiblyLoops := possiblyContinues || possiblyFallsThrough
	if possiblyLoops {
		incrementor := statement.Incrementor
		if incrementor != nil {
			c.CurrentFlow = fl
			bodyStmts = append(bodyStmts,
				c.CompileExpression(incrementor, types.TypeVoid, ConstraintsConvImplicit|ConstraintsWillDrop),
			)
		}
		bodyStmts = append(bodyStmts, mod.Br(loopLabel, 0, 0))

		// Detect if local flags are incompatible before and after looping, and if
		// so recompile by unifying local flags between iterations. Note that this
		// may be necessary multiple times where locals depend on each other.
		if outerFlow.ResetIfNeedsRecompile(bodyFlow.ForkThen(condExpr, false, false), numLocalsBefore) {
			c.CurrentFlow = outerFlow
			return c.doCompileForStatement(statement)
		}
	}

	// Finalize
	outerFlow.Inherit(fl)
	c.CurrentFlow = outerFlow
	expr := mod.If(condExprTrueish,
		mod.Flatten(bodyStmts, module.TypeRefNone),
		0,
	)
	if possiblyLoops {
		expr = mod.Loop(loopLabel, expr)
	}
	if possiblyBreaks {
		expr = mod.Block(breakLabel, []module.ExpressionRef{expr}, module.TypeRefNone)
	}
	stmts = append(stmts, expr)
	if outerFlow.Is(flow.FlowFlagTerminates) {
		stmts = append(stmts, mod.Unreachable())
	}
	return mod.Flatten(stmts, module.TypeRefNone)
}

// compileForOfStatement compiles a for-of statement.
// Ported from: assemblyscript/src/compiler.ts compileForOfStatement (lines 2632-2703).
func (c *Compiler) compileForOfStatement(statement *ast.ForOfStatement) module.ExpressionRef {
	// For-of requires iterator protocol support which depends on builtins.
	// Stub: error for now.
	c.Error(
		diagnostics.DiagnosticCodeNotImplemented0,
		statement.GetRange(),
		"for-of loops", "", "",
	)
	return c.Module().Unreachable()
}

// compileIfStatement compiles an if statement.
// Ported from: assemblyscript/src/compiler.ts compileIfStatement (lines 2705-2789).
func (c *Compiler) compileIfStatement(statement *ast.IfStatement) module.ExpressionRef {
	mod := c.Module()
	ifTrue := statement.IfTrue
	ifFalse := statement.IfFalse

	// Precompute the condition (always executes)
	condExpr := c.CompileExpression(statement.Condition, types.TypeBool, ConstraintsConvImplicit)
	condExprTrueish := c.makeIsTrueish(condExpr, c.CurrentType, statement.Condition)
	condKind := c.evaluateCondition(condExprTrueish)

	// Shortcut if the condition is constant
	switch condKind {
	case flow.ConditionKindTrue:
		return mod.Block("", []module.ExpressionRef{
			mod.Drop(condExprTrueish),
			c.CompileStatement(ifTrue),
		}, module.TypeRefNone)
	case flow.ConditionKindFalse:
		if ifFalse != nil {
			return mod.Block("", []module.ExpressionRef{
				mod.Drop(condExprTrueish),
				c.CompileStatement(ifFalse),
			}, module.TypeRefNone)
		}
		return mod.Drop(condExprTrueish)
	}

	// From here on condition is always unknown
	fl := c.CurrentFlow

	// Compile ifTrue assuming the condition turned out true
	thenFlow := fl.ForkThen(condExpr, false, false)
	c.CurrentFlow = thenFlow
	thenStmt := c.CompileStatement(ifTrue)
	c.CurrentFlow = fl

	// Compile ifFalse assuming the condition turned out false, if present
	elseFlow := fl.ForkElse(condExpr)
	if ifFalse != nil {
		c.CurrentFlow = elseFlow
		elseStmt := c.CompileStatement(ifFalse)
		fl.InheritAlternatives(thenFlow, elseFlow) // terminates if both do
		c.CurrentFlow = fl
		return mod.If(condExprTrueish, thenStmt, elseStmt)
	}

	// No else branch
	if thenFlow.IsAny(flow.FlowFlagTerminates | flow.FlowFlagBreaks) {
		// Only getting past if condition was false (acts like else)
		fl.Inherit(elseFlow)
		fl.MergeSideEffects(thenFlow)
	} else {
		// Otherwise getting past conditionally
		fl.InheritAlternatives(thenFlow, elseFlow)
	}
	c.CurrentFlow = fl
	return mod.If(condExprTrueish, thenStmt, 0)
}

// compileReturnStatement compiles a return statement.
// Ported from: assemblyscript/src/compiler.ts compileReturnStatement (lines 2815-2862).
func (c *Compiler) compileReturnStatement(statement *ast.ReturnStatement) module.ExpressionRef {
	mod := c.Module()
	var expr module.ExpressionRef
	fl := c.CurrentFlow
	returnType := fl.ReturnType()

	valueExpression := statement.Value
	if valueExpression != nil {
		constraints := ConstraintsConvImplicit
		if fl.SourceFunction().Is(uint32(common.CommonFlagsModuleExport)) {
			constraints |= ConstraintsMustWrap
		}

		expr = c.CompileExpression(valueExpression, returnType, constraints)
		if !fl.CanOverflow(expr, returnType) {
			fl.SetFlag(flow.FlowFlagReturnsWrapped)
		}
		if fl.IsNonnull(expr, returnType) {
			fl.SetFlag(flow.FlowFlagReturnsNonNull)
		}
		if fl.SourceFunction().Is(uint32(common.CommonFlagsConstructor)) && valueExpression.GetKind() != ast.NodeKindThis {
			fl.SetFlag(flow.FlowFlagMayReturnNonThis)
		}
	} else if returnType != types.TypeVoid {
		c.Error(
			diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
			statement.GetRange(),
			"void", returnType.String(), "",
		)
		c.CurrentType = returnType
		return mod.Unreachable()
	}

	// Remember that this flow returns
	fl.SetFlag(flow.FlowFlagReturns | flow.FlowFlagTerminates)

	// Handle inline return
	if fl.IsInline() {
		inlineReturnLabel := fl.InlineReturnLabel
		if expr != 0 {
			if c.CurrentType == types.TypeVoid {
				return mod.Block("", []module.ExpressionRef{expr, mod.Br(inlineReturnLabel, 0, 0)}, module.TypeRefNone)
			}
			return mod.Br(inlineReturnLabel, 0, expr)
		}
		return mod.Br(inlineReturnLabel, 0, 0)
	}

	// Otherwise emit a normal return
	if expr != 0 {
		if c.CurrentType == types.TypeVoid {
			return mod.Block("", []module.ExpressionRef{expr, mod.Return(0)}, module.TypeRefNone)
		}
		return mod.Return(expr)
	}
	return mod.Return(0)
}

// compileSwitchStatement compiles a switch statement.
// Ported from: assemblyscript/src/compiler.ts compileSwitchStatement (lines 2864-2986).
func (c *Compiler) compileSwitchStatement(statement *ast.SwitchStatement) module.ExpressionRef {
	mod := c.Module()
	outerFlow := c.CurrentFlow
	cases := statement.Cases
	numCases := len(cases)

	// Compile the condition (always executes)
	condExpr := c.CompileExpression(statement.Condition, types.TypeAuto, ConstraintsNone)
	condType := c.CurrentType

	// Shortcut if there are no cases
	if numCases == 0 {
		return mod.Drop(condExpr)
	}

	// Assign the condition to a temporary local as we compare it multiple times
	tempLocal := outerFlow.GetTempLocal(condType)
	tempIndex := tempLocal.FlowIndex()
	breaks := make([]module.ExpressionRef, 0, 1+numCases+1)
	breaks = append(breaks, mod.LocalSet(tempIndex, condExpr, condType.IsManaged()))

	// Make one br_if per labeled case
	defaultIndex := -1
	label := outerFlow.PushControlFlowLabel()
	for i, sc := range cases {
		if sc.IsDefault() {
			defaultIndex = i
			continue
		}
		leftExpr := mod.LocalGet(tempIndex, condType.ToRef())
		rightExpr := c.CompileExpression(sc.Label, condType, ConstraintsConvImplicit)
		rightType := c.CurrentType
		eqExpr := c.compileCommutativeCompareBinaryExpressionFromParts(
			tokenizer.TokenEqualsEquals,
			statement.Condition, leftExpr, condType,
			sc.Label, rightExpr, rightType,
			condType,
			statement,
		)
		breaks = append(breaks, mod.Br(fmt.Sprintf("case%d|%d", i, label), eqExpr, 0))
	}

	// If there is a default case, break to it, otherwise break out of the switch
	breakLabel := fmt.Sprintf("break|%d", label)
	if defaultIndex >= 0 {
		breaks = append(breaks, mod.Br(fmt.Sprintf("case%d|%d", defaultIndex, label), 0, 0))
	} else {
		breaks = append(breaks, mod.Br(breakLabel, 0, 0))
	}

	// Nest the case blocks in order, to be targeted by the br_if sequence
	currentBlock := mod.Block(fmt.Sprintf("case0|%d", label), breaks, module.TypeRefNone)
	var fallThroughFlow *flow.Flow
	var breakingFlowAlternatives *flow.Flow

	for i, sc := range cases {
		// Can get here by matching the case or possibly by fall-through
		innerFlow := outerFlow.Fork(true, false)
		if fallThroughFlow != nil {
			innerFlow.MergeBranch(fallThroughFlow)
		}
		c.CurrentFlow = innerFlow
		innerFlow.BreakLabel = breakLabel

		isLast := i == numCases-1
		var nextLabel string
		if isLast {
			nextLabel = breakLabel
		} else {
			nextLabel = fmt.Sprintf("case%d|%d", i+1, label)
		}

		stmts := make([]module.ExpressionRef, 0, 1+len(sc.Statements))
		stmts = append(stmts, currentBlock)

		possiblyFallsThrough := true
		for _, statement := range sc.Statements {
			stmt := c.CompileStatement(statement)
			if module.GetExpressionId(stmt) != module.ExpressionIdNop {
				stmts = append(stmts, stmt)
			}
			if innerFlow.IsAny(flow.FlowFlagTerminates | flow.FlowFlagBreaks) {
				possiblyFallsThrough = false
				break
			}
		}

		if possiblyFallsThrough {
			fallThroughFlow = innerFlow
		} else {
			fallThroughFlow = nil
		}
		possiblyBreaks := innerFlow.IsAny(flow.FlowFlagBreaks | flow.FlowFlagConditionallyBreaks)
		innerFlow.UnsetFlag(flow.FlowFlagBreaks | flow.FlowFlagConditionallyBreaks)

		// Combine all alternatives that merge again with outer flow
		if possiblyBreaks || (isLast && possiblyFallsThrough) {
			if breakingFlowAlternatives != nil {
				breakingFlowAlternatives.InheritAlternatives(breakingFlowAlternatives, innerFlow)
			} else {
				breakingFlowAlternatives = innerFlow
			}
		} else if !possiblyFallsThrough {
			outerFlow.MergeSideEffects(innerFlow)
		}

		c.CurrentFlow = outerFlow
		currentBlock = mod.Block(nextLabel, stmts, module.TypeRefNone)
	}
	outerFlow.PopControlFlowLabel(label)

	// If the switch has a default, we only get past through any breaking flow
	if defaultIndex >= 0 {
		if breakingFlowAlternatives != nil {
			outerFlow.Inherit(breakingFlowAlternatives)
		} else {
			outerFlow.SetFlag(flow.FlowFlagTerminates)
		}
	} else if breakingFlowAlternatives != nil {
		outerFlow.MergeBranch(breakingFlowAlternatives)
	}

	c.CurrentFlow = outerFlow
	return currentBlock
}

// compileThrowStatement compiles a throw statement.
// Ported from: assemblyscript/src/compiler.ts compileThrowStatement (lines 2988-3008).
func (c *Compiler) compileThrowStatement(statement *ast.ThrowStatement) module.ExpressionRef {
	// TODO: requires exception-handling spec.
	fl := c.CurrentFlow

	// Remember that this branch throws
	fl.SetFlag(flow.FlowFlagThrows | flow.FlowFlagTerminates)

	stmts := make([]module.ExpressionRef, 0, 1)
	value := statement.Value
	var message ast.Node
	if value.GetKind() == ast.NodeKindNew {
		newExpr := value.(*ast.NewExpression)
		newArgs := newExpr.Args
		if len(newArgs) > 0 {
			message = newArgs[0] // FIXME: naively assumes type string
		}
	}
	stmts = append(stmts, c.makeAbort(message, statement))
	return c.Module().Flatten(stmts, module.TypeRefNone)
}

// compileTryStatement compiles a try statement.
// Ported from: assemblyscript/src/compiler.ts compileTryStatement (lines 3010-3021).
func (c *Compiler) compileTryStatement(statement *ast.TryStatement) module.ExpressionRef {
	// TODO: can't yet support something like: try { return ... } finally { ... }
	// worthwhile to investigate lowering returns to block results (here)?
	c.Error(
		diagnostics.DiagnosticCodeNotImplemented0,
		statement.GetRange(),
		"Exceptions", "", "",
	)
	return c.Module().Unreachable()
}

// compileVariableStatement compiles a variable statement.
// Ported from: assemblyscript/src/compiler.ts compileVariableStatement (lines 3024-3227).
func (c *Compiler) compileVariableStatement(statement *ast.VariableStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	resolver := c.Resolver()
	initializers := make([]module.ExpressionRef, 0, len(statement.Declarations))

	for _, declaration := range statement.Declarations {
		name := declaration.Name.Text
		var resolvedType *types.Type
		var initExpr module.ExpressionRef
		var initType *types.Type

		if declaration.Is(int32(common.CommonFlagsDefinitelyAssigned)) {
			c.Warning(
				diagnostics.DiagnosticCodeDefinitiveAssignmentHasNoEffectOnLocalVariables,
				declaration.Name.GetRange(),
				"", "", "",
			)
		}

		// Resolve type if annotated
		typeNode := declaration.Type
		initializerNode := declaration.Initializer
		if typeNode != nil {
			resolvedType = resolver.ResolveType(
				typeNode, fl,
				fl.SourceFunction().(program.Element),
				fl.ContextualTypeArguments(),
				program.ReportModeReport,
			)
			if resolvedType == nil {
				continue
			}
			c.Program.CheckTypeSupported(resolvedType, typeNode)

			if initializerNode != nil {
				dummy := fl.AddScopedDummyLocal(name, resolvedType, declaration)
				c.PendingElements[dummy.(program.Element)] = struct{}{}
				initExpr = c.CompileExpression(initializerNode, resolvedType, ConstraintsConvImplicit)
				initType = c.CurrentType
				delete(c.PendingElements, dummy.(program.Element))
				fl.FreeScopedDummyLocal(name)
			}

		// Otherwise infer type from initializer
		} else if initializerNode != nil {
			temp := fl.AddScopedDummyLocal(name, types.TypeAuto, declaration)
			c.PendingElements[temp.(program.Element)] = struct{}{}
			initExpr = c.CompileExpression(initializerNode, types.TypeAuto, ConstraintsNone)
			initType = c.CurrentType
			delete(c.PendingElements, temp.(program.Element))
			fl.FreeScopedDummyLocal(name)

			if c.CurrentType == types.TypeVoid {
				c.Error(
					diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
					declaration.GetRange(),
					c.CurrentType.ToString(false), "<auto>", "",
				)
				continue
			}
			resolvedType = initType

		// Error if there's neither a type nor an initializer
		} else {
			c.Error(
				diagnostics.DiagnosticCodeTypeExpected,
				declaration.Name.GetRange().AtEnd(),
				"", "", "",
			)
			continue
		}

		// Handle constants, and try to inline if value is static
		isConst := declaration.Is(int32(common.CommonFlagsConst))
		isStatic := false
		if isConst {
			if initExpr != 0 {
				precomp := mod.RunExpression(initExpr, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
				if precomp != 0 {
					initExpr = precomp // always use precomputed initExpr
					var inlinedLocal *program.Local
					exprTypeRef := module.GetExpressionType(initExpr)
					switch exprTypeRef {
					case module.TypeRefI32:
						inlinedLocal = program.NewLocal(name, -1, resolvedType, fl.TargetFunction.(*program.Function), declaration)
						inlinedLocal.SetConstantIntegerValue(int64(module.GetConstValueI32(initExpr)), resolvedType)
					case module.TypeRefI64:
						inlinedLocal = program.NewLocal(name, -1, resolvedType, fl.TargetFunction.(*program.Function), declaration)
						lo := int64(uint32(module.GetConstValueI64Low(initExpr)))
						hi := int64(module.GetConstValueI64High(initExpr)) << 32
						inlinedLocal.SetConstantIntegerValue(lo|hi, resolvedType)
					case module.TypeRefF32:
						inlinedLocal = program.NewLocal(name, -1, resolvedType, fl.TargetFunction.(*program.Function), declaration)
						inlinedLocal.SetConstantFloatValue(float64(module.GetConstValueF32(initExpr)), resolvedType)
					case module.TypeRefF64:
						inlinedLocal = program.NewLocal(name, -1, resolvedType, fl.TargetFunction.(*program.Function), declaration)
						inlinedLocal.SetConstantFloatValue(module.GetConstValueF64(initExpr), resolvedType)
					}
					if inlinedLocal != nil {
						// Add as a dummy local that doesn't actually exist in WebAssembly
						scopedLocals := fl.ScopedLocals
						if scopedLocals == nil {
							scopedLocals = make(map[string]flow.FlowLocalRef)
							fl.ScopedLocals = scopedLocals
						} else if existing, exists := scopedLocals[name]; exists {
							c.ErrorRelated(
								diagnostics.DiagnosticCodeDuplicateIdentifier0,
								declaration.Name.GetRange(),
								existing.DeclarationNameRange().(*diagnostics.Range),
								name, "", "",
							)
							return mod.Unreachable()
						}
						scopedLocals[name] = inlinedLocal
						isStatic = true
					}
				}
			} else {
				c.Error(
					diagnostics.DiagnosticCodeConstDeclarationsMustBeInitialized,
					declaration.GetRange(),
					"", "", "",
				)
			}
		}

		// Otherwise compile as mutable
		if !isStatic {
			var local flow.FlowLocalRef
			if declaration.IsAny(int32(common.CommonFlagsLet|common.CommonFlagsConst)) ||
				fl.IsInline() {
				// here: not top-level
				existingLocal := fl.GetScopedLocal(name)
				if existingLocal != nil {
					if !existingLocal.DeclarationIsNative() {
						c.ErrorRelated(
							diagnostics.DiagnosticCodeDuplicateIdentifier0,
							declaration.Name.GetRange(),
							existingLocal.DeclarationNameRange().(*diagnostics.Range),
							name, "", "",
						)
					} else {
						// scoped locals are shared temps that don't track declarations
						c.Error(
							diagnostics.DiagnosticCodeDuplicateIdentifier0,
							declaration.Name.GetRange(),
							name, "", "",
						)
					}
					local = existingLocal
				} else {
					local = fl.AddScopedLocal(name, resolvedType)
				}
				if isConst {
					fl.SetLocalFlag(local.FlowIndex(), flow.LocalFlagConstant)
				}
			} else {
				existing := fl.LookupLocal(name)
				if existing != nil {
					c.ErrorRelated(
						diagnostics.DiagnosticCodeDuplicateIdentifier0,
						declaration.Name.GetRange(),
						existing.DeclarationNameRange().(*diagnostics.Range),
						name, "", "",
					)
					continue
				}
				addedLocal := fl.TargetFunction.(*program.Function).AddLocal(resolvedType, name, declaration)
				local = addedLocal
				fl.UnsetLocalFlag(addedLocal.Index, ^flow.LocalFlags(0))
				if isConst {
					fl.SetLocalFlag(addedLocal.Index, flow.LocalFlagConstant)
				}
			}
			if initExpr != 0 {
				if initType == nil {
					initType = resolvedType
				}
				initializers = append(initializers,
					c.makeLocalAssignment(local.(*program.Local), initExpr, initType, false),
				)
			} else {
				// no need to assign zero
				if local.GetType().IsShortIntegerValue() {
					fl.SetLocalFlag(local.FlowIndex(), flow.LocalFlagWrapped)
				}
			}
		}
	}
	c.CurrentType = types.TypeVoid
	if len(initializers) == 0 {
		return mod.Nop()
	}
	return mod.Flatten(initializers, module.TypeRefNone)
}

// compileVoidStatement compiles a void statement.
// Ported from: assemblyscript/src/compiler.ts compileVoidStatement (lines 3229-3235).
func (c *Compiler) compileVoidStatement(statement *ast.VoidStatement) module.ExpressionRef {
	return c.CompileExpression(statement.Expression, types.TypeVoid, ConstraintsConvExplicit|ConstraintsWillDrop)
}

// compileWhileStatement compiles a while statement.
// Ported from: assemblyscript/src/compiler.ts compileWhileStatement (lines 3237-3242).
func (c *Compiler) compileWhileStatement(statement *ast.WhileStatement) module.ExpressionRef {
	return c.doCompileWhileStatement(statement)
}

// doCompileWhileStatement is the inner implementation of compileWhileStatement,
// called recursively when loop convergence requires recompilation.
// Ported from: assemblyscript/src/compiler.ts doCompileWhileStatement (lines 3244-3345).
func (c *Compiler) doCompileWhileStatement(statement *ast.WhileStatement) module.ExpressionRef {
	mod := c.Module()
	outerFlow := c.CurrentFlow
	numLocalsBefore := len(outerFlow.TargetFunction.FlowLocalsByIndex())

	// Compile and evaluate the condition (always executes)
	condExpr := c.CompileExpression(statement.Condition, types.TypeBool, ConstraintsConvImplicit)
	condExprTrueish := c.makeIsTrueish(condExpr, c.CurrentType, statement.Condition)
	condKind := c.evaluateCondition(condExprTrueish)

	// Shortcut if condition is always false (body never runs)
	if condKind == flow.ConditionKindFalse {
		return mod.Drop(condExprTrueish)
	}

	// Compile the body assuming the condition turned out true
	thenFlow := outerFlow.ForkThen(condExpr, true, false)
	label := thenFlow.PushControlFlowLabel()
	breakLabel := fmt.Sprintf("while-break|%d", label)
	continueLabel := fmt.Sprintf("while-continue|%d", label)
	thenFlow.BreakLabel = breakLabel
	thenFlow.ContinueLabel = continueLabel
	c.CurrentFlow = thenFlow
	bodyStmts := make([]module.ExpressionRef, 0)
	body := statement.Body
	if body.GetKind() == ast.NodeKindBlock {
		bodyStmts = c.CompileStatements(body.(*ast.BlockStatement).Statements, bodyStmts)
	} else {
		bodyStmts = append(bodyStmts, c.CompileStatement(body))
	}
	bodyStmts = append(bodyStmts, mod.Br(continueLabel, 0, 0))
	thenFlow.PopControlFlowLabel(label)

	possiblyContinues := thenFlow.IsAny(flow.FlowFlagContinues | flow.FlowFlagConditionallyContinues)
	possiblyBreaks := thenFlow.IsAny(flow.FlowFlagBreaks | flow.FlowFlagConditionallyBreaks)
	possiblyFallsThrough := !thenFlow.IsAny(flow.FlowFlagTerminates | flow.FlowFlagBreaks)

	// Detect if local flags are incompatible before and after looping, and
	// if so recompile by unifying local flags between iterations.
	possiblyLoops := possiblyContinues || possiblyFallsThrough
	if possiblyLoops && outerFlow.ResetIfNeedsRecompile(thenFlow, numLocalsBefore) {
		c.CurrentFlow = outerFlow
		return c.doCompileWhileStatement(statement)
	}

	// If the condition is always true, the body's effects always happen
	alwaysTerminates := false
	if condKind == flow.ConditionKindTrue {
		outerFlow.Inherit(thenFlow)
		// If the body also never breaks, the overall flow terminates
		if !possiblyBreaks {
			alwaysTerminates = true
			outerFlow.SetFlag(flow.FlowFlagTerminates)
		}
	} else {
		// Otherwise loop conditionally
		elseFlow := outerFlow.ForkElse(condExpr)
		if !possiblyFallsThrough && !possiblyBreaks {
			// Only getting past if condition was false
			outerFlow.Inherit(elseFlow)
			outerFlow.MergeSideEffects(thenFlow)
		} else {
			// Otherwise getting past conditionally
			outerFlow.InheritAlternatives(thenFlow, elseFlow)
		}
	}

	// Finalize and leave everything else to the optimizer
	c.CurrentFlow = outerFlow
	stmts := []module.ExpressionRef{
		mod.Loop(continueLabel,
			mod.If(condExprTrueish,
				mod.Flatten(bodyStmts, module.TypeRefNone),
				0,
			),
		),
	}
	if alwaysTerminates {
		stmts = append(stmts, mod.Unreachable())
	}
	return mod.Block(breakLabel, stmts, module.TypeRefNone)
}

// --- Helper methods ---

// makeIsTrueish converts an expression to a boolean (i32) truthy check.
// Ported from: assemblyscript/src/compiler.ts makeIsTrueish (lines 10326-10391).
// makeIsTrueish creates a comparison whether an expression is 'true' in a broader sense.
// Ported from: assemblyscript/src/compiler.ts makeIsTrueish (lines 10166-10274).
func (c *Compiler) makeIsTrueish(expr module.ExpressionRef, typ *types.Type, reportNode ast.Node) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16, types.TypeKindU8, types.TypeKindU16:
		expr = c.ensureSmallIntegerWrap(expr, typ)
		return expr // fall-through to i32 behavior
	case types.TypeKindBool, types.TypeKindI32, types.TypeKindU32:
		return expr
	case types.TypeKindI64, types.TypeKindU64:
		return mod.Binary(module.BinaryOpNeI64, expr, mod.I64(0))
	case types.TypeKindIsize, types.TypeKindUsize:
		if typ.Size == 64 {
			return mod.Binary(module.BinaryOpNeI64, expr, mod.I64(0))
		}
		return expr
	case types.TypeKindF32:
		options := c.Options()
		if options.ShrinkLevelHint > 1 && options.HasFeature(common.FeatureNontrappingF2I) {
			// !!(i32.trunc_sat_f32_u(f32.ceil(f32.abs(x))))
			return mod.Unary(module.UnaryOpEqzI32,
				mod.Unary(module.UnaryOpEqzI32,
					mod.Unary(module.UnaryOpTruncSatF32ToU32,
						mod.Unary(module.UnaryOpCeilF32,
							mod.Unary(module.UnaryOpAbsF32, expr)))))
		}
		// (reinterpret<u32>(x) << 1) - (1 << 1) <= ((0x7F800000 - 1) << 1)
		return mod.Binary(module.BinaryOpLeU32,
			mod.Binary(module.BinaryOpSubI32,
				mod.Binary(module.BinaryOpShlI32,
					mod.Unary(module.UnaryOpReinterpretF32ToI32, expr),
					mod.I32(1)),
				mod.I32(2)),
			mod.I32(-16777218)) // 0xFEFFFFFE = (0x7F800000 - 1) << 1
	case types.TypeKindF64:
		options := c.Options()
		if options.ShrinkLevelHint > 1 && options.HasFeature(common.FeatureNontrappingF2I) {
			// !!(i32.trunc_sat_f64_u(f64.ceil(f64.abs(x))))
			return mod.Unary(module.UnaryOpEqzI32,
				mod.Unary(module.UnaryOpEqzI32,
					mod.Unary(module.UnaryOpTruncSatF64ToU32,
						mod.Unary(module.UnaryOpCeilF64,
							mod.Unary(module.UnaryOpAbsF64, expr)))))
		}
		// (reinterpret<u64>(x) << 1) - (1 << 1) <= ((0x7FF0000000000000 - 1) << 1)
		return mod.Binary(module.BinaryOpLeU64,
			mod.Binary(module.BinaryOpSubI64,
				mod.Binary(module.BinaryOpShlI64,
					mod.Unary(module.UnaryOpReinterpretF64ToI64, expr),
					mod.I64(1)),
				mod.I64(2)),
			mod.I64(-9007199254740994)) // 0xFFDFFFFFFFFFFFFE = (0x7FF0000000000000 - 1) << 1
	case types.TypeKindV128:
		return mod.Unary(module.UnaryOpAnyTrueV128, expr)
	case types.TypeKindFunc, types.TypeKindExtern, types.TypeKindAny,
		types.TypeKindEq, types.TypeKindStruct, types.TypeKindArray,
		types.TypeKindI31, types.TypeKindString,
		types.TypeKindStringviewWTF8, types.TypeKindStringviewWTF16,
		types.TypeKindStringviewIter:
		return mod.Unary(module.UnaryOpEqzI32, mod.RefIsNull(expr))
	default:
		return expr
	}
}

// addDebugLocation adds a debug location for source maps.
func (c *Compiler) addDebugLocation(expr module.ExpressionRef, rng *diagnostics.Range) {
	if rng == nil {
		return
	}
	// Debug location support requires mapping source ranges to file indices.
	// Stub: needs source file index tracking.
}

// GetAutoreleaseLocal on Flow — check if it exists, otherwise use addScopedLocal.
// This is a helper bridge.
