// stars_test.go — Faithful 1:1 port of picomatch/test/stars.js
package picomatch

import (
	"testing"
)

func TestStars(t *testing.T) {
	t.Run("issue related", func(t *testing.T) {
		t.Run("should respect dots defined in glob pattern (micromatch/#23)", func(t *testing.T) {
			// stars.js line 9
			assertMatch(t, true, "z.js", "z*")
			// stars.js line 10
			assertMatch(t, false, "zzjs", "z*.js")
			// stars.js line 11
			assertMatch(t, false, "zzjs", "*z.js")
		})
	})

	t.Run("single stars", func(t *testing.T) {
		t.Run("should match anything except slashes and leading dots", func(t *testing.T) {
			// stars.js line 17
			assertMatch(t, false, "a/b/c/z.js", "*.js")
			// stars.js line 18
			assertMatch(t, false, "a/b/z.js", "*.js")
			// stars.js line 19
			assertMatch(t, false, "a/z.js", "*.js")
			// stars.js line 20
			assertMatch(t, true, "z.js", "*.js")

			// stars.js line 22
			assertMatch(t, false, "a/.ab", "*/*")
			// stars.js line 23
			assertMatch(t, false, ".ab", "*")

			// stars.js line 25
			assertMatch(t, true, "z.js", "z*.js")
			// stars.js line 26
			assertMatch(t, true, "a/z", "*/*")
			// stars.js line 27
			assertMatch(t, true, "a/z.js", "*/z*.js")
			// stars.js line 28
			assertMatch(t, true, "a/z.js", "a/z*.js")

			// stars.js line 30
			assertMatch(t, true, "ab", "*")
			// stars.js line 31
			assertMatch(t, true, "abc", "*")

			// stars.js line 33
			assertMatch(t, false, "bar", "f*")
			// stars.js line 34
			assertMatch(t, false, "foo", "*r")
			// stars.js line 35
			assertMatch(t, false, "foo", "b*")
			// stars.js line 36
			assertMatch(t, false, "foo/bar", "*")
			// stars.js line 37
			assertMatch(t, true, "abc", "*c")
			// stars.js line 38
			assertMatch(t, true, "abc", "a*")
			// stars.js line 39
			assertMatch(t, true, "abc", "a*c")
			// stars.js line 40
			assertMatch(t, true, "bar", "*r")
			// stars.js line 41
			assertMatch(t, true, "bar", "b*")
			// stars.js line 42
			assertMatch(t, true, "foo", "f*")
		})

		t.Run("should match spaces", func(t *testing.T) {
			// stars.js line 46
			assertMatch(t, true, "one abc two", "*abc*")
			// stars.js line 47
			assertMatch(t, true, "a         b", "a*b")
		})

		t.Run("should support multiple non-consecutive stars in a path segment", func(t *testing.T) {
			// stars.js line 51
			assertMatch(t, false, "foo", "*a*")
			// stars.js line 52
			assertMatch(t, true, "bar", "*a*")
			// stars.js line 53
			assertMatch(t, true, "oneabctwo", "*abc*")
			// stars.js line 54
			assertMatch(t, false, "a-b.c-d", "*-bc-*")
			// stars.js line 55
			assertMatch(t, true, "a-b.c-d", "*-*.*-*")
			// stars.js line 56
			assertMatch(t, true, "a-b.c-d", "*-b*c-*")
			// stars.js line 57
			assertMatch(t, true, "a-b.c-d", "*-b.c-*")
			// stars.js line 58
			assertMatch(t, true, "a-b.c-d", "*.*")
			// stars.js line 59
			assertMatch(t, true, "a-b.c-d", "*.*-*")
			// stars.js line 60
			assertMatch(t, true, "a-b.c-d", "*.*-d")
			// stars.js line 61
			assertMatch(t, true, "a-b.c-d", "*.c-*")
			// stars.js line 62
			assertMatch(t, true, "a-b.c-d", "*b.*d")
			// stars.js line 63
			assertMatch(t, true, "a-b.c-d", "a*.c*")
			// stars.js line 64
			assertMatch(t, true, "a-b.c-d", "a-*.*-d")
			// stars.js line 65
			assertMatch(t, true, "a.b", "*.*")
			// stars.js line 66
			assertMatch(t, true, "a.b", "*.b")
			// stars.js line 67
			assertMatch(t, true, "a.b", "a.*")
			// stars.js line 68
			assertMatch(t, true, "a.b", "a.b")
		})

		t.Run("should support multiple stars in a segment", func(t *testing.T) {
			// stars.js line 72
			assertMatch(t, false, "a-b.c-d", "**-bc-**")
			// stars.js line 73
			assertMatch(t, true, "a-b.c-d", "**-**.**-**")
			// stars.js line 74
			assertMatch(t, true, "a-b.c-d", "**-b**c-**")
			// stars.js line 75
			assertMatch(t, true, "a-b.c-d", "**-b.c-**")
			// stars.js line 76
			assertMatch(t, true, "a-b.c-d", "**.**")
			// stars.js line 77
			assertMatch(t, true, "a-b.c-d", "**.**-**")
			// stars.js line 78
			assertMatch(t, true, "a-b.c-d", "**.**-d")
			// stars.js line 79
			assertMatch(t, true, "a-b.c-d", "**.c-**")
			// stars.js line 80
			assertMatch(t, true, "a-b.c-d", "**b.**d")
			// stars.js line 81
			assertMatch(t, true, "a-b.c-d", "a**.c**")
			// stars.js line 82
			assertMatch(t, true, "a-b.c-d", "a-**.**-d")
			// stars.js line 83
			assertMatch(t, true, "a.b", "**.**")
			// stars.js line 84
			assertMatch(t, true, "a.b", "**.b")
			// stars.js line 85
			assertMatch(t, true, "a.b", "a.**")
			// stars.js line 86
			assertMatch(t, true, "a.b", "a.b")
		})

		t.Run("should return true when one of the given patterns matches the string", func(t *testing.T) {
			// stars.js line 90
			assertMatch(t, true, "/ab", "*/*")
			// stars.js line 91
			assertMatch(t, true, ".", ".")
			// stars.js line 92
			assertMatch(t, false, "a/.b", "a/")
			// stars.js line 93
			assertMatch(t, true, "/ab", "/*")
			// stars.js line 94
			assertMatch(t, true, "/ab", "/??")
			// stars.js line 95
			assertMatch(t, true, "/ab", "/?b")
			// stars.js line 96
			assertMatch(t, true, "/cd", "/*")
			// stars.js line 97
			assertMatch(t, true, "a", "a")
			// stars.js line 98
			assertMatch(t, true, "a/.b", "a/.*")
			// stars.js line 99
			assertMatch(t, true, "a/b", "?/?")
			// stars.js line 100
			assertMatch(t, true, "a/b/c/d/e/j/n/p/o/z/c.md", "a/**/j/**/z/*.md")
			// stars.js line 101
			assertMatch(t, true, "a/b/c/d/e/z/c.md", "a/**/z/*.md")
			// stars.js line 102
			assertMatch(t, true, "a/b/c/xyz.md", "a/b/c/*.md")
			// stars.js line 103
			assertMatch(t, true, "a/b/c/xyz.md", "a/b/c/*.md")
			// stars.js line 104
			assertMatch(t, true, "a/b/z/.a", "a/*/z/.a")
			// stars.js line 105
			assertMatch(t, false, "a/b/z/.a", "bz")
			// stars.js line 106
			assertMatch(t, true, "a/bb.bb/aa/b.b/aa/c/xyz.md", "a/**/c/*.md")
			// stars.js line 107
			assertMatch(t, true, "a/bb.bb/aa/bb/aa/c/xyz.md", "a/**/c/*.md")
			// stars.js line 108
			assertMatch(t, true, "a/bb.bb/c/xyz.md", "a/*/c/*.md")
			// stars.js line 109
			assertMatch(t, true, "a/bb/c/xyz.md", "a/*/c/*.md")
			// stars.js line 110
			assertMatch(t, true, "a/bbbb/c/xyz.md", "a/*/c/*.md")
			// stars.js line 111
			assertMatch(t, true, "aaa", "*")
			// stars.js line 112
			assertMatch(t, true, "ab", "*")
			// stars.js line 113
			assertMatch(t, true, "ab", "ab")
		})

		t.Run("should return false when the path does not match the pattern", func(t *testing.T) {
			// stars.js line 117
			assertMatch(t, false, "/ab", "*/")
			// stars.js line 118
			assertMatch(t, false, "/ab", "*/a")
			// stars.js line 119
			assertMatch(t, false, "/ab", "/")
			// stars.js line 120
			assertMatch(t, false, "/ab", "/?")
			// stars.js line 121
			assertMatch(t, false, "/ab", "/a")
			// stars.js line 122
			assertMatch(t, false, "/ab", "?/?")
			// stars.js line 123
			assertMatch(t, false, "/ab", "a/*")
			// stars.js line 124
			assertMatch(t, false, "a/.b", "a/")
			// stars.js line 125
			assertMatch(t, false, "a/b/c", "a/*")
			// stars.js line 126
			assertMatch(t, false, "a/b/c", "a/b")
			// stars.js line 127
			assertMatch(t, false, "a/b/c/d/e/z/c.md", "b/c/d/e")
			// stars.js line 128
			assertMatch(t, false, "a/b/z/.a", "b/z")
			// stars.js line 129
			assertMatch(t, false, "ab", "*/*")
			// stars.js line 130
			assertMatch(t, false, "ab", "/a")
			// stars.js line 131
			assertMatch(t, false, "ab", "a")
			// stars.js line 132
			assertMatch(t, false, "ab", "b")
			// stars.js line 133
			assertMatch(t, false, "ab", "c")
			// stars.js line 134
			assertMatch(t, false, "abcd", "ab")
			// stars.js line 135
			assertMatch(t, false, "abcd", "bc")
			// stars.js line 136
			assertMatch(t, false, "abcd", "c")
			// stars.js line 137
			assertMatch(t, false, "abcd", "cd")
			// stars.js line 138
			assertMatch(t, false, "abcd", "d")
			// stars.js line 139
			assertMatch(t, false, "abcd", "f")
			// stars.js line 140
			assertMatch(t, false, "ef", "/*")
		})

		t.Run("should match a path segment for each single star", func(t *testing.T) {
			// stars.js line 144
			assertMatch(t, false, "aaa", "*/*/*")
			// stars.js line 145
			assertMatch(t, false, "aaa/bb/aa/rr", "*/*/*")
			// stars.js line 146
			assertMatch(t, false, "aaa/bba/ccc", "aaa*")
			// stars.js line 147
			assertMatch(t, false, "aaa/bba/ccc", "aaa**")
			// stars.js line 148
			assertMatch(t, false, "aaa/bba/ccc", "aaa/*")
			// stars.js line 149
			assertMatch(t, false, "aaa/bba/ccc", "aaa/*ccc")
			// stars.js line 150
			assertMatch(t, false, "aaa/bba/ccc", "aaa/*z")
			// stars.js line 151
			assertMatch(t, false, "aaa/bbb", "*/*/*")
			// stars.js line 152
			assertMatch(t, false, "ab/zzz/ejkl/hi", "*/*jk*/*i")
			// stars.js line 153
			assertMatch(t, true, "aaa/bba/ccc", "*/*/*")
			// stars.js line 154
			assertMatch(t, true, "aaa/bba/ccc", "aaa/**")
			// stars.js line 155
			assertMatch(t, true, "aaa/bbb", "aaa/*")
			// stars.js line 156
			assertMatch(t, true, "ab/zzz/ejkl/hi", "*/*z*/*/*i")
			// stars.js line 157
			assertMatch(t, true, "abzzzejklhi", "*j*i")
		})

		t.Run("should support single globs (*)", func(t *testing.T) {
			// stars.js line 161
			assertMatch(t, true, "a", "*")
			// stars.js line 162
			assertMatch(t, true, "b", "*")
			// stars.js line 163
			assertMatch(t, false, "a/a", "*")
			// stars.js line 164
			assertMatch(t, false, "a/a/a", "*")
			// stars.js line 165
			assertMatch(t, false, "a/a/b", "*")
			// stars.js line 166
			assertMatch(t, false, "a/a/a/a", "*")
			// stars.js line 167
			assertMatch(t, false, "a/a/a/a/a", "*")

			// stars.js line 169
			assertMatch(t, false, "a", "*/*")
			// stars.js line 170
			assertMatch(t, true, "a/a", "*/*")
			// stars.js line 171
			assertMatch(t, false, "a/a/a", "*/*")

			// stars.js line 173
			assertMatch(t, false, "a", "*/*/*")
			// stars.js line 174
			assertMatch(t, false, "a/a", "*/*/*")
			// stars.js line 175
			assertMatch(t, true, "a/a/a", "*/*/*")
			// stars.js line 176
			assertMatch(t, false, "a/a/a/a", "*/*/*")

			// stars.js line 178
			assertMatch(t, false, "a", "*/*/*/*")
			// stars.js line 179
			assertMatch(t, false, "a/a", "*/*/*/*")
			// stars.js line 180
			assertMatch(t, false, "a/a/a", "*/*/*/*")
			// stars.js line 181
			assertMatch(t, true, "a/a/a/a", "*/*/*/*")
			// stars.js line 182
			assertMatch(t, false, "a/a/a/a/a", "*/*/*/*")

			// stars.js line 184
			assertMatch(t, false, "a", "*/*/*/*/*")
			// stars.js line 185
			assertMatch(t, false, "a/a", "*/*/*/*/*")
			// stars.js line 186
			assertMatch(t, false, "a/a/a", "*/*/*/*/*")
			// stars.js line 187
			assertMatch(t, false, "a/a/b", "*/*/*/*/*")
			// stars.js line 188
			assertMatch(t, false, "a/a/a/a", "*/*/*/*/*")
			// stars.js line 189
			assertMatch(t, true, "a/a/a/a/a", "*/*/*/*/*")
			// stars.js line 190
			assertMatch(t, false, "a/a/a/a/a/a", "*/*/*/*/*")

			// stars.js line 192
			assertMatch(t, false, "a", "a/*")
			// stars.js line 193
			assertMatch(t, true, "a/a", "a/*")
			// stars.js line 194
			assertMatch(t, false, "a/a/a", "a/*")
			// stars.js line 195
			assertMatch(t, false, "a/a/a/a", "a/*")
			// stars.js line 196
			assertMatch(t, false, "a/a/a/a/a", "a/*")

			// stars.js line 198
			assertMatch(t, false, "a", "a/*/*")
			// stars.js line 199
			assertMatch(t, false, "a/a", "a/*/*")
			// stars.js line 200
			assertMatch(t, true, "a/a/a", "a/*/*")
			// stars.js line 201
			assertMatch(t, false, "b/a/a", "a/*/*")
			// stars.js line 202
			assertMatch(t, false, "a/a/a/a", "a/*/*")
			// stars.js line 203
			assertMatch(t, false, "a/a/a/a/a", "a/*/*")

			// stars.js line 205
			assertMatch(t, false, "a", "a/*/*/*")
			// stars.js line 206
			assertMatch(t, false, "a/a", "a/*/*/*")
			// stars.js line 207
			assertMatch(t, false, "a/a/a", "a/*/*/*")
			// stars.js line 208
			assertMatch(t, true, "a/a/a/a", "a/*/*/*")
			// stars.js line 209
			assertMatch(t, false, "a/a/a/a/a", "a/*/*/*")

			// stars.js line 211
			assertMatch(t, false, "a", "a/*/*/*/*")
			// stars.js line 212
			assertMatch(t, false, "a/a", "a/*/*/*/*")
			// stars.js line 213
			assertMatch(t, false, "a/a/a", "a/*/*/*/*")
			// stars.js line 214
			assertMatch(t, false, "a/a/b", "a/*/*/*/*")
			// stars.js line 215
			assertMatch(t, false, "a/a/a/a", "a/*/*/*/*")
			// stars.js line 216
			assertMatch(t, true, "a/a/a/a/a", "a/*/*/*/*")

			// stars.js line 218
			assertMatch(t, false, "a", "a/*/a")
			// stars.js line 219
			assertMatch(t, false, "a/a", "a/*/a")
			// stars.js line 220
			assertMatch(t, true, "a/a/a", "a/*/a")
			// stars.js line 221
			assertMatch(t, false, "a/a/b", "a/*/a")
			// stars.js line 222
			assertMatch(t, false, "a/a/a/a", "a/*/a")
			// stars.js line 223
			assertMatch(t, false, "a/a/a/a/a", "a/*/a")

			// stars.js line 225
			assertMatch(t, false, "a", "a/*/b")
			// stars.js line 226
			assertMatch(t, false, "a/a", "a/*/b")
			// stars.js line 227
			assertMatch(t, false, "a/a/a", "a/*/b")
			// stars.js line 228
			assertMatch(t, true, "a/a/b", "a/*/b")
			// stars.js line 229
			assertMatch(t, false, "a/a/a/a", "a/*/b")
			// stars.js line 230
			assertMatch(t, false, "a/a/a/a/a", "a/*/b")
		})

		t.Run("should only match a single folder per star when globstars are used", func(t *testing.T) {
			// stars.js line 234
			assertMatch(t, false, "a", "*/**/a")
			// stars.js line 235
			assertMatch(t, false, "a/a/b", "*/**/a")
			// stars.js line 236
			assertMatch(t, true, "a/a", "*/**/a")
			// stars.js line 237
			assertMatch(t, true, "a/a/a", "*/**/a")
			// stars.js line 238
			assertMatch(t, true, "a/a/a/a", "*/**/a")
			// stars.js line 239
			assertMatch(t, true, "a/a/a/a/a", "*/**/a")
		})

		t.Run("should not match a trailing slash when a star is last char", func(t *testing.T) {
			// stars.js line 243
			assertMatch(t, false, "a", "*/")
			// stars.js line 244
			assertMatch(t, false, "a", "*/*")
			// stars.js line 245
			assertMatch(t, false, "a", "a/*")
			// stars.js line 246
			assertMatch(t, false, "a/", "*/*")
			// stars.js line 247
			assertMatch(t, false, "a/", "a/*")
			// stars.js line 248
			assertMatch(t, false, "a/a", "*")
			// stars.js line 249
			assertMatch(t, false, "a/a", "*/")
			// stars.js line 250
			assertMatch(t, false, "a/x/y", "*/")
			// stars.js line 251
			assertMatch(t, false, "a/x/y", "*/*")
			// stars.js line 252
			assertMatch(t, false, "a/x/y", "a/*")
			// stars.js line 253
			assertMatch(t, false, "a/", "*", &Options{StrictSlashes: true})
			// stars.js line 254
			assertMatch(t, true, "a/", "*")
			// stars.js line 255
			assertMatch(t, true, "a", "*")
			// stars.js line 256
			assertMatch(t, true, "a/", "*/")
			// stars.js line 257
			assertMatch(t, true, "a/", "*{,/}")
			// stars.js line 258
			assertMatch(t, true, "a/a", "*/*")
			// stars.js line 259
			assertMatch(t, true, "a/a", "a/*")
		})

		t.Run("should work with file extensions", func(t *testing.T) {
			// stars.js line 263
			assertMatch(t, false, "a.txt", "a/**/*.txt")
			// stars.js line 264
			assertMatch(t, true, "a/x/y.txt", "a/**/*.txt")
			// stars.js line 265
			assertMatch(t, false, "a/x/y/z", "a/**/*.txt")

			// stars.js line 267
			assertMatch(t, false, "a.txt", "a/*.txt")
			// stars.js line 268
			assertMatch(t, true, "a/b.txt", "a/*.txt")
			// stars.js line 269
			assertMatch(t, false, "a/x/y.txt", "a/*.txt")
			// stars.js line 270
			assertMatch(t, false, "a/x/y/z", "a/*.txt")

			// stars.js line 272
			assertMatch(t, true, "a.txt", "a*.txt")
			// stars.js line 273
			assertMatch(t, false, "a/b.txt", "a*.txt")
			// stars.js line 274
			assertMatch(t, false, "a/x/y.txt", "a*.txt")
			// stars.js line 275
			assertMatch(t, false, "a/x/y/z", "a*.txt")

			// stars.js line 277
			assertMatch(t, true, "a.txt", "*.txt")
			// stars.js line 278
			assertMatch(t, false, "a/b.txt", "*.txt")
			// stars.js line 279
			assertMatch(t, false, "a/x/y.txt", "*.txt")
			// stars.js line 280
			assertMatch(t, false, "a/x/y/z", "*.txt")
		})

		t.Run("should not match slashes when globstars are not exclusive in a path segment", func(t *testing.T) {
			// stars.js line 284
			assertMatch(t, false, "foo/baz/bar", "foo**bar")
			// stars.js line 285
			assertMatch(t, true, "foobazbar", "foo**bar")
		})

		t.Run("should match slashes when defined in braces", func(t *testing.T) {
			// stars.js line 289
			assertMatch(t, true, "foo", "foo{,/**}")
		})

		t.Run("should correctly match slashes", func(t *testing.T) {
			// stars.js line 293
			assertMatch(t, false, "a/b", "a*")
			// stars.js line 294
			assertMatch(t, false, "a/a/bb", "a/**/b")
			// stars.js line 295
			assertMatch(t, false, "a/bb", "a/**/b")

			// stars.js line 297
			assertMatch(t, false, "foo", "*/**")
			// stars.js line 298
			assertMatch(t, false, "foo/bar", "**/")
			// stars.js line 299
			assertMatch(t, false, "foo/bar", "**/*/")
			// stars.js line 300
			assertMatch(t, false, "foo/bar", "*/*/")
			// stars.js line 301
			assertMatch(t, false, "foo/bar/", "**/*", &Options{StrictSlashes: true})

			// stars.js line 303
			assertMatch(t, true, "/home/foo/..", "**/..") // stars.js line 303
			// stars.js line 304
			assertMatch(t, true, "a", "**/a")
			// stars.js line 305
			assertMatch(t, true, "a/a", "**")
			// stars.js line 306
			assertMatch(t, true, "a/a", "a/**")
			// stars.js line 307
			assertMatch(t, true, "a/", "a/**")
			// stars.js line 308
			assertMatch(t, true, "a", "a/**")
			// stars.js line 309
			assertMatch(t, false, "a/a", "**/")
			// stars.js line 310
			assertMatch(t, true, "a", "**/a/**")
			// stars.js line 311
			assertMatch(t, true, "a", "a/**")
			// stars.js line 312
			assertMatch(t, false, "a/a", "**/")
			// stars.js line 313
			assertMatch(t, true, "a/a", "*/**/a")
			// stars.js line 314
			assertMatch(t, true, "a", "a/**")
			// stars.js line 315
			assertMatch(t, true, "foo/", "*/**")
			// stars.js line 316
			assertMatch(t, true, "foo/bar", "**/*")
			// stars.js line 317
			assertMatch(t, true, "foo/bar", "*/*")
			// stars.js line 318
			assertMatch(t, true, "foo/bar", "*/**")
			// stars.js line 319
			assertMatch(t, true, "foo/bar/", "**/")
			// stars.js line 320
			assertMatch(t, true, "foo/bar/", "**/*")
			// stars.js line 321
			assertMatch(t, true, "foo/bar/", "**/*/")
			// stars.js line 322
			assertMatch(t, true, "foo/bar/", "*/**")
			// stars.js line 323
			assertMatch(t, true, "foo/bar/", "*/*/")

			// stars.js line 325
			assertMatch(t, false, "bar/baz/foo", "*/foo")
			// stars.js line 326
			assertMatch(t, false, "deep/foo/bar", "**/bar/*")
			// stars.js line 327
			assertMatch(t, false, "deep/foo/bar/baz/x", "*/bar/**")
			// stars.js line 328
			assertMatch(t, false, "ef", "/*")
			// stars.js line 329
			assertMatch(t, false, "foo/bar", "foo?bar")
			// stars.js line 330
			assertMatch(t, false, "foo/bar/baz", "**/bar*")
			// stars.js line 331
			assertMatch(t, false, "foo/bar/baz", "**/bar**")
			// stars.js line 332
			assertMatch(t, false, "foo/baz/bar", "foo**bar")
			// stars.js line 333
			assertMatch(t, false, "foo/baz/bar", "foo*bar")
			// stars.js line 334
			assertMatch(t, true, "foo", "foo/**")
			// stars.js line 335
			assertMatch(t, true, "/ab", "/*")
			// stars.js line 336
			assertMatch(t, true, "/cd", "/*")
			// stars.js line 337
			assertMatch(t, true, "/ef", "/*")
			// stars.js line 338
			assertMatch(t, true, "a/b/j/c/z/x.md", "a/**/j/**/z/*.md")
			// stars.js line 339
			assertMatch(t, true, "a/j/z/x.md", "a/**/j/**/z/*.md")

			// stars.js line 341
			assertMatch(t, true, "bar/baz/foo", "**/foo")
			// stars.js line 342
			assertMatch(t, true, "deep/foo/bar/baz", "**/bar/*")
			// stars.js line 343
			assertMatch(t, true, "deep/foo/bar/baz/", "**/bar/**")
			// stars.js line 344
			assertMatch(t, true, "deep/foo/bar/baz/x", "**/bar/*/*")
			// stars.js line 345
			assertMatch(t, true, "foo/b/a/z/bar", "foo/**/**/bar")
			// stars.js line 346
			assertMatch(t, true, "foo/b/a/z/bar", "foo/**/bar")
			// stars.js line 347
			assertMatch(t, true, "foo/bar", "foo/**/**/bar")
			// stars.js line 348
			assertMatch(t, true, "foo/bar", "foo/**/bar")
			// stars.js line 349
			assertMatch(t, true, "foo/bar/baz/x", "*/bar/**")
			// stars.js line 350
			assertMatch(t, true, "foo/baz/bar", "foo/**/**/bar")
			// stars.js line 351
			assertMatch(t, true, "foo/baz/bar", "foo/**/bar")
			// stars.js line 352
			assertMatch(t, true, "XXX/foo", "**/foo")
		})

		t.Run("should ignore leading ./ when defined on pattern", func(t *testing.T) {
			// stars.js line 356
			assertMatch(t, true, "ab", "./*")
			// stars.js line 357
			assertMatch(t, false, "ab", "./*/")
			// stars.js line 358
			assertMatch(t, true, "ab/", "./*/")
		})

		t.Run("should optionally match trailing slashes with braces", func(t *testing.T) {
			// stars.js line 362
			assertMatch(t, true, "foo", "**/*")
			// stars.js line 363
			assertMatch(t, true, "foo", "**/*{,/}")
			// stars.js line 364
			assertMatch(t, true, "foo/", "**/*{,/}")
			// stars.js line 365
			assertMatch(t, true, "foo/bar", "**/*{,/}")
			// stars.js line 366
			assertMatch(t, true, "foo/bar/", "**/*{,/}")
		})
	})
}
