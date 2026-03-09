package ast

import (
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

// IdentifierExpression represents an identifier expression.
type IdentifierExpression struct {
	NodeBase
	Text     string
	IsQuoted bool
}

// NewIdentifierExpression creates an identifier expression.
func NewIdentifierExpression(text string, rng diagnostics.Range, isQuoted bool) *IdentifierExpression {
	return &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindIdentifier, Range: rng},
		Text:     text,
		IsQuoted: isQuoted,
	}
}

// NewEmptyIdentifierExpression creates an empty identifier expression.
func NewEmptyIdentifierExpression(rng diagnostics.Range) *IdentifierExpression {
	return NewIdentifierExpression("", rng, false)
}

// NewConstructorExpression creates a constructor expression.
func NewConstructorExpression(rng diagnostics.Range) *IdentifierExpression {
	return &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindConstructor, Range: rng},
		Text:     "constructor",
		IsQuoted: false,
	}
}

// NewNullExpression creates a null expression.
func NewNullExpression(rng diagnostics.Range) *IdentifierExpression {
	return &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindNull, Range: rng},
		Text:     "null",
		IsQuoted: false,
	}
}

// NewSuperExpression creates a super expression.
func NewSuperExpression(rng diagnostics.Range) *IdentifierExpression {
	return &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindSuper, Range: rng},
		Text:     "super",
		IsQuoted: false,
	}
}

// NewThisExpression creates a this expression.
func NewThisExpression(rng diagnostics.Range) *IdentifierExpression {
	return &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindThis, Range: rng},
		Text:     "this",
		IsQuoted: false,
	}
}

// NewTrueExpression creates a true expression.
func NewTrueExpression(rng diagnostics.Range) *IdentifierExpression {
	return &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindTrue, Range: rng},
		Text:     "true",
		IsQuoted: false,
	}
}

// NewFalseExpression creates a false expression.
func NewFalseExpression(rng diagnostics.Range) *IdentifierExpression {
	return &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindFalse, Range: rng},
		Text:     "false",
		IsQuoted: false,
	}
}

// LiteralExpression is the base for all literal expressions.
// Concrete literal types embed this and add their own fields.
type LiteralExpression struct {
	NodeBase
	LiteralKind LiteralKind
}

// ArrayLiteralExpression represents an `[]` literal expression.
type ArrayLiteralExpression struct {
	LiteralExpression
	ElementExpressions []Node // Expression elements
}

// NewArrayLiteralExpression creates an array literal expression.
func NewArrayLiteralExpression(elementExpressions []Node, rng diagnostics.Range) *ArrayLiteralExpression {
	return &ArrayLiteralExpression{
		LiteralExpression: LiteralExpression{
			NodeBase:    NodeBase{Kind: NodeKindLiteral, Range: rng},
			LiteralKind: LiteralKindArray,
		},
		ElementExpressions: elementExpressions,
	}
}

// AssertionExpression represents an assertion expression.
type AssertionExpression struct {
	NodeBase
	AssertionKind AssertionKind
	Expression    Node // Expression
	ToType        Node // TypeNode, or nil
}

// NewAssertionExpression creates an assertion expression.
func NewAssertionExpression(assertionKind AssertionKind, expression Node, toType Node, rng diagnostics.Range) *AssertionExpression {
	return &AssertionExpression{
		NodeBase:      NodeBase{Kind: NodeKindAssertion, Range: rng},
		AssertionKind: assertionKind,
		Expression:    expression,
		ToType:        toType,
	}
}

// BinaryExpression represents a binary expression.
type BinaryExpression struct {
	NodeBase
	Operator tokenizer.Token
	Left     Node // Expression
	Right    Node // Expression
}

// NewBinaryExpression creates a binary expression.
func NewBinaryExpression(operator tokenizer.Token, left Node, right Node, rng diagnostics.Range) *BinaryExpression {
	return &BinaryExpression{
		NodeBase: NodeBase{Kind: NodeKindBinary, Range: rng},
		Operator: operator,
		Left:     left,
		Right:    right,
	}
}

// CallExpression represents a call expression.
type CallExpression struct {
	NodeBase
	Expression    Node   // Expression
	TypeArguments []Node // TypeNode elements, or nil
	Args          []Node // Expression elements
}

// NewCallExpression creates a call expression.
func NewCallExpression(expression Node, typeArguments []Node, args []Node, rng diagnostics.Range) *CallExpression {
	return &CallExpression{
		NodeBase:      NodeBase{Kind: NodeKindCall, Range: rng},
		Expression:    expression,
		TypeArguments: typeArguments,
		Args:          args,
	}
}

// TypeArgumentsRange returns the type arguments range for reporting.
func (c *CallExpression) TypeArgumentsRange() diagnostics.Range {
	if c.TypeArguments != nil {
		n := len(c.TypeArguments)
		if n > 0 {
			return *diagnostics.JoinRanges(
				c.TypeArguments[0].GetRange(),
				c.TypeArguments[n-1].GetRange(),
			)
		}
	}
	return *c.Expression.GetRange()
}

// ArgumentsRange returns the arguments range for reporting.
func (c *CallExpression) ArgumentsRange() diagnostics.Range {
	n := len(c.Args)
	if n > 0 {
		return *diagnostics.JoinRanges(
			c.Args[0].GetRange(),
			c.Args[n-1].GetRange(),
		)
	}
	return *c.Expression.GetRange()
}

// ClassExpression represents a class expression using the 'class' keyword.
type ClassExpression struct {
	NodeBase
	Declaration *ClassDeclaration
}

// NewClassExpression creates a class expression.
func NewClassExpression(declaration *ClassDeclaration) *ClassExpression {
	return &ClassExpression{
		NodeBase:    NodeBase{Kind: NodeKindClass, Range: declaration.Range},
		Declaration: declaration,
	}
}

// CommaExpression represents a comma expression composed of multiple expressions.
type CommaExpression struct {
	NodeBase
	Expressions []Node // Expression elements
}

// NewCommaExpression creates a comma expression.
func NewCommaExpression(expressions []Node, rng diagnostics.Range) *CommaExpression {
	return &CommaExpression{
		NodeBase:    NodeBase{Kind: NodeKindComma, Range: rng},
		Expressions: expressions,
	}
}

// ElementAccessExpression represents an element access expression, e.g., array access.
type ElementAccessExpression struct {
	NodeBase
	Expression        Node // Expression being accessed
	ElementExpression Node // Element of the expression being accessed
}

// NewElementAccessExpression creates an element access expression.
func NewElementAccessExpression(expression Node, elementExpression Node, rng diagnostics.Range) *ElementAccessExpression {
	return &ElementAccessExpression{
		NodeBase:          NodeBase{Kind: NodeKindElementAccess, Range: rng},
		Expression:        expression,
		ElementExpression: elementExpression,
	}
}

// FloatLiteralExpression represents a float literal expression.
type FloatLiteralExpression struct {
	LiteralExpression
	Value float64
}

// NewFloatLiteralExpression creates a float literal expression.
func NewFloatLiteralExpression(value float64, rng diagnostics.Range) *FloatLiteralExpression {
	return &FloatLiteralExpression{
		LiteralExpression: LiteralExpression{
			NodeBase:    NodeBase{Kind: NodeKindLiteral, Range: rng},
			LiteralKind: LiteralKindFloat,
		},
		Value: value,
	}
}

// FunctionExpression represents a function expression using the 'function' keyword.
type FunctionExpression struct {
	NodeBase
	Declaration *FunctionDeclaration
}

// NewFunctionExpression creates a function expression.
func NewFunctionExpression(declaration *FunctionDeclaration) *FunctionExpression {
	return &FunctionExpression{
		NodeBase:    NodeBase{Kind: NodeKindFunction, Range: declaration.Range},
		Declaration: declaration,
	}
}

// InstanceOfExpression represents an instanceof expression.
type InstanceOfExpression struct {
	NodeBase
	Expression Node // Expression being asserted
	IsType     Node // TypeNode to test for
}

// NewInstanceOfExpression creates an instanceof expression.
func NewInstanceOfExpression(expression Node, isType Node, rng diagnostics.Range) *InstanceOfExpression {
	return &InstanceOfExpression{
		NodeBase:   NodeBase{Kind: NodeKindInstanceOf, Range: rng},
		Expression: expression,
		IsType:     isType,
	}
}

// IntegerLiteralExpression represents an integer literal expression.
type IntegerLiteralExpression struct {
	LiteralExpression
	Value int64
}

// NewIntegerLiteralExpression creates an integer literal expression.
func NewIntegerLiteralExpression(value int64, rng diagnostics.Range) *IntegerLiteralExpression {
	return &IntegerLiteralExpression{
		LiteralExpression: LiteralExpression{
			NodeBase:    NodeBase{Kind: NodeKindLiteral, Range: rng},
			LiteralKind: LiteralKindInteger,
		},
		Value: value,
	}
}

// NewExpression represents a `new` expression.
type NewExpression struct {
	NodeBase
	TypeName      *TypeName
	TypeArguments []Node // TypeNode elements, or nil
	Args          []Node // Expression elements
}

// NewNewExpression creates a new expression.
func NewNewExpression(typeName *TypeName, typeArguments []Node, args []Node, rng diagnostics.Range) *NewExpression {
	return &NewExpression{
		NodeBase:      NodeBase{Kind: NodeKindNew, Range: rng},
		TypeName:      typeName,
		TypeArguments: typeArguments,
		Args:          args,
	}
}

// TypeArgumentsRange returns the type arguments range for reporting.
func (n *NewExpression) TypeArgumentsRange() diagnostics.Range {
	if n.TypeArguments != nil {
		count := len(n.TypeArguments)
		if count > 0 {
			return *diagnostics.JoinRanges(
				n.TypeArguments[0].GetRange(),
				n.TypeArguments[count-1].GetRange(),
			)
		}
	}
	return n.TypeName.Range
}

// ArgumentsRange returns the arguments range for reporting.
func (n *NewExpression) ArgumentsRange() diagnostics.Range {
	count := len(n.Args)
	if count > 0 {
		return *diagnostics.JoinRanges(
			n.Args[0].GetRange(),
			n.Args[count-1].GetRange(),
		)
	}
	return n.TypeName.Range
}

// ObjectLiteralExpression represents an object literal expression.
type ObjectLiteralExpression struct {
	LiteralExpression
	Names  []*IdentifierExpression
	Values []Node // Expression elements
}

// NewObjectLiteralExpression creates an object literal expression.
func NewObjectLiteralExpression(names []*IdentifierExpression, values []Node, rng diagnostics.Range) *ObjectLiteralExpression {
	return &ObjectLiteralExpression{
		LiteralExpression: LiteralExpression{
			NodeBase:    NodeBase{Kind: NodeKindLiteral, Range: rng},
			LiteralKind: LiteralKindObject,
		},
		Names:  names,
		Values: values,
	}
}

// OmittedExpression represents an omitted expression, e.g. within an array literal.
type OmittedExpression struct {
	NodeBase
}

// NewOmittedExpression creates an omitted expression.
func NewOmittedExpression(rng diagnostics.Range) *OmittedExpression {
	return &OmittedExpression{
		NodeBase: NodeBase{Kind: NodeKindOmitted, Range: rng},
	}
}

// ParenthesizedExpression represents a parenthesized expression.
type ParenthesizedExpression struct {
	NodeBase
	Expression Node // Expression
}

// NewParenthesizedExpression creates a parenthesized expression.
func NewParenthesizedExpression(expression Node, rng diagnostics.Range) *ParenthesizedExpression {
	return &ParenthesizedExpression{
		NodeBase:   NodeBase{Kind: NodeKindParenthesized, Range: rng},
		Expression: expression,
	}
}

// PropertyAccessExpression represents a property access expression.
type PropertyAccessExpression struct {
	NodeBase
	Expression Node // Expression being accessed
	Property   *IdentifierExpression
}

// NewPropertyAccessExpression creates a property access expression.
func NewPropertyAccessExpression(expression Node, property *IdentifierExpression, rng diagnostics.Range) *PropertyAccessExpression {
	return &PropertyAccessExpression{
		NodeBase:   NodeBase{Kind: NodeKindPropertyAccess, Range: rng},
		Expression: expression,
		Property:   property,
	}
}

// RegexpLiteralExpression represents a regular expression literal expression.
type RegexpLiteralExpression struct {
	LiteralExpression
	Pattern      string
	PatternFlags string
}

// NewRegexpLiteralExpression creates a regular expression literal expression.
func NewRegexpLiteralExpression(pattern string, patternFlags string, rng diagnostics.Range) *RegexpLiteralExpression {
	return &RegexpLiteralExpression{
		LiteralExpression: LiteralExpression{
			NodeBase:    NodeBase{Kind: NodeKindLiteral, Range: rng},
			LiteralKind: LiteralKindRegExp,
		},
		Pattern:      pattern,
		PatternFlags: patternFlags,
	}
}

// TernaryExpression represents a ternary expression.
type TernaryExpression struct {
	NodeBase
	Condition Node // Expression
	IfThen    Node // Expression
	IfElse    Node // Expression
}

// NewTernaryExpression creates a ternary expression.
func NewTernaryExpression(condition Node, ifThen Node, ifElse Node, rng diagnostics.Range) *TernaryExpression {
	return &TernaryExpression{
		NodeBase:  NodeBase{Kind: NodeKindTernary, Range: rng},
		Condition: condition,
		IfThen:    ifThen,
		IfElse:    ifElse,
	}
}

// StringLiteralExpression represents a string literal expression.
type StringLiteralExpression struct {
	LiteralExpression
	Value string
}

// NewStringLiteralExpression creates a string literal expression.
func NewStringLiteralExpression(value string, rng diagnostics.Range) *StringLiteralExpression {
	return &StringLiteralExpression{
		LiteralExpression: LiteralExpression{
			NodeBase:    NodeBase{Kind: NodeKindLiteral, Range: rng},
			LiteralKind: LiteralKindString,
		},
		Value: value,
	}
}

// TemplateLiteralExpression represents a template literal expression.
type TemplateLiteralExpression struct {
	LiteralExpression
	Tag         Node     // Expression, or nil
	Parts       []string
	RawParts    []string
	Expressions []Node   // Expression elements
}

// NewTemplateLiteralExpression creates a template literal expression.
func NewTemplateLiteralExpression(tag Node, parts []string, rawParts []string, expressions []Node, rng diagnostics.Range) *TemplateLiteralExpression {
	return &TemplateLiteralExpression{
		LiteralExpression: LiteralExpression{
			NodeBase:    NodeBase{Kind: NodeKindLiteral, Range: rng},
			LiteralKind: LiteralKindTemplate,
		},
		Tag:         tag,
		Parts:       parts,
		RawParts:    rawParts,
		Expressions: expressions,
	}
}

// UnaryPostfixExpression represents a unary postfix expression.
type UnaryPostfixExpression struct {
	NodeBase
	Operator tokenizer.Token
	Operand  Node // Expression
}

// NewUnaryPostfixExpression creates a unary postfix expression.
func NewUnaryPostfixExpression(operator tokenizer.Token, operand Node, rng diagnostics.Range) *UnaryPostfixExpression {
	return &UnaryPostfixExpression{
		NodeBase: NodeBase{Kind: NodeKindUnaryPostfix, Range: rng},
		Operator: operator,
		Operand:  operand,
	}
}

// UnaryPrefixExpression represents a unary prefix expression.
type UnaryPrefixExpression struct {
	NodeBase
	Operator tokenizer.Token
	Operand  Node // Expression
}

// NewUnaryPrefixExpression creates a unary prefix expression.
func NewUnaryPrefixExpression(operator tokenizer.Token, operand Node, rng diagnostics.Range) *UnaryPrefixExpression {
	return &UnaryPrefixExpression{
		NodeBase: NodeBase{Kind: NodeKindUnaryPrefix, Range: rng},
		Operator: operator,
		Operand:  operand,
	}
}

// CompiledExpression represents a special pre-compiled expression.
// ExpressionRef and Type are typed as `any` since those packages are not yet ported.
type CompiledExpression struct {
	NodeBase
	Expr any // module.ExpressionRef
	Type any // types.Type
}

// NewCompiledExpression creates a compiled expression.
func NewCompiledExpression(expr any, typ any, rng diagnostics.Range) *CompiledExpression {
	return &CompiledExpression{
		NodeBase: NodeBase{Kind: NodeKindCompiled, Range: rng},
		Expr:     expr,
		Type:     typ,
	}
}
