package graymatter

import "strings"

// Test checks if the input string starts with a frontmatter delimiter.
// Returns true if frontmatter is present, false otherwise.
func Test(str string, opts ...Options) bool {
	if str == "" {
		return false
	}

	resolved := resolveOptions(opts...)
	delims := NormalizeDelimiters(resolved.Delimiters)
	open := delims[0]

	return strings.HasPrefix(str, open)
}
