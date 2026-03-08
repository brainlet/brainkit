// options_onmatch_test.go — Faithful 1:1 port of picomatch/test/options.onMatch.js
package picomatch

import (
	"sort"
	"strings"
	"testing"
)

// matchWithOnMatch mirrors the JS match() helper behavior from test/support/match.js,
// but also wires up the onMatch callback defined in the JS test.
//
// JS source (options.onMatch.js lines 13-25):
//
//	const format = str => str.replace(/^\.\//, '');
//	const options = () => {
//	  return {
//	    format,
//	    onMatch({ pattern, regex, input, output }, matches) {
//	      if (output.length > 2 && (output.startsWith('./') || output.startsWith('.\\'))) {
//	        output = output.slice(2);
//	      }
//	      if (matches) {
//	        matches.add(output);
//	      }
//	    }
//	  };
//	};
//
// IMPORTANT: In the JS picomatch source, when returnObject=true and onMatch is set,
// the onMatch callback receives the result object AND a Set (matches) passed from the
// match() helper via options.matches. The onMatch callback adds the transformed output
// to the matches Set. Our Go implementation doesn't pass a matches set to onMatch,
// so we capture outputs via the OnMatch callback closure instead.
func matchWithOnMatch(list []string, pattern string) []string {
	format := func(str string) string {
		return strings.TrimPrefix(str, "./")
	}

	var collected []string
	seen := map[string]bool{}

	opts := &Options{
		Format: format,
		OnMatch: func(result *MatchResult) {
			// JS source: options.onMatch.js lines 16-22
			// Replicate the onMatch behavior: strip leading ./ or .\ from output
			output := result.Output
			if len(output) > 2 && (strings.HasPrefix(output, "./") || strings.HasPrefix(output, `.`+`\`)) {
				output = output[2:]
			}
			if !seen[output] {
				seen[output] = true
				collected = append(collected, output)
			}
		},
	}

	// Run the matcher on each item - OnMatch callback collects results
	isMatchFn := Compile(pattern, opts)
	for _, item := range list {
		isMatchFn(item)
	}

	if collected == nil {
		collected = []string{}
	}
	return collected
}

// assertOnMatchSorted compares matchWithOnMatch results after sorting both slices.
func assertOnMatchSorted(t *testing.T, list []string, pattern string, expected []string) {
	t.Helper()
	result := matchWithOnMatch(list, pattern)
	sort.Strings(result)
	sortedExpected := make([]string, len(expected))
	copy(sortedExpected, expected)
	sort.Strings(sortedExpected)
	if len(result) != len(sortedExpected) {
		t.Errorf("matchWithOnMatch(%v, %q): expected %v (len %d), got %v (len %d)",
			list, pattern, sortedExpected, len(sortedExpected), result, len(result))
		return
	}
	for i := range sortedExpected {
		if result[i] != sortedExpected[i] {
			t.Errorf("matchWithOnMatch(%v, %q)[%d]: expected %q, got %q (full expected: %v, full got: %v)",
				list, pattern, i, sortedExpected[i], result[i], sortedExpected, result)
			return
		}
	}
}

func TestOptionsOnMatch(t *testing.T) {
	t.Run("options.onMatch", func(t *testing.T) {
		t.Run("should call options.onMatch on each matching string", func(t *testing.T) {
			// options.onMatch.js line 29
			fixtures := []string{
				"a", "./a", "b", "a/a", "./a/b", "a/c", "./a/x",
				"./a/a/a", "a/a/b", "./a/a/a/a", "./a/a/a/a/a", "x/y", "./z/z",
			}

			format := func(str string) string {
				return strings.TrimPrefix(str, "./")
			}

			// options.onMatch.js line 31
			assertMatch(t, false, "./.a", "*.a", &Options{Format: format})
			// options.onMatch.js line 32
			assertMatch(t, false, "./.a", "./*.a", &Options{Format: format})
			// options.onMatch.js line 33
			assertMatch(t, false, "./.a", "a/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 34
			assertMatch(t, false, "./a/b/c/d/e/z/c.md", "./a/**/j/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 35
			assertMatch(t, false, "./a/b/c/j/e/z/c.txt", "./a/**/j/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 36
			assertMatch(t, false, "a/b/c/d/e/z/c.md", "./a/**/j/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 37
			assertMatch(t, true, "./.a", "./.a", &Options{Format: format})
			// options.onMatch.js line 38
			assertMatch(t, true, "./a/b/c.md", "a/**/*.md", &Options{Format: format})
			// options.onMatch.js line 39
			assertMatch(t, true, "./a/b/c/d/e/j/n/p/o/z/c.md", "./a/**/j/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 40
			assertMatch(t, true, "./a/b/c/d/e/z/c.md", "**/*.md", &Options{Format: format})
			// options.onMatch.js line 41
			assertMatch(t, true, "./a/b/c/d/e/z/c.md", "./a/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 42
			assertMatch(t, true, "./a/b/c/d/e/z/c.md", "a/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 43
			assertMatch(t, true, "./a/b/c/j/e/z/c.md", "./a/**/j/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 44
			assertMatch(t, true, "./a/b/c/j/e/z/c.md", "a/**/j/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 45
			assertMatch(t, true, "./a/b/z/.a", "./a/**/z/.a", &Options{Format: format})
			// options.onMatch.js line 46
			assertMatch(t, true, "./a/b/z/.a", "a/**/z/.a", &Options{Format: format})
			// options.onMatch.js line 47
			assertMatch(t, true, ".a", "./.a", &Options{Format: format})
			// options.onMatch.js line 48
			assertMatch(t, true, "a/b/c.md", "./a/**/*.md", &Options{Format: format})
			// options.onMatch.js line 49
			assertMatch(t, true, "a/b/c.md", "a/**/*.md", &Options{Format: format})
			// options.onMatch.js line 50
			assertMatch(t, true, "a/b/c/d/e/z/c.md", "a/**/z/*.md", &Options{Format: format})
			// options.onMatch.js line 51
			assertMatch(t, true, "a/b/c/j/e/z/c.md", "a/**/j/**/z/*.md", &Options{Format: format})

			// options.onMatch.js line 53
			assertOnMatchSorted(t, fixtures, "*", []string{"a", "b"})
			// options.onMatch.js line 54
			assertOnMatchSorted(t, fixtures, "**/a/**",
				[]string{"a", "a/a", "a/c", "a/b", "a/x", "a/a/a", "a/a/b", "a/a/a/a", "a/a/a/a/a"})
			// options.onMatch.js line 55
			assertOnMatchSorted(t, fixtures, "*/*",
				[]string{"a/a", "a/b", "a/c", "a/x", "x/y", "z/z"})
			// options.onMatch.js line 56
			assertOnMatchSorted(t, fixtures, "*/*/*",
				[]string{"a/a/a", "a/a/b"})
			// options.onMatch.js line 57
			assertOnMatchSorted(t, fixtures, "*/*/*/*",
				[]string{"a/a/a/a"})
			// options.onMatch.js line 58
			assertOnMatchSorted(t, fixtures, "*/*/*/*/*",
				[]string{"a/a/a/a/a"})
			// options.onMatch.js line 59
			assertOnMatchSorted(t, fixtures, "./*",
				[]string{"a", "b"})
			// options.onMatch.js line 60
			assertOnMatchSorted(t, fixtures, "./**/a/**",
				[]string{"a", "a/a", "a/b", "a/c", "a/x", "a/a/a", "a/a/b", "a/a/a/a", "a/a/a/a/a"})
			// options.onMatch.js line 61
			assertOnMatchSorted(t, fixtures, "./a/*/a",
				[]string{"a/a/a"})
			// options.onMatch.js line 62
			assertOnMatchSorted(t, fixtures, "a/*",
				[]string{"a/a", "a/b", "a/c", "a/x"})
			// options.onMatch.js line 63
			assertOnMatchSorted(t, fixtures, "a/*/*",
				[]string{"a/a/a", "a/a/b"})
			// options.onMatch.js line 64
			assertOnMatchSorted(t, fixtures, "a/*/*/*",
				[]string{"a/a/a/a"})
			// options.onMatch.js line 65
			assertOnMatchSorted(t, fixtures, "a/*/*/*/*",
				[]string{"a/a/a/a/a"})
			// options.onMatch.js line 66
			assertOnMatchSorted(t, fixtures, "a/*/a",
				[]string{"a/a/a"})
		})
	})
}
