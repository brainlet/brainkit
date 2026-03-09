package ast

import (
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// --- Declaration base ---

// DeclarationBase provides common fields for declaration statements.
type DeclarationBase struct {
	NodeBase
	Name                 *IdentifierExpression
	Decorators           []*DecoratorNode
	Flags                int32 // CommonFlags
	OverriddenModuleName string // empty string means nil/unset
	HasOverriddenModule  bool
}

// Is tests if this declaration has the specified flag or flags.
func (d *DeclarationBase) Is(flag int32) bool { return (d.Flags & flag) == flag }

// IsAny tests if this declaration has one of the specified flags.
func (d *DeclarationBase) IsAny(flag int32) bool { return (d.Flags & flag) != 0 }

// Set sets a specific flag or flags on this declaration.
func (d *DeclarationBase) Set(flag int32) { d.Flags |= flag }

// --- Index signature ---

// IndexSignatureNode represents an index signature.
type IndexSignatureNode struct {
	NodeBase
	KeyType   *NamedTypeNode
	ValueType Node // TypeNode
	Flags     int32
}

// NewIndexSignatureNode creates an index signature node.
func NewIndexSignatureNode(keyType *NamedTypeNode, valueType Node, flags int32, rng diagnostics.Range) *IndexSignatureNode {
	return &IndexSignatureNode{
		NodeBase:  NodeBase{Kind: NodeKindIndexSignature, Range: rng},
		KeyType:   keyType,
		ValueType: valueType,
		Flags:     flags,
	}
}

// --- Statements ---

// BlockStatement represents a block statement.
type BlockStatement struct {
	NodeBase
	Statements []Node // Statement elements
}

// NewBlockStatement creates a block statement.
func NewBlockStatement(statements []Node, rng diagnostics.Range) *BlockStatement {
	return &BlockStatement{
		NodeBase:   NodeBase{Kind: NodeKindBlock, Range: rng},
		Statements: statements,
	}
}

// BreakStatement represents a `break` statement.
type BreakStatement struct {
	NodeBase
	Label *IdentifierExpression // or nil
}

// NewBreakStatement creates a break statement.
func NewBreakStatement(label *IdentifierExpression, rng diagnostics.Range) *BreakStatement {
	return &BreakStatement{
		NodeBase: NodeBase{Kind: NodeKindBreak, Range: rng},
		Label:    label,
	}
}

// ContinueStatement represents a `continue` statement.
type ContinueStatement struct {
	NodeBase
	Label *IdentifierExpression // or nil
}

// NewContinueStatement creates a continue statement.
func NewContinueStatement(label *IdentifierExpression, rng diagnostics.Range) *ContinueStatement {
	return &ContinueStatement{
		NodeBase: NodeBase{Kind: NodeKindContinue, Range: rng},
		Label:    label,
	}
}

// DoStatement represents a `do` statement.
type DoStatement struct {
	NodeBase
	Body      Node // Statement
	Condition Node // Expression
}

// NewDoStatement creates a do statement.
func NewDoStatement(body Node, condition Node, rng diagnostics.Range) *DoStatement {
	return &DoStatement{
		NodeBase:  NodeBase{Kind: NodeKindDo, Range: rng},
		Body:      body,
		Condition: condition,
	}
}

// EmptyStatement represents an empty statement.
type EmptyStatement struct {
	NodeBase
}

// NewEmptyStatement creates an empty statement.
func NewEmptyStatement(rng diagnostics.Range) *EmptyStatement {
	return &EmptyStatement{
		NodeBase: NodeBase{Kind: NodeKindEmpty, Range: rng},
	}
}

// ExpressionStatement represents an expression used as a statement.
type ExpressionStatement struct {
	NodeBase
	Expression Node // Expression
}

// NewExpressionStatement creates an expression statement.
func NewExpressionStatement(expression Node) *ExpressionStatement {
	return &ExpressionStatement{
		NodeBase:   NodeBase{Kind: NodeKindExpression, Range: *expression.GetRange()},
		Expression: expression,
	}
}

// ForStatement represents a `for` statement.
type ForStatement struct {
	NodeBase
	Initializer Node // Statement, or nil
	Condition   Node // Expression, or nil
	Incrementor Node // Expression, or nil
	Body        Node // Statement
}

// NewForStatement creates a for statement.
func NewForStatement(initializer Node, condition Node, incrementor Node, body Node, rng diagnostics.Range) *ForStatement {
	return &ForStatement{
		NodeBase:    NodeBase{Kind: NodeKindFor, Range: rng},
		Initializer: initializer,
		Condition:   condition,
		Incrementor: incrementor,
		Body:        body,
	}
}

// ForOfStatement represents a `for..of` statement.
type ForOfStatement struct {
	NodeBase
	Variable Node // Statement
	Iterable Node // Expression
	Body     Node // Statement
}

// NewForOfStatement creates a for-of statement.
func NewForOfStatement(variable Node, iterable Node, body Node, rng diagnostics.Range) *ForOfStatement {
	return &ForOfStatement{
		NodeBase: NodeBase{Kind: NodeKindForOf, Range: rng},
		Variable: variable,
		Iterable: iterable,
		Body:     body,
	}
}

// IfStatement represents an `if` statement.
type IfStatement struct {
	NodeBase
	Condition Node // Expression
	IfTrue    Node // Statement
	IfFalse   Node // Statement, or nil
}

// NewIfStatement creates an if statement.
func NewIfStatement(condition Node, ifTrue Node, ifFalse Node, rng diagnostics.Range) *IfStatement {
	return &IfStatement{
		NodeBase:  NodeBase{Kind: NodeKindIf, Range: rng},
		Condition: condition,
		IfTrue:    ifTrue,
		IfFalse:   ifFalse,
	}
}

// ReturnStatement represents a `return` statement.
type ReturnStatement struct {
	NodeBase
	Value Node // Expression, or nil
}

// NewReturnStatement creates a return statement.
func NewReturnStatement(value Node, rng diagnostics.Range) *ReturnStatement {
	return &ReturnStatement{
		NodeBase: NodeBase{Kind: NodeKindReturn, Range: rng},
		Value:    value,
	}
}

// SwitchCase represents a single `case` within a `switch` statement.
type SwitchCase struct {
	NodeBase
	Label      Node   // Expression, or nil (nil = default case)
	Statements []Node // Statement elements
}

// NewSwitchCase creates a switch case.
func NewSwitchCase(label Node, statements []Node, rng diagnostics.Range) *SwitchCase {
	return &SwitchCase{
		NodeBase:   NodeBase{Kind: NodeKindSwitchCase, Range: rng},
		Label:      label,
		Statements: statements,
	}
}

// IsDefault returns true if this is the default case.
func (s *SwitchCase) IsDefault() bool {
	return s.Label == nil
}

// SwitchStatement represents a `switch` statement.
type SwitchStatement struct {
	NodeBase
	Condition Node          // Expression
	Cases     []*SwitchCase
}

// NewSwitchStatement creates a switch statement.
func NewSwitchStatement(condition Node, cases []*SwitchCase, rng diagnostics.Range) *SwitchStatement {
	return &SwitchStatement{
		NodeBase:  NodeBase{Kind: NodeKindSwitch, Range: rng},
		Condition: condition,
		Cases:     cases,
	}
}

// ThrowStatement represents a `throw` statement.
type ThrowStatement struct {
	NodeBase
	Value Node // Expression
}

// NewThrowStatement creates a throw statement.
func NewThrowStatement(value Node, rng diagnostics.Range) *ThrowStatement {
	return &ThrowStatement{
		NodeBase: NodeBase{Kind: NodeKindThrow, Range: rng},
		Value:    value,
	}
}

// TryStatement represents a `try` statement.
type TryStatement struct {
	NodeBase
	BodyStatements    []Node                // Statement elements
	CatchVariable     *IdentifierExpression // or nil
	CatchStatements   []Node                // Statement elements, or nil
	FinallyStatements []Node                // Statement elements, or nil
}

// NewTryStatement creates a try statement.
func NewTryStatement(bodyStatements []Node, catchVariable *IdentifierExpression, catchStatements []Node, finallyStatements []Node, rng diagnostics.Range) *TryStatement {
	return &TryStatement{
		NodeBase:          NodeBase{Kind: NodeKindTry, Range: rng},
		BodyStatements:    bodyStatements,
		CatchVariable:     catchVariable,
		CatchStatements:   catchStatements,
		FinallyStatements: finallyStatements,
	}
}

// VoidStatement represents a void statement dropping an expression's value.
type VoidStatement struct {
	NodeBase
	Expression Node // Expression
}

// NewVoidStatement creates a void statement.
func NewVoidStatement(expression Node, rng diagnostics.Range) *VoidStatement {
	return &VoidStatement{
		NodeBase:   NodeBase{Kind: NodeKindVoid, Range: rng},
		Expression: expression,
	}
}

// WhileStatement represents a `while` statement.
type WhileStatement struct {
	NodeBase
	Condition Node // Expression
	Body      Node // Statement
}

// NewWhileStatement creates a while statement.
func NewWhileStatement(condition Node, body Node, rng diagnostics.Range) *WhileStatement {
	return &WhileStatement{
		NodeBase:  NodeBase{Kind: NodeKindWhile, Range: rng},
		Condition: condition,
		Body:      body,
	}
}

// ModuleDeclaration represents a `module` statement.
type ModuleDeclaration struct {
	NodeBase
	ModuleName string
	Flags      int32 // CommonFlags
}

// NewModuleDeclaration creates a module declaration.
func NewModuleDeclaration(name string, flags int32, rng diagnostics.Range) *ModuleDeclaration {
	return &ModuleDeclaration{
		NodeBase:   NodeBase{Kind: NodeKindModule, Range: rng},
		ModuleName: name,
		Flags:      flags,
	}
}

// --- Declaration statements ---

// ClassDeclaration represents a `class` declaration.
type ClassDeclaration struct {
	DeclarationBase
	TypeParameters []*TypeParameterNode
	ExtendsType    *NamedTypeNode
	ImplementsTypes []*NamedTypeNode
	Members        []Node // DeclarationStatement elements
	IndexSignature *IndexSignatureNode
}

// NewClassDeclaration creates a class declaration.
func NewClassDeclaration(
	name *IdentifierExpression,
	decorators []*DecoratorNode,
	flags int32,
	typeParameters []*TypeParameterNode,
	extendsType *NamedTypeNode,
	implementsTypes []*NamedTypeNode,
	members []Node,
	rng diagnostics.Range,
) *ClassDeclaration {
	return &ClassDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindClassDeclaration, Range: rng},
			Name:       name,
			Decorators: decorators,
			Flags:      flags,
		},
		TypeParameters:  typeParameters,
		ExtendsType:     extendsType,
		ImplementsTypes: implementsTypes,
		Members:         members,
	}
}

// IsGeneric returns true if this class has type parameters.
func (c *ClassDeclaration) IsGeneric() bool {
	return c.TypeParameters != nil && len(c.TypeParameters) > 0
}

// NewInterfaceDeclaration creates an interface declaration.
// In TS, InterfaceDeclaration extends ClassDeclaration and changes the kind.
func NewInterfaceDeclaration(
	name *IdentifierExpression,
	decorators []*DecoratorNode,
	flags int32,
	typeParameters []*TypeParameterNode,
	extendsType *NamedTypeNode,
	implementsTypes []*NamedTypeNode,
	members []Node,
	rng diagnostics.Range,
) *ClassDeclaration {
	decl := NewClassDeclaration(name, decorators, flags, typeParameters, extendsType, implementsTypes, members, rng)
	decl.Kind = NodeKindInterfaceDeclaration
	return decl
}

// EnumDeclaration represents an `enum` declaration.
type EnumDeclaration struct {
	DeclarationBase
	Values []*EnumValueDeclaration
}

// NewEnumDeclaration creates an enum declaration.
func NewEnumDeclaration(name *IdentifierExpression, decorators []*DecoratorNode, flags int32, values []*EnumValueDeclaration, rng diagnostics.Range) *EnumDeclaration {
	return &EnumDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindEnumDeclaration, Range: rng},
			Name:       name,
			Decorators: decorators,
			Flags:      flags,
		},
		Values: values,
	}
}

// EnumValueDeclaration represents a value of an `enum` declaration.
type EnumValueDeclaration struct {
	DeclarationBase
	Type        Node // TypeNode, or nil
	Initializer Node // Expression, or nil
}

// NewEnumValueDeclaration creates an enum value declaration.
func NewEnumValueDeclaration(name *IdentifierExpression, flags int32, initializer Node, rng diagnostics.Range) *EnumValueDeclaration {
	return &EnumValueDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindEnumValueDeclaration, Range: rng},
			Name:       name,
			Decorators: nil,
			Flags:      flags,
		},
		Type:        nil,
		Initializer: initializer,
	}
}

// FieldDeclaration represents a field declaration within a `class`.
type FieldDeclaration struct {
	DeclarationBase
	Type           Node // TypeNode, or nil
	Initializer    Node // Expression, or nil
	ParameterIndex int32
}

// NewFieldDeclaration creates a field declaration.
func NewFieldDeclaration(name *IdentifierExpression, decorators []*DecoratorNode, flags int32, typ Node, initializer Node, rng diagnostics.Range) *FieldDeclaration {
	return &FieldDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindFieldDeclaration, Range: rng},
			Name:       name,
			Decorators: decorators,
			Flags:      flags,
		},
		Type:           typ,
		Initializer:    initializer,
		ParameterIndex: -1,
	}
}

// FunctionDeclaration represents a `function` declaration.
type FunctionDeclaration struct {
	DeclarationBase
	TypeParameters []*TypeParameterNode
	Signature      *FunctionTypeNode
	Body           Node // Statement, or nil
	ArrowKind      ArrowKind
}

// NewFunctionDeclaration creates a function declaration.
func NewFunctionDeclaration(
	name *IdentifierExpression,
	decorators []*DecoratorNode,
	flags int32,
	typeParameters []*TypeParameterNode,
	signature *FunctionTypeNode,
	body Node,
	arrowKind ArrowKind,
	rng diagnostics.Range,
) *FunctionDeclaration {
	return &FunctionDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindFunctionDeclaration, Range: rng},
			Name:       name,
			Decorators: decorators,
			Flags:      flags,
		},
		TypeParameters: typeParameters,
		Signature:      signature,
		Body:           body,
		ArrowKind:      arrowKind,
	}
}

// IsGeneric returns true if this function has type parameters.
func (f *FunctionDeclaration) IsGeneric() bool {
	return f.TypeParameters != nil && len(f.TypeParameters) > 0
}

// Clone clones this function declaration.
func (f *FunctionDeclaration) Clone() *FunctionDeclaration {
	return NewFunctionDeclaration(
		f.Name, f.Decorators, f.Flags,
		f.TypeParameters, f.Signature, f.Body,
		f.ArrowKind, f.Range,
	)
}

// NewMethodDeclaration creates a method declaration.
// In TS, MethodDeclaration extends FunctionDeclaration and changes the kind.
func NewMethodDeclaration(
	name *IdentifierExpression,
	decorators []*DecoratorNode,
	flags int32,
	typeParameters []*TypeParameterNode,
	signature *FunctionTypeNode,
	body Node,
	rng diagnostics.Range,
) *FunctionDeclaration {
	decl := NewFunctionDeclaration(name, decorators, flags, typeParameters, signature, body, ArrowKindNone, rng)
	decl.Kind = NodeKindMethodDeclaration
	return decl
}

// ImportDeclaration represents an `import` declaration part of an ImportStatement.
type ImportDeclaration struct {
	DeclarationBase
	ForeignName *IdentifierExpression
}

// NewImportDeclaration creates an import declaration.
func NewImportDeclaration(foreignName *IdentifierExpression, name *IdentifierExpression, rng diagnostics.Range) *ImportDeclaration {
	if name == nil {
		name = foreignName
	}
	return &ImportDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindImportDeclaration, Range: rng},
			Name:       name,
			Decorators: nil,
			Flags:      0, // CommonFlags.None
		},
		ForeignName: foreignName,
	}
}

// ImportStatement represents an `import` statement.
type ImportStatement struct {
	NodeBase
	Declarations  []*ImportDeclaration
	NamespaceName *IdentifierExpression // or nil
	Path          *StringLiteralExpression
	InternalPath  string
}

// NewImportStatement creates an import statement.
func NewImportStatement(declarations []*ImportDeclaration, path *StringLiteralExpression, rng diagnostics.Range) *ImportStatement {
	stmt := &ImportStatement{
		NodeBase:     NodeBase{Kind: NodeKindImport, Range: rng},
		Declarations: declarations,
		Path:         path,
	}
	stmt.InternalPath = resolveModulePath(path.Value, rng)
	return stmt
}

// NewWildcardImportStatement creates a wildcard import statement.
func NewWildcardImportStatement(namespaceName *IdentifierExpression, path *StringLiteralExpression, rng diagnostics.Range) *ImportStatement {
	stmt := &ImportStatement{
		NodeBase:      NodeBase{Kind: NodeKindImport, Range: rng},
		NamespaceName: namespaceName,
		Path:          path,
	}
	stmt.InternalPath = resolveModulePath(path.Value, rng)
	return stmt
}

// ExportMember represents a member of an `export` statement.
type ExportMember struct {
	NodeBase
	LocalName    *IdentifierExpression
	ExportedName *IdentifierExpression
}

// NewExportMember creates an export member.
func NewExportMember(localName *IdentifierExpression, exportedName *IdentifierExpression, rng diagnostics.Range) *ExportMember {
	if exportedName == nil {
		exportedName = localName
	}
	return &ExportMember{
		NodeBase:     NodeBase{Kind: NodeKindExportMember, Range: rng},
		LocalName:    localName,
		ExportedName: exportedName,
	}
}

// ExportStatement represents an `export` statement.
type ExportStatement struct {
	NodeBase
	Members      []*ExportMember
	Path         *StringLiteralExpression // or nil
	IsDeclare    bool
	InternalPath string
	HasInternal  bool
}

// NewExportStatement creates an export statement.
func NewExportStatement(members []*ExportMember, path *StringLiteralExpression, isDeclare bool, rng diagnostics.Range) *ExportStatement {
	stmt := &ExportStatement{
		NodeBase:  NodeBase{Kind: NodeKindExport, Range: rng},
		Members:   members,
		Path:      path,
		IsDeclare: isDeclare,
	}
	if path != nil {
		normalizedPath := util.NormalizePath(path.Value)
		if strings.HasPrefix(path.Value, ".") {
			normalizedPath = util.ResolvePath(normalizedPath, sourceInternalPath(rng))
		} else {
			if !strings.HasPrefix(normalizedPath, common.LIBRARY_PREFIX) {
				normalizedPath = common.LIBRARY_PREFIX + normalizedPath
			}
		}
		stmt.InternalPath = normalizedPath
		stmt.HasInternal = true
	}
	return stmt
}

// ExportDefaultStatement represents an `export default` statement.
type ExportDefaultStatement struct {
	NodeBase
	Declaration Node // DeclarationStatement
}

// NewExportDefaultStatement creates an export default statement.
func NewExportDefaultStatement(declaration Node, rng diagnostics.Range) *ExportDefaultStatement {
	return &ExportDefaultStatement{
		NodeBase:    NodeBase{Kind: NodeKindExportDefault, Range: rng},
		Declaration: declaration,
	}
}

// ExportImportStatement represents an `export import` statement.
type ExportImportStatement struct {
	NodeBase
	Name         *IdentifierExpression
	ExternalName *IdentifierExpression
}

// NewExportImportStatement creates an export import statement.
func NewExportImportStatement(name *IdentifierExpression, externalName *IdentifierExpression, rng diagnostics.Range) *ExportImportStatement {
	return &ExportImportStatement{
		NodeBase:     NodeBase{Kind: NodeKindExportImport, Range: rng},
		Name:         name,
		ExternalName: externalName,
	}
}

// NamespaceDeclaration represents a `namespace` declaration.
type NamespaceDeclaration struct {
	DeclarationBase
	Members []Node // Statement elements
}

// NewNamespaceDeclaration creates a namespace declaration.
func NewNamespaceDeclaration(name *IdentifierExpression, decorators []*DecoratorNode, flags int32, members []Node, rng diagnostics.Range) *NamespaceDeclaration {
	return &NamespaceDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindNamespaceDeclaration, Range: rng},
			Name:       name,
			Decorators: decorators,
			Flags:      flags,
		},
		Members: members,
	}
}

// TypeDeclaration represents a `type` declaration.
type TypeDeclaration struct {
	DeclarationBase
	TypeParameters []*TypeParameterNode
	Type           Node // TypeNode
}

// NewTypeDeclaration creates a type declaration.
func NewTypeDeclaration(name *IdentifierExpression, decorators []*DecoratorNode, flags int32, typeParameters []*TypeParameterNode, typ Node, rng diagnostics.Range) *TypeDeclaration {
	return &TypeDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindTypeDeclaration, Range: rng},
			Name:       name,
			Decorators: decorators,
			Flags:      flags,
		},
		TypeParameters: typeParameters,
		Type:           typ,
	}
}

// VariableDeclaration represents a variable declaration part of a VariableStatement.
type VariableDeclaration struct {
	DeclarationBase
	Type        Node // TypeNode, or nil
	Initializer Node // Expression, or nil
}

// NewVariableDeclaration creates a variable declaration.
func NewVariableDeclaration(name *IdentifierExpression, decorators []*DecoratorNode, flags int32, typ Node, initializer Node, rng diagnostics.Range) *VariableDeclaration {
	return &VariableDeclaration{
		DeclarationBase: DeclarationBase{
			NodeBase:   NodeBase{Kind: NodeKindVariableDeclaration, Range: rng},
			Name:       name,
			Decorators: decorators,
			Flags:      flags,
		},
		Type:        typ,
		Initializer: initializer,
	}
}

// VariableStatement represents a variable statement wrapping VariableDeclarations.
type VariableStatement struct {
	NodeBase
	Decorators   []*DecoratorNode
	Declarations []*VariableDeclaration
}

// NewVariableStatement creates a variable statement.
func NewVariableStatement(decorators []*DecoratorNode, declarations []*VariableDeclaration, rng diagnostics.Range) *VariableStatement {
	return &VariableStatement{
		NodeBase:     NodeBase{Kind: NodeKindVariable, Range: rng},
		Decorators:   decorators,
		Declarations: declarations,
	}
}

// --- Helper ---

// sourceInternalPath extracts the internal path from the range's source.
// The source is always an *ast.Source during parsing.
func sourceInternalPath(rng diagnostics.Range) string {
	if src, ok := rng.Source.(*Source); ok {
		return src.InternalPath
	}
	// Fallback: mangle the normalized path
	return MangleInternalPath(rng.Source.SourceNormalizedPath())
}

// resolveModulePath resolves a module path to an internal path.
func resolveModulePath(pathValue string, rng diagnostics.Range) string {
	normalizedPath := util.NormalizePath(pathValue)
	if strings.HasPrefix(pathValue, ".") {
		normalizedPath = util.ResolvePath(normalizedPath, sourceInternalPath(rng))
	} else {
		if !strings.HasPrefix(normalizedPath, common.LIBRARY_PREFIX) {
			normalizedPath = common.LIBRARY_PREFIX + normalizedPath
		}
	}
	return MangleInternalPath(normalizedPath)
}
