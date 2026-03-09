package program

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// OperatorKind indicates the kind of an overloaded operator.
type OperatorKind = int32

const (
	// OperatorKindInvalid indicates an invalid operator kind.
	OperatorKindInvalid OperatorKind = iota
	// OperatorKindIndexedGet represents a[] .
	OperatorKindIndexedGet
	// OperatorKindIndexedSet represents a[]=b .
	OperatorKindIndexedSet
	// OperatorKindUncheckedIndexedGet represents unchecked(a[]) .
	OperatorKindUncheckedIndexedGet
	// OperatorKindUncheckedIndexedSet represents unchecked(a[]=b) .
	OperatorKindUncheckedIndexedSet
	// OperatorKindAdd represents a + b .
	OperatorKindAdd
	// OperatorKindSub represents a - b .
	OperatorKindSub
	// OperatorKindMul represents a * b .
	OperatorKindMul
	// OperatorKindDiv represents a / b .
	OperatorKindDiv
	// OperatorKindRem represents a % b .
	OperatorKindRem
	// OperatorKindPow represents a ** b .
	OperatorKindPow
	// OperatorKindBitwiseAnd represents a & b .
	OperatorKindBitwiseAnd
	// OperatorKindBitwiseOr represents a | b .
	OperatorKindBitwiseOr
	// OperatorKindBitwiseXor represents a ^ b .
	OperatorKindBitwiseXor
	// OperatorKindBitwiseShl represents a << b .
	OperatorKindBitwiseShl
	// OperatorKindBitwiseShr represents a >> b .
	OperatorKindBitwiseShr
	// OperatorKindBitwiseShrU represents a >>> b .
	OperatorKindBitwiseShrU
	// OperatorKindEq represents a == b .
	OperatorKindEq
	// OperatorKindNe represents a != b .
	OperatorKindNe
	// OperatorKindGt represents a > b .
	OperatorKindGt
	// OperatorKindGe represents a >= b .
	OperatorKindGe
	// OperatorKindLt represents a < b .
	OperatorKindLt
	// OperatorKindLe represents a <= b .
	OperatorKindLe
	// OperatorKindPlus represents +a (unary plus).
	OperatorKindPlus
	// OperatorKindMinus represents -a (unary minus).
	OperatorKindMinus
	// OperatorKindNot represents !a .
	OperatorKindNot
	// OperatorKindBitwiseNot represents ~a .
	OperatorKindBitwiseNot
	// OperatorKindPrefixInc represents ++a .
	OperatorKindPrefixInc
	// OperatorKindPrefixDec represents --a .
	OperatorKindPrefixDec
	// OperatorKindPostfixInc represents a++ .
	OperatorKindPostfixInc
	// OperatorKindPostfixDec represents a-- .
	OperatorKindPostfixDec
)

// OperatorKindFromDecorator converts a decorator kind and its string argument
// to the corresponding OperatorKind.
func OperatorKindFromDecorator(decoratorKind ast.DecoratorKind, arg string) OperatorKind {
	switch decoratorKind {
	case ast.DecoratorKindOperator, ast.DecoratorKindOperatorBinary:
		if len(arg) == 0 {
			return OperatorKindInvalid
		}
		switch int32(arg[0]) {
		case util.CharCodeOpenBracket: // '['
			if arg == "[]" {
				return OperatorKindIndexedGet
			}
			if arg == "[]=" {
				return OperatorKindIndexedSet
			}
		case util.CharCodeOpenBrace: // '{'
			if arg == "{}" {
				return OperatorKindUncheckedIndexedGet
			}
			if arg == "{}=" {
				return OperatorKindUncheckedIndexedSet
			}
		case util.CharCodePlus: // '+'
			return OperatorKindAdd
		case util.CharCodeMinus: // '-'
			return OperatorKindSub
		case util.CharCodeAsterisk: // '*'
			if arg == "**" {
				return OperatorKindPow
			}
			return OperatorKindMul
		case util.CharCodeSlash: // '/'
			return OperatorKindDiv
		case util.CharCodePercent: // '%'
			return OperatorKindRem
		case util.CharCodeAmpersand: // '&'
			return OperatorKindBitwiseAnd
		case util.CharCodeBar: // '|'
			return OperatorKindBitwiseOr
		case util.CharCodeCaret: // '^'
			return OperatorKindBitwiseXor
		case util.CharCodeEquals: // '='
			if arg == "==" {
				return OperatorKindEq
			}
		case util.CharCodeExclamation: // '!'
			if arg == "!=" {
				return OperatorKindNe
			}
		case util.CharCodeGreaterThan: // '>'
			if arg == ">=" {
				return OperatorKindGe
			}
			if arg == ">>" {
				return OperatorKindBitwiseShr
			}
			if arg == ">>>" {
				return OperatorKindBitwiseShrU
			}
			return OperatorKindGt
		case util.CharCodeLessThan: // '<'
			if arg == "<=" {
				return OperatorKindLe
			}
			if arg == "<<" {
				return OperatorKindBitwiseShl
			}
			return OperatorKindLt
		}
		return OperatorKindInvalid

	case ast.DecoratorKindOperatorPrefix:
		if len(arg) == 0 {
			return OperatorKindInvalid
		}
		switch int32(arg[0]) {
		case util.CharCodePlus: // '+'
			if arg == "++" {
				return OperatorKindPrefixInc
			}
			return OperatorKindPlus
		case util.CharCodeMinus: // '-'
			if arg == "--" {
				return OperatorKindPrefixDec
			}
			return OperatorKindMinus
		case util.CharCodeExclamation: // '!'
			return OperatorKindNot
		case util.CharCodeTilde: // '~'
			return OperatorKindBitwiseNot
		}
		return OperatorKindInvalid

	case ast.DecoratorKindOperatorPostfix:
		if arg == "++" {
			return OperatorKindPostfixInc
		}
		if arg == "--" {
			return OperatorKindPostfixDec
		}
		return OperatorKindInvalid

	default:
		return OperatorKindInvalid
	}
}

// OperatorKindFromBinaryToken converts a binary operator token to the
// corresponding OperatorKind.
func OperatorKindFromBinaryToken(token tokenizer.Token) OperatorKind {
	switch token {
	case tokenizer.TokenPlus,
		tokenizer.TokenPlusEquals:
		return OperatorKindAdd
	case tokenizer.TokenMinus,
		tokenizer.TokenMinusEquals:
		return OperatorKindSub
	case tokenizer.TokenAsterisk,
		tokenizer.TokenAsteriskEquals:
		return OperatorKindMul
	case tokenizer.TokenAsteriskAsterisk,
		tokenizer.TokenAsteriskAsteriskEquals:
		return OperatorKindPow
	case tokenizer.TokenSlash,
		tokenizer.TokenSlashEquals:
		return OperatorKindDiv
	case tokenizer.TokenPercent,
		tokenizer.TokenPercentEquals:
		return OperatorKindRem
	case tokenizer.TokenAmpersand,
		tokenizer.TokenAmpersandEquals:
		return OperatorKindBitwiseAnd
	case tokenizer.TokenBar,
		tokenizer.TokenBarEquals:
		return OperatorKindBitwiseOr
	case tokenizer.TokenCaret,
		tokenizer.TokenCaretEquals:
		return OperatorKindBitwiseXor
	case tokenizer.TokenLessThanLessThan,
		tokenizer.TokenLessThanLessThanEquals:
		return OperatorKindBitwiseShl
	case tokenizer.TokenGreaterThanGreaterThan,
		tokenizer.TokenGreaterThanGreaterThanEquals:
		return OperatorKindBitwiseShr
	case tokenizer.TokenGreaterThanGreaterThanGreaterThan,
		tokenizer.TokenGreaterThanGreaterThanGreaterThanEquals:
		return OperatorKindBitwiseShrU
	case tokenizer.TokenEqualsEquals,
		tokenizer.TokenEqualsEqualsEquals:
		return OperatorKindEq
	case tokenizer.TokenExclamationEquals,
		tokenizer.TokenExclamationEqualsEquals:
		return OperatorKindNe
	case tokenizer.TokenGreaterThan:
		return OperatorKindGt
	case tokenizer.TokenGreaterThanEquals:
		return OperatorKindGe
	case tokenizer.TokenLessThan:
		return OperatorKindLt
	case tokenizer.TokenLessThanEquals:
		return OperatorKindLe
	default:
		return OperatorKindInvalid
	}
}

// OperatorKindFromUnaryPrefixToken converts a unary prefix operator token to
// the corresponding OperatorKind.
func OperatorKindFromUnaryPrefixToken(token tokenizer.Token) OperatorKind {
	switch token {
	case tokenizer.TokenPlus:
		return OperatorKindPlus
	case tokenizer.TokenMinus:
		return OperatorKindMinus
	case tokenizer.TokenExclamation:
		return OperatorKindNot
	case tokenizer.TokenTilde:
		return OperatorKindBitwiseNot
	case tokenizer.TokenPlusPlus:
		return OperatorKindPrefixInc
	case tokenizer.TokenMinusMinus:
		return OperatorKindPrefixDec
	default:
		return OperatorKindInvalid
	}
}

// OperatorKindFromUnaryPostfixToken converts a unary postfix operator token to
// the corresponding OperatorKind.
func OperatorKindFromUnaryPostfixToken(token tokenizer.Token) OperatorKind {
	switch token {
	case tokenizer.TokenPlusPlus:
		return OperatorKindPostfixInc
	case tokenizer.TokenMinusMinus:
		return OperatorKindPostfixDec
	default:
		return OperatorKindInvalid
	}
}
