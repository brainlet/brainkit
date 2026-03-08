package graymatter

import "strings"

// LanguageResult mirrors gray-matter's return value for language detection.
type LanguageResult struct {
	Raw  string
	Name string
}

// Language detects the language line immediately after the opening delimiter.
// When called on a string without a leading delimiter it simply returns the first line.
func Language(str string, opts ...Options) LanguageResult {
	resolved := resolveOptions(opts...)
	open := NormalizeDelimiters(resolved.Delimiters)[0]

	if Test(str, resolved) {
		str = str[len(open):]
	}

	line := str
	if idx := strings.IndexByte(str, '\n'); idx != -1 {
		line = str[:idx]
	}

	line = strings.TrimRight(line, "\r")
	return LanguageResult{
		Raw:  line,
		Name: strings.TrimSpace(line),
	}
}
