// Ported from: packages/core/src/processors/processors/unicode-normalizer.ts
package concreteprocessors

import (
	"regexp"
	"strings"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"

	"golang.org/x/text/unicode/norm"
)

// ---------------------------------------------------------------------------
// UnicodeNormalizerOptions
// ---------------------------------------------------------------------------

// UnicodeNormalizerOptions holds configuration for UnicodeNormalizer.
type UnicodeNormalizerOptions struct {
	// StripControlChars removes control characters (except \t, \n, \r) when true.
	// Default: false.
	StripControlChars bool

	// PreserveEmojis keeps emojis intact when stripping control chars.
	// Default: true.
	PreserveEmojis bool

	// CollapseWhitespace collapses consecutive whitespace to single instances.
	// Default: true.
	CollapseWhitespace bool

	// Trim trims leading and trailing whitespace.
	// Default: true.
	Trim bool
}

// ---------------------------------------------------------------------------
// UnicodeNormalizer
// ---------------------------------------------------------------------------

// UnicodeNormalizer normalizes Unicode text by applying NFKC normalization,
// optionally stripping control characters, collapsing whitespace, and trimming.
type UnicodeNormalizer struct {
	processors.BaseProcessor
	options UnicodeNormalizerOptions
}

// NewUnicodeNormalizer creates a new UnicodeNormalizer with the given options.
// If opts is nil, defaults are used: StripControlChars=false, PreserveEmojis=true,
// CollapseWhitespace=true, Trim=true.
func NewUnicodeNormalizer(opts *UnicodeNormalizerOptions) *UnicodeNormalizer {
	o := UnicodeNormalizerOptions{
		StripControlChars:  false,
		PreserveEmojis:     true,
		CollapseWhitespace: true,
		Trim:               true,
	}
	if opts != nil {
		o.StripControlChars = opts.StripControlChars
		o.PreserveEmojis = opts.PreserveEmojis
		o.CollapseWhitespace = opts.CollapseWhitespace
		o.Trim = opts.Trim
	}
	return &UnicodeNormalizer{
		BaseProcessor: processors.NewBaseProcessor("unicode-normalizer", "Unicode Normalizer"),
		options:       o,
	}
}

// Regex patterns for control character stripping.
var (
	// Conservative: only remove specific problematic control chars while preserving emojis.
	controlCharsConservative = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F-\x9F]`)
	// Aggressive: remove all control characters except tab, newline, carriage return.
	controlCharsAggressive = regexp.MustCompile(`[^\x09\x0A\x0D\x20-\x7E\x{00A0}-\x{FFFF}]`)
	// Whitespace patterns.
	crlfPattern       = regexp.MustCompile(`\r\n`)
	crPattern         = regexp.MustCompile(`\r`)
	multiNewline      = regexp.MustCompile(`\n+`)
	multiSpaceOrTab   = regexp.MustCompile(`[ \t]+`)
)

// normalizeText applies all normalization steps to a string.
func (u *UnicodeNormalizer) normalizeText(text string) string {
	normalized := text

	// Step 1: Unicode normalization to NFKC (security-friendly).
	normalized = norm.NFKC.String(normalized)

	// Step 2: Strip control characters if enabled.
	if u.options.StripControlChars {
		if u.options.PreserveEmojis {
			normalized = controlCharsConservative.ReplaceAllString(normalized, "")
		} else {
			normalized = controlCharsAggressive.ReplaceAllString(normalized, "")
		}
	}

	// Step 3: Collapse whitespace if enabled.
	if u.options.CollapseWhitespace {
		// Normalize line endings.
		normalized = crlfPattern.ReplaceAllString(normalized, "\n")
		normalized = crPattern.ReplaceAllString(normalized, "\n")
		// Collapse multiple consecutive newlines.
		normalized = multiNewline.ReplaceAllString(normalized, "\n")
		// Collapse multiple consecutive spaces/tabs.
		normalized = multiSpaceOrTab.ReplaceAllString(normalized, " ")
	}

	// Step 4: Trim if enabled.
	if u.options.Trim {
		normalized = strings.TrimSpace(normalized)
	}

	return normalized
}

// ProcessInput normalizes Unicode text in all message parts.
// This is not a critical processor; errors are silently ignored
// and original messages returned.
func (u *UnicodeNormalizer) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	defer func() {
		// do nothing on panic, this isn't a critical processor
		recover() //nolint:errcheck
	}()

	result := make([]processors.MastraDBMessage, len(args.Messages))
	for i, message := range args.Messages {
		msg := message
		// Normalize parts.
		if len(msg.Content.Parts) > 0 {
			parts := make([]processors.MastraMessagePart, len(msg.Content.Parts))
			for j, part := range msg.Content.Parts {
				p := part
				if p.Type == "text" && p.Text != "" {
					p.Text = u.normalizeText(p.Text)
				}
				parts[j] = p
			}
			msg.Content.Parts = parts
		}
		// Normalize content string.
		if msg.Content.Content != "" {
			msg.Content.Content = u.normalizeText(msg.Content.Content)
		}
		result[i] = msg
	}
	return result, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (u *UnicodeNormalizer) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (u *UnicodeNormalizer) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (u *UnicodeNormalizer) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (u *UnicodeNormalizer) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}
