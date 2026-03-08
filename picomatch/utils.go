package picomatch

// Ported from: picomatch/lib/utils.js

import (
	"runtime"
	"strings"

	"github.com/dlclark/regexp2"
)

// --- Compiled regexes for utils ---

var (
	reBackslash          = regexp2.MustCompile(regexBackslashPattern, regexp2.None)
	reRemoveBackslash    = regexp2.MustCompile(regexRemoveBackslashPattern, regexp2.None)
	reSpecialChars       = regexp2.MustCompile(regexSpecialCharsPattern, regexp2.None)
	reSpecialCharsGlobal = regexp2.MustCompile(regexSpecialCharsGlobalPattern, regexp2.None)
)

// isObject checks if a value is a non-nil object-like thing.
// JS source: utils.js line 11
// In Go this is used to check if a ParseState is populated.

// hasRegexChars returns true if str contains special regex characters.
// JS source: utils.js line 12
func hasRegexChars(str string) bool {
	ok, _ := reSpecialChars.MatchString(str)
	return ok
}

// isRegexChar returns true if str is a single special regex character.
// JS source: utils.js line 13
func isRegexChar(str string) bool {
	return len(str) == 1 && hasRegexChars(str)
}

// escapeRegex escapes special regex characters in str.
// JS source: utils.js line 14
func escapeRegex(str string) string {
	result, _ := reSpecialCharsGlobal.Replace(str, `\$1`, -1, -1)
	return result
}

// toPosixSlashes replaces backslashes with forward slashes.
// JS source: utils.js line 15
func toPosixSlashes(str string) string {
	result, _ := reBackslash.Replace(str, "/", -1, -1)
	return result
}

// isWindows returns true if running on Windows.
// JS source: utils.js lines 17-28
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// removeBackslashes removes escape backslashes from str.
// JS source: utils.js lines 30-34
// The JS regex: /(?:\[.*?[^\\]\]|\\(?=.))/g
// This matches bracket expressions (keep as-is) or escaped chars (remove backslash).
// We use the manual implementation since regexp2 doesn't support callback Replace.
func removeBackslashes(str string) string {
	return removeBackslashesManual(str)
}

// removeBackslashesManual is a manual implementation of removeBackslashes.
func removeBackslashesManual(str string) string {
	var b strings.Builder
	b.Grow(len(str))
	i := 0
	for i < len(str) {
		if str[i] == '[' {
			end := -1
			for j := i + 1; j < len(str); j++ {
				if str[j] == ']' && j > i+1 && str[j-1] != '\\' {
					end = j
					break
				}
			}
			if end != -1 {
				b.WriteString(str[i : end+1])
				i = end + 1
				continue
			}
		}

		if str[i] == '\\' && i+1 < len(str) {
			i++
			continue
		}

		if str[i] == '\\' {
			b.WriteByte(str[i])
			i++
			continue
		}

		b.WriteByte(str[i])
		i++
	}
	return b.String()
}

// escapeLast escapes the last occurrence of char in input.
// JS source: utils.js lines 36-41
func escapeLast(input string, char byte, lastIdx int) string {
	if lastIdx < 0 {
		lastIdx = len(input) - 1
	}
	idx := strings.LastIndexByte(input[:lastIdx+1], char)
	if idx == -1 {
		return input
	}
	if idx > 0 && input[idx-1] == '\\' {
		return escapeLast(input, char, idx-1)
	}
	return input[:idx] + `\` + input[idx:]
}

// removePrefix strips a leading "./" from input and records the prefix in state.
// JS source: utils.js lines 43-50
func removePrefix(input string, state *ParseState) string {
	output := input
	if strings.HasPrefix(output, "./") {
		output = output[2:]
		state.Prefix = "./"
	}
	return output
}

// wrapOutput wraps regex output with anchors and handles negation.
// JS source: utils.js lines 52-61
func wrapOutput(input string, state *ParseState, opts *Options) string {
	prepend := "^"
	appendStr := "$"
	if opts != nil && opts.Contains {
		prepend = ""
		appendStr = ""
	}

	output := prepend + "(?:" + input + ")" + appendStr
	if state != nil && state.Negated {
		output = "(?:^(?!" + output + ").*$)"
	}
	return output
}

// basename extracts the last path segment.
// JS source: utils.js lines 63-72
func basename(path string, windows bool) string {
	var segs []string
	if windows {
		// Split on both / and \
		segs = splitPathWindows(path)
	} else {
		segs = strings.Split(path, "/")
	}
	last := segs[len(segs)-1]
	if last == "" && len(segs) >= 2 {
		return segs[len(segs)-2]
	}
	return last
}

// splitPathWindows splits a path on both / and \.
func splitPathWindows(path string) []string {
	var result []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' || path[i] == '\\' {
			result = append(result, path[start:i])
			start = i + 1
		}
	}
	result = append(result, path[start:])
	return result
}
