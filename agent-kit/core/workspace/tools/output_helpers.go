// Ported from: packages/core/src/workspace/tools/output-helpers.ts
package tools

import (
	"regexp"
	"strconv"
	"strings"
)

// =============================================================================
// Constants
// =============================================================================

// DefaultTailLines is the default number of tail lines for command output.
const DefaultTailLines = 200

// defaultMaxOutputTokens is the default token limit for tool output.
const defaultMaxOutputTokens = 16000

// approximateCharsPerToken is the rough ratio for token estimation.
const approximateCharsPerToken = 4

// =============================================================================
// ANSI Stripping
// =============================================================================

// ansiRegex matches ANSI escape sequences.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b\[.*?[@-~]`)

// StripAnsi removes ANSI escape sequences from a string.
func StripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// =============================================================================
// Tail / Token Limit
// =============================================================================

// ApplyTail keeps only the last N lines of output.
// If n is 0 or negative, the full output is returned.
func ApplyTail(output string, n int) string {
	if n <= 0 {
		return output
	}

	lines := strings.Split(output, "\n")
	if len(lines) <= n {
		return output
	}

	skipped := len(lines) - n
	tail := lines[len(lines)-n:]
	return "[" + strconv.Itoa(skipped) + " lines above omitted]\n" + strings.Join(tail, "\n")
}

// ApplyTokenLimit truncates output to fit within a token budget.
// The from parameter controls where truncation happens: "end", "start", or "sandwich".
// Returns the (possibly truncated) output.
func ApplyTokenLimit(output string, maxTokens *int, from string) string {
	limit := defaultMaxOutputTokens
	if maxTokens != nil {
		limit = *maxTokens
	}
	if limit <= 0 {
		return output
	}

	maxChars := limit * approximateCharsPerToken
	if len(output) <= maxChars {
		return output
	}

	switch from {
	case "start":
		// Keep the end, truncate the start
		kept := output[len(output)-maxChars:]
		return "[output truncated — showing last portion]\n" + kept

	case "sandwich":
		return ApplyTokenLimitSandwich(output, maxChars)

	default: // "end"
		// Keep the start, truncate the end
		kept := output[:maxChars]
		return kept + "\n[output truncated]"
	}
}

// ApplyTokenLimitSandwich keeps the first and last portions of output,
// removing the middle. Useful for command output where both the beginning
// (compilation errors) and end (final status) are important.
func ApplyTokenLimitSandwich(output string, maxChars int) string {
	if len(output) <= maxChars {
		return output
	}

	// Split roughly 40% head / 60% tail to favor recent output
	headBudget := maxChars * 40 / 100
	tailBudget := maxChars - headBudget

	head := output[:headBudget]
	tail := output[len(output)-tailBudget:]
	omitted := len(output) - headBudget - tailBudget

	return head + "\n\n[" + strconv.Itoa(omitted) + " chars omitted]\n\n" + tail
}

// TruncateOutput applies both tail and token limit to command output.
// Strips ANSI codes, applies tail, then applies token limit.
func TruncateOutput(output string, tail *int, tokenLimit *int, from string) string {
	if output == "" {
		return ""
	}

	// Strip ANSI escape sequences
	cleaned := StripAnsi(output)

	// Apply tail limit
	tailLines := DefaultTailLines
	if tail != nil {
		tailLines = *tail
	}
	if tailLines > 0 {
		cleaned = ApplyTail(cleaned, tailLines)
	}

	// Apply token limit
	return ApplyTokenLimit(cleaned, tokenLimit, from)
}

// SandboxToModelOutput is a helper for formatting sandbox tool output.
// It strips ANSI codes from the output string.
func SandboxToModelOutput(output interface{}) interface{} {
	if s, ok := output.(string); ok {
		return StripAnsi(s)
	}
	return output
}
