package program

import "github.com/brainlet/brainkit/wasm-kit/ast"

// ElementKind represents the kind of a program element.
type ElementKind = int32

const (
	// ElementKindGlobal indicates a global variable.
	ElementKindGlobal ElementKind = iota
	// ElementKindLocal indicates a local variable.
	ElementKindLocal
	// ElementKindEnum indicates an enum declaration.
	ElementKindEnum
	// ElementKindEnumValue indicates a value within an enum.
	ElementKindEnumValue
	// ElementKindFunctionPrototype indicates an unresolved function prototype.
	ElementKindFunctionPrototype
	// ElementKindFunction indicates a resolved function instance.
	ElementKindFunction
	// ElementKindClassPrototype indicates an unresolved class prototype.
	ElementKindClassPrototype
	// ElementKindClass indicates a resolved class instance.
	ElementKindClass
	// ElementKindInterfacePrototype indicates an unresolved interface prototype.
	ElementKindInterfacePrototype
	// ElementKindInterface indicates a resolved interface instance.
	ElementKindInterface
	// ElementKindPropertyPrototype indicates an unresolved property prototype.
	ElementKindPropertyPrototype
	// ElementKindProperty indicates a resolved property instance.
	ElementKindProperty
	// ElementKindNamespace indicates a namespace element.
	ElementKindNamespace
	// ElementKindFile indicates a file-level element.
	ElementKindFile
	// ElementKindTypeDefinition indicates a type alias or definition.
	ElementKindTypeDefinition
	// ElementKindIndexSignature indicates an index signature element.
	ElementKindIndexSignature
)

// DecoratorFlags represents a bitmask of decorator attributes applied to an element.
type DecoratorFlags = uint32

const (
	// DecoratorFlagsNone indicates no decorators are applied.
	DecoratorFlagsNone DecoratorFlags = 0
	// DecoratorFlagsGlobal indicates the @global decorator.
	DecoratorFlagsGlobal DecoratorFlags = 1 << 0
	// DecoratorFlagsOperatorBinary indicates the @operator (binary) decorator.
	DecoratorFlagsOperatorBinary DecoratorFlags = 1 << 1
	// DecoratorFlagsOperatorPrefix indicates the @operator.prefix decorator.
	DecoratorFlagsOperatorPrefix DecoratorFlags = 1 << 2
	// DecoratorFlagsOperatorPostfix indicates the @operator.postfix decorator.
	DecoratorFlagsOperatorPostfix DecoratorFlags = 1 << 3
	// DecoratorFlagsUnmanaged indicates the @unmanaged decorator.
	DecoratorFlagsUnmanaged DecoratorFlags = 1 << 4
	// DecoratorFlagsFinal indicates the @final decorator.
	DecoratorFlagsFinal DecoratorFlags = 1 << 5
	// DecoratorFlagsInline indicates the @inline decorator.
	DecoratorFlagsInline DecoratorFlags = 1 << 6
	// DecoratorFlagsExternal indicates the @external decorator.
	DecoratorFlagsExternal DecoratorFlags = 1 << 7
	// DecoratorFlagsExternalJs indicates the @external.js decorator.
	DecoratorFlagsExternalJs DecoratorFlags = 1 << 8
	// DecoratorFlagsBuiltin indicates the @builtin decorator.
	DecoratorFlagsBuiltin DecoratorFlags = 1 << 9
	// DecoratorFlagsLazy indicates the @lazy decorator.
	DecoratorFlagsLazy DecoratorFlags = 1 << 10
	// DecoratorFlagsUnsafe indicates the @unsafe decorator.
	DecoratorFlagsUnsafe DecoratorFlags = 1 << 11
)

// DecoratorFlagsFromKind converts an ast.DecoratorKind to the corresponding DecoratorFlags bitmask value.
func DecoratorFlagsFromKind(kind ast.DecoratorKind) DecoratorFlags {
	switch kind {
	case ast.DecoratorKindGlobal:
		return DecoratorFlagsGlobal
	case ast.DecoratorKindOperator, ast.DecoratorKindOperatorBinary:
		return DecoratorFlagsOperatorBinary
	case ast.DecoratorKindOperatorPrefix:
		return DecoratorFlagsOperatorPrefix
	case ast.DecoratorKindOperatorPostfix:
		return DecoratorFlagsOperatorPostfix
	case ast.DecoratorKindUnmanaged:
		return DecoratorFlagsUnmanaged
	case ast.DecoratorKindFinal:
		return DecoratorFlagsFinal
	case ast.DecoratorKindInline:
		return DecoratorFlagsInline
	case ast.DecoratorKindExternal:
		return DecoratorFlagsExternal
	case ast.DecoratorKindExternalJs:
		return DecoratorFlagsExternalJs
	case ast.DecoratorKindBuiltin:
		return DecoratorFlagsBuiltin
	case ast.DecoratorKindLazy:
		return DecoratorFlagsLazy
	case ast.DecoratorKindUnsafe:
		return DecoratorFlagsUnsafe
	default:
		return DecoratorFlagsNone
	}
}

// ConstantValueKind represents the kind of a compile-time constant value.
type ConstantValueKind int32

const (
	// ConstantValueKindNone indicates no constant value.
	ConstantValueKindNone ConstantValueKind = iota
	// ConstantValueKindInteger indicates an integer constant value.
	ConstantValueKindInteger
	// ConstantValueKindFloat indicates a floating-point constant value.
	ConstantValueKindFloat
)

// declaredElements tracks which ElementKind values represent declared elements.
var declaredElements = map[ElementKind]bool{}

// typedElements tracks which ElementKind values represent typed elements.
var typedElements = map[ElementKind]bool{}

// IsDeclaredElement reports whether the given ElementKind represents a declared element.
func IsDeclaredElement(kind ElementKind) bool {
	return declaredElements[kind]
}

// IsTypedElement reports whether the given ElementKind represents a typed element.
func IsTypedElement(kind ElementKind) bool {
	return typedElements[kind]
}

// RegisterDeclaredElementKind registers an ElementKind as a declared element.
// This is intended to be called by element constructors during initialization.
func RegisterDeclaredElementKind(kind ElementKind) {
	declaredElements[kind] = true
}

// RegisterTypedElementKind registers an ElementKind as a typed element.
// This is intended to be called by element constructors during initialization.
func RegisterTypedElementKind(kind ElementKind) {
	typedElements[kind] = true
}
