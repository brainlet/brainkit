// Ported from: node-ignore/test/ignore.test.js and test/others.test.js
package ignore

import (
	"runtime"
	"testing"
)

func TestUpstreamCases(t *testing.T) {
	for _, tc := range loadUpstreamCases(t) {
		tc := tc
		t.Run(tc.Description, func(t *testing.T) {
			if tc.Scopes.allows("filter") {
				t.Run("filter", func(t *testing.T) {
					ig := newIgnoreWithFixturePatterns(tc.Patterns)
					assertSameStrings(t, ig.Filter(tc.Paths), tc.Expected)
				})

				if runtime.GOOS == "windows" {
					t.Run("win32 filter", func(t *testing.T) {
						withWindowsPathMode(t, func() {
							winPaths := make([]string, len(tc.Paths))
							for i, path := range tc.Paths {
								winPaths[i] = makeWin32(path)
							}

							expected := make([]string, len(tc.Expected))
							for i, path := range tc.Expected {
								expected[i] = makeWin32(path)
							}

							ig := newIgnoreWithFixturePatterns(tc.Patterns)
							assertSameStrings(t, ig.Filter(winPaths), expected)
						})
					})
				}
			}

			if tc.Scopes.allows("createFilter") {
				t.Run("createFilter", func(t *testing.T) {
					ig := newIgnoreWithFixturePatterns(tc.Patterns)
					got := filterWithPredicate(tc.Paths, ig.CreateFilter())
					assertSameStrings(t, got, tc.Expected)
				})
			}

			if tc.Scopes.allows("ignores") {
				t.Run("ignores", func(t *testing.T) {
					ig := newIgnoreWithFixturePatterns(tc.Patterns)
					for path, expected := range tc.PathsObject {
						if got := ig.Ignores(path); got != (expected != 0) {
							t.Fatalf("Ignores(%q) = %v, want %v", path, got, expected != 0)
						}
					}
				})
			}

			if tc.Scopes.allows("checkIgnore") {
				t.Run("checkIgnore", func(t *testing.T) {
					ig := newIgnoreWithFixturePatterns(tc.Patterns)
					for path, expected := range tc.PathsObject {
						got := ig.CheckIgnore(path)
						if got.Ignored != (expected != 0) {
							t.Fatalf("CheckIgnore(%q).Ignored = %v, want %v", path, got.Ignored, expected != 0)
						}
					}
				})
			}
		})
	}
}

func TestAddIgnore(t *testing.T) {
	a := New().Add([]string{".abc/*", "!.abc/d/"})
	b := New().Add(a).Add("!.abc/e/")

	paths := []string{
		".abc/a.js",
		".abc/d/e.js",
		".abc/e/e.js",
	}

	assertSameStrings(t, a.Filter(paths), []string{".abc/d/e.js"})
	assertSameStrings(t, b.Filter(paths), []string{".abc/d/e.js", ".abc/e/e.js"})
}

type fakeRuleSource struct {
	rules []*Rule
}

func (f fakeRuleSource) IgnoreRules() []*Rule { return f.rules }

func TestAddCompatibleRuleSource(t *testing.T) {
	a := New().Add([]string{".abc/*", "!.abc/d/"})
	b := New().Add(fakeRuleSource{rules: a.IgnoreRules()}).Add("!.abc/e/")

	paths := []string{
		".abc/a.js",
		".abc/d/e.js",
		".abc/e/e.js",
	}

	assertSameStrings(t, a.Filter(paths), []string{".abc/d/e.js"})
	assertSameStrings(t, b.Filter(paths), []string{".abc/d/e.js", ".abc/e/e.js"})
}

func TestIgnoreCaseOption(t *testing.T) {
	ig := New(Options{
		IgnoreCase: Bool(false),
	})
	ig.Add("*.[jJ][pP]g")

	if !ig.Ignores("a.jpg") {
		t.Fatalf("expected a.jpg to be ignored")
	}
	if !ig.Ignores("a.JPg") {
		t.Fatalf("expected a.JPg to be ignored")
	}
	if ig.Ignores("a.JPG") {
		t.Fatalf("expected a.JPG to not be ignored")
	}
}

func TestIgnoreCacheRespectsIgnoreCase(t *testing.T) {
	rule := "*.[jJ][pP]g"

	ig := New(Options{IgnoreCase: Bool(false)})
	ig.Add(rule)
	if ig.Ignores("a.JPG") {
		t.Fatalf("expected a.JPG to not be ignored when IgnoreCase=false")
	}

	ig2 := New(Options{IgnoreCase: Bool(true)})
	ig2.Add(rule)
	if !ig2.Ignores("a.JPG") {
		t.Fatalf("expected a.JPG to be ignored when IgnoreCase=true")
	}
}

func TestInvalidPathsPanic(t *testing.T) {
	ig := New()

	assertPanicsWith(t, "path must not be empty", func() {
		ig.Ignores("")
	})
	assertPanicsWith(t, "path must be a string, but got `false`", func() {
		ig.Ignores(false)
	})
	assertPanicsWith(t, "path.relative()", func() {
		ig.Ignores("/a")
	})
	assertPanicsWith(t, "path must not be empty", func() {
		ig.Filter([]any{""})
	})
	assertPanicsWith(t, "path must not be empty", func() {
		New().CreateFilter()("")
	})

	withWindowsPathMode(t, func() {
		windowsIgnore := New()
		assertPanicsWith(t, "path.relative()", func() {
			windowsIgnore.Ignores(`c:\a`)
		})
		assertPanicsWith(t, "path.relative()", func() {
			windowsIgnore.Ignores(`C:\a`)
		})
	})
}

func TestIsPathValid(t *testing.T) {
	t.Run("posix", func(t *testing.T) {
		paths := []any{
			".",
			"./foo",
			"../foo",
			"/foo",
			false,
			"foo",
		}

		var got []any
		for _, path := range paths {
			if IsPathValid(path) {
				got = append(got, path)
			}
		}

		if len(got) != 1 || got[0] != "foo" {
			t.Fatalf("IsPathValid results = %v, want [foo]", got)
		}
	})

	if runtime.GOOS != "windows" {
		t.Run("windows", func(t *testing.T) {
			withWindowsPathMode(t, func() {
				paths := []any{
					".",
					"./foo",
					"../foo",
					"/foo",
					false,
					"foo",
					`..\foo`,
					`.\foo`,
					`\foo`,
					`\\foo`,
					`C:\foo`,
					`d:\foo`,
				}

				var got []any
				for _, path := range paths {
					if IsPathValid(path) {
						got = append(got, path)
					}
				}

				if len(got) != 1 || got[0] != "foo" {
					t.Fatalf("IsPathValid results = %v, want [foo]", got)
				}
			})
		})
	}
}

func TestBehavioralTestResults(t *testing.T) {
	cases := []struct {
		description string
		patterns    any
		path        string
		expected    TestResult
	}{
		{
			description: "no rule",
			path:        "foo",
			expected:    TestResult{Ignored: false, Unignored: false},
		},
		{
			description: "has rule, no match",
			patterns:    "bar",
			path:        "foo",
			expected:    TestResult{Ignored: false, Unignored: false},
		},
		{
			description: "only negative",
			patterns:    "!foo",
			path:        "foo",
			expected:    TestResult{Ignored: false, Unignored: true},
		},
		{
			description: "ignored then unignored",
			patterns:    []string{"foo", "!foo"},
			path:        "foo",
			expected:    TestResult{Ignored: false, Unignored: true},
		},
		{
			description: "dir ignored then unignored -> not matched",
			patterns:    []string{"foo", "!foo"},
			path:        "foo/bar",
			expected:    TestResult{Ignored: false, Unignored: false},
		},
		{
			description: "ignored by wildcard then unignored",
			patterns:    []string{"*.js", "!a/a.js"},
			path:        "a/a.js",
			expected:    TestResult{Ignored: false, Unignored: true},
		},
	}

	if runtime.GOOS != "windows" {
		cases = append(cases, struct {
			description string
			patterns    any
			path        string
			expected    TestResult
		}{
			description: "file named ...",
			path:        "...",
			expected:    TestResult{Ignored: false, Unignored: false},
		})
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			ig := New()
			if tc.patterns != nil {
				ig.Add(tc.patterns)
			}

			got := ig.Test(tc.path)
			if got.Ignored != tc.expected.Ignored || got.Unignored != tc.expected.Unignored {
				t.Fatalf("Test(%q) = %+v, want %+v", tc.path, got, tc.expected)
			}
		})
	}
}

func TestAllowRelativePaths(t *testing.T) {
	ig := New(Options{AllowRelativePaths: true})
	ig.Add("foo")
	if !ig.Ignores("../foo/bar.js") {
		t.Fatalf("expected relative parent path to be ignored when AllowRelativePaths=true")
	}

	assertPanicsWith(t, "path.relative()", func() {
		New().Ignores("../foo/bar.js")
	})
}

func TestAllowRelativePathsDefaultFalse(t *testing.T) {
	ig := New()
	ig.Add("foo")

	assertPanicsWith(t, "path.relative()", func() {
		ig.Ignores("../foo/bar.js")
	})
	assertPanicsWith(t, "path.relative()", func() {
		ig.Ignores("/foo/bar.js")
	})
}
