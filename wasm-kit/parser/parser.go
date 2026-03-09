package parser

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// CommentHandler is a callback invoked when comments are encountered.
type CommentHandler func(kind tokenizer.CommentKind, text string, rng *diagnostics.Range)

// Dependee tracks where a dependency was imported from.
type Dependee struct {
	Source *ast.Source
	Path   *ast.StringLiteralExpression
}

// Parser parses an AssemblyScript source file into an AST.
type Parser struct {
	diagnostics.DiagnosticEmitter

	// backlog contains files queued for parsing.
	backlog []string
	// seenlog tracks files that have been encountered.
	seenlog map[string]bool
	// donelog tracks files that have been fully parsed.
	donelog map[string]bool

	// OnComment is a callback invoked when comments are encountered.
	OnComment CommentHandler

	// currentSource is the source currently being parsed.
	currentSource *ast.Source
	// dependees maps internal paths to their dependees.
	dependees map[string]*Dependee
	// sources holds all parsed sources.
	sources []*ast.Source
	// currentModuleName is set by module declarations.
	currentModuleName string

	// tryParseSignatureIsSignature indicates whether tryParseFunctionType
	// determined that it is handling a signature.
	tryParseSignatureIsSignature bool
	// parseParametersThis is the explicit this type from parseParameters.
	parseParametersThis *ast.NamedTypeNode
}

// NewParser creates a new parser.
func NewParser(diags []*diagnostics.DiagnosticMessage) *Parser {
	return &Parser{
		DiagnosticEmitter: diagnostics.NewDiagnosticEmitter(diags),
		seenlog:           make(map[string]bool),
		donelog:           make(map[string]bool),
		dependees:         make(map[string]*Dependee),
	}
}

// ParseFile parses a file by path and text, adding it to the program.
func (p *Parser) ParseFile(
	text string,
	path string,
	isEntry bool,
) {
	normalizedPath := util.NormalizePath(path)
	if _, ok := p.donelog[normalizedPath]; ok {
		return
	}
	p.donelog[normalizedPath] = true
	p.seenlog[normalizedPath] = true

	source := ast.NewSource(ast.SourceKindUser, normalizedPath, text)
	p.currentSource = source
	p.sources = append(p.sources, source)

	tn := tokenizer.NewTokenizer(source, p.DiagnosticEmitter.Diagnostics)
	if p.OnComment != nil {
		tn.OnComment = func(kind tokenizer.CommentKind, text string, rng interface{}) {
			p.OnComment(kind, text, rng.(*diagnostics.Range))
		}
	}

	for tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32) != tokenizer.TokenEndOfFile {
		statement := p.parseTopLevelStatement(tn, nil)
		if statement != nil {
			source.Statements = append(source.Statements, statement)
		} else {
			p.skipStatement(tn)
			if tn.Skip(tokenizer.TokenEndOfFile, tokenizer.IdentifierHandlingDefault) {
				break
			}
		}
	}
	source.Range.End = tn.Pos
	p.currentModuleName = ""
}

// NextFile returns the next file to parse from the backlog, or empty string if empty.
func (p *Parser) NextFile() string {
	if len(p.backlog) == 0 {
		return ""
	}
	file := p.backlog[0]
	p.backlog = p.backlog[1:]
	return file
}

// GetDependee returns the internal path of the dependee of the given imported file.
func (p *Parser) GetDependee(dependent string) string {
	if dep, ok := p.dependees[dependent]; ok {
		return dep.Source.InternalPath
	}
	return ""
}

// Finish finalizes parsing.
func (p *Parser) Finish() {
	if len(p.backlog) > 0 {
		panic("backlog is not empty")
	}
	p.backlog = nil
	p.seenlog = make(map[string]bool)
	p.donelog = make(map[string]bool)
	p.dependees = make(map[string]*Dependee)
}

// Sources returns all parsed sources.
func (p *Parser) Sources() []*ast.Source {
	return p.sources
}

// MaxInt32 is used as a default max compound length for tokenizer methods.
const MaxInt32 = int32(0x7FFFFFFF)

// error emits an error diagnostic at the given range with optional string arguments.
func (p *Parser) error(code diagnostics.DiagnosticCode, rng *diagnostics.Range, args ...string) {
	arg0, arg1, arg2 := "", "", ""
	if len(args) > 0 {
		arg0 = args[0]
	}
	if len(args) > 1 {
		arg1 = args[1]
	}
	if len(args) > 2 {
		arg2 = args[2]
	}
	p.Error(code, rng, arg0, arg1, arg2)
}

// checkASI checks for Automatic Semicolon Insertion.
func (p *Parser) checkASI(tn *tokenizer.Tokenizer) {
	nextToken := tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32)
	if nextToken == tokenizer.TokenEndOfFile || nextToken == tokenizer.TokenCloseBrace || tn.PeekOnNewLine() {
		return
	}
	p.error(
		diagnostics.DiagnosticCodeUnexpectedToken,
		tn.MakeRange(tn.NextTokenPos(), -1),
	)
}

// skipStatement skips over a statement on errors.
func (p *Parser) skipStatement(tn *tokenizer.Tokenizer) {
	if tn.PeekOnNewLine() {
		tn.Next(tokenizer.IdentifierHandlingDefault)
	}
	for {
		nextToken := tn.Peek(tokenizer.IdentifierHandlingDefault, MaxInt32)
		if nextToken == tokenizer.TokenEndOfFile || nextToken == tokenizer.TokenSemicolon {
			tn.Next(tokenizer.IdentifierHandlingDefault)
			break
		}
		if tn.PeekOnNewLine() {
			break
		}
		switch tn.Next(tokenizer.IdentifierHandlingDefault) {
		case tokenizer.TokenIdentifier:
			tn.ReadIdentifier()
		case tokenizer.TokenStringLiteral, tokenizer.TokenTemplateLiteral:
			tn.ReadString(0, false)
		case tokenizer.TokenIntegerLiteral:
			tn.ReadInteger()
			tn.CheckForIdentifierStartAfterNumericLiteral()
		case tokenizer.TokenFloatLiteral:
			tn.ReadFloat()
			tn.CheckForIdentifierStartAfterNumericLiteral()
		case tokenizer.TokenOpenBrace:
			p.skipBlock(tn)
		}
	}
	tn.ReadingTemplateString = false
}

// skipBlock skips over a block on errors.
func (p *Parser) skipBlock(tn *tokenizer.Tokenizer) {
	depth := int32(1)
	for depth > 0 {
		switch tn.Next(tokenizer.IdentifierHandlingDefault) {
		case tokenizer.TokenEndOfFile:
			p.error(diagnostics.DiagnosticCode0Expected, tn.MakeRange(-1, -1), "}")
			return
		case tokenizer.TokenOpenBrace:
			depth++
		case tokenizer.TokenCloseBrace:
			depth--
		case tokenizer.TokenIdentifier:
			tn.ReadIdentifier()
		case tokenizer.TokenStringLiteral:
			tn.ReadString(0, false)
		case tokenizer.TokenTemplateLiteral:
			tn.ReadString(0, false)
			for tn.ReadingTemplateString {
				p.skipBlock(tn)
				tn.ReadString('`', false)
			}
		case tokenizer.TokenIntegerLiteral:
			tn.ReadInteger()
			tn.CheckForIdentifierStartAfterNumericLiteral()
		case tokenizer.TokenFloatLiteral:
			tn.ReadFloat()
			tn.CheckForIdentifierStartAfterNumericLiteral()
		}
	}
}

// Ensure common package is used.
var _ = common.CommonFlagsNone
