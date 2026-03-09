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
	startFunctionBody := make([]module.ExpressionRef, 0)
	c.CurrentBody = startFunctionBody

	// Compile top-level statements
	previousFlow := c.CurrentFlow
	flow := startFunction.Flow
	c.CurrentFlow = flow
	for _, stmt := range file.Source.Statements {
		c.CompileTopLevelStatement(stmt, &startFunctionBody)
	}
	c.CurrentFlow = previousFlow
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
		previousBody = append(previousBody, mod.Call(startFunction.FlowInternalName(), nil, module.TypeRefNone))
		c.CurrentBody = previousBody
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
func (c *Compiler) CompileEnum(enum *program.Enum) bool {
	if enum.Is(common.CommonFlagsCompiled) {
		return !enum.Is(common.CommonFlagsErrored)
	}
	enum.Set(common.CommonFlagsCompiled)

	mod := c.Module()
	members := enum.GetMembers()
	if members == nil {
		return true
	}

	isConst := enum.Is(common.CommonFlagsConst)
	isDeclaredInline := enum.HasDecorator(program.DecoratorFlagsInline)
	previousValue := int32(-1) // auto-increment starts at 0
	previousValueIsConst := true

	// Iterate members in declaration order
	enumDecl := enum.GetDeclaration().(*ast.EnumDeclaration)
	for _, valueDecl := range enumDecl.Values {
		name := valueDecl.Name.Text
		member, ok := members[name]
		if !ok {
			continue
		}
		enumValue := member.(*program.EnumValue)

		if enumValue.Is(common.CommonFlagsCompiled) {
			continue
		}
		enumValue.Set(common.CommonFlagsCompiled)

		internalName := enumValue.GetInternalName()
		initializerNode := enumValue.ValueNode()

		var initExpr module.ExpressionRef
		if initializerNode != nil {
			// Compile the initializer expression
			previousFlow := c.CurrentFlow
			c.CurrentFlow = enum.File().StartFunction.Flow
			previousParent := c.CurrentParent
			c.CurrentParent = enum
			initExpr = c.CompileExpression(initializerNode, types.TypeI32, ConstraintsConvImplicit)
			c.CurrentParent = previousParent
			c.CurrentFlow = previousFlow

			// Try to precompute to a constant
			precomp := mod.RunExpression(initExpr, module.ExpressionRunnerFlagsDefault, 8, 1)
			if precomp != 0 {
				initExpr = precomp
				previousValue = module.GetConstValueI32(precomp)
				previousValueIsConst = true
			} else {
				previousValueIsConst = false
			}
		} else {
			// Auto-increment from previous value
			if previousValueIsConst {
				previousValue++
				initExpr = mod.I32(previousValue)
			} else {
				// Previous value was not const, can't auto-increment at compile time.
				// Need to compute at runtime: previousGlobal + 1
				c.Error(
					diagnostics.DiagnosticCodeEnumMemberMustHaveInitializer,
					enumValue.IdentifierNode().GetRange(),
					"", "", "",
				)
				enumValue.Set(common.CommonFlagsErrored)
				continue
			}
		}

		if mod.IsConstExpression(initExpr) {
			val := module.GetConstValueI32(initExpr)
			enumValue.SetConstantIntegerValue(int64(val), types.TypeI32)

			if isConst || isDeclaredInline {
				// Fully inlined const enum value, no wasm global needed
				enumValue.IsImmutable = true
				continue
			}

			// Non-const enum: immutable global
			mod.AddGlobal(internalName, module.TypeRefI32, false, initExpr)
			enumValue.IsImmutable = true
		} else {
			// Non-constant: create mutable global, initialize in start function
			mod.AddGlobal(internalName, module.TypeRefI32, true, mod.I32(0))
			c.CurrentBody = append(c.CurrentBody,
				mod.GlobalSet(internalName, initExpr),
			)
			enumValue.IsImmutable = false
		}
	}

	return true
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
