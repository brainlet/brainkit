// qmarks_test.go — Faithful 1:1 port of picomatch/test/qmarks.js
package picomatch

import (
	"runtime"
	"testing"
)

func TestQmarksAndStars(t *testing.T) {
	t.Run("should match question marks with question marks", func(t *testing.T) {
		// test/qmarks.js line 9
		assertMatchList(t, []string{"?", "??", "???"}, "?", []string{"?"})
		// test/qmarks.js line 10
		assertMatchList(t, []string{"?", "??", "???"}, "??", []string{"??"})
		// test/qmarks.js line 11
		assertMatchList(t, []string{"?", "??", "???"}, "???", []string{"???"})
	})

	t.Run("should match question marks and stars with question marks and stars", func(t *testing.T) {
		// test/qmarks.js line 15
		assertMatchList(t, []string{"?", "??", "???"}, "?*", []string{"?", "??", "???"})
		// test/qmarks.js line 16
		assertMatchList(t, []string{"?", "??", "???"}, "*?", []string{"?", "??", "???"})
		// test/qmarks.js line 17
		assertMatchList(t, []string{"?", "??", "???"}, "?*?", []string{"??", "???"})
		// test/qmarks.js line 18
		assertMatchList(t, []string{"?*", "?*?", "?*?*?"}, "?*", []string{"?*", "?*?", "?*?*?"})
		// test/qmarks.js line 19
		assertMatchList(t, []string{"?*", "?*?", "?*?*?"}, "*?", []string{"?*", "?*?", "?*?*?"})
		// test/qmarks.js line 20
		assertMatchList(t, []string{"?*", "?*?", "?*?*?"}, "?*?", []string{"?*", "?*?", "?*?*?"})
	})

	t.Run("should support consecutive stars and question marks", func(t *testing.T) {
		// test/qmarks.js line 24
		assertMatchList(t, []string{"aaa", "aac", "abc"}, "a*?c", []string{"aac", "abc"})
		// test/qmarks.js line 25
		assertMatchList(t, []string{"abc", "abb", "acc"}, "a**?c", []string{"abc", "acc"})
		// test/qmarks.js line 26
		assertMatchList(t, []string{"abc", "aaaabbbbbbccccc"}, "a*****?c", []string{"abc", "aaaabbbbbbccccc"})
		// test/qmarks.js line 27
		assertMatchList(t, []string{"a", "ab", "abc", "abcd"}, "*****?", []string{"a", "ab", "abc", "abcd"})
		// test/qmarks.js line 28
		assertMatchList(t, []string{"a", "ab", "abc", "abcd"}, "*****??", []string{"ab", "abc", "abcd"})
		// test/qmarks.js line 29
		assertMatchList(t, []string{"a", "ab", "abc", "abcd"}, "?*****??", []string{"abc", "abcd"})
		// test/qmarks.js line 30
		assertMatchList(t, []string{"abc", "abb", "zzz"}, "?*****?c", []string{"abc"})
		// test/qmarks.js line 31
		assertMatchList(t, []string{"abc", "bbb", "zzz"}, "?***?****?", []string{"abc", "bbb", "zzz"})
		// test/qmarks.js line 32
		assertMatchList(t, []string{"abc", "bbb", "zzz"}, "?***?****c", []string{"abc"})
		// test/qmarks.js line 33
		assertMatchList(t, []string{"abc"}, "*******?", []string{"abc"})
		// test/qmarks.js line 34
		assertMatchList(t, []string{"abc"}, "*******c", []string{"abc"})
		// test/qmarks.js line 35
		assertMatchList(t, []string{"abc"}, "?***?****", []string{"abc"})
		// test/qmarks.js line 36
		assertMatchList(t, []string{"abcdecdhjk"}, "a****c**?**??*****", []string{"abcdecdhjk"})
		// test/qmarks.js line 37
		assertMatchList(t, []string{"abcdecdhjk"}, "a**?**cd**?**??***k", []string{"abcdecdhjk"})
		// test/qmarks.js line 38
		assertMatchList(t, []string{"abcdecdhjk"}, "a**?**cd**?**??***k**", []string{"abcdecdhjk"})
		// test/qmarks.js line 39
		assertMatchList(t, []string{"abcdecdhjk"}, "a**?**cd**?**??k", []string{"abcdecdhjk"})
		// test/qmarks.js line 40
		assertMatchList(t, []string{"abcdecdhjk"}, "a**?**cd**?**??k***", []string{"abcdecdhjk"})
		// test/qmarks.js line 41
		assertMatchList(t, []string{"abcdecdhjk"}, "a*cd**?**??k", []string{"abcdecdhjk"})
	})

	t.Run("should match backslashes with question marks when not on windows", func(t *testing.T) {
		// test/qmarks.js line 44-49
		// JS: if (process.platform !== 'win32') { ... }
		if runtime.GOOS != "windows" {
			// test/qmarks.js line 46
			assertMatch(t, false, "aaa\\\\bbb", "aaa?bbb")
			// test/qmarks.js line 47
			assertMatch(t, true, "aaa\\\\bbb", "aaa??bbb")
			// test/qmarks.js line 48
			assertMatch(t, true, "aaa\\bbb", "aaa?bbb")
		}
	})

	t.Run("should match one character per question mark", func(t *testing.T) {
		fixtures := []string{"a", "aa", "ab", "aaa", "abcdefg"}
		// test/qmarks.js line 54
		assertMatchList(t, fixtures, "?", []string{"a"})
		// test/qmarks.js line 55
		assertMatchList(t, fixtures, "??", []string{"aa", "ab"})
		// test/qmarks.js line 56
		assertMatchList(t, fixtures, "???", []string{"aaa"})
		// test/qmarks.js line 57
		assertMatchList(t, []string{"a/", "/a/", "/a/b/", "/a/b/c/", "/a/b/c/d/"}, "??", []string{})
		// test/qmarks.js line 58
		assertMatchList(t, []string{"a/b/c.md"}, "a/?/c.md", []string{"a/b/c.md"})
		// test/qmarks.js line 59
		assertMatchList(t, []string{"a/bb/c.md"}, "a/?/c.md", []string{})
		// test/qmarks.js line 60
		assertMatchList(t, []string{"a/bb/c.md"}, "a/??/c.md", []string{"a/bb/c.md"})
		// test/qmarks.js line 61
		assertMatchList(t, []string{"a/bbb/c.md"}, "a/??/c.md", []string{})
		// test/qmarks.js line 62
		assertMatchList(t, []string{"a/bbb/c.md"}, "a/???/c.md", []string{"a/bbb/c.md"})
		// test/qmarks.js line 63
		assertMatchList(t, []string{"a/bbbb/c.md"}, "a/????/c.md", []string{"a/bbbb/c.md"})
	})

	t.Run("should not match slashes question marks", func(t *testing.T) {
		fixtures := []string{"//", "a/", "/a", "/a/", "aa", "/aa", "a/a", "aaa", "/aaa"}
		// test/qmarks.js line 68
		assertMatchList(t, fixtures, "/?", []string{"/a"})
		// test/qmarks.js line 69
		assertMatchList(t, fixtures, "/??", []string{"/aa"})
		// test/qmarks.js line 70
		assertMatchList(t, fixtures, "/???", []string{"/aaa"})
		// test/qmarks.js line 71
		assertMatchList(t, fixtures, "/?/", []string{"/a/"})
		// test/qmarks.js line 72
		assertMatchList(t, fixtures, "??", []string{"aa"})
		// test/qmarks.js line 73
		assertMatchList(t, fixtures, "?/?", []string{"a/a"})
		// test/qmarks.js line 74
		assertMatchList(t, fixtures, "???", []string{"aaa"})
		// test/qmarks.js line 75
		assertMatchList(t, fixtures, "a?a", []string{"aaa"})
		// test/qmarks.js line 76
		assertMatchList(t, fixtures, "aa?", []string{"aaa"})
		// test/qmarks.js line 77
		assertMatchList(t, fixtures, "?aa", []string{"aaa"})
	})

	t.Run("should support question marks and stars between slashes", func(t *testing.T) {
		// test/qmarks.js line 81
		assertMatchList(t, []string{"a/b.bb/c/d/efgh.ijk/e"}, "a/*/?/**/e", []string{"a/b.bb/c/d/efgh.ijk/e"})
		// test/qmarks.js line 82
		assertMatchList(t, []string{"a/b/c/d/e"}, "a/?/c/?/*/e", []string{})
		// test/qmarks.js line 83
		assertMatchList(t, []string{"a/b/c/d/e/e"}, "a/?/c/?/*/e", []string{"a/b/c/d/e/e"})
		// test/qmarks.js line 84
		assertMatchList(t, []string{"a/b/c/d/efgh.ijk/e"}, "a/*/?/**/e", []string{"a/b/c/d/efgh.ijk/e"})
		// test/qmarks.js line 85
		assertMatchList(t, []string{"a/b/c/d/efghijk/e"}, "a/*/?/**/e", []string{"a/b/c/d/efghijk/e"})
		// test/qmarks.js line 86
		assertMatchList(t, []string{"a/b/c/d/efghijk/e"}, "a/?/**/e", []string{"a/b/c/d/efghijk/e"})
		// test/qmarks.js line 87
		assertMatchList(t, []string{"a/b/c/d/efghijk/e"}, "a/?/c/?/*/e", []string{"a/b/c/d/efghijk/e"})
		// test/qmarks.js line 88
		assertMatchList(t, []string{"a/bb/e"}, "a/?/**/e", []string{})
		// test/qmarks.js line 89
		assertMatchList(t, []string{"a/bb/e"}, "a/?/e", []string{})
		// test/qmarks.js line 90
		assertMatchList(t, []string{"a/bbb/c/d/efgh.ijk/e"}, "a/*/?/**/e", []string{"a/bbb/c/d/efgh.ijk/e"})
	})

	t.Run("should match no more than one character between slashes", func(t *testing.T) {
		fixtures := []string{"a/a", "a/a/a", "a/aa/a", "a/aaa/a", "a/aaaa/a", "a/aaaaa/a"}
		// test/qmarks.js line 95
		assertMatchList(t, fixtures, "?/?", []string{"a/a"})
		// test/qmarks.js line 96
		assertMatchList(t, fixtures, "?/???/?", []string{"a/aaa/a"})
		// test/qmarks.js line 97
		assertMatchList(t, fixtures, "?/????/?", []string{"a/aaaa/a"})
		// test/qmarks.js line 98
		assertMatchList(t, fixtures, "?/?????/?", []string{"a/aaaaa/a"})
		// test/qmarks.js line 99
		assertMatchList(t, fixtures, "a/?", []string{"a/a"})
		// test/qmarks.js line 100
		assertMatchList(t, fixtures, "a/?/a", []string{"a/a/a"})
		// test/qmarks.js line 101
		assertMatchList(t, fixtures, "a/??/a", []string{"a/aa/a"})
		// test/qmarks.js line 102
		assertMatchList(t, fixtures, "a/???/a", []string{"a/aaa/a"})
		// test/qmarks.js line 103
		assertMatchList(t, fixtures, "a/????/a", []string{"a/aaaa/a"})
		// test/qmarks.js line 104
		assertMatchList(t, fixtures, "a/????a/a", []string{"a/aaaaa/a"})
	})

	t.Run("should not match non-leading dots with question marks", func(t *testing.T) {
		fixtures := []string{".", ".a", "a", "aa", "a.a", "aa.a", "aaa", "aaa.a", "aaaa.a", "aaaaa"}
		// test/qmarks.js line 109
		assertMatchList(t, fixtures, "?", []string{"a"})
		// test/qmarks.js line 110
		assertMatchList(t, fixtures, ".?", []string{".a"})
		// test/qmarks.js line 111
		assertMatchList(t, fixtures, "?a", []string{"aa"})
		// test/qmarks.js line 112
		assertMatchList(t, fixtures, "??", []string{"aa"})
		// test/qmarks.js line 113
		assertMatchList(t, fixtures, "?a?", []string{"aaa"})
		// test/qmarks.js line 114
		assertMatchList(t, fixtures, "aaa?a", []string{"aaa.a", "aaaaa"})
		// test/qmarks.js line 115
		assertMatchList(t, fixtures, "a?a?a", []string{"aaa.a", "aaaaa"})
		// test/qmarks.js line 116
		assertMatchList(t, fixtures, "a???a", []string{"aaa.a", "aaaaa"})
		// test/qmarks.js line 117
		assertMatchList(t, fixtures, "a?????", []string{"aaaa.a"})
	})

	t.Run("should match non-leading dots with question marks when options.dot is true", func(t *testing.T) {
		fixtures := []string{".", ".a", "a", "aa", "a.a", "aa.a", ".aa", "aaa.a", "aaaa.a", "aaaaa"}
		opts := &Options{Dot: true}
		// test/qmarks.js line 123
		assertMatchList(t, fixtures, "?", []string{".", "a"}, opts)
		// test/qmarks.js line 124
		assertMatchList(t, fixtures, ".?", []string{".a"}, opts)
		// test/qmarks.js line 125
		assertMatchList(t, fixtures, "?a", []string{".a", "aa"}, opts)
		// test/qmarks.js line 126
		assertMatchList(t, fixtures, "??", []string{".a", "aa"}, opts)
		// test/qmarks.js line 127
		assertMatchList(t, fixtures, "?a?", []string{".aa"}, opts)
	})
}
