package parser

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

// cf converts CommonFlags to int32 for use with AST flag fields.
func cf(f common.CommonFlags) int32 { return int32(f) }

// parseTopLevelStatement parses a top-level statement.
func (p *Parser) parseTopLevelStatement(tn *tokenizer.Tokenizer, namespace *ast.NamespaceDeclaration) ast.Node {
	var flags int32
	var statement ast.Node
	var decorators []*ast.DecoratorNode

	startPos := int32(-1)

	// check for decorators
	for tn.Skip(tokenizer.TokenAt, tokenizer.IdentifierHandlingDefault) {
		if startPos < 0 {
			startPos = tn.TokenPos
		}
		decorator := p.parseDecorator(tn)
		if decorator == nil {
			break
		}
		decorators = append(decorators, decorator)
	}

	// check for 'export'
	var exportStart, exportEnd int32
	if tn.Skip(tokenizer.TokenExport, tokenizer.IdentifierHandlingDefault) {
		if startPos < 0 {
			startPos = tn.TokenPos
		}
		flags |= cf(common.CommonFlagsExport)
		exportStart = tn.TokenPos
		exportEnd = tn.Pos
	}

	// check for 'default'
	var defaultStart, defaultEnd int32
	if exportEnd != 0 && tn.Skip(tokenizer.TokenDefault, tokenizer.IdentifierHandlingDefault) {
		defaultStart = tn.TokenPos
		defaultEnd = tn.Pos
	}

	// check for 'declare'
	var declareStart, declareEnd int32
	if tn.Skip(tokenizer.TokenDeclare, tokenizer.IdentifierHandlingDefault) {
		if startPos < 0 {
			startPos = tn.TokenPos
		}
		if namespace != nil && namespace.Is(cf(common.CommonFlagsAmbient)) {
			p.error(diagnostics.DiagnosticCodeADeclareModifierCannotBeUsedInAnAlreadyAmbientContext, tn.MakeRange(-1, -1))
		}
		flags |= cf(common.CommonFlagsDeclare | common.CommonFlagsAmbient)
		declareStart = tn.TokenPos
		declareEnd = tn.Pos
	} else if namespace != nil {
		if namespace.Is(cf(common.CommonFlagsAmbient)) {
			flags |= cf(common.CommonFlagsAmbient)
		}
	}

	// parse the actual statement
	token := tn.Peek(tokenizer.IdentifierHandlingPrefer, MaxInt32)
	if startPos < 0 {
		startPos = tn.TokenPos + 1
	}

	switch token {
	case tokenizer.TokenConst:
		tn.Next(tokenizer.IdentifierHandlingDefault)
		flags |= cf(common.CommonFlagsConst)
		if tn.Skip(tokenizer.TokenEnum, tokenizer.IdentifierHandlingDefault) {
			statement = p.parseEnum(tn, flags, decorators, startPos)
		} else {
			statement = p.parseVariable(tn, flags, decorators, startPos, false)
		}
		decorators = nil

	case tokenizer.TokenLet:
		tn.Next(tokenizer.IdentifierHandlingDefault)
		flags |= cf(common.CommonFlagsLet)
		statement = p.parseVariable(tn, flags, decorators, startPos, false)
		decorators = nil

	case tokenizer.TokenVar:
		tn.Next(tokenizer.IdentifierHandlingDefault)
		statement = p.parseVariable(tn, flags, decorators, startPos, false)
		decorators = nil

	case tokenizer.TokenEnum:
		tn.Next(tokenizer.IdentifierHandlingDefault)
		statement = p.parseEnum(tn, flags, decorators, startPos)
		decorators = nil

	case tokenizer.TokenFunction:
		tn.Next(tokenizer.IdentifierHandlingDefault)
		statement = p.parseFunction(tn, flags, decorators, startPos)
		decorators = nil

	case tokenizer.TokenAbstract:
		state := tn.Mark()
		tn.Next(tokenizer.IdentifierHandlingDefault)
		abstractStart := tn.TokenPos
		abstractEnd := tn.Pos
		if tn.PeekOnNewLine() {
			tn.Reset(state)
			statement = p.parseStatement(tn, true)
			break
		}
		next := tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32)
		if next != tokenizer.TokenClass {
			if next == tokenizer.TokenInterface {
				p.error(
					diagnostics.DiagnosticCodeAbstractModifierCanOnlyAppearOnAClassMethodOrPropertyDeclaration,
					tn.MakeRange(abstractStart, abstractEnd),
				)
			}
			tn.Reset(state)
			statement = p.parseStatement(tn, true)
			break
		}
		tn.Discard(state)
		flags |= cf(common.CommonFlagsAbstract)
		fallthrough

	case tokenizer.TokenClass, tokenizer.TokenInterface:
		tn.Next(tokenizer.IdentifierHandlingDefault)
		statement = p.parseClassOrInterface(tn, flags, decorators, startPos)
		decorators = nil

	case tokenizer.TokenNamespace:
		state := tn.Mark()
		tn.Next(tokenizer.IdentifierHandlingDefault)
		if tn.Peek(tokenizer.IdentifierHandlingPrefer, MaxInt32) == tokenizer.TokenIdentifier {
			tn.Discard(state)
			statement = p.parseNamespace(tn, flags, decorators, startPos)
			decorators = nil
		} else {
			tn.Reset(state)
			statement = p.parseStatement(tn, true)
		}

	case tokenizer.TokenImport:
		tn.Next(tokenizer.IdentifierHandlingDefault)
		flags |= cf(common.CommonFlagsImport)
		if flags&cf(common.CommonFlagsExport) != 0 {
			statement = p.parseExportImport(tn, startPos)
		} else {
			statement = p.parseImport(tn)
		}

	case tokenizer.TokenType:
		state := tn.Mark()
		tn.Next(tokenizer.IdentifierHandlingDefault)
		if tn.Peek(tokenizer.IdentifierHandlingPrefer, MaxInt32) == tokenizer.TokenIdentifier {
			tn.Discard(state)
			statement = p.parseTypeDeclaration(tn, flags, decorators, startPos)
			decorators = nil
		} else {
			tn.Reset(state)
			statement = p.parseStatement(tn, true)
		}

	case tokenizer.TokenModule:
		state := tn.Mark()
		tn.Next(tokenizer.IdentifierHandlingDefault)
		if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenStringLiteral && !tn.PeekOnNewLine() {
			tn.Discard(state)
			statement = p.parseModuleDeclaration(tn, flags)
		} else {
			tn.Reset(state)
			statement = p.parseStatement(tn, true)
		}

	default:
		// handle plain exports
		if flags&cf(common.CommonFlagsExport) != 0 {
			if defaultEnd != 0 && tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer) {
				if declareEnd != 0 {
					p.error(
						diagnostics.DiagnosticCodeAnExportAssignmentCannotHaveModifiers,
						tn.MakeRange(declareStart, declareEnd),
					)
				}
				statement = p.parseExportDefaultAlias(tn, startPos, defaultStart, defaultEnd)
				defaultStart = 0
				defaultEnd = 0
			} else {
				statement = p.parseExport(tn, startPos, flags&cf(common.CommonFlagsDeclare) != 0)
			}
		} else {
			if exportEnd != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
					tn.MakeRange(exportStart, exportEnd), "export")
			}
			if declareEnd != 0 {
				p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
					tn.MakeRange(declareStart, declareEnd), "declare")
			}
			if namespace != nil {
				p.error(diagnostics.DiagnosticCodeNamespaceCanOnlyHaveDeclarations,
					tn.MakeRange(startPos, -1))
			} else {
				statement = p.parseStatement(tn, true)
			}
		}
	}

	// check for decorators that weren't consumed
	if decorators != nil {
		for _, d := range decorators {
			p.error(diagnostics.DiagnosticCodeDecoratorsAreNotValidHere, d.GetRange())
		}
	}

	// check if this is an `export default` declaration
	if defaultEnd != 0 && statement != nil {
		switch statement.GetKind() {
		case ast.NodeKindEnumDeclaration, ast.NodeKindFunctionDeclaration,
			ast.NodeKindClassDeclaration, ast.NodeKindInterfaceDeclaration,
			ast.NodeKindNamespaceDeclaration:
			return ast.NewExportDefaultStatement(statement, *tn.MakeRange(startPos, tn.Pos))
		default:
			p.error(diagnostics.DiagnosticCode0ModifierCannotBeUsedHere,
				tn.MakeRange(defaultStart, defaultEnd), "default")
		}
	}
	return statement
}

// parseStatement parses a statement.
func (p *Parser) parseStatement(tn *tokenizer.Tokenizer, topLevel bool) ast.Node {
	state := tn.Mark()
	token := tn.Next(tokenizer.IdentifierHandlingDefault)
	var statement ast.Node

	switch token {
	case tokenizer.TokenBreak:
		statement = p.parseBreak(tn)
	case tokenizer.TokenConst:
		statement = p.parseVariable(tn, cf(common.CommonFlagsConst), nil, tn.TokenPos, false)
	case tokenizer.TokenContinue:
		statement = p.parseContinue(tn)
	case tokenizer.TokenDo:
		statement = p.parseDoStatement(tn)
	case tokenizer.TokenFor:
		statement = p.parseForStatement(tn)
	case tokenizer.TokenIf:
		statement = p.parseIfStatement(tn)
	case tokenizer.TokenLet:
		statement = p.parseVariable(tn, cf(common.CommonFlagsLet), nil, tn.TokenPos, false)
	case tokenizer.TokenVar:
		statement = p.parseVariable(tn, cf(common.CommonFlagsNone), nil, tn.TokenPos, false)
	case tokenizer.TokenOpenBrace:
		statement = p.parseBlockStatement(tn, topLevel)
	case tokenizer.TokenReturn:
		if topLevel {
			p.error(diagnostics.DiagnosticCodeAReturnStatementCanOnlyBeUsedWithinAFunctionBody, tn.MakeRange(-1, -1))
		}
		statement = p.parseReturn(tn)
	case tokenizer.TokenSemicolon:
		return ast.NewEmptyStatement(*tn.MakeRange(tn.TokenPos, -1))
	case tokenizer.TokenSwitch:
		statement = p.parseSwitchStatement(tn)
	case tokenizer.TokenThrow:
		statement = p.parseThrowStatement(tn)
	case tokenizer.TokenTry:
		statement = p.parseTryStatement(tn)
	case tokenizer.TokenVoid:
		statement = p.parseVoidStatement(tn)
	case tokenizer.TokenWhile:
		statement = p.parseWhileStatement(tn)
	case tokenizer.TokenType:
		if tn.Peek(tokenizer.IdentifierHandlingPrefer, MaxInt32) == tokenizer.TokenIdentifier {
			statement = p.parseTypeDeclaration(tn, cf(common.CommonFlagsNone), nil, tn.TokenPos)
			break
		}
		fallthrough
	default:
		tn.Reset(state)
		statement = p.parseExpressionStatement(tn)
	}

	if statement == nil {
		tn.Reset(state)
		p.skipStatement(tn)
	} else {
		tn.Discard(state)
	}
	return statement
}

// parseBlockStatement parses a block statement: '{' Statement* '}' ';'?
func (p *Parser) parseBlockStatement(tn *tokenizer.Tokenizer, topLevel bool) *ast.BlockStatement {
	startPos := tn.TokenPos
	var statements []ast.Node
	for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
		state := tn.Mark()
		statement := p.parseStatement(tn, topLevel)
		if statement == nil {
			if tn.Token == tokenizer.TokenEndOfFile {
				return nil
			}
			tn.Reset(state)
			p.skipStatement(tn)
		} else {
			tn.Discard(state)
			statements = append(statements, statement)
		}
	}
	ret := ast.NewBlockStatement(statements, *tn.MakeRange(startPos, tn.Pos))
	if topLevel {
		tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	}
	return ret
}

// parseBreak parses a break statement: 'break' Identifier? ';'?
func (p *Parser) parseBreak(tn *tokenizer.Tokenizer) *ast.BreakStatement {
	var identifier *ast.IdentifierExpression
	if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenIdentifier && !tn.PeekOnNewLine() {
		tn.Next(tokenizer.IdentifierHandlingPrefer)
		identifier = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	}
	ret := ast.NewBreakStatement(identifier, *tn.MakeRange(-1, -1))
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseContinue parses a continue statement: 'continue' Identifier? ';'?
func (p *Parser) parseContinue(tn *tokenizer.Tokenizer) *ast.ContinueStatement {
	var identifier *ast.IdentifierExpression
	if tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) == tokenizer.TokenIdentifier && !tn.PeekOnNewLine() {
		tn.Next(tokenizer.IdentifierHandlingPrefer)
		identifier = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	}
	ret := ast.NewContinueStatement(identifier, *tn.MakeRange(-1, -1))
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseDoStatement parses a do-while statement: 'do' Statement 'while' '(' Expression ')' ';'?
func (p *Parser) parseDoStatement(tn *tokenizer.Tokenizer) *ast.DoStatement {
	startPos := tn.TokenPos
	statement := p.parseStatement(tn, false)
	if statement == nil {
		return nil
	}
	if tn.Skip(tokenizer.TokenWhile, tokenizer.IdentifierHandlingDefault) {
		if tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
			condition := p.parseExpression(tn, PrecedenceComma)
			if condition == nil {
				return nil
			}
			if tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
				ret := ast.NewDoStatement(statement, condition, *tn.MakeRange(startPos, tn.Pos))
				tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
				return ret
			}
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
		} else {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "(")
		}
	} else {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "while")
	}
	return nil
}

// parseExpressionStatement parses an expression statement.
func (p *Parser) parseExpressionStatement(tn *tokenizer.Tokenizer) *ast.ExpressionStatement {
	expr := p.parseExpression(tn, PrecedenceComma)
	if expr == nil {
		return nil
	}
	ret := ast.NewExpressionStatement(expr)
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseForStatement parses a for statement.
func (p *Parser) parseForStatement(tn *tokenizer.Tokenizer) ast.Node {
	startPos := tn.TokenPos
	if !tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "(")
		return nil
	}

	var initializer ast.Node
	if tn.Skip(tokenizer.TokenConst, tokenizer.IdentifierHandlingDefault) {
		initializer = p.parseVariable(tn, cf(common.CommonFlagsConst), nil, tn.TokenPos, true)
	} else if tn.Skip(tokenizer.TokenLet, tokenizer.IdentifierHandlingDefault) {
		initializer = p.parseVariable(tn, cf(common.CommonFlagsLet), nil, tn.TokenPos, true)
	} else if tn.Skip(tokenizer.TokenVar, tokenizer.IdentifierHandlingDefault) {
		initializer = p.parseVariable(tn, cf(common.CommonFlagsNone), nil, tn.TokenPos, true)
	} else if !tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault) {
		initializer = p.parseExpressionStatement(tn)
		if initializer == nil {
			return nil
		}
	}

	if initializer != nil {
		if tn.Skip(tokenizer.TokenOf, tokenizer.IdentifierHandlingDefault) {
			if initializer.GetKind() == ast.NodeKindExpression {
				es := initializer.(*ast.ExpressionStatement)
				if es.Expression.GetKind() != ast.NodeKindIdentifier {
					p.error(diagnostics.DiagnosticCodeIdentifierExpected, initializer.GetRange())
					return nil
				}
				return p.parseForOfStatement(tn, startPos, initializer)
			}
			if initializer.GetKind() == ast.NodeKindVariable {
				vs := initializer.(*ast.VariableStatement)
				for _, decl := range vs.Declarations {
					if decl.Initializer != nil {
						p.error(
							diagnostics.DiagnosticCodeTheVariableDeclarationOfAForOfStatementCannotHaveAnInitializer,
							decl.Initializer.GetRange(),
						)
					}
				}
				return p.parseForOfStatement(tn, startPos, initializer)
			}
			p.error(diagnostics.DiagnosticCodeIdentifierExpected, initializer.GetRange())
			return nil
		}
		// non-for..of needs type or initializer
		if initializer.GetKind() == ast.NodeKindVariable {
			vs := initializer.(*ast.VariableStatement)
			for _, decl := range vs.Declarations {
				if decl.Initializer == nil {
					if decl.Flags&cf(common.CommonFlagsConst) != 0 {
						p.error(diagnostics.DiagnosticCodeConstDeclarationsMustBeInitialized, decl.Name.GetRange())
					} else if decl.Type == nil {
						p.error(diagnostics.DiagnosticCodeTypeExpected, decl.Name.GetRange().AtEnd())
					}
				}
			}
		}
	}

	if tn.Token == tokenizer.TokenSemicolon {
		var condition *ast.ExpressionStatement
		if !tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault) {
			condition = p.parseExpressionStatement(tn)
			if condition == nil {
				return nil
			}
		}

		if tn.Token == tokenizer.TokenSemicolon {
			var incrementor ast.Node
			if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
				incrementor = p.parseExpression(tn, PrecedenceComma)
				if incrementor == nil {
					return nil
				}
				if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
					p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
					return nil
				}
			}
			statement := p.parseStatement(tn, false)
			if statement == nil {
				return nil
			}
			var condExpr ast.Node
			if condition != nil {
				condExpr = condition.Expression
			}
			return ast.NewForStatement(initializer, condExpr, incrementor, statement, *tn.MakeRange(startPos, tn.Pos))
		}
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ";")
	} else {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ";")
	}
	return nil
}

// parseForOfStatement parses a for-of statement: 'of' Expression ')' Statement
func (p *Parser) parseForOfStatement(tn *tokenizer.Tokenizer, startPos int32, variable ast.Node) *ast.ForOfStatement {
	iterable := p.parseExpression(tn, PrecedenceComma)
	if iterable == nil {
		return nil
	}
	if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
		return nil
	}
	statement := p.parseStatement(tn, false)
	if statement == nil {
		return nil
	}
	return ast.NewForOfStatement(variable, iterable, statement, *tn.MakeRange(startPos, tn.Pos))
}

// parseIfStatement parses an if statement: 'if' '(' Expression ')' Statement ('else' Statement)?
func (p *Parser) parseIfStatement(tn *tokenizer.Tokenizer) *ast.IfStatement {
	startPos := tn.TokenPos
	if !tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "(")
		return nil
	}
	condition := p.parseExpression(tn, PrecedenceComma)
	if condition == nil {
		return nil
	}
	if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
		return nil
	}
	statement := p.parseStatement(tn, false)
	if statement == nil {
		return nil
	}
	var elseStatement ast.Node
	if tn.Skip(tokenizer.TokenElse, tokenizer.IdentifierHandlingDefault) {
		elseStatement = p.parseStatement(tn, false)
		if elseStatement == nil {
			return nil
		}
	}
	return ast.NewIfStatement(condition, statement, elseStatement, *tn.MakeRange(startPos, tn.Pos))
}

// parseSwitchStatement parses a switch statement.
func (p *Parser) parseSwitchStatement(tn *tokenizer.Tokenizer) *ast.SwitchStatement {
	startPos := tn.TokenPos
	if !tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "(")
		return nil
	}
	condition := p.parseExpression(tn, PrecedenceComma)
	if condition == nil {
		return nil
	}
	if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
		return nil
	}
	if !tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
		return nil
	}
	var switchCases []*ast.SwitchCase
	for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
		switchCase := p.parseSwitchCase(tn)
		if switchCase == nil {
			return nil
		}
		switchCases = append(switchCases, switchCase)
	}
	ret := ast.NewSwitchStatement(condition, switchCases, *tn.MakeRange(startPos, tn.Pos))
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseSwitchCase parses a switch case.
func (p *Parser) parseSwitchCase(tn *tokenizer.Tokenizer) *ast.SwitchCase {
	startPos := tn.TokenPos

	if tn.Skip(tokenizer.TokenCase, tokenizer.IdentifierHandlingDefault) {
		label := p.parseExpression(tn, PrecedenceComma)
		if label == nil {
			return nil
		}
		if !tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ":")
			return nil
		}
		var statements []ast.Node
		for tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenCase &&
			tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenDefault &&
			tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenCloseBrace {
			statement := p.parseStatement(tn, false)
			if statement == nil {
				return nil
			}
			statements = append(statements, statement)
		}
		return ast.NewSwitchCase(label, statements, *tn.MakeRange(startPos, tn.Pos))
	} else if tn.Skip(tokenizer.TokenDefault, tokenizer.IdentifierHandlingDefault) {
		if !tn.Skip(tokenizer.TokenColon, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ":")
			return nil
		}
		var statements []ast.Node
		for tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenCase &&
			tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenDefault &&
			tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenCloseBrace {
			statement := p.parseStatement(tn, false)
			if statement == nil {
				return nil
			}
			statements = append(statements, statement)
		}
		return ast.NewSwitchCase(nil, statements, *tn.MakeRange(startPos, tn.Pos))
	}
	p.error(diagnostics.DiagnosticCodeCaseOrDefaultExpected, tn.MakeRange(-1, -1))
	return nil
}

// parseThrowStatement parses a throw statement: 'throw' Expression ';'?
func (p *Parser) parseThrowStatement(tn *tokenizer.Tokenizer) *ast.ThrowStatement {
	startPos := tn.TokenPos
	expression := p.parseExpression(tn, PrecedenceComma)
	if expression == nil {
		return nil
	}
	ret := ast.NewThrowStatement(expression, *tn.MakeRange(startPos, tn.Pos))
	if !tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault) {
		p.checkASI(tn)
	}
	return ret
}

// parseTryStatement parses a try statement.
func (p *Parser) parseTryStatement(tn *tokenizer.Tokenizer) *ast.TryStatement {
	startPos := tn.TokenPos
	if !tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
		return nil
	}
	var bodyStatements []ast.Node
	for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
		stmt := p.parseStatement(tn, false)
		if stmt == nil {
			return nil
		}
		bodyStatements = append(bodyStatements, stmt)
	}
	var catchVariable *ast.IdentifierExpression
	var catchStatements []ast.Node
	var finallyStatements []ast.Node
	if tn.Skip(tokenizer.TokenCatch, tokenizer.IdentifierHandlingDefault) {
		if !tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "(")
			return nil
		}
		if !tn.SkipIdentifier(tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
			return nil
		}
		catchVariable = ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
		if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
			return nil
		}
		if !tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
			return nil
		}
		catchStatements = []ast.Node{}
		for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
			stmt := p.parseStatement(tn, false)
			if stmt == nil {
				return nil
			}
			catchStatements = append(catchStatements, stmt)
		}
	}
	if tn.Skip(tokenizer.TokenFinally, tokenizer.IdentifierHandlingDefault) {
		if !tn.Skip(tokenizer.TokenOpenBrace, tokenizer.IdentifierHandlingDefault) {
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "{")
			return nil
		}
		finallyStatements = []ast.Node{}
		for !tn.Skip(tokenizer.TokenCloseBrace, tokenizer.IdentifierHandlingDefault) {
			stmt := p.parseStatement(tn, false)
			if stmt == nil {
				return nil
			}
			finallyStatements = append(finallyStatements, stmt)
		}
	}
	if catchStatements == nil && finallyStatements == nil {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "catch")
		return nil
	}
	ret := ast.NewTryStatement(bodyStatements, catchVariable, catchStatements, finallyStatements, *tn.MakeRange(startPos, tn.Pos))
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseReturn parses a return statement: 'return' Expression? ';'?
func (p *Parser) parseReturn(tn *tokenizer.Tokenizer) *ast.ReturnStatement {
	startPos := tn.TokenPos
	var expr ast.Node
	nextToken := tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32)
	if nextToken != tokenizer.TokenSemicolon &&
		nextToken != tokenizer.TokenCloseBrace &&
		!tn.PeekOnNewLine() {
		expr = p.parseExpression(tn, PrecedenceComma)
		if expr == nil {
			return nil
		}
	}
	ret := ast.NewReturnStatement(expr, *tn.MakeRange(startPos, tn.Pos))
	if !tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault) {
		p.checkASI(tn)
	}
	return ret
}

// parseVoidStatement parses a void statement: 'void' Expression ';'?
func (p *Parser) parseVoidStatement(tn *tokenizer.Tokenizer) *ast.VoidStatement {
	startPos := tn.TokenPos
	expression := p.parseExpression(tn, PrecedenceGrouping)
	if expression == nil {
		return nil
	}
	ret := ast.NewVoidStatement(expression, *tn.MakeRange(startPos, tn.Pos))
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseWhileStatement parses a while statement: 'while' '(' Expression ')' Statement ';'?
func (p *Parser) parseWhileStatement(tn *tokenizer.Tokenizer) *ast.WhileStatement {
	startPos := tn.TokenPos
	if !tn.Skip(tokenizer.TokenOpenParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "(")
		return nil
	}
	expression := p.parseExpression(tn, PrecedenceComma)
	if expression == nil {
		return nil
	}
	if !tn.Skip(tokenizer.TokenCloseParen, tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), ")")
		return nil
	}
	statement := p.parseStatement(tn, false)
	if statement == nil {
		return nil
	}
	ret := ast.NewWhileStatement(expression, statement, *tn.MakeRange(startPos, tn.Pos))
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}

// parseTypeDeclaration parses a type declaration: 'type' Identifier ('=' Type)? ';'?
func (p *Parser) parseTypeDeclaration(tn *tokenizer.Tokenizer, flags int32, decorators []*ast.DecoratorNode, startPos int32) *ast.TypeDeclaration {
	if !tn.SkipIdentifier(tokenizer.IdentifierHandlingDefault) {
		p.error(diagnostics.DiagnosticCodeIdentifierExpected, tn.MakeRange(-1, -1))
		return nil
	}
	name := ast.NewIdentifierExpression(tn.ReadIdentifier(), *tn.MakeRange(-1, -1), false)
	var typeParameters []*ast.TypeParameterNode
	if tn.Skip(tokenizer.TokenLessThan, tokenizer.IdentifierHandlingDefault) {
		typeParameters = p.parseTypeParameters(tn)
		if typeParameters == nil {
			return nil
		}
		flags |= cf(common.CommonFlagsGeneric)
	}
	if tn.Skip(tokenizer.TokenEquals, tokenizer.IdentifierHandlingDefault) {
		tn.Skip(tokenizer.TokenBar, tokenizer.IdentifierHandlingDefault)
		typ := p.parseType(tn, true, false)
		if typ == nil {
			return nil
		}
		if isCircularTypeAlias(name.Text, typ) {
			p.error(diagnostics.DiagnosticCodeTypeAlias0CircularlyReferencesItself, name.GetRange(), name.Text)
			return nil
		}
		ret := ast.NewTypeDeclaration(name, decorators, flags, typeParameters, typ, *tn.MakeRange(startPos, tn.Pos))
		tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
		ret.OverriddenModuleName = p.currentModuleName
		ret.HasOverriddenModule = p.currentModuleName != ""
		return ret
	}
	p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "=")
	return nil
}

// parseModuleDeclaration parses a module declaration: 'module' StringLiteral ';'?
func (p *Parser) parseModuleDeclaration(tn *tokenizer.Tokenizer, flags int32) *ast.ModuleDeclaration {
	startPos := tn.TokenPos
	tn.Next(tokenizer.IdentifierHandlingDefault) // consume string literal
	moduleName := tn.ReadString(0, false)
	ret := ast.NewModuleDeclaration(moduleName, flags, *tn.MakeRange(startPos, tn.Pos))
	p.currentModuleName = moduleName
	tn.Skip(tokenizer.TokenSemicolon, tokenizer.IdentifierHandlingDefault)
	return ret
}
