// non_globs_test.go — Faithful 1:1 port of picomatch/test/non-globs.js
package picomatch

import (
	"testing"
)

func TestNonGlobs(t *testing.T) {
	t.Run("non-globs", func(t *testing.T) {
		t.Run("should match non-globs", func(t *testing.T) {
			// non-globs.js line 8
			assertMatch(t, false, "/ab", "/a")
			// non-globs.js line 9
			assertMatch(t, false, "a/a", "a/b")
			// non-globs.js line 10
			assertMatch(t, false, "a/a", "a/c")
			// non-globs.js line 11
			assertMatch(t, false, "a/b", "a/c")
			// non-globs.js line 12
			assertMatch(t, false, "a/c", "a/b")
			// non-globs.js line 13
			assertMatch(t, false, "aaa", "aa")
			// non-globs.js line 14
			assertMatch(t, false, "ab", "/a")
			// non-globs.js line 15
			assertMatch(t, false, "ab", "a")

			// non-globs.js line 17
			assertMatch(t, true, "/a", "/a")
			// non-globs.js line 18
			assertMatch(t, true, "/a/", "/a/")
			// non-globs.js line 19
			assertMatch(t, true, "/a/a", "/a/a")
			// non-globs.js line 20
			assertMatch(t, true, "/a/a/", "/a/a/")
			// non-globs.js line 21
			assertMatch(t, true, "/a/a/a", "/a/a/a")
			// non-globs.js line 22
			assertMatch(t, true, "/a/a/a/", "/a/a/a/")
			// non-globs.js line 23
			assertMatch(t, true, "/a/a/a/a", "/a/a/a/a")
			// non-globs.js line 24
			assertMatch(t, true, "/a/a/a/a/a", "/a/a/a/a/a")

			// non-globs.js line 26
			assertMatch(t, true, "a", "a")
			// non-globs.js line 27
			assertMatch(t, true, "a/", "a/")
			// non-globs.js line 28
			assertMatch(t, true, "a/a", "a/a")
			// non-globs.js line 29
			assertMatch(t, true, "a/a/", "a/a/")
			// non-globs.js line 30
			assertMatch(t, true, "a/a/a", "a/a/a")
			// non-globs.js line 31
			assertMatch(t, true, "a/a/a/", "a/a/a/")
			// non-globs.js line 32
			assertMatch(t, true, "a/a/a/a", "a/a/a/a")
			// non-globs.js line 33
			assertMatch(t, true, "a/a/a/a/a", "a/a/a/a/a")
		})

		t.Run("should match literal dots", func(t *testing.T) {
			// non-globs.js line 37
			assertMatch(t, true, ".", ".")
			// non-globs.js line 38
			assertMatch(t, true, "..", "..")
			// non-globs.js line 39
			assertMatch(t, false, "...", "..")
			// non-globs.js line 40
			assertMatch(t, true, "...", "...")
			// non-globs.js line 41
			assertMatch(t, true, "....", "....")
			// non-globs.js line 42
			assertMatch(t, false, "....", "...")
		})

		t.Run("should handle escaped characters as literals", func(t *testing.T) {
			// non-globs.js line 46
			assertMatch(t, false, "abc", "abc\\*")
			// non-globs.js line 47
			assertMatch(t, true, "abc*", "abc\\*")
		})

		t.Run("should match windows paths", func(t *testing.T) {
			// non-globs.js line 51
			assertMatch(t, true, "aaa\\bbb", "aaa/bbb", &Options{Windows: true})
			// non-globs.js line 52
			assertMatch(t, true, "aaa/bbb", "aaa/bbb", &Options{Windows: true})
		})
	})
}
