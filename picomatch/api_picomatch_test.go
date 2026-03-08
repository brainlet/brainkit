// api_picomatch_test.go — Faithful 1:1 port of picomatch/test/api.picomatch.js
package picomatch

import (
	"testing"
)

func TestPicomatch(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		t.Run("should throw an error when invalid arguments are given", func(t *testing.T) {
			// api.picomatch.js line 15
			assertPanics(t, func() { IsMatch("foo", "", nil) }, "empty string pattern")
			// api.picomatch.js line 16
			// In Go, nil interface{} is handled by Compile panicking.
			// We test with empty string since Go doesn't have null strings.
		})
	})

	t.Run("multiple patterns", func(t *testing.T) {
		t.Run("should return true when any of the patterns match", func(t *testing.T) {
			// api.picomatch.js line 22
			assertMatch(t, true, ".", ".")
			// api.picomatch.js line 23
			assertMatch(t, true, "a", "a")
			// api.picomatch.js line 24 — IsMatch with []string patterns
			assertMatchMulti(t, true, "ab", []string{"*", "foo", "bar"})
			// api.picomatch.js line 25
			assertMatchMulti(t, true, "ab", []string{"*b", "foo", "bar"})
			// api.picomatch.js line 26
			assertMatchMulti(t, true, "ab", []string{"./*", "foo", "bar"})
			// api.picomatch.js line 27
			assertMatchMulti(t, true, "ab", []string{"a*", "foo", "bar"})
			// api.picomatch.js line 28
			assertMatchMulti(t, true, "ab", []string{"ab", "foo"})
		})

		t.Run("should return false when none of the patterns match", func(t *testing.T) {
			// api.picomatch.js line 32
			assertMatchMulti(t, false, "/ab", []string{"/a", "foo"})
			// api.picomatch.js line 33
			assertMatchMulti(t, false, "/ab", []string{"?/?", "foo", "bar"})
			// api.picomatch.js line 34
			assertMatchMulti(t, false, "/ab", []string{"a/*", "foo", "bar"})
			// api.picomatch.js line 35
			assertMatchMulti(t, false, "a/b/c", []string{"a/b", "foo"})
			// api.picomatch.js line 36
			assertMatchMulti(t, false, "ab", []string{"*/*", "foo", "bar"})
			// api.picomatch.js line 37
			assertMatchMulti(t, false, "ab", []string{"/a", "foo", "bar"})
			// api.picomatch.js line 38
			assertMatchMulti(t, false, "ab", []string{"a", "foo"})
			// api.picomatch.js line 39
			assertMatchMulti(t, false, "ab", []string{"b", "foo"})
			// api.picomatch.js line 40
			assertMatchMulti(t, false, "ab", []string{"c", "foo", "bar"})
			// api.picomatch.js line 41
			assertMatchMulti(t, false, "abcd", []string{"ab", "foo"})
			// api.picomatch.js line 42
			assertMatchMulti(t, false, "abcd", []string{"bc", "foo"})
			// api.picomatch.js line 43
			assertMatchMulti(t, false, "abcd", []string{"c", "foo"})
			// api.picomatch.js line 44
			assertMatchMulti(t, false, "abcd", []string{"cd", "foo"})
			// api.picomatch.js line 45
			assertMatchMulti(t, false, "abcd", []string{"d", "foo"})
			// api.picomatch.js line 46
			assertMatchMulti(t, false, "abcd", []string{"f", "foo", "bar"})
			// api.picomatch.js line 47
			assertMatchMulti(t, false, "ef", []string{"/*", "foo", "bar"})
		})
	})

	t.Run("file extensions", func(t *testing.T) {
		t.Run("should match files that contain the given extension", func(t *testing.T) {
			// api.picomatch.js line 53
			assertMatch(t, false, ".c.md", "*.md")
			// api.picomatch.js line 54
			assertMatch(t, false, ".c.md", ".c.")
			// api.picomatch.js line 55
			assertMatch(t, false, ".c.md", ".md")
			// api.picomatch.js line 56
			assertMatch(t, false, ".md", "*.md")
			// api.picomatch.js line 57
			assertMatch(t, false, ".md", ".m")
			// api.picomatch.js line 58
			assertMatch(t, false, "a/b/c.md", "*.md")
			// api.picomatch.js line 59
			assertMatch(t, false, "a/b/c.md", ".md")
			// api.picomatch.js line 60
			assertMatch(t, false, "a/b/c.md", "a/*.md")
			// api.picomatch.js line 61
			assertMatch(t, false, "a/b/c/c.md", "*.md")
			// api.picomatch.js line 62
			assertMatch(t, false, "a/b/c/c.md", "c.js")
			// api.picomatch.js line 63
			assertMatch(t, true, ".c.md", ".*.md")
			// api.picomatch.js line 64
			assertMatch(t, true, ".md", ".md")
			// api.picomatch.js line 65
			assertMatch(t, true, "a/b/c.js", "a/**/*.*")
			// api.picomatch.js line 66
			assertMatch(t, true, "a/b/c.md", "**/*.md")
			// api.picomatch.js line 67
			assertMatch(t, true, "a/b/c.md", "a/*/*.md")
			// api.picomatch.js line 68
			assertMatch(t, true, "c.md", "*.md")
		})
	})

	t.Run("dot files", func(t *testing.T) {
		t.Run("should not match dotfiles when a leading dot is not defined in a path segment", func(t *testing.T) {
			// api.picomatch.js line 74
			assertMatch(t, false, ".a", "(a)*")
			// api.picomatch.js line 75
			assertMatch(t, false, ".a", "*(a|b)")
			// api.picomatch.js line 76
			assertMatch(t, false, ".a", "*.md")
			// api.picomatch.js line 77
			assertMatch(t, false, ".a", "*[a]")
			// api.picomatch.js line 78
			assertMatch(t, false, ".a", "*[a]*")
			// api.picomatch.js line 79
			assertMatch(t, false, ".a", "*a")
			// api.picomatch.js line 80
			assertMatch(t, false, ".a", "*a*")
			// api.picomatch.js line 81
			assertMatch(t, false, ".a.md", "a/b/c/*.md")
			// api.picomatch.js line 82
			assertMatch(t, false, ".ab", "*.*")
			// api.picomatch.js line 83
			assertMatch(t, false, ".abc", ".a")
			// api.picomatch.js line 84
			assertMatch(t, false, ".ba", ".a")
			// api.picomatch.js line 85
			assertMatch(t, false, ".c.md", "*.md")
			// api.picomatch.js line 86
			assertMatch(t, false, ".md", "a/b/c/*.md")
			// api.picomatch.js line 87
			assertMatch(t, false, ".txt", ".md")
			// api.picomatch.js line 88
			assertMatch(t, false, ".verb.txt", "*.md")
			// api.picomatch.js line 89
			assertMatch(t, false, "a/.c.md", "*.md")
			// api.picomatch.js line 90
			assertMatch(t, false, "a/b/d/.md", "a/b/c/*.md")
			// api.picomatch.js line 91
			assertMatch(t, true, ".a", ".a")
			// api.picomatch.js line 92
			assertMatch(t, true, ".ab", ".*")
			// api.picomatch.js line 93
			assertMatch(t, true, ".ab", ".a*")
			// api.picomatch.js line 94
			assertMatch(t, true, ".b", ".b*")
			// api.picomatch.js line 95
			assertMatch(t, true, ".md", ".md")
			// api.picomatch.js line 96
			assertMatch(t, true, "a/.c.md", "a/.c.md")
			// api.picomatch.js line 97
			assertMatch(t, true, "a/b/c/.xyz.md", "a/b/c/.*.md")
			// api.picomatch.js line 98
			assertMatch(t, true, "a/b/c/d.a.md", "a/b/c/*.md")
		})

		t.Run("should match dotfiles when options.dot is true", func(t *testing.T) {
			dot := &Options{Dot: true}
			// api.picomatch.js line 102
			assertMatch(t, false, "a/b/c/.xyz.md", ".*.md", dot)
			// api.picomatch.js line 103
			assertMatch(t, true, ".c.md", "*.md", dot)
			// api.picomatch.js line 104
			assertMatch(t, true, ".c.md", ".*", dot)
			// api.picomatch.js line 105
			assertMatch(t, true, "a/b/c/.xyz.md", "**/*.md", dot)
			// api.picomatch.js line 106
			assertMatch(t, true, "a/b/c/.xyz.md", "**/.*.md", dot)
			// api.picomatch.js line 107
			assertMatch(t, true, "a/b/c/.xyz.md", "a/b/c/*.md", dot)
			// api.picomatch.js line 108
			assertMatch(t, true, "a/b/c/.xyz.md", "a/b/c/.*.md", dot)
		})
	})

	t.Run("matching", func(t *testing.T) {
		t.Run("should escape plus signs to match string literals", func(t *testing.T) {
			// api.picomatch.js line 114
			assertMatch(t, true, "a+b/src/glimini.js", "a+b/src/*.js")
			// api.picomatch.js line 115
			assertMatch(t, true, "+b/src/glimini.js", "+b/src/*.js")
			// api.picomatch.js line 116
			assertMatch(t, true, "coffee+/src/glimini.js", "coffee+/src/*.js")
			// api.picomatch.js line 117
			assertMatch(t, true, "coffee+/src/glimini.js", "coffee+/src/*")
		})

		t.Run("should match with non-glob patterns", func(t *testing.T) {
			// api.picomatch.js line 121
			assertMatch(t, true, ".", ".")
			// api.picomatch.js line 122
			assertMatch(t, true, "/a", "/a")
			// api.picomatch.js line 123
			assertMatch(t, false, "/ab", "/a")
			// api.picomatch.js line 124
			assertMatch(t, true, "a", "a")
			// api.picomatch.js line 125
			assertMatch(t, false, "ab", "/a")
			// api.picomatch.js line 126
			assertMatch(t, false, "ab", "a")
			// api.picomatch.js line 127
			assertMatch(t, true, "ab", "ab")
			// api.picomatch.js line 128
			assertMatch(t, false, "abcd", "cd")
			// api.picomatch.js line 129
			assertMatch(t, false, "abcd", "bc")
			// api.picomatch.js line 130
			assertMatch(t, false, "abcd", "ab")
		})

		t.Run("should match file names", func(t *testing.T) {
			// api.picomatch.js line 134
			assertMatch(t, true, "a.b", "a.b")
			// api.picomatch.js line 135
			assertMatch(t, true, "a.b", "*.b")
			// api.picomatch.js line 136
			assertMatch(t, true, "a.b", "a.*")
			// api.picomatch.js line 137
			assertMatch(t, true, "a.b", "*.*")
			// api.picomatch.js line 138
			assertMatch(t, true, "a-b.c-d", "a*.c*")
			// api.picomatch.js line 139
			assertMatch(t, true, "a-b.c-d", "*b.*d")
			// api.picomatch.js line 140
			assertMatch(t, true, "a-b.c-d", "*.*")
			// api.picomatch.js line 141
			assertMatch(t, true, "a-b.c-d", "*.*-*")
			// api.picomatch.js line 142
			assertMatch(t, true, "a-b.c-d", "*-*.*-*")
			// api.picomatch.js line 143
			assertMatch(t, true, "a-b.c-d", "*.c-*")
			// api.picomatch.js line 144
			assertMatch(t, true, "a-b.c-d", "*.*-d")
			// api.picomatch.js line 145
			assertMatch(t, true, "a-b.c-d", "a-*.*-d")
			// api.picomatch.js line 146
			assertMatch(t, true, "a-b.c-d", "*-b.c-*")
			// api.picomatch.js line 147
			assertMatch(t, true, "a-b.c-d", "*-b*c-*")
			// api.picomatch.js line 148
			assertMatch(t, false, "a-b.c-d", "*-bc-*")
		})

		t.Run("should match with common glob patterns", func(t *testing.T) {
			// api.picomatch.js line 152
			assertMatch(t, false, "/ab", "./*/")
			// api.picomatch.js line 153
			assertMatch(t, false, "/ef", "*")
			// api.picomatch.js line 154
			assertMatch(t, false, "ab", "./*/")
			// api.picomatch.js line 155
			assertMatch(t, false, "ef", "/*")
			// api.picomatch.js line 156
			assertMatch(t, true, "/ab", "/*")
			// api.picomatch.js line 157
			assertMatch(t, true, "/cd", "/*")
			// api.picomatch.js line 158
			assertMatch(t, true, "ab", "*")
			// api.picomatch.js line 159
			assertMatch(t, true, "ab", "./*")
			// api.picomatch.js line 160
			assertMatch(t, true, "ab", "ab")
			// api.picomatch.js line 161
			assertMatch(t, true, "ab/", "./*/")
		})

		t.Run("should match files with the given extension", func(t *testing.T) {
			// api.picomatch.js line 165
			assertMatch(t, false, ".md", "*.md")
			// api.picomatch.js line 166
			assertMatch(t, true, ".md", ".md")
			// api.picomatch.js line 167
			assertMatch(t, false, ".c.md", "*.md")
			// api.picomatch.js line 168
			assertMatch(t, true, ".c.md", ".*.md")
			// api.picomatch.js line 169
			assertMatch(t, true, "c.md", "*.md")
			// api.picomatch.js line 170
			assertMatch(t, true, "c.md", "*.md")
			// api.picomatch.js line 171
			assertMatch(t, false, "a/b/c/c.md", "*.md")
			// api.picomatch.js line 172
			assertMatch(t, false, "a/b/c.md", "a/*.md")
			// api.picomatch.js line 173
			assertMatch(t, true, "a/b/c.md", "a/*/*.md")
			// api.picomatch.js line 174
			assertMatch(t, true, "a/b/c.md", "**/*.md")
			// api.picomatch.js line 175
			assertMatch(t, true, "a/b/c.js", "a/**/*.*")
		})

		t.Run("should match wildcards", func(t *testing.T) {
			// api.picomatch.js line 179
			assertMatch(t, false, "a/b/c/z.js", "*.js")
			// api.picomatch.js line 180
			assertMatch(t, false, "a/b/z.js", "*.js")
			// api.picomatch.js line 181
			assertMatch(t, false, "a/z.js", "*.js")
			// api.picomatch.js line 182
			assertMatch(t, true, "z.js", "*.js")

			// api.picomatch.js line 184
			assertMatch(t, true, "z.js", "z*.js")
			// api.picomatch.js line 185
			assertMatch(t, true, "a/z.js", "a/z*.js")
			// api.picomatch.js line 186
			assertMatch(t, true, "a/z.js", "*/z*.js")
		})

		t.Run("should match globstars", func(t *testing.T) {
			// api.picomatch.js line 190
			assertMatch(t, true, "a/b/c/z.js", "**/*.js")
			// api.picomatch.js line 191
			assertMatch(t, true, "a/b/z.js", "**/*.js")
			// api.picomatch.js line 192
			assertMatch(t, true, "a/z.js", "**/*.js")
			// api.picomatch.js line 193
			assertMatch(t, true, "a/b/c/d/e/z.js", "a/b/**/*.js")
			// api.picomatch.js line 194
			assertMatch(t, true, "a/b/c/d/z.js", "a/b/**/*.js")
			// api.picomatch.js line 195
			assertMatch(t, true, "a/b/c/z.js", "a/b/c/**/*.js")
			// api.picomatch.js line 196
			assertMatch(t, true, "a/b/c/z.js", "a/b/c**/*.js")
			// api.picomatch.js line 197
			assertMatch(t, true, "a/b/c/z.js", "a/b/**/*.js")
			// api.picomatch.js line 198
			assertMatch(t, true, "a/b/z.js", "a/b/**/*.js")

			// api.picomatch.js line 200
			assertMatch(t, false, "a/z.js", "a/b/**/*.js")
			// api.picomatch.js line 201
			assertMatch(t, false, "z.js", "a/b/**/*.js")

			// api.picomatch.js line 203 — https://github.com/micromatch/micromatch/issues/15
			// api.picomatch.js line 204
			assertMatch(t, true, "z.js", "z*")
			// api.picomatch.js line 205
			assertMatch(t, true, "z.js", "**/z*")
			// api.picomatch.js line 206
			assertMatch(t, true, "z.js", "**/z*.js")
			// api.picomatch.js line 207
			assertMatch(t, true, "z.js", "**/*.js")
			// api.picomatch.js line 208
			assertMatch(t, true, "foo", "**/foo")
		})

		t.Run("issue #23", func(t *testing.T) {
			// api.picomatch.js line 212
			assertMatch(t, false, "zzjs", "z*.js")
			// api.picomatch.js line 213
			assertMatch(t, false, "zzjs", "*z.js")
		})

		t.Run("issue #24 - should match zero or more directories", func(t *testing.T) {
			// api.picomatch.js line 217
			assertMatch(t, false, "a/b/c/d/", "a/b/**/f")
			// api.picomatch.js line 218
			assertMatch(t, true, "a", "a/**")
			// api.picomatch.js line 219
			assertMatch(t, true, "a", "**")
			// api.picomatch.js line 220
			assertMatch(t, true, "a/", "**")
			// api.picomatch.js line 221
			assertMatch(t, true, "a/b-c/d/e/z.js", "a/b-*/**/z.js")
			// api.picomatch.js line 222
			assertMatch(t, true, "a/b-c/z.js", "a/b-*/**/z.js")
			// api.picomatch.js line 223
			assertMatch(t, true, "a/b/c/d", "**")
			// api.picomatch.js line 224
			assertMatch(t, true, "a/b/c/d/", "**")
			// api.picomatch.js line 225
			assertMatch(t, true, "a/b/c/d/", "**/**")
			// api.picomatch.js line 226
			assertMatch(t, true, "a/b/c/d/", "**/b/**")
			// api.picomatch.js line 227
			assertMatch(t, true, "a/b/c/d/", "a/b/**")
			// api.picomatch.js line 228
			assertMatch(t, true, "a/b/c/d/", "a/b/**/")
			// api.picomatch.js line 229
			assertMatch(t, true, "a/b/c/d/", "a/b/**/c/**/")
			// api.picomatch.js line 230
			assertMatch(t, true, "a/b/c/d/", "a/b/**/c/**/d/")
			// api.picomatch.js line 231
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/**/*.*")
			// api.picomatch.js line 232
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/*.*")
			// api.picomatch.js line 233
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/c/**/d/*.*")
			// api.picomatch.js line 234
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/d/**/*.*")
			// api.picomatch.js line 235
			assertMatch(t, true, "a/b/c/d/g/e.f", "a/b/**/d/**/*.*")
			// api.picomatch.js line 236
			assertMatch(t, true, "a/b/c/d/g/g/e.f", "a/b/**/d/**/*.*")
		})

		t.Run("should match slashes", func(t *testing.T) {
			// api.picomatch.js line 240
			assertMatch(t, false, "bar/baz/foo", "*/foo")
			// api.picomatch.js line 241
			assertMatch(t, false, "deep/foo/bar", "**/bar/*")
			// api.picomatch.js line 242
			assertMatch(t, false, "deep/foo/bar/baz/x", "*/bar/**")
			// api.picomatch.js line 243
			assertMatch(t, false, "foo/bar", "foo?bar")
			// api.picomatch.js line 244
			assertMatch(t, false, "foo/bar/baz", "**/bar*")
			// api.picomatch.js line 245
			assertMatch(t, false, "foo/bar/baz", "**/bar**")
			// api.picomatch.js line 246
			assertMatch(t, false, "foo/baz/bar", "foo**bar")
			// api.picomatch.js line 247
			assertMatch(t, false, "foo/baz/bar", "foo*bar")
			// api.picomatch.js line 248
			assertMatch(t, false, "deep/foo/bar/baz", "**/bar/*/")
			// api.picomatch.js line 249
			assertMatch(t, false, "deep/foo/bar/baz/", "**/bar/*", &Options{StrictSlashes: true})
			// api.picomatch.js line 250
			assertMatch(t, true, "deep/foo/bar/baz/", "**/bar/*")
			// api.picomatch.js line 251
			assertMatch(t, true, "deep/foo/bar/baz", "**/bar/*")
			// api.picomatch.js line 252
			assertMatch(t, true, "foo", "foo/**")
			// api.picomatch.js line 253
			assertMatch(t, true, "deep/foo/bar/baz/", "**/bar/*{,/}")
			// api.picomatch.js line 254
			assertMatch(t, true, "a/b/j/c/z/x.md", "a/**/j/**/z/*.md")
			// api.picomatch.js line 255
			assertMatch(t, true, "a/j/z/x.md", "a/**/j/**/z/*.md")
			// api.picomatch.js line 256
			assertMatch(t, true, "bar/baz/foo", "**/foo")
			// api.picomatch.js line 257
			assertMatch(t, true, "deep/foo/bar/", "**/bar/**")
			// api.picomatch.js line 258
			assertMatch(t, true, "deep/foo/bar/baz", "**/bar/*")
			// api.picomatch.js line 259
			assertMatch(t, true, "deep/foo/bar/baz/", "**/bar/*/")
			// api.picomatch.js line 260
			assertMatch(t, true, "deep/foo/bar/baz/", "**/bar/**")
			// api.picomatch.js line 261
			assertMatch(t, true, "deep/foo/bar/baz/x", "**/bar/*/*")
			// api.picomatch.js line 262
			assertMatch(t, true, "foo/b/a/z/bar", "foo/**/**/bar")
			// api.picomatch.js line 263
			assertMatch(t, true, "foo/b/a/z/bar", "foo/**/bar")
			// api.picomatch.js line 264
			assertMatch(t, true, "foo/bar", "foo/**/**/bar")
			// api.picomatch.js line 265
			assertMatch(t, true, "foo/bar", "foo/**/bar")
			// api.picomatch.js line 266
			assertMatch(t, true, "foo/bar", "foo[/]bar")
			// api.picomatch.js line 267
			assertMatch(t, true, "foo/bar/baz/x", "*/bar/**")
			// api.picomatch.js line 268
			assertMatch(t, true, "foo/baz/bar", "foo/**/**/bar")
			// api.picomatch.js line 269
			assertMatch(t, true, "foo/baz/bar", "foo/**/bar")
			// api.picomatch.js line 270
			assertMatch(t, true, "foobazbar", "foo**bar")
			// api.picomatch.js line 271
			assertMatch(t, true, "XXX/foo", "**/foo")

			// api.picomatch.js line 273 — https://github.com/micromatch/micromatch/issues/89
			// api.picomatch.js line 274
			assertMatch(t, true, "foo//baz.md", "foo//baz.md")
			// api.picomatch.js line 275
			assertMatch(t, true, "foo//baz.md", "foo//*baz.md")
			// api.picomatch.js line 276
			assertMatch(t, true, "foo//baz.md", "foo{/,//}baz.md")
			// api.picomatch.js line 277
			assertMatch(t, true, "foo/baz.md", "foo{/,//}baz.md")
			// api.picomatch.js line 278
			assertMatch(t, false, "foo//baz.md", "foo/+baz.md")
			// api.picomatch.js line 279
			assertMatch(t, false, "foo//baz.md", "foo//+baz.md")
			// api.picomatch.js line 280
			assertMatch(t, false, "foo//baz.md", "foo/baz.md")
			// api.picomatch.js line 281
			assertMatch(t, false, "foo/baz.md", "foo//baz.md")
		})

		t.Run("question marks should not match slashes", func(t *testing.T) {
			// api.picomatch.js line 285
			assertMatch(t, false, "aaa/bbb", "aaa?bbb")
		})

		t.Run("should not match dotfiles when dot or dotfiles are not set", func(t *testing.T) {
			// api.picomatch.js line 289
			assertMatch(t, false, ".c.md", "*.md")
			// api.picomatch.js line 290
			assertMatch(t, false, "a/.c.md", "*.md")
			// api.picomatch.js line 291
			assertMatch(t, true, "a/.c.md", "a/.c.md")
			// api.picomatch.js line 292
			assertMatch(t, false, ".a", "*.md")
			// api.picomatch.js line 293
			assertMatch(t, false, ".verb.txt", "*.md")
			// api.picomatch.js line 294
			assertMatch(t, true, "a/b/c/.xyz.md", "a/b/c/.*.md")
			// api.picomatch.js line 295
			assertMatch(t, true, ".md", ".md")
			// api.picomatch.js line 296
			assertMatch(t, false, ".txt", ".md")
			// api.picomatch.js line 297
			assertMatch(t, true, ".md", ".md")
			// api.picomatch.js line 298
			assertMatch(t, true, ".a", ".a")
			// api.picomatch.js line 299
			assertMatch(t, true, ".b", ".b*")
			// api.picomatch.js line 300
			assertMatch(t, true, ".ab", ".a*")
			// api.picomatch.js line 301
			assertMatch(t, true, ".ab", ".*")
			// api.picomatch.js line 302
			assertMatch(t, false, ".ab", "*.*")
			// api.picomatch.js line 303
			assertMatch(t, false, ".md", "a/b/c/*.md")
			// api.picomatch.js line 304
			assertMatch(t, false, ".a.md", "a/b/c/*.md")
			// api.picomatch.js line 305
			assertMatch(t, true, "a/b/c/d.a.md", "a/b/c/*.md")
			// api.picomatch.js line 306
			assertMatch(t, false, "a/b/d/.md", "a/b/c/*.md")
		})

		t.Run("should match dotfiles when dot or dotfiles is set", func(t *testing.T) {
			dot := &Options{Dot: true}
			// api.picomatch.js line 310
			assertMatch(t, true, ".c.md", "*.md", dot)
			// api.picomatch.js line 311
			assertMatch(t, true, ".c.md", ".*", dot)
			// api.picomatch.js line 312
			assertMatch(t, true, "a/b/c/.xyz.md", "a/b/c/*.md", dot)
			// api.picomatch.js line 313
			assertMatch(t, true, "a/b/c/.xyz.md", "a/b/c/.*.md", dot)
		})
	})

	t.Run(".parse", func(t *testing.T) {
		t.Run("tokens", func(t *testing.T) {
			t.Run("should return result for pattern that matched by fastpath", func(t *testing.T) {
				// api.picomatch.js line 320
				parsed := Parse("a*.txt", nil)

				expected := [][2]string{
					{"bos", ""},
					{"text", "a"},
					{"star", "*"},
					{"text", ".txt"},
				}

				// api.picomatch.js line 329
				assertTokens(t, parsed.Tokens, expected)
			})

			t.Run("should return result for pattern", func(t *testing.T) {
				// api.picomatch.js line 332
				parsed := Parse("{a,b}*", nil)

				expected := [][2]string{
					{"bos", ""},
					{"brace", "{"},
					{"text", "a"},
					{"comma", ","},
					{"text", "b"},
					{"brace", "}"},
					{"star", "*"},
					{"maybe_slash", ""},
				}

				// api.picomatch.js line 346
				assertTokens(t, parsed.Tokens, expected)
			})

			t.Run("picomatch issue#125 issue#100", func(t *testing.T) {
				// api.picomatch.js line 349
				parsed := Parse("foo.(m|c|)js", nil)

				// Verify token types and values match the expected pattern.
				// The JS test checks type + {output, value} for each token.
				// api.picomatch.js line 352-358
				type tokenCheck struct {
					Type   string
					Output string // "" means undefined in JS
					Value  string
				}

				expected := []tokenCheck{
					{Type: "bos", Output: "", Value: ""},
					{Type: "text", Output: "foo.", Value: "foo."},
					{Type: "paren", Value: "("},
					{Type: "text", Value: "m|c|"},
					{Type: "paren", Value: ")"},
					{Type: "text", Value: "js"},
				}

				if len(parsed.Tokens) != len(expected) {
					t.Fatalf("expected %d tokens, got %d", len(expected), len(parsed.Tokens))
				}

				for i, exp := range expected {
					tok := parsed.Tokens[i]
					if tok.Type != exp.Type {
						t.Errorf("token[%d].Type: expected %q, got %q", i, exp.Type, tok.Type)
					}
					if tok.Value != exp.Value {
						t.Errorf("token[%d].Value: expected %q, got %q", i, exp.Value, tok.Value)
					}
				}
			})
		})
	})

	t.Run("state", func(t *testing.T) {
		t.Run("negatedExtglob", func(t *testing.T) {
			t.Run("should return true", func(t *testing.T) {
				// api.picomatch.js line 370
				// In JS: picomatch('!(abc)', {}, true).state.negatedExtglob
				// We compile and check the parse state via MakeRe.
				assertNegatedExtglob(t, true, "!(abc)")
				// api.picomatch.js line 371
				assertNegatedExtglob(t, true, "!(abc)**")
				// api.picomatch.js line 372
				assertNegatedExtglob(t, true, "!(abc)/**")
			})

			t.Run("should return false", func(t *testing.T) {
				// api.picomatch.js line 376
				assertNegatedExtglob(t, false, "(!(abc))")
				// api.picomatch.js line 377
				assertNegatedExtglob(t, false, "**!(abc)")
			})
		})
	})
}

// --- Test helpers specific to this file ---

// assertPanics checks that fn panics.
func assertPanics(t *testing.T, fn func(), msg string) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for %s, but did not panic", msg)
		}
	}()
	fn()
}

// assertMatchMulti checks IsMatch with a slice of patterns.
// JS: isMatch(str, [patterns...])
func assertMatchMulti(t *testing.T, expected bool, input string, patterns []string) {
	t.Helper()
	result := IsMatch(input, patterns, nil)
	if result != expected {
		if expected {
			t.Errorf("expected isMatch(%q, %v) to be true, got false", input, patterns)
		} else {
			t.Errorf("expected isMatch(%q, %v) to be false, got true", input, patterns)
		}
	}
}

// assertTokens checks that token [type, value] pairs match expected.
// Ported from: api.picomatch.js lines 7-10
func assertTokens(t *testing.T, tokens []*Token, expected [][2]string) {
	t.Helper()
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		tok := tokens[i]
		if tok.Type != exp[0] {
			t.Errorf("token[%d].Type: expected %q, got %q", i, exp[0], tok.Type)
		}
		if tok.Value != exp[1] {
			t.Errorf("token[%d].Value: expected %q, got %q", i, exp[1], tok.Value)
		}
	}
}

// assertNegatedExtglob checks the NegatedExtglob field of the compiled parse state.
// JS: picomatch(pattern, {}, true).state.negatedExtglob
func assertNegatedExtglob(t *testing.T, expected bool, pattern string) {
	t.Helper()
	compiled := MakeRe(pattern, nil)
	if compiled.state.NegatedExtglob != expected {
		t.Errorf("MakeRe(%q).state.NegatedExtglob: expected %v, got %v",
			pattern, expected, compiled.state.NegatedExtglob)
	}
}
