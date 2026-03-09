package ast

// NodeKind indicates the kind of a node.
type NodeKind int32

const (
	NodeKindSource NodeKind = iota

	// types
	NodeKindNamedType
	NodeKindFunctionType
	NodeKindTypeName
	NodeKindTypeParameter
	NodeKindParameter

	// expressions
	NodeKindIdentifier
	NodeKindAssertion
	NodeKindBinary
	NodeKindCall
	NodeKindClass
	NodeKindComma
	NodeKindElementAccess
	NodeKindFalse
	NodeKindFunction
	NodeKindInstanceOf
	NodeKindLiteral
	NodeKindNew
	NodeKindNull
	NodeKindOmitted
	NodeKindParenthesized
	NodeKindPropertyAccess
	NodeKindTernary
	NodeKindSuper
	NodeKindThis
	NodeKindTrue
	NodeKindConstructor
	NodeKindUnaryPostfix
	NodeKindUnaryPrefix
	NodeKindCompiled

	// statements
	NodeKindBlock
	NodeKindBreak
	NodeKindContinue
	NodeKindDo
	NodeKindEmpty
	NodeKindExport
	NodeKindExportDefault
	NodeKindExportImport
	NodeKindExpression
	NodeKindFor
	NodeKindForOf
	NodeKindIf
	NodeKindImport
	NodeKindReturn
	NodeKindSwitch
	NodeKindThrow
	NodeKindTry
	NodeKindVariable
	NodeKindVoid
	NodeKindWhile
	NodeKindModule

	// declaration statements
	NodeKindClassDeclaration
	NodeKindEnumDeclaration
	NodeKindEnumValueDeclaration
	NodeKindFieldDeclaration
	NodeKindFunctionDeclaration
	NodeKindImportDeclaration
	NodeKindInterfaceDeclaration
	NodeKindMethodDeclaration
	NodeKindNamespaceDeclaration
	NodeKindTypeDeclaration
	NodeKindVariableDeclaration

	// special
	NodeKindDecorator
	NodeKindExportMember
	NodeKindSwitchCase
	NodeKindIndexSignature
	NodeKindComment
)

// ParameterKind represents the kind of a parameter.
type ParameterKind int32

const (
	ParameterKindDefault  ParameterKind = iota
	ParameterKindOptional
	ParameterKindRest
)

// LiteralKind indicates the kind of a literal.
type LiteralKind int32

const (
	LiteralKindFloat   LiteralKind = iota
	LiteralKindInteger
	LiteralKindString
	LiteralKindTemplate
	LiteralKindRegExp
	LiteralKindArray
	LiteralKindObject
)

// AssertionKind indicates the kind of an assertion.
type AssertionKind int32

const (
	AssertionKindPrefix  AssertionKind = iota
	AssertionKindAs
	AssertionKindNonNull
	AssertionKindConst
)

// ArrowKind indicates the kind of an arrow function.
type ArrowKind int32

const (
	ArrowKindNone          ArrowKind = iota
	ArrowKindParenthesized
	ArrowKindSingle
)

// SourceKind indicates the specific kind of a source.
type SourceKind int32

const (
	SourceKindUser         SourceKind = 0
	SourceKindUserEntry    SourceKind = 1
	SourceKindLibrary      SourceKind = 2
	SourceKindLibraryEntry SourceKind = 3
)

// DecoratorKind represents built-in decorator kinds.
type DecoratorKind int32

const (
	DecoratorKindCustom DecoratorKind = iota
	DecoratorKindGlobal
	DecoratorKindOperator
	DecoratorKindOperatorBinary
	DecoratorKindOperatorPrefix
	DecoratorKindOperatorPostfix
	DecoratorKindUnmanaged
	DecoratorKindFinal
	DecoratorKindInline
	DecoratorKindExternal
	DecoratorKindExternalJs
	DecoratorKindBuiltin
	DecoratorKindLazy
	DecoratorKindUnsafe
)
