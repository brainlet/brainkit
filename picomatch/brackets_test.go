// brackets_test.go — Faithful 1:1 port of picomatch/test/brackets.js
package picomatch

import "testing"

func TestBrackets(t *testing.T) {
	t.Run("trailing stars", func(t *testing.T) {
		t.Run("should support stars following brackets", func(t *testing.T) {
			// brackets.js line 9
			assertMatch(t, true, "a", "[a]*")
			// brackets.js line 10
			assertMatch(t, true, "aa", "[a]*")
			// brackets.js line 11
			assertMatch(t, true, "aaa", "[a]*")
			// brackets.js line 12
			assertMatch(t, true, "az", "[a-z]*")
			// brackets.js line 13
			assertMatch(t, true, "zzz", "[a-z]*")
		})

		t.Run("should match slashes defined in brackets", func(t *testing.T) {
			// brackets.js line 17
			assertMatch(t, true, "foo/bar", "foo[/]bar")
			// brackets.js line 18
			assertMatch(t, true, "foo/bar/", "foo[/]bar[/]")
			// brackets.js line 19
			assertMatch(t, true, "foo/bar/baz", "foo[/]bar[/]baz")
		})

		t.Run("should not match slashes following brackets", func(t *testing.T) {
			// brackets.js line 23
			assertMatch(t, false, "a/b", "[a]*")
		})
	})
}
