// options_format_test.go — Faithful 1:1 port of picomatch/test/options.format.js
package picomatch

import (
	"sort"
	"strings"
	"testing"
)

// matchOutput is like match() but collects the formatted output strings instead
// of the original input strings. This mirrors the JS match() helper behavior from
// test/support/match.js which adds `match.output` to the result set, not the
// original input. When a Format function is set, the output is the formatted
// string.
//
// JS source (match.js lines 5-17):
//
//	const isMatch = picomatch(pattern, options, true);
//	const matches = options.matches || new Set();
//	for (const item of [].concat(list)) {
//	  const match = isMatch(item, true);
//	  if (match && match.output && match.isMatch === true) {
//	    matches.add(match.output);
//	  }
//	}
//	return [...matches];
func matchOutput(list []string, pattern string, opts *Options) []string {
	if opts == nil {
		opts = &Options{}
	}

	// Compile the regex and ignore matcher the same way Compile() does
	re := MakeRe(pattern, opts)
	posix := opts.Windows

	var isIgnored Matcher
	if opts.Ignore != nil {
		ignoreOpts := *opts
		ignoreOpts.Ignore = nil
		ignoreOpts.OnMatch = nil
		ignoreOpts.OnResult = nil
		isIgnored = Compile(opts.Ignore, &ignoreOpts)
	}

	var matches []string
	seen := map[string]bool{}

	for _, item := range list {
		testResult := Test(item, re.re, opts, pattern, posix)

		if testResult.IsMatch {
			// Check ignore
			if isIgnored != nil && isIgnored(item) {
				continue
			}
			output := testResult.Output
			if output != "" && !seen[output] {
				seen[output] = true
				matches = append(matches, output)
			}
		}
	}
	if matches == nil {
		matches = []string{}
	}
	return matches
}

// assertMatchOutputSorted compares matchOutput results after sorting both slices.
func assertMatchOutputSorted(t *testing.T, list []string, pattern string, expected []string, opts *Options) {
	t.Helper()
	result := matchOutput(list, pattern, opts)
	sort.Strings(result)
	sortedExpected := make([]string, len(expected))
	copy(sortedExpected, expected)
	sort.Strings(sortedExpected)
	if len(result) != len(sortedExpected) {
		t.Errorf("matchOutput(%v, %q): expected %v (len %d), got %v (len %d)",
			list, pattern, sortedExpected, len(sortedExpected), result, len(result))
		return
	}
	for i := range sortedExpected {
		if result[i] != sortedExpected[i] {
			t.Errorf("matchOutput(%v, %q)[%d]: expected %q, got %q (full expected: %v, full got: %v)",
				list, pattern, i, sortedExpected[i], result[i], sortedExpected, result)
			return
		}
	}
}

func TestOptionsFormat(t *testing.T) {
	t.Run("options.format", func(t *testing.T) {
		// see https://github.com/isaacs/minimatch/issues/30
		t.Run("should match the string returned by options.format", func(t *testing.T) {
			// options.format.js lines 15-16
			// opts = { format: str => str.replace(/\\/g, '/').replace(/^\.\//, ''), strictSlashes: true }
			opts := &Options{
				Format: func(str string) string {
					s := strings.ReplaceAll(str, "\\", "/")
					return strings.TrimPrefix(s, "./")
				},
				StrictSlashes: true,
			}
			fixtures := []string{
				"a", "./a", "b", "a/a", "./a/b", "a/c", "./a/x",
				"./a/a/a", "a/a/b", "./a/a/a/a", "./a/a/a/a/a", "x/y", "./z/z",
			}

			// options.format.js line 18
			assertMatch(t, false, "./.a", "*.a", opts)
			// options.format.js line 19
			assertMatch(t, false, "./.a", "./*.a", opts)
			// options.format.js line 20
			assertMatch(t, false, "./.a", "a/**/z/*.md", opts)
			// options.format.js line 21
			assertMatch(t, false, "./a/b/c/d/e/z/c.md", "./a/**/j/**/z/*.md", opts)
			// options.format.js line 22
			assertMatch(t, false, "./a/b/c/j/e/z/c.txt", "./a/**/j/**/z/*.md", opts)
			// options.format.js line 23
			assertMatch(t, false, "a/b/c/d/e/z/c.md", "./a/**/j/**/z/*.md", opts)
			// options.format.js line 24
			assertMatch(t, true, "./.a", "./.a", opts)
			// options.format.js line 25
			assertMatch(t, true, "./a/b/c.md", "a/**/*.md", opts)
			// options.format.js line 26
			assertMatch(t, true, "./a/b/c/d/e/j/n/p/o/z/c.md", "./a/**/j/**/z/*.md", opts)
			// options.format.js line 27
			assertMatch(t, true, "./a/b/c/d/e/z/c.md", "**/*.md", opts)
			// options.format.js line 28
			assertMatch(t, true, "./a/b/c/d/e/z/c.md", "./a/**/z/*.md", opts)
			// options.format.js line 29
			assertMatch(t, true, "./a/b/c/d/e/z/c.md", "a/**/z/*.md", opts)
			// options.format.js line 30
			assertMatch(t, true, "./a/b/c/j/e/z/c.md", "./a/**/j/**/z/*.md", opts)
			// options.format.js line 31
			assertMatch(t, true, "./a/b/c/j/e/z/c.md", "a/**/j/**/z/*.md", opts)
			// options.format.js line 32
			assertMatch(t, true, "./a/b/z/.a", "./a/**/z/.a", opts)
			// options.format.js line 33
			assertMatch(t, true, "./a/b/z/.a", "a/**/z/.a", opts)
			// options.format.js line 34
			assertMatch(t, true, ".a", "./.a", opts)
			// options.format.js line 35
			assertMatch(t, true, "a/b/c.md", "./a/**/*.md", opts)
			// options.format.js line 36
			assertMatch(t, true, "a/b/c.md", "a/**/*.md", opts)
			// options.format.js line 37
			assertMatch(t, true, "a/b/c/d/e/z/c.md", "a/**/z/*.md", opts)
			// options.format.js line 38
			assertMatch(t, true, "a/b/c/j/e/z/c.md", "a/**/j/**/z/*.md", opts)
			// options.format.js line 39
			assertMatch(t, true, "./a", "*", opts)

			// options.format.js line 41
			assertMatch(t, true, "./foo/bar.js", "**/foo/**", opts)
			// options.format.js line 42
			assertMatch(t, true, "./foo/bar.js", "./**/foo/**", opts)
			// options.format.js line 43
			assertMatch(t, true, `.\foo\bar.js`, "**/foo/**", &Options{
				Format: func(str string) string {
					s := strings.ReplaceAll(str, "\\", "/")
					return strings.TrimPrefix(s, "./")
				},
				StrictSlashes: true,
				Windows:       false,
			})
			// options.format.js line 44
			assertMatch(t, true, `.\foo\bar.js`, "./**/foo/**", opts)

			// options.format.js line 45
			// equal(match(fixtures, '*', opts), ['a', 'b']);
			// The JS match() helper adds match.output (formatted string) to the set.
			// So ./a → format → "a", and "a" (already in set) are deduplicated.
			assertMatchOutputSorted(t, fixtures, "*", []string{"a", "b"}, opts)

			// options.format.js line 46
			assertMatchOutputSorted(t, fixtures, "**/a/**",
				[]string{"a/a", "a/c", "a/b", "a/x", "a/a/a", "a/a/b", "a/a/a/a", "a/a/a/a/a"}, opts)
			// options.format.js line 47
			assertMatchOutputSorted(t, fixtures, "*/*",
				[]string{"a/a", "a/b", "a/c", "a/x", "x/y", "z/z"}, opts)
			// options.format.js line 48
			assertMatchOutputSorted(t, fixtures, "*/*/*",
				[]string{"a/a/a", "a/a/b"}, opts)
			// options.format.js line 49
			assertMatchOutputSorted(t, fixtures, "*/*/*/*",
				[]string{"a/a/a/a"}, opts)
			// options.format.js line 50
			assertMatchOutputSorted(t, fixtures, "*/*/*/*/*",
				[]string{"a/a/a/a/a"}, opts)
			// options.format.js line 51 — duplicate of line 45
			assertMatchOutputSorted(t, fixtures, "*", []string{"a", "b"}, opts)
			// options.format.js line 52 — duplicate of line 46
			assertMatchOutputSorted(t, fixtures, "**/a/**",
				[]string{"a/a", "a/c", "a/b", "a/x", "a/a/a", "a/a/b", "a/a/a/a", "a/a/a/a/a"}, opts)
			// options.format.js line 53
			assertMatchOutputSorted(t, fixtures, "a/*/a",
				[]string{"a/a/a"}, opts)
			// options.format.js line 54
			assertMatchOutputSorted(t, fixtures, "a/*",
				[]string{"a/a", "a/b", "a/c", "a/x"}, opts)
			// options.format.js line 55
			assertMatchOutputSorted(t, fixtures, "a/*/*",
				[]string{"a/a/a", "a/a/b"}, opts)
			// options.format.js line 56
			assertMatchOutputSorted(t, fixtures, "a/*/*/*",
				[]string{"a/a/a/a"}, opts)
			// options.format.js line 57
			assertMatchOutputSorted(t, fixtures, "a/*/*/*/*",
				[]string{"a/a/a/a/a"}, opts)
			// options.format.js line 58 — duplicate of line 53
			assertMatchOutputSorted(t, fixtures, "a/*/a",
				[]string{"a/a/a"}, opts)
		})
	})
}
