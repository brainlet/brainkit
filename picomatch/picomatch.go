package picomatch

// Ported from: picomatch/lib/picomatch.js + picomatch/index.js

import (
	"runtime"
	"strings"

	"github.com/dlclark/regexp2"
)

// Options configures picomatch behavior.
// JS source: picomatch.js — options used across all functions
type Options struct {
	// Core matching options
	Windows       bool // Treat paths as Windows-style (backslash separator)
	Posix         bool // Force POSIX mode (forward slash separator, no Windows auto-detect)
	Dot           bool // Match dotfiles (files starting with .)
	Nocase        bool // Case-insensitive matching
	Contains      bool // Match anywhere in string (don't anchor to start/end)
	MatchBase     bool // Match basename only (like find's -name)
	Basename      bool // Alias for MatchBase
	Bash          bool // Bash-style matching
	Capture       bool // Create capturing groups in regex
	Regex         *bool // When true, treat glob as regex in certain contexts
	StrictSlashes bool // Don't add optional trailing slash
	StrictBrackets bool // Throw on unmatched brackets

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
	KeepQuotes     bool // Keep double quotes in output
	LiteralBrackets *bool // Force literal bracket matching
	Unescape       bool // Remove backslashes from output

	// String processing
	MaxLength int    // Maximum allowed pattern length
	Prepend   string // String to prepend to regex output
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
		fns := make([]Matcher, len(globs))
		for i, g := range globs {
			fns[i] = Compile(g, opts)
		}
		return func(str string) bool {
			for _, fn := range fns {
				if fn(str) {
					return true
				}
			}
			return false
		}
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

	// Build ignore matcher
	// JS source: picomatch.js lines 59-63
	var isIgnored Matcher
	if opts.Ignore != nil {
		ignoreOpts := *opts
		ignoreOpts.Ignore = nil
		ignoreOpts.OnMatch = nil
		ignoreOpts.OnResult = nil
		isIgnored = Compile(opts.Ignore, &ignoreOpts)
	}

	// Return matcher function
	// JS source: picomatch.js lines 65-96
	return func(input string) bool {
		testResult := Test(input, re.re, opts, pattern, posix)
		result := &MatchResult{
			Glob:    pattern,
			State:   state,
			Regex:   re.re,
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
			return false
		}

		if isIgnored != nil && isIgnored(input) {
			if opts.OnIgnore != nil {
				opts.OnIgnore(result)
			}
			return false
		}

		if opts.OnMatch != nil {
			opts.OnMatch(result)
		}

		return true
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
		if m != nil {
			return TestResult{IsMatch: true, Match: m, Output: output}
		}
	}

	return TestResult{IsMatch: matched, Match: nil, Output: output}
}

// MatchBase matches the basename of a filepath against a regex.
// JS source: picomatch.js lines 160-163
func MatchBase(input string, regex *regexp2.Regexp, posix bool) bool {
	base := basename(input, !posix)
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
		parsed = Parse(input, opts)
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

// ToRegex creates a regexp2.Regexp from a source string.
// JS source: picomatch.js lines 320-328
func ToRegex(source string, options *Options) *regexp2.Regexp {
	opts := options
	if opts == nil {
		opts = &Options{}
	}

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

	return re
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
