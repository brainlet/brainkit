package tokenizer

import (
	"math"
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
)

// mockSource satisfies diagnostics.Source for testing.
type mockSource struct {
	text           string
	normalizedPath string
	lineStarts     []int32
	lastColumn     int32
}

func newMockSource(path, text string) *mockSource {
	s := &mockSource{text: text, normalizedPath: path}
	s.lineStarts = []int32{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			s.lineStarts = append(s.lineStarts, int32(i+1))
		}
	}
	s.lineStarts = append(s.lineStarts, 0x7fffffff)
	return s
}

func (s *mockSource) SourceText() string          { return s.text }
func (s *mockSource) SourceNormalizedPath() string { return s.normalizedPath }
func (s *mockSource) LineAt(pos int32) int32 {
	l, r := 0, len(s.lineStarts)-1
	for l < r {
		m := l + ((r - l) >> 1)
		if pos < s.lineStarts[m] {
			r = m
		} else if pos < s.lineStarts[m+1] {
			s.lastColumn = pos - s.lineStarts[m] + 1
			return int32(m + 1)
		} else {
			l = m + 1
		}
	}
	panic("unreachable")
}
func (s *mockSource) ColumnAt() int32 { return s.lastColumn }

func tokenize(text string) *Tokenizer {
	src := newMockSource("test.ts", text)
	return NewTokenizer(src, nil)
}

func TestTokenFromKeyword(t *testing.T) {
	tests := []struct {
		text  string
		token Token
	}{
		{"abstract", TokenAbstract},
		{"as", TokenAs},
		{"async", TokenAsync},
		{"await", TokenAwait},
		{"break", TokenBreak},
		{"case", TokenCase},
		{"catch", TokenCatch},
		{"class", TokenClass},
		{"const", TokenConst},
		{"continue", TokenContinue},
		{"constructor", TokenConstructor},
		{"debugger", TokenDebugger},
		{"declare", TokenDeclare},
		{"default", TokenDefault},
		{"delete", TokenDelete},
		{"do", TokenDo},
		{"else", TokenElse},
		{"enum", TokenEnum},
		{"export", TokenExport},
		{"extends", TokenExtends},
		{"false", TokenFalse},
		{"finally", TokenFinally},
		{"for", TokenFor},
		{"from", TokenFrom},
		{"function", TokenFunction},
		{"get", TokenGet},
		{"if", TokenIf},
		{"implements", TokenImplements},
		{"import", TokenImport},
		{"in", TokenIn},
		{"instanceof", TokenInstanceOf},
		{"interface", TokenInterface},
		{"is", TokenIs},
		{"keyof", TokenKeyOf},
		{"let", TokenLet},
		{"module", TokenModule},
		{"namespace", TokenNamespace},
		{"new", TokenNew},
		{"null", TokenNull},
		{"of", TokenOf},
		{"override", TokenOverride},
		{"package", TokenPackage},
		{"private", TokenPrivate},
		{"protected", TokenProtected},
		{"public", TokenPublic},
		{"readonly", TokenReadonly},
		{"return", TokenReturn},
		{"set", TokenSet},
		{"static", TokenStatic},
		{"super", TokenSuper},
		{"switch", TokenSwitch},
		{"this", TokenThis},
		{"throw", TokenThrow},
		{"true", TokenTrue},
		{"try", TokenTry},
		{"type", TokenType},
		{"typeof", TokenTypeOf},
		{"var", TokenVar},
		{"void", TokenVoid},
		{"while", TokenWhile},
		{"with", TokenWith},
		{"yield", TokenYield},
		{"notakeyword", TokenInvalid},
		{"x", TokenInvalid},
		{"foo", TokenInvalid},
	}
	for _, tt := range tests {
		got := TokenFromKeyword(tt.text)
		if got != tt.token {
			t.Errorf("TokenFromKeyword(%q) = %d, want %d", tt.text, got, tt.token)
		}
	}
}

func TestTokenIsAlsoIdentifier(t *testing.T) {
	if !TokenIsAlsoIdentifier(TokenAbstract) {
		t.Error("abstract should be also identifier")
	}
	if !TokenIsAlsoIdentifier(TokenGet) {
		t.Error("get should be also identifier")
	}
	if TokenIsAlsoIdentifier(TokenIf) {
		t.Error("if should NOT be also identifier")
	}
	if TokenIsAlsoIdentifier(TokenWhile) {
		t.Error("while should NOT be also identifier")
	}
}

func TestIsIllegalVariableIdentifier(t *testing.T) {
	if !IsIllegalVariableIdentifier("delete") {
		t.Error("delete should be illegal")
	}
	if !IsIllegalVariableIdentifier("void") {
		t.Error("void should be illegal")
	}
	if IsIllegalVariableIdentifier("foo") {
		t.Error("foo should not be illegal")
	}
}

func TestOperatorTokenToString(t *testing.T) {
	if OperatorTokenToString(TokenPlus) != "+" {
		t.Error("+ mismatch")
	}
	if OperatorTokenToString(TokenAmpersandAmpersand) != "&&" {
		t.Error("&& mismatch")
	}
	if OperatorTokenToString(TokenGreaterThanGreaterThanGreaterThanEquals) != ">>>=" {
		t.Error(">>>= mismatch")
	}
}

func TestTokenizerBasicPunctuation(t *testing.T) {
	tok := tokenize("( ) { } [ ] ; , . :")
	expected := []Token{
		TokenOpenParen, TokenCloseParen,
		TokenOpenBrace, TokenCloseBrace,
		TokenOpenBracket, TokenCloseBracket,
		TokenSemicolon, TokenComma,
		TokenDot, TokenColon,
		TokenEndOfFile,
	}
	for _, exp := range expected {
		got := tok.Next(IdentifierHandlingDefault)
		if got != exp {
			t.Errorf("expected token %d, got %d", exp, got)
		}
	}
}

func TestTokenizerOperators(t *testing.T) {
	tok := tokenize("+ - * / % ** ++ -- << >> >>> & | ^ ~ ! = == === != !== < > <= >= && || ? =>")
	expected := []Token{
		TokenPlus, TokenMinus, TokenAsterisk, TokenSlash, TokenPercent,
		TokenAsteriskAsterisk, TokenPlusPlus, TokenMinusMinus,
		TokenLessThanLessThan, TokenGreaterThanGreaterThan, TokenGreaterThanGreaterThanGreaterThan,
		TokenAmpersand, TokenBar, TokenCaret, TokenTilde, TokenExclamation,
		TokenEquals, TokenEqualsEquals, TokenEqualsEqualsEquals,
		TokenExclamationEquals, TokenExclamationEqualsEquals,
		TokenLessThan, TokenGreaterThan, TokenLessThanEquals, TokenGreaterThanEquals,
		TokenAmpersandAmpersand, TokenBarBar, TokenQuestion, TokenEqualsGreaterThan,
		TokenEndOfFile,
	}
	for _, exp := range expected {
		got := tok.Next(IdentifierHandlingDefault)
		if got != exp {
			t.Errorf("expected token %d, got %d", exp, got)
		}
	}
}

func TestTokenizerAssignmentOperators(t *testing.T) {
	tok := tokenize("+= -= *= /= %= **= <<= >>= >>>= &= |= ^=")
	expected := []Token{
		TokenPlusEquals, TokenMinusEquals, TokenAsteriskEquals,
		TokenSlashEquals, TokenPercentEquals, TokenAsteriskAsteriskEquals,
		TokenLessThanLessThanEquals, TokenGreaterThanGreaterThanEquals,
		TokenGreaterThanGreaterThanGreaterThanEquals,
		TokenAmpersandEquals, TokenBarEquals, TokenCaretEquals,
		TokenEndOfFile,
	}
	for _, exp := range expected {
		got := tok.Next(IdentifierHandlingDefault)
		if got != exp {
			t.Errorf("expected token %d, got %d", exp, got)
		}
	}
}

func TestTokenizerKeywords(t *testing.T) {
	tok := tokenize("if else while for return var let const class function")
	expected := []Token{
		TokenIf, TokenElse, TokenWhile, TokenFor, TokenReturn,
		TokenVar, TokenLet, TokenConst, TokenClass, TokenFunction,
		TokenEndOfFile,
	}
	for _, exp := range expected {
		got := tok.Next(IdentifierHandlingDefault)
		if got != exp {
			t.Errorf("expected token %d, got %d", exp, got)
		}
	}
}

func TestTokenizerIdentifiers(t *testing.T) {
	tok := tokenize("foo bar baz _private $dollar")
	for i := 0; i < 5; i++ {
		got := tok.Next(IdentifierHandlingDefault)
		if got != TokenIdentifier {
			t.Errorf("expected Identifier, got %d", got)
		}
		_ = tok.ReadIdentifier()
	}
	if tok.Next(IdentifierHandlingDefault) != TokenEndOfFile {
		t.Error("expected EOF")
	}
}

func TestTokenizerIdentifierReading(t *testing.T) {
	tok := tokenize("hello world")
	tok.Next(IdentifierHandlingDefault)
	id := tok.ReadIdentifier()
	if id != "hello" {
		t.Errorf("expected 'hello', got %q", id)
	}
	tok.Next(IdentifierHandlingDefault)
	id = tok.ReadIdentifier()
	if id != "world" {
		t.Errorf("expected 'world', got %q", id)
	}
}

func TestTokenizerStringLiterals(t *testing.T) {
	tok := tokenize(`"hello" 'world'`)

	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenStringLiteral {
		t.Fatalf("expected StringLiteral, got %d", got)
	}
	s := tok.ReadString(0, false)
	if s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}

	got = tok.Next(IdentifierHandlingDefault)
	if got != TokenStringLiteral {
		t.Fatalf("expected StringLiteral, got %d", got)
	}
	s = tok.ReadString(0, false)
	if s != "world" {
		t.Errorf("expected 'world', got %q", s)
	}
}

func TestTokenizerStringEscapes(t *testing.T) {
	tok := tokenize(`"hello\nworld"`)
	tok.Next(IdentifierHandlingDefault)
	s := tok.ReadString(0, false)
	if s != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got %q", s)
	}
}

func TestTokenizerStringHexEscape(t *testing.T) {
	tok := tokenize(`"\x41\x42"`)
	tok.Next(IdentifierHandlingDefault)
	s := tok.ReadString(0, false)
	if s != "AB" {
		t.Errorf("expected 'AB', got %q", s)
	}
}

func TestTokenizerStringUnicodeEscape(t *testing.T) {
	tok := tokenize(`"\u0041\u0042"`)
	tok.Next(IdentifierHandlingDefault)
	s := tok.ReadString(0, false)
	if s != "AB" {
		t.Errorf("expected 'AB', got %q", s)
	}
}

func TestTokenizerIntegerLiterals(t *testing.T) {
	tests := []struct {
		text string
		val  int64
	}{
		{"42", 42},
		{"0", 0},
		{"123", 123},
		{"0x1F", 31},
		{"0xFF", 255},
		{"0b1010", 10},
		{"0o777", 511},
		{"1_000", 1000},
		{"0xFF_FF", 0xFFFF},
	}
	for _, tt := range tests {
		tok := tokenize(tt.text)
		got := tok.Next(IdentifierHandlingDefault)
		if got != TokenIntegerLiteral {
			t.Errorf("%s: expected IntegerLiteral, got %d", tt.text, got)
			continue
		}
		val := tok.ReadInteger()
		if val != tt.val {
			t.Errorf("%s: expected %d, got %d", tt.text, tt.val, val)
		}
	}
}

func TestTokenizerFloatLiterals(t *testing.T) {
	tests := []struct {
		text string
		val  float64
	}{
		{"1.5", 1.5},
		{"0.5", 0.5},
		{".5", 0.5},
		{"1e10", 1e10},
		{"1.5e2", 150.0},
		{"1_000.5", 1000.5},
	}
	for _, tt := range tests {
		tok := tokenize(tt.text)
		got := tok.Next(IdentifierHandlingDefault)
		if got != TokenFloatLiteral {
			t.Errorf("%s: expected FloatLiteral, got %d", tt.text, got)
			continue
		}
		val := tok.ReadFloat()
		if val != tt.val {
			t.Errorf("%s: expected %f, got %f", tt.text, tt.val, val)
		}
	}
}

func TestTokenizerSkipsBOM(t *testing.T) {
	// UTF-8 BOM: EF BB BF
	tok := tokenize("\xEF\xBB\xBFlet")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenLet {
		t.Errorf("expected Let after BOM, got %d", got)
	}
}

func TestTokenizerSkipsShebang(t *testing.T) {
	tok := tokenize("#!/usr/bin/env node\nlet")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenLet {
		t.Errorf("expected Let after shebang, got %d", got)
	}
}

func TestTokenizerComments(t *testing.T) {
	comments := make([]string, 0)
	tok := tokenize("// line comment\nlet /* block */ x")
	tok.OnComment = func(kind CommentKind, text string, rng interface{}) {
		comments = append(comments, text)
	}

	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenLet {
		t.Errorf("expected Let, got %d", got)
	}
	got = tok.Next(IdentifierHandlingDefault)
	if got != TokenIdentifier {
		t.Errorf("expected Identifier, got %d", got)
	}

	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0] != "// line comment\n" {
		t.Errorf("comment[0] = %q", comments[0])
	}
	if comments[1] != "/* block */" {
		t.Errorf("comment[1] = %q", comments[1])
	}
}

func TestTokenizerTripleSlashComment(t *testing.T) {
	var kinds []CommentKind
	tok := tokenize("/// triple\nlet")
	tok.OnComment = func(kind CommentKind, text string, rng interface{}) {
		kinds = append(kinds, kind)
	}
	tok.Next(IdentifierHandlingDefault)
	if len(kinds) != 1 || kinds[0] != CommentKindTriple {
		t.Error("expected triple slash comment kind")
	}
}

func TestTokenizerPeek(t *testing.T) {
	tok := tokenize("let x = 5")
	tok.Next(IdentifierHandlingDefault) // let
	peeked := tok.Peek(IdentifierHandlingDefault, math.MaxInt32)
	if peeked != TokenIdentifier {
		t.Errorf("peek expected Identifier, got %d", peeked)
	}
	// next should still return the same peeked token
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenIdentifier {
		t.Errorf("next after peek expected Identifier, got %d", got)
	}
}

func TestTokenizerPeekOnNewLine(t *testing.T) {
	tok := tokenize("let\nx")
	tok.Next(IdentifierHandlingDefault) // let
	if !tok.PeekOnNewLine() {
		t.Error("expected next token on new line")
	}

	tok2 := tokenize("let x")
	tok2.Next(IdentifierHandlingDefault) // let
	if tok2.PeekOnNewLine() {
		t.Error("expected next token NOT on new line")
	}
}

func TestTokenizerMarkReset(t *testing.T) {
	tok := tokenize("let x = 5")
	tok.Next(IdentifierHandlingDefault) // let
	state := tok.Mark()

	tok.Next(IdentifierHandlingDefault) // x
	tok.Next(IdentifierHandlingDefault) // =

	tok.Reset(state)
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenIdentifier {
		t.Errorf("after reset, expected Identifier, got %d", got)
	}
}

func TestTokenizerSkip(t *testing.T) {
	tok := tokenize("let x = 5")
	tok.Next(IdentifierHandlingDefault) // let

	if !tok.Skip(TokenIdentifier, IdentifierHandlingDefault) {
		t.Error("expected skip Identifier to succeed")
	}
	tok.ReadIdentifier() // must consume identifier to advance past it
	if !tok.Skip(TokenEquals, IdentifierHandlingDefault) {
		t.Error("expected skip Equals to succeed")
	}
	if tok.Skip(TokenSemicolon, IdentifierHandlingDefault) {
		t.Error("expected skip Semicolon to fail")
	}
}

func TestTokenizerRange(t *testing.T) {
	tok := tokenize("let x")
	tok.Next(IdentifierHandlingDefault) // let
	rng := tok.MakeRange(-1, -1)
	if rng.Start != 0 || rng.End != 3 {
		t.Errorf("range = {%d,%d}, want {0,3}", rng.Start, rng.End)
	}
}

func TestTokenizerDotDotDot(t *testing.T) {
	tok := tokenize("...x")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenDotDotDot {
		t.Errorf("expected DotDotDot, got %d", got)
	}
}

func TestTokenizerEmptySource(t *testing.T) {
	tok := tokenize("")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenEndOfFile {
		t.Errorf("expected EOF, got %d", got)
	}
}

func TestTokenizerWhitespaceOnly(t *testing.T) {
	tok := tokenize("   \t\n\r\n   ")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenEndOfFile {
		t.Errorf("expected EOF, got %d", got)
	}
}

func TestTokenizerIdentifierHandlingAlways(t *testing.T) {
	tok := tokenize("if")
	got := tok.Next(IdentifierHandlingAlways)
	if got != TokenIdentifier {
		t.Errorf("expected Identifier with Always handling, got %d", got)
	}
}

func TestTokenizerIdentifierHandlingPrefer(t *testing.T) {
	tok := tokenize("get")
	// 'get' with Prefer should return Identifier since it's also an identifier
	got := tok.Next(IdentifierHandlingPrefer)
	if got != TokenIdentifier {
		t.Errorf("expected Identifier with Prefer for 'get', got %d", got)
	}

	tok2 := tokenize("if")
	// 'if' with Prefer should still return If keyword since it's NOT also an identifier
	got2 := tok2.Next(IdentifierHandlingPrefer)
	if got2 != TokenIf {
		t.Errorf("expected If with Prefer for 'if', got %d", got2)
	}
}

func TestTokenizerArrowFunction(t *testing.T) {
	tok := tokenize("() => 42")
	expected := []Token{
		TokenOpenParen, TokenCloseParen, TokenEqualsGreaterThan,
		TokenIntegerLiteral, TokenEndOfFile,
	}
	for _, exp := range expected {
		got := tok.Next(IdentifierHandlingDefault)
		if got != exp {
			t.Errorf("expected %d, got %d", exp, got)
		}
		if got == TokenIntegerLiteral {
			tok.ReadInteger()
		}
	}
}

func TestTokenizerAtDecorator(t *testing.T) {
	tok := tokenize("@decorator")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenAt {
		t.Errorf("expected At, got %d", got)
	}
	got = tok.Next(IdentifierHandlingDefault)
	if got != TokenIdentifier {
		t.Errorf("expected Identifier, got %d", got)
	}
}

func TestTokenizerDiagnosticOnInvalidChar(t *testing.T) {
	tok := tokenize("\x01let")
	tok.Next(IdentifierHandlingDefault) // should skip invalid and return let
	if tok.Token != TokenLet {
		t.Errorf("expected Let after invalid char, got %d", tok.Token)
	}
	if len(tok.Diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(tok.Diagnostics))
	}
}

func TestTokenizerRegexp(t *testing.T) {
	// After reading a '/', the parser would call ReadRegexpPattern
	tok := tokenize("/abc/gi")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenSlash {
		t.Fatalf("expected Slash, got %d", got)
	}
	pattern := tok.ReadRegexpPattern()
	if pattern != "abc" {
		t.Errorf("expected 'abc', got %q", pattern)
	}
	tok.Pos++ // skip closing /
	flags := tok.ReadRegexpFlags()
	if flags != "gi" {
		t.Errorf("expected 'gi', got %q", flags)
	}
}

func TestTokenizerTemplateLiteral(t *testing.T) {
	tok := tokenize("`hello`")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenTemplateLiteral {
		t.Fatalf("expected TemplateLiteral, got %d", got)
	}
	s := tok.ReadString(0, false)
	if s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}
}

func TestTokenizerTemplateWithExpression(t *testing.T) {
	tok := tokenize("`hello ${name}`")
	got := tok.Next(IdentifierHandlingDefault)
	if got != TokenTemplateLiteral {
		t.Fatalf("expected TemplateLiteral, got %d", got)
	}
	s := tok.ReadString(0, false)
	if s != "hello " {
		t.Errorf("expected 'hello ', got %q", s)
	}
	if !tok.ReadingTemplateString {
		t.Error("expected readingTemplateString to be true")
	}
}

func TestTokenizerNumberSeparatorErrors(t *testing.T) {
	tok := tokenize("1__0")
	tok.Next(IdentifierHandlingDefault)
	tok.ReadInteger()
	if len(tok.Diagnostics) == 0 {
		t.Error("expected diagnostic for consecutive separators")
	}
}

func TestDiagnosticCodeUsage(t *testing.T) {
	// Verify that the diagnostic code constants we reference actually exist
	_ = diagnostics.DiagnosticCode0Expected
	_ = diagnostics.DiagnosticCodeInvalidCharacter
	_ = diagnostics.DiagnosticCodeUnterminatedStringLiteral
	_ = diagnostics.DiagnosticCodeUnexpectedEndOfText
	_ = diagnostics.DiagnosticCodeHexadecimalDigitExpected
	_ = diagnostics.DiagnosticCodeDigitExpected
	_ = diagnostics.DiagnosticCodeBinaryDigitExpected
	_ = diagnostics.DiagnosticCodeOctalDigitExpected
}
