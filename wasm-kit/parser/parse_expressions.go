package parser

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// parseExpressionStart parses the start of an expression (prefix position).
func (p *Parser) parseExpressionStart(tn *tokenizer.Tokenizer) ast.Node {
	token := tn.Next(tokenizer.IdentifierHandlingPrefer)
	startPos := tn.TokenPos
	switch token {

	// TODO: SpreadExpression, YieldExpression
	case tokenizer.TokenDotDotDot,
		tokenizer.TokenYield:
		// fallthrough to unsupported UnaryPrefixExpression
		fallthrough

	// UnaryPrefixExpression
	case tokenizer.TokenExclamation,
		tokenizer.TokenTilde,
		tokenizer.TokenPlus,
		tokenizer.TokenMinus,
		tokenizer.TokenTypeOf,
		tokenizer.TokenVoid,
		tokenizer.TokenDelete:
		operand := p.parseExpression(tn, PrecedenceUnaryPrefix)
		if operand == nil {
			return nil
		}
		return ast.NewUnaryPrefixExpression(token, operand, *tn.MakeRange(startPos, tn.Pos))

	case tokenizer.TokenPlusPlus,
		tokenizer.TokenMinusMinus:
		operand := p.parseExpression(tn, PrecedenceUnaryPrefix)
		if operand == nil {
			return nil
		}
		switch operand.GetKind() {
		case ast.NodeKindIdentifier,
			ast.NodeKindElementAccess,
			ast.NodeKindPropertyAccess:
			// ok
		default:
			p.error(
				diagnostics.DiagnosticCodeTheOperandOfAnIncrementOrDecrementOperatorMustBeAVariableOrAPropertyAccess,
				operand.GetRange(),
			)
		}
		return ast.NewUnaryPrefixExpression(token, operand, *tn.MakeRange(startPos, tn.Pos))

	// NewExpression
	case tokenizer.TokenNew:
		if !tn.SkipIdentifier(tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
			return nil
		}
		typeName := p.parseTypeName(tn)
		if typeName == nil {
			return nil
		}
		var typeArguments []ast.Node
		var arguments []ast.Node
		if tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
			arguments = p.parseArguments(tn)
			if arguments == nil {
				return nil
			}
		} else {
			typeArguments = p.tryParseTypeArgumentsBeforeArguments(tn)
			if typeArguments != nil {
				arguments = p.parseArguments(tn)
				if arguments == nil {
					return nil
				}
			} else {
				arguments = []ast.Node{} // new Type;
			}
		}
		return ast.NewNewExpression(typeName, typeArguments, arguments, *tn.MakeRange(startPos, tn.Pos))

	// Special IdentifierExpression
	case tokenizer.TokenNull:
		return ast.NewNullExpression(*tn.MakeRange(-1, -1))
	case tokenizer.TokenTrue:
		return ast.NewTrueExpression(*tn.MakeRange(-1, -1))
	case tokenizer.TokenFalse:
		return ast.NewFalseExpression(*tn.MakeRange(-1, -1))
	case tokenizer.TokenThis:
		return ast.NewThisExpression(*tn.MakeRange(-1, -1))
	case tokenizer.TokenConstructor:
		return ast.NewConstructorExpression(*tn.MakeRange(-1, -1))

	// ParenthesizedExpression or FunctionExpression
	case tokenizer.TokenOpenParen:
		// determine whether this is a function expression
		if tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
			// must be a function expression (fast route)
			return p.parseFunctionExpressionCommon(
				tn,
				ast.NewEmptyIdentifierExpression(*tn.MakeRange(startPos, -1)),
				nil,
				nil,
				ast.ArrowKindParenthesized,
				-1,
				-1,
			)
		}
		state := tn.Mark()
		again := true
		for again {
			switch tn.Next(tokenizer.IdentifierHandlingPrefer) {

			// function expression
			case tokenizer.TokenDotDotDot:
				tn.Reset(state)
				return p.parseFunctionExpression(tn)

			// can be both
			case tokenizer.TokenIdentifier:
				tn.ReadIdentifier()
				switch tn.Next(tokenizer.IdentifierHandlingDefault) {

				// if we got here, check for arrow
				case tokenizer.TokenCloseParen:
					// `Identifier):Type =>` is function expression
					if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
						typ := p.parseType(tn, true, true)
						if typ == nil {
							again = false
							continue
						}
					}
					if !tn.Skip(tokenizer.TokenEqualsGreaterThan, tokenizer.IdentifierHandlingDefault) {
						again = false
						continue
					}
					// fall-through to Colon case
					tn.Reset(state)
					return p.parseFunctionExpression(tn)

				// function expression
				case tokenizer.TokenColon:
					tn.Reset(state)
					return p.parseFunctionExpression(tn)

				// optional parameter or parenthesized
				case tokenizer.TokenQuestion:
					if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) ||
						tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) ||
						tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
						tn.Reset(state)
						return p.parseFunctionExpression(tn)
					}
					again = false // parenthesized

				case tokenizer.TokenComma:
					// continue

				// parenthesized expression
				default:
					again = false
				}

			// parenthesized expression
			default:
				again = false
			}
		}
		tn.Reset(state)

		// parse parenthesized
		inner := p.parseExpression(tn, PrecedenceComma)
		if inner == nil {
			return nil
		}
		if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
			return nil
		}
		inner = ast.NewParenthesizedExpression(inner, *tn.MakeRange(startPos, tn.Pos))
		return p.maybeParseCallExpression(tn, inner, false)

	// ArrayLiteralExpression
	case tokenizer.TokenOpenBracket:
		var elementExpressions []ast.Node
		for !tn.Skip(tokenizer.TokenCloseBracket, tokenizer.IdentifierHandlingDefault) {
			var expr ast.Node
			if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenComma {
				expr = ast.NewOmittedExpression(*tn.MakeRange(tn.Pos, -1))
			} else {
				expr = p.parseExpression(tn, PrecedenceComma+1)
				if expr == nil {
					return nil
				}
			}
			elementExpressions = append(elementExpressions, expr)
			if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
				if tn.Skip(tokenizer.TokenCloseBracket, tokenizer.IdentifierHandlingDefault) {
					break
				} else {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "]")
					return nil
				}
			}
		}
		return ast.NewArrayLiteralExpression(elementExpressions, *tn.MakeRange(startPos, tn.Pos))

	// ObjectLiteralExpression
	case tokenizer.TokenOpenBrace:
		objStartPos := tn.TokenPos
		var names []*ast.IdentifierExpression
		var values []ast.Node
		for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
			var name *ast.IdentifierExpression
			if !tn.SkipIdentifier(tokenizer.IdentifierHandlingDefault) {
				if !tn.Skip(tokenizer.TokenStringLiteral, tokenizer.IdentifierHandlingDefault) {
					p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
					return nil
				}
				name = ast.NewIdentifierExpression(tn.ReadString(0, false), *tn.MakeRange(-1, -1), false)
				name.IsQuoted = true
			} else {
				name = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
			}
			names = append(names, name)
			if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
				value := p.parseExpression(tn, PrecedenceComma+1)
				if value == nil {
					return nil
				}
				values = append(values, value)
			} else if !name.IsQuoted {
				values = append(values, name)
			} else {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ":")
				return nil
			}
			if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
				if tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
					break
				} else {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
					return nil
				}
			}
		}
		return ast.NewObjectLiteralExpression(names, values, *tn.MakeRange(objStartPos, tn.Pos))

	// AssertionExpression (unary prefix)
	case tokenizer.TokenLessThan:
		toType := p.parseType(tn, false, false)
		if toType == nil {
			return nil
		}
		if !tn.Skip(tokenizer.TokenGreaterThan, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ">")
			return nil
		}
		expr := p.parseExpression(tn, PrecedenceCall)
		if expr == nil {
			return nil
		}
		return ast.NewAssertionExpression(
			ast.AssertionKindPrefix,
			expr,
			toType,
			*tn.MakeRange(startPos, tn.Pos),
		)

	case tokenizer.TokenIdentifier:
		identifierText := tn.ReadIdentifier()
		if identifierText == "null" {
			return ast.NewNullExpression(*tn.MakeRange(-1, -1)) // special
		}
		identifier := ast.NewIdentifierExpression(identifierText, *tn.MakeRange(startPos, tn.Pos), false)
		if tn.Skip(tokenizer.TokenTemplateLiteral, tokenizer.IdentifierHandlingDefault) {
			return p.parseTemplateLiteral(tn, identifier)
		}
		if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenEqualsGreaterThan && !tn.PeekOnNewLine() {
			return p.parseFunctionExpressionCommon(
				tn,
				ast.NewEmptyIdentifierExpression(*tn.MakeRange(startPos, -1)),
				[]*ast.ParameterNode{
					ast.NewParameterNode(
						ast.ParameterKindDefault,
						identifier,
						ast.NewOmittedType(*identifier.GetRange().AtEnd()),
						nil,
						*identifier.GetRange(),
					),
				},
				nil,
				ast.ArrowKindSingle,
				startPos,
				-1,
			)
		}
		return p.maybeParseCallExpression(tn, identifier, true)

	case tokenizer.TokenSuper:
		nextTok := tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32)
		if nextTok != tokenizer.TokenDot && nextTok != tokenizer.TokenOpenParen {
			p.error(
				diagnostics.DiagnosticCodeSuperMustBeFollowedByAnArgumentListOrMemberAccess,
				tn.MakeRange(-1, -1),
			)
		}
		expr := ast.NewSuperExpression(*tn.MakeRange(startPos, tn.Pos))
		return p.maybeParseCallExpression(tn, expr, false)

	case tokenizer.TokenStringLiteral:
		return ast.NewStringLiteralExpression(tn.ReadString(0, false), *tn.MakeRange(startPos, tn.Pos))

	case tokenizer.TokenTemplateLiteral:
		return p.parseTemplateLiteral(tn, nil)

	case tokenizer.TokenIntegerLiteral:
		value := tn.ReadInteger()
		tn.CheckForIdentifierStartAfterNumericLiteral()
		return ast.NewIntegerLiteralExpression(value, *tn.MakeRange(startPos, tn.Pos))

	case tokenizer.TokenFloatLiteral:
		value := tn.ReadFloat()
		tn.CheckForIdentifierStartAfterNumericLiteral()
		return ast.NewFloatLiteralExpression(value, *tn.MakeRange(startPos, tn.Pos))

	// RegexpLiteralExpression
	case tokenizer.TokenSlash:
		regexpPattern := tn.ReadRegexpPattern()
		if !tn.Skip(tokenizer.TokenSlash, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "/")
			return nil
		}
		return ast.NewRegexpLiteralExpression(
			regexpPattern,
			tn.ReadRegexpFlags(),
			*tn.MakeRange(startPos, tn.Pos),
		)

	case tokenizer.TokenFunction:
		expr := p.parseFunctionExpression(tn)
		if expr == nil {
			return nil
		}
		return p.maybeParseCallExpression(tn, expr, false)

	case tokenizer.TokenClass:
		return p.parseClassExpression(tn)

	default:
		if token == tokenizer.TokenEndOfFile {
			p.error(diagnostics.DiagnosticCodeUnexpectedEndOfText, tn.MakeRange(startPos, -1))
		} else {
			p.error(diagnostics.DiagnosticCodeExpressionExpected, tn.MakeRange(-1, -1))
		}
		return nil
	}
}

// tryParseTypeArgumentsBeforeArguments tries to parse type arguments before argument list.
// at '<': Type (',' Type)* '>' '('
func (p *Parser) tryParseTypeArgumentsBeforeArguments(tn *tokenizer.Tokenizer) []ast.Node {
	state := tn.Mark()
	if !tn.Skip(tokenizer.TokenLessThan, tokenizer.IdentifierHandlingDefault) {
		return nil
	}
	start := tn.TokenPos
	var typeArguments []ast.Node
	for {
		if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenGreaterThan {
			break
		}
		typ := p.parseType(tn, true, true)
		if typ == nil {
			tn.Reset(state)
			return nil
		}
		typeArguments = append(typeArguments, typ)
		if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			break
		}
	}
	if tn.Skip(tokenizer.TokenGreaterThan, tokenizer.IdentifierHandlingDefault) {
		end := tn.Pos
		if tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
			if typeArguments == nil {
				p.error(
					diagnostics.DiagnosticCodeTypeArgumentListCannotBeEmpty,
					tn.MakeRange(start, end),
				)
			}
			return typeArguments
		}
	}
	tn.Reset(state)
	return nil
}

// parseArguments parses an argument list.
// at '(': (Expression (',' Expression)*)? ')'
func (p *Parser) parseArguments(tn *tokenizer.Tokenizer) []ast.Node {
	var args []ast.Node
	for !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
		expr := p.parseExpression(tn, PrecedenceComma+1)
		if expr == nil {
			return nil
		}
		args = append(args, expr)
		if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
				break
			} else {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
				return nil
			}
		}
	}
	if args == nil {
		args = []ast.Node{}
	}
	return args
}

// parseExpression parses an expression using precedence climbing.
func (p *Parser) parseExpression(tn *tokenizer.Tokenizer, precedence Precedence) ast.Node {
	if precedence == PrecedenceNone {
		panic("precedence must not be None")
	}
	expr := p.parseExpressionStart(tn)
	if expr == nil {
		return nil
	}
	startPos := expr.GetRange().Start

	// precedence climbing
	// see: http://www.engr.mun.ca/~theo/Misc/exp_parsing.htm#climbing
	for {
		nextPrecedence := determinePrecedence(tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32))
		if nextPrecedence < precedence {
			break
		}
		token := tn.Next(tokenizer.IdentifierHandlingDefault)
		switch token {

		// AssertionExpression
		case tokenizer.TokenAs:
			if tn.Skip(tokenizer.TokenConst, tokenizer.IdentifierHandlingDefault) {
				expr = ast.NewAssertionExpression(
					ast.AssertionKindConst,
					expr,
					nil,
					*tn.MakeRange(startPos, tn.Pos),
				)
			} else {
				toType := p.parseType(tn, false, false)
				if toType == nil {
					return nil
				}
				expr = ast.NewAssertionExpression(
					ast.AssertionKindAs,
					expr,
					toType,
					*tn.MakeRange(startPos, tn.Pos),
				)
			}

		case tokenizer.TokenExclamation:
			expr = ast.NewAssertionExpression(
				ast.AssertionKindNonNull,
				expr,
				nil,
				*tn.MakeRange(startPos, tn.Pos),
			)
			expr = p.maybeParseCallExpression(tn, expr, false)

		// InstanceOfExpression
		case tokenizer.TokenInstanceOf:
			isType := p.parseType(tn, false, false)
			if isType == nil {
				return nil
			}
			expr = ast.NewInstanceOfExpression(
				expr,
				isType,
				*tn.MakeRange(startPos, tn.Pos),
			)

		// ElementAccessExpression
		case tokenizer.TokenOpenBracket:
			next := p.parseExpression(tn, PrecedenceComma)
			if next == nil {
				return nil
			}
			if !tn.Skip(tokenizer.TokenCloseBracket, tokenizer.IdentifierHandlingDefault) {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "]")
				return nil
			}
			expr = ast.NewElementAccessExpression(
				expr,
				next,
				*tn.MakeRange(startPos, tn.Pos),
			)
			expr = p.maybeParseCallExpression(tn, expr, false)

		// UnaryPostfixExpression
		case tokenizer.TokenPlusPlus,
			tokenizer.TokenMinusMinus:
			if expr.GetKind() != ast.NodeKindIdentifier &&
				expr.GetKind() != ast.NodeKindElementAccess &&
				expr.GetKind() != ast.NodeKindPropertyAccess {
				p.error(
					diagnostics.DiagnosticCodeTheOperandOfAnIncrementOrDecrementOperatorMustBeAVariableOrAPropertyAccess,
					expr.GetRange(),
				)
			}
			expr = ast.NewUnaryPostfixExpression(
				token,
				expr,
				*tn.MakeRange(startPos, tn.Pos),
			)

		// TernaryExpression
		case tokenizer.TokenQuestion:
			ifThen := p.parseExpression(tn, PrecedenceComma)
			if ifThen == nil {
				return nil
			}
			if !tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ":")
				return nil
			}
			ifElsePrec := PrecedenceComma
			if precedence > PrecedenceComma {
				ifElsePrec = PrecedenceComma + 1
			}
			ifElse := p.parseExpression(tn, ifElsePrec)
			if ifElse == nil {
				return nil
			}
			expr = ast.NewTernaryExpression(
				expr,
				ifThen,
				ifElse,
				*tn.MakeRange(startPos, tn.Pos),
			)

		// CommaExpression
		case tokenizer.TokenComma:
			commaExprs := []ast.Node{expr}
			for {
				expr = p.parseExpression(tn, PrecedenceComma+1)
				if expr == nil {
					return nil
				}
				commaExprs = append(commaExprs, expr)
				if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
					break
				}
			}
			expr = ast.NewCommaExpression(commaExprs, *tn.MakeRange(startPos, tn.Pos))

		// PropertyAccessExpression
		case tokenizer.TokenDot:
			if tn.SkipIdentifier(tokenizer.IdentifierHandlingAlways) {
				next := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
				expr = ast.NewPropertyAccessExpression(
					expr,
					next,
					*tn.MakeRange(startPos, tn.Pos),
				)
			} else {
				next := p.parseExpression(tn, nextPrecedence+1)
				if next == nil {
					return nil
				}
				if next.GetKind() == ast.NodeKindCall {
					expr = p.joinPropertyCall(tn, startPos, expr, next.(*ast.CallExpression))
					if expr == nil {
						return nil
					}
				} else {
					p.error(diagnostics.DiagnosticCodeIdentifierExpected, next.GetRange())
					return nil
				}
			}
			if tn.Skip(tokenizer.TokenTemplateLiteral, tokenizer.IdentifierHandlingDefault) {
				expr = p.parseTemplateLiteral(tn, expr)
				if expr == nil {
					return nil
				}
			} else {
				expr = p.maybeParseCallExpression(tn, expr, true)
			}

		// BinaryExpression (right associative)
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
			tokenizer.TokenBarEquals,
			tokenizer.TokenAsteriskAsterisk:
			next := p.parseExpression(tn, nextPrecedence)
			if next == nil {
				return nil
			}
			expr = ast.NewBinaryExpression(token, expr, next, *tn.MakeRange(startPos, tn.Pos))

		// BinaryExpression (left associative)
		case tokenizer.TokenLessThan,
			tokenizer.TokenGreaterThan,
			tokenizer.TokenLessThanEquals,
			tokenizer.TokenGreaterThanEquals,
			tokenizer.TokenEqualsEquals,
			tokenizer.TokenEqualsEqualsEquals,
			tokenizer.TokenExclamationEqualsEquals,
			tokenizer.TokenExclamationEquals,
			tokenizer.TokenPlus,
			tokenizer.TokenMinus,
			tokenizer.TokenAsterisk,
			tokenizer.TokenSlash,
			tokenizer.TokenPercent,
			tokenizer.TokenLessThanLessThan,
			tokenizer.TokenGreaterThanGreaterThan,
			tokenizer.TokenGreaterThanGreaterThanGreaterThan,
			tokenizer.TokenAmpersand,
			tokenizer.TokenBar,
			tokenizer.TokenCaret,
			tokenizer.TokenAmpersandAmpersand,
			tokenizer.TokenBarBar,
			tokenizer.TokenIn:
			next := p.parseExpression(tn, nextPrecedence+1)
			if next == nil {
				return nil
			}
			expr = ast.NewBinaryExpression(token, expr, next, *tn.MakeRange(startPos, tn.Pos))

		default:
			panic("filtered by determinePrecedence")
		}
	}
	return expr
}

// parseTemplateLiteral parses a template literal expression.
// at '`': ... '`'
func (p *Parser) parseTemplateLiteral(tn *tokenizer.Tokenizer, tag ast.Node) ast.Node {
	startPos := tn.TokenPos
	if tag != nil {
		startPos = tag.GetRange().Start
	}
	var parts []string
	var rawParts []string
	var exprs []ast.Node
	parts = append(parts, tn.ReadString(0, tag != nil))
	rawParts = append(rawParts, p.currentSource.Text[tn.ReadStringStart:tn.ReadStringEnd])
	for tn.ReadingTemplateString {
		expr := p.parseExpression(tn, PrecedenceComma)
		if expr == nil {
			return nil
		}
		exprs = append(exprs, expr)
		if !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
			return nil
		}
		parts = append(parts, tn.ReadString(util.CharCodeBacktick, tag != nil))
		rawParts = append(rawParts, p.currentSource.Text[tn.ReadStringStart:tn.ReadStringEnd])
	}
	return ast.NewTemplateLiteralExpression(tag, parts, rawParts, exprs, *tn.MakeRange(startPos, tn.Pos))
}

// joinPropertyCall joins a property access with a call expression.
func (p *Parser) joinPropertyCall(
	tn *tokenizer.Tokenizer,
	startPos int32,
	expr ast.Node,
	call *ast.CallExpression,
) ast.Node {
	callee := call.Expression
	switch callee.GetKind() {
	case ast.NodeKindIdentifier:
		// join property access and use as call target
		call.Expression = ast.NewPropertyAccessExpression(
			expr,
			callee.(*ast.IdentifierExpression),
			*tn.MakeRange(startPos, tn.Pos),
		)
	case ast.NodeKindCall:
		// join call target and wrap the original call around it
		inner := p.joinPropertyCall(tn, startPos, expr, callee.(*ast.CallExpression))
		if inner == nil {
			return nil
		}
		call.Expression = inner
		call.Range = *tn.MakeRange(startPos, tn.Pos)
	default:
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, call.GetRange())
		return nil
	}
	return call
}

// maybeParseCallExpression checks if the expression is followed by a call.
func (p *Parser) maybeParseCallExpression(
	tn *tokenizer.Tokenizer,
	expr ast.Node,
	potentiallyGeneric bool,
) ast.Node {
	var typeArguments []ast.Node
	for {
		if tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
			args := p.parseArguments(tn)
			if args == nil {
				break
			}
			expr = ast.NewCallExpression(
				expr,
				typeArguments,
				args,
				*tn.MakeRange(expr.GetRange().Start, tn.Pos),
			)
			typeArguments = nil
			potentiallyGeneric = false
			continue
		}
		if potentiallyGeneric {
			typeArguments = p.tryParseTypeArgumentsBeforeArguments(tn)
			if typeArguments != nil {
				continue
			}
		}
		break
	}
	return expr
}

// Ensure util package is used.
var _ = util.CharCodeBacktick
