// braces_test.go — Faithful 1:1 port of picomatch/test/braces.js
package picomatch

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// fillRangeToRegex emulates fill-range with { toRegex: true } for numeric and alpha ranges.
// Given a start and end value, it produces a regex alternation string wrapped in parens.
// For example: fillRangeToRegex("4", "10") => "(4|5|6|7|8|9|10)"
//              fillRangeToRegex("a", "c")  => "(a|b|c)"
//              fillRangeToRegex("0", "5")  => "([0-5])"
// This is a simplified version of the fill-range npm package's toRegex mode.
// JS source: braces.js line 193 — const expandRange = (a, b) => `(${fill(a, b, { toRegex: true })})`
func fillRangeToRegex(a, b string) string {
	// Try numeric first
	aNum, errA := strconv.Atoi(a)
	bNum, errB := strconv.Atoi(b)
	if errA == nil && errB == nil {
		// Numeric range
		if aNum > bNum {
			aNum, bNum = bNum, aNum
		}
		parts := make([]string, 0, bNum-aNum+1)
		for i := aNum; i <= bNum; i++ {
			parts = append(parts, strconv.Itoa(i))
		}
		return "(" + strings.Join(parts, "|") + ")"
	}

	// Alpha range (single chars)
	if len(a) == 1 && len(b) == 1 {
		aChar := rune(a[0])
		bChar := rune(b[0])
		if aChar > bChar {
			aChar, bChar = bChar, aChar
		}
		parts := make([]string, 0, int(bChar-aChar)+1)
		for c := aChar; c <= bChar; c++ {
			parts = append(parts, string(c))
		}
		return "(" + strings.Join(parts, "|") + ")"
	}

	return fmt.Sprintf("(%s|%s)", a, b)
}

// testExpandRange creates an ExpandRange callback for testing that uses fillRangeToRegex.
// This matches the JS: const expandRange = (a, b) => `(${fill(a, b, { toRegex: true })})`
// JS source: braces.js line 193
func testExpandRange(args []string, opts *Options) string {
	if len(args) < 2 {
		return ""
	}
	return fillRangeToRegex(args[0], args[1])
}

func TestBraces(t *testing.T) {
	nobrace := &Options{Nobrace: true}

	t.Run("should not match with brace patterns when disabled", func(t *testing.T) {
		// braces.js line 10
		assertMatchList(t, []string{"a", "b", "c"}, "{a,b,c,d}", []string{"a", "b", "c"})
		// braces.js line 11
		assertMatchList(t, []string{"a", "b", "c"}, "{a,b,c,d}", []string{}, nobrace)
		// braces.js line 12
		assertMatchList(t, []string{"1", "2", "3"}, "{1..2}", []string{}, nobrace)
		// braces.js line 13
		assertMatch(t, false, "a/a", "a/{a,b}", nobrace)
		// braces.js line 14
		assertMatch(t, false, "a/b", "a/{a,b}", nobrace)
		// braces.js line 15
		assertMatch(t, false, "a/c", "a/{a,b}", nobrace)
		// braces.js line 16
		assertMatch(t, false, "b/b", "a/{a,b}", nobrace)
		// braces.js line 17
		assertMatch(t, false, "b/b", "a/{a,b,c}", nobrace)
		// braces.js line 18
		assertMatch(t, false, "a/c", "a/{a,b,c}", nobrace)
		// braces.js line 19
		assertMatch(t, false, "a/a", "a/{a..c}", nobrace)
		// braces.js line 20
		assertMatch(t, false, "a/b", "a/{a..c}", nobrace)
		// braces.js line 21
		assertMatch(t, false, "a/c", "a/{a..c}", nobrace)
	})

	t.Run("should treat single-set braces as literals", func(t *testing.T) {
		// braces.js line 25
		assertMatch(t, true, "a {abc} b", "a {abc} b")
		// braces.js line 26
		assertMatch(t, true, "a {a-b-c} b", "a {a-b-c} b")
		// braces.js line 27
		assertMatch(t, true, "a {a.c} b", "a {a.c} b")
	})

	t.Run("should match literal braces when escaped", func(t *testing.T) {
		// braces.js line 31
		assertMatch(t, true, "a {1,2}", "a \\{1,2\\}")
		// braces.js line 32
		assertMatch(t, true, "a {a..b}", "a \\{a..b\\}")
	})

	t.Run("should match using brace patterns", func(t *testing.T) {
		// braces.js line 36
		assertMatch(t, false, "a/c", "a/{a,b}")
		// braces.js line 37
		assertMatch(t, false, "b/b", "a/{a,b,c}")
		// braces.js line 38
		assertMatch(t, false, "b/b", "a/{a,b}")
		// braces.js line 39
		assertMatch(t, true, "a/a", "a/{a,b}")
		// braces.js line 40
		assertMatch(t, true, "a/b", "a/{a,b}")
		// braces.js line 41
		assertMatch(t, true, "a/c", "a/{a,b,c}")
	})

	t.Run("should support brace ranges", func(t *testing.T) {
		// braces.js line 45
		assertMatch(t, true, "a/a", "a/{a..c}")
		// braces.js line 46
		assertMatch(t, true, "a/b", "a/{a..c}")
		// braces.js line 47
		assertMatch(t, true, "a/c", "a/{a..c}")
	})

	t.Run("should support Kleene stars", func(t *testing.T) {
		// braces.js line 51
		assertMatch(t, true, "ab", "{ab,c}*")
		// braces.js line 52
		assertMatch(t, true, "abab", "{ab,c}*")
		// braces.js line 53
		assertMatch(t, true, "abc", "{ab,c}*")
		// braces.js line 54
		assertMatch(t, true, "c", "{ab,c}*")
		// braces.js line 55
		assertMatch(t, true, "cab", "{ab,c}*")
		// braces.js line 56
		assertMatch(t, true, "cc", "{ab,c}*")
		// braces.js line 57
		assertMatch(t, true, "ababab", "{ab,c}*")
		// braces.js line 58
		assertMatch(t, true, "ababc", "{ab,c}*")
		// braces.js line 59
		assertMatch(t, true, "abcab", "{ab,c}*")
		// braces.js line 60
		assertMatch(t, true, "abcc", "{ab,c}*")
		// braces.js line 61
		assertMatch(t, true, "cabab", "{ab,c}*")
		// braces.js line 62
		assertMatch(t, true, "cabc", "{ab,c}*")
		// braces.js line 63
		assertMatch(t, true, "ccab", "{ab,c}*")
		// braces.js line 64
		assertMatch(t, true, "ccc", "{ab,c}*")
	})

	t.Run("should not convert braces inside brackets", func(t *testing.T) {
		// braces.js line 68
		assertMatch(t, true, "foo{}baz", "foo[{a,b}]+baz")
		// braces.js line 69
		assertMatch(t, true, "{a}{b}{c}", "[abc{}]+")
	})

	t.Run("should support braces containing slashes", func(t *testing.T) {
		// braces.js line 73
		assertMatch(t, true, "a", "{/,}a/**")
		// braces.js line 74
		assertMatch(t, true, "aa.txt", "a{a,b/}*.txt")
		// braces.js line 75
		assertMatch(t, true, "ab/.txt", "a{a,b/}*.txt")
		// braces.js line 76
		assertMatch(t, true, "ab/a.txt", "a{a,b/}*.txt")
		// braces.js line 77
		assertMatch(t, true, "a/", "a/**{/,}")
		// braces.js line 78
		assertMatch(t, true, "a/a", "a/**{/,}")
		// braces.js line 79
		assertMatch(t, true, "a/a/", "a/**{/,}")
	})

	t.Run("should support braces with empty elements", func(t *testing.T) {
		// braces.js line 83
		assertMatch(t, false, "abc.txt", "a{,b}.txt")
		// braces.js line 84
		assertMatch(t, false, "abc.txt", "a{a,b,}.txt")
		// braces.js line 85
		assertMatch(t, false, "abc.txt", "a{b,}.txt")
		// braces.js line 86
		assertMatch(t, true, "a.txt", "a{,b}.txt")
		// braces.js line 87
		assertMatch(t, true, "a.txt", "a{b,}.txt")
		// braces.js line 88
		assertMatch(t, true, "aa.txt", "a{a,b,}.txt")
		// braces.js line 89
		assertMatch(t, true, "aa.txt", "a{a,b,}.txt")
		// braces.js line 90
		assertMatch(t, true, "ab.txt", "a{,b}.txt")
		// braces.js line 91
		assertMatch(t, true, "ab.txt", "a{b,}.txt")
	})

	t.Run("should support braces with slashes and empty elements", func(t *testing.T) {
		// braces.js line 95
		assertMatch(t, true, "a.txt", "a{,/}*.txt")
		// braces.js line 96
		assertMatch(t, true, "ab.txt", "a{,/}*.txt")
		// braces.js line 97
		assertMatch(t, true, "a/b.txt", "a{,/}*.txt")
		// braces.js line 98
		assertMatch(t, true, "a/ab.txt", "a{,/}*.txt")
	})

	t.Run("should support braces with stars", func(t *testing.T) {
		// braces.js line 102
		assertMatch(t, true, "a.txt", "a{,.*{foo,db},\\(bar\\)}.txt")
		// braces.js line 103
		assertMatch(t, false, "adb.txt", "a{,.*{foo,db},\\(bar\\)}.txt")
		// braces.js line 104
		assertMatch(t, true, "a.db.txt", "a{,.*{foo,db},\\(bar\\)}.txt")

		// braces.js line 106
		assertMatch(t, true, "a.txt", "a{,*.{foo,db},\\(bar\\)}.txt")
		// braces.js line 107
		assertMatch(t, false, "adb.txt", "a{,*.{foo,db},\\(bar\\)}.txt")
		// braces.js line 108
		assertMatch(t, true, "a.db.txt", "a{,*.{foo,db},\\(bar\\)}.txt")

		// braces.js line 110
		assertMatch(t, true, "a", "a{,.*{foo,db},\\(bar\\)}")
		// braces.js line 111
		assertMatch(t, false, "adb", "a{,.*{foo,db},\\(bar\\)}")
		// braces.js line 112
		assertMatch(t, true, "a.db", "a{,.*{foo,db},\\(bar\\)}")

		// braces.js line 114
		assertMatch(t, true, "a", "a{,*.{foo,db},\\(bar\\)}")
		// braces.js line 115
		assertMatch(t, false, "adb", "a{,*.{foo,db},\\(bar\\)}")
		// braces.js line 116
		assertMatch(t, true, "a.db", "a{,*.{foo,db},\\(bar\\)}")

		// braces.js line 118
		assertMatch(t, false, "a", "{,.*{foo,db},\\(bar\\)}")
		// braces.js line 119
		assertMatch(t, false, "adb", "{,.*{foo,db},\\(bar\\)}")
		// braces.js line 120
		assertMatch(t, false, "a.db", "{,.*{foo,db},\\(bar\\)}")
		// braces.js line 121
		assertMatch(t, true, ".db", "{,.*{foo,db},\\(bar\\)}")

		// braces.js line 123
		assertMatch(t, false, "a", "{,*.{foo,db},\\(bar\\)}")
		// braces.js line 124
		assertMatch(t, true, "a", "{*,*.{foo,db},\\(bar\\)}")
		// braces.js line 125
		assertMatch(t, false, "adb", "{,*.{foo,db},\\(bar\\)}")
		// braces.js line 126
		assertMatch(t, true, "a.db", "{,*.{foo,db},\\(bar\\)}")
	})

	t.Run("should support braces in patterns with globstars", func(t *testing.T) {
		// braces.js line 130
		assertMatch(t, false, "a/b/c/xyz.md", "a/b/**/c{d,e}/**/xyz.md")
		// braces.js line 131
		assertMatch(t, false, "a/b/d/xyz.md", "a/b/**/c{d,e}/**/xyz.md")
		// braces.js line 132
		assertMatch(t, true, "a/b/cd/xyz.md", "a/b/**/c{d,e}/**/xyz.md")
		// braces.js line 133
		assertMatch(t, true, "a/b/c/xyz.md", "a/b/**/{c,d,e}/**/xyz.md")
		// braces.js line 134
		assertMatch(t, true, "a/b/d/xyz.md", "a/b/**/{c,d,e}/**/xyz.md")
	})

	t.Run("should support braces with globstars slashes and empty elements", func(t *testing.T) {
		// braces.js line 138
		assertMatch(t, true, "a.txt", "a{,/**/}*.txt")
		// braces.js line 139
		assertMatch(t, true, "a/b.txt", "a{,/**/,/}*.txt")
		// braces.js line 140
		assertMatch(t, true, "a/x/y.txt", "a{,/**/}*.txt")
		// braces.js line 141
		assertMatch(t, false, "a/x/y/z", "a{,/**/}*.txt")
	})

	t.Run("should support braces with globstars and empty elements", func(t *testing.T) {
		// braces.js line 145
		assertMatch(t, true, "a/b/foo/bar/baz.qux", "a/b{,/**}/bar{,/**}/*.*")
		// braces.js line 146
		assertMatch(t, true, "a/b/bar/baz.qux", "a/b{,/**}/bar{,/**}/*.*")
	})

	t.Run("should support Kleene plus", func(t *testing.T) {
		// braces.js line 150
		assertMatch(t, true, "ab", "{ab,c}+")
		// braces.js line 151
		assertMatch(t, true, "abab", "{ab,c}+")
		// braces.js line 152
		assertMatch(t, true, "abc", "{ab,c}+")
		// braces.js line 153
		assertMatch(t, true, "c", "{ab,c}+")
		// braces.js line 154
		assertMatch(t, true, "cab", "{ab,c}+")
		// braces.js line 155
		assertMatch(t, true, "cc", "{ab,c}+")
		// braces.js line 156
		assertMatch(t, true, "ababab", "{ab,c}+")
		// braces.js line 157
		assertMatch(t, true, "ababc", "{ab,c}+")
		// braces.js line 158
		assertMatch(t, true, "abcab", "{ab,c}+")
		// braces.js line 159
		assertMatch(t, true, "abcc", "{ab,c}+")
		// braces.js line 160
		assertMatch(t, true, "cabab", "{ab,c}+")
		// braces.js line 161
		assertMatch(t, true, "cabc", "{ab,c}+")
		// braces.js line 162
		assertMatch(t, true, "ccab", "{ab,c}+")
		// braces.js line 163
		assertMatch(t, true, "ccc", "{ab,c}+")
		// braces.js line 164
		assertMatch(t, true, "ccc", "{a,b,c}+")

		// braces.js line 166
		assertMatch(t, true, "a", "{a,b,c}+")
		// braces.js line 167
		assertMatch(t, true, "b", "{a,b,c}+")
		// braces.js line 168
		assertMatch(t, true, "c", "{a,b,c}+")
		// braces.js line 169
		assertMatch(t, true, "aa", "{a,b,c}+")
		// braces.js line 170
		assertMatch(t, true, "ab", "{a,b,c}+")
		// braces.js line 171
		assertMatch(t, true, "ac", "{a,b,c}+")
		// braces.js line 172
		assertMatch(t, true, "ba", "{a,b,c}+")
		// braces.js line 173
		assertMatch(t, true, "bb", "{a,b,c}+")
		// braces.js line 174
		assertMatch(t, true, "bc", "{a,b,c}+")
		// braces.js line 175
		assertMatch(t, true, "ca", "{a,b,c}+")
		// braces.js line 176
		assertMatch(t, true, "cb", "{a,b,c}+")
		// braces.js line 177
		assertMatch(t, true, "cc", "{a,b,c}+")
		// braces.js line 178
		assertMatch(t, true, "aaa", "{a,b,c}+")
		// braces.js line 179
		assertMatch(t, true, "aab", "{a,b,c}+")
		// braces.js line 180
		assertMatch(t, true, "abc", "{a,b,c}+")
	})

	t.Run("should support braces", func(t *testing.T) {
		// braces.js line 184
		assertMatch(t, true, "a", "{a,b,c}")
		// braces.js line 185
		assertMatch(t, true, "b", "{a,b,c}")
		// braces.js line 186
		assertMatch(t, true, "c", "{a,b,c}")
		// braces.js line 187
		assertMatch(t, false, "aa", "{a,b,c}")
		// braces.js line 188
		assertMatch(t, false, "bb", "{a,b,c}")
		// braces.js line 189
		assertMatch(t, false, "cc", "{a,b,c}")
	})

	t.Run("should match special chars and expand ranges in parentheses", func(t *testing.T) {
		// JS source: braces.js line 193 — const expandRange = (a, b) => `(${fill(a, b, { toRegex: true })})`
		er := &Options{ExpandRange: testExpandRange}

		// braces.js line 195
		assertMatch(t, false, "foo/bar - 1", "*/* {4..10}", er)
		// braces.js line 196
		assertMatch(t, false, "foo/bar - copy (1)", "*/* - * \\({4..10}\\)", er)
		// braces.js line 197
		assertMatch(t, false, "foo/bar (1)", "*/* \\({4..10}\\)", er)
		// braces.js line 198
		assertMatch(t, true, "foo/bar (4)", "*/* \\({4..10}\\)", er)
		// braces.js line 199
		assertMatch(t, true, "foo/bar (7)", "*/* \\({4..10}\\)", er)
		// braces.js line 200
		assertMatch(t, false, "foo/bar (42)", "*/* \\({4..10}\\)", er)
		// braces.js line 201
		assertMatch(t, true, "foo/bar (42)", "*/* \\({4..43}\\)", er)
		// braces.js line 202
		assertMatch(t, true, "foo/bar - copy [1]", "*/* \\[{0..5}\\]", er)
		// braces.js line 203
		assertMatch(t, true, "foo/bar - foo + bar - copy [1]", "*/* \\[{0..5}\\]", er)
		// braces.js line 204
		assertMatch(t, false, "foo/bar - 1", "*/* \\({4..10}\\)", er)
		// braces.js line 205
		assertMatch(t, false, "foo/bar - copy (1)", "*/* \\({4..10}\\)", er)
		// braces.js line 206
		assertMatch(t, false, "foo/bar (1)", "*/* \\({4..10}\\)", er)
		// braces.js line 207
		assertMatch(t, true, "foo/bar (4)", "*/* \\({4..10}\\)", er)
		// braces.js line 208
		assertMatch(t, true, "foo/bar (7)", "*/* \\({4..10}\\)", er)
		// braces.js line 209
		assertMatch(t, false, "foo/bar (42)", "*/* \\({4..10}\\)", er)
		// braces.js line 210
		assertMatch(t, false, "foo/bar - copy [1]", "*/* \\({4..10}\\)", er)
		// braces.js line 211
		assertMatch(t, false, "foo/bar - foo + bar - copy [1]", "*/* \\({4..10}\\)", er)
	})
}
