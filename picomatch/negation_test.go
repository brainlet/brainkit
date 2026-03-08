// negation_test.go — Faithful 1:1 port of picomatch/test/negation.js
package picomatch

import "testing"

func TestNegation(t *testing.T) {
	t.Run("should treat patterns with a leading ! as negated/inverted globs", func(t *testing.T) {
		// negation.js line 8
		assertMatch(t, false, "abc", "!*")
		// negation.js line 9
		assertMatch(t, false, "abc", "!abc")
		// negation.js line 10
		assertMatch(t, false, "bar.md", "*!.md")
		// negation.js line 11
		assertMatch(t, false, "bar.md", "foo!.md")
		// negation.js line 12
		assertMatch(t, false, "foo!.md", "\\!*!*.md")
		// negation.js line 13
		assertMatch(t, false, "foo!bar.md", "\\!*!*.md")
		// negation.js line 14
		assertMatch(t, true, "!foo!.md", "*!*.md")
		// negation.js line 15
		assertMatch(t, true, "!foo!.md", "\\!*!*.md")
		// negation.js line 16
		assertMatch(t, true, "abc", "!*foo")
		// negation.js line 17
		assertMatch(t, true, "abc", "!foo*")
		// negation.js line 18
		assertMatch(t, true, "abc", "!xyz")
		// negation.js line 19
		assertMatch(t, true, "ba!r.js", "*!*.*")
		// negation.js line 20
		assertMatch(t, true, "bar.md", "*.md")
		// negation.js line 21
		assertMatch(t, true, "foo!.md", "*!*.*")
		// negation.js line 22
		assertMatch(t, true, "foo!.md", "*!*.md")
		// negation.js line 23
		assertMatch(t, true, "foo!.md", "*!.md")
		// negation.js line 24
		assertMatch(t, true, "foo!.md", "*.md")
		// negation.js line 25
		assertMatch(t, true, "foo!.md", "foo!.md")
		// negation.js line 26
		assertMatch(t, true, "foo!bar.md", "*!*.md")
		// negation.js line 27
		assertMatch(t, true, "foobar.md", "*b*.md")
	})

	t.Run("should treat non-leading ! as literal characters", func(t *testing.T) {
		// negation.js line 31
		assertMatch(t, false, "a", "a!!b")
		// negation.js line 32
		assertMatch(t, false, "aa", "a!!b")
		// negation.js line 33
		assertMatch(t, false, "a/b", "a!!b")
		// negation.js line 34
		assertMatch(t, false, "a!b", "a!!b")
		// negation.js line 35
		assertMatch(t, true, "a!!b", "a!!b")
		// negation.js line 36
		assertMatch(t, false, "a/!!/b", "a!!b")
	})

	t.Run("should support negation in globs that have no other special characters", func(t *testing.T) {
		// negation.js line 40
		assertMatch(t, false, "a/b", "!a/b")
		// negation.js line 41
		assertMatch(t, true, "a", "!a/b")
		// negation.js line 42
		assertMatch(t, true, "a.b", "!a/b")
		// negation.js line 43
		assertMatch(t, true, "a/a", "!a/b")
		// negation.js line 44
		assertMatch(t, true, "a/c", "!a/b")
		// negation.js line 45
		assertMatch(t, true, "b/a", "!a/b")
		// negation.js line 46
		assertMatch(t, true, "b/b", "!a/b")
		// negation.js line 47
		assertMatch(t, true, "b/c", "!a/b")
	})

	t.Run("should support multiple leading ! to toggle negation", func(t *testing.T) {
		// negation.js line 51
		assertMatch(t, false, "abc", "!abc")
		// negation.js line 52
		assertMatch(t, true, "abc", "!!abc")
		// negation.js line 53
		assertMatch(t, false, "abc", "!!!abc")
		// negation.js line 54
		assertMatch(t, true, "abc", "!!!!abc")
		// negation.js line 55
		assertMatch(t, false, "abc", "!!!!!abc")
		// negation.js line 56
		assertMatch(t, true, "abc", "!!!!!!abc")
		// negation.js line 57
		assertMatch(t, false, "abc", "!!!!!!!abc")
		// negation.js line 58
		assertMatch(t, true, "abc", "!!!!!!!!abc")
	})

	t.Run("should support negation extglobs after leading !", func(t *testing.T) {
		// negation.js line 62
		assertMatch(t, false, "abc", "!(abc)")
		// negation.js line 63
		assertMatch(t, true, "abc", "!!(abc)")
		// negation.js line 64
		assertMatch(t, false, "abc", "!!!(abc)")
		// negation.js line 65
		assertMatch(t, true, "abc", "!!!!(abc)")
		// negation.js line 66
		assertMatch(t, false, "abc", "!!!!!(abc)")
		// negation.js line 67
		assertMatch(t, true, "abc", "!!!!!!(abc)")
		// negation.js line 68
		assertMatch(t, false, "abc", "!!!!!!!(abc)")
		// negation.js line 69
		assertMatch(t, true, "abc", "!!!!!!!!(abc)")
	})

	t.Run("should support negation with globs", func(t *testing.T) {
		// negation.js line 73
		assertMatch(t, false, "a/a", "!(*/*)")
		// negation.js line 74
		assertMatch(t, false, "a/b", "!(*/*)")
		// negation.js line 75
		assertMatch(t, false, "a/c", "!(*/*)")
		// negation.js line 76
		assertMatch(t, false, "b/a", "!(*/*)")
		// negation.js line 77
		assertMatch(t, false, "b/b", "!(*/*)")
		// negation.js line 78
		assertMatch(t, false, "b/c", "!(*/*)")
		// negation.js line 79
		assertMatch(t, false, "a/b", "!(*/b)")
		// negation.js line 80
		assertMatch(t, false, "b/b", "!(*/b)")
		// negation.js line 81
		assertMatch(t, false, "a/b", "!(a/b)")
		// negation.js line 82
		assertMatch(t, false, "a", "!*")
		// negation.js line 83
		assertMatch(t, false, "a.b", "!*")
		// negation.js line 84
		assertMatch(t, false, "a/a", "!*/*")
		// negation.js line 85
		assertMatch(t, false, "a/b", "!*/*")
		// negation.js line 86
		assertMatch(t, false, "a/c", "!*/*")
		// negation.js line 87
		assertMatch(t, false, "b/a", "!*/*")
		// negation.js line 88
		assertMatch(t, false, "b/b", "!*/*")
		// negation.js line 89
		assertMatch(t, false, "b/c", "!*/*")
		// negation.js line 90
		assertMatch(t, false, "a/b", "!*/b")
		// negation.js line 91
		assertMatch(t, false, "b/b", "!*/b")
		// negation.js line 92
		assertMatch(t, false, "a/c", "!*/c")
		// negation.js line 93
		assertMatch(t, false, "a/c", "!*/c")
		// negation.js line 94
		assertMatch(t, false, "b/c", "!*/c")
		// negation.js line 95
		assertMatch(t, false, "b/c", "!*/c")
		// negation.js line 96
		assertMatch(t, false, "bar", "!*a*")
		// negation.js line 97
		assertMatch(t, false, "fab", "!*a*")
		// negation.js line 98
		assertMatch(t, false, "a/a", "!a/(*)")
		// negation.js line 99
		assertMatch(t, false, "a/b", "!a/(*)")
		// negation.js line 100
		assertMatch(t, false, "a/c", "!a/(*)")
		// negation.js line 101
		assertMatch(t, false, "a/b", "!a/(b)")
		// negation.js line 102
		assertMatch(t, false, "a/a", "!a/*")
		// negation.js line 103
		assertMatch(t, false, "a/b", "!a/*")
		// negation.js line 104
		assertMatch(t, false, "a/c", "!a/*")
		// negation.js line 105
		assertMatch(t, false, "fab", "!f*b")
		// negation.js line 106
		assertMatch(t, true, "a", "!(*/*)")
		// negation.js line 107
		assertMatch(t, true, "a.b", "!(*/*)")
		// negation.js line 108
		assertMatch(t, true, "a", "!(*/b)")
		// negation.js line 109
		assertMatch(t, true, "a.b", "!(*/b)")
		// negation.js line 110
		assertMatch(t, true, "a/a", "!(*/b)")
		// negation.js line 111
		assertMatch(t, true, "a/c", "!(*/b)")
		// negation.js line 112
		assertMatch(t, true, "b/a", "!(*/b)")
		// negation.js line 113
		assertMatch(t, true, "b/c", "!(*/b)")
		// negation.js line 114
		assertMatch(t, true, "a", "!(a/b)")
		// negation.js line 115
		assertMatch(t, true, "a.b", "!(a/b)")
		// negation.js line 116
		assertMatch(t, true, "a/a", "!(a/b)")
		// negation.js line 117
		assertMatch(t, true, "a/c", "!(a/b)")
		// negation.js line 118
		assertMatch(t, true, "b/a", "!(a/b)")
		// negation.js line 119
		assertMatch(t, true, "b/b", "!(a/b)")
		// negation.js line 120
		assertMatch(t, true, "b/c", "!(a/b)")
		// negation.js line 121
		assertMatch(t, true, "a/a", "!*")
		// negation.js line 122
		assertMatch(t, true, "a/b", "!*")
		// negation.js line 123
		assertMatch(t, true, "a/c", "!*")
		// negation.js line 124
		assertMatch(t, true, "b/a", "!*")
		// negation.js line 125
		assertMatch(t, true, "b/b", "!*")
		// negation.js line 126
		assertMatch(t, true, "b/c", "!*")
		// negation.js line 127
		assertMatch(t, true, "a", "!*/*")
		// negation.js line 128
		assertMatch(t, true, "a.b", "!*/*")
		// negation.js line 129
		assertMatch(t, true, "a", "!*/b")
		// negation.js line 130
		assertMatch(t, true, "a.b", "!*/b")
		// negation.js line 131
		assertMatch(t, true, "a/a", "!*/b")
		// negation.js line 132
		assertMatch(t, true, "a/c", "!*/b")
		// negation.js line 133
		assertMatch(t, true, "b/a", "!*/b")
		// negation.js line 134
		assertMatch(t, true, "b/c", "!*/b")
		// negation.js line 135
		assertMatch(t, true, "a", "!*/c")
		// negation.js line 136
		assertMatch(t, true, "a.b", "!*/c")
		// negation.js line 137
		assertMatch(t, true, "a/a", "!*/c")
		// negation.js line 138
		assertMatch(t, true, "a/b", "!*/c")
		// negation.js line 139
		assertMatch(t, true, "b/a", "!*/c")
		// negation.js line 140
		assertMatch(t, true, "b/b", "!*/c")
		// negation.js line 141
		assertMatch(t, true, "foo", "!*a*")
		// negation.js line 142
		assertMatch(t, true, "a", "!a/(*)")
		// negation.js line 143
		assertMatch(t, true, "a.b", "!a/(*)")
		// negation.js line 144
		assertMatch(t, true, "b/a", "!a/(*)")
		// negation.js line 145
		assertMatch(t, true, "b/b", "!a/(*)")
		// negation.js line 146
		assertMatch(t, true, "b/c", "!a/(*)")
		// negation.js line 147
		assertMatch(t, true, "a", "!a/(b)")
		// negation.js line 148
		assertMatch(t, true, "a.b", "!a/(b)")
		// negation.js line 149
		assertMatch(t, true, "a/a", "!a/(b)")
		// negation.js line 150
		assertMatch(t, true, "a/c", "!a/(b)")
		// negation.js line 151
		assertMatch(t, true, "b/a", "!a/(b)")
		// negation.js line 152
		assertMatch(t, true, "b/b", "!a/(b)")
		// negation.js line 153
		assertMatch(t, true, "b/c", "!a/(b)")
		// negation.js line 154
		assertMatch(t, true, "a", "!a/*")
		// negation.js line 155
		assertMatch(t, true, "a.b", "!a/*")
		// negation.js line 156
		assertMatch(t, true, "b/a", "!a/*")
		// negation.js line 157
		assertMatch(t, true, "b/b", "!a/*")
		// negation.js line 158
		assertMatch(t, true, "b/c", "!a/*")
		// negation.js line 159
		assertMatch(t, true, "bar", "!f*b")
		// negation.js line 160
		assertMatch(t, true, "foo", "!f*b")
	})

	t.Run("should negate files with extensions", func(t *testing.T) {
		// negation.js line 164
		assertMatch(t, false, ".md", "!.md")
		// negation.js line 165
		assertMatch(t, true, "a.js", "!**/*.md")
		// negation.js line 166
		assertMatch(t, false, "b.md", "!**/*.md")
		// negation.js line 167
		assertMatch(t, true, "c.txt", "!**/*.md")
		// negation.js line 168
		assertMatch(t, true, "a.js", "!*.md")
		// negation.js line 169
		assertMatch(t, false, "b.md", "!*.md")
		// negation.js line 170
		assertMatch(t, true, "c.txt", "!*.md")
		// negation.js line 171
		assertMatch(t, false, "abc.md", "!*.md")
		// negation.js line 172
		assertMatch(t, true, "abc.txt", "!*.md")
		// negation.js line 173
		assertMatch(t, false, "foo.md", "!*.md")
		// negation.js line 174
		assertMatch(t, true, "foo.md", "!.md")
	})

	t.Run("should support negated single stars", func(t *testing.T) {
		// negation.js line 178
		assertMatch(t, true, "a.js", "!*.md")
		// negation.js line 179
		assertMatch(t, true, "b.txt", "!*.md")
		// negation.js line 180
		assertMatch(t, false, "c.md", "!*.md")
		// negation.js line 181
		assertMatch(t, false, "a/a/a.js", "!a/*/a.js")
		// negation.js line 182
		assertMatch(t, false, "a/b/a.js", "!a/*/a.js")
		// negation.js line 183
		assertMatch(t, false, "a/c/a.js", "!a/*/a.js")
		// negation.js line 184
		assertMatch(t, false, "a/a/a/a.js", "!a/*/*/a.js")
		// negation.js line 185
		assertMatch(t, true, "b/a/b/a.js", "!a/*/*/a.js")
		// negation.js line 186
		assertMatch(t, true, "c/a/c/a.js", "!a/*/*/a.js")
		// negation.js line 187
		assertMatch(t, false, "a/a.txt", "!a/a*.txt")
		// negation.js line 188
		assertMatch(t, true, "a/b.txt", "!a/a*.txt")
		// negation.js line 189
		assertMatch(t, true, "a/c.txt", "!a/a*.txt")
		// negation.js line 190
		assertMatch(t, false, "a.a.txt", "!a.a*.txt")
		// negation.js line 191
		assertMatch(t, true, "a.b.txt", "!a.a*.txt")
		// negation.js line 192
		assertMatch(t, true, "a.c.txt", "!a.a*.txt")
		// negation.js line 193
		assertMatch(t, false, "a/a.txt", "!a/*.txt")
		// negation.js line 194
		assertMatch(t, false, "a/b.txt", "!a/*.txt")
		// negation.js line 195
		assertMatch(t, false, "a/c.txt", "!a/*.txt")
	})

	t.Run("should support negated globstars (multiple stars)", func(t *testing.T) {
		// negation.js line 199
		assertMatch(t, true, "a.js", "!*.md")
		// negation.js line 200
		assertMatch(t, true, "b.txt", "!*.md")
		// negation.js line 201
		assertMatch(t, false, "c.md", "!*.md")
		// negation.js line 202
		assertMatch(t, false, "a/a/a.js", "!**/a.js")
		// negation.js line 203
		assertMatch(t, false, "a/b/a.js", "!**/a.js")
		// negation.js line 204
		assertMatch(t, false, "a/c/a.js", "!**/a.js")
		// negation.js line 205
		assertMatch(t, true, "a/a/b.js", "!**/a.js")
		// negation.js line 206
		assertMatch(t, false, "a/a/a/a.js", "!a/**/a.js")
		// negation.js line 207
		assertMatch(t, true, "b/a/b/a.js", "!a/**/a.js")
		// negation.js line 208
		assertMatch(t, true, "c/a/c/a.js", "!a/**/a.js")
		// negation.js line 209
		assertMatch(t, true, "a/b.js", "!**/*.md")
		// negation.js line 210
		assertMatch(t, true, "a.js", "!**/*.md")
		// negation.js line 211
		assertMatch(t, false, "a/b.md", "!**/*.md")
		// negation.js line 212
		assertMatch(t, false, "a.md", "!**/*.md")
		// negation.js line 213
		assertMatch(t, false, "a/b.js", "**/*.md")
		// negation.js line 214
		assertMatch(t, false, "a.js", "**/*.md")
		// negation.js line 215
		assertMatch(t, true, "a/b.md", "**/*.md")
		// negation.js line 216
		assertMatch(t, true, "a.md", "**/*.md")
		// negation.js line 217
		assertMatch(t, true, "a/b.js", "!**/*.md")
		// negation.js line 218
		assertMatch(t, true, "a.js", "!**/*.md")
		// negation.js line 219
		assertMatch(t, false, "a/b.md", "!**/*.md")
		// negation.js line 220
		assertMatch(t, false, "a.md", "!**/*.md")
		// negation.js line 221
		assertMatch(t, true, "a/b.js", "!*.md")
		// negation.js line 222
		assertMatch(t, true, "a.js", "!*.md")
		// negation.js line 223
		assertMatch(t, true, "a/b.md", "!*.md")
		// negation.js line 224
		assertMatch(t, false, "a.md", "!*.md")
		// negation.js line 225
		assertMatch(t, true, "a.js", "!**/*.md")
		// negation.js line 226
		assertMatch(t, false, "b.md", "!**/*.md")
		// negation.js line 227
		assertMatch(t, true, "c.txt", "!**/*.md")
	})

	t.Run("should not negate when inside quoted strings", func(t *testing.T) {
		// negation.js line 231
		assertMatch(t, false, "foo.md", "\"!*\".md")
		// negation.js line 232
		assertMatch(t, true, "\"!*\".md", "\"!*\".md")
		// negation.js line 233
		assertMatch(t, true, "!*.md", "\"!*\".md")

		keepQuotes := &Options{KeepQuotes: true}
		// negation.js line 235
		assertMatch(t, false, "foo.md", "\"!*\".md", keepQuotes)
		// negation.js line 236
		assertMatch(t, true, "\"!*\".md", "\"!*\".md", keepQuotes)
		// negation.js line 237
		assertMatch(t, false, "!*.md", "\"!*\".md", keepQuotes)

		// negation.js line 239
		assertMatch(t, false, "foo.md", "\"**\".md")
		// negation.js line 240
		assertMatch(t, true, "\"**\".md", "\"**\".md")
		// negation.js line 241
		assertMatch(t, true, "**.md", "\"**\".md")

		// negation.js line 243
		assertMatch(t, false, "foo.md", "\"**\".md", keepQuotes)
		// negation.js line 244
		assertMatch(t, true, "\"**\".md", "\"**\".md", keepQuotes)
		// negation.js line 245
		assertMatch(t, false, "**.md", "\"**\".md", keepQuotes)
	})

	t.Run("should negate dotfiles", func(t *testing.T) {
		// negation.js line 249
		assertMatch(t, false, ".dotfile.md", "!.*.md")
		// negation.js line 250
		assertMatch(t, true, ".dotfile.md", "!*.md")
		// negation.js line 251
		assertMatch(t, true, ".dotfile.txt", "!*.md")
		// negation.js line 252
		assertMatch(t, true, ".dotfile.txt", "!*.md")
		// negation.js line 253
		assertMatch(t, true, "a/b/.dotfile", "!*.md")
		// negation.js line 254
		assertMatch(t, false, ".gitignore", "!.gitignore")
		// negation.js line 255
		assertMatch(t, true, "a", "!.gitignore")
		// negation.js line 256
		assertMatch(t, true, "b", "!.gitignore")
	})

	t.Run("should not match slashes with a single star", func(t *testing.T) {
		// negation.js line 260
		assertMatch(t, true, "foo/bar.md", "!*.md")
		// negation.js line 261
		assertMatch(t, false, "foo.md", "!*.md")
	})

	t.Run("should match nested directories with globstars", func(t *testing.T) {
		// negation.js line 265
		assertMatch(t, false, "a", "!a/**")
		// negation.js line 266
		assertMatch(t, false, "a/", "!a/**")
		// negation.js line 267
		assertMatch(t, false, "a/b", "!a/**")
		// negation.js line 268
		assertMatch(t, false, "a/b/c", "!a/**")
		// negation.js line 269
		assertMatch(t, true, "b", "!a/**")
		// negation.js line 270
		assertMatch(t, true, "b/c", "!a/**")

		// negation.js line 272
		assertMatch(t, true, "foo", "!f*b")
		// negation.js line 273
		assertMatch(t, true, "bar", "!f*b")
		// negation.js line 274
		assertMatch(t, false, "fab", "!f*b")
	})
}
