// Ported from: assemblyscript/src/compiler.ts file compilation methods (lines 1078-1138)
// and compileTopLevelStatement (lines 2150-2231).
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// CompileFileByPath compiles the file matching the specified path.
// Ported from: assemblyscript/src/compiler.ts compileFileByPath (lines 1078-1094).
func (c *Compiler) CompileFileByPath(normalizedPathWithoutExtension string, reportNode ast.Node) {
	filesByName := c.Program.FilesByName
	var file *program.File
	if f, ok := filesByName[normalizedPathWithoutExtension]; ok {
		file = f
	} else if f, ok := filesByName[normalizedPathWithoutExtension+common.INDEX_SUFFIX]; ok {
		file = f
	} else {
		c.Error(
			diagnostics.DiagnosticCodeFile0NotFound,
			reportNode.GetRange(),
			normalizedPathWithoutExtension, "", "",
		)
		return
	}
	c.CompileFile(file)
}

// CompileFile compiles the specified file.
// Ported from: assemblyscript/src/compiler.ts compileFile (lines 1097-1138).
func (c *Compiler) CompileFile(file *program.File) {
	if file.Is(common.CommonFlagsCompiled) {
		return
	}
	file.Set(common.CommonFlagsCompiled)

	// Compile top-level statements within the file's start function
	startFunction := file.StartFunction
	startSignature := startFunction.Signature
	previousBody := c.CurrentBody
	c.CurrentBody = make([]module.ExpressionRef, 0)

	// Compile top-level statements
	// Pass &c.CurrentBody so both CompileTopLevelStatement (which appends through body pointer)
	// and CompileGlobal/CompileEnum (which append through c.CurrentBody directly) always write
	// to the same slice. In JS/TS both variables reference the same array object; in Go we must
	// ensure the pointer and field stay in sync.
	previousFlow := c.CurrentFlow
	flow := startFunction.Flow
	c.CurrentFlow = flow
	for _, stmt := range file.Source.Statements {
		c.CompileTopLevelStatement(stmt, &c.CurrentBody)
	}
	c.CurrentFlow = previousFlow

	// Capture the file's body AFTER compilation (c.CurrentBody reflects all appends)
	startFunctionBody := c.CurrentBody
	c.CurrentBody = previousBody

	// If top-level statements are present, make the per-file start function and call it in start
	if len(startFunctionBody) > 0 {
		mod := c.Module()
		funcRef := mod.AddFunction(
			startFunction.FlowInternalName(),
			startSignature.ParamRefs(),
			startSignature.ResultRefs(),
			typesToRefs(startFunction.GetNonParameterLocalTypes()),
			mod.Flatten(startFunctionBody, module.TypeRefNone),
		)
		startFunction.Finalize(mod, funcRef)
		c.CurrentBody = append(c.CurrentBody, mod.Call(startFunction.FlowInternalName(), nil, module.TypeRefNone))
	}
}

// CompileTopLevelStatement compiles a top-level statement within a file.
// Dispatches based on statement kind to compile declarations, variables, imports, exports.
// Ported from: assemblyscript/src/compiler.ts compileTopLevelStatement (lines 2150-2231).
func (c *Compiler) CompileTopLevelStatement(statement ast.Node, body *[]module.ExpressionRef) {
	prog := c.Program
	switch statement.GetKind() {
	case ast.NodeKindClassDeclaration:
		classDecl := statement.(*ast.ClassDeclaration)
		for _, member := range classDecl.Members {
			c.CompileTopLevelStatement(member, body)
		}

	case ast.NodeKindEnumDeclaration:
		element := prog.GetElementByDeclaration(statement)
		if element != nil {
			if !element.HasDecorator(program.DecoratorFlagsLazy) {
				c.CompileEnum(element.(*program.Enum))
			}
		}

	case ast.NodeKindNamespaceDeclaration:
		nsDecl := statement.(*ast.NamespaceDeclaration)
		element := prog.GetElementByDeclaration(statement)
		if element != nil {
			previousParent := c.CurrentParent
			c.CurrentParent = element
			for _, member := range nsDecl.Members {
				c.CompileTopLevelStatement(member, body)
			}
			c.CurrentParent = previousParent
		}

	case ast.NodeKindVariable:
		varStmt := statement.(*ast.VariableStatement)
		for _, decl := range varStmt.Declarations {
			element := prog.GetElementByDeclaration(decl)
			if element != nil {
				if !element.Is(common.CommonFlagsAmbient) && !element.HasDecorator(program.DecoratorFlagsLazy) {
					c.CompileGlobal(element.(*program.Global))
				}
			}
		}

	case ast.NodeKindFieldDeclaration:
		element := prog.GetElementByDeclaration(statement)
		if element != nil && element.GetElementKind() == program.ElementKindGlobal { // static
			if !element.HasDecorator(program.DecoratorFlagsLazy) {
				c.CompileGlobal(element.(*program.Global))
			}
		}

	case ast.NodeKindExport:
		exportStmt := statement.(*ast.ExportStatement)
		if exportStmt.HasInternal {
			c.CompileFileByPath(exportStmt.InternalPath, exportStmt.Path)
		}

	case ast.NodeKindExportDefault:
		exportDefault := statement.(*ast.ExportDefaultStatement)
		c.CompileTopLevelStatement(exportDefault.Declaration, body)

	case ast.NodeKindImport:
		importStmt := statement.(*ast.ImportStatement)
		c.CompileFileByPath(importStmt.InternalPath, importStmt.Path)

	case ast.NodeKindFunctionDeclaration,
		ast.NodeKindMethodDeclaration,
		ast.NodeKindInterfaceDeclaration,
		ast.NodeKindIndexSignature,
		ast.NodeKindTypeDeclaration:
		// These declarations are compiled on-demand, not at top level.

	default:
		// Otherwise a top-level statement that is part of the start function's body
		stmt := c.CompileStatement(statement)
		if module.GetExpressionId(stmt) != module.ExpressionIdNop {
			*body = append(*body, stmt)
		}
	}
}

// CompileEnum compiles an enum declaration.
// Ported from: assemblyscript/src/compiler.ts compileEnum (lines 1419-1523).
func (c *Compiler) CompileEnum(element *program.Enum) bool {
	if element.Is(common.CommonFlagsCompiled) {
		return !element.Is(common.CommonFlagsErrored)
	}
	element.Set(common.CommonFlagsCompiled)

	pendingElements := c.PendingElements
	pendingElements[element] = struct{}{}

	mod := c.Module()
	previousParent := c.CurrentParent
	c.CurrentParent = element
	var previousValue *program.EnumValue
	previousValueIsMut := false
	isInline := element.Is(common.CommonFlagsConst) || element.HasDecorator(program.DecoratorFlagsInline)

	members := element.GetMembers()
	if members != nil {
		// Iterate in declaration order (TS Map preserves insertion order, Go map does not).
		// Use the enum declaration's Values slice which is ordered.
		enumDecl := element.GetDeclaration().(*ast.EnumDeclaration)
		for _, valueDecl := range enumDecl.Values {
			memberName := valueDecl.Name.Text
			member := members[memberName]
			if member == nil || member.GetElementKind() != program.ElementKindEnumValue {
				continue // happens if an enum is also a namespace
			}
			initInStart := false
			enumValue := member.(*program.EnumValue)
			valueNode := enumValue.ValueNode()
			enumValue.Set(common.CommonFlagsCompiled)
			previousFlow := c.CurrentFlow
			if element.HasDecorator(program.DecoratorFlagsLazy) {
				c.CurrentFlow = element.File().StartFunction.Flow
			}
			var initExpr module.ExpressionRef
			if valueNode != nil {
				initExpr = c.CompileExpression(valueNode, types.TypeI32, ConstraintsConvImplicit)
				if module.GetExpressionId(initExpr) != module.ExpressionIdConst {
					precomp := mod.RunExpression(initExpr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
					if precomp != 0 {
						initExpr = precomp
					} else {
						if element.Is(common.CommonFlagsConst) {
							c.Error(
								diagnostics.DiagnosticCodeInConstEnumDeclarationsMemberInitializerMustBeConstantExpression,
								valueNode.GetRange(),
								"", "", "",
							)
						}
						initInStart = true
					}
				}
			} else if previousValue == nil {
				initExpr = mod.I32(0)
			} else {
				if previousValueIsMut {
					c.Error(
						diagnostics.DiagnosticCodeEnumMemberMustHaveInitializer,
						enumValue.IdentifierNode().GetRange().AtEnd(),
						"", "", "",
					)
				}
				if isInline {
					value := previousValue.GetConstantIntegerValue() + 1
					initExpr = mod.I32(int32(value))
				} else {
					initExpr = mod.Binary(module.BinaryOpAddI32,
						mod.GlobalGet(previousValue.GetInternalName(), module.TypeRefI32),
						mod.I32(1),
					)
					precomp := mod.RunExpression(initExpr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
					if precomp != 0 {
						initExpr = precomp
					} else {
						if element.Is(common.CommonFlagsConst) {
							c.Error(
								diagnostics.DiagnosticCodeInConstEnumDeclarationsMemberInitializerMustBeConstantExpression,
								member.GetDeclaration().GetRange(),
								"", "", "",
							)
						}
						initInStart = true
					}
				}
			}
			c.CurrentFlow = previousFlow
			if initInStart {
				mod.AddGlobal(enumValue.GetInternalName(), module.TypeRefI32, true, mod.I32(0))
				c.CurrentBody = append(c.CurrentBody,
					c.makeGlobalAssignment(enumValue, initExpr, types.TypeI32, false),
				)
				previousValueIsMut = true
			} else {
				if isInline {
					enumValue.SetConstantIntegerValue(int64(module.GetConstValueI32(initExpr)), types.TypeI32)
					if enumValue.Is(common.CommonFlagsModuleExport) {
						mod.AddGlobal(enumValue.GetInternalName(), module.TypeRefI32, false, initExpr)
					}
				} else {
					mod.AddGlobal(enumValue.GetInternalName(), module.TypeRefI32, false, initExpr)
				}
				enumValue.IsImmutable = true
				previousValueIsMut = false
			}
			previousValue = enumValue
		}
	}
	c.CurrentParent = previousParent
	delete(pendingElements, element)
	return true
}

// ensureEnumToString ensures a toString function exists for the given enum.
// Generates a function with an if-chain comparing enum values and returning string literals.
// When values are the same, returns the last enum value name (iterates in reverse).
// Ported from: assemblyscript/src/compiler.ts ensureEnumToString (lines 1525-1566).
func (c *Compiler) ensureEnumToString(enumElement *program.Enum, reportNode ast.Node) string {
	if enumElement.ToStringFunctionName != "" {
		return enumElement.ToStringFunctionName
	}

	if !c.CompileEnum(enumElement) {
		return ""
	}
	if enumElement.Is(common.CommonFlagsConst) {
		c.ErrorRelated(
			diagnostics.DiagnosticCodeAConstEnumMemberCanOnlyBeAccessedUsingAStringLiteral,
			reportNode.GetRange(), enumElement.IdentifierNode().GetRange(),
			"", "", "",
		)
		return ""
	}

	members := enumElement.GetMembers()
	if members == nil {
		return ""
	}

	mod := c.Module()
	isInline := enumElement.HasDecorator(program.DecoratorFlagsInline)

	functionName := enumElement.GetInternalName() + "#" + common.CommonNameEnumToString
	enumElement.ToStringFunctionName = functionName

	exprs := make([]module.ExpressionRef, 0)
	// When the values are the same, TS returns the last enum value name that appears,
	// so iterate in reverse declaration order.
	enumDecl := enumElement.GetDeclaration().(*ast.EnumDeclaration)
	values := enumDecl.Values
	for i := len(values) - 1; i >= 0; i-- {
		valueDecl := values[i]
		enumValueName := valueDecl.Name.Text
		member, ok := members[enumValueName]
		if !ok {
			continue
		}
		if member.GetElementKind() != program.ElementKindEnumValue {
			continue
		}
		enumValue := member.(*program.EnumValue)

		var enumValueExpr module.ExpressionRef
		if isInline {
			enumValueExpr = mod.I32(int32(enumValue.GetConstantIntegerValue()))
		} else {
			enumValueExpr = mod.GlobalGet(enumValue.GetInternalName(), module.TypeRefI32)
		}
		expr := mod.If(
			mod.Binary(module.BinaryOpEqI32, enumValueExpr, mod.LocalGet(0, module.TypeRefI32)),
			mod.Return(c.EnsureStaticString(enumValueName)),
			0,
		)
		exprs = append(exprs, expr)
	}
	exprs = append(exprs, mod.Unreachable())
	mod.AddFunction(functionName, module.TypeRefI32, module.TypeRefI32, nil, mod.Block("", exprs, module.TypeRefI32))

	return functionName
}

// CompileStatement is now in compile_statement.go

// typesToRefs converts a slice of types to a slice of TypeRefs.
func typesToRefs(typs []*types.Type) []module.TypeRef {
	refs := make([]module.TypeRef, len(typs))
	for i, t := range typs {
		refs[i] = t.ToRef()
	}
	return refs
}
