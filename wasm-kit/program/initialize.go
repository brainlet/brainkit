package program

// This file contains the initializeXxx worker methods, queue types, and helpers
// used by Program.Initialize(). Ported 1:1 from assemblyscript/src/program.ts.

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
)

// ---------------------------------------------------------------------------
// Queue types (deferred element resolution during initialization)
// ---------------------------------------------------------------------------

// QueuedImport represents a yet unresolved `import`.
// Ported from: assemblyscript/src/program.ts QueuedImport.
type QueuedImport struct {
	LocalFile         *File
	LocalIdentifier   *ast.IdentifierExpression
	ForeignIdentifier *ast.IdentifierExpression // nil indicates import *
	ForeignPath       string
	ForeignPathAlt    string
}

// QueuedExport represents a yet unresolved `export`.
// Ported from: assemblyscript/src/program.ts QueuedExport.
type QueuedExport struct {
	LocalIdentifier   *ast.IdentifierExpression
	ForeignIdentifier *ast.IdentifierExpression
	ForeignPath       string // empty string = no re-export path (local export)
	ForeignPathAlt    string
}

// QueuedExportStar represents a yet unresolved `export *`.
// Ported from: assemblyscript/src/program.ts QueuedExportStar.
type QueuedExportStar struct {
	ForeignPath    string
	ForeignPathAlt string
	PathLiteral    *ast.StringLiteralExpression
}

// ---------------------------------------------------------------------------
// Foreign file / element lookup helpers
// ---------------------------------------------------------------------------

// lookupForeignFile tries to locate a foreign file given its normalized path.
// Ported from: assemblyscript/src/program.ts Program.lookupForeignFile.
func (p *Program) lookupForeignFile(foreignPath string, foreignPathAlt string) *File {
	if file, ok := p.FilesByName[foreignPath]; ok {
		return file
	}
	if file, ok := p.FilesByName[foreignPathAlt]; ok {
		return file
	}
	return nil
}

// lookupForeign tries to locate a foreign element by traversing exports and queued exports.
// Ported from: assemblyscript/src/program.ts Program.lookupForeign.
func (p *Program) lookupForeign(
	foreignName string,
	foreignFile *File,
	queuedExports map[*File]map[string]*QueuedExport,
) DeclaredElement {
	for {
		// check if already resolved
		element := foreignFile.LookupExport(foreignName)
		if element != nil {
			return element
		}

		// follow queued exports
		if fileExports, ok := queuedExports[foreignFile]; ok {
			if queuedExport, ok := fileExports[foreignName]; ok {
				queuedExportForeignPath := queuedExport.ForeignPath

				// re-exported from another file
				if queuedExportForeignPath != "" {
					otherFile := p.lookupForeignFile(queuedExportForeignPath, queuedExport.ForeignPathAlt)
					if otherFile == nil {
						return nil
					}
					foreignName = queuedExport.LocalIdentifier.Text
					foreignFile = otherFile
					continue
				}

				// exported from this file
				element = foreignFile.GetMember(queuedExport.LocalIdentifier.Text)
				if element != nil {
					return element
				}
			}
		}
		break
	}

	// follow star exports
	if foreignFile.ExportsStar != nil {
		for _, starFile := range foreignFile.ExportsStar {
			element := p.lookupForeign(foreignName, starFile, queuedExports)
			if element != nil {
				return element
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Decorator validation
// ---------------------------------------------------------------------------

// checkDecorators validates that only supported decorators are present.
// Ported from: assemblyscript/src/program.ts Program.checkDecorators.
func (p *Program) checkDecorators(decorators []*ast.DecoratorNode, acceptedFlags DecoratorFlags) DecoratorFlags {
	var flags DecoratorFlags
	if decorators != nil {
		for _, decorator := range decorators {
			kind := decorator.DecoratorKind
			flag := DecoratorFlagsFromKind(kind)
			if flag != 0 {
				if (acceptedFlags & flag) == 0 {
					nameRange := decorator.Name.GetRange()
					p.Error(
						diagnostics.DiagnosticCodeDecorator0IsNotValidHere,
						decorator.GetRange(),
						rangeToString(nameRange),
					)
				} else if (flags & flag) != 0 {
					p.Error(
						diagnostics.DiagnosticCodeDuplicateDecorator,
						decorator.GetRange(),
					)
				} else {
					flags |= flag
				}
			}
		}
	}
	return flags
}

// rangeToString returns a string representation of a range using its source text.
func rangeToString(r *diagnostics.Range) string {
	if r == nil || r.Source == nil {
		return ""
	}
	src, ok := r.Source.(interface{ SourceText() string })
	if !ok {
		return ""
	}
	text := src.SourceText()
	start := int(r.Start)
	end := int(r.End)
	if start < 0 {
		start = 0
	}
	if end > len(text) {
		end = len(text)
	}
	if start >= end {
		return ""
	}
	return text[start:end]
}

// ---------------------------------------------------------------------------
// Operator overload checking
// ---------------------------------------------------------------------------

// checkOperatorOverloads checks that operator overloads are generally valid.
// Ported from: assemblyscript/src/program.ts Program.checkOperatorOverloads.
func (p *Program) checkOperatorOverloads(
	decorators []*ast.DecoratorNode,
	prototype *FunctionPrototype,
	classPrototype *ClassPrototype,
) {
	if decorators == nil {
		return
	}
	for _, decorator := range decorators {
		switch decorator.DecoratorKind {
		case ast.DecoratorKindOperator, ast.DecoratorKindOperatorBinary,
			ast.DecoratorKindOperatorPrefix, ast.DecoratorKindOperatorPostfix:
			args := decorator.Args
			numArgs := len(args)
			if numArgs == 1 {
				firstArg := args[0]
				if ast.IsLiteralKind(firstArg, ast.LiteralKindString) {
					text := firstArg.(*ast.StringLiteralExpression).Value
					kind := OperatorKindFromDecorator(decorator.DecoratorKind, text)
					if kind == OperatorKindInvalid {
						p.Error(
							diagnostics.DiagnosticCode0IsNotAValidOperator,
							firstArg.GetRange(), text,
						)
					} else {
						if _, exists := classPrototype.OperatorOverloadPrototypes[kind]; exists {
							p.Error(
								diagnostics.DiagnosticCodeDuplicateFunctionImplementation,
								firstArg.GetRange(),
							)
						} else {
							prototype.OperatorKind = kind
							classPrototype.OperatorOverloadPrototypes[kind] = prototype
						}
					}
				} else {
					p.Error(
						diagnostics.DiagnosticCodeStringLiteralExpected,
						firstArg.GetRange(),
					)
				}
			} else {
				p.Error(
					diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
					decorator.GetRange(), "1", fmt.Sprintf("%d", numArgs),
				)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// initializeClass
// ---------------------------------------------------------------------------

// initializeClass initializes a class declaration.
// Ported from: assemblyscript/src/program.ts Program.initializeClass.
func (p *Program) initializeClass(
	declaration *ast.ClassDeclaration,
	parent Element,
	queuedExtends *[]*ClassPrototype,
	queuedImplements *[]*ClassPrototype,
) *ClassPrototype {
	name := declaration.Name.Text
	element := NewClassPrototype(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators,
			DecoratorFlagsGlobal|
				DecoratorFlagsFinal|
				DecoratorFlagsUnmanaged,
		),
		false,
	)
	if !parent.Add(name, element, nil) {
		return nil
	}

	// remember classes that implement interfaces
	implementsTypes := declaration.ImplementsTypes
	if implementsTypes != nil && len(implementsTypes) > 0 {
		numImplementsTypes := len(implementsTypes)
		if element.HasDecorator(DecoratorFlagsUnmanaged) {
			p.Error(
				diagnostics.DiagnosticCodeUnmanagedClassesCannotImplementInterfaces,
				diagnostics.JoinRanges(
					declaration.Name.GetRange(),
					implementsTypes[numImplementsTypes-1].GetRange(),
				),
			)
		} else {
			*queuedImplements = append(*queuedImplements, element)
		}
	}

	// remember classes that extend another class
	if declaration.ExtendsType != nil {
		*queuedExtends = append(*queuedExtends, element)
	} else if !element.HasDecorator(DecoratorFlagsUnmanaged) &&
		element.GetInternalName() != common.BuiltinNameObject {
		element.ImplicitlyExtendsObject = true
	}

	// initialize members
	memberDeclarations := declaration.Members
	for _, memberDeclaration := range memberDeclarations {
		switch memberDeclaration.GetKind() {
		case ast.NodeKindFieldDeclaration:
			p.initializeField(memberDeclaration.(*ast.FieldDeclaration), element)
		case ast.NodeKindMethodDeclaration:
			methodDeclaration := memberDeclaration.(*ast.FunctionDeclaration)
			if (common.CommonFlags(methodDeclaration.Flags) & (common.CommonFlagsGet | common.CommonFlagsSet)) != 0 {
				p.initializeProperty(methodDeclaration, element)
			} else {
				method := p.initializeMethod(methodDeclaration, element)
				if method != nil && methodDeclaration.Name.GetKind() == ast.NodeKindConstructor {
					element.ConstructorPrototype = method
				}
			}
		case ast.NodeKindIndexSignature:
			// ignored for now
		}
	}
	return element
}

// ---------------------------------------------------------------------------
// initializeField
// ---------------------------------------------------------------------------

// initializeField initializes a field of a class or interface.
// Ported from: assemblyscript/src/program.ts Program.initializeField.
func (p *Program) initializeField(
	declaration *ast.FieldDeclaration,
	parent *ClassPrototype,
) {
	name := declaration.Name.Text
	decorators := declaration.Decorators
	acceptedFlags := DecoratorFlags(DecoratorFlagsUnsafe)
	if parent.Is(common.CommonFlagsAmbient) {
		acceptedFlags |= DecoratorFlagsExternal
	}
	if (common.CommonFlags(declaration.Flags) & common.CommonFlagsStatic) != 0 { // global variable
		if parent.GetElementKind() == ElementKindInterfacePrototype {
			panic("static field on interface prototype")
		}
		acceptedFlags |= DecoratorFlagsLazy
		if (common.CommonFlags(declaration.Flags) & common.CommonFlagsReadonly) != 0 {
			acceptedFlags |= DecoratorFlagsInline
		}
		element := NewGlobal(
			name,
			parent,
			p.checkDecorators(decorators, acceptedFlags),
			declaration,
		)
		if !parent.Add(name, element, nil) {
			return
		}
	} else { // actual instance field
		if (common.CommonFlags(declaration.Flags) & (common.CommonFlagsAbstract | common.CommonFlagsGet | common.CommonFlagsSet)) != 0 {
			panic("instance field cannot be abstract, get, or set")
		}
		element := PropertyPrototypeForField(
			name,
			parent,
			declaration,
			p.checkDecorators(decorators, acceptedFlags),
		)
		if !parent.AddInstance(name, element) {
			return
		}
	}
}

// ---------------------------------------------------------------------------
// initializeMethod
// ---------------------------------------------------------------------------

// initializeMethod initializes a method of a class or interface.
// Ported from: assemblyscript/src/program.ts Program.initializeMethod.
func (p *Program) initializeMethod(
	declaration *ast.FunctionDeclaration,
	parent *ClassPrototype,
) *FunctionPrototype {
	name := declaration.Name.Text
	isStatic := (common.CommonFlags(declaration.Flags) & common.CommonFlagsStatic) != 0
	acceptedFlags := DecoratorFlags(DecoratorFlagsInline | DecoratorFlagsUnsafe)
	if (common.CommonFlags(declaration.Flags) & common.CommonFlagsGeneric) == 0 {
		acceptedFlags |= DecoratorFlagsOperatorBinary |
			DecoratorFlagsOperatorPrefix |
			DecoratorFlagsOperatorPostfix
	}
	if parent.Is(common.CommonFlagsAmbient) {
		acceptedFlags |= DecoratorFlagsExternal
	}
	if declaration.GetRange() != nil {
		if src, ok := declaration.GetRange().Source.(*ast.Source); ok && src.IsLibrary() {
			acceptedFlags |= DecoratorFlagsBuiltin
		}
	}
	element := NewFunctionPrototype(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators, acceptedFlags),
	)
	if element.HasDecorator(DecoratorFlagsBuiltin) && !isBuiltinFunction(element.GetInternalName()) {
		p.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			declaration.GetRange(),
			fmt.Sprintf("Builtin '%s'", element.GetInternalName()),
		)
	}
	if isStatic { // global function
		if !parent.Add(name, element, nil) {
			return nil
		}
	} else { // actual instance method
		if !parent.AddInstance(name, element) {
			return nil
		}
	}
	p.checkOperatorOverloads(declaration.Decorators, element, parent)
	return element
}

// isBuiltinFunction checks if a function name is a registered builtin.
func isBuiltinFunction(name string) bool {
	if BuiltinFunctions == nil {
		return false
	}
	_, ok := BuiltinFunctions[name]
	return ok
}

// isBuiltinVariableOnAccess checks if a variable name is a registered builtin on-access.
func isBuiltinVariableOnAccess(name string) bool {
	if BuiltinVariablesOnAccess == nil {
		return false
	}
	_, ok := BuiltinVariablesOnAccess[name]
	return ok
}

// ---------------------------------------------------------------------------
// ensureProperty / initializeProperty
// ---------------------------------------------------------------------------

// ensureProperty ensures that the property introduced by a getter or setter exists.
// Ported from: assemblyscript/src/program.ts Program.ensureProperty.
func (p *Program) ensureProperty(
	declaration *ast.FunctionDeclaration,
	parent *ClassPrototype,
) *PropertyPrototype {
	name := declaration.Name.Text
	if (common.CommonFlags(declaration.Flags) & common.CommonFlagsStatic) != 0 {
		parentMembers := parent.GetMembers()
		if parentMembers != nil {
			if existing, ok := parentMembers[name]; ok {
				if existing.GetElementKind() == ElementKindPropertyPrototype {
					return existing.(*PropertyPrototype)
				}
			}
		}
		// no existing static member
		if parentMembers == nil || parentMembers[name] == nil {
			element := NewPropertyPrototype(name, parent, declaration)
			if !parent.Add(name, element, nil) {
				return nil
			}
			return element
		}
	} else {
		parentMembers := parent.InstanceMembers
		if parentMembers != nil {
			if existing, ok := parentMembers[name]; ok {
				if existing.GetElementKind() == ElementKindPropertyPrototype {
					return existing.(*PropertyPrototype)
				}
			}
		}
		// no existing instance member
		if parentMembers == nil || parentMembers[name] == nil {
			element := NewPropertyPrototype(name, parent, declaration)
			if !parent.AddInstance(name, element) {
				return nil
			}
			return element
		}
	}
	p.Error(
		diagnostics.DiagnosticCodeDuplicateProperty0,
		declaration.Name.GetRange(), name,
	)
	return nil
}

// initializeProperty initializes a property of a class.
// Ported from: assemblyscript/src/program.ts Program.initializeProperty.
func (p *Program) initializeProperty(
	declaration *ast.FunctionDeclaration,
	parent *ClassPrototype,
) {
	property := p.ensureProperty(declaration, parent)
	if property == nil {
		return
	}
	name := declaration.Name.Text
	isGetter := (common.CommonFlags(declaration.Flags) & common.CommonFlagsGet) != 0
	if isGetter {
		if property.GetterPrototype != nil {
			p.Error(
				diagnostics.DiagnosticCodeDuplicateProperty0,
				declaration.Name.GetRange(), name,
			)
			return
		}
	} else {
		if property.SetterPrototype != nil {
			p.Error(
				diagnostics.DiagnosticCodeDuplicateProperty0,
				declaration.Name.GetRange(), name,
			)
			return
		}
	}
	prefix := common.SETTER_PREFIX
	if isGetter {
		prefix = common.GETTER_PREFIX
	}
	element := NewFunctionPrototype(
		prefix+name,
		property.GetParent(), // same level as property
		declaration,
		p.checkDecorators(declaration.Decorators,
			DecoratorFlagsInline|DecoratorFlagsUnsafe,
		),
	)
	if isGetter {
		property.GetterPrototype = element
	} else {
		property.SetterPrototype = element
	}
}

// ---------------------------------------------------------------------------
// initializeEnum / initializeEnumValue
// ---------------------------------------------------------------------------

// initializeEnum initializes an enum.
// Ported from: assemblyscript/src/program.ts Program.initializeEnum.
func (p *Program) initializeEnum(
	declaration *ast.EnumDeclaration,
	parent Element,
) *Enum {
	name := declaration.Name.Text
	element := NewEnum(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators,
			DecoratorFlagsGlobal|
				DecoratorFlagsInline|
				DecoratorFlagsLazy,
		),
	)
	if !parent.Add(name, element, nil) {
		return nil
	}
	values := declaration.Values
	for _, value := range values {
		p.initializeEnumValue(value, element)
	}
	return element
}

// initializeEnumValue initializes an enum value.
// Ported from: assemblyscript/src/program.ts Program.initializeEnumValue.
func (p *Program) initializeEnumValue(
	declaration *ast.EnumValueDeclaration,
	parent *Enum,
) {
	name := declaration.Name.Text
	element := NewEnumValue(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators, DecoratorFlagsNone),
	)
	parent.Add(name, element, nil)
}

// ---------------------------------------------------------------------------
// initializeExports / initializeExport / initializeExportDefault
// ---------------------------------------------------------------------------

// initializeExports initializes an `export` statement.
// Ported from: assemblyscript/src/program.ts Program.initializeExports.
func (p *Program) initializeExports(
	statement *ast.ExportStatement,
	parent *File,
	queuedExports map[*File]map[string]*QueuedExport,
	queuedExportsStar map[*File][]*QueuedExportStar,
) {
	members := statement.Members
	if members != nil { // export { foo, bar } [from "./baz"]
		for _, member := range members {
			internalPath := ""
			if statement.HasInternal {
				internalPath = statement.InternalPath
			}
			p.initializeExport(member, parent, internalPath, queuedExports)
		}
	} else { // export * from "./baz"
		foreignPath := statement.InternalPath // must be set for export *
		foreignPathAlt := foreignPath
		if strings.HasSuffix(foreignPath, common.INDEX_SUFFIX) {
			foreignPathAlt = foreignPath[:len(foreignPath)-len(common.INDEX_SUFFIX)]
		} else {
			foreignPathAlt = foreignPath + common.INDEX_SUFFIX
		}
		if queuedExportsStar[parent] == nil {
			queuedExportsStar[parent] = make([]*QueuedExportStar, 0)
		}
		queuedExportsStar[parent] = append(queuedExportsStar[parent], &QueuedExportStar{
			ForeignPath:    foreignPath,
			ForeignPathAlt: foreignPathAlt,
			PathLiteral:    statement.Path,
		})
	}
}

// initializeExport initializes a single `export` member.
// Ported from: assemblyscript/src/program.ts Program.initializeExport.
func (p *Program) initializeExport(
	member *ast.ExportMember,
	localFile *File,
	foreignPath string,
	queuedExports map[*File]map[string]*QueuedExport,
) {
	localName := member.LocalName.Text
	foreignName := member.ExportedName.Text

	// check for duplicates
	element := localFile.LookupExport(foreignName)
	if element != nil {
		p.Error(
			diagnostics.DiagnosticCodeExportDeclarationConflictsWithExportedDeclarationOf0,
			member.ExportedName.GetRange(), foreignName,
		)
		return
	}

	// local element, i.e. export { foo [as bar] }
	if foreignPath == "" {
		// resolve right away if the local element already exists
		if localElement := localFile.GetMember(localName); localElement != nil {
			localFile.EnsureExport(foreignName, localElement)
		} else {
			// otherwise queue it
			if queuedExports[localFile] == nil {
				queuedExports[localFile] = make(map[string]*QueuedExport)
			}
			queuedExports[localFile][foreignName] = &QueuedExport{
				LocalIdentifier:   member.LocalName,
				ForeignIdentifier: member.ExportedName,
				ForeignPath:       "",
				ForeignPathAlt:    "",
			}
		}
	} else {
		// foreign element, i.e. export { foo } from "./bar"
		foreignPathAlt := foreignPath
		if strings.HasSuffix(foreignPath, common.INDEX_SUFFIX) {
			foreignPathAlt = foreignPath[:len(foreignPath)-len(common.INDEX_SUFFIX)]
		} else {
			foreignPathAlt = foreignPath + common.INDEX_SUFFIX
		}
		if queuedExports[localFile] == nil {
			queuedExports[localFile] = make(map[string]*QueuedExport)
		}
		queuedExports[localFile][foreignName] = &QueuedExport{
			LocalIdentifier:   member.LocalName,
			ForeignIdentifier: member.ExportedName,
			ForeignPath:       foreignPath,
			ForeignPathAlt:    foreignPathAlt,
		}
	}
}

// initializeExportDefault initializes an `export default` statement.
// Ported from: assemblyscript/src/program.ts Program.initializeExportDefault.
func (p *Program) initializeExportDefault(
	statement *ast.ExportDefaultStatement,
	parent *File,
	queuedExtends *[]*ClassPrototype,
	queuedImplements *[]*ClassPrototype,
) {
	declaration := statement.Declaration
	var element DeclaredElement
	switch declaration.GetKind() {
	case ast.NodeKindEnumDeclaration:
		element = p.initializeEnum(declaration.(*ast.EnumDeclaration), parent)
	case ast.NodeKindFunctionDeclaration:
		element = p.initializeFunction(declaration.(*ast.FunctionDeclaration), parent)
	case ast.NodeKindClassDeclaration:
		element = p.initializeClass(declaration.(*ast.ClassDeclaration), parent, queuedExtends, queuedImplements)
	case ast.NodeKindInterfaceDeclaration:
		element = p.initializeInterface(declaration.(*ast.ClassDeclaration), parent, queuedExtends)
	case ast.NodeKindNamespaceDeclaration:
		element = p.initializeNamespace(declaration.(*ast.NamespaceDeclaration), parent, queuedExtends, queuedImplements)
	}
	if element != nil {
		if parent.Exports == nil {
			parent.Exports = make(map[string]DeclaredElement)
		} else {
			if existing, ok := parent.Exports["default"]; ok {
				p.ErrorRelated(
					diagnostics.DiagnosticCodeDuplicateIdentifier0,
					getDeclName(declaration).GetRange(),
					existing.IdentifierNode().GetRange(),
					"default",
				)
				return
			}
		}
		parent.Exports["default"] = element
	}
}

// getDeclName extracts the Name IdentifierExpression from a declaration node.
func getDeclName(node ast.Node) *ast.IdentifierExpression {
	switch decl := node.(type) {
	case *ast.ClassDeclaration:
		return decl.Name
	case *ast.FunctionDeclaration:
		return decl.Name
	case *ast.EnumDeclaration:
		return decl.Name
	case *ast.NamespaceDeclaration:
		return decl.Name
	case *ast.TypeDeclaration:
		return decl.Name
	case *ast.VariableDeclaration:
		return decl.Name
	}
	return nil
}

// ---------------------------------------------------------------------------
// initializeImports / initializeImport
// ---------------------------------------------------------------------------

// initializeImports initializes an `import` statement.
// Ported from: assemblyscript/src/program.ts Program.initializeImports.
func (p *Program) initializeImports(
	statement *ast.ImportStatement,
	parent *File,
	queuedImports *[]*QueuedImport,
	queuedExports map[*File]map[string]*QueuedExport,
) {
	declarations := statement.Declarations
	if declarations != nil { // import { foo [as bar] } from "./baz"
		for _, decl := range declarations {
			p.initializeImport(
				decl,
				parent,
				statement.InternalPath,
				queuedImports,
				queuedExports,
			)
		}
	} else {
		namespaceName := statement.NamespaceName
		if namespaceName != nil { // import * as foo from "./bar"
			*queuedImports = append(*queuedImports, &QueuedImport{
				LocalFile:         parent,
				LocalIdentifier:   namespaceName,
				ForeignIdentifier: nil, // indicates import *
				ForeignPath:       statement.InternalPath,
				ForeignPathAlt:    statement.InternalPath + common.INDEX_SUFFIX,
			})
		}
		// else: import "./foo" (side-effect only)
	}
}

// initializeImport initializes a single `import` declaration.
// Ported from: assemblyscript/src/program.ts Program.initializeImport.
func (p *Program) initializeImport(
	declaration *ast.ImportDeclaration,
	parent *File,
	foreignPath string,
	queuedImports *[]*QueuedImport,
	queuedExports map[*File]map[string]*QueuedExport,
) {
	foreignPathAlt := foreignPath
	if strings.HasSuffix(foreignPath, common.INDEX_SUFFIX) {
		foreignPathAlt = foreignPath[:len(foreignPath)-len(common.INDEX_SUFFIX)]
	} else {
		foreignPathAlt = foreignPath + common.INDEX_SUFFIX
	}

	// resolve right away if the element exists
	foreignFile := p.lookupForeignFile(foreignPath, foreignPathAlt)
	if foreignFile != nil {
		element := p.lookupForeign(declaration.ForeignName.Text, foreignFile, queuedExports)
		if element != nil {
			parent.Add(declaration.Name.Text, element, declaration.Name)
			return
		}
	}

	// otherwise queue it
	*queuedImports = append(*queuedImports, &QueuedImport{
		LocalFile:         parent,
		LocalIdentifier:   declaration.Name,
		ForeignIdentifier: declaration.ForeignName,
		ForeignPath:       foreignPath,
		ForeignPathAlt:    foreignPathAlt,
	})
}

// ---------------------------------------------------------------------------
// initializeFunction
// ---------------------------------------------------------------------------

// initializeFunction initializes a function. Does not handle methods.
// Ported from: assemblyscript/src/program.ts Program.initializeFunction.
func (p *Program) initializeFunction(
	declaration *ast.FunctionDeclaration,
	parent Element,
) *FunctionPrototype {
	name := declaration.Name.Text
	validDecorators := DecoratorFlags(DecoratorFlagsUnsafe)
	if (common.CommonFlags(declaration.Flags) & common.CommonFlagsAmbient) != 0 {
		validDecorators |= DecoratorFlagsExternal | DecoratorFlagsExternalJs
	} else {
		validDecorators |= DecoratorFlagsInline
		isLibrary := false
		if declaration.GetRange() != nil {
			if src, ok := declaration.GetRange().Source.(*ast.Source); ok {
				isLibrary = src.IsLibrary()
			}
		}
		if isLibrary || (common.CommonFlags(declaration.Flags)&common.CommonFlagsExport) != 0 {
			validDecorators |= DecoratorFlagsLazy
		}
	}
	if (common.CommonFlags(declaration.Flags) & common.CommonFlagsInstance) == 0 {
		if parent.GetElementKind() != ElementKindClassPrototype {
			validDecorators |= DecoratorFlagsGlobal
		}
	}
	isLibrary := false
	if declaration.GetRange() != nil {
		if src, ok := declaration.GetRange().Source.(*ast.Source); ok {
			isLibrary = src.IsLibrary()
		}
	}
	if isLibrary {
		validDecorators |= DecoratorFlagsBuiltin
	}
	element := NewFunctionPrototype(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators, validDecorators),
	)
	if element.HasDecorator(DecoratorFlagsBuiltin) && !isBuiltinFunction(element.GetInternalName()) {
		p.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			declaration.GetRange(),
			fmt.Sprintf("Builtin '%s'", element.GetInternalName()),
		)
	}
	if !parent.Add(name, element, nil) {
		return nil
	}
	return element
}

// ---------------------------------------------------------------------------
// initializeInterface
// ---------------------------------------------------------------------------

// initializeInterface initializes an interface.
// Ported from: assemblyscript/src/program.ts Program.initializeInterface.
func (p *Program) initializeInterface(
	declaration *ast.ClassDeclaration,
	parent Element,
	queuedExtends *[]*ClassPrototype,
) *InterfacePrototype {
	name := declaration.Name.Text
	element := NewInterfacePrototype(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators,
			DecoratorFlagsGlobal,
		),
	)
	if !parent.Add(name, element, nil) {
		return nil
	}

	// remember interfaces that extend another interface
	if declaration.ExtendsType != nil {
		*queuedExtends = append(*queuedExtends, &element.ClassPrototype)
	}

	memberDeclarations := declaration.Members
	for _, memberDeclaration := range memberDeclarations {
		switch memberDeclaration.GetKind() {
		case ast.NodeKindFieldDeclaration:
			p.initializeFieldAsProperty(memberDeclaration.(*ast.FieldDeclaration), element)
		case ast.NodeKindMethodDeclaration:
			methodDeclaration := memberDeclaration.(*ast.FunctionDeclaration)
			if (common.CommonFlags(methodDeclaration.Flags) & (common.CommonFlagsGet | common.CommonFlagsSet)) != 0 {
				p.initializeProperty(methodDeclaration, &element.ClassPrototype)
			} else {
				p.initializeMethod(methodDeclaration, &element.ClassPrototype)
			}
		}
	}
	return element
}

// ---------------------------------------------------------------------------
// initializeFieldAsProperty
// ---------------------------------------------------------------------------

// initializeFieldAsProperty initializes a field of an interface, as a property.
// Ported from: assemblyscript/src/program.ts Program.initializeFieldAsProperty.
func (p *Program) initializeFieldAsProperty(
	declaration *ast.FieldDeclaration,
	parent *InterfacePrototype,
) {
	initializer := declaration.Initializer
	if initializer != nil {
		p.Error(diagnostics.DiagnosticCodeAnInterfacePropertyCannotHaveAnInitializer, initializer.GetRange())
	}
	typeNode := declaration.Type
	if typeNode == nil {
		typeNode = ast.NewOmittedType(*declaration.Name.GetRange().AtEnd())
	}

	// Create getter declaration (use declaration.GetRange() matching TS)
	declRange := declaration.GetRange()
	getterDeclaration := ast.NewMethodDeclaration(
		declaration.Name,
		declaration.Decorators,
		declaration.Flags|int32(common.CommonFlagsGet),
		nil,
		ast.NewFunctionTypeNode(nil, typeNode, nil, false, *declRange),
		nil,
		*declRange,
	)
	p.initializeProperty(getterDeclaration, &parent.ClassPrototype)

	if (common.CommonFlags(declaration.Flags) & common.CommonFlagsReadonly) == 0 {
		// Create setter declaration
		nameRangeAtEnd := declaration.Name.GetRange().AtEnd()
		setterDeclaration := ast.NewMethodDeclaration(
			declaration.Name,
			declaration.Decorators,
			declaration.Flags|int32(common.CommonFlagsSet),
			nil,
			ast.NewFunctionTypeNode(
				[]*ast.ParameterNode{
					ast.NewParameterNode(ast.ParameterKindDefault, declaration.Name, typeNode, nil, *declRange),
				},
				ast.NewOmittedType(*nameRangeAtEnd),
				nil,
				false,
				*declRange,
			),
			nil,
			*declRange,
		)
		p.initializeProperty(setterDeclaration, &parent.ClassPrototype)
	}
}

// ---------------------------------------------------------------------------
// initializeNamespace
// ---------------------------------------------------------------------------

// initializeNamespace initializes a namespace.
// Ported from: assemblyscript/src/program.ts Program.initializeNamespace.
func (p *Program) initializeNamespace(
	declaration *ast.NamespaceDeclaration,
	parent Element,
	queuedExtends *[]*ClassPrototype,
	queuedImplements *[]*ClassPrototype,
) DeclaredElement {
	name := declaration.Name.Text
	original := NewNamespace(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators, DecoratorFlagsGlobal),
	)
	if !parent.Add(name, original, nil) {
		return nil
	}
	element := parent.GetMember(name) // possibly merged
	members := declaration.Members
	for _, member := range members {
		switch member.GetKind() {
		case ast.NodeKindClassDeclaration:
			p.initializeClass(member.(*ast.ClassDeclaration), original, queuedExtends, queuedImplements)
		case ast.NodeKindEnumDeclaration:
			p.initializeEnum(member.(*ast.EnumDeclaration), original)
		case ast.NodeKindFunctionDeclaration:
			p.initializeFunction(member.(*ast.FunctionDeclaration), original)
		case ast.NodeKindInterfaceDeclaration:
			p.initializeInterface(member.(*ast.ClassDeclaration), original, queuedExtends)
		case ast.NodeKindNamespaceDeclaration:
			p.initializeNamespace(member.(*ast.NamespaceDeclaration), original, queuedExtends, queuedImplements)
		case ast.NodeKindTypeDeclaration:
			p.initializeTypeDefinition(member.(*ast.TypeDeclaration), original)
		case ast.NodeKindVariable:
			p.initializeVariables(member.(*ast.VariableStatement), original)
		}
	}
	if original != element {
		CopyMembers(original, element) // keep original parent
	}
	return element
}

// ---------------------------------------------------------------------------
// initializeTypeDefinition
// ---------------------------------------------------------------------------

// initializeTypeDefinition initializes a `type` definition.
// Ported from: assemblyscript/src/program.ts Program.initializeTypeDefinition.
func (p *Program) initializeTypeDefinition(
	declaration *ast.TypeDeclaration,
	parent Element,
) {
	name := declaration.Name.Text
	element := NewTypeDefinition(
		name,
		parent,
		declaration,
		p.checkDecorators(declaration.Decorators, DecoratorFlagsNone),
	)
	parent.Add(name, element, nil)
}

// ---------------------------------------------------------------------------
// initializeVariables
// ---------------------------------------------------------------------------

// initializeVariables initializes a variable statement.
// Ported from: assemblyscript/src/program.ts Program.initializeVariables.
func (p *Program) initializeVariables(
	statement *ast.VariableStatement,
	parent Element,
) {
	declarations := statement.Declarations
	for _, declaration := range declarations {
		name := declaration.Name.Text
		acceptedFlags := DecoratorFlags(DecoratorFlagsGlobal | DecoratorFlagsLazy)
		if (common.CommonFlags(declaration.Flags) & common.CommonFlagsAmbient) != 0 {
			acceptedFlags |= DecoratorFlagsExternal
		}
		if (common.CommonFlags(declaration.Flags) & common.CommonFlagsConst) != 0 {
			acceptedFlags |= DecoratorFlagsInline
		}
		isLibrary := false
		if declaration.GetRange() != nil {
			if src, ok := declaration.GetRange().Source.(*ast.Source); ok {
				isLibrary = src.IsLibrary()
			}
		}
		if isLibrary {
			acceptedFlags |= DecoratorFlagsBuiltin
		}
		element := NewGlobal(
			name,
			parent,
			p.checkDecorators(declaration.Decorators, acceptedFlags),
			declaration,
		)
		if element.HasDecorator(DecoratorFlagsBuiltin) && !isBuiltinVariableOnAccess(element.GetInternalName()) {
			p.Error(
				diagnostics.DiagnosticCodeNotImplemented0,
				declaration.GetRange(),
				fmt.Sprintf("Builtin '%s'", element.GetInternalName()),
			)
		}
		parent.Add(name, element, nil)
	}
}

// ---------------------------------------------------------------------------
// Override processing
// ---------------------------------------------------------------------------

// processOverrides processes overridden members by this class in a base class.
// Ported from: assemblyscript/src/program.ts Program.processOverrides.
func (p *Program) processOverrides(
	thisPrototype *ClassPrototype,
	basePrototype *ClassPrototype,
) {
	thisInstanceMembers := thisPrototype.InstanceMembers
	if thisInstanceMembers == nil {
		return
	}
	// Collect members slice
	thisMembers := make([]DeclaredElement, 0, len(thisInstanceMembers))
	for _, member := range thisInstanceMembers {
		thisMembers = append(thisMembers, member)
	}
	seen := make(map[*ClassPrototype]struct{})
	for {
		baseInstanceMembers := basePrototype.InstanceMembers
		if baseInstanceMembers != nil {
			for _, thisMember := range thisMembers {
				if baseMember, ok := baseInstanceMembers[thisMember.GetName()]; ok {
					p.doProcessOverride(thisPrototype, thisMember, basePrototype, baseMember)
				}
			}
		}
		// A class can have a base class and multiple interfaces, but from the
		// base member alone we only get one. Make sure we don't miss any.
		baseInterfacePrototypes := basePrototype.InterfacePrototypes
		if baseInterfacePrototypes != nil {
			for _, baseInterfacePrototype := range baseInterfacePrototypes {
				if baseInterfacePrototype != (*InterfacePrototype)(nil) && &baseInterfacePrototype.ClassPrototype != basePrototype {
					p.processOverrides(thisPrototype, &baseInterfacePrototype.ClassPrototype)
				}
			}
		}
		nextPrototype := basePrototype.BasePrototype
		if nextPrototype == nil {
			break
		}
		// Break on circular inheritance. Is diagnosed later, when resolved.
		seen[basePrototype] = struct{}{}
		if _, ok := seen[nextPrototype]; ok {
			break
		}
		basePrototype = nextPrototype
	}
}

// doProcessOverride processes a single overridden member.
// Ported from: assemblyscript/src/program.ts Program.doProcessOverride.
func (p *Program) doProcessOverride(
	thisClass *ClassPrototype,
	thisMember DeclaredElement,
	baseClass *ClassPrototype,
	baseMember DeclaredElement,
) {
	// Constructors and private members do not override
	if thisMember.IsAny(common.CommonFlagsConstructor | common.CommonFlagsPrivate) {
		return
	}
	if thisMember.GetElementKind() == ElementKindFunctionPrototype &&
		baseMember.GetElementKind() == ElementKindFunctionPrototype {
		thisMethod := thisMember.(*FunctionPrototype)
		baseMethod := baseMember.(*FunctionPrototype)
		if !thisMethod.VisibilityEquals(baseMethod) {
			p.ErrorRelated(
				diagnostics.DiagnosticCodeOverloadSignaturesMustAllBePublicPrivateOrProtected,
				thisMethod.IdentifierNode().GetRange(),
				baseMethod.IdentifierNode().GetRange(),
			)
		}
		baseMember.Set(common.CommonFlagsOverridden)
		if baseMethod.UnboundOverrides == nil {
			baseMethod.UnboundOverrides = make(map[*FunctionPrototype]struct{})
		}
		baseMethod.UnboundOverrides[thisMethod] = struct{}{}
		if baseMethod.Instances != nil {
			for _, baseMethodInstance := range baseMethod.Instances {
				baseMethodInstance.Set(common.CommonFlagsOverridden)
			}
		}
	} else if thisMember.GetElementKind() == ElementKindPropertyPrototype &&
		baseMember.GetElementKind() == ElementKindPropertyPrototype {
		thisProperty := thisMember.(*PropertyPrototype)
		baseProperty := baseMember.(*PropertyPrototype)
		if !thisProperty.VisibilityEquals(baseProperty) {
			p.ErrorRelated(
				diagnostics.DiagnosticCodeOverloadSignaturesMustAllBePublicPrivateOrProtected,
				thisProperty.IdentifierNode().GetRange(),
				baseProperty.IdentifierNode().GetRange(),
			)
		}
		if baseProperty.GetParent().GetElementKind() != ElementKindInterfacePrototype {
			// Interface fields/properties can be implemented by either, but other
			// members must match to retain compatibility with TS/JS.
			thisIsField := thisProperty.IsField()
			if thisIsField != baseProperty.IsField() {
				if thisIsField { // base is property
					p.ErrorRelated(
						diagnostics.DiagnosticCode0IsDefinedAsAnAccessorInClass1ButIsOverriddenHereIn2AsAnInstanceProperty,
						thisProperty.IdentifierNode().GetRange(),
						baseProperty.IdentifierNode().GetRange(),
						thisProperty.GetName(), baseClass.GetInternalName(), thisClass.GetInternalName(),
					)
				} else { // this is property, base is field
					p.ErrorRelated(
						diagnostics.DiagnosticCode0IsDefinedAsAPropertyInClass1ButIsOverriddenHereIn2AsAnAccessor,
						thisProperty.IdentifierNode().GetRange(),
						baseProperty.IdentifierNode().GetRange(),
						thisProperty.GetName(), baseClass.GetInternalName(), thisClass.GetInternalName(),
					)
				}
				return
			} else if thisIsField { // base is also field
				// Fields don't override other fields and can only be redeclared
				return
			}
		}
		baseProperty.Set(common.CommonFlagsOverridden)
		baseGetter := baseProperty.GetterPrototype
		if baseGetter != nil {
			baseGetter.Set(common.CommonFlagsOverridden)
			thisGetter := thisProperty.GetterPrototype
			if thisGetter != nil {
				if baseGetter.UnboundOverrides == nil {
					baseGetter.UnboundOverrides = make(map[*FunctionPrototype]struct{})
				}
				baseGetter.UnboundOverrides[thisGetter] = struct{}{}
			}
			if baseGetter.Instances != nil {
				for _, baseGetterInstance := range baseGetter.Instances {
					baseGetterInstance.Set(common.CommonFlagsOverridden)
				}
			}
		}
		baseSetter := baseProperty.SetterPrototype
		if baseSetter != nil && thisProperty.SetterPrototype != nil {
			baseSetter.Set(common.CommonFlagsOverridden)
			thisSetter := thisProperty.SetterPrototype
			if thisSetter != nil {
				if baseSetter.UnboundOverrides == nil {
					baseSetter.UnboundOverrides = make(map[*FunctionPrototype]struct{})
				}
				baseSetter.UnboundOverrides[thisSetter] = struct{}{}
			}
			if baseSetter.Instances != nil {
				for _, baseSetterInstance := range baseSetter.Instances {
					baseSetterInstance.Set(common.CommonFlagsOverridden)
				}
			}
		}
	} else {
		p.ErrorRelated(
			diagnostics.DiagnosticCodeProperty0InType1IsNotAssignableToTheSamePropertyInBaseType2,
			thisMember.IdentifierNode().GetRange(),
			baseMember.IdentifierNode().GetRange(),
			thisMember.GetName(), thisClass.GetInternalName(), baseClass.GetInternalName(),
		)
	}
}

// ---------------------------------------------------------------------------
// Module export marking
// ---------------------------------------------------------------------------

// markModuleExports marks all exports of the specified file as module exports.
// Ported from: assemblyscript/src/program.ts Program.markModuleExports.
func (p *Program) markModuleExports(file *File) {
	if file.Exports != nil {
		for _, element := range file.Exports {
			p.markModuleExport(element)
		}
	}
	if file.ExportsStar != nil {
		for _, starFile := range file.ExportsStar {
			p.markModuleExports(starFile)
		}
	}
}

// markModuleExport marks an element and its children as a module export.
// Ported from: assemblyscript/src/program.ts Program.markModuleExport.
func (p *Program) markModuleExport(element Element) {
	element.Set(common.CommonFlagsModuleExport)
	switch element.GetElementKind() {
	case ElementKindClassPrototype:
		cp := element.(*ClassPrototype)
		if cp.InstanceMembers != nil {
			for _, member := range cp.InstanceMembers {
				p.markModuleExport(member)
			}
		}
	case ElementKindPropertyPrototype:
		pp := element.(*PropertyPrototype)
		if pp.GetterPrototype != nil {
			p.markModuleExport(pp.GetterPrototype)
		}
		if pp.SetterPrototype != nil {
			p.markModuleExport(pp.SetterPrototype)
		}
	}
	staticMembers := element.GetMembers()
	if staticMembers != nil {
		for _, member := range staticMembers {
			p.markModuleExport(member)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers for getting node flags from ast.Node
// ---------------------------------------------------------------------------

// getNodeFlags extracts the Flags field from a declaration node.
func getNodeFlags(node ast.Node) int32 {
	switch decl := node.(type) {
	case *ast.ClassDeclaration:
		return decl.Flags
	case *ast.FunctionDeclaration:
		return decl.Flags
	case *ast.FieldDeclaration:
		return decl.Flags
	case *ast.EnumDeclaration:
		return decl.Flags
	case *ast.NamespaceDeclaration:
		return decl.Flags
	case *ast.VariableDeclaration:
		return decl.Flags
	case *ast.TypeDeclaration:
		return decl.Flags
	case *ast.EnumValueDeclaration:
		return decl.Flags
	}
	return 0
}
