package tokenizer

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// Tokenizer scans over a source file and returns one syntactic token at a time.
type Tokenizer struct {
	diagnostics.DiagnosticEmitter

	source diagnostics.Source
	text   string // cached from source.SourceText()
	end    int32

	Pos      int32
	Token    Token
	TokenPos int32

	nextToken          Token
	nextTokenPos       int32
	nextTokenOnNewLine onNewLine

	OnComment CommentHandler

	ReadingTemplateString bool
	ReadStringStart       int32
	ReadStringEnd         int32
}

// NewTokenizer constructs a new tokenizer for the given source.
func NewTokenizer(source diagnostics.Source, diags []*diagnostics.DiagnosticMessage) *Tokenizer {
	t := &Tokenizer{
		DiagnosticEmitter: diagnostics.NewDiagnosticEmitter(diags),
		source:            source,
		text:              source.SourceText(),
		nextToken:         -1,
	}

	text := t.text
	end := int32(len(text))
	pos := int32(0)

	// skip BOM
	if pos < end && text[pos] == 0xEF && pos+2 < end && text[pos+1] == 0xBB && text[pos+2] == 0xBF {
		pos += 3 // UTF-8 BOM is 3 bytes
	}

	// skip shebang
	if pos+1 < end && text[pos] == '#' && text[pos+1] == '!' {
		pos += 2
		for pos < end && text[pos] != '\n' {
			pos++
		}
	}

	t.Pos = pos
	t.end = end
	return t
}

// Next advances to the next token, skipping invalid tokens.
func (t *Tokenizer) Next(identifierHandling IdentifierHandling) Token {
	t.clearNextToken()
	var token Token
	for {
		token = t.unsafeNext(identifierHandling, math.MaxInt32)
		if token != TokenInvalid {
			break
		}
	}
	t.Token = token
	return token
}

// unsafeNext scans the next token without skipping invalid ones.
func (t *Tokenizer) unsafeNext(identifierHandling IdentifierHandling, maxTokenLength int32) Token {
	text := t.text
	end := t.end
	pos := t.Pos

	for pos < end {
		t.TokenPos = pos
		c := int32(text[pos])

		switch c {
		case '\r':
			pos++
			if pos < end && text[pos] == '\n' {
				pos++
			}
			continue
		case '\n', '\t', 0x0B, 0x0C, ' ':
			pos++
			continue

		case '!':
			pos++
			if maxTokenLength > 1 && pos < end && text[pos] == '=' {
				pos++
				if maxTokenLength > 2 && pos < end && text[pos] == '=' {
					t.Pos = pos + 1
					return TokenExclamationEqualsEquals
				}
				t.Pos = pos
				return TokenExclamationEquals
			}
			t.Pos = pos
			return TokenExclamation

		case '"', '\'':
			t.Pos = pos
			return TokenStringLiteral

		case '`':
			t.Pos = pos
			return TokenTemplateLiteral

		case '%':
			pos++
			if maxTokenLength > 1 && pos < end && text[pos] == '=' {
				t.Pos = pos + 1
				return TokenPercentEquals
			}
			t.Pos = pos
			return TokenPercent

		case '&':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '&' {
					t.Pos = pos + 1
					return TokenAmpersandAmpersand
				}
				if chr == '=' {
					t.Pos = pos + 1
					return TokenAmpersandEquals
				}
			}
			t.Pos = pos
			return TokenAmpersand

		case '(':
			t.Pos = pos + 1
			return TokenOpenParen

		case ')':
			t.Pos = pos + 1
			return TokenCloseParen

		case '*':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '=' {
					t.Pos = pos + 1
					return TokenAsteriskEquals
				}
				if chr == '*' {
					pos++
					if maxTokenLength > 2 && pos < end && text[pos] == '=' {
						t.Pos = pos + 1
						return TokenAsteriskAsteriskEquals
					}
					t.Pos = pos
					return TokenAsteriskAsterisk
				}
			}
			t.Pos = pos
			return TokenAsterisk

		case '+':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '+' {
					t.Pos = pos + 1
					return TokenPlusPlus
				}
				if chr == '=' {
					t.Pos = pos + 1
					return TokenPlusEquals
				}
			}
			t.Pos = pos
			return TokenPlus

		case ',':
			t.Pos = pos + 1
			return TokenComma

		case '-':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '-' {
					t.Pos = pos + 1
					return TokenMinusMinus
				}
				if chr == '=' {
					t.Pos = pos + 1
					return TokenMinusEquals
				}
			}
			t.Pos = pos
			return TokenMinus

		case '.':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := int32(text[pos])
				if util.IsDecimal(chr) {
					t.Pos = pos - 1
					return TokenFloatLiteral
				}
				if maxTokenLength > 2 && pos+1 < end && chr == '.' && text[pos+1] == '.' {
					t.Pos = pos + 2
					return TokenDotDotDot
				}
			}
			t.Pos = pos
			return TokenDot

		case '/':
			commentStartPos := pos
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '/' { // single-line comment
					commentKind := CommentKindLine
					if pos+1 < end && text[pos+1] == '/' {
						pos++
						commentKind = CommentKindTriple
					}
					for {
						pos++
						if pos >= end {
							break
						}
						if text[pos] == '\n' {
							pos++
							break
						}
					}
					if t.OnComment != nil {
						t.OnComment(commentKind, text[commentStartPos:pos], t.makeRange(int32(commentStartPos), pos))
					}
					continue // break from switch, continue outer loop
				}
				if chr == '*' { // multi-line comment
					closed := false
					for {
						pos++
						if pos >= end {
							break
						}
						if text[pos] == '*' && pos+1 < end && text[pos+1] == '/' {
							pos += 2
							closed = true
							break
						}
					}
					if !closed {
						t.Error(
							diagnostics.DiagnosticCode0Expected,
							t.makeRange(pos, -1), "*/", "", "",
						)
					} else if t.OnComment != nil {
						t.OnComment(CommentKindBlock, text[commentStartPos:pos], t.makeRange(int32(commentStartPos), pos))
					}
					continue
				}
				if chr == '=' {
					t.Pos = pos + 1
					return TokenSlashEquals
				}
			}
			t.Pos = pos
			return TokenSlash

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			t.Pos = pos
			if t.testInteger() {
				return TokenIntegerLiteral
			}
			return TokenFloatLiteral

		case ':':
			t.Pos = pos + 1
			return TokenColon

		case ';':
			t.Pos = pos + 1
			return TokenSemicolon

		case '<':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '<' {
					pos++
					if maxTokenLength > 2 && pos < end && text[pos] == '=' {
						t.Pos = pos + 1
						return TokenLessThanLessThanEquals
					}
					t.Pos = pos
					return TokenLessThanLessThan
				}
				if chr == '=' {
					t.Pos = pos + 1
					return TokenLessThanEquals
				}
			}
			t.Pos = pos
			return TokenLessThan

		case '=':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '=' {
					pos++
					if maxTokenLength > 2 && pos < end && text[pos] == '=' {
						t.Pos = pos + 1
						return TokenEqualsEqualsEquals
					}
					t.Pos = pos
					return TokenEqualsEquals
				}
				if chr == '>' {
					t.Pos = pos + 1
					return TokenEqualsGreaterThan
				}
			}
			t.Pos = pos
			return TokenEquals

		case '>':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '>' {
					pos++
					if maxTokenLength > 2 && pos < end {
						chr = text[pos]
						if chr == '>' {
							pos++
							if maxTokenLength > 3 && pos < end && text[pos] == '=' {
								t.Pos = pos + 1
								return TokenGreaterThanGreaterThanGreaterThanEquals
							}
							t.Pos = pos
							return TokenGreaterThanGreaterThanGreaterThan
						}
						if chr == '=' {
							t.Pos = pos + 1
							return TokenGreaterThanGreaterThanEquals
						}
					}
					t.Pos = pos
					return TokenGreaterThanGreaterThan
				}
				if chr == '=' {
					t.Pos = pos + 1
					return TokenGreaterThanEquals
				}
			}
			t.Pos = pos
			return TokenGreaterThan

		case '?':
			t.Pos = pos + 1
			return TokenQuestion

		case '[':
			t.Pos = pos + 1
			return TokenOpenBracket

		case ']':
			t.Pos = pos + 1
			return TokenCloseBracket

		case '^':
			pos++
			if maxTokenLength > 1 && pos < end && text[pos] == '=' {
				t.Pos = pos + 1
				return TokenCaretEquals
			}
			t.Pos = pos
			return TokenCaret

		case '{':
			t.Pos = pos + 1
			return TokenOpenBrace

		case '|':
			pos++
			if maxTokenLength > 1 && pos < end {
				chr := text[pos]
				if chr == '|' {
					t.Pos = pos + 1
					return TokenBarBar
				}
				if chr == '=' {
					t.Pos = pos + 1
					return TokenBarEquals
				}
			}
			t.Pos = pos
			return TokenBar

		case '}':
			t.Pos = pos + 1
			return TokenCloseBrace

		case '~':
			t.Pos = pos + 1
			return TokenTilde

		case '@':
			t.Pos = pos + 1
			return TokenAt

		default:
			// Unicode-aware from here on
			r, size := utf8.DecodeRuneInString(text[pos:])
			cp := int32(r)

			if util.IsIdentifierStart(cp) {
				posBefore := pos
				pos += int32(size)
				for pos < end {
					r, size = utf8.DecodeRuneInString(text[pos:])
					if !util.IsIdentifierPart(int32(r)) {
						break
					}
					pos += int32(size)
				}
				if identifierHandling != IdentifierHandlingAlways {
					maybeKeywordToken := TokenFromKeyword(text[posBefore:pos])
					if maybeKeywordToken != TokenInvalid &&
						!(identifierHandling == IdentifierHandlingPrefer && TokenIsAlsoIdentifier(maybeKeywordToken)) {
						t.Pos = pos
						return maybeKeywordToken
					}
				}
				t.Pos = posBefore
				return TokenIdentifier
			} else if util.IsWhiteSpace(cp) {
				pos++ // assume no supplementary whitespaces
				continue
			}

			start := pos
			pos += int32(size)
			t.Error(
				diagnostics.DiagnosticCodeInvalidCharacter,
				t.makeRange(int32(start), pos), "", "", "",
			)
			t.Pos = pos
			return TokenInvalid
		}
	}
	t.Pos = pos
	return TokenEndOfFile
}

// Peek returns the next token without consuming it.
func (t *Tokenizer) Peek(identifierHandling IdentifierHandling, maxCompoundLength int32) Token {
	nextToken := t.nextToken
	if nextToken < 0 {
		posBefore := t.Pos
		tokenBefore := t.Token
		tokenPosBefore := t.TokenPos
		for {
			nextToken = t.unsafeNext(identifierHandling, maxCompoundLength)
			if nextToken != TokenInvalid {
				break
			}
		}
		t.nextToken = nextToken
		t.nextTokenPos = t.TokenPos
		t.nextTokenOnNewLine = onNewLineUnknown
		t.Pos = posBefore
		t.Token = tokenBefore
		t.TokenPos = tokenPosBefore
	}
	return nextToken
}

// PeekOnNewLine returns true if the next token starts on a new line.
func (t *Tokenizer) PeekOnNewLine() bool {
	switch t.nextTokenOnNewLine {
	case onNewLineNo:
		return false
	case onNewLineYes:
		return true
	}
	t.Peek(IdentifierHandlingDefault, math.MaxInt32)
	text := t.text
	for pos := t.Pos; pos < t.nextTokenPos; pos++ {
		if util.IsLineBreak(int32(text[pos])) {
			t.nextTokenOnNewLine = onNewLineYes
			return true
		}
	}
	t.nextTokenOnNewLine = onNewLineNo
	return false
}

// SkipIdentifier attempts to skip an identifier token.
func (t *Tokenizer) SkipIdentifier(identifierHandling IdentifierHandling) bool {
	return t.Skip(TokenIdentifier, identifierHandling)
}

// Skip attempts to skip the specified token, returning true if found.
func (t *Tokenizer) Skip(token Token, identifierHandling IdentifierHandling) bool {
	posBefore := t.Pos
	tokenBefore := t.Token
	tokenPosBefore := t.TokenPos
	maxCompoundLength := int32(math.MaxInt32)
	if token == TokenGreaterThan {
		maxCompoundLength = 1
	}
	var nextToken Token
	for {
		nextToken = t.unsafeNext(identifierHandling, maxCompoundLength)
		if nextToken != TokenInvalid {
			break
		}
	}
	if nextToken == token {
		t.Token = token
		t.clearNextToken()
		return true
	}
	t.Pos = posBefore
	t.Token = tokenBefore
	t.TokenPos = tokenPosBefore
	return false
}

// Mark saves the current tokenizer state.
func (t *Tokenizer) Mark() *State {
	return &State{
		Pos:      t.Pos,
		Token:    t.Token,
		TokenPos: t.TokenPos,
	}
}

// Discard discards a previously saved state (no-op in Go, no pooling needed).
func (t *Tokenizer) Discard(state *State) {
	// No pooling needed in Go
}

// Reset restores the tokenizer to a previously saved state.
func (t *Tokenizer) Reset(state *State) {
	t.Pos = state.Pos
	t.Token = state.Token
	t.TokenPos = state.TokenPos
	t.clearNextToken()
}

func (t *Tokenizer) clearNextToken() {
	t.nextToken = -1
	t.nextTokenPos = 0
	t.nextTokenOnNewLine = onNewLineUnknown
}

// MakeRange creates a Range from the tokenizer's current state.
// If start < 0, uses tokenPos..pos. If end < 0, end = start.
func (t *Tokenizer) MakeRange(start, end int32) *diagnostics.Range {
	return t.makeRange(start, end)
}

func (t *Tokenizer) makeRange(start, end int32) *diagnostics.Range {
	if start < 0 {
		start = t.TokenPos
		end = t.Pos
	} else if end < 0 {
		end = start
	}
	rng := diagnostics.NewRange(start, end)
	rng.Source = t.source
	return rng
}

// ReadIdentifier reads an identifier string starting at the current position.
func (t *Tokenizer) ReadIdentifier() string {
	text := t.text
	end := t.end
	pos := t.Pos
	start := pos

	r, size := utf8.DecodeRuneInString(text[pos:])
	if !util.IsIdentifierStart(int32(r)) {
		panic("assertion failed: not at identifier start")
	}
	pos += int32(size)
	for pos < end {
		r, size = utf8.DecodeRuneInString(text[pos:])
		if !util.IsIdentifierPart(int32(r)) {
			break
		}
		pos += int32(size)
	}
	t.Pos = pos
	return text[start:pos]
}

// ReadString reads a string literal. If quote is 0, reads and consumes the opening quote.
func (t *Tokenizer) ReadString(quote int32, isTaggedTemplate bool) string {
	text := t.text
	end := t.end
	pos := t.Pos
	if quote == 0 {
		quote = int32(text[pos])
		pos++
	}
	start := pos
	t.ReadStringStart = start
	var result strings.Builder

	for {
		if pos >= end {
			result.WriteString(text[start:pos])
			t.Error(
				diagnostics.DiagnosticCodeUnterminatedStringLiteral,
				t.makeRange(start-1, end), "", "", "",
			)
			t.ReadStringEnd = end
			break
		}
		c := int32(text[pos])
		if c == quote {
			t.ReadStringEnd = pos
			result.WriteString(text[start:pos])
			pos++
			break
		}
		if c == '\\' {
			result.WriteString(text[start:pos])
			t.Pos = pos
			result.WriteString(t.readEscapeSequence(isTaggedTemplate))
			pos = t.Pos
			start = pos
			continue
		}
		if quote == '`' {
			if c == '$' && pos+1 < end && text[pos+1] == '{' {
				result.WriteString(text[start:pos])
				t.ReadStringEnd = pos
				t.Pos = pos + 2
				t.ReadingTemplateString = true
				return result.String()
			}
		} else if util.IsLineBreak(c) {
			result.WriteString(text[start:pos])
			t.Error(
				diagnostics.DiagnosticCodeUnterminatedStringLiteral,
				t.makeRange(start-1, pos), "", "", "",
			)
			t.ReadStringEnd = pos
			break
		}
		pos++
	}
	t.Pos = pos
	t.ReadingTemplateString = false
	return result.String()
}

func (t *Tokenizer) readEscapeSequence(isTaggedTemplate bool) string {
	start := t.Pos
	end := t.end
	t.Pos++
	if t.Pos >= end {
		t.Error(
			diagnostics.DiagnosticCodeUnexpectedEndOfText,
			t.makeRange(end, -1), "", "", "",
		)
		return ""
	}

	text := t.text
	c := text[t.Pos]
	t.Pos++

	switch c {
	case '0':
		if isTaggedTemplate && t.Pos < end && util.IsDecimal(int32(text[t.Pos])) {
			t.Pos++
			return text[start:t.Pos]
		}
		return "\x00"
	case 'b':
		return "\b"
	case 't':
		return "\t"
	case 'n':
		return "\n"
	case 'v':
		return "\v"
	case 'f':
		return "\f"
	case 'r':
		return "\r"
	case '\'':
		return "'"
	case '"':
		return "\""
	case 'u':
		if t.Pos < end && text[t.Pos] == '{' {
			t.Pos++
			startIfTagged := int32(-1)
			if isTaggedTemplate {
				startIfTagged = start
			}
			return t.readExtendedUnicodeEscape(startIfTagged)
		}
		startIfTagged := int32(-1)
		if isTaggedTemplate {
			startIfTagged = start
		}
		return t.readUnicodeEscape(startIfTagged)
	case 'x':
		startIfTagged := int32(-1)
		if isTaggedTemplate {
			startIfTagged = start
		}
		return t.readHexadecimalEscape(2, startIfTagged)
	case '\r':
		if t.Pos < end && text[t.Pos] == '\n' {
			t.Pos++
		}
		return ""
	case '\n':
		return ""
	default:
		// Check for LineSeparator (U+2028) and ParagraphSeparator (U+2029) in UTF-8
		if c == 0xE2 && t.Pos+1 < end && text[t.Pos-1] == 0xE2 {
			// These are multi-byte UTF-8 sequences, handle via rune decoding
			r, _ := utf8.DecodeRuneInString(text[t.Pos-1:])
			if r == 0x2028 || r == 0x2029 {
				// Already advanced past first byte, skip remaining bytes
				t.Pos += 2
				return ""
			}
		}
		// For any other character, return it as-is
		// Re-decode since we advanced past a byte
		t.Pos-- // back up
		r, size := utf8.DecodeRuneInString(text[t.Pos:])
		t.Pos += int32(size)
		return string(r)
	}
}

// ReadRegexpPattern reads a regular expression pattern.
func (t *Tokenizer) ReadRegexpPattern() string {
	text := t.text
	start := t.Pos
	end := t.end
	escaped := false

	for {
		if t.Pos >= end {
			t.Error(
				diagnostics.DiagnosticCodeUnterminatedRegularExpressionLiteral,
				t.makeRange(start, end), "", "", "",
			)
			break
		}
		if text[t.Pos] == '\\' {
			t.Pos++
			escaped = true
			continue
		}
		c := text[t.Pos]
		if !escaped && c == '/' {
			break
		}
		if util.IsLineBreak(int32(c)) {
			t.Error(
				diagnostics.DiagnosticCodeUnterminatedRegularExpressionLiteral,
				t.makeRange(start, t.Pos), "", "", "",
			)
			break
		}
		t.Pos++
		escaped = false
	}
	return text[start:t.Pos]
}

// ReadRegexpFlags reads regular expression flags.
func (t *Tokenizer) ReadRegexpFlags() string {
	text := t.text
	start := t.Pos
	end := t.end
	flags := int32(0)

	for t.Pos < end {
		c := int32(text[t.Pos])
		if !util.IsIdentifierPart(c) {
			break
		}
		t.Pos++

		switch c {
		case 'g':
			if flags&1 != 0 {
				flags = -1
			} else {
				flags |= 1
			}
		case 'i':
			if flags&2 != 0 {
				flags = -1
			} else {
				flags |= 2
			}
		case 'm':
			if flags&4 != 0 {
				flags = -1
			} else {
				flags |= 4
			}
		default:
			flags = -1
		}
	}
	if flags == -1 {
		t.Error(
			diagnostics.DiagnosticCodeInvalidRegularExpressionFlags,
			t.makeRange(start, t.Pos), "", "", "",
		)
	}
	return text[start:t.Pos]
}

func (t *Tokenizer) testInteger() bool {
	text := t.text
	pos := t.Pos
	end := t.end

	if pos+1 < end && text[pos] == '0' {
		if pos+2 < end {
			switch text[pos+2] | 32 {
			case 'x', 'b', 'o':
				return true
			}
		}
	}
	for pos < end {
		c := int32(text[pos])
		if c == '.' || (c|32) == 'e' {
			return false
		}
		if c != '_' && (c < '0' || c > '9') {
			break
		}
		pos++
	}
	return true
}

// ReadInteger reads an integer literal and returns its value.
func (t *Tokenizer) ReadInteger() int64 {
	text := t.text
	pos := t.Pos
	if pos+2 < t.end && text[pos] == '0' {
		switch text[pos+1] | 32 {
		case 'x':
			t.Pos = pos + 2
			return t.readHexInteger()
		case 'b':
			t.Pos = pos + 2
			return t.readBinaryInteger()
		case 'o':
			t.Pos = pos + 2
			return t.readOctalInteger()
		}
		if util.IsOctal(int32(text[pos+1])) {
			start := pos
			t.Pos = pos + 1
			value := t.readOctalInteger()
			t.Error(
				diagnostics.DiagnosticCodeOctalLiteralsAreNotAllowedInStrictMode,
				t.makeRange(start, t.Pos), "", "", "",
			)
			return value
		}
	}
	return t.readDecimalInteger()
}

func (t *Tokenizer) readHexInteger() int64 {
	text := t.text
	pos := t.Pos
	end := t.end
	start := pos
	sepEnd := start
	value := int64(0)
	nextValue := value
	overflowOccurred := false

	for pos < end {
		c := int32(text[pos])
		if util.IsDecimal(c) {
			nextValue = (value << 4) + int64(c-'0')
		} else if util.IsHexBase(c) {
			nextValue = (value << 4) + int64((c|32)+10-'a')
		} else if c == '_' {
			if sepEnd == pos {
				code := diagnostics.DiagnosticCodeMultipleConsecutiveNumericSeparatorsAreNotPermitted
				if sepEnd == start {
					code = diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere
				}
				t.Error(code, t.makeRange(pos, -1), "", "", "")
			}
			sepEnd = pos + 1
		} else {
			break
		}
		if uint64(value) > uint64(nextValue) {
			overflowOccurred = true
		}
		value = nextValue
		pos++
	}

	if pos == start {
		t.Error(diagnostics.DiagnosticCodeHexadecimalDigitExpected, t.makeRange(start, -1), "", "", "")
	} else if sepEnd == pos {
		t.Error(diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere, t.makeRange(sepEnd-1, -1), "", "", "")
	}
	if overflowOccurred {
		t.Error(
			diagnostics.DiagnosticCodeLiteral0DoesNotFitIntoI64OrU64Types,
			t.makeRange(start-2, pos),
			text[start-2:pos], "", "",
		)
	}
	t.Pos = pos
	return value
}

func (t *Tokenizer) readDecimalInteger() int64 {
	text := t.text
	pos := t.Pos
	end := t.end
	start := pos
	sepEnd := start
	value := int64(0)
	nextValue := value
	overflowOccurred := false

	for pos < end {
		c := int32(text[pos])
		if util.IsDecimal(c) {
			nextValue = value*10 + int64(c-'0')
		} else if c == '_' {
			if sepEnd == pos {
				code := diagnostics.DiagnosticCodeMultipleConsecutiveNumericSeparatorsAreNotPermitted
				if sepEnd == start {
					code = diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere
				}
				t.Error(code, t.makeRange(pos, -1), "", "", "")
			} else if pos-1 == start && text[pos-1] == '0' {
				t.Error(diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere, t.makeRange(pos, -1), "", "", "")
			}
			sepEnd = pos + 1
		} else {
			break
		}
		if uint64(value) > uint64(nextValue) {
			overflowOccurred = true
		}
		value = nextValue
		pos++
	}

	if pos == start {
		t.Error(diagnostics.DiagnosticCodeDigitExpected, t.makeRange(start, -1), "", "", "")
	} else if sepEnd == pos {
		t.Error(diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere, t.makeRange(sepEnd-1, -1), "", "", "")
	} else if overflowOccurred {
		t.Error(
			diagnostics.DiagnosticCodeLiteral0DoesNotFitIntoI64OrU64Types,
			t.makeRange(start, pos),
			text[start:pos], "", "",
		)
	}
	t.Pos = pos
	return value
}

func (t *Tokenizer) readOctalInteger() int64 {
	text := t.text
	pos := t.Pos
	end := t.end
	start := pos
	sepEnd := start
	value := int64(0)
	nextValue := value
	overflowOccurred := false

	for pos < end {
		c := int32(text[pos])
		if util.IsOctal(c) {
			nextValue = (value << 3) + int64(c-'0')
		} else if c == '_' {
			if sepEnd == pos {
				code := diagnostics.DiagnosticCodeMultipleConsecutiveNumericSeparatorsAreNotPermitted
				if sepEnd == start {
					code = diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere
				}
				t.Error(code, t.makeRange(pos, -1), "", "", "")
			}
			sepEnd = pos + 1
		} else {
			break
		}
		if uint64(value) > uint64(nextValue) {
			overflowOccurred = true
		}
		value = nextValue
		pos++
	}

	if pos == start {
		t.Error(diagnostics.DiagnosticCodeOctalDigitExpected, t.makeRange(start, -1), "", "", "")
	} else if sepEnd == pos {
		t.Error(diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere, t.makeRange(sepEnd-1, -1), "", "", "")
	} else if overflowOccurred {
		t.Error(
			diagnostics.DiagnosticCodeLiteral0DoesNotFitIntoI64OrU64Types,
			t.makeRange(start-2, pos),
			text[start-2:pos], "", "",
		)
	}
	t.Pos = pos
	return value
}

func (t *Tokenizer) readBinaryInteger() int64 {
	text := t.text
	pos := t.Pos
	end := t.end
	start := pos
	sepEnd := start
	value := int64(0)
	nextValue := value
	overflowOccurred := false

	for pos < end {
		c := int32(text[pos])
		if c == '0' {
			nextValue = value << 1
		} else if c == '1' {
			nextValue = (value << 1) | 1
		} else if c == '_' {
			if sepEnd == pos {
				code := diagnostics.DiagnosticCodeMultipleConsecutiveNumericSeparatorsAreNotPermitted
				if sepEnd == start {
					code = diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere
				}
				t.Error(code, t.makeRange(pos, -1), "", "", "")
			}
			sepEnd = pos + 1
		} else {
			break
		}
		if value > nextValue {
			overflowOccurred = true
		}
		value = nextValue
		pos++
	}

	if pos == start {
		t.Error(diagnostics.DiagnosticCodeBinaryDigitExpected, t.makeRange(start, -1), "", "", "")
	} else if sepEnd == pos {
		t.Error(diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere, t.makeRange(sepEnd-1, -1), "", "", "")
	} else if overflowOccurred {
		t.Error(
			diagnostics.DiagnosticCodeLiteral0DoesNotFitIntoI64OrU64Types,
			t.makeRange(start-2, pos),
			text[start-2:pos], "", "",
		)
	}
	t.Pos = pos
	return value
}

// ReadFloat reads a floating-point literal.
func (t *Tokenizer) ReadFloat() float64 {
	return t.readDecimalFloat()
}

func (t *Tokenizer) readDecimalFloat() float64 {
	text := t.text
	end := t.end
	start := t.Pos
	sepCount := t.readDecimalFloatPartial(false)
	if t.Pos < end && text[t.Pos] == '.' {
		t.Pos++
		sepCount += t.readDecimalFloatPartial(true)
	}
	if t.Pos < end {
		c := text[t.Pos]
		if (c | 32) == 'e' {
			t.Pos++
			if t.Pos < end {
				c = text[t.Pos]
				if c == '-' || c == '+' {
					if t.Pos+1 < end && util.IsDecimal(int32(text[t.Pos+1])) {
						t.Pos++
					}
				}
			}
			sepCount += t.readDecimalFloatPartial(true)
		}
	}
	result := text[start:t.Pos]
	if sepCount > 0 {
		result = strings.ReplaceAll(result, "_", "")
	}
	f, _ := strconv.ParseFloat(result, 64)
	return f
}

func (t *Tokenizer) readDecimalFloatPartial(allowLeadingZeroSep bool) int32 {
	text := t.text
	pos := t.Pos
	start := pos
	end := t.end
	sepEnd := start
	sepCount := int32(0)

	for pos < end {
		c := int32(text[pos])
		if c == '_' {
			if sepEnd == pos {
				code := diagnostics.DiagnosticCodeMultipleConsecutiveNumericSeparatorsAreNotPermitted
				if sepEnd == start {
					code = diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere
				}
				t.Error(code, t.makeRange(pos, -1), "", "", "")
			} else if !allowLeadingZeroSep && pos-1 == start && text[pos-1] == '0' {
				t.Error(diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere, t.makeRange(pos, -1), "", "", "")
			}
			sepEnd = pos + 1
			sepCount++
		} else if !util.IsDecimal(c) {
			break
		}
		pos++
	}

	if pos != start && sepEnd == pos {
		t.Error(diagnostics.DiagnosticCodeNumericSeparatorsAreNotAllowedHere, t.makeRange(sepEnd-1, -1), "", "", "")
	}

	t.Pos = pos
	return sepCount
}

func (t *Tokenizer) readHexadecimalEscape(remain int32, startIfTaggedTemplate int32) string {
	value := int32(0)
	text := t.text
	pos := t.Pos
	end := t.end

	for pos < end {
		c := int32(text[pos])
		pos++
		if util.IsDecimal(c) {
			value = (value << 4) + c - '0'
		} else if util.IsHexBase(c) {
			value = (value << 4) + (c | 32) + 10 - 'a'
		} else if startIfTaggedTemplate >= 0 {
			pos--
			t.Pos = pos
			return text[startIfTaggedTemplate:pos]
		} else {
			t.Pos = pos
			t.Error(
				diagnostics.DiagnosticCodeHexadecimalDigitExpected,
				t.makeRange(pos-1, pos), "", "", "",
			)
			return ""
		}
		remain--
		if remain == 0 {
			break
		}
	}

	if remain > 0 {
		t.Pos = pos
		if startIfTaggedTemplate >= 0 {
			return text[startIfTaggedTemplate:pos]
		}
		t.Error(
			diagnostics.DiagnosticCodeUnexpectedEndOfText,
			t.makeRange(pos, -1), "", "", "",
		)
		return ""
	}
	t.Pos = pos
	return string(rune(value))
}

// CheckForIdentifierStartAfterNumericLiteral emits a diagnostic if an identifier follows a number.
func (t *Tokenizer) CheckForIdentifierStartAfterNumericLiteral() {
	pos := t.Pos
	if pos < t.end && util.IsIdentifierStart(int32(t.text[pos])) {
		t.Error(
			diagnostics.DiagnosticCodeAnIdentifierOrKeywordCannotImmediatelyFollowANumericLiteral,
			t.makeRange(pos, -1), "", "", "",
		)
	}
}

func (t *Tokenizer) readUnicodeEscape(startIfTaggedTemplate int32) string {
	return t.readHexadecimalEscape(4, startIfTaggedTemplate)
}

func (t *Tokenizer) readExtendedUnicodeEscape(startIfTaggedTemplate int32) string {
	start := t.Pos
	value := t.readHexInteger()
	value32 := int32(value)
	invalid := false

	if value>>32 != 0 {
		panic("assertion failed: extended unicode high bits")
	}
	if value32 > 0x10FFFF {
		if startIfTaggedTemplate == -1 {
			t.Error(
				diagnostics.DiagnosticCodeAnExtendedUnicodeEscapeValueMustBeBetween0x0And0x10ffffInclusive,
				t.makeRange(start, t.Pos), "", "", "",
			)
		}
		invalid = true
	}

	end := t.end
	text := t.text
	if t.Pos >= end {
		if startIfTaggedTemplate == -1 {
			t.Error(
				diagnostics.DiagnosticCodeUnexpectedEndOfText,
				t.makeRange(start, end), "", "", "",
			)
		}
		invalid = true
	} else if text[t.Pos] == '}' {
		t.Pos++
	} else {
		if startIfTaggedTemplate == -1 {
			t.Error(
				diagnostics.DiagnosticCodeUnterminatedUnicodeEscapeSequence,
				t.makeRange(start, t.Pos), "", "", "",
			)
		}
		invalid = true
	}

	if invalid {
		if startIfTaggedTemplate >= 0 {
			return text[startIfTaggedTemplate:t.Pos]
		}
		return ""
	}
	return string(rune(value32))
}

// Source returns the tokenizer's source.
func (t *Tokenizer) Source() diagnostics.Source {
	return t.source
}

// NextTokenPos returns the position of the peeked next token.
// Only valid after a call to Peek.
func (t *Tokenizer) NextTokenPos() int32 {
	return t.nextTokenPos
}

// State represents a saved tokenizer state for mark/reset.
type State struct {
	Pos      int32
	Token    Token
	TokenPos int32
}
