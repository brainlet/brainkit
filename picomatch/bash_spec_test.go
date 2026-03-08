// bash_spec_test.go — Faithful 1:1 port of picomatch/test/bash.spec.js
package picomatch

import (
	"testing"
)

func TestBashSpec(t *testing.T) {
	t.Run("dotglob", func(t *testing.T) {
		// bash.spec.js line 9 — "a/b/.x" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 13 — ".x" should match "**/.x/**"
		assertMatch(t, true, ".x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 17 — ".x/" should match "**/.x/**"
		assertMatch(t, true, ".x/", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 21 — ".x/a" should match "**/.x/**"
		assertMatch(t, true, ".x/a", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 25 — ".x/a/b" should match "**/.x/**"
		assertMatch(t, true, ".x/a/b", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 29 — ".x/.x" should match "**/.x/**"
		assertMatch(t, true, ".x/.x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 33 — "a/.x" should match "**/.x/**"
		assertMatch(t, true, "a/.x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 37 — "a/b/.x/c" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/c", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 41 — "a/b/.x/c/d" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/c/d", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 45 — "a/b/.x/c/d/e" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/c/d/e", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 49 — "a/b/.x/" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 53 — "a/.x/b" should match "**/.x/**"
		assertMatch(t, true, "a/.x/b", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 57 — "a/.x/b/.x/c" should not match "**/.x/**"
		assertMatch(t, false, "a/.x/b/.x/c", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 61 — ".bashrc" should not match "?bashrc"
		assertMatch(t, false, ".bashrc", "?bashrc", &Options{Bash: true})

		// bash.spec.js line 65 — should match trailing slashes with stars
		assertMatch(t, true, ".bar.baz/", ".*.*", &Options{Bash: true})

		// bash.spec.js line 69 — ".bar.baz/" should match ".*.*/"
		assertMatch(t, true, ".bar.baz/", ".*.*/", &Options{Bash: true})

		// bash.spec.js line 73 — ".bar.baz" should match ".*.*"
		assertMatch(t, true, ".bar.baz", ".*.*", &Options{Bash: true})
	})

	t.Run("glob", func(t *testing.T) {
		// bash.spec.js line 79 — "a/b/.x" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 83 — ".x" should match "**/.x/**"
		assertMatch(t, true, ".x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 87 — ".x/" should match "**/.x/**"
		assertMatch(t, true, ".x/", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 91 — ".x/a" should match "**/.x/**"
		assertMatch(t, true, ".x/a", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 95 — ".x/a/b" should match "**/.x/**"
		assertMatch(t, true, ".x/a/b", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 99 — ".x/.x" should match "**/.x/**"
		assertMatch(t, true, ".x/.x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 103 — "a/.x" should match "**/.x/**"
		assertMatch(t, true, "a/.x", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 107 — "a/b/.x/c" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/c", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 111 — "a/b/.x/c/d" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/c/d", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 115 — "a/b/.x/c/d/e" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/c/d/e", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 119 — "a/b/.x/" should match "**/.x/**"
		assertMatch(t, true, "a/b/.x/", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 123 — "a/.x/b" should match "**/.x/**"
		assertMatch(t, true, "a/.x/b", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 127 — "a/.x/b/.x/c" should not match "**/.x/**"
		assertMatch(t, false, "a/.x/b/.x/c", "**/.x/**", &Options{Bash: true})

		// bash.spec.js line 131 — "a/c/b" should match "a/*/b"
		assertMatch(t, true, "a/c/b", "a/*/b", &Options{Bash: true})

		// bash.spec.js line 135 — "a/.d/b" should not match "a/*/b"
		assertMatch(t, false, "a/.d/b", "a/*/b", &Options{Bash: true})

		// bash.spec.js line 139 — "a/./b" should not match "a/*/b"
		assertMatch(t, false, "a/./b", "a/*/b", &Options{Bash: true})

		// bash.spec.js line 143 — "a/../b" should not match "a/*/b"
		assertMatch(t, false, "a/../b", "a/*/b", &Options{Bash: true})

		// bash.spec.js line 147 — "ab" should match "ab**"
		assertMatch(t, true, "ab", "ab**", &Options{Bash: true})

		// bash.spec.js line 151 — "abcdef" should match "ab**"
		assertMatch(t, true, "abcdef", "ab**", &Options{Bash: true})

		// bash.spec.js line 155 — "abef" should match "ab**"
		assertMatch(t, true, "abef", "ab**", &Options{Bash: true})

		// bash.spec.js line 159 — "abcfef" should match "ab**"
		assertMatch(t, true, "abcfef", "ab**", &Options{Bash: true})

		// bash.spec.js line 163 — "ab" should not match "ab***ef"
		assertMatch(t, false, "ab", "ab***ef", &Options{Bash: true})

		// bash.spec.js line 167 — "abcdef" should match "ab***ef"
		assertMatch(t, true, "abcdef", "ab***ef", &Options{Bash: true})

		// bash.spec.js line 171 — "abef" should match "ab***ef"
		assertMatch(t, true, "abef", "ab***ef", &Options{Bash: true})

		// bash.spec.js line 175 — "abcfef" should match "ab***ef"
		assertMatch(t, true, "abcfef", "ab***ef", &Options{Bash: true})

		// bash.spec.js line 179 — ".bashrc" should not match "?bashrc"
		assertMatch(t, false, ".bashrc", "?bashrc", &Options{Bash: true})

		// bash.spec.js line 183 — "abbc" should not match "ab?bc"
		assertMatch(t, false, "abbc", "ab?bc", &Options{Bash: true})

		// bash.spec.js line 187 — "abc" should not match "ab?bc"
		assertMatch(t, false, "abc", "ab?bc", &Options{Bash: true})

		// bash.spec.js line 191 — "a.a" should match "[a-d]*.[a-b]"
		assertMatch(t, true, "a.a", "[a-d]*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 195 — "a.b" should match "[a-d]*.[a-b]"
		assertMatch(t, true, "a.b", "[a-d]*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 199 — "c.a" should match "[a-d]*.[a-b]"
		assertMatch(t, true, "c.a", "[a-d]*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 203 — "a.a.a" should match "[a-d]*.[a-b]"
		assertMatch(t, true, "a.a.a", "[a-d]*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 207 — "a.a.a" should match "[a-d]*.[a-b]*.[a-b]"
		assertMatch(t, true, "a.a.a", "[a-d]*.[a-b]*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 211 — "a.a" should match "*.[a-b]"
		assertMatch(t, true, "a.a", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 215 — "a.b" should match "*.[a-b]"
		assertMatch(t, true, "a.b", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 219 — "a.a.a" should match "*.[a-b]"
		assertMatch(t, true, "a.a.a", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 223 — "c.a" should match "*.[a-b]"
		assertMatch(t, true, "c.a", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 227 — "d.a.d" should not match "*.[a-b]"
		assertMatch(t, false, "d.a.d", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 231 — "a.bb" should not match "*.[a-b]"
		assertMatch(t, false, "a.bb", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 235 — "a.ccc" should not match "*.[a-b]"
		assertMatch(t, false, "a.ccc", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 239 — "c.ccc" should not match "*.[a-b]"
		assertMatch(t, false, "c.ccc", "*.[a-b]", &Options{Bash: true})

		// bash.spec.js line 243 — "a.a" should match "*.[a-b]*"
		assertMatch(t, true, "a.a", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 247 — "a.b" should match "*.[a-b]*"
		assertMatch(t, true, "a.b", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 251 — "a.a.a" should match "*.[a-b]*"
		assertMatch(t, true, "a.a.a", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 255 — "c.a" should match "*.[a-b]*"
		assertMatch(t, true, "c.a", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 259 — "d.a.d" should match "*.[a-b]*"
		assertMatch(t, true, "d.a.d", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 263 — "d.a.d" should not match "*.[a-b]*.[a-b]*"
		assertMatch(t, false, "d.a.d", "*.[a-b]*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 267 — "d.a.d" should match "*.[a-d]*.[a-d]*"
		assertMatch(t, true, "d.a.d", "*.[a-d]*.[a-d]*", &Options{Bash: true})

		// bash.spec.js line 271 — "a.bb" should match "*.[a-b]*"
		assertMatch(t, true, "a.bb", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 275 — "a.ccc" should not match "*.[a-b]*"
		assertMatch(t, false, "a.ccc", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 279 — "c.ccc" should not match "*.[a-b]*"
		assertMatch(t, false, "c.ccc", "*.[a-b]*", &Options{Bash: true})

		// bash.spec.js line 283 — "a.a" should match "*[a-b].[a-b]*"
		assertMatch(t, true, "a.a", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 287 — "a.b" should match "*[a-b].[a-b]*"
		assertMatch(t, true, "a.b", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 291 — "a.a.a" should match "*[a-b].[a-b]*"
		assertMatch(t, true, "a.a.a", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 295 — "c.a" should not match "*[a-b].[a-b]*"
		assertMatch(t, false, "c.a", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 299 — "d.a.d" should not match "*[a-b].[a-b]*"
		assertMatch(t, false, "d.a.d", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 303 — "a.bb" should match "*[a-b].[a-b]*"
		assertMatch(t, true, "a.bb", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 307 — "a.ccc" should not match "*[a-b].[a-b]*"
		assertMatch(t, false, "a.ccc", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 311 — "c.ccc" should not match "*[a-b].[a-b]*"
		assertMatch(t, false, "c.ccc", "*[a-b].[a-b]*", &Options{Bash: true})

		// bash.spec.js line 315 — "abd" should match "[a-y]*[^c]"
		assertMatch(t, true, "abd", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 319 — "abe" should match "[a-y]*[^c]"
		assertMatch(t, true, "abe", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 323 — "bb" should match "[a-y]*[^c]"
		assertMatch(t, true, "bb", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 327 — "bcd" should match "[a-y]*[^c]"
		assertMatch(t, true, "bcd", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 331 — "ca" should match "[a-y]*[^c]"
		assertMatch(t, true, "ca", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 335 — "cb" should match "[a-y]*[^c]"
		assertMatch(t, true, "cb", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 339 — "dd" should match "[a-y]*[^c]"
		assertMatch(t, true, "dd", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 343 — "de" should match "[a-y]*[^c]"
		assertMatch(t, true, "de", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 347 — "bdir/" should match "[a-y]*[^c]"
		assertMatch(t, true, "bdir/", "[a-y]*[^c]", &Options{Bash: true})

		// bash.spec.js line 351 — "abd" should match "**/*"
		assertMatch(t, true, "abd", "**/*", &Options{Bash: true})
	})

	t.Run("globstar", func(t *testing.T) {
		// bash.spec.js line 357 — "a.js" should match "**/*.js"
		assertMatch(t, true, "a.js", "**/*.js", &Options{Bash: true})

		// bash.spec.js line 361 — "a/a.js" should match "**/*.js"
		assertMatch(t, true, "a/a.js", "**/*.js", &Options{Bash: true})

		// bash.spec.js line 365 — "a/a/b.js" should match "**/*.js"
		assertMatch(t, true, "a/a/b.js", "**/*.js", &Options{Bash: true})

		// bash.spec.js line 369 — "a/b/z.js" should match "a/b/**/*.js"
		assertMatch(t, true, "a/b/z.js", "a/b/**/*.js", &Options{Bash: true})

		// bash.spec.js line 373 — "a/b/c/z.js" should match "a/b/**/*.js"
		assertMatch(t, true, "a/b/c/z.js", "a/b/**/*.js", &Options{Bash: true})

		// bash.spec.js line 377 — "foo.md" should match "**/*.md"
		assertMatch(t, true, "foo.md", "**/*.md", &Options{Bash: true})

		// bash.spec.js line 381 — "foo/bar.md" should match "**/*.md"
		assertMatch(t, true, "foo/bar.md", "**/*.md", &Options{Bash: true})

		// bash.spec.js line 385 — "foo/bar" should match "foo/**/bar"
		assertMatch(t, true, "foo/bar", "foo/**/bar", &Options{Bash: true})

		// bash.spec.js line 389 — "foo/bar" should match "foo/**bar"
		assertMatch(t, true, "foo/bar", "foo/**bar", &Options{Bash: true})

		// bash.spec.js line 393 — "ab/a/d" should match "**/*"
		assertMatch(t, true, "ab/a/d", "**/*", &Options{Bash: true})

		// bash.spec.js line 397 — "ab/b" should match "**/*"
		assertMatch(t, true, "ab/b", "**/*", &Options{Bash: true})

		// bash.spec.js line 401 — "a/b/c/d/a.js" should match "**/*"
		assertMatch(t, true, "a/b/c/d/a.js", "**/*", &Options{Bash: true})

		// bash.spec.js line 405 — "a/b/c.js" should match "**/*"
		assertMatch(t, true, "a/b/c.js", "**/*", &Options{Bash: true})

		// bash.spec.js line 409 — "a/b/c.txt" should match "**/*"
		assertMatch(t, true, "a/b/c.txt", "**/*", &Options{Bash: true})

		// bash.spec.js line 413 — "a/b/.js/c.txt" should match "**/*"
		assertMatch(t, true, "a/b/.js/c.txt", "**/*", &Options{Bash: true})

		// bash.spec.js line 417 — "a.js" should match "**/*"
		assertMatch(t, true, "a.js", "**/*", &Options{Bash: true})

		// bash.spec.js line 421 — "za.js" should match "**/*"
		assertMatch(t, true, "za.js", "**/*", &Options{Bash: true})

		// bash.spec.js line 425 — "ab" should match "**/*"
		assertMatch(t, true, "ab", "**/*", &Options{Bash: true})

		// bash.spec.js line 429 — "a.b" should match "**/*"
		assertMatch(t, true, "a.b", "**/*", &Options{Bash: true})

		// bash.spec.js line 433 — "foo/" should match "foo/**/"
		assertMatch(t, true, "foo/", "foo/**/", &Options{Bash: true})

		// bash.spec.js line 437 — "foo/bar" should not match "foo/**/"
		assertMatch(t, false, "foo/bar", "foo/**/", &Options{Bash: true})

		// bash.spec.js line 441 — "foo/bazbar" should not match "foo/**/"
		assertMatch(t, false, "foo/bazbar", "foo/**/", &Options{Bash: true})

		// bash.spec.js line 445 — "foo/barbar" should not match "foo/**/"
		assertMatch(t, false, "foo/barbar", "foo/**/", &Options{Bash: true})

		// bash.spec.js line 449 — "foo/bar/baz/qux" should not match "foo/**/"
		assertMatch(t, false, "foo/bar/baz/qux", "foo/**/", &Options{Bash: true})

		// bash.spec.js line 453 — "foo/bar/baz/qux/" should match "foo/**/"
		assertMatch(t, true, "foo/bar/baz/qux/", "foo/**/", &Options{Bash: true})
	})
}
