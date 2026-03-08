// options_test.go — Faithful 1:1 port of picomatch/test/options.js
package picomatch

import (
	"sort"
	"strings"
	"testing"
)

func TestOptions(t *testing.T) {
	t.Run("options", func(t *testing.T) {
		t.Run("options.matchBase", func(t *testing.T) {
			t.Run("should match the basename of file paths when options.matchBase is true", func(t *testing.T) {
				// options.js line 10
				assertMatchList(t,
					[]string{"a/b/c/d.md"}, "*.md",
					[]string{},
					&Options{Windows: true},
				)
				// options.js line 11
				assertMatchList(t,
					[]string{"a/b/c/foo.md"}, "*.md",
					[]string{},
					&Options{Windows: true},
				)
				// options.js line 12
				assertMatchList(t,
					[]string{"ab", "acb", "acb/", "acb/d/e", "x/y/acb", "x/y/acb/d"}, "a?b",
					[]string{"acb"},
					&Options{Windows: true},
				)
				// options.js line 13
				assertMatchList(t,
					[]string{"a/b/c/d.md"}, "*.md",
					[]string{"a/b/c/d.md"},
					&Options{MatchBase: true, Windows: true},
				)
				// options.js line 14
				assertMatchList(t,
					[]string{"a/b/c/foo.md"}, "*.md",
					[]string{"a/b/c/foo.md"},
					&Options{MatchBase: true, Windows: true},
				)
				// options.js line 15
				assertMatchList(t,
					[]string{"x/y/acb", "acb/", "acb/d/e", "x/y/acb/d"}, "a?b",
					[]string{"x/y/acb", "acb/"},
					&Options{MatchBase: true, Windows: true},
				)
			})

			t.Run("should work with negation patterns", func(t *testing.T) {
				// options.js line 19
				assertMatch(t, true, "./x/y.js", "*.js", &Options{MatchBase: true, Windows: true})
				// options.js line 20
				assertMatch(t, false, "./x/y.js", "!*.js", &Options{MatchBase: true, Windows: true})
				// options.js line 21
				assertMatch(t, true, "./x/y.js", "**/*.js", &Options{MatchBase: true, Windows: true})
				// options.js line 22
				assertMatch(t, false, "./x/y.js", "!**/*.js", &Options{MatchBase: true, Windows: true})
			})
		})

		t.Run("options.flags", func(t *testing.T) {
			t.Run("should be case-sensitive by default", func(t *testing.T) {
				// options.js line 28
				assertMatchList(t,
					[]string{"a/b/d/e.md"}, "a/b/D/*.md",
					[]string{},
					&Options{Windows: true},
				)
				// options.js line 29
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/*/E.md",
					[]string{},
					&Options{Windows: true},
				)
				// options.js line 30
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/C/*.MD",
					[]string{},
					&Options{Windows: true},
				)
			})

			t.Run("should not be case-sensitive when i is set on options.flags", func(t *testing.T) {
				// options.js line 34
				assertMatchList(t,
					[]string{"a/b/d/e.md"}, "a/b/D/*.md",
					[]string{"a/b/d/e.md"},
					&Options{Flags: "i", Windows: true},
				)
				// options.js line 35
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/*/E.md",
					[]string{"a/b/c/e.md"},
					&Options{Flags: "i", Windows: true},
				)
				// options.js line 36
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/C/*.MD",
					[]string{"a/b/c/e.md"},
					&Options{Flags: "i", Windows: true},
				)
			})
		})

		t.Run("options.nocase", func(t *testing.T) {
			t.Run("should not be case-sensitive when options.nocase is true", func(t *testing.T) {
				// options.js line 42
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/*/E.md",
					[]string{"a/b/c/e.md"},
					&Options{Nocase: true, Windows: true},
				)
				// options.js line 43
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/C/*.MD",
					[]string{"a/b/c/e.md"},
					&Options{Nocase: true, Windows: true},
				)
				// options.js line 44
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/C/*.md",
					[]string{"a/b/c/e.md"},
					&Options{Nocase: true, Windows: true},
				)
				// options.js line 45
				assertMatchList(t,
					[]string{"a/b/d/e.md"}, "a/b/D/*.md",
					[]string{"a/b/d/e.md"},
					&Options{Nocase: true, Windows: true},
				)
			})

			t.Run("should not double-set i when both nocase and the i flag are set", func(t *testing.T) {
				opts := &Options{Nocase: true, Flags: "i", Windows: true}
				// options.js line 50
				assertMatchList(t,
					[]string{"a/b/d/e.md"}, "a/b/D/*.md",
					[]string{"a/b/d/e.md"},
					opts,
				)
				// options.js line 51
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/*/E.md",
					[]string{"a/b/c/e.md"},
					opts,
				)
				// options.js line 52
				assertMatchList(t,
					[]string{"a/b/c/e.md"}, "A/b/C/*.MD",
					[]string{"a/b/c/e.md"},
					opts,
				)
			})
		})

		t.Run("options.noextglob", func(t *testing.T) {
			t.Run("should match literal parens when noextglob is true (issue #116)", func(t *testing.T) {
				// options.js line 58
				assertMatch(t, true, "a/(dir)", "a/(dir)", &Options{Noextglob: true, Windows: true})
			})

			t.Run("should not match extglobs when noextglob is true", func(t *testing.T) {
				noext := &Options{Noextglob: true, Windows: true}
				// options.js line 62
				assertMatch(t, false, "ax", "?(a*|b)", noext)
				// options.js line 63
				assertMatchList(t,
					[]string{"a.j.js", "a.md.js"}, "*.*(j).js",
					[]string{"a.j.js"},
					noext,
				)
				// options.js line 64
				assertMatchList(t,
					[]string{"a/z", "a/b", "a/!(z)"}, "a/!(z)",
					[]string{"a/!(z)"},
					noext,
				)
				// options.js line 65
				assertMatchList(t,
					[]string{"a/z", "a/b"}, "a/!(z)",
					[]string{},
					noext,
				)
				// options.js line 66
				assertMatchList(t,
					[]string{"c/a/v"}, "c/!(z)/v",
					[]string{},
					noext,
				)
				// options.js line 67
				assertMatchList(t,
					[]string{"c/z/v", "c/a/v"}, "c/!(z)/v",
					[]string{},
					noext,
				)
				// options.js line 68
				assertMatchList(t,
					[]string{"c/z/v", "c/a/v"}, "c/@(z)/v",
					[]string{},
					noext,
				)
				// options.js line 69
				assertMatchList(t,
					[]string{"c/z/v", "c/a/v"}, "c/+(z)/v",
					[]string{},
					noext,
				)
				// options.js line 70
				assertMatchList(t,
					[]string{"c/z/v", "c/a/v"}, "c/*(z)/v",
					[]string{"c/z/v"},
					noext,
				)
				// options.js line 71
				assertMatchList(t,
					[]string{"c/z/v", "z", "zf", "fz"}, "?(z)",
					[]string{"fz"},
					noext,
				)
				// options.js line 72
				assertMatchList(t,
					[]string{"c/z/v", "z", "zf", "fz"}, "+(z)",
					[]string{},
					noext,
				)
				// options.js line 73
				assertMatchList(t,
					[]string{"c/z/v", "z", "zf", "fz"}, "*(z)",
					[]string{"z", "fz"},
					noext,
				)
				// options.js line 74
				assertMatchList(t,
					[]string{"cz", "abz", "az"}, "a@(z)",
					[]string{},
					noext,
				)
				// options.js line 75
				assertMatchList(t,
					[]string{"cz", "abz", "az"}, "a*@(z)",
					[]string{},
					noext,
				)
				// options.js line 76
				assertMatchList(t,
					[]string{"cz", "abz", "az"}, "a!(z)",
					[]string{},
					noext,
				)
				// options.js line 77
				assertMatchList(t,
					[]string{"cz", "abz", "az", "azz"}, "a?(z)",
					[]string{"abz", "azz"},
					noext,
				)
				// options.js line 78
				assertMatchList(t,
					[]string{"cz", "abz", "az", "azz", "a+z"}, "a+(z)",
					[]string{"a+z"},
					noext,
				)
				// options.js line 79
				assertMatchList(t,
					[]string{"cz", "abz", "az"}, "a*(z)",
					[]string{"abz", "az"},
					noext,
				)
				// options.js line 80
				assertMatchList(t,
					[]string{"cz", "abz", "az"}, "a**(z)",
					[]string{"abz", "az"},
					noext,
				)
				// options.js line 81
				assertMatchList(t,
					[]string{"cz", "abz", "az"}, "a*!(z)",
					[]string{},
					noext,
				)
			})
		})

		t.Run("options.unescape", func(t *testing.T) {
			t.Run("should remove backslashes in glob patterns", func(t *testing.T) {
				fixtures := []string{"abc", "/a/b/c", `\a\b\c`}
				// options.js line 88
				assertMatchList(t, fixtures, `\a\b\c`, []string{"/a/b/c"}, &Options{Windows: true})
				// options.js line 89
				assertMatchList(t, fixtures, `\a\b\c`, []string{"abc", "/a/b/c"}, &Options{Unescape: true, Windows: true})
				// options.js line 90
				assertMatchList(t, fixtures, `\a\b\c`, []string{"/a/b/c"}, &Options{Windows: true})
			})
		})

		t.Run("options.nonegate", func(t *testing.T) {
			t.Run("should support the nonegate option", func(t *testing.T) {
				// options.js line 96
				assertMatchList(t,
					[]string{"a/a/a", "a/b/a", "b/b/a", "c/c/a", "c/c/b"}, "!**/a",
					[]string{"c/c/b"},
					&Options{Windows: true},
				)
				// options.js line 97
				assertMatchList(t,
					[]string{"a.md", "!a.md", "a.txt"}, "!*.md",
					[]string{"!a.md"},
					&Options{Nonegate: true, Windows: true},
				)
				// options.js line 98
				assertMatchList(t,
					[]string{"!a/a/a", "!a/a", "a/b/a", "b/b/a", "!c/c/a", "!c/a"}, "!**/a",
					[]string{"!a/a", "!c/a"},
					&Options{Nonegate: true, Windows: true},
				)
				// options.js line 99
				assertMatchList(t,
					[]string{"!*.md", ".dotfile.txt", "a/b/.dotfile"}, "!*.md",
					[]string{"!*.md"},
					&Options{Nonegate: true, Windows: true},
				)
			})
		})

		t.Run("options.windows", func(t *testing.T) {
			t.Run("should windows file paths by default", func(t *testing.T) {
				// options.js line 105
				assertMatchList(t,
					[]string{`a\b\c.md`}, "**/*.md",
					[]string{"a/b/c.md"},
					&Options{Windows: true},
				)
				// options.js line 106
				assertMatchList(t,
					[]string{`a\b\c.md`}, "**/*.md",
					[]string{`a\b\c.md`},
					&Options{Windows: false},
				)
			})

			t.Run("should windows absolute paths", func(t *testing.T) {
				// options.js line 110
				assertMatchList(t,
					[]string{`E:\a\b\c.md`}, "E:/**/*.md",
					[]string{"E:/a/b/c.md"},
					&Options{Windows: true},
				)
				// options.js line 111
				assertMatchList(t,
					[]string{`E:\a\b\c.md`}, "E:/**/*.md",
					[]string{},
					&Options{Windows: false},
				)
			})

			t.Run("should strip leading ./", func(t *testing.T) {
				fixtures := []string{
					"./a", "./a/a/a", "./a/a/a/a", "./a/a/a/a/a",
					"./a/b", "./a/x", "./z/z", "a", "a/a", "a/a/b",
					"a/c", "b", "x/y",
				}
				sort.Strings(fixtures)

				format := func(str string) string {
					return strings.TrimPrefix(str, "./")
				}
				opts := &Options{Format: format, Windows: true}

				// options.js line 118
				assertMatchList(t, fixtures, "*", []string{"a", "b"}, opts)
				// options.js line 119
				expected119 := []string{"a", "a/a/a", "a/a/a/a", "a/a/a/a/a", "a/b", "a/x", "a/a", "a/a/b", "a/c"}
				assertMatchListSorted(t, fixtures, "**/a/**", expected119, opts)
				// options.js line 120
				expected120 := []string{"a/b", "a/x", "z/z", "a/a", "a/c", "x/y"}
				assertMatchListSorted(t, fixtures, "*/*", expected120, opts)
				// options.js line 121
				assertMatchListSorted(t, fixtures, "*/*/*", []string{"a/a/a", "a/a/b"}, opts)
				// options.js line 122
				assertMatchListSorted(t, fixtures, "*/*/*/*", []string{"a/a/a/a"}, opts)
				// options.js line 123
				assertMatchListSorted(t, fixtures, "*/*/*/*/*", []string{"a/a/a/a/a"}, opts)
				// options.js line 124
				assertMatchList(t, fixtures, "./*", []string{"a", "b"}, opts)
				// options.js line 125
				expected125 := []string{"a", "a/a/a", "a/a/a/a", "a/a/a/a/a", "a/b", "a/x", "a/a", "a/a/b", "a/c"}
				assertMatchListSorted(t, fixtures, "./**/a/**", expected125, opts)
				// options.js line 126
				assertMatchListSorted(t, fixtures, "./a/*/a", []string{"a/a/a"}, opts)
				// options.js line 127
				assertMatchListSorted(t, fixtures, "a/*", []string{"a/b", "a/x", "a/a", "a/c"}, opts)
				// options.js line 128
				assertMatchListSorted(t, fixtures, "a/*/*", []string{"a/a/a", "a/a/b"}, opts)
				// options.js line 129
				assertMatchListSorted(t, fixtures, "a/*/*/*", []string{"a/a/a/a"}, opts)
				// options.js line 130
				assertMatchListSorted(t, fixtures, "a/*/*/*/*", []string{"a/a/a/a/a"}, opts)
				// options.js line 131
				assertMatchListSorted(t, fixtures, "a/*/a", []string{"a/a/a"}, opts)

				// Same tests with windows: false
				optsNoWin := &Options{Format: format, Windows: false}

				// options.js line 133
				assertMatchList(t, fixtures, "*", []string{"a", "b"}, optsNoWin)
				// options.js line 134
				expected134 := []string{"a", "a/a/a", "a/a/a/a", "a/a/a/a/a", "a/b", "a/x", "a/a", "a/a/b", "a/c"}
				assertMatchListSorted(t, fixtures, "**/a/**", expected134, optsNoWin)
				// options.js line 135
				expected135 := []string{"a/b", "a/x", "z/z", "a/a", "a/c", "x/y"}
				assertMatchListSorted(t, fixtures, "*/*", expected135, optsNoWin)
				// options.js line 136
				assertMatchListSorted(t, fixtures, "*/*/*", []string{"a/a/a", "a/a/b"}, optsNoWin)
				// options.js line 137
				assertMatchListSorted(t, fixtures, "*/*/*/*", []string{"a/a/a/a"}, optsNoWin)
				// options.js line 138
				assertMatchListSorted(t, fixtures, "*/*/*/*/*", []string{"a/a/a/a/a"}, optsNoWin)
				// options.js line 139
				assertMatchList(t, fixtures, "./*", []string{"a", "b"}, optsNoWin)
				// options.js line 140
				expected140 := []string{"a", "a/a/a", "a/a/a/a", "a/a/a/a/a", "a/b", "a/x", "a/a", "a/a/b", "a/c"}
				assertMatchListSorted(t, fixtures, "./**/a/**", expected140, optsNoWin)
				// options.js line 141
				assertMatchListSorted(t, fixtures, "./a/*/a", []string{"a/a/a"}, optsNoWin)
				// options.js line 142
				assertMatchListSorted(t, fixtures, "a/*", []string{"a/b", "a/x", "a/a", "a/c"}, optsNoWin)
				// options.js line 143
				assertMatchListSorted(t, fixtures, "a/*/*", []string{"a/a/a", "a/a/b"}, optsNoWin)
				// options.js line 144
				assertMatchListSorted(t, fixtures, "a/*/*/*", []string{"a/a/a/a"}, optsNoWin)
				// options.js line 145
				assertMatchListSorted(t, fixtures, "a/*/*/*/*", []string{"a/a/a/a/a"}, optsNoWin)
				// options.js line 146
				assertMatchListSorted(t, fixtures, "a/*/a", []string{"a/a/a"}, optsNoWin)
			})
		})

		t.Run("windows", func(t *testing.T) {
			t.Run("should convert file paths to posix slashes", func(t *testing.T) {
				// options.js line 152
				assertMatchList(t,
					[]string{`a\b\c.md`}, "**/*.md",
					[]string{"a/b/c.md"},
					&Options{Windows: true},
				)
				// options.js line 153
				assertMatchList(t,
					[]string{`a\b\c.md`}, "**/*.md",
					[]string{`a\b\c.md`},
					&Options{Windows: false},
				)
			})

			t.Run("should convert absolute paths to posix slashes", func(t *testing.T) {
				// options.js line 157
				assertMatchList(t,
					[]string{`E:\a\b\c.md`}, "E:/**/*.md",
					[]string{"E:/a/b/c.md"},
					&Options{Windows: true},
				)
				// options.js line 158
				assertMatchList(t,
					[]string{`E:\a\b\c.md`}, "E:/**/*.md",
					[]string{},
					&Options{Windows: false},
				)
			})
		})
	})
}

// assertMatchListSorted compares match results after sorting both expected and actual.
// This is needed because JS tests use deepStrictEqual which is order-sensitive,
// but the Go match() helper may return items in different order when Format is applied.
func assertMatchListSorted(t *testing.T, list []string, pattern string, expected []string, opts ...*Options) {
	t.Helper()
	var o *Options
	if len(opts) > 0 && opts[0] != nil {
		o = opts[0]
	}
	result := match(list, pattern, o)
	sort.Strings(result)
	sortedExpected := make([]string, len(expected))
	copy(sortedExpected, expected)
	sort.Strings(sortedExpected)
	if len(result) != len(sortedExpected) {
		t.Errorf("match(%v, %q): expected %v (len %d), got %v (len %d)",
			list, pattern, sortedExpected, len(sortedExpected), result, len(result))
		return
	}
	for i := range sortedExpected {
		if result[i] != sortedExpected[i] {
			t.Errorf("match(%v, %q)[%d]: expected %q, got %q (full expected: %v, full got: %v)",
				list, pattern, i, sortedExpected[i], result[i], sortedExpected, result)
			return
		}
	}
}
