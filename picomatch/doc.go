// Package picomatch provides a Go port of picomatch.
//
// This is a faithful 1:1 port of https://github.com/micromatch/picomatch
// JS source: lib/ (2089 lines across 5 files)
//
// Key differences from JS:
//   - Go's standard regexp (RE2) does not support lookaheads; this port
//     uses github.com/dlclark/regexp2 which provides full .NET-compatible
//     regex including (?=...) and (?!...) lookaheads.
//   - regexp2.Regexp.MatchString returns (bool, error); errors from
//     malformed patterns are handled gracefully (return false).
//   - JS RegExp constructor failures silently return /$^/ (never-match);
//     Go mirrors this with a nil-safe fallback.
//   - platform detection (process.platform / navigator.platform) replaced
//     with runtime.GOOS == "windows".
//   - JS mutable regex.state attachment replaced with Go struct fields.
//   - Callback options (onMatch, onResult, onIgnore) use Go func types.
//
// Ported features:
//   - Star matching (*, **)
//   - Question marks (?)
//   - Bracket expressions ([abc], [^abc], [a-z], POSIX classes)
//   - Brace expansion ({foo,bar}, {1..5})
//   - Extended globs (+(a|b), *(a|b), ?(a|b), @(a|b), !(a|b))
//   - Negation (!pattern)
//   - Dot file handling (opt-in via Dot option)
//   - Windows path separator support
//   - Pattern scanning (Scan)
//   - Pattern parsing with token stream (Parse)
//   - Regex compilation (MakeRe, CompileRe, ToRegex)
//   - Ignore patterns
//   - matchBase / basename matching
//   - Fast paths for common patterns
//
// Usage:
//
//	isMatch := picomatch.IsMatch("foo/bar.js", "**/*.js", nil)  // true
//
//	matcher := picomatch.Compile("*.js", nil)
//	matcher("foo.js")    // true
//	matcher("foo.ts")    // false
//
//	resultMatcher := picomatch.CompileWithResult("*.js", nil)
//	result := resultMatcher("foo.js", true)
//	_ = result.IsMatch
//
//	re := picomatch.MakeRe("*.js", nil)
//	ok, _ := re.MatchString("foo.js")  // true
//
//	output := picomatch.MakeReOutput("*.js", nil) // mirrors makeRe(..., true)
//	_ = output
package picomatch
