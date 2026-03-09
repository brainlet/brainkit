package ast

import (
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

// Node is the interface implemented by all AST nodes.
type Node interface {
	GetKind() NodeKind
	GetRange() *diagnostics.Range
	SetRange(r diagnostics.Range)
}

// NodeBase provides the common fields for all AST nodes.
type NodeBase struct {
	Kind  NodeKind
	Range diagnostics.Range
}

func (n *NodeBase) GetKind() NodeKind              { return n.Kind }
func (n *NodeBase) GetRange() *diagnostics.Range    { return &n.Range }
func (n *NodeBase) SetRange(r diagnostics.Range)    { n.Range = r }

// LiteralNode is implemented by all literal expression types that embed LiteralExpression.
type LiteralNode interface {
	Node
	GetLiteralKind() LiteralKind
}

// GetLiteralKind returns the literal kind for embedded LiteralExpression.
func (l *LiteralExpression) GetLiteralKind() LiteralKind { return l.LiteralKind }

// IsLiteralKind tests if a node is a literal of the specified kind.
func IsLiteralKind(n Node, literalKind LiteralKind) bool {
	if n.GetKind() == NodeKindLiteral {
		if le, ok := n.(LiteralNode); ok {
			return le.GetLiteralKind() == literalKind
		}
	}
	return false
}

// IsNumericLiteral tests if a node is a literal of a numeric kind (float or integer).
func IsNumericLiteral(n Node) bool {
	if n.GetKind() == NodeKindLiteral {
		if le, ok := n.(LiteralNode); ok {
			switch le.GetLiteralKind() {
			case LiteralKindFloat, LiteralKindInteger:
				return true
			}
		}
	}
	return false
}

// CompilesToConst tests whether a node is guaranteed to compile to a constant value.
func CompilesToConst(n Node) bool {
	switch n.GetKind() {
	case NodeKindLiteral:
		if le, ok := n.(LiteralNode); ok {
			switch le.GetLiteralKind() {
			case LiteralKindFloat, LiteralKindInteger, LiteralKindString:
				return true
			}
		}
	case NodeKindNull, NodeKindTrue, NodeKindFalse:
		return true
	}
	return false
}

// IsAccessOnThis checks if a node accesses a method or property on `this`.
func IsAccessOnThis(n Node) bool {
	return isAccessOn(n, NodeKindThis)
}

// IsAccessOnSuper checks if a node accesses a method or property on `super`.
func IsAccessOnSuper(n Node) bool {
	return isAccessOn(n, NodeKindSuper)
}

func isAccessOn(n Node, kind NodeKind) bool {
	if n.GetKind() == NodeKindCall {
		n = n.(*CallExpression).Expression
	}
	if n.GetKind() == NodeKindPropertyAccess {
		target := n.(*PropertyAccessExpression).Expression
		if target.GetKind() == kind {
			return true
		}
	}
	return false
}

// IsEmpty tests if a node is an empty statement.
func IsEmpty(n Node) bool {
	return n.GetKind() == NodeKindEmpty
}

// --- Type nodes ---

// TypeName represents a type name.
type TypeName struct {
	NodeBase
	Identifier *IdentifierExpression
	Next       *TypeName
}

// NamedTypeNode represents a named type.
type NamedTypeNode struct {
	NodeBase
	Name               *TypeName
	TypeArguments      []Node // TypeNode elements
	IsNullable         bool
	CurrentlyResolving bool
}

// HasTypeArguments checks if this type node has type arguments.
func (n *NamedTypeNode) HasTypeArguments() bool {
	return n.TypeArguments != nil && len(n.TypeArguments) > 0
}

// IsNull tests if this type is "null".
func (n *NamedTypeNode) IsNull() bool {
	return n.Name.Identifier.Text == "null"
}

// FunctionTypeNode represents a function type.
type FunctionTypeNode struct {
	NodeBase
	Parameters       []*ParameterNode
	ReturnType       Node // TypeNode
	ExplicitThisType *NamedTypeNode
	IsNullable       bool
	CurrentlyResolving bool
}

// TypeParameterNode represents a type parameter.
type TypeParameterNode struct {
	NodeBase
	Name        *IdentifierExpression
	ExtendsType *NamedTypeNode
	DefaultType *NamedTypeNode
}

// ParameterNode represents a function parameter.
type ParameterNode struct {
	NodeBase
	ParameterKind          ParameterKind
	Name                   *IdentifierExpression
	Type                   Node // TypeNode
	Initializer            Node // Expression, or nil
	ImplicitFieldDeclaration *FieldDeclaration
	Flags                  int32 // CommonFlags
}

// Is tests if this node has the specified flag or flags.
func (n *ParameterNode) Is(flag int32) bool { return (n.Flags & flag) == flag }

// IsAny tests if this node has one of the specified flags.
func (n *ParameterNode) IsAny(flag int32) bool { return (n.Flags & flag) != 0 }

// Set sets a specific flag or flags.
func (n *ParameterNode) Set(flag int32) { n.Flags |= flag }

// HasGenericComponent tests if a type node has a generic component matching one of the given type parameters.
func HasGenericComponent(typeNode Node, typeParameterNodes []*TypeParameterNode) bool {
	if typeNode.GetKind() == NodeKindNamedType {
		namedTypeNode := typeNode.(*NamedTypeNode)
		if namedTypeNode.Name.Next == nil {
			typeArgs := namedTypeNode.TypeArguments
			if typeArgs != nil && len(typeArgs) > 0 {
				for _, ta := range typeArgs {
					if HasGenericComponent(ta, typeParameterNodes) {
						return true
					}
				}
			} else {
				name := namedTypeNode.Name.Identifier.Text
				for _, tp := range typeParameterNodes {
					if tp.Name.Text == name {
						return true
					}
				}
			}
		}
	} else if typeNode.GetKind() == NodeKindFunctionType {
		functionTypeNode := typeNode.(*FunctionTypeNode)
		for _, param := range functionTypeNode.Parameters {
			if HasGenericComponent(param.Type, typeParameterNodes) {
				return true
			}
		}
		if HasGenericComponent(functionTypeNode.ReturnType, typeParameterNodes) {
			return true
		}
		if functionTypeNode.ExplicitThisType != nil {
			if HasGenericComponent(functionTypeNode.ExplicitThisType, typeParameterNodes) {
				return true
			}
		}
	}
	return false
}

// --- Factory functions for type nodes ---

// NewSimpleTypeName creates a simple type name from a string.
func NewSimpleTypeName(name string, rng diagnostics.Range) *TypeName {
	return &TypeName{
		NodeBase:   NodeBase{Kind: NodeKindTypeName, Range: rng},
		Identifier: NewIdentifierExpression(name, rng, false),
		Next:       nil,
	}
}

// NewNamedTypeNode creates a named type node.
func NewNamedTypeNode(name *TypeName, typeArguments []Node, isNullable bool, rng diagnostics.Range) *NamedTypeNode {
	return &NamedTypeNode{
		NodeBase:      NodeBase{Kind: NodeKindNamedType, Range: rng},
		Name:          name,
		TypeArguments: typeArguments,
		IsNullable:    isNullable,
	}
}

// NewOmittedType creates an omitted type node.
func NewOmittedType(rng diagnostics.Range) *NamedTypeNode {
	return NewNamedTypeNode(NewSimpleTypeName("", rng), nil, false, rng)
}

// NewFunctionTypeNode creates a function type node.
func NewFunctionTypeNode(parameters []*ParameterNode, returnType Node, explicitThisType *NamedTypeNode, isNullable bool, rng diagnostics.Range) *FunctionTypeNode {
	return &FunctionTypeNode{
		NodeBase:         NodeBase{Kind: NodeKindFunctionType, Range: rng},
		Parameters:       parameters,
		ReturnType:       returnType,
		ExplicitThisType: explicitThisType,
		IsNullable:       isNullable,
	}
}

// NewTypeParameterNode creates a type parameter node.
func NewTypeParameterNode(name *IdentifierExpression, extendsType *NamedTypeNode, defaultType *NamedTypeNode, rng diagnostics.Range) *TypeParameterNode {
	return &TypeParameterNode{
		NodeBase:    NodeBase{Kind: NodeKindTypeParameter, Range: rng},
		Name:        name,
		ExtendsType: extendsType,
		DefaultType: defaultType,
	}
}

// NewParameterNode creates a function parameter node.
func NewParameterNode(parameterKind ParameterKind, name *IdentifierExpression, typ Node, initializer Node, rng diagnostics.Range) *ParameterNode {
	return &ParameterNode{
		NodeBase:      NodeBase{Kind: NodeKindParameter, Range: rng},
		ParameterKind: parameterKind,
		Name:          name,
		Type:          typ,
		Initializer:   initializer,
	}
}

// --- Decorator and comment nodes ---

// DecoratorKindFromNode returns the kind of the specified decorator name node.
func DecoratorKindFromNode(nameNode Node) DecoratorKind {
	if nameNode.GetKind() == NodeKindIdentifier {
		nameStr := nameNode.(*IdentifierExpression).Text
		if len(nameStr) == 0 {
			panic("assertion failed: decorator name must not be empty")
		}
		switch nameStr[0] {
		case 'b':
			if nameStr == "builtin" {
				return DecoratorKindBuiltin
			}
		case 'e':
			if nameStr == "external" {
				return DecoratorKindExternal
			}
		case 'f':
			if nameStr == "final" {
				return DecoratorKindFinal
			}
		case 'g':
			if nameStr == "global" {
				return DecoratorKindGlobal
			}
		case 'i':
			if nameStr == "inline" {
				return DecoratorKindInline
			}
		case 'l':
			if nameStr == "lazy" {
				return DecoratorKindLazy
			}
		case 'o':
			if nameStr == "operator" {
				return DecoratorKindOperator
			}
		case 'u':
			if nameStr == "unmanaged" {
				return DecoratorKindUnmanaged
			}
			if nameStr == "unsafe" {
				return DecoratorKindUnsafe
			}
		}
	} else if nameNode.GetKind() == NodeKindPropertyAccess {
		propAccess := nameNode.(*PropertyAccessExpression)
		expression := propAccess.Expression
		if expression.GetKind() == NodeKindIdentifier {
			nameStr := expression.(*IdentifierExpression).Text
			if len(nameStr) == 0 {
				panic("assertion failed: decorator name must not be empty")
			}
			propStr := propAccess.Property.Text
			if len(propStr) == 0 {
				panic("assertion failed: decorator property must not be empty")
			}
			if nameStr == "operator" {
				switch propStr[0] {
				case 'b':
					if propStr == "binary" {
						return DecoratorKindOperatorBinary
					}
				case 'p':
					if propStr == "prefix" {
						return DecoratorKindOperatorPrefix
					}
					if propStr == "postfix" {
						return DecoratorKindOperatorPostfix
					}
				}
			} else if nameStr == "external" {
				if propStr[0] == 'j' && propStr == "js" {
					return DecoratorKindExternalJs
				}
			}
		}
	}
	return DecoratorKindCustom
}

// DecoratorNode represents a decorator.
type DecoratorNode struct {
	NodeBase
	DecoratorKind DecoratorKind
	Name          Node // Expression
	Args          []Node // Expression elements, or nil
}

// NewDecoratorNode creates a decorator node.
func NewDecoratorNode(name Node, args []Node, rng diagnostics.Range) *DecoratorNode {
	return &DecoratorNode{
		NodeBase:      NodeBase{Kind: NodeKindDecorator, Range: rng},
		DecoratorKind: DecoratorKindFromNode(name),
		Name:          name,
		Args:          args,
	}
}

// CommentNode represents a comment.
type CommentNode struct {
	NodeBase
	CommentKind tokenizer.CommentKind
	Text        string
}

// NewCommentNode creates a comment node.
func NewCommentNode(commentKind tokenizer.CommentKind, text string, rng diagnostics.Range) *CommentNode {
	return &CommentNode{
		NodeBase:    NodeBase{Kind: NodeKindComment, Range: rng},
		CommentKind: commentKind,
		Text:        text,
	}
}

// IsTypeOmitted tests if the specified type node represents an omitted type.
func IsTypeOmitted(typeNode Node) bool {
	if typeNode.GetKind() == NodeKindNamedType {
		name := typeNode.(*NamedTypeNode).Name
		return name.Next == nil && len(name.Identifier.Text) == 0
	}
	return false
}

// FindDecorator finds the first decorator matching the specified kind.
func FindDecorator(kind DecoratorKind, decorators []*DecoratorNode) *DecoratorNode {
	for _, decorator := range decorators {
		if decorator.DecoratorKind == kind {
			return decorator
		}
	}
	return nil
}

// MangleInternalPath mangles an external to an internal path.
func MangleInternalPath(path string) string {
	if len(path) > 0 && path[len(path)-1] == '/' {
		path += "index"
	} else if len(path) > 3 && path[len(path)-3:] == ".ts" {
		path = path[:len(path)-3]
	}
	return path
}
