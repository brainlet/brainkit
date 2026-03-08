// options_ignore_test.go — Faithful 1:1 port of picomatch/test/options.ignore.js
package picomatch

import (
	"sort"
	"testing"
)

func TestOptionsIgnore(t *testing.T) {
	t.Run("options.ignore", func(t *testing.T) {
		t.Run("should not match ignored patterns", func(t *testing.T) {
			// options.ignore.js line 9
			assertMatch(t, true, "a+b/src/glimini.js", "a+b/src/*.js", &Options{Ignore: []string{"**/f*"}})
			// options.ignore.js line 10
			assertMatch(t, false, "a+b/src/glimini.js", "a+b/src/*.js", &Options{Ignore: []string{"**/g*"}})
			// options.ignore.js line 11
			assertMatch(t, true, "+b/src/glimini.md", "+b/src/*", &Options{Ignore: []string{"**/*.js"}})
			// options.ignore.js line 12
			assertMatch(t, false, "+b/src/glimini.js", "+b/src/*", &Options{Ignore: []string{"**/*.js"}})
		})

		// options.ignore.js line 15
		negations := []string{"a/a", "a/b", "a/c", "a/d", "a/e", "b/a", "b/b", "b/c"}

		// options.ignore.js line 16
		globs := []string{".a", ".a/a", ".a/a/a", ".a/a/a/a", "a", "a/.a", "a/a", "a/a/.a", "a/a/a", "a/a/a/a", "a/a/a/a/a", "a/a/b", "a/b", "a/b/c", "a/c", "a/x", "b", "b/b/b", "b/b/c", "c/c/c", "e/f/g", "h/i/a", "x/x/x", "x/y", "z/z", "z/z/z"}
		sort.Strings(globs)

		t.Run("should filter out ignored patterns", func(t *testing.T) {
			opts := &Options{Ignore: []string{"a/**"}, StrictSlashes: true}
			dotOpts := &Options{Ignore: []string{"a/**"}, StrictSlashes: true, Dot: true}

			// options.ignore.js line 22
			assertMatchList(t, globs, "*", []string{"a", "b"}, opts)
			// options.ignore.js line 23
			assertMatchList(t, globs, "*", []string{"b"}, &Options{Ignore: []string{"a/**"}, StrictSlashes: false})
			// options.ignore.js line 24
			assertMatchList(t, globs, "*", []string{"b"}, &Options{Ignore: "**/a"})
			// options.ignore.js line 25
			assertMatchList(t, globs, "*/*", []string{"x/y", "z/z"}, opts)
			// options.ignore.js line 26
			assertMatchList(t, globs, "*/*/*", []string{"b/b/b", "b/b/c", "c/c/c", "e/f/g", "h/i/a", "x/x/x", "z/z/z"}, opts)
			// options.ignore.js line 27
			assertMatchList(t, globs, "*/*/*/*", []string{}, opts)
			// options.ignore.js line 28
			assertMatchList(t, globs, "*/*/*/*/*", []string{}, opts)
			// options.ignore.js line 29
			assertMatchList(t, globs, "a/*", []string{}, opts)
			// options.ignore.js line 30
			assertMatchList(t, globs, "**/*/x", []string{"x/x/x"}, opts)
			// options.ignore.js line 31
			assertMatchList(t, globs, "**/*/[b-z]", []string{"b/b/b", "b/b/c", "c/c/c", "e/f/g", "x/x/x", "x/y", "z/z", "z/z/z"}, opts)

			// options.ignore.js line 33
			assertMatchList(t, globs, "*", []string{".a", "b"}, &Options{Ignore: "**/a", Dot: true})
			// options.ignore.js line 34
			assertMatchList(t, globs, "*", []string{".a", "a", "b"}, dotOpts)

			// options.ignore.js line 35
			dotResult := match(globs, "*/*", dotOpts)
			sort.Strings(dotResult)
			dotExpected := []string{".a/a", "x/y", "z/z"}
			sort.Strings(dotExpected)
			assertMatchList(t, globs, "*/*", dotExpected, dotOpts)

			// options.ignore.js line 36
			dot3Result := match(globs, "*/*/*", dotOpts)
			sort.Strings(dot3Result)
			dot3Expected := []string{".a/a/a", "b/b/b", "b/b/c", "c/c/c", "e/f/g", "h/i/a", "x/x/x", "z/z/z"}
			sort.Strings(dot3Expected)
			assertMatchList(t, globs, "*/*/*", dot3Expected, dotOpts)

			// options.ignore.js line 37
			assertMatchList(t, globs, "*/*/*/*", []string{".a/a/a/a"}, dotOpts)
			// options.ignore.js line 38
			assertMatchList(t, globs, "*/*/*/*/*", []string{}, dotOpts)
			// options.ignore.js line 39
			assertMatchList(t, globs, "a/*", []string{}, dotOpts)
			// options.ignore.js line 40
			assertMatchList(t, globs, "**/*/x", []string{"x/x/x"}, dotOpts)

			// options.ignore.js line 43 — see https://github.com/jonschlinkert/micromatch/issues/79
			assertMatchList(t, []string{"foo.js", "a/foo.js"}, "**/foo.js", []string{"foo.js", "a/foo.js"})
			// options.ignore.js line 44
			assertMatchList(t, []string{"foo.js", "a/foo.js"}, "**/foo.js", []string{"foo.js", "a/foo.js"}, &Options{Dot: true})

			// options.ignore.js line 46
			assertMatchList(t, negations, "!b/a", []string{"b/b", "b/c"}, opts)
			// options.ignore.js line 47
			assertMatchList(t, negations, "!b/(a)", []string{"b/b", "b/c"}, opts)
			// options.ignore.js line 48
			assertMatchList(t, negations, "!(b/(a))", []string{"b/b", "b/c"}, opts)
			// options.ignore.js line 49
			assertMatchList(t, negations, "!(b/a)", []string{"b/b", "b/c"}, opts)

			// options.ignore.js line 51
			assertMatchList(t, negations, "**", negations)
			// options.ignore.js line 52
			assertMatchList(t, negations, "**", []string{"a/c", "a/d", "a/e", "b/c"}, &Options{Ignore: []string{"*/b", "*/a"}})
			// options.ignore.js line 53
			assertMatchList(t, negations, "**", []string{}, &Options{Ignore: []string{"**"}})
		})
	})
}
