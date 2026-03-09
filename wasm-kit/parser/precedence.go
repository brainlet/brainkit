package parser

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

// Precedence represents operator precedence from least to largest.
type Precedence int32

const (
	PrecedenceNone           Precedence = iota
	PrecedenceComma
	PrecedenceSpread
	PrecedenceYield
	PrecedenceAssignment
	PrecedenceConditional
	PrecedenceLogicalOr
	PrecedenceLogicalAnd
	PrecedenceBitwiseOr
	PrecedenceBitwiseXor
	PrecedenceBitwiseAnd
	PrecedenceEquality
	PrecedenceRelational
	PrecedenceShift
	PrecedenceAdditive
	PrecedenceMultiplicative
	PrecedenceExponentiated
	PrecedenceUnaryPrefix
	PrecedenceUnaryPostfix
	PrecedenceCall
	PrecedenceMemberAccess
	PrecedenceGrouping
)

// determinePrecedence returns the precedence of a non-starting token.
func determinePrecedence(kind tokenizer.Token) Precedence {
	switch kind {
	case tokenizer.TokenComma:
		return PrecedenceComma
	case tokenizer.TokenEquals,
		tokenizer.TokenPlusEquals,
		tokenizer.TokenMinusEquals,
		tokenizer.TokenAsteriskAsteriskEquals,
		tokenizer.TokenAsteriskEquals,
		tokenizer.TokenSlashEquals,
		tokenizer.TokenPercentEquals,
		tokenizer.TokenLessThanLessThanEquals,
		tokenizer.TokenGreaterThanGreaterThanEquals,
		tokenizer.TokenGreaterThanGreaterThanGreaterThanEquals,
		tokenizer.TokenAmpersandEquals,
		tokenizer.TokenCaretEquals,
		tokenizer.TokenBarEquals:
		return PrecedenceAssignment
	case tokenizer.TokenQuestion:
		return PrecedenceConditional
	case tokenizer.TokenBarBar:
		return PrecedenceLogicalOr
	case tokenizer.TokenAmpersandAmpersand:
		return PrecedenceLogicalAnd
	case tokenizer.TokenBar:
		return PrecedenceBitwiseOr
	case tokenizer.TokenCaret:
		return PrecedenceBitwiseXor
	case tokenizer.TokenAmpersand:
		return PrecedenceBitwiseAnd
	case tokenizer.TokenEqualsEquals,
		tokenizer.TokenExclamationEquals,
		tokenizer.TokenEqualsEqualsEquals,
		tokenizer.TokenExclamationEqualsEquals:
		return PrecedenceEquality
	case tokenizer.TokenAs,
		tokenizer.TokenIn,
		tokenizer.TokenInstanceOf,
		tokenizer.TokenLessThan,
		tokenizer.TokenGreaterThan,
		tokenizer.TokenLessThanEquals,
		tokenizer.TokenGreaterThanEquals:
		return PrecedenceRelational
	case tokenizer.TokenLessThanLessThan,
		tokenizer.TokenGreaterThanGreaterThan,
		tokenizer.TokenGreaterThanGreaterThanGreaterThan:
		return PrecedenceShift
	case tokenizer.TokenPlus,
		tokenizer.TokenMinus:
		return PrecedenceAdditive
	case tokenizer.TokenAsterisk,
		tokenizer.TokenSlash,
		tokenizer.TokenPercent:
		return PrecedenceMultiplicative
	case tokenizer.TokenAsteriskAsterisk:
		return PrecedenceExponentiated
	case tokenizer.TokenPlusPlus,
		tokenizer.TokenMinusMinus:
		return PrecedenceUnaryPostfix
	case tokenizer.TokenDot,
		tokenizer.TokenOpenBracket,
		tokenizer.TokenExclamation:
		return PrecedenceMemberAccess
	}
	return PrecedenceNone
}

// isCircularTypeAlias checks if the type alias of the given name and type is circular.
func isCircularTypeAlias(name string, typeNode ast.Node) bool {
	switch typeNode.GetKind() {
	case ast.NodeKindNamedType:
		namedType := typeNode.(*ast.NamedTypeNode)
		if namedType.Name.Identifier.Text == name {
			return true
		}
		if namedType.TypeArguments != nil {
			for _, ta := range namedType.TypeArguments {
				if isCircularTypeAlias(name, ta) {
					return true
				}
			}
		}
	case ast.NodeKindFunctionType:
		functionType := typeNode.(*ast.FunctionTypeNode)
		if isCircularTypeAlias(name, functionType.ReturnType) {
			return true
		}
		for _, param := range functionType.Parameters {
			if isCircularTypeAlias(name, param.Type) {
				return true
			}
		}
	}
	return false
}
