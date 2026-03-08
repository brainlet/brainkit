package picomatch

// minimatch_test.go — Faithful 1:1 port of picomatch/test/minimatch.js
// Tests picomatch parity with minimatch, covering various minimatch GitHub issues.
//
// Source: https://github.com/micromatch/picomatch/blob/master/test/minimatch.js

import (
	"strings"
	"testing"
)

// format replicates the JS helper: const format = str => str.replace(/^\.\//, '');
// minimatch.js:4
func formatStripDotSlash(str string) string {
	return strings.TrimPrefix(str, "./")
}

func TestMinimatchParity(t *testing.T) {
	t.Run("minimatch issues (as of 12/7/2016)", func(t *testing.T) {

		// minimatch.js:9 — https://github.com/isaacs/minimatch/issues/29
		t.Run("issue 29", func(t *testing.T) {
			// minimatch.js:10
			assertMatch(t, true, "foo/bar.txt", "foo/**/*.txt")
			// minimatch.js:11
			re29 := MakeRe("foo/**/*.txt", nil)
			ok29, _ := re29.re.MatchString("foo/bar.txt")
			if !ok29 {
				t.Errorf("expected makeRe(%q).test(%q) to be true", "foo/**/*.txt", "foo/bar.txt")
			}
			// minimatch.js:12
			assertMatch(t, false, "n/axios/a.js", "n/!(axios)/**")
			// minimatch.js:13
			re29b := MakeRe("n/!(axios)/**", nil)
			ok29b, _ := re29b.re.MatchString("n/axios/a.js")
			if ok29b {
				t.Errorf("expected makeRe(%q).test(%q) to be false", "n/!(axios)/**", "n/axios/a.js")
			}
		})

		// minimatch.js:16 — https://github.com/isaacs/minimatch/issues/30
		t.Run("issue 30", func(t *testing.T) {
			formatOpts := &Options{
				Format: formatStripDotSlash,
			}

			// minimatch.js:17
			assertMatch(t, true, "foo/bar.js", "**/foo/**", formatOpts)
			// minimatch.js:18
			assertMatch(t, true, "./foo/bar.js", "./**/foo/**", formatOpts)
			// minimatch.js:19
			assertMatch(t, true, "./foo/bar.js", "**/foo/**", formatOpts)
			// minimatch.js:20
			assertMatch(t, true, "./foo/bar.txt", "foo/**/*.txt", formatOpts)

			// minimatch.js:21
			re30 := MakeRe("./foo/**/*.txt", nil)
			ok30, _ := re30.re.MatchString("foo/bar.txt")
			if !ok30 {
				t.Errorf("expected makeRe(%q).test(%q) to be true", "./foo/**/*.txt", "foo/bar.txt")
			}

			// minimatch.js:22
			assertMatch(t, false, "foo/bar/a.js", "./foo/!(bar)/**", formatOpts)

			// minimatch.js:23
			re30b := MakeRe("./foo/!(bar)/**", nil)
			ok30b, _ := re30b.re.MatchString("foo/bar/a.js")
			if ok30b {
				t.Errorf("expected makeRe(%q).test(%q) to be false", "./foo/!(bar)/**", "foo/bar/a.js")
			}
		})

		// minimatch.js:26 — https://github.com/isaacs/minimatch/issues/50
		t.Run("issue 50", func(t *testing.T) {
			// minimatch.js:27
			assertMatch(t, true, "foo/bar-[ABC].txt", "foo/**/*-\\[ABC\\].txt")
			// minimatch.js:28
			assertMatch(t, false, "foo/bar-[ABC].txt", "foo/**/*-\\[abc\\].txt")
			// minimatch.js:29
			assertMatch(t, true, "foo/bar-[ABC].txt", "foo/**/*-\\[abc\\].txt", &Options{Nocase: true})
		})

		// minimatch.js:32 — https://github.com/isaacs/minimatch/issues/67
		// (should work consistently with `makeRe` and matcher functions)
		t.Run("issue 67", func(t *testing.T) {
			// minimatch.js:33
			re67 := MakeRe("node_modules/foobar/**/*.bar", nil)
			// minimatch.js:34
			ok67, _ := re67.re.MatchString("node_modules/foobar/foo.bar")
			if !ok67 {
				t.Errorf("expected makeRe(%q).test(%q) to be true",
					"node_modules/foobar/**/*.bar", "node_modules/foobar/foo.bar")
			}
			// minimatch.js:35
			assertMatch(t, true, "node_modules/foobar/foo.bar", "node_modules/foobar/**/*.bar")
		})

		// minimatch.js:38 — https://github.com/isaacs/minimatch/issues/75
		t.Run("issue 75", func(t *testing.T) {
			// minimatch.js:39
			assertMatch(t, true, "foo/baz.qux.js", "foo/@(baz.qux).js")
			// minimatch.js:40
			assertMatch(t, true, "foo/baz.qux.js", "foo/+(baz.qux).js")
			// minimatch.js:41
			assertMatch(t, true, "foo/baz.qux.js", "foo/*(baz.qux).js")
			// minimatch.js:42
			assertMatch(t, false, "foo/baz.qux.js", "foo/!(baz.qux).js")
			// minimatch.js:43
			assertMatch(t, false, "foo/bar/baz.qux.js", "foo/*/!(baz.qux).js")
			// minimatch.js:44
			assertMatch(t, false, "foo/bar/bazqux.js", "**/!(bazqux).js")
			// minimatch.js:45
			assertMatch(t, false, "foo/bar/bazqux.js", "**/bar/!(bazqux).js")
			// minimatch.js:46
			assertMatch(t, false, "foo/bar/bazqux.js", "foo/**/!(bazqux).js")
			// minimatch.js:47
			assertMatch(t, false, "foo/bar/bazqux.js", "foo/**/!(bazqux)*.js")
			// minimatch.js:48
			assertMatch(t, false, "foo/bar/baz.qux.js", "foo/**/!(baz.qux)*.js")
			// minimatch.js:49
			assertMatch(t, false, "foo/bar/baz.qux.js", "foo/**/!(baz.qux).js")
			// minimatch.js:50
			assertMatch(t, false, "foobar.js", "!(foo)*.js")
			// minimatch.js:51
			assertMatch(t, false, "foo.js", "!(foo).js")
			// minimatch.js:52
			assertMatch(t, false, "foo.js", "!(foo)*.js")
		})

		// minimatch.js:55 — https://github.com/isaacs/minimatch/issues/78
		t.Run("issue 78", func(t *testing.T) {
			winOpts := &Options{Windows: true}
			// minimatch.js:56
			assertMatch(t, true, "a\\b\\c.txt", "a/**/*.txt", winOpts)
			// minimatch.js:57
			assertMatch(t, true, "a/b/c.txt", "a/**/*.txt", winOpts)
		})

		// minimatch.js:60 — https://github.com/isaacs/minimatch/issues/82
		t.Run("issue 82", func(t *testing.T) {
			formatOpts := &Options{
				Format: formatStripDotSlash,
			}
			// minimatch.js:61
			assertMatch(t, true, "./src/test/a.js", "**/test/**", formatOpts)
			// minimatch.js:62
			assertMatch(t, true, "src/test/a.js", "**/test/**")
		})

		// minimatch.js:65 — https://github.com/isaacs/minimatch/issues/83
		t.Run("issue 83", func(t *testing.T) {
			// minimatch.js:66
			re83 := MakeRe("foo/!(bar)/**", nil)
			ok83, _ := re83.re.MatchString("foo/bar/a.js")
			if ok83 {
				t.Errorf("expected makeRe(%q).test(%q) to be false", "foo/!(bar)/**", "foo/bar/a.js")
			}
			// minimatch.js:67
			assertMatch(t, false, "foo/bar/a.js", "foo/!(bar)/**")
		})
	})
}
