// parens_test.go — Faithful 1:1 port of picomatch/test/parens.js
package picomatch

import "testing"

func TestParensNonExtglobs(t *testing.T) {
	t.Run("should support stars following parens", func(t *testing.T) {
		// parens.js line 8
		assertMatch(t, true, "a", "(a)*")
		// parens.js line 9
		assertMatch(t, true, "az", "(a)*")
		// parens.js line 10
		assertMatch(t, false, "zz", "(a)*")
		// parens.js line 11
		assertMatch(t, true, "ab", "(a|b)*")
		// parens.js line 12
		assertMatch(t, true, "abc", "(a|b)*")
		// parens.js line 13
		assertMatch(t, true, "aa", "(a)*")
		// parens.js line 14
		assertMatch(t, true, "aaab", "(a|b)*")
		// parens.js line 15
		assertMatch(t, true, "aaabbb", "(a|b)*")
	})

	t.Run("should not match slashes with single stars", func(t *testing.T) {
		// parens.js line 19
		assertMatch(t, false, "a/b", "(a)*")
		// parens.js line 20
		assertMatch(t, false, "a/b", "(a|b)*")
	})
}
