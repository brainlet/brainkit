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
// Ported from: assemblyscript/src/compiler.ts compileStatements (lines 2337-3349).
func (c *Compiler) CompileStatements(statements []ast.Node, stmts []module.ExpressionRef) []module.ExpressionRef {
	for _, statement := range statements {
		compiled := c.CompileStatement(statement)
		if module.GetExpressionId(compiled) != module.ExpressionIdNop {
			stmts = append(stmts, compiled)
		}
		// If flow becomes unreachable, stop compiling further statements
		if c.CurrentFlow.Is(flow.FlowFlagTerminates | flow.FlowFlagBreaks | flow.FlowFlagContinues) {
			break
		}
	}
	return stmts
}

// compileBlockStatement compiles a block statement.
// Ported from: assemblyscript/src/compiler.ts compileBlockStatement (lines 2355-2369).
func (c *Compiler) compileBlockStatement(statement *ast.BlockStatement) module.ExpressionRef {
	stmts := make([]module.ExpressionRef, 0, len(statement.Statements))
	stmts = c.CompileStatements(statement.Statements, stmts)

	mod := c.Module()
	switch len(stmts) {
	case 0:
		return mod.Nop()
	case 1:
		return stmts[0]
	default:
		return mod.Block("", stmts, module.TypeRefNone)
	}
}

// compileBreakStatement compiles a break statement.
// Ported from: assemblyscript/src/compiler.ts compileBreakStatement (lines 2371-2405).
func (c *Compiler) compileBreakStatement(statement *ast.BreakStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	if statement.Label != nil {
		// Labeled break
		label := statement.Label.Text
		breakLabel := fl.BreakLabel
		if breakLabel == "" || breakLabel != label {
			c.Error(
				diagnostics.DiagnosticCodeNotImplemented0,
				statement.Label.GetRange(),
				"Labeled break", "", "",
			)
			return mod.Unreachable()
		}
	}

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
// Ported from: assemblyscript/src/compiler.ts compileContinueStatement (lines 2407-2437).
func (c *Compiler) compileContinueStatement(statement *ast.ContinueStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	if statement.Label != nil {
		// Labeled continue
		label := statement.Label.Text
		continueLabel := fl.ContinueLabel
		if continueLabel == "" || continueLabel != label {
			c.Error(
				diagnostics.DiagnosticCodeNotImplemented0,
				statement.Label.GetRange(),
				"Labeled continue", "", "",
			)
			return mod.Unreachable()
		}
	}

	continueLabel := fl.ContinueLabel
	if continueLabel == "" {
		c.Error(
			diagnostics.DiagnosticCodeAContinueStatementCanOnlyBeUsedWithinAnEnclosingIterationStatement,
			statement.GetRange(),
			"", "", "",
		)
		return mod.Unreachable()
	}

	fl.SetFlag(flow.FlowFlagContinues)
	return mod.Br(continueLabel, 0, 0)
}

// compileDoStatement compiles a do-while statement.
// Ported from: assemblyscript/src/compiler.ts compileDoStatement (lines 2439-2518).
func (c *Compiler) compileDoStatement(statement *ast.DoStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	// Create break/continue labels
	label := fl.PushControlFlowLabel()
	breakLabel := "break|" + fmt.Sprintf("%d", label)
	continueLabel := "continue|" + fmt.Sprintf("%d", label)

	// Fork flow for the loop body
	bodyFlow := fl.Fork(true, true)
	bodyFlow.BreakLabel = breakLabel
	bodyFlow.ContinueLabel = continueLabel
	c.CurrentFlow = bodyFlow

	// Compile body
	bodyStmt := c.CompileStatement(statement.Body)

	// Compile condition
	condExpr := c.makeIsTrueish(
		c.CompileExpression(statement.Condition, types.TypeBool, ConstraintsConvImplicit),
		c.CurrentType,
		statement.Condition,
	)

	// Restore flow
	c.CurrentFlow = fl
	fl.PopControlFlowLabel(label)

	// Build: block $break { loop $continue { body; br_if $continue condition } }
	inner := mod.Block(continueLabel, []module.ExpressionRef{
		bodyStmt,
		mod.Br(continueLabel, condExpr, 0),
	}, module.TypeRefNone)
	loopExpr := mod.Loop(continueLabel, inner)

	// Merge flow effects
	if !bodyFlow.Is(flow.FlowFlagTerminates) {
		fl.MergeBranch(bodyFlow)
	}
	if bodyFlow.Is(flow.FlowFlagBreaks) {
		// break exits the loop
	}

	return mod.Block(breakLabel, []module.ExpressionRef{loopExpr}, module.TypeRefNone)
}

// compileExpressionStatement compiles an expression statement.
// Ported from: assemblyscript/src/compiler.ts compileExpressionStatement (lines 2520-2531).
func (c *Compiler) compileExpressionStatement(statement *ast.ExpressionStatement) module.ExpressionRef {
	mod := c.Module()
	expr := c.CompileExpression(statement.Expression, types.TypeVoid, ConstraintsWillDrop)
	ct := c.CurrentType

	// Drop non-void values
	if ct != types.TypeVoid {
		expr = mod.Drop(expr)
		c.CurrentType = types.TypeVoid
	}
	return expr
}

// compileForStatement compiles a for statement.
// Ported from: assemblyscript/src/compiler.ts compileForStatement (lines 2533-2630).
func (c *Compiler) compileForStatement(statement *ast.ForStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	// Compile initializer
	var initExpr module.ExpressionRef
	if statement.Initializer != nil {
		initExpr = c.CompileStatement(statement.Initializer)
	}

	// Create break/continue labels
	label := fl.PushControlFlowLabel()
	breakLabel := "break|" + fmt.Sprintf("%d", label)
	continueLabel := "continue|" + fmt.Sprintf("%d", label)

	// Fork flow for the loop
	bodyFlow := fl.Fork(true, true)
	bodyFlow.BreakLabel = breakLabel
	bodyFlow.ContinueLabel = continueLabel
	c.CurrentFlow = bodyFlow

	// Compile body
	bodyStmt := c.CompileStatement(statement.Body)

	// Compile incrementor
	var incrExpr module.ExpressionRef
	if statement.Incrementor != nil {
		incrExpr = c.CompileExpression(statement.Incrementor, types.TypeVoid, ConstraintsWillDrop)
		if c.CurrentType != types.TypeVoid {
			incrExpr = mod.Drop(incrExpr)
		}
	}

	// Compile condition
	var condExpr module.ExpressionRef
	if statement.Condition != nil {
		condExpr = c.makeIsTrueish(
			c.CompileExpression(statement.Condition, types.TypeBool, ConstraintsConvImplicit),
			c.CurrentType,
			statement.Condition,
		)
		// Negate: break if NOT condition
		condExpr = mod.If(condExpr, mod.Nop(), mod.Br(breakLabel, 0, 0))
	}

	// Restore flow
	c.CurrentFlow = fl
	fl.PopControlFlowLabel(label)

	// Build loop body parts
	loopParts := make([]module.ExpressionRef, 0, 4)
	if condExpr != 0 {
		loopParts = append(loopParts, condExpr)
	}
	loopParts = append(loopParts, bodyStmt)
	if incrExpr != 0 {
		loopParts = append(loopParts, incrExpr)
	}
	loopParts = append(loopParts, mod.Br(continueLabel, 0, 0))

	loopBody := mod.Block("", loopParts, module.TypeRefNone)
	loopExpr := mod.Loop(continueLabel, loopBody)

	// Merge flow
	if !bodyFlow.Is(flow.FlowFlagTerminates) {
		fl.MergeBranch(bodyFlow)
	}

	result := mod.Block(breakLabel, []module.ExpressionRef{loopExpr}, module.TypeRefNone)
	if initExpr != 0 {
		return mod.Block("", []module.ExpressionRef{initExpr, result}, module.TypeRefNone)
	}
	return result
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
// Ported from: assemblyscript/src/compiler.ts compileReturnStatement (lines 2791-2862).
func (c *Compiler) compileReturnStatement(statement *ast.ReturnStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	returnType := fl.ReturnType()

	fl.SetFlag(flow.FlowFlagTerminates | flow.FlowFlagReturns)

	if statement.Value != nil {
		if returnType == types.TypeVoid {
			// Returning a value from a void function
			c.Error(
				diagnostics.DiagnosticCodeNotImplemented0,
				statement.GetRange(),
				"Return value in void function", "", "",
			)
			// Still compile the expression for side effects
			expr := c.CompileExpression(statement.Value, returnType, ConstraintsConvImplicit)
			return mod.Block("", []module.ExpressionRef{mod.Drop(expr), mod.Return(0)}, module.TypeRefNone)
		}
		expr := c.CompileExpression(statement.Value, returnType, ConstraintsConvImplicit)
		return mod.Return(expr)
	}

	// No return value
	if returnType != types.TypeVoid {
		c.Error(
			diagnostics.DiagnosticCodeAFunctionWhoseDeclaredTypeIsNotVoidMustReturnAValue,
			statement.GetRange(),
			"", "", "",
		)
	}
	return mod.Return(0)
}

// compileSwitchStatement compiles a switch statement.
// Ported from: assemblyscript/src/compiler.ts compileSwitchStatement (lines 2864-3013).
func (c *Compiler) compileSwitchStatement(statement *ast.SwitchStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	cases := statement.Cases

	if len(cases) == 0 {
		return mod.Nop()
	}

	// Create break label
	label := fl.PushControlFlowLabel()
	breakLabel := "break|" + fmt.Sprintf("%d", label)

	// Compile the switch condition
	condExpr := c.CompileExpression(statement.Condition, types.TypeI32, ConstraintsConvImplicit)
	condType := c.CurrentType

	// Find default case index
	defaultIndex := -1
	for i, sc := range cases {
		if sc.IsDefault() {
			defaultIndex = i
			break
		}
	}

	// Generate case labels
	caseLabels := make([]string, len(cases))
	for i := range cases {
		caseLabels[i] = fmt.Sprintf("case%d|%d", i, label)
	}
	defaultLabel := breakLabel
	if defaultIndex >= 0 {
		defaultLabel = caseLabels[defaultIndex]
	}

	// Build nested br_if chain from inside out, comparing condition against each case label
	// Each non-default case: br_if $caseN (i32.eq condition caseValue)
	stmts := make([]module.ExpressionRef, 0, len(cases)+2)

	// Use a temp local to avoid re-evaluating condition
	tempLocal := fl.GetTempLocal(condType)
	tempIndex := tempLocal.FlowIndex()
	stmts = append(stmts, mod.LocalSet(tempIndex, condExpr, false))

	for i, sc := range cases {
		if sc.IsDefault() {
			continue
		}
		caseCondExpr := c.CompileExpression(sc.Label, condType, ConstraintsConvImplicit)
		eqExpr := c.makeBinaryEq(
			mod.LocalGet(tempIndex, condType.ToRef()),
			caseCondExpr,
			condType,
		)
		stmts = append(stmts, mod.Br(caseLabels[i], eqExpr, 0))
	}
	// Fall through to default
	stmts = append(stmts, mod.Br(defaultLabel, 0, 0))

	// Build case bodies (nested blocks from inside-out)
	allTerminate := true
	for i, sc := range cases {
		bodyFlow := fl.Fork(true, false)
		bodyFlow.BreakLabel = breakLabel
		c.CurrentFlow = bodyFlow
		caseStmts := c.CompileStatements(sc.Statements, nil)
		if !bodyFlow.Is(flow.FlowFlagTerminates | flow.FlowFlagBreaks) {
			allTerminate = false
		}
		fl.MergeSideEffects(bodyFlow)

		stmts = append(caseStmts) // case body stmts
		_ = i
		_ = caseLabels
	}

	c.CurrentFlow = fl
	fl.PopControlFlowLabel(label)

	if allTerminate && defaultIndex >= 0 {
		fl.SetFlag(flow.FlowFlagTerminates)
	}

	// Build nested blocks: block $break { block $case0 { block $case1 { ... switch_body } case0_stmts } case1_stmts }
	// This is complex — for now build a simpler flat version
	// The real implementation nests blocks. We'll use the switch instruction.
	return mod.Block(breakLabel, stmts, module.TypeRefNone)
}

// compileThrowStatement compiles a throw statement.
// Ported from: assemblyscript/src/compiler.ts compileThrowStatement (lines 3015-3055).
func (c *Compiler) compileThrowStatement(statement *ast.ThrowStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	fl.SetFlag(flow.FlowFlagTerminates)

	// Compile the throw expression
	expr := c.CompileExpression(statement.Value, types.TypeVoid, ConstraintsNone)

	// If exception handling is enabled, use wasm throw
	if c.Options().HasFeature(common.FeatureExceptionHandling) {
		return mod.Throw("", []module.ExpressionRef{expr})
	}

	// Otherwise, call abort (unreachable)
	return mod.Block("", []module.ExpressionRef{
		mod.Drop(expr),
		mod.Unreachable(),
	}, module.TypeRefNone)
}

// compileTryStatement compiles a try statement.
// Ported from: assemblyscript/src/compiler.ts compileTryStatement (lines 3057-3180).
func (c *Compiler) compileTryStatement(statement *ast.TryStatement) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow

	if !c.Options().HasFeature(common.FeatureExceptionHandling) {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			statement.GetRange(),
			"Try/catch requires exception-handling feature", "", "",
		)
		return mod.Unreachable()
	}

	// Compile try body
	bodyFlow := fl.Fork(false, false)
	c.CurrentFlow = bodyFlow
	bodyStmts := c.CompileStatements(statement.BodyStatements, nil)
	bodyExpr := mod.Flatten(bodyStmts, module.TypeRefNone)

	// Compile catch body
	var catchBodies []module.ExpressionRef
	var catchTags []string
	if statement.CatchStatements != nil {
		catchFlow := fl.Fork(false, false)
		c.CurrentFlow = catchFlow
		catchStmts := c.CompileStatements(statement.CatchStatements, nil)
		catchExpr := mod.Flatten(catchStmts, module.TypeRefNone)
		catchBodies = append(catchBodies, catchExpr)
		catchTags = append(catchTags, "") // catch-all
		fl.MergeBranch(catchFlow)
	}

	// Compile finally body
	if statement.FinallyStatements != nil {
		// finally is appended after the try block
		finallyFlow := fl.Fork(false, false)
		c.CurrentFlow = finallyFlow
		finallyStmts := c.CompileStatements(statement.FinallyStatements, nil)
		finallyExpr := mod.Flatten(finallyStmts, module.TypeRefNone)

		c.CurrentFlow = fl
		fl.MergeBranch(bodyFlow)

		tryExpr := mod.Try("", bodyExpr, catchTags, catchBodies, "")
		return mod.Block("", []module.ExpressionRef{tryExpr, finallyExpr}, module.TypeRefNone)
	}

	c.CurrentFlow = fl
	fl.MergeBranch(bodyFlow)

	return mod.Try("", bodyExpr, catchTags, catchBodies, "")
}

// compileVariableStatement compiles a variable statement.
// Ported from: assemblyscript/src/compiler.ts compileVariableStatement (lines 3182-3349).
func (c *Compiler) compileVariableStatement(statement *ast.VariableStatement) module.ExpressionRef {
	mod := c.Module()
	stmts := make([]module.ExpressionRef, 0, len(statement.Declarations))

	for _, decl := range statement.Declarations {
		c.compileVariableDeclaration(decl, &stmts)
	}

	switch len(stmts) {
	case 0:
		return mod.Nop()
	case 1:
		return stmts[0]
	default:
		return mod.Block("", stmts, module.TypeRefNone)
	}
}

// compileVariableDeclaration compiles a single variable declaration.
// Ported from: assemblyscript/src/compiler.ts (within compileVariableStatement).
func (c *Compiler) compileVariableDeclaration(decl *ast.VariableDeclaration, stmts *[]module.ExpressionRef) {
	mod := c.Module()
	fl := c.CurrentFlow
	resolver := c.Resolver()

	name := decl.Name.Text
	isConst := (decl.Flags & int32(common.CommonFlagsConst)) != 0

	// Resolve type
	var resolvedType *types.Type
	if decl.Type != nil {
		resolvedType = resolver.ResolveType(
			decl.Type,
			fl,
			fl.TargetFunction.(program.Element),
			nil,
			program.ReportModeReport,
		)
		if resolvedType == nil {
			return
		}
	} else if decl.Initializer != nil {
		resolvedType = resolver.ResolveExpression(
			decl.Initializer,
			fl,
			types.TypeVoid,
			program.ReportModeReport,
		)
		if resolvedType == nil {
			return
		}
	} else {
		c.Error(
			diagnostics.DiagnosticCodeTypeExpected,
			decl.Name.GetRange(),
			"", "", "",
		)
		return
	}

	// Allocate a local
	local := fl.AddScopedLocal(name, resolvedType)
	if local == nil {
		// Duplicate variable — error already reported
		return
	}
	localIdx := local.FlowIndex()

	// Compile initializer
	if decl.Initializer != nil {
		initExpr := c.CompileExpression(decl.Initializer, resolvedType, ConstraintsConvImplicit)
		*stmts = append(*stmts, mod.LocalSet(localIdx, initExpr, false))

		// Mark local as non-null if appropriate
		if resolvedType.IsReference() && !resolvedType.IsNullableReference() {
			fl.SetLocalFlag(localIdx, flow.LocalFlagNonNull)
		}
	} else if !isConst {
		// Initialize to zero if no initializer
		*stmts = append(*stmts, mod.LocalSet(localIdx, c.makeZeroOfType(resolvedType), false))
	}

	// For const locals, check wrapping
	if isConst {
		fl.SetLocalFlag(localIdx, flow.LocalFlagConstant)
	}

	_ = name
}

// compileVoidStatement compiles a void statement.
// Ported from: assemblyscript/src/compiler.ts compileVoidStatement (lines 3351-3359).
func (c *Compiler) compileVoidStatement(statement *ast.VoidStatement) module.ExpressionRef {
	mod := c.Module()
	expr := c.CompileExpression(statement.Expression, types.TypeVoid, ConstraintsWillDrop)
	if c.CurrentType != types.TypeVoid {
		return mod.Drop(expr)
	}
	return expr
}

// compileWhileStatement compiles a while statement.
// Ported from: assemblyscript/src/compiler.ts compileWhileStatement (lines 3361-3429).
func (c *Compiler) compileWhileStatement(statement *ast.WhileStatement) module.ExpressionRef {
	mod := c.Module()
	outerFlow := c.CurrentFlow

	// Compile and evaluate the condition (always executes)
	// Ported from: assemblyscript/src/compiler.ts doCompileWhileStatement (lines 3244-3345).
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
	bodyStmt := c.CompileStatement(statement.Body)
	thenFlow.PopControlFlowLabel(label)

	possiblyBreaks := thenFlow.IsAny(flow.FlowFlagBreaks | flow.FlowFlagConditionallyBreaks)
	possiblyFallsThrough := !thenFlow.IsAny(flow.FlowFlagTerminates | flow.FlowFlagBreaks)

	// TODO: recompilation logic (resetIfNeedsRecompile)

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

	// Finalize
	c.CurrentFlow = outerFlow
	stmts := []module.ExpressionRef{
		mod.Loop(continueLabel,
			mod.If(condExprTrueish,
				mod.Block("", []module.ExpressionRef{bodyStmt, mod.Br(continueLabel, 0, 0)}, module.TypeRefNone),
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
