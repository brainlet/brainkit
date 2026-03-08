package picomatch

// Ported from: picomatch/lib/picomatch.js + picomatch/index.js

import (
	"reflect"
	"runtime"
	"strings"
	"unsafe"

	"github.com/dlclark/regexp2"
)

// Options configures picomatch behavior.
// JS source: picomatch.js — options used across all functions
type Options struct {
	// Core matching options
	Windows        bool  // Treat paths as Windows-style (backslash separator)
	Posix          bool  // Force POSIX mode (forward slash separator, no Windows auto-detect)
	Dot            bool  // Match dotfiles (files starting with .)
	Nocase         bool  // Case-insensitive matching
	Contains       bool  // Match anywhere in string (don't anchor to start/end)
	MatchBase      bool  // Match basename only (like find's -name)
	Basename       bool  // Alias for MatchBase
	Bash           bool  // Bash-style matching
	Capture        bool  // Create capturing groups in regex
	Regex          *bool // When true, treat glob as regex in certain contexts
	StrictSlashes  bool  // Don't add optional trailing slash
	StrictBrackets bool  // Throw on unmatched brackets

	// Feature toggles
	Nobrace    bool // Disable brace expansion
	Nobracket  bool // Disable bracket expressions
	Noextglob  bool // Disable extended globs +(a|b)
	Noext      bool // Alias for Noextglob (minimatch compat)
	Noglobstar bool // Disable globstar (**)
	Nonegate   bool // Disable negation
	NoPosix    bool // Disable POSIX bracket classes
	Noparen    bool // Disable parentheses

	// Fastpath control
	Fastpaths *bool // Enable/disable fastpaths (default: true)

	// Content options
	KeepQuotes      bool  // Keep double quotes in output
	LiteralBrackets *bool // Force literal bracket matching
	Unescape        bool  // Remove backslashes from output

	// String processing
	MaxLength int                 // Maximum allowed pattern length
	Prepend   string              // String to prepend to regex output
	Format    func(string) string // Custom format function

	// Callbacks
	OnResult func(result *MatchResult) // Called on every test result
	OnMatch  func(result *MatchResult) // Called on match
	OnIgnore func(result *MatchResult) // Called when ignored

	// Ignore patterns
	Ignore interface{} // string, []string, or compiled matcher

	// Range expansion
	ExpandRange func(args []string, opts *Options) string

	// Flags for compiled regex
	Flags string

	// Debug mode — if true, throw on invalid regex instead of returning never-match
	Debug bool
}

// MatchResult contains the result of testing a string against a pattern.
// JS source: picomatch.js line 67
type MatchResult struct {
	Glob    string
	State   *ParseState
	Regex   *regexp2.Regexp
	Posix   bool
	Input   string
	Output  string
	Match   *regexp2.Match
	IsMatch bool
}

// Matcher is a compiled matcher function that tests strings.
// JS source: picomatch.js line 65
type Matcher func(input string) bool

// MatcherWithResult is a matcher that can optionally return detailed results.
type MatcherWithResult func(input string, returnObject bool) *MatchResult

// Compile creates a matcher function from one or more glob patterns.
// This is the main entry point — equivalent to picomatch() in JS.
// JS source: picomatch.js lines 31-97 + index.js lines 6-14
func Compile(glob interface{}, opts *Options) Matcher {
	matcher := CompileWithResult(glob, opts)
	return func(input string) bool {
		result := matcher(input, false)
		return result != nil && result.IsMatch
	}
}

// CompileWithResult creates a matcher that can return a MatchResult object.
// This is the Go equivalent of calling the JS matcher with returnObject=true.
func CompileWithResult(glob interface{}, opts *Options) MatcherWithResult {
	if opts == nil {
		opts = &Options{}
	}

	// Auto-detect Windows if not explicitly set and not in Posix mode
	// JS source: index.js lines 8-11
	if !opts.Posix && runtime.GOOS == "windows" {
		opts.Windows = true
	}

	// Handle array of globs
	// JS source: picomatch.js lines 32-42
	if globs, ok := glob.([]string); ok {
		fns := make([]MatcherWithResult, len(globs))
		for i, g := range globs {
			fns[i] = CompileWithResult(g, opts)
		}
		return func(str string, returnObject bool) *MatchResult {
			for _, fn := range fns {
				result := fn(str, true)
				if result != nil && result.IsMatch {
					return result
				}
			}
			if returnObject {
				return &MatchResult{Input: str, Output: str, IsMatch: false}
			}
			return nil
		}
	}

	if state, ok := glob.(*ParseState); ok {
		re := CompileRe(state, opts, false, true)
		posix := opts.Windows
		return newMatcherWithResult(glob, state, re.re, posix, opts)
	}

	pattern, ok := glob.(string)
	if !ok {
		panic("Expected pattern to be a non-empty string")
	}
	if pattern == "" {
		panic("Expected pattern to be a non-empty string")
	}

	// Compile the regex
	// JS source: picomatch.js lines 52-54
	re := MakeRe(pattern, opts)
	state := re.state

	posix := opts.Windows

	return newMatcherWithResult(pattern, state, re.re, posix, opts)
}

func newMatcherWithResult(glob interface{}, state *ParseState, regex *regexp2.Regexp, posix bool, opts *Options) MatcherWithResult {
	var isIgnored Matcher
	if opts.Ignore != nil {
		ignoreOpts := *opts
		ignoreOpts.Ignore = nil
		ignoreOpts.OnMatch = nil
		ignoreOpts.OnResult = nil
		isIgnored = Compile(opts.Ignore, &ignoreOpts)
	}

	globPattern := globToString(glob)

	return func(input string, returnObject bool) *MatchResult {
		testResult := Test(input, regex, opts, globPattern, posix)
		result := &MatchResult{
			Glob:    globPattern,
			State:   state,
			Regex:   regex,
			Posix:   posix,
			Input:   input,
			Output:  testResult.Output,
			Match:   testResult.Match,
			IsMatch: testResult.IsMatch,
		}

		if opts.OnResult != nil {
			opts.OnResult(result)
		}

		if !result.IsMatch {
			if returnObject {
				return result
			}
			return nil
		}

		if isIgnored != nil && isIgnored(input) {
			if opts.OnIgnore != nil {
				opts.OnIgnore(result)
			}
			result.IsMatch = false
			if returnObject {
				return result
			}
			return nil
		}

		if opts.OnMatch != nil {
			opts.OnMatch(result)
		}

		return result
	}
}

// compiledRegex wraps a regexp2.Regexp with its associated parse state.
type compiledRegex struct {
	re    *regexp2.Regexp
	state *ParseState
}

// TestResult holds the result of Test().
type TestResult struct {
	IsMatch bool
	Match   *regexp2.Match
	Output  string
}

// Test tests input against a compiled regex.
// JS source: picomatch.js lines 116-144
func Test(input string, regex *regexp2.Regexp, options *Options, glob string, posix bool) TestResult {
	if input == "" {
		return TestResult{IsMatch: false, Output: ""}
	}

	opts := options
	if opts == nil {
		opts = &Options{}
	}

	var format func(string) string
	if opts.Format != nil {
		format = opts.Format
	} else if posix {
		format = toPosixSlashes
	}

	matched := input == glob
	output := input
	if matched && format != nil {
		output = format(input)
	}

	if !matched {
		if format != nil {
			output = format(input)
		} else {
			output = input
		}
		matched = output == glob
	}

	if !matched || opts.Capture {
		if opts.MatchBase || opts.Basename {
			return TestResult{
				IsMatch: MatchBase(input, regex, posix),
				Match:   nil,
				Output:  output,
			}
		}
		m, _ := regex.FindStringMatch(output)
		return TestResult{IsMatch: m != nil, Match: m, Output: output}
	}

	return TestResult{IsMatch: matched, Match: nil, Output: output}
}

// MatchBase matches the basename of a filepath against a regex.
// JS source: picomatch.js lines 160-163
func MatchBase(input string, regex *regexp2.Regexp, _ bool) bool {
	// Upstream matchBase() calls utils.basename(input) without forwarding
	// the windows option, so matching is always split on '/' here.
	base := basename(input, false)
	ok, _ := regex.MatchString(base)
	return ok
}

// IsMatch returns true if any of the given patterns match the string.
// JS source: picomatch.js line 182
func IsMatch(str string, patterns interface{}, options *Options) bool {
	return Compile(patterns, options)(str)
}

// MakeRe creates a regular expression from a glob pattern.
// JS source: picomatch.js lines 285-301
func MakeRe(input string, options *Options) *compiledRegex {
	if input == "" {
		panic("Expected a non-empty string")
	}

	opts := options
	if opts == nil {
		opts = &Options{}
	}

	parsed := &ParseState{Negated: false, Fastpaths: true}

	// Try fastpaths first
	// JS source: picomatch.js lines 292-294
	if opts.Fastpaths == nil || *opts.Fastpaths {
		if len(input) > 0 && (input[0] == '.' || input[0] == '*') {
			parsed.Output = Fastpaths(input, opts)
		}
	}

	if parsed.Output == "" {
		parsed = parseInternal(input, opts)
	}

	return CompileRe(parsed, opts, false, true)
}

// CompileRe compiles a regular expression from a ParseState.
// JS source: picomatch.js lines 244-264
func CompileRe(state *ParseState, options *Options, returnOutput bool, returnState bool) *compiledRegex {
	opts := options
	if opts == nil {
		opts = &Options{}
	}

	prepend := "^"
	appendStr := "$"
	if opts.Contains {
		prepend = ""
		appendStr = ""
	}

	source := prepend + "(?:" + state.Output + ")" + appendStr
	if state.Negated {
		source = "^(?!" + source + ").*$"
	}

	re := ToRegex(source, opts)

	result := &compiledRegex{re: re}
	if returnState {
		result.state = state
	}

	return result
}

// CompileReOutput returns the raw parser output string.
// This preserves the upstream compileRe(..., returnOutput=true) capability
// without changing the static return type of CompileRe in Go.
func CompileReOutput(state *ParseState, options *Options) string {
	return state.Output
}

// MakeReOutput mirrors upstream makeRe(..., returnOutput=true).
// When makeRe takes a fastpath, the returned string may already be wrapped.
func MakeReOutput(input string, options *Options) string {
	if input == "" {
		panic("Expected a non-empty string")
	}

	opts := options
	if opts == nil {
		opts = &Options{}
	}

	parsed := &ParseState{Negated: false, Fastpaths: true}
	if opts.Fastpaths == nil || *opts.Fastpaths {
		if len(input) > 0 && (input[0] == '.' || input[0] == '*') {
			parsed.Output = Fastpaths(input, opts)
		}
	}

	if parsed.Output == "" {
		parsed = parseInternal(input, opts)
	}

	return CompileReOutput(parsed, opts)
}

// ToRegex creates a regexp2.Regexp from a source string.
// JS source: picomatch.js lines 320-328
func ToRegex(source string, options *Options) *regexp2.Regexp {
	opts := options
	if opts == nil {
		opts = &Options{}
	}

	publicSource := source
	source = normalizeJSRegexSource(source)

	flags := regexp2.None
	if opts.Flags != "" {
		if strings.Contains(opts.Flags, "i") {
			flags |= regexp2.IgnoreCase
		}
		if strings.Contains(opts.Flags, "m") {
			flags |= regexp2.Multiline
		}
	} else if opts.Nocase {
		flags |= regexp2.IgnoreCase
	}

	re, err := regexp2.Compile(source, flags)
	if err != nil {
		if opts.Debug {
			panic(err)
		}
		// Return a never-matching regex (equivalent to JS /$^/)
		re, _ = regexp2.Compile(`$^`, regexp2.None)
	}

	if publicSource != source {
		setRegexpSource(re, publicSource)
	}

	return re
}

func normalizeJSRegexSource(source string) string {
	// Keep parser output/source fidelity separate from regexp2 compatibility:
	// any transformation here should only adapt valid JS regex syntax that
	// regexp2 cannot compile as-is.
	source = normalizeJSCharClasses(source)

	var b strings.Builder
	b.Grow(len(source))

	for i := 0; i < len(source); {
		if source[i] != '\\' {
			b.WriteByte(source[i])
			i++
			continue
		}

		j := i
		for j < len(source) && source[j] == '\\' {
			j++
		}

		if j < len(source) && isASCIIAlpha(source[j]) && (j-i)%2 == 1 {
			for k := 0; k < (j-i)/2; k++ {
				b.WriteString(`\\`)
			}
			if isSupportedJSEscapeStart(source[j]) {
				b.WriteByte('\\')
			}
			b.WriteByte(source[j])
			i = j + 1
			continue
		}

		b.WriteString(source[i:j])
		i = j
	}

	return b.String()
}

func normalizeJSCharClasses(source string) string {
	var b strings.Builder
	b.Grow(len(source))

	inClass := false
	escaped := false

	for i := 0; i < len(source); i++ {
		ch := source[i]

		if !inClass {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '[' {
				inClass = true
			}
			continue
		}

		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			b.WriteByte(ch)
			escaped = true
		case '[':
			b.WriteString(`\[`)
		case ']':
			b.WriteByte(ch)
			inClass = false
		default:
			b.WriteByte(ch)
		}
	}

	return b.String()
}

func isASCIIAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isSupportedJSEscapeStart(ch byte) bool {
	switch ch {
	case 'b', 'B', 'd', 'D', 'f', 'n', 'r', 's', 'S', 't', 'u', 'v', 'w', 'W', 'x':
		return true
	default:
		return false
	}
}

// CompilePosix creates a matcher without Windows auto-detection.
// Equivalent to require('picomatch/posix').
// JS source: posix.js
func CompilePosix(glob interface{}, opts *Options) Matcher {
	if opts == nil {
		opts = &Options{}
	}
	opts.Posix = true
	return Compile(glob, opts)
}

func globToString(glob interface{}) string {
	if s, ok := glob.(string); ok {
		return s
	}
	if state, ok := glob.(*ParseState); ok && state != nil {
		return state.Input
	}
	return ""
}

func setRegexpSource(re *regexp2.Regexp, source string) {
	field := reflect.ValueOf(re).Elem().FieldByName("pattern")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().SetString(source)
}
