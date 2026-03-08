// Ported from: packages/ai/src/generate-text/extract-text-content.ts
package generatetext

import "strings"

// ExtractTextContent extracts text from language model content parts.
// Returns empty string if no text parts are found (Go equivalent of TS returning undefined).
func ExtractTextContent(content []LanguageModelV4Content) string {
	var parts []string
	for _, c := range content {
		if c.Type == "text" {
			parts = append(parts, c.Text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "")
}

// HasTextContent returns true if text content exists.
func HasTextContent(content []LanguageModelV4Content) bool {
	return ExtractTextContent(content) != ""
}
