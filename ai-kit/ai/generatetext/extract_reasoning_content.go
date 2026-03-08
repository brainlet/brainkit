// Ported from: packages/ai/src/generate-text/extract-reasoning-content.ts
package generatetext

import "strings"

// ExtractReasoningContent extracts reasoning text from language model content parts.
// Returns empty string if no reasoning parts are found (Go equivalent of TS returning undefined).
func ExtractReasoningContent(content []LanguageModelV4Content) string {
	var parts []string
	for _, c := range content {
		if c.Type == "reasoning" {
			parts = append(parts, c.Text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

// HasReasoningContent returns true if reasoning content exists.
func HasReasoningContent(content []LanguageModelV4Content) bool {
	return ExtractReasoningContent(content) != ""
}
