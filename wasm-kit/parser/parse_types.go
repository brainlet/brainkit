package parser

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

// parseTypeName parses a type name: Identifier ('.' Identifier)*
func (p *Parser) parseTypeName(tn *tokenizer.Tokenizer) *ast.TypeName {
	first := ast.NewSimpleTypeName(tn.ReadIdentifier(), *tn.MakeRange(-1, -1))
	current := first
	for tn.Skip(tokenizer.TokenDot, tokenizer.IdentifierHandlingDefault) {
		if tn.Skip(tokenizer.TokenIdentifier, tokenizer.IdentifierHandlingDefault) {
			next := ast.NewSimpleTypeName(tn.ReadIdentifier(), *tn.MakeRange(-1, -1))
			current.Next = next
			current = next
		} else {
			p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(tn.Pos, -1))
			return nil
		}
	}
	return first
}

// parseType parses a type.
func (p *Parser) parseType(
	tn *tokenizer.Tokenizer,
	acceptParenthesized bool,
	suppressErrors bool,
) ast.Node {
	token := tn.Next(tokenizer.IdentifierHandlingDefault)
	startPos := tn.TokenPos

	var typ ast.Node

	if token == tokenizer.TokenOpenParen {
		// '(' ...
		isInnerParenthesized := tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault)
		signature := p.tryParseFunctionType(tn)
		if signature != nil {
			if isInnerParenthesized {
				if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
					if !suppressErrors {
						p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
					}
					return nil
				}
			}
			typ = signature
		} else if isInnerParenthesized || p.tryParseSignatureIsSignature {
			if !suppressErrors {
				p.error(diagnostics.DiagnosticCodeUnexpectedToken, tn.MakeRange(-1, -1))
			}
			return nil
		} else if acceptParenthesized {
			innerType := p.parseType(tn, false, suppressErrors)
			if innerType == nil {
				return nil
			}
			if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
				if !suppressErrors {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(tn.Pos, -1), ")")
				}
				return nil
			}
			typ = innerType
			typ.GetRange().Start = startPos
			typ.GetRange().End = tn.Pos
		} else {
			if !suppressErrors {
				p.error(diagnostics.DiagnosticCodeUnexpectedToken, tn.MakeRange(-1, -1))
			}
			return nil
		}

	} else if token == tokenizer.TokenVoid {
		typ = ast.NewNamedTypeNode(
			ast.NewSimpleTypeName("void", *tn.MakeRange(-1, -1)),
			nil, false, *tn.MakeRange(startPos, tn.Pos),
		)
	} else if token == tokenizer.TokenThis {
		typ = ast.NewNamedTypeNode(
			ast.NewSimpleTypeName("this", *tn.MakeRange(-1, -1)),
			nil, false, *tn.MakeRange(startPos, tn.Pos),
		)
	} else if token == tokenizer.TokenTrue || token == tokenizer.TokenFalse {
		typ = ast.NewNamedTypeNode(
			ast.NewSimpleTypeName("bool", *tn.MakeRange(-1, -1)),
			nil, false, *tn.MakeRange(startPos, tn.Pos),
		)
	} else if token == tokenizer.TokenNull {
		typ = ast.NewNamedTypeNode(
			ast.NewSimpleTypeName("null", *tn.MakeRange(-1, -1)),
			nil, false, *tn.MakeRange(startPos, tn.Pos),
		)
	} else if token == tokenizer.TokenStringLiteral {
		tn.ReadString(0, false)
		typ = ast.NewNamedTypeNode(
			ast.NewSimpleTypeName("string", *tn.MakeRange(-1, -1)),
			nil, false, *tn.MakeRange(startPos, tn.Pos),
		)
	} else if token == tokenizer.TokenIdentifier {
		name := p.parseTypeName(tn)
		if name == nil {
			return nil
		}
		var parameters []ast.Node

		// Name<T>
		if tn.Skip(tokenizer.TokenLessThan, tokenizer.IdentifierHandlingDefault) {
			for {
				parameter := p.parseType(tn, true, suppressErrors)
				if parameter == nil {
					return nil
				}
				parameters = append(parameters, parameter)
				if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
					break
				}
			}
			if !tn.Skip(tokenizer.TokenGreaterThan, tokenizer.IdentifierHandlingDefault) {
				if !suppressErrors {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(tn.Pos, -1), ">")
				}
				return nil
			}
		}
		if parameters == nil {
			parameters = []ast.Node{}
		}
		typ = ast.NewNamedTypeNode(name, parameters, false, *tn.MakeRange(startPos, tn.Pos))
	} else {
		if !suppressErrors {
			p.error(diagnostics.DiagnosticCodeTypeExpected, tn.MakeRange(-1, -1))
		}
		return nil
	}

	// ... | type (union types — only null union supported)
	for tn.Skip(tokenizer.TokenBar, tokenizer.IdentifierHandlingDefault) {
		nextType := p.parseType(tn, true, false)
		if nextType == nil {
			return nil
		}
		typeIsNull := typ.GetKind() == ast.NodeKindNamedType && typ.(*ast.NamedTypeNode).IsNull()
		nextTypeIsNull := nextType.GetKind() == ast.NodeKindNamedType && nextType.(*ast.NamedTypeNode).IsNull()
		if !typeIsNull && !nextTypeIsNull {
			if !suppressErrors {
				p.error(diagnostics.DiagnosticCodeNotImplemented0, nextType.GetRange(), "union types")
			}
			return nil
		} else if nextTypeIsNull {
			setNullable(typ, true)
			typ.GetRange().End = nextType.GetRange().End
		} else if typeIsNull {
			nextType.GetRange().Start = typ.GetRange().Start
			setNullable(nextType, true)
			typ = nextType
		} else {
			// null | null still null
			typ.GetRange().End = nextType.GetRange().End
		}
	}

	// ... [][]
	for tn.Skip(tokenizer.TokenOpenBracket, tokenizer.IdentifierHandlingDefault) {
		bracketStart := tn.TokenPos
		if !tn.Skip(tokenizer.TokenCloseBracket, tokenizer.IdentifierHandlingDefault) {
			if !suppressErrors {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "]")
			}
			return nil
		}
		bracketRange := *tn.MakeRange(bracketStart, tn.Pos)

		// ...[] | null
		nullable := false
		if tn.Skip(tokenizer.TokenBar, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenNull, tokenizer.IdentifierHandlingDefault) {
				nullable = true
			} else {
				if !suppressErrors {
					p.error(diagnostics.DiagnosticCodeNotImplemented0, tn.MakeRange(-1, -1), "union types")
				}
				return nil
			}
		}
		typ = ast.NewNamedTypeNode(
			ast.NewSimpleTypeName("Array", bracketRange),
			[]ast.Node{typ},
			nullable,
			*tn.MakeRange(startPos, tn.Pos),
		)
		if nullable {
			break
		}
	}

	return typ
}

// setNullable sets the IsNullable field on a type node.
func setNullable(n ast.Node, nullable bool) {
	switch v := n.(type) {
	case *ast.NamedTypeNode:
		v.IsNullable = nullable
	case *ast.FunctionTypeNode:
		v.IsNullable = nullable
	}
}

// tryParseFunctionType tries to parse a function type signature.
func (p *Parser) tryParseFunctionType(tn *tokenizer.Tokenizer) *ast.FunctionTypeNode {
	state := tn.Mark()
	startPos := tn.TokenPos
	var parameters []*ast.ParameterNode
	var thisType *ast.NamedTypeNode
	isSignature := false
	var firstParamNameNoType *ast.IdentifierExpression
	firstParamKind := ast.ParameterKindDefault

	if tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
		isSignature = true
		tn.Discard(state)
		parameters = []*ast.ParameterNode{}
	} else {
		for {
			paramStart := int32(-1)
			kind := ast.ParameterKindDefault
			if tn.Skip(tokenizer.TokenDotDotDot, tokenizer.IdentifierHandlingDefault) {
				paramStart = tn.TokenPos
				isSignature = true
				tn.Discard(state)
				kind = ast.ParameterKindRest
			}
			if tn.Skip(tokenizer.TokenThis, tokenizer.IdentifierHandlingDefault) {
				if paramStart < 0 {
					paramStart = tn.TokenPos
				}
				if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
					isSignature = true
					tn.Discard(state)
					typ := p.parseType(tn, false, false)
					if typ == nil {
						return nil
					}
					if typ.GetKind() != ast.NodeKindNamedType {
						p.error(diagnostics.DiagnosticCodeIdentifierExpected, typ.GetRange())
						p.tryParseSignatureIsSignature = true
						return nil
					}
					thisType = typ.(*ast.NamedTypeNode)
				} else {
					tn.Reset(state)
					p.tryParseSignatureIsSignature = false
					return nil
				}
			} else if tn.SkipIdentifier(tokenizer.IdentifierHandlingDefault) {
				if paramStart < 0 {
					paramStart = tn.TokenPos
				}
				name := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(tn.TokenPos, tn.Pos), false)
				if tn.Skip(tokenizer.TokenQuestion, tokenizer.IdentifierHandlingDefault) {
					isSignature = true
					tn.Discard(state)
					if kind == ast.ParameterKindRest {
						p.error(diagnostics.DiagnosticCodeARestParameterCannotBeOptional, tn.MakeRange(-1, -1))
					} else {
						kind = ast.ParameterKindOptional
					}
				}
				if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
					isSignature = true
					tn.Discard(state)
					typ := p.parseType(tn, true, false)
					if typ == nil {
						p.tryParseSignatureIsSignature = isSignature
						return nil
					}
					param := ast.NewParameterNode(kind, name, typ, nil, *tn.MakeRange(paramStart, tn.Pos))
					parameters = append(parameters, param)
				} else {
					if !isSignature {
						if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenComma {
							isSignature = true
							tn.Discard(state)
						}
					}
					if isSignature {
						param := ast.NewParameterNode(kind, name, ast.NewOmittedType(*tn.MakeRange(tn.Pos, -1)), nil, *tn.MakeRange(paramStart, tn.Pos))
						parameters = append(parameters, param)
						p.error(diagnostics.DiagnosticCodeTypeExpected, param.Type.GetRange())
					} else if parameters == nil {
						firstParamNameNoType = name
						firstParamKind = kind
					}
				}
			} else {
				if isSignature {
					if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenCloseParen {
						break // allow trailing comma
					}
					p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
				} else {
					tn.Reset(state)
				}
				p.tryParseSignatureIsSignature = isSignature
				return nil
			}
			if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
				break
			}
		}
		if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
			if isSignature {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
			} else {
				tn.Reset(state)
			}
			p.tryParseSignatureIsSignature = isSignature
			return nil
		}
	}

	var returnType ast.Node
	if tn.Skip(tokenizer.TokenEqualsGreaterThan, tokenizer.IdentifierHandlingDefault) {
		if !isSignature {
			isSignature = true
			tn.Discard(state)
			if firstParamNameNoType != nil {
				param := ast.NewParameterNode(
					firstParamKind,
					firstParamNameNoType,
					ast.NewOmittedType(*firstParamNameNoType.GetRange().AtEnd()),
					nil,
					firstParamNameNoType.Range,
				)
				parameters = append(parameters, param)
				p.error(diagnostics.DiagnosticCodeTypeExpected, param.Type.GetRange())
			}
		}
		returnType = p.parseType(tn, true, false)
		if returnType == nil {
			p.tryParseSignatureIsSignature = isSignature
			return nil
		}
	} else {
		if isSignature {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "=>")
		} else {
			tn.Reset(state)
		}
		p.tryParseSignatureIsSignature = isSignature
		return nil
	}
	p.tryParseSignatureIsSignature = true

	if parameters == nil {
		parameters = []*ast.ParameterNode{}
	}

	return ast.NewFunctionTypeNode(
		parameters,
		returnType,
		thisType,
		false,
		*tn.MakeRange(startPos, tn.Pos),
	)
}

// parseTypeParameters parses type parameters: '<' TypeParameter (',' TypeParameter)* '>'
func (p *Parser) parseTypeParameters(tn *tokenizer.Tokenizer) []*ast.TypeParameterNode {
	var typeParameters []*ast.TypeParameterNode
	seenOptional := false
	start := tn.TokenPos
	for !tn.Skip(tokenizer.TokenGreaterThan, tokenizer.IdentifierHandlingDefault) {
		typeParameter := p.parseTypeParameter(tn)
		if typeParameter == nil {
			return nil
		}
		if typeParameter.DefaultType != nil {
			seenOptional = true
		} else if seenOptional {
			p.error(diagnostics.DiagnosticCodeRequiredTypeParametersMayNotFollowOptionalTypeParameters, typeParameter.GetRange())
			typeParameter.DefaultType = nil
		}
		typeParameters = append(typeParameters, typeParameter)
		if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenGreaterThan, tokenizer.IdentifierHandlingDefault) {
				break
			} else {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ">")
				return nil
			}
		}
	}
	if len(typeParameters) == 0 {
		p.error(diagnostics.DiagnosticCodeTypeParameterListCannotBeEmpty, tn.MakeRange(start, tn.Pos))
	}
	return typeParameters
}

// parseTypeParameter parses a single type parameter: Identifier ('extends' Type)? ('=' Type)?
func (p *Parser) parseTypeParameter(tn *tokenizer.Tokenizer) *ast.TypeParameterNode {
	if tn.Next(tokenizer.IdentifierHandlingDefault) == tokenizer.TokenIdentifier {
		identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
		var extendsType *ast.NamedTypeNode
		if tn.Skip(tokenizer.TokenExtends, tokenizer.IdentifierHandlingDefault) {
			typ := p.parseType(tn, true, false)
			if typ == nil {
				return nil
			}
			if typ.GetKind() != ast.NodeKindNamedType {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, typ.GetRange())
				return nil
			}
			extendsType = typ.(*ast.NamedTypeNode)
		}
		var defaultType *ast.NamedTypeNode
		if tn.Skip(tokenizer.TokenEquals, tokenizer.IdentifierHandlingDefault) {
			typ := p.parseType(tn, true, false)
			if typ == nil {
				return nil
			}
			if typ.GetKind() != ast.NodeKindNamedType {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, typ.GetRange())
				return nil
			}
			defaultType = typ.(*ast.NamedTypeNode)
		}
		return ast.NewTypeParameterNode(
			identifier, extendsType, defaultType,
			*diagnostics.JoinRanges(identifier.GetRange(), tn.MakeRange(-1, -1)),
		)
	}
	p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	return nil
}
