// options_expand_range_test.go — Faithful 1:1 port of picomatch/test/options.expandRange.js
package picomatch

import (
	"fmt"
	"testing"
)

func TestOptionsExpandRange(t *testing.T) {
	t.Run("options.expandRange", func(t *testing.T) {
		t.Run("should support a custom function for expanding ranges in brace patterns", func(t *testing.T) {
			// options.expandRange.js line 9
			// expandRange: (a, b) => `([${a}-${b}])`
			expandCharRange := func(args []string, opts *Options) string {
				if len(args) >= 2 {
					return fmt.Sprintf("([%s-%s])", args[0], args[1])
				}
				return ""
			}
			assertMatch(t, true, "a/c", "a/{a..c}", &Options{ExpandRange: expandCharRange})
			// options.expandRange.js line 10
			assertMatch(t, false, "a/z", "a/{a..c}", &Options{ExpandRange: expandCharRange})

			// options.expandRange.js lines 11-15
			// expandRange uses fill-range with toRegex:true to expand numeric ranges.
			// fill(1, 100, { toRegex: true }) produces something like ([1-9]|[1-9][0-9]|100)
			// We approximate this with a custom Go function that generates the same regex.
			expandNumRange := func(args []string, opts *Options) string {
				// For the test case a/{1..100}, we need a regex that matches 1-100.
				// fill-range with toRegex:true for (1, 100) produces: ([1-9]|[1-9][0-9]|100)
				if len(args) >= 2 && args[0] == "1" && args[1] == "100" {
					return "([1-9]|[1-9][0-9]|100)"
				}
				return ""
			}
			assertMatch(t, true, "a/99", "a/{1..100}", &Options{ExpandRange: expandNumRange})
		})
	})
}
