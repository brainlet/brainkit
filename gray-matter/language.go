package graymatter

import (
	"strings"
)

// LanguageResult represents the result of language detection.
type LanguageResult struct {
	Raw  string
	Name string
}

// Language detects the front-matter language from a string.
// It returns a LanguageResult with Raw (the language string as-is) and Name (trimmed).
// If no delimiter is found or the input is empty, returns empty strings.
func Language(str string, opts ...Options) LanguageResult {
	resolved := resolveOptions(opts...)
	delims := NormalizeDelimiters(resolved.Delimiters)
	open := delims[0]

	// If string doesn't start with delimiter, return empty
	if !strings.HasPrefix(str, open) {
		return LanguageResult{}
	}

	// Remove the opening delimiter
	str = str[len(open):]

	// Find first newline to get the first line
	newlineIdx := strings.IndexByte(str, '\n')
	var firstLine string
	if newlineIdx == -1 {
		firstLine = str
	} else {
		firstLine = str[:newlineIdx]
	}

	// Remove carriage return for Windows line endings
	firstLine = strings.TrimRight(firstLine, "\r")

	return LanguageResult{
		Raw:  firstLine,
		Name: strings.TrimSpace(firstLine),
	}
}
