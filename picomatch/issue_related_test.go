// issue_related_test.go — Faithful 1:1 port of picomatch/test/issue-related.js
package picomatch

import (
	"testing"
)

func TestIssueRelated(t *testing.T) {
	// Ported from: picomatch/test/issue-related.js lines 6-64

	t.Run("should match with braces (see picomatch/issues#8)", func(t *testing.T) {
		// issue-related.js line 8
		assertMatch(t, true, "directory/.test.txt", "{file.txt,directory/**/*}", &Options{Dot: true})
		// issue-related.js line 9
		assertMatch(t, true, "directory/test.txt", "{file.txt,directory/**/*}", &Options{Dot: true})
		// issue-related.js line 10
		assertMatch(t, false, "directory/.test.txt", "{file.txt,directory/**/*}")
		// issue-related.js line 11
		assertMatch(t, true, "directory/test.txt", "{file.txt,directory/**/*}")
	})

	t.Run("should match Japanese characters (see micromatch/issues#127)", func(t *testing.T) {
		// issue-related.js line 15
		assertMatch(t, true, "\u30d5\u30a9\u30eb\u30c0/aaa.js", "\u30d5*/**/*")
		// issue-related.js line 16
		assertMatch(t, true, "\u30d5\u30a9\u30eb\u30c0/aaa.js", "\u30d5\u30a9*/**/*")
		// issue-related.js line 17
		assertMatch(t, true, "\u30d5\u30a9\u30eb\u30c0/aaa.js", "\u30d5\u30a9\u30eb*/**/*")
		// issue-related.js line 18
		assertMatch(t, true, "\u30d5\u30a9\u30eb\u30c0/aaa.js", "\u30d5*\u30eb*/**/*")
		// issue-related.js line 19
		assertMatch(t, true, "\u30d5\u30a9\u30eb\u30c0/aaa.js", "\u30d5\u30a9\u30eb\u30c0/**/*")
	})

	t.Run("micromatch issue#15", func(t *testing.T) {
		// issue-related.js line 23
		assertMatch(t, true, "a/b-c/d/e/z.js", "a/b-*/**/z.js")
		// issue-related.js line 24
		assertMatch(t, true, "z.js", "z*")
		// issue-related.js line 25
		assertMatch(t, true, "z.js", "**/z*")
		// issue-related.js line 26
		assertMatch(t, true, "z.js", "**/z*.js")
		// issue-related.js line 27
		assertMatch(t, true, "z.js", "**/*.js")
		// issue-related.js line 28
		assertMatch(t, true, "foo", "**/foo")
	})

	t.Run("micromatch issue#23", func(t *testing.T) {
		// issue-related.js line 32
		assertMatch(t, false, "zzjs", "z*.js")
		// issue-related.js line 33
		assertMatch(t, false, "zzjs", "*z.js")
	})

	t.Run("micromatch issue#24", func(t *testing.T) {
		// issue-related.js line 37
		assertMatch(t, false, "a/b/c/d/", "a/b/**/f")
		// issue-related.js line 38
		assertMatch(t, true, "a", "a/**")
		// issue-related.js line 39
		assertMatch(t, true, "a", "**")
		// issue-related.js line 40
		assertMatch(t, true, "a/", "**")
		// issue-related.js line 41
		assertMatch(t, true, "a/b/c/d", "**")
		// issue-related.js line 42
		assertMatch(t, true, "a/b/c/d/", "**")
		// issue-related.js line 43
		assertMatch(t, true, "a/b/c/d/", "**/**")
		// issue-related.js line 44
		assertMatch(t, true, "a/b/c/d/", "**/b/**")
		// issue-related.js line 45
		assertMatch(t, true, "a/b/c/d/", "a/b/**")
		// issue-related.js line 46
		assertMatch(t, true, "a/b/c/d/", "a/b/**/")
		// issue-related.js line 47
		assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/**/*.*")
		// issue-related.js line 48
		assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/*.*")
		// issue-related.js line 49
		assertMatch(t, true, "a/b/c/d/g/e.f", "a/b/**/d/**/*.*")
		// issue-related.js line 50
		assertMatch(t, true, "a/b/c/d/g/g/e.f", "a/b/**/d/**/*.*")
	})

	t.Run("micromatch issue#58 - only match nested dirs when ** is the only thing in a segment", func(t *testing.T) {
		// issue-related.js line 54
		assertMatch(t, false, "a/b/c", "a/b**")
		// issue-related.js line 55
		assertMatch(t, false, "a/c/b", "a/**b")
	})

	t.Run("micromatch issue#79", func(t *testing.T) {
		// issue-related.js line 59
		assertMatch(t, true, "a/foo.js", "**/foo.js")
		// issue-related.js line 60
		assertMatch(t, true, "foo.js", "**/foo.js")
		// issue-related.js line 61
		assertMatch(t, true, "a/foo.js", "**/foo.js", &Options{Dot: true})
		// issue-related.js line 62
		assertMatch(t, true, "foo.js", "**/foo.js", &Options{Dot: true})
	})
}
