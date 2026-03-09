package tokenizer

// Token represents named token types.
type Token int32

const (
	// keywords

	TokenAbstract Token = iota
	TokenAs
	TokenAsync
	TokenAwait
	TokenBreak
	TokenCase
	TokenCatch
	TokenClass
	TokenConst
	TokenContinue
	TokenConstructor
	TokenDebugger
	TokenDeclare
	TokenDefault
	TokenDelete
	TokenDo
	TokenElse
	TokenEnum
	TokenExport
	TokenExtends
	TokenFalse
	TokenFinally
	TokenFor
	TokenFrom
	TokenFunction
	TokenGet
	TokenIf
	TokenImplements
	TokenImport
	TokenIn
	TokenInstanceOf
	TokenInterface
	TokenIs
	TokenKeyOf
	TokenLet
	TokenModule
	TokenNamespace
	TokenNew
	TokenNull
	TokenOf
	TokenOverride
	TokenPackage
	TokenPrivate
	TokenProtected
	TokenPublic
	TokenReadonly
	TokenReturn
	TokenSet
	TokenStatic
	TokenSuper
	TokenSwitch
	TokenThis
	TokenThrow
	TokenTrue
	TokenTry
	TokenType
	TokenTypeOf
	TokenVar
	TokenVoid
	TokenWhile
	TokenWith
	TokenYield

	// punctuation

	TokenOpenBrace
	TokenCloseBrace
	TokenOpenParen
	TokenCloseParen
	TokenOpenBracket
	TokenCloseBracket
	TokenDot
	TokenDotDotDot
	TokenSemicolon
	TokenComma
	TokenLessThan
	TokenGreaterThan
	TokenLessThanEquals
	TokenGreaterThanEquals
	TokenEqualsEquals
	TokenExclamationEquals
	TokenEqualsEqualsEquals
	TokenExclamationEqualsEquals
	TokenEqualsGreaterThan
	TokenPlus
	TokenMinus
	TokenAsteriskAsterisk
	TokenAsterisk
	TokenSlash
	TokenPercent
	TokenPlusPlus
	TokenMinusMinus
	TokenLessThanLessThan
	TokenGreaterThanGreaterThan
	TokenGreaterThanGreaterThanGreaterThan
	TokenAmpersand
	TokenBar
	TokenCaret
	TokenExclamation
	TokenTilde
	TokenAmpersandAmpersand
	TokenBarBar
	TokenQuestion
	TokenColon
	TokenEquals
	TokenPlusEquals
	TokenMinusEquals
	TokenAsteriskEquals
	TokenAsteriskAsteriskEquals
	TokenSlashEquals
	TokenPercentEquals
	TokenLessThanLessThanEquals
	TokenGreaterThanGreaterThanEquals
	TokenGreaterThanGreaterThanGreaterThanEquals
	TokenAmpersandEquals
	TokenBarEquals
	TokenCaretEquals
	TokenAt

	// literals

	TokenIdentifier
	TokenStringLiteral
	TokenIntegerLiteral
	TokenFloatLiteral
	TokenTemplateLiteral

	// meta

	TokenInvalid
	TokenEndOfFile
)

// IdentifierHandling controls how the tokenizer resolves keywords vs identifiers.
type IdentifierHandling int32

const (
	IdentifierHandlingDefault IdentifierHandling = iota
	IdentifierHandlingPrefer
	IdentifierHandlingAlways
)

// CommentKind indicates the kind of a comment.
type CommentKind int32

const (
	CommentKindLine   CommentKind = iota
	CommentKindTriple
	CommentKindBlock
)

// CommentHandler is a callback for intercepting comments while tokenizing.
type CommentHandler func(kind CommentKind, text string, rng interface{})

// onNewLine tracks whether a token begins on a new line.
type onNewLine int32

const (
	onNewLineNo onNewLine = iota
	onNewLineYes
	onNewLineUnknown
)

// TokenFromKeyword returns the token for a keyword string, or TokenInvalid.
func TokenFromKeyword(text string) Token {
	n := len(text)
	if n == 0 {
		panic("assertion failed: keyword must not be empty")
	}
	switch text[0] {
	case 'a':
		if n == 5 {
			if text == "async" {
				return TokenAsync
			}
			if text == "await" {
				return TokenAwait
			}
			break
		}
		if text == "as" {
			return TokenAs
		}
		if text == "abstract" {
			return TokenAbstract
		}
	case 'b':
		if text == "break" {
			return TokenBreak
		}
	case 'c':
		if n == 5 {
			if text == "const" {
				return TokenConst
			}
			if text == "class" {
				return TokenClass
			}
			if text == "catch" {
				return TokenCatch
			}
			break
		}
		if text == "case" {
			return TokenCase
		}
		if text == "continue" {
			return TokenContinue
		}
		if text == "constructor" {
			return TokenConstructor
		}
	case 'd':
		if n == 7 {
			if text == "default" {
				return TokenDefault
			}
			if text == "declare" {
				return TokenDeclare
			}
			break
		}
		if text == "do" {
			return TokenDo
		}
		if text == "delete" {
			return TokenDelete
		}
		if text == "debugger" {
			return TokenDebugger
		}
	case 'e':
		if n == 4 {
			if text == "else" {
				return TokenElse
			}
			if text == "enum" {
				return TokenEnum
			}
			break
		}
		if text == "export" {
			return TokenExport
		}
		if text == "extends" {
			return TokenExtends
		}
	case 'f':
		if n <= 5 {
			if text == "false" {
				return TokenFalse
			}
			if text == "for" {
				return TokenFor
			}
			if text == "from" {
				return TokenFrom
			}
			break
		}
		if text == "function" {
			return TokenFunction
		}
		if text == "finally" {
			return TokenFinally
		}
	case 'g':
		if text == "get" {
			return TokenGet
		}
	case 'i':
		if n == 2 {
			if text == "if" {
				return TokenIf
			}
			if text == "in" {
				return TokenIn
			}
			if text == "is" {
				return TokenIs
			}
			break
		}
		if n > 3 {
			switch text[3] {
			case 'l':
				if text == "implements" {
					return TokenImplements
				}
			case 'o':
				if text == "import" {
					return TokenImport
				}
			case 't':
				if text == "instanceof" {
					return TokenInstanceOf
				}
			case 'e':
				if text == "interface" {
					return TokenInterface
				}
			}
		}
	case 'k':
		if text == "keyof" {
			return TokenKeyOf
		}
	case 'l':
		if text == "let" {
			return TokenLet
		}
	case 'm':
		if text == "module" {
			return TokenModule
		}
	case 'n':
		if text == "new" {
			return TokenNew
		}
		if text == "null" {
			return TokenNull
		}
		if text == "namespace" {
			return TokenNamespace
		}
	case 'o':
		if text == "of" {
			return TokenOf
		}
		if text == "override" {
			return TokenOverride
		}
	case 'p':
		if n == 7 {
			if text == "private" {
				return TokenPrivate
			}
			if text == "package" {
				return TokenPackage
			}
			break
		}
		if text == "public" {
			return TokenPublic
		}
		if text == "protected" {
			return TokenProtected
		}
	case 'r':
		if text == "return" {
			return TokenReturn
		}
		if text == "readonly" {
			return TokenReadonly
		}
	case 's':
		if n == 6 {
			if text == "switch" {
				return TokenSwitch
			}
			if text == "static" {
				return TokenStatic
			}
			break
		}
		if text == "set" {
			return TokenSet
		}
		if text == "super" {
			return TokenSuper
		}
	case 't':
		if n == 4 {
			if text == "true" {
				return TokenTrue
			}
			if text == "this" {
				return TokenThis
			}
			if text == "type" {
				return TokenType
			}
			break
		}
		if text == "try" {
			return TokenTry
		}
		if text == "throw" {
			return TokenThrow
		}
		if text == "typeof" {
			return TokenTypeOf
		}
	case 'v':
		if text == "var" {
			return TokenVar
		}
		if text == "void" {
			return TokenVoid
		}
	case 'w':
		if text == "while" {
			return TokenWhile
		}
		if text == "with" {
			return TokenWith
		}
	case 'y':
		if text == "yield" {
			return TokenYield
		}
	}
	return TokenInvalid
}

// TokenIsAlsoIdentifier returns true if a keyword token can also be used as an identifier.
func TokenIsAlsoIdentifier(token Token) bool {
	switch token {
	case TokenAbstract,
		TokenAs,
		TokenConstructor,
		TokenDeclare,
		TokenDelete,
		TokenFrom,
		TokenFor,
		TokenGet,
		TokenInstanceOf,
		TokenIs,
		TokenKeyOf,
		TokenModule,
		TokenNamespace,
		TokenNull,
		TokenReadonly,
		TokenSet,
		TokenType,
		TokenVoid:
		return true
	}
	return false
}

// IsIllegalVariableIdentifier returns true if the name is an illegal variable identifier.
func IsIllegalVariableIdentifier(name string) bool {
	if len(name) == 0 {
		panic("assertion failed: name must not be empty")
	}
	switch name[0] {
	case 'd':
		return name == "delete"
	case 'f':
		return name == "for"
	case 'i':
		return name == "instanceof"
	case 'n':
		return name == "null"
	case 'v':
		return name == "void"
	}
	return false
}

// OperatorTokenToString returns the string representation of an operator token.
func OperatorTokenToString(token Token) string {
	switch token {
	case TokenDelete:
		return "delete"
	case TokenIn:
		return "in"
	case TokenInstanceOf:
		return "instanceof"
	case TokenNew:
		return "new"
	case TokenTypeOf:
		return "typeof"
	case TokenVoid:
		return "void"
	case TokenYield:
		return "yield"
	case TokenDotDotDot:
		return "..."
	case TokenComma:
		return ","
	case TokenLessThan:
		return "<"
	case TokenGreaterThan:
		return ">"
	case TokenLessThanEquals:
		return "<="
	case TokenGreaterThanEquals:
		return ">="
	case TokenEqualsEquals:
		return "=="
	case TokenExclamationEquals:
		return "!="
	case TokenEqualsEqualsEquals:
		return "==="
	case TokenExclamationEqualsEquals:
		return "!=="
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenAsteriskAsterisk:
		return "**"
	case TokenAsterisk:
		return "*"
	case TokenSlash:
		return "/"
	case TokenPercent:
		return "%"
	case TokenPlusPlus:
		return "++"
	case TokenMinusMinus:
		return "--"
	case TokenLessThanLessThan:
		return "<<"
	case TokenGreaterThanGreaterThan:
		return ">>"
	case TokenGreaterThanGreaterThanGreaterThan:
		return ">>>"
	case TokenAmpersand:
		return "&"
	case TokenBar:
		return "|"
	case TokenCaret:
		return "^"
	case TokenExclamation:
		return "!"
	case TokenTilde:
		return "~"
	case TokenAmpersandAmpersand:
		return "&&"
	case TokenBarBar:
		return "||"
	case TokenEquals:
		return "="
	case TokenPlusEquals:
		return "+="
	case TokenMinusEquals:
		return "-="
	case TokenAsteriskEquals:
		return "*="
	case TokenAsteriskAsteriskEquals:
		return "**="
	case TokenSlashEquals:
		return "/="
	case TokenPercentEquals:
		return "%="
	case TokenLessThanLessThanEquals:
		return "<<="
	case TokenGreaterThanGreaterThanEquals:
		return ">>="
	case TokenGreaterThanGreaterThanGreaterThanEquals:
		return ">>>="
	case TokenAmpersandEquals:
		return "&="
	case TokenBarEquals:
		return "|="
	case TokenCaretEquals:
		return "^="
	default:
		panic("not an operator token")
	}
}
