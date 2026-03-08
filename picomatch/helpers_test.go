package picomatch

// helpers_test.go — Test helpers ported from picomatch/test/support/match.js
// These helpers provide the `match()` function used across all picomatch tests.

import (
	"testing"
)

// match filters a list of strings against a glob pattern, returning those that match.
// Returns match.output (the formatted/transformed string) for each match, not the original input.
// This faithfully mirrors the JS match() helper which adds match.output to the result set.
//
// Ported from: test/support/match.js lines 5-17
// JS source:
//
//	module.exports = (list, pattern, options = {}) => {
//	  const isMatch = picomatch(pattern, options, true);
//	  const matches = options.matches || new Set();
//	  for (const item of [].concat(list)) {
//	    const match = isMatch(item, true);
//	    if (match && match.output && match.isMatch === true) {
//	      matches.add(match.output);
//	    }
//	  }
//	  return [...matches];
//	};
func match(list []string, pattern string, opts ...*Options) []string {
	var o *Options
	if len(opts) > 0 && opts[0] != nil {
		o = opts[0]
	}
	if o == nil {
		o = &Options{}
	}

	// Compile the regex the same way Compile() does internally
	// JS source: picomatch.js lines 52-54
	re := MakeRe(pattern, o)
	posix := o.Windows

	// Build ignore matcher
	// JS source: picomatch.js lines 59-63
	var isIgnored Matcher
	if o.Ignore != nil {
		ignoreOpts := *o
		ignoreOpts.Ignore = nil
		ignoreOpts.OnMatch = nil
		ignoreOpts.OnResult = nil
		isIgnored = Compile(o.Ignore, &ignoreOpts)
	}

	var matches []string
	seen := map[string]bool{}

	for _, item := range list {
		// Use Test() to get the output (formatted string)
		// JS source: picomatch.js lines 65-96
		testResult := Test(item, re.re, o, pattern, posix)

		if o.OnResult != nil {
			o.OnResult(&MatchResult{
				Glob:    pattern,
				State:   re.state,
				Regex:   re.re,
				Posix:   posix,
				Input:   item,
				Output:  testResult.Output,
				Match:   testResult.Match,
				IsMatch: testResult.IsMatch,
			})
		}

		if !testResult.IsMatch {
			continue
		}

		// Check ignore
		if isIgnored != nil && isIgnored(item) {
			continue
		}

		if o.OnMatch != nil {
			o.OnMatch(&MatchResult{
				Glob:    pattern,
				State:   re.state,
				Regex:   re.re,
				Posix:   posix,
				Input:   item,
				Output:  testResult.Output,
				Match:   testResult.Match,
				IsMatch: testResult.IsMatch,
			})
		}

		// Add match.output (not the original item) — matches JS behavior
		output := testResult.Output
		if output != "" && !seen[output] {
			seen[output] = true
			matches = append(matches, output)
		}
	}
	if matches == nil {
		matches = []string{}
	}
	return matches
}

// assertMatch checks that isMatch(input, pattern, opts...) returns the expected result.
// Includes source file and line reference for debugging.
func assertMatch(t *testing.T, expected bool, input, pattern string, opts ...*Options) {
	t.Helper()
	var o *Options
	if len(opts) > 0 && opts[0] != nil {
		o = opts[0]
	}
	result := IsMatch(input, pattern, o)
	if result != expected {
		if expected {
			t.Errorf("expected isMatch(%q, %q) to be true, got false", input, pattern)
		} else {
			t.Errorf("expected isMatch(%q, %q) to be false, got true", input, pattern)
		}
	}
}

// assertMatchList checks that match(list, pattern, opts...) returns the expected list.
func assertMatchList(t *testing.T, list []string, pattern string, expected []string, opts ...*Options) {
	t.Helper()
	var o *Options
	if len(opts) > 0 && opts[0] != nil {
		o = opts[0]
	}
	result := match(list, pattern, o)
	if len(result) != len(expected) {
		t.Errorf("match(%v, %q): expected %v (len %d), got %v (len %d)",
			list, pattern, expected, len(expected), result, len(result))
		return
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("match(%v, %q)[%d]: expected %q, got %q",
				list, pattern, i, expected[i], result[i])
			return
		}
	}
}
