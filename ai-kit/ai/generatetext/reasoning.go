// Ported from: packages/ai/src/generate-text/reasoning.ts
package generatetext

import "strings"

// AsReasoningText joins reasoning parts into a single string.
// Returns empty string if there are no reasoning parts (Go equivalent of TS returning undefined).
func AsReasoningText(reasoningParts []ReasoningPart) string {
	var texts []string
	for _, part := range reasoningParts {
		texts = append(texts, part.Text)
	}
	return strings.Join(texts, "")
}

// HasReasoningText returns true if there is non-empty reasoning text.
func HasReasoningText(reasoningParts []ReasoningPart) bool {
	return AsReasoningText(reasoningParts) != ""
}
