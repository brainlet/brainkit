package parser

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

// parseDecorator parses a decorator: '@' Identifier ('.' Identifier)* ('(' Arguments ')')?
func (p *Parser) parseDecorator(tn *tokenizer.Tokenizer) *ast.DecoratorNode {
	startPos := tn.TokenPos
	if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		name := tn.ReadIdentifier()
		var expression ast.Node = ast.NewIdentifierExpression(name, *tn.MakeRange(startPos, tn.Pos), false)
		for tn.Skip(tokenizer.TokenDot, tokenizer.IdentifierHandlingDefault) {
			if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
				propName := tn.ReadIdentifier()
				expression = ast.NewPropertyAccessExpression(
					expression,
					ast.NewIdentifierExpression(propName, *tn.MakeRange(-1, -1), false),
					*tn.MakeRange(startPos, tn.Pos),
				)
			} else {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
				return nil
			}
		}
		if tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
			args := p.parseArguments(tn)
			if args != nil {
				return ast.NewDecoratorNode(expression, args, *tn.MakeRange(startPos, tn.Pos))
			}
		} else {
			return ast.NewDecoratorNode(expression, nil, *tn.MakeRange(startPos, tn.Pos))
		}
	} else {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	}
	return nil
}

// parseVariable parses a variable statement: ('const' | 'let' | 'var') VariableDeclaration (',' VariableDeclaration)* ';'?
func (p *Parser) parseVariable(
	tn *tokenizer.Tokenizer,
	flags int32,
	decorators []*ast.DecoratorNode,
	startPos int32,
	isFor bool,
) *ast.VariableStatement {
	var declarations []*ast.VariableDeclaration
	for {
		declaration := p.parseVariableDeclaration(tn, flags, decorators, isFor)
		if declaration == nil {
			return nil
		}
		declaration.OverriddenModuleName = p.currentModuleName
		declaration.HasOverriddenModule = p.currentModuleName != ""
		declarations = append(declarations, declaration)
		if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			break
		}
	}
	ret := ast.NewVariableStatement(decorators, declarations, *tn.MakeRange(startPos, tn.Pos))
	if !tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault) && !isFor {
		p.checkASI(tn)
	}
	return ret
}

// parseVariableDeclaration parses a single variable declaration: Identifier (':' Type)? ('=' Expression)?
func (p *Parser) parseVariableDeclaration(
	tn *tokenizer.Tokenizer,
	parentFlags int32,
	parentDecorators []*ast.DecoratorNode,
	isFor bool,
) *ast.VariableDeclaration {
	if !tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
		return nil
	}
	identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	if tokenizer.IsIllegalVariableIdentifier(identifier.Text) {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, &identifier.Range)
	}
	flags := parentFlags
	if tn.Skip(tokenizer.TokenExclamation, tokenizer.IdentifierHandlingDefault) {
		flags |= cf(common.CommonFlagsDefinitelyAssigned)
	}

	var typ ast.Node
	if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
		typ = p.parseType(tn, true, false)
	}

	var initializer ast.Node
	if tn.Skip(tokenizer.TokenEquals, tokenizer.IdentifierHandlingDefault) {
		if flags&cf(common.CommonFlagsAmbient) != 0 {
			p.error(
				diagnostics.DiagnosticCodeInitializersAreNotAllowedInAmbientContexts,
				tn.MakeRange(-1, -1),
			) // recoverable
		}
		initializer = p.parseExpression(tn, PrecedenceComma+1)
		if initializer == nil {
			return nil
		}
		if flags&cf(common.CommonFlagsDefinitelyAssigned) != 0 {
			p.error(
				diagnostics.DiagnosticCodeDeclarationsWithInitializersCannotAlsoHaveDefiniteAssignmentAssertions,
				initializer.GetRange(),
			)
		}
	} else if !isFor {
		if flags&cf(common.CommonFlagsConst) != 0 {
			if flags&cf(common.CommonFlagsAmbient) == 0 {
				p.error(
					diagnostics.DiagnosticCodeConstDeclarationsMustBeInitialized,
					&identifier.Range,
				) // recoverable
			}
		} else if typ == nil { // neither type nor initializer
			p.error(
				diagnostics.DiagnosticCodeTypeExpected,
				tn.MakeRange(tn.Pos, -1),
			) // recoverable
		}
	}
	rng := diagnostics.JoinRanges(&identifier.Range, tn.MakeRange(-1, -1))
	if flags&cf(common.CommonFlagsDefinitelyAssigned) != 0 && flags&cf(common.CommonFlagsAmbient) != 0 {
		p.error(
			diagnostics.DiagnosticCodeADefiniteAssignmentAssertionIsNotPermittedInThisContext,
			rng,
		)
	}
	return ast.NewVariableDeclaration(
		identifier,
		parentDecorators,
		flags,
		typ,
		initializer,
		*rng,
	)
}

// parseEnum parses an enum declaration: 'enum' Identifier '{' (EnumValue (',' EnumValue)*)? '}' ';'?
func (p *Parser) parseEnum(
	tn *tokenizer.Tokenizer,
	flags int32,
	decorators []*ast.DecoratorNode,
	startPos int32,
) *ast.EnumDeclaration {
	if tn.Next(tokenizer.IdentifierHandlingDefault) != tokenizer.TokenIdentifier {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
		return nil
	}
	identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	if tn.Next(tokenizer.IdentifierHandlingDefault) != tokenizer.TokenOpenBrace {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
		return nil
	}
	var members []*ast.EnumValueDeclaration
	for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
		member := p.parseEnumValue(tn, cf(common.CommonFlagsNone))
		if member == nil {
			return nil
		}
		members = append(members, member)
		if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
				break
			} else {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
				return nil
			}
		}
	}
	ret := ast.NewEnumDeclaration(identifier, decorators, flags, members, *tn.MakeRange(startPos, tn.Pos))
	ret.OverriddenModuleName = p.currentModuleName
	ret.HasOverriddenModule = p.currentModuleName != ""
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseEnumValue parses an enum value: Identifier ('=' Expression)?
func (p *Parser) parseEnumValue(tn *tokenizer.Tokenizer, parentFlags int32) *ast.EnumValueDeclaration {
	if !tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
		return nil
	}
	identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	var value ast.Node
	if tn.Skip(tokenizer.TokenEquals, tokenizer.IdentifierHandlingDefault) {
		value = p.parseExpression(tn, PrecedenceComma+1)
		if value == nil {
			return nil
		}
	}
	return ast.NewEnumValueDeclaration(
		identifier,
		parentFlags,
		value,
		*diagnostics.JoinRanges(&identifier.Range, tn.MakeRange(-1, -1)),
	)
}

// parseParameters parses function parameters: '(' (Parameter (',' Parameter)*)? ')'
func (p *Parser) parseParameters(tn *tokenizer.Tokenizer, isConstructor bool) []*ast.ParameterNode {
	parameters := []*ast.ParameterNode{}
	var seenRest *ast.ParameterNode
	seenOptional := false
	reportedRest := false

	// check if there is a leading `this` parameter
	p.parseParametersThis = nil
	if tn.Skip(tokenizer.TokenThis, tokenizer.IdentifierHandlingDefault) {
		if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
			thisType := p.parseType(tn, true, false)
			if thisType == nil {
				return nil
			}
			if thisType.GetKind() == ast.NodeKindNamedType {
				p.parseParametersThis = thisType.(*ast.NamedTypeNode)
			} else {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, thisType.GetRange())
			}
		} else {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ":")
			return nil
		}
		if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
				return parameters
			}
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
			return nil
		}
	}

	for !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
		param := p.parseParameter(tn, isConstructor)
		if param == nil {
			return nil
		}
		if seenRest != nil && !reportedRest {
			p.error(
				diagnostics.DiagnosticCodeARestParameterMustBeLastInAParameterList,
				seenRest.Name.GetRange(),
			)
			reportedRest = true
		}
		switch param.ParameterKind {
		case ast.ParameterKindOptional:
			seenOptional = true
		case ast.ParameterKindRest:
			seenRest = param
		default:
			if seenOptional {
				p.error(
					diagnostics.DiagnosticCodeARequiredParameterCannotFollowAnOptionalParameter,
					param.Name.GetRange(),
				)
			}
		}
		parameters = append(parameters, param)
		if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
				break
			}
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
			return nil
		}
	}
	return parameters
}

// parseParameter parses a single parameter: ('public' | 'private' | 'protected' | '...')? Identifier '?'? (':' Type)? ('=' Expression)?
func (p *Parser) parseParameter(tn *tokenizer.Tokenizer, isConstructor bool) *ast.ParameterNode {
	isRest := false
	isOptional := false
	var startRange *diagnostics.Range
	var accessFlags common.CommonFlags

	if isConstructor {
		if tn.Skip(tokenizer.TokenPublic, tokenizer.IdentifierHandlingDefault) {
			r := tn.MakeRange(-1, -1)
			startRange = r
			accessFlags |= common.CommonFlagsPublic
		} else if tn.Skip(tokenizer.TokenProtected, tokenizer.IdentifierHandlingDefault) {
			r := tn.MakeRange(-1, -1)
			startRange = r
			accessFlags |= common.CommonFlagsProtected
		} else if tn.Skip(tokenizer.TokenPrivate, tokenizer.IdentifierHandlingDefault) {
			r := tn.MakeRange(-1, -1)
			startRange = r
			accessFlags |= common.CommonFlagsPrivate
		}
		if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenReadonly {
			state := tn.Mark()
			tn.Next(tokenizer.IdentifierHandlingDefault)
			if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenColon { // modifier
				tn.Discard(state)
				if startRange == nil {
					r := tn.MakeRange(-1, -1)
					startRange = r
				}
				accessFlags |= common.CommonFlagsReadonly
			} else { // identifier
				tn.Reset(state)
			}
		}
	}

	if tn.Skip(tokenizer.TokenDotDotDot, tokenizer.IdentifierHandlingDefault) {
		if accessFlags != 0 {
			p.error(
				diagnostics.DiagnosticCodeAParameterPropertyCannotBeDeclaredUsingARestParameter,
				tn.MakeRange(-1, -1),
			)
		} else {
			r := tn.MakeRange(-1, -1)
			startRange = r
		}
		isRest = true
	}

	if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		if !isRest {
			r := tn.MakeRange(-1, -1)
			startRange = r
		}
		identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)

		if tn.Skip(tokenizer.TokenQuestion, tokenizer.IdentifierHandlingDefault) {
			isOptional = true
			if isRest {
				p.error(diagnostics.DiagnosticCodeARestParameterCannotBeOptional, &identifier.Range)
			}
		}

		var typ ast.Node
		if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
			typ = p.parseType(tn, true, false)
			if typ == nil {
				return nil
			}
		} else {
			typ = ast.NewOmittedType(*tn.MakeRange(tn.Pos, -1))
		}

		var initializer ast.Node
		if tn.Skip(tokenizer.TokenEquals, tokenizer.IdentifierHandlingDefault) {
			if isRest {
				p.error(diagnostics.DiagnosticCodeARestParameterCannotHaveAnInitializer, &identifier.Range)
			}
			if isOptional {
				p.error(diagnostics.DiagnosticCodeParameterCannotHaveQuestionMarkAndInitializer, &identifier.Range)
			} else {
				isOptional = true
			}
			initializer = p.parseExpression(tn, PrecedenceComma+1)
			if initializer == nil {
				return nil
			}
		}

		kind := ast.ParameterKindDefault
		if isRest {
			kind = ast.ParameterKindRest
		} else if isOptional {
			kind = ast.ParameterKindOptional
		}
		param := ast.NewParameterNode(
			kind,
			identifier,
			typ,
			initializer,
			*diagnostics.JoinRanges(startRange, tn.MakeRange(-1, -1)),
		)
		param.Flags |= int32(accessFlags)
		return param
	}
	p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	return nil
}

// parseFunction parses a function declaration.
func (p *Parser) parseFunction(
	tn *tokenizer.Tokenizer,
	flags int32,
	decorators []*ast.DecoratorNode,
	startPos int32,
) *ast.FunctionDeclaration {
	if !tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(tn.Pos, -1))
		return nil
	}
	name := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	signatureStart := int32(-1)

	var typeParameters []*ast.TypeParameterNode
	if tn.Skip(tokenizer.TokenLessThan, tokenizer.IdentifierHandlingDefault) {
		signatureStart = tn.TokenPos
		typeParameters = p.parseTypeParameters(tn)
		if typeParameters == nil {
			return nil
		}
		flags |= cf(common.CommonFlagsGeneric)
	}

	if !tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(tn.Pos, -1), "(")
		return nil
	}

	if signatureStart < 0 {
		signatureStart = tn.TokenPos
	}

	parameters := p.parseParameters(tn, false)
	if parameters == nil {
		return nil
	}
	thisType := p.parseParametersThis

	isSetter := flags&cf(common.CommonFlagsSet) != 0
	if isSetter {
		if len(parameters) != 1 {
			p.error(diagnostics.DiagnosticCodeASetAccessorMustHaveExactlyOneParameter, name.GetRange())
		}
		if len(parameters) > 0 && parameters[0].Initializer != nil {
			p.error(diagnostics.DiagnosticCodeASetAccessorParameterCannotHaveAnInitializer, name.GetRange())
		}
	}

	if flags&cf(common.CommonFlagsGet) != 0 {
		if len(parameters) > 0 {
			p.error(diagnostics.DiagnosticCodeAGetAccessorCannotHaveParameters, name.GetRange())
		}
	}

	var returnType ast.Node
	if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
		returnType = p.parseType(tn, true, isSetter)
		if returnType == nil {
			return nil
		}
	}

	if returnType == nil {
		returnType = ast.NewOmittedType(*tn.MakeRange(tn.Pos, -1))
		if !isSetter {
			p.error(diagnostics.DiagnosticCodeTypeExpected, returnType.GetRange())
		}
	}

	signature := ast.NewFunctionTypeNode(
		parameters, returnType, thisType, false,
		*tn.MakeRange(signatureStart, tn.Pos),
	)

	var body ast.Node
	if tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
		if flags&cf(common.CommonFlagsAmbient) != 0 {
			p.error(
				diagnostics.DiagnosticCodeAnImplementationCannotBeDeclaredInAmbientContexts,
				tn.MakeRange(-1, -1),
			) // recoverable
		}
		body = p.parseBlockStatement(tn, false)
		if body == nil {
			return nil
		}
	} else if flags&cf(common.CommonFlagsAmbient) == 0 {
		p.error(
			diagnostics.DiagnosticCodeFunctionImplementationIsMissingOrNotImmediatelyFollowingTheDeclaration,
			tn.MakeRange(tn.Pos, -1),
		)
	}

	ret := ast.NewFunctionDeclaration(
		name, decorators, flags, typeParameters, signature, body,
		ast.ArrowKindNone, *tn.MakeRange(startPos, tn.Pos),
	)
	ret.OverriddenModuleName = p.currentModuleName
	ret.HasOverriddenModule = p.currentModuleName != ""
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseFunctionExpression parses a function expression or arrow function.
func (p *Parser) parseFunctionExpression(tn *tokenizer.Tokenizer) *ast.FunctionExpression {
	startPos := tn.TokenPos
	var name *ast.IdentifierExpression
	arrowKind := ast.ArrowKindNone

	if tn.Token == tokenizer.TokenFunction {
		if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
			name = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
		} else {
			name = ast.NewEmptyIdentifierExpression(*tn.MakeRange(tn.Pos, -1))
		}
		if !tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(tn.Pos, -1), "(")
			return nil
		}
	} else {
		arrowKind = ast.ArrowKindParenthesized
		name = ast.NewEmptyIdentifierExpression(*tn.MakeRange(tn.TokenPos, -1))
	}

	signatureStart := tn.Pos
	parameters := p.parseParameters(tn, false)
	if parameters == nil {
		return nil
	}

	return p.parseFunctionExpressionCommon(tn, name, parameters, p.parseParametersThis, arrowKind, startPos, signatureStart)
}

// parseFunctionExpressionCommon completes parsing a function expression or arrow function.
func (p *Parser) parseFunctionExpressionCommon(
	tn *tokenizer.Tokenizer,
	name *ast.IdentifierExpression,
	parameters []*ast.ParameterNode,
	explicitThis *ast.NamedTypeNode,
	arrowKind ast.ArrowKind,
	startPos int32,
	signatureStart int32,
) *ast.FunctionExpression {
	if startPos < 0 {
		startPos = name.Range.Start
	}
	if signatureStart < 0 {
		signatureStart = startPos
	}

	var returnType ast.Node
	if arrowKind != ast.ArrowKindSingle && tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
		returnType = p.parseType(tn, true, false)
		if returnType == nil {
			return nil
		}
	} else {
		returnType = ast.NewOmittedType(*tn.MakeRange(tn.Pos, -1))
	}

	if arrowKind != ast.ArrowKindNone {
		if !tn.Skip(tokenizer.TokenEqualsGreaterThan, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(tn.Pos, -1), "=>")
			return nil
		}
	}

	signature := ast.NewFunctionTypeNode(
		parameters, returnType, explicitThis, false,
		*tn.MakeRange(signatureStart, tn.Pos),
	)

	var body ast.Node
	if arrowKind != ast.ArrowKindNone {
		if tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
			body = p.parseBlockStatement(tn, false)
		} else {
			bodyExpression := p.parseExpression(tn, PrecedenceComma+1)
			if bodyExpression != nil {
				body = ast.NewExpressionStatement(bodyExpression)
			}
		}
	} else {
		if !tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(tn.Pos, -1), "{")
			return nil
		}
		body = p.parseBlockStatement(tn, false)
	}
	if body == nil {
		return nil
	}

	declaration := ast.NewFunctionDeclaration(
		name, nil, cf(common.CommonFlagsNone), nil, signature, body,
		arrowKind, *tn.MakeRange(startPos, tn.Pos),
	)
	return ast.NewFunctionExpression(declaration)
}

// parseClassOrInterface parses a class or interface declaration.
func (p *Parser) parseClassOrInterface(
	tn *tokenizer.Tokenizer,
	flags int32,
	decorators []*ast.DecoratorNode,
	startPos int32,
) *ast.ClassDeclaration {
	isInterface := tn.Token == tokenizer.TokenInterface

	if !tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
		return nil
	}
	identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)

	var typeParameters []*ast.TypeParameterNode
	if tn.Skip(tokenizer.TokenLessThan, tokenizer.IdentifierHandlingDefault) {
		typeParameters = p.parseTypeParameters(tn)
		if typeParameters == nil {
			return nil
		}
		flags |= cf(common.CommonFlagsGeneric)
	}

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

	var implementsTypes []*ast.NamedTypeNode
	if tn.Skip(tokenizer.TokenImplements, tokenizer.IdentifierHandlingDefault) {
		if isInterface {
			p.error(
				diagnostics.DiagnosticCodeInterfaceDeclarationCannotHaveImplementsClause,
				tn.MakeRange(-1, -1),
			) // recoverable
		}
		for {
			typ := p.parseType(tn, true, false)
			if typ == nil {
				return nil
			}
			if typ.GetKind() != ast.NodeKindNamedType {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, typ.GetRange())
				return nil
			}
			if !isInterface {
				implementsTypes = append(implementsTypes, typ.(*ast.NamedTypeNode))
			}
			if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
				break
			}
		}
	}

	if !tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
		return nil
	}

	var members []ast.Node
	var declaration *ast.ClassDeclaration
	if isInterface {
		declaration = ast.NewInterfaceDeclaration(
			identifier, decorators, flags, typeParameters,
			extendsType, nil, members,
			*tn.MakeRange(startPos, tn.Pos),
		)
	} else {
		declaration = ast.NewClassDeclaration(
			identifier, decorators, flags, typeParameters,
			extendsType, implementsTypes, members,
			*tn.MakeRange(startPos, tn.Pos),
		)
	}

	if !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
		for {
			member := p.parseClassMember(tn, declaration)
			if member != nil {
				if member.GetKind() == ast.NodeKindIndexSignature {
					declaration.IndexSignature = member.(*ast.IndexSignatureNode)
				} else {
					declaration.Members = append(declaration.Members, member)
				}
			} else {
				p.skipStatement(tn)
				if tn.Skip(tokenizer.TokenEndOfFile, tokenizer.IdentifierHandlingDefault) {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
					return nil
				}
			}
			if tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
				break
			}
		}
	}
	declaration.Range.End = tn.Pos
	declaration.OverriddenModuleName = p.currentModuleName
	declaration.HasOverriddenModule = p.currentModuleName != ""
	return declaration
}

// parseClassExpression parses a class expression: 'class' Identifier? '{' ... '}'
func (p *Parser) parseClassExpression(tn *tokenizer.Tokenizer) *ast.ClassExpression {
	startPos := tn.TokenPos
	var name *ast.IdentifierExpression

	if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		name = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	} else {
		name = ast.NewEmptyIdentifierExpression(*tn.MakeRange(tn.Pos, -1))
	}

	if !tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(tn.Pos, -1), "{")
		return nil
	}

	var members []ast.Node
	declaration := ast.NewClassDeclaration(
		name, nil, cf(common.CommonFlagsNone), nil, nil, nil,
		members, *tn.MakeRange(startPos, tn.Pos),
	)

	if !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
		for {
			member := p.parseClassMember(tn, declaration)
			if member != nil {
				if member.GetKind() == ast.NodeKindIndexSignature {
					declaration.IndexSignature = member.(*ast.IndexSignatureNode)
				} else {
					declaration.Members = append(declaration.Members, member)
				}
			} else {
				p.skipStatement(tn)
				if tn.Skip(tokenizer.TokenEndOfFile, tokenizer.IdentifierHandlingDefault) {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
					return nil
				}
			}
			if tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
				break
			}
		}
	}
	declaration.Range.End = tn.Pos
	return ast.NewClassExpression(declaration)
}

// parseClassMember parses a class or interface member.
func (p *Parser) parseClassMember(tn *tokenizer.Tokenizer, parent *ast.ClassDeclaration) ast.Node {
	isInterface := parent.GetKind() == ast.NodeKindInterfaceDeclaration
	startPos := int32(0)

	var decorators []*ast.DecoratorNode
	if tn.Skip(tokenizer.TokenAt, tokenizer.IdentifierHandlingDefault) {
		startPos = tn.TokenPos
		for {
			decorator := p.parseDecorator(tn)
			if decorator == nil {
				break
			}
			decorators = append(decorators, decorator)
			if !tn.Skip(tokenizer.TokenAt, tokenizer.IdentifierHandlingDefault) {
				break
			}
		}
		if isInterface && decorators != nil {
			p.error(
				diagnostics.DiagnosticCodeDecoratorsAreNotValidHere,
				diagnostics.JoinRanges(decorators[0].GetRange(), decorators[len(decorators)-1].GetRange()),
			)
		}
	}

	// inherit ambient status
	flags := parent.Flags & cf(common.CommonFlagsAmbient)

	// interface methods are always overridden if used
	if isInterface {
		flags |= cf(common.CommonFlagsOverridden)
	}

	var declareStart, declareEnd int32
	contextIsAmbient := parent.Is(cf(common.CommonFlagsAmbient))
	if tn.Skip(tokenizer.TokenDeclare, tokenizer.IdentifierHandlingDefault) {
		if isInterface {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(-1, -1), "declare")
		} else {
			if contextIsAmbient {
				p.error(diagnostics.DiagnosticCodeADeclareModifierCannotBeUsedInAnAlreadyAmbientContext, tn.MakeRange(-1, -1))
			} else {
				flags |= cf(common.CommonFlagsDeclare | common.CommonFlagsAmbient)
				declareStart = tn.TokenPos
				declareEnd = tn.Pos
			}
		}
		if startPos == 0 {
			startPos = tn.TokenPos
		}
	} else if contextIsAmbient {
		flags |= cf(common.CommonFlagsAmbient)
	}

	var accessStart, accessEnd int32
	if tn.Skip(tokenizer.TokenPublic, tokenizer.IdentifierHandlingDefault) {
		if isInterface {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(-1, -1), "public")
		} else {
			flags |= cf(common.CommonFlagsPublic)
			accessStart = tn.TokenPos
			accessEnd = tn.Pos
		}
		if startPos == 0 {
			startPos = tn.TokenPos
		}
	} else if tn.Skip(tokenizer.TokenPrivate, tokenizer.IdentifierHandlingDefault) {
		if isInterface {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(-1, -1), "private")
		} else {
			flags |= cf(common.CommonFlagsPrivate)
			accessStart = tn.TokenPos
			accessEnd = tn.Pos
		}
		if startPos == 0 {
			startPos = tn.TokenPos
		}
	} else if tn.Skip(tokenizer.TokenProtected, tokenizer.IdentifierHandlingDefault) {
		if isInterface {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(-1, -1), "protected")
		} else {
			flags |= cf(common.CommonFlagsProtected)
			accessStart = tn.TokenPos
			accessEnd = tn.Pos
		}
		if startPos == 0 {
			startPos = tn.TokenPos
		}
	}

	var staticStart, staticEnd int32
	var abstractStart, abstractEnd int32
	if tn.Skip(tokenizer.TokenStatic, tokenizer.IdentifierHandlingDefault) {
		if isInterface {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(-1, -1), "static")
		} else {
			flags |= cf(common.CommonFlagsStatic)
			staticStart = tn.TokenPos
			staticEnd = tn.Pos
		}
		if startPos == 0 {
			startPos = tn.TokenPos
		}
	} else {
		flags |= cf(common.CommonFlagsInstance)
		if tn.Skip(tokenizer.TokenAbstract, tokenizer.IdentifierHandlingDefault) {
			if isInterface || !parent.Is(cf(common.CommonFlagsAbstract)) {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(-1, -1), "abstract")
			} else {
				flags |= cf(common.CommonFlagsAbstract)
				abstractStart = tn.TokenPos
				abstractEnd = tn.Pos
			}
			if startPos == 0 {
				startPos = tn.TokenPos
			}
		}
		if parent.Flags&cf(common.CommonFlagsGeneric) != 0 {
			flags |= cf(common.CommonFlagsGenericContext)
		}
	}

	var overrideStart, overrideEnd int32
	if tn.Skip(tokenizer.TokenOverride, tokenizer.IdentifierHandlingDefault) {
		if isInterface || parent.ExtendsType == nil {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(-1, -1), "override")
		} else {
			flags |= cf(common.CommonFlagsOverride)
			overrideStart = tn.TokenPos
			overrideEnd = tn.Pos
		}
		if startPos == 0 {
			startPos = tn.TokenPos
		}
	}

	var readonlyStart, readonlyEnd int32
	if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenReadonly {
		state := tn.Mark()
		tn.Next(tokenizer.IdentifierHandlingDefault)
		if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenColon { // modifier
			tn.Discard(state)
			flags |= cf(common.CommonFlagsReadonly)
			readonlyStart = tn.TokenPos
			readonlyEnd = tn.Pos
			if startPos == 0 {
				startPos = readonlyStart
			}
		} else { // identifier
			tn.Reset(state)
		}
	}

	// check if accessor: ('get' | 'set') ^\n Identifier
	state := tn.Mark()
	isConstructor := false
	isGetter := false
	var getStart, getEnd int32
	isSetter := false
	var setStart, setEnd int32

	if !isInterface {
		if tn.Skip(tokenizer.TokenGet, tokenizer.IdentifierHandlingDefault) {
			if tn.Peek(tokenizer.IdentifierHandlingPrefer, MaxInt32) == tokenizer.TokenIdentifier && !tn.PeekOnNewLine() {
				flags |= cf(common.CommonFlagsGet)
				isGetter = true
				getStart = tn.TokenPos
				getEnd = tn.Pos
				if startPos == 0 {
					startPos = getStart
				}
				if flags&cf(common.CommonFlagsReadonly) != 0 {
					p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
						tn.MakeRange(readonlyStart, readonlyEnd), "readonly")
				}
			} else {
				tn.Reset(state)
			}
		} else if tn.Skip(tokenizer.TokenSet, tokenizer.IdentifierHandlingDefault) {
			if tn.Peek(tokenizer.IdentifierHandlingPrefer, MaxInt32) == tokenizer.TokenIdentifier && !tn.PeekOnNewLine() {
				flags |= cf(common.CommonFlagsSet)
				isSetter = true
				setStart = tn.TokenPos
				setEnd = tn.Pos
				if startPos == 0 {
					startPos = setStart
				}
				if flags&cf(common.CommonFlagsReadonly) != 0 {
					p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
						tn.MakeRange(readonlyStart, readonlyEnd), "readonly")
				}
			} else {
				tn.Reset(state)
			}
		} else if tn.Skip(tokenizer.TokenConstructor, tokenizer.IdentifierHandlingDefault) {
			flags |= cf(common.CommonFlagsConstructor)
			isConstructor = true
			if startPos == 0 {
				startPos = tn.TokenPos
			}
			if flags&cf(common.CommonFlagsStatic) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
					tn.MakeRange(staticStart, staticEnd), "static")
			}
			if flags&cf(common.CommonFlagsAbstract) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
					tn.MakeRange(abstractStart, abstractEnd), "abstract")
			}
			if flags&cf(common.CommonFlagsReadonly) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
					tn.MakeRange(readonlyStart, readonlyEnd), "readonly")
			}
		}
	}

	isGetterOrSetter := isGetter || isSetter

	var name *ast.IdentifierExpression
	if isConstructor {
		name = ast.NewConstructorExpression(*tn.MakeRange(-1, -1))
	} else {
		if !isGetterOrSetter && tn.Skip(tokenizer.TokenOpenBracket, tokenizer.IdentifierHandlingDefault) {
			if startPos == 0 {
				startPos = tn.TokenPos
			}
			// index signature — check for invalid modifiers
			if flags&cf(common.CommonFlagsPublic) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(accessStart, accessEnd), "public")
			} else if flags&cf(common.CommonFlagsProtected) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(accessStart, accessEnd), "protected")
			} else if flags&cf(common.CommonFlagsPrivate) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(accessStart, accessEnd), "private")
			}
			if flags&cf(common.CommonFlagsStatic) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(staticStart, staticEnd), "static")
			}
			if flags&cf(common.CommonFlagsOverride) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(overrideStart, overrideEnd), "override")
			}
			if flags&cf(common.CommonFlagsAbstract) != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(abstractStart, abstractEnd), "abstract")
			}
			retIndex := p.parseIndexSignature(tn, flags, decorators)
			if retIndex == nil {
				if flags&cf(common.CommonFlagsReadonly) != 0 {
					p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(readonlyStart, readonlyEnd), "readonly")
				}
				return nil
			}
			tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
			return retIndex
		}
		if !tn.SkipIdentifier(tokenizer.IdentifierHandlingAlways) {
			p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
			return nil
		}
		if startPos == 0 {
			startPos = tn.TokenPos
		}
		name = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	}

	var typeParameters []*ast.TypeParameterNode
	if tn.Skip(tokenizer.TokenLessThan, tokenizer.IdentifierHandlingDefault) {
		typeParametersStart := tn.TokenPos
		typeParameters = p.parseTypeParameters(tn)
		if typeParameters == nil {
			return nil
		}
		if isConstructor {
			p.error(diagnostics.DiagnosticCodeTypeParametersCannotAppearOnAConstructorDeclaration,
				tn.MakeRange(typeParametersStart, tn.Pos))
		} else if isGetterOrSetter {
			p.error(diagnostics.DiagnosticCodeAnAccessorCannotHaveTypeParameters,
				tn.MakeRange(typeParametersStart, tn.Pos))
		} else {
			flags |= cf(common.CommonFlagsGeneric)
		}
	}

	// method: '(' Parameters (':' Type)? '{' Statement* '}' ';'?
	if tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
		if flags&cf(common.CommonFlagsDeclare) != 0 {
			p.error(diagnostics.DiagnosticCode0ModifierCannotAppearOnClassElementsOfThisKind,
				tn.MakeRange(declareStart, declareEnd), "declare")
		}

		signatureStart := tn.TokenPos
		parameters := p.parseParameters(tn, isConstructor)
		if parameters == nil {
			return nil
		}
		thisType := p.parseParametersThis

		if isConstructor {
			for i, parameter := range parameters {
				if parameter.IsAny(cf(common.CommonFlagsPublic | common.CommonFlagsProtected | common.CommonFlagsPrivate | common.CommonFlagsReadonly)) {
					implicitFieldDeclaration := ast.NewFieldDeclaration(
						parameter.Name,
						nil,
						parameter.Flags|cf(common.CommonFlagsInstance),
						parameter.Type,
						nil, // initialized via parameter
						parameter.Range,
					)
					implicitFieldDeclaration.ParameterIndex = int32(i)
					parameter.ImplicitFieldDeclaration = implicitFieldDeclaration
					parent.Members = append(parent.Members, implicitFieldDeclaration)
				}
			}
		} else if isGetter {
			if len(parameters) > 0 {
				p.error(diagnostics.DiagnosticCodeAGetAccessorCannotHaveParameters, name.GetRange())
			}
		} else if isSetter {
			if len(parameters) != 1 {
				p.error(diagnostics.DiagnosticCodeASetAccessorMustHaveExactlyOneParameter, name.GetRange())
			}
			if len(parameters) > 0 && parameters[0].Initializer != nil {
				p.error(diagnostics.DiagnosticCodeASetAccessorParameterCannotHaveAnInitializer, name.GetRange())
			}
		} else if name.Text == "constructor" {
			p.error(diagnostics.DiagnosticCode0KeywordCannotBeUsedHere, name.GetRange(), "constructor")
		}

		var returnType ast.Node
		if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
			if name.GetKind() == ast.NodeKindConstructor {
				p.error(diagnostics.DiagnosticCodeTypeAnnotationCannotAppearOnAConstructorDeclaration, tn.MakeRange(-1, -1))
			} else if isSetter {
				p.error(diagnostics.DiagnosticCodeASetAccessorCannotHaveAReturnTypeAnnotation, tn.MakeRange(-1, -1))
			}
			returnType = p.parseType(tn, isSetter || name.GetKind() == ast.NodeKindConstructor, false)
			if returnType == nil {
				return nil
			}
		} else {
			returnType = ast.NewOmittedType(*tn.MakeRange(tn.Pos, -1))
			if !isSetter && name.GetKind() != ast.NodeKindConstructor {
				p.error(diagnostics.DiagnosticCodeTypeExpected, returnType.GetRange())
			}
		}

		signature := ast.NewFunctionTypeNode(
			parameters, returnType, thisType, false,
			*tn.MakeRange(signatureStart, tn.Pos),
		)

		var body ast.Node
		if tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
			if flags&cf(common.CommonFlagsAmbient) != 0 {
				p.error(diagnostics.DiagnosticCodeAnImplementationCannotBeDeclaredInAmbientContexts, tn.MakeRange(-1, -1))
			} else if flags&cf(common.CommonFlagsAbstract) != 0 {
				p.error(diagnostics.DiagnosticCodeMethod0CannotHaveAnImplementationBecauseItIsMarkedAbstract,
					tn.MakeRange(-1, -1), name.Text)
			} else if isInterface {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ";")
			}
			body = p.parseBlockStatement(tn, false)
			if body == nil {
				return nil
			}
		} else if !isInterface && flags&cf(common.CommonFlagsAmbient|common.CommonFlagsAbstract) == 0 {
			p.error(diagnostics.DiagnosticCodeFunctionImplementationIsMissingOrNotImmediatelyFollowingTheDeclaration,
				tn.MakeRange(-1, -1))
		}

		retMethod := ast.NewMethodDeclaration(
			name, decorators, flags, typeParameters, signature, body,
			*tn.MakeRange(startPos, tn.Pos),
		)
		if !(isInterface && tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault)) {
			tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
		}
		return retMethod

	} else if isConstructor {
		p.error(diagnostics.DiagnosticCodeConstructorImplementationIsMissing, name.GetRange())

	} else if isGetterOrSetter {
		p.error(diagnostics.DiagnosticCodeFunctionImplementationIsMissingOrNotImmediatelyFollowingTheDeclaration, name.GetRange())

	// field: (':' Type)? ('=' Expression)? ';'?
	} else {
		if flags&cf(common.CommonFlagsDeclare) != 0 {
			p.error(diagnostics.DiagnosticCodeNotImplemented0, tn.MakeRange(declareStart, declareEnd), "Ambient fields")
		}
		if flags&cf(common.CommonFlagsAbstract) != 0 {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(abstractStart, abstractEnd), "abstract")
		}
		if flags&cf(common.CommonFlagsGet) != 0 {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(getStart, getEnd), "get")
		}
		if flags&cf(common.CommonFlagsSet) != 0 {
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere, tn.MakeRange(setStart, setEnd), "set")
		}

		var typ ast.Node
		if tn.Skip(tokenizer.TokenQuestion, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCodeOptionalPropertiesAreNotSupported, tn.MakeRange(startPos, tn.Pos))
		}
		if tn.Skip(tokenizer.TokenExclamation, tokenizer.IdentifierHandlingDefault) {
			flags |= cf(common.CommonFlagsDefinitelyAssigned)
		}
		if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
			typ = p.parseType(tn, true, false)
			if typ == nil {
				return nil
			}
		} else {
			p.error(diagnostics.DiagnosticCodeTypeExpected, tn.MakeRange(-1, -1))
		}

		var initializer ast.Node
		if tn.Skip(tokenizer.TokenEquals, tokenizer.IdentifierHandlingDefault) {
			if flags&cf(common.CommonFlagsAmbient) != 0 {
				p.error(diagnostics.DiagnosticCodeInitializersAreNotAllowedInAmbientContexts, tn.MakeRange(-1, -1))
			}
			initializer = p.parseExpression(tn, PrecedenceComma)
			if initializer == nil {
				return nil
			}
			if flags&cf(common.CommonFlagsDefinitelyAssigned) != 0 {
				p.error(diagnostics.DiagnosticCodeDeclarationsWithInitializersCannotAlsoHaveDefiniteAssignmentAssertions, name.GetRange())
			}
		}
		rng := *tn.MakeRange(startPos, tn.Pos)
		if flags&cf(common.CommonFlagsDefinitelyAssigned) != 0 && (isInterface || flags&cf(common.CommonFlagsAmbient) != 0) {
			p.error(diagnostics.DiagnosticCodeADefiniteAssignmentAssertionIsNotPermittedInThisContext, &rng)
		}
		retField := ast.NewFieldDeclaration(
			name, decorators, flags, typ, initializer, rng,
		)
		if !(isInterface && tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault)) {
			tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
		}
		return retField
	}
	return nil
}

// parseIndexSignature parses an index signature: '[' 'key' ':' Type ']' ':' Type
func (p *Parser) parseIndexSignature(
	tn *tokenizer.Tokenizer,
	flags int32,
	decorators []*ast.DecoratorNode,
) *ast.IndexSignatureNode {
	if len(decorators) > 0 {
		p.error(
			diagnostics.DiagnosticCodeDecoratorsAreNotValidHere,
			diagnostics.JoinRanges(decorators[0].GetRange(), decorators[len(decorators)-1].GetRange()),
		)
	}

	start := tn.TokenPos
	if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		id := tn.ReadIdentifier()
		if id == "key" {
			if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
				keyType := p.parseType(tn, true, false)
				if keyType == nil {
					return nil
				}
				if keyType.GetKind() != ast.NodeKindNamedType {
					p.error(diagnostics.DiagnosticCodeTypeExpected, tn.MakeRange(-1, -1))
					return nil
				}
				if tn.Skip(tokenizer.TokenCloseBracket, tokenizer.IdentifierHandlingDefault) {
					if tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
						valueType := p.parseType(tn, true, false)
						if valueType == nil {
							return nil
						}
						if valueType.GetKind() != ast.NodeKindNamedType {
							p.error(diagnostics.DiagnosticCodeIdentifierExpected, valueType.GetRange())
							return nil
						}
						return ast.NewIndexSignatureNode(keyType.(*ast.NamedTypeNode), valueType, flags, *tn.MakeRange(start, tn.Pos))
					}
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ":")
				} else {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "]")
				}
			} else {
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ":")
			}
		} else {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "key")
		}
	} else {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	}
	return nil
}

// parseNamespace parses a namespace declaration: 'namespace' Identifier '{' TopLevelStatement* '}'
func (p *Parser) parseNamespace(
	tn *tokenizer.Tokenizer,
	flags int32,
	decorators []*ast.DecoratorNode,
	startPos int32,
) *ast.NamespaceDeclaration {
	if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
		if tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
			var members []ast.Node
			declaration := ast.NewNamespaceDeclaration(
				identifier, decorators, flags, members,
				*tn.MakeRange(startPos, tn.Pos),
			)
			for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
				member := p.parseTopLevelStatement(tn, declaration)
				if member != nil {
					if member.GetKind() == ast.NodeKindExport {
						p.error(diagnostics.DiagnosticCodeADefaultExportCanOnlyBeUsedInAModule, member.GetRange())
						return nil
					}
					declaration.Members = append(declaration.Members, member)
				} else {
					p.skipStatement(tn)
					if tn.Skip(tokenizer.TokenEndOfFile, tokenizer.IdentifierHandlingDefault) {
						p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
						return nil
					}
				}
			}
			declaration.Range.End = tn.Pos
			declaration.OverriddenModuleName = p.currentModuleName
			declaration.HasOverriddenModule = p.currentModuleName != ""
			tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
			return declaration
		}
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
	} else {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	}
	return nil
}

// parseExport parses an export statement: 'export' '{' ExportMember (',' ExportMember)* '}' ('from' StringLiteral)? ';'?
func (p *Parser) parseExport(tn *tokenizer.Tokenizer, startPos int32, isDeclare bool) *ast.ExportStatement {
	currentSource := p.currentSource
	if tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
		var members []*ast.ExportMember
		for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
			member := p.parseExportMember(tn)
			if member == nil {
				return nil
			}
			members = append(members, member)
			if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
				if tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
					break
				}
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
				return nil
			}
		}
		var path *ast.StringLiteralExpression
		if tn.Skip(tokenizer.TokenFrom, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenStringLiteral, tokenizer.IdentifierHandlingDefault) {
				path = ast.NewStringLiteralExpression(tn.ReadString(0, false), *tn.MakeRange(-1, -1))
			} else {
				p.error(diagnostics.DiagnosticCodeStringLiteralExpected, tn.MakeRange(-1, -1))
				return nil
			}
		}
		ret := ast.NewExportStatement(members, path, isDeclare, *tn.MakeRange(startPos, tn.Pos))
		if path != nil {
			internalPath := ret.InternalPath
			if !p.seenlog[internalPath] {
				p.dependees[internalPath] = &Dependee{Source: currentSource, Path: path}
				p.backlog = append(p.backlog, internalPath)
				p.seenlog[internalPath] = true
			}
		}
		tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
		return ret
	} else if tn.Skip(tokenizer.TokenAsterisk, tokenizer.IdentifierHandlingDefault) {
		if tn.Skip(tokenizer.TokenFrom, tokenizer.IdentifierHandlingDefault) {
			if tn.Skip(tokenizer.TokenStringLiteral, tokenizer.IdentifierHandlingDefault) {
				path := ast.NewStringLiteralExpression(tn.ReadString(0, false), *tn.MakeRange(-1, -1))
				ret := ast.NewExportStatement(nil, path, isDeclare, *tn.MakeRange(startPos, tn.Pos))
				internalPath := ret.InternalPath
				if p.currentSource.ExportPaths == nil {
					p.currentSource.ExportPaths = []string{internalPath}
				} else {
					found := false
					for _, ep := range p.currentSource.ExportPaths {
						if ep == internalPath {
							found = true
							break
						}
					}
					if !found {
						p.currentSource.ExportPaths = append(p.currentSource.ExportPaths, internalPath)
					}
				}
				if !p.seenlog[internalPath] {
					p.dependees[internalPath] = &Dependee{Source: currentSource, Path: path}
					p.backlog = append(p.backlog, internalPath)
				}
				tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
				return ret
			}
			p.error(diagnostics.DiagnosticCodeStringLiteralExpected, tn.MakeRange(-1, -1))
		} else {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "from")
		}
	} else {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
	}
	return nil
}

// parseExportMember parses an export member: Identifier ('as' Identifier)?
func (p *Parser) parseExportMember(tn *tokenizer.Tokenizer) *ast.ExportMember {
	if tn.SkipIdentifier(tokenizer.IdentifierHandlingAlways) {
		identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
		var asIdentifier *ast.IdentifierExpression
		if tn.Skip(tokenizer.TokenAs, tokenizer.IdentifierHandlingDefault) {
			if tn.SkipIdentifier(tokenizer.IdentifierHandlingAlways) {
				asIdentifier = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
			} else {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
				return nil
			}
		}
		if asIdentifier != nil {
			return ast.NewExportMember(
				identifier, asIdentifier,
				*diagnostics.JoinRanges(&identifier.Range, &asIdentifier.Range),
			)
		}
		return ast.NewExportMember(identifier, nil, identifier.Range)
	}
	p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	return nil
}

// parseExportDefaultAlias parses: 'export' 'default' Identifier ';'?
func (p *Parser) parseExportDefaultAlias(
	tn *tokenizer.Tokenizer,
	startPos int32,
	defaultStart int32,
	defaultEnd int32,
) *ast.ExportStatement {
	name := tn.ReadIdentifier()
	rng := tn.MakeRange(-1, -1)
	members := []*ast.ExportMember{
		ast.NewExportMember(
			ast.NewIdentifierExpression(name, *rng, false),
			ast.NewIdentifierExpression("default", *tn.MakeRange(defaultStart, defaultEnd), false),
			*rng,
		),
	}
	ret := ast.NewExportStatement(members, nil, false, *tn.MakeRange(startPos, tn.Pos))
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseImport parses an import statement.
func (p *Parser) parseImport(tn *tokenizer.Tokenizer) *ast.ImportStatement {
	startPos := tn.TokenPos
	var members []*ast.ImportDeclaration
	var namespaceName *ast.IdentifierExpression
	skipFrom := false

	if tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
		// import { ... } from "file"
		members = []*ast.ImportDeclaration{}
		for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
			member := p.parseImportDeclaration(tn)
			if member == nil {
				return nil
			}
			members = append(members, member)
			if !tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
				if tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
					break
				}
				p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
				return nil
			}
		}
	} else if tn.Skip(tokenizer.TokenAsterisk, tokenizer.IdentifierHandlingDefault) {
		// import * as Name from "file"
		if tn.Skip(tokenizer.TokenAs, tokenizer.IdentifierHandlingDefault) {
			if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
				namespaceName = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
			} else {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
				return nil
			}
		} else {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "as")
			return nil
		}
	} else if tn.Skip(tokenizer.TokenIdentifier, tokenizer.IdentifierHandlingPrefer) {
		// import Name from "file"
		name := tn.ReadIdentifier()
		rng := *tn.MakeRange(-1, -1)
		members = []*ast.ImportDeclaration{
			ast.NewImportDeclaration(
				ast.NewIdentifierExpression("default", rng, false),
				ast.NewIdentifierExpression(name, rng, false),
				rng,
			),
		}
		if tn.Skip(tokenizer.TokenComma, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCodeNotImplemented0, tn.MakeRange(-1, -1), "Mixed default and named imports")
			return nil
		}
	} else {
		// import "file"
		skipFrom = true
	}

	if skipFrom || tn.Skip(tokenizer.TokenFrom, tokenizer.IdentifierHandlingDefault) {
		if tn.Skip(tokenizer.TokenStringLiteral, tokenizer.IdentifierHandlingDefault) {
			path := ast.NewStringLiteralExpression(tn.ReadString(0, false), *tn.MakeRange(-1, -1))
			var ret *ast.ImportStatement
			if namespaceName != nil {
				ret = ast.NewWildcardImportStatement(namespaceName, path, *tn.MakeRange(startPos, tn.Pos))
			} else {
				ret = ast.NewImportStatement(members, path, *tn.MakeRange(startPos, tn.Pos))
			}
			internalPath := ret.InternalPath
			if !p.seenlog[internalPath] {
				p.dependees[internalPath] = &Dependee{Source: p.currentSource, Path: path}
				p.backlog = append(p.backlog, internalPath)
			}
			tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
			return ret
		}
		p.error(diagnostics.DiagnosticCodeStringLiteralExpected, tn.MakeRange(-1, -1))
	} else {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "from")
	}
	return nil
}

// parseImportDeclaration parses an import member: Identifier ('as' Identifier)?
func (p *Parser) parseImportDeclaration(tn *tokenizer.Tokenizer) *ast.ImportDeclaration {
	if tn.SkipIdentifier(tokenizer.IdentifierHandlingAlways) {
		identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
		var asIdentifier *ast.IdentifierExpression
		if tn.Skip(tokenizer.TokenAs, tokenizer.IdentifierHandlingDefault) {
			if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
				asIdentifier = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
			} else {
				p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
				return nil
			}
		}
		if asIdentifier != nil {
			return ast.NewImportDeclaration(
				identifier, asIdentifier,
				*diagnostics.JoinRanges(&identifier.Range, &asIdentifier.Range),
			)
		}
		return ast.NewImportDeclaration(identifier, nil, identifier.Range)
	}
	p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	return nil
}

// parseExportImport parses: 'export' 'import' Identifier '=' Identifier ';'?
func (p *Parser) parseExportImport(tn *tokenizer.Tokenizer, startPos int32) *ast.ExportImportStatement {
	if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
		asIdentifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
		if tn.Skip(tokenizer.TokenEquals, tokenizer.IdentifierHandlingDefault) {
			if tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
				identifier := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
				ret := ast.NewExportImportStatement(identifier, asIdentifier, *tn.MakeRange(startPos, tn.Pos))
				tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
				return ret
			}
			p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
		} else {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "=")
		}
	} else {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
	}
	return nil
}
