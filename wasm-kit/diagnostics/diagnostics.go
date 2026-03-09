package diagnostics

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/util"
)

// Source is an interface satisfied by the concrete ast.Source type.
// It is defined here to break the circular dependency between
// diagnostics (Range references Source) and ast (nodes reference Range).
type Source interface {
	// SourceText returns the full source text.
	SourceText() string
	// SourceNormalizedPath returns the normalized file path.
	SourceNormalizedPath() string
	// LineAt determines the line number at the specified position. Starts at 1.
	LineAt(pos int32) int32
	// ColumnAt returns the column number at the last position queried with LineAt. Starts at 1.
	ColumnAt() int32
}

// DiagnosticCategory indicates the category of a DiagnosticMessage.
type DiagnosticCategory int32

const (
	DiagnosticCategoryPedantic DiagnosticCategory = iota
	DiagnosticCategoryInfo
	DiagnosticCategoryWarning
	DiagnosticCategoryError
)

// DiagnosticCategoryToString returns the string representation of the given category.
func DiagnosticCategoryToString(category DiagnosticCategory) string {
	switch category {
	case DiagnosticCategoryPedantic:
		return "PEDANTIC"
	case DiagnosticCategoryInfo:
		return "INFO"
	case DiagnosticCategoryWarning:
		return "WARNING"
	case DiagnosticCategoryError:
		return "ERROR"
	default:
		panic("invalid diagnostic category")
	}
}

// DiagnosticCategoryToColor returns the ANSI escape sequence for the given category.
func DiagnosticCategoryToColor(category DiagnosticCategory) string {
	switch category {
	case DiagnosticCategoryPedantic:
		return util.ColorMagenta
	case DiagnosticCategoryInfo:
		return util.ColorCyan
	case DiagnosticCategoryWarning:
		return util.ColorYellow
	case DiagnosticCategoryError:
		return util.ColorRed
	default:
		panic("invalid diagnostic category")
	}
}

// Range represents a range within a source file.
type Range struct {
	Start  int32
	End    int32
	Source Source
}

// NewRange creates a new Range with the given start and end positions.
func NewRange(start, end int32) *Range {
	return &Range{Start: start, End: end}
}

// Join creates a range spanning from the minimum start to the maximum end of two ranges.
func JoinRanges(a, b *Range) *Range {
	if a.Source != b.Source {
		panic("source mismatch")
	}
	start := a.Start
	if b.Start < start {
		start = b.Start
	}
	end := a.End
	if b.End > end {
		end = b.End
	}
	r := &Range{Start: start, End: end}
	r.Source = a.Source
	return r
}

// Equals tests if this range equals the specified range.
func (r *Range) Equals(other *Range) bool {
	return r.Source == other.Source &&
		r.Start == other.Start &&
		r.End == other.End
}

// AtStart returns a zero-width range at the start position.
func (r *Range) AtStart() *Range {
	rng := &Range{Start: r.Start, End: r.Start}
	rng.Source = r.Source
	return rng
}

// AtEnd returns a zero-width range at the end position.
func (r *Range) AtEnd() *Range {
	rng := &Range{Start: r.End, End: r.End}
	rng.Source = r.Source
	return rng
}

// String returns the source text covered by this range.
func (r *Range) String() string {
	text := r.Source.SourceText()
	return text[r.Start:r.End]
}

// DiagnosticMessage represents a diagnostic message.
type DiagnosticMessage struct {
	Code         int32
	Category     DiagnosticCategory
	Message      string
	Range        *Range
	RelatedRange *Range
}

// NewDiagnosticMessage creates a new diagnostic message of the specified category.
func NewDiagnosticMessage(
	code DiagnosticCode,
	category DiagnosticCategory,
	arg0 string,
	arg1 string,
	arg2 string,
) *DiagnosticMessage {
	message := DiagnosticCodeToString(code)
	if arg0 != "" {
		message = strings.Replace(message, "{0}", arg0, 1)
	}
	if arg1 != "" {
		message = strings.Replace(message, "{1}", arg1, 1)
	}
	if arg2 != "" {
		message = strings.Replace(message, "{2}", arg2, 1)
	}
	return &DiagnosticMessage{
		Code:     int32(code),
		Category: category,
		Message:  message,
	}
}

// Equals tests if this message equals the specified message.
func (m *DiagnosticMessage) Equals(other *DiagnosticMessage) bool {
	if m.Code != other.Code {
		return false
	}
	if m.Range != nil {
		if other.Range == nil || !m.Range.Equals(other.Range) {
			return false
		}
	} else if other.Range != nil {
		return false
	}
	if m.RelatedRange != nil {
		if other.RelatedRange == nil || !m.RelatedRange.Equals(other.RelatedRange) {
			return false
		}
	} else if other.RelatedRange != nil {
		return false
	}
	return m.Message == other.Message
}

// WithRange adds a source range to this message and returns it.
func (m *DiagnosticMessage) WithRange(rng *Range) *DiagnosticMessage {
	m.Range = rng
	return m
}

// WithRelatedRange adds a related source range to this message and returns it.
func (m *DiagnosticMessage) WithRelatedRange(rng *Range) *DiagnosticMessage {
	m.RelatedRange = rng
	return m
}

// String converts this message to a string.
func (m *DiagnosticMessage) String() string {
	category := DiagnosticCategoryToString(m.Category)
	if m.Range != nil {
		source := m.Range.Source
		path := source.SourceNormalizedPath()
		line := source.LineAt(m.Range.Start)
		column := source.ColumnAt()
		length := m.Range.End - m.Range.Start
		return fmt.Sprintf(`%s %d: "%s" in %s(%d,%d+%d)`, category, m.Code, m.Message, path, line, column, length)
	}
	return fmt.Sprintf("%s %d: %s", category, m.Code, m.Message)
}

// FormatDiagnosticMessage formats a diagnostic message, optionally with terminal colors and source context.
func FormatDiagnosticMessage(message *DiagnosticMessage, useColors bool, showContext bool) string {
	wasColorsEnabled := util.SetColorsEnabled(useColors)

	var sb strings.Builder

	if util.IsColorsEnabled() {
		sb.WriteString(DiagnosticCategoryToColor(message.Category))
	}
	sb.WriteString(DiagnosticCategoryToString(message.Category))
	if util.IsColorsEnabled() {
		sb.WriteString(util.ColorReset)
	}
	if message.Code < 1000 {
		sb.WriteString(" AS")
	} else {
		sb.WriteString(" TS")
	}
	sb.WriteString(fmt.Sprintf("%d", message.Code))
	sb.WriteString(": ")
	sb.WriteString(message.Message)

	rng := message.Range
	if rng != nil {
		source := rng.Source
		relatedRange := message.RelatedRange
		var minLine int32
		if relatedRange != nil {
			ml1 := source.LineAt(rng.Start)
			ml2 := relatedRange.Source.LineAt(relatedRange.Start)
			if ml1 > ml2 {
				minLine = ml1
			} else {
				minLine = ml2
			}
		}

		if showContext {
			sb.WriteString("\n")
			sb.WriteString(formatDiagnosticContext(rng, minLine))
		} else {
			sb.WriteString("\n in ")
			sb.WriteString(source.SourceNormalizedPath())
		}
		sb.WriteString("(")
		sb.WriteString(fmt.Sprintf("%d", source.LineAt(rng.Start)))
		sb.WriteString(",")
		sb.WriteString(fmt.Sprintf("%d", source.ColumnAt()))
		sb.WriteString(")")

		if relatedRange != nil {
			relatedSource := relatedRange.Source
			if showContext {
				sb.WriteString("\n")
				sb.WriteString(formatDiagnosticContext(relatedRange, minLine))
			} else {
				sb.WriteString("\n in ")
				sb.WriteString(relatedSource.SourceNormalizedPath())
			}
			sb.WriteString("(")
			sb.WriteString(fmt.Sprintf("%d", relatedSource.LineAt(relatedRange.Start)))
			sb.WriteString(",")
			sb.WriteString(fmt.Sprintf("%d", relatedSource.ColumnAt()))
			sb.WriteString(")")
		}
	}

	util.SetColorsEnabled(wasColorsEnabled)
	return sb.String()
}

// formatDiagnosticContext formats the diagnostic context for the specified range.
func formatDiagnosticContext(rng *Range, minLine int32) string {
	source := rng.Source
	text := source.SourceText()
	length := int32(len(text))
	start := rng.Start
	end := start
	lineNumber := fmt.Sprintf("%d", source.LineAt(start))
	lineNumberLength := len(lineNumber)
	if minLine != 0 {
		minLineStr := fmt.Sprintf("%d", minLine)
		if len(minLineStr) > lineNumberLength {
			lineNumberLength = len(minLineStr)
		}
	}
	lineSpace := strings.Repeat(" ", lineNumberLength)

	// Find preceding line break
	for start > 0 && !util.IsLineBreak(int32(text[start-1])) {
		start--
	}
	// Skip leading whitespace
	for start < length && util.IsWhiteSpace(int32(text[start])) {
		start++
	}
	// Find next line break
	for end < length && !util.IsLineBreak(int32(text[end])) {
		end++
	}

	var sb strings.Builder
	sb.WriteString(lineSpace)
	sb.WriteString("  :\n ")
	sb.WriteString(strings.Repeat(" ", lineNumberLength-len(lineNumber)))
	sb.WriteString(lineNumber)
	sb.WriteString(" \u2502 ")
	sb.WriteString(strings.ReplaceAll(text[start:end], "\t", "  "))
	sb.WriteString("\n ")
	sb.WriteString(lineSpace)
	sb.WriteString(" \u2502 ")

	pos := start
	for pos < rng.Start {
		if text[pos] == '\t' {
			sb.WriteString("  ")
			pos += 2
		} else {
			sb.WriteString(" ")
			pos++
		}
	}

	if util.IsColorsEnabled() {
		sb.WriteString(util.ColorRed)
	}
	if rng.Start == rng.End {
		sb.WriteString("^")
	} else {
		for pos < rng.End {
			pos++
			if pos >= length {
				break
			}
			cc := int32(text[pos])
			if cc == '\t' {
				sb.WriteString("~~")
			} else if util.IsLineBreak(cc) {
				if pos == int32(rng.Start+1) {
					sb.WriteString("^")
				} else {
					sb.WriteString("~")
				}
				break
			} else {
				sb.WriteString("~")
			}
		}
	}
	if util.IsColorsEnabled() {
		sb.WriteString(util.ColorReset)
	}
	sb.WriteString("\n ")
	sb.WriteString(lineSpace)
	sb.WriteString(" \u2514\u2500 in ")
	sb.WriteString(source.SourceNormalizedPath())
	return sb.String()
}

// DiagnosticEmitter is the base for all types that emit diagnostics.
// It is embedded in Parser, Compiler, Resolver, and Program.
type DiagnosticEmitter struct {
	Diagnostics []*DiagnosticMessage
	seen        map[Source]map[int32][]*DiagnosticMessage
}

// NewDiagnosticEmitter creates a new DiagnosticEmitter, optionally sharing a diagnostics slice.
func NewDiagnosticEmitter(diagnostics []*DiagnosticMessage) DiagnosticEmitter {
	if diagnostics == nil {
		diagnostics = make([]*DiagnosticMessage, 0)
	}
	return DiagnosticEmitter{
		Diagnostics: diagnostics,
		seen:        make(map[Source]map[int32][]*DiagnosticMessage),
	}
}

// EmitDiagnostic emits a diagnostic message of the specified category.
func (e *DiagnosticEmitter) EmitDiagnostic(
	code DiagnosticCode,
	category DiagnosticCategory,
	rng *Range,
	relatedRange *Range,
	arg0 string,
	arg1 string,
	arg2 string,
) {
	message := NewDiagnosticMessage(code, category, arg0, arg1, arg2)
	if rng != nil {
		message = message.WithRange(rng)
	}
	if relatedRange != nil {
		message.RelatedRange = relatedRange
	}

	// Deduplicate diagnostics: same code+range+message should not be emitted twice.
	if rng != nil {
		seenInSource, ok := e.seen[rng.Source]
		if ok {
			seenAtPos, ok := seenInSource[rng.Start]
			if ok {
				for _, existing := range seenAtPos {
					if existing.Equals(message) {
						return
					}
				}
				seenInSource[rng.Start] = append(seenAtPos, message)
			} else {
				seenInSource[rng.Start] = []*DiagnosticMessage{message}
			}
		} else {
			seenInSource = make(map[int32][]*DiagnosticMessage)
			seenInSource[rng.Start] = []*DiagnosticMessage{message}
			e.seen[rng.Source] = seenInSource
		}
	}

	e.Diagnostics = append(e.Diagnostics, message)
}

// Pedantic emits an overly pedantic diagnostic message.
func (e *DiagnosticEmitter) Pedantic(code DiagnosticCode, rng *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryPedantic, rng, nil, arg0, arg1, arg2)
}

// PedanticRelated emits an overly pedantic diagnostic message with a related range.
func (e *DiagnosticEmitter) PedanticRelated(code DiagnosticCode, rng *Range, relatedRange *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryPedantic, rng, relatedRange, arg0, arg1, arg2)
}

// Info emits an informatory diagnostic message.
func (e *DiagnosticEmitter) Info(code DiagnosticCode, rng *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryInfo, rng, nil, arg0, arg1, arg2)
}

// InfoRelated emits an informatory diagnostic message with a related range.
func (e *DiagnosticEmitter) InfoRelated(code DiagnosticCode, rng *Range, relatedRange *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryInfo, rng, relatedRange, arg0, arg1, arg2)
}

// Warning emits a warning diagnostic message.
func (e *DiagnosticEmitter) Warning(code DiagnosticCode, rng *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryWarning, rng, nil, arg0, arg1, arg2)
}

// WarningRelated emits a warning diagnostic message with a related range.
func (e *DiagnosticEmitter) WarningRelated(code DiagnosticCode, rng *Range, relatedRange *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryWarning, rng, relatedRange, arg0, arg1, arg2)
}

// Error emits an error diagnostic message.
func (e *DiagnosticEmitter) Error(code DiagnosticCode, rng *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryError, rng, nil, arg0, arg1, arg2)
}

// ErrorRelated emits an error diagnostic message with a related range.
func (e *DiagnosticEmitter) ErrorRelated(code DiagnosticCode, rng *Range, relatedRange *Range, arg0, arg1, arg2 string) {
	e.EmitDiagnostic(code, DiagnosticCategoryError, rng, relatedRange, arg0, arg1, arg2)
}
