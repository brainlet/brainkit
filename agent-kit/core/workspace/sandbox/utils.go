// Ported from: packages/core/src/workspace/sandbox/utils.ts
package sandbox

import (
	"regexp"
)

// safeCharsRegex matches strings containing only safe shell characters.
var safeCharsRegex = regexp.MustCompile(`^[a-zA-Z0-9._\-/=:@]+$`)

// ShellQuote shell-quotes an argument for safe interpolation into a shell command string.
// Safe characters (alphanumeric, `.`, `_`, `-`, `/`, `=`, `:`, `@`) pass through.
// Everything else is wrapped in single quotes with embedded quotes escaped.
func ShellQuote(arg string) string {
	if safeCharsRegex.MatchString(arg) {
		return arg
	}
	// Wrap in single quotes, escaping embedded single quotes
	escaped := ""
	for _, ch := range arg {
		if ch == '\'' {
			escaped += "'\\''"
		} else {
			escaped += string(ch)
		}
	}
	return "'" + escaped + "'"
}
