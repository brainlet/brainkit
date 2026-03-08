// globstars_test.go — Faithful 1:1 port of picomatch/test/globstars.js
package picomatch

import (
	"testing"
)

func TestGlobstars(t *testing.T) {
	t.Run("issue related", func(t *testing.T) {
		t.Run("should match paths with no slashes (micromatch/#15)", func(t *testing.T) {
			// test/globstars.js line 10
			assertMatch(t, true, "a.js", "**/*.js")
			// test/globstars.js line 11
			assertMatch(t, true, "a.js", "**/a*")
			// test/globstars.js line 12
			assertMatch(t, true, "a.js", "**/a*.js")
			// test/globstars.js line 13
			assertMatch(t, true, "abc", "**/abc")
		})

		t.Run("should regard non-exclusive double-stars as single stars", func(t *testing.T) {
			fixtures := []string{"a", "a/", "a/a", "a/a/", "a/a/a", "a/a/a/", "a/a/a/a", "a/a/a/a/", "a/a/a/a/a", "a/a/a/a/a/", "a/a/b", "a/a/b/", "a/b", "a/b/", "a/b/c/.d/e/", "a/c", "a/c/", "a/b", "a/x/", "b", "b/", "x/y", "x/y/", "z/z", "z/z/"}

			// test/globstars.js line 19
			assertMatchList(t, fixtures, "**a/a/*/", []string{"a/a/a/", "a/a/b/"})
			// test/globstars.js line 20
			assertMatch(t, false, "aaa/bba/ccc", "aaa/**ccc")
			// test/globstars.js line 21
			assertMatch(t, false, "aaa/bba/ccc", "aaa/**z")
			// test/globstars.js line 22
			assertMatch(t, true, "aaa/bba/ccc", "aaa/**b**/ccc")
			// test/globstars.js line 23
			assertMatch(t, false, "a/b/c", "**c")
			// test/globstars.js line 24
			assertMatch(t, false, "a/b/c", "a/**c")
			// test/globstars.js line 25
			assertMatch(t, false, "a/b/c", "a/**z")
			// test/globstars.js line 26
			assertMatch(t, false, "a/b/c/b/c", "a/**b**/c")
			// test/globstars.js line 27
			assertMatch(t, false, "a/b/c/d/e.js", "a/b/c**/*.js")
			// test/globstars.js line 28
			assertMatch(t, true, "a/b/c/b/c", "a/**/b/**/c")
			// test/globstars.js line 29
			assertMatch(t, true, "a/aba/c", "a/**b**/c")
			// test/globstars.js line 30
			assertMatch(t, true, "a/b/c", "a/**b**/c")
			// test/globstars.js line 31
			assertMatch(t, true, "a/b/c/d.js", "a/b/c**/*.js")
		})

		t.Run("should support globstars followed by braces", func(t *testing.T) {
			// test/globstars.js line 35
			assertMatch(t, true, "a/b/c/d/e/z/foo.md", "a/**/c/**{,(/z|/x)}/*.md")
			// test/globstars.js line 36
			assertMatch(t, true, "a/b/c/d/e/z/foo.md", "a/**{,(/x|/z)}/*.md")
		})

		t.Run("should support globstars followed by braces with nested extglobs", func(t *testing.T) {
			// test/globstars.js line 40
			assertMatch(t, true, "/x/foo.md", "@(/x|/z)/*.md")
			// test/globstars.js line 41
			assertMatch(t, true, "/z/foo.md", "@(/x|/z)/*.md")
			// test/globstars.js line 42
			assertMatch(t, true, "a/b/c/d/e/z/foo.md", "a/**/c/**@(/z|/x)/*.md")
			// test/globstars.js line 43
			assertMatch(t, true, "a/b/c/d/e/z/foo.md", "a/**@(/x|/z)/*.md")
		})

		t.Run("should support multiple globstars in one pattern", func(t *testing.T) {
			// test/globstars.js line 47
			assertMatch(t, false, "a/b/c/d/e/z/foo.md", "a/**/j/**/z/*.md")
			// test/globstars.js line 48
			assertMatch(t, false, "a/b/c/j/e/z/foo.txt", "a/**/j/**/z/*.md")
			// test/globstars.js line 49
			assertMatch(t, true, "a/b/c/d/e/j/n/p/o/z/foo.md", "a/**/j/**/z/*.md")
			// test/globstars.js line 50
			assertMatch(t, true, "a/b/c/d/e/z/foo.md", "a/**/z/*.md")
			// test/globstars.js line 51
			assertMatch(t, true, "a/b/c/j/e/z/foo.md", "a/**/j/**/z/*.md")
		})

		t.Run("should match file extensions", func(t *testing.T) {
			// test/globstars.js line 55
			assertMatchList(t, []string{".md", "a.md", "a/b/c.md", ".txt"}, "**/*.md", []string{"a.md", "a/b/c.md"})
			// test/globstars.js line 56
			assertMatchList(t, []string{".md/.md", ".md", "a/.md", "a/b/.md"}, "**/.md", []string{".md", "a/.md", "a/b/.md"})
			// test/globstars.js line 57
			assertMatchList(t, []string{".md/.md", ".md/foo/.md", ".md", "a/.md", "a/b/.md"}, ".md/**/.md", []string{".md/.md", ".md/foo/.md"})
		})

		t.Run("should respect trailing slashes on patterns", func(t *testing.T) {
			fixtures := []string{"a", "a/", "a/a", "a/a/", "a/a/a", "a/a/a/", "a/a/a/a", "a/a/a/a/", "a/a/a/a/a", "a/a/a/a/a/", "a/a/b", "a/a/b/", "a/b", "a/b/", "a/b/c/.d/e/", "a/c", "a/c/", "a/b", "a/x/", "b", "b/", "x/y", "x/y/", "z/z", "z/z/"}

			// test/globstars.js line 63
			assertMatchList(t, fixtures, "**/*/a/", []string{"a/a/", "a/a/a/", "a/a/a/a/", "a/a/a/a/a/"})
			// test/globstars.js line 64
			assertMatchList(t, fixtures, "**/*/a/*/", []string{"a/a/a/", "a/a/a/a/", "a/a/a/a/a/", "a/a/b/"})
			// test/globstars.js line 65
			assertMatchList(t, fixtures, "**/*/x/", []string{"a/x/"})
			// test/globstars.js line 66
			assertMatchList(t, fixtures, "**/*/*/*/*/", []string{"a/a/a/a/", "a/a/a/a/a/"})
			// test/globstars.js line 67
			assertMatchList(t, fixtures, "**/*/*/*/*/*/", []string{"a/a/a/a/a/"})
			// test/globstars.js line 68
			assertMatchList(t, fixtures, "*a/a/*/", []string{"a/a/a/", "a/a/b/"})
			// test/globstars.js line 69
			assertMatchList(t, fixtures, "**a/a/*/", []string{"a/a/a/", "a/a/b/"})
			// test/globstars.js line 70
			assertMatchList(t, fixtures, "**/a/*/*/", []string{"a/a/a/", "a/a/a/a/", "a/a/a/a/a/", "a/a/b/"})
			// test/globstars.js line 71
			assertMatchList(t, fixtures, "**/a/*/*/*/", []string{"a/a/a/a/", "a/a/a/a/a/"})
			// test/globstars.js line 72
			assertMatchList(t, fixtures, "**/a/*/*/*/*/", []string{"a/a/a/a/a/"})
			// test/globstars.js line 73
			assertMatchList(t, fixtures, "**/a/*/a/", []string{"a/a/a/", "a/a/a/a/", "a/a/a/a/a/"})
			// test/globstars.js line 74
			assertMatchList(t, fixtures, "**/a/*/b/", []string{"a/a/b/"})
		})

		t.Run("should match literal globstars when stars are escaped", func(t *testing.T) {
			fixtures := []string{".md", "**a.md", "**.md", ".md", "**"}
			// test/globstars.js line 79
			assertMatchList(t, fixtures, "\\*\\**.md", []string{"**a.md", "**.md"})
			// test/globstars.js line 80
			assertMatchList(t, fixtures, "\\*\\*.md", []string{"**.md"})
		})

		t.Run("single dots", func(t *testing.T) {
			// test/globstars.js line 84
			assertMatch(t, false, ".a/a", "**")
			// test/globstars.js line 85
			assertMatch(t, false, "a/.a", "**")
			// test/globstars.js line 86
			assertMatch(t, false, ".a/a", "**/")
			// test/globstars.js line 87
			assertMatch(t, false, "a/.a", "**/")
			// test/globstars.js line 88
			assertMatch(t, false, ".a/a", "**/**")
			// test/globstars.js line 89
			assertMatch(t, false, "a/.a", "**/**")
			// test/globstars.js line 90
			assertMatch(t, false, ".a/a", "**/**/*")
			// test/globstars.js line 91
			assertMatch(t, false, "a/.a", "**/**/*")
			// test/globstars.js line 92
			assertMatch(t, false, ".a/a", "**/**/x")
			// test/globstars.js line 93
			assertMatch(t, false, "a/.a", "**/**/x")
			// test/globstars.js line 94
			assertMatch(t, false, ".a/a", "**/x")
			// test/globstars.js line 95
			assertMatch(t, false, "a/.a", "**/x")
			// test/globstars.js line 96
			assertMatch(t, false, ".a/a", "**/x/*")
			// test/globstars.js line 97
			assertMatch(t, false, "a/.a", "**/x/*")
			// test/globstars.js line 98
			assertMatch(t, false, ".a/a", "**/x/**")
			// test/globstars.js line 99
			assertMatch(t, false, "a/.a", "**/x/**")
			// test/globstars.js line 100
			assertMatch(t, false, ".a/a", "**/x/*/*")
			// test/globstars.js line 101
			assertMatch(t, false, "a/.a", "**/x/*/*")
			// test/globstars.js line 102
			assertMatch(t, false, ".a/a", "*/x/**")
			// test/globstars.js line 103
			assertMatch(t, false, "a/.a", "*/x/**")
			// test/globstars.js line 104
			assertMatch(t, false, ".a/a", "a/**")
			// test/globstars.js line 105
			assertMatch(t, false, "a/.a", "a/**")
			// test/globstars.js line 106
			assertMatch(t, false, ".a/a", "a/**/*")
			// test/globstars.js line 107
			assertMatch(t, false, "a/.a", "a/**/*")
			// test/globstars.js line 108
			assertMatch(t, false, ".a/a", "a/**/**/*")
			// test/globstars.js line 109
			assertMatch(t, false, "a/.a", "a/**/**/*")
			// test/globstars.js line 110
			assertMatch(t, false, ".a/a", "b/**")
			// test/globstars.js line 111
			assertMatch(t, false, "a/.a", "b/**")
		})

		t.Run("double dots", func(t *testing.T) {
			// test/globstars.js line 115
			assertMatch(t, false, "a/../a", "**")
			// test/globstars.js line 116
			assertMatch(t, false, "ab/../ac", "**")
			// test/globstars.js line 117
			assertMatch(t, false, "../a", "**")
			// test/globstars.js line 118
			assertMatch(t, false, "../../b", "**")
			// test/globstars.js line 119
			assertMatch(t, false, "../c", "**")
			// test/globstars.js line 120
			assertMatch(t, false, "../c/d", "**")
			// test/globstars.js line 121
			assertMatch(t, false, "a/../a", "**/")
			// test/globstars.js line 122
			assertMatch(t, false, "ab/../ac", "**/")
			// test/globstars.js line 123
			assertMatch(t, false, "../a", "**/")
			// test/globstars.js line 124
			assertMatch(t, false, "../../b", "**/")
			// test/globstars.js line 125
			assertMatch(t, false, "../c", "**/")
			// test/globstars.js line 126
			assertMatch(t, false, "../c/d", "**/")
			// test/globstars.js line 127
			assertMatch(t, false, "a/../a", "**/**")
			// test/globstars.js line 128
			assertMatch(t, false, "ab/../ac", "**/**")
			// test/globstars.js line 129
			assertMatch(t, false, "../a", "**/**")
			// test/globstars.js line 130
			assertMatch(t, false, "../../b", "**/**")
			// test/globstars.js line 131
			assertMatch(t, false, "../c", "**/**")
			// test/globstars.js line 132
			assertMatch(t, false, "../c/d", "**/**")
			// test/globstars.js line 133
			assertMatch(t, false, "a/../a", "**/**/*")
			// test/globstars.js line 134
			assertMatch(t, false, "ab/../ac", "**/**/*")
			// test/globstars.js line 135
			assertMatch(t, false, "../a", "**/**/*")
			// test/globstars.js line 136
			assertMatch(t, false, "../../b", "**/**/*")
			// test/globstars.js line 137
			assertMatch(t, false, "../c", "**/**/*")
			// test/globstars.js line 138
			assertMatch(t, false, "../c/d", "**/**/*")
			// test/globstars.js line 139
			assertMatch(t, false, "a/../a", "**/**/x")
			// test/globstars.js line 140
			assertMatch(t, false, "ab/../ac", "**/**/x")
			// test/globstars.js line 141
			assertMatch(t, false, "../a", "**/**/x")
			// test/globstars.js line 142
			assertMatch(t, false, "../../b", "**/**/x")
			// test/globstars.js line 143
			assertMatch(t, false, "../c", "**/**/x")
			// test/globstars.js line 144
			assertMatch(t, false, "../c/d", "**/**/x")
			// test/globstars.js line 145
			assertMatch(t, false, "a/../a", "**/x")
			// test/globstars.js line 146
			assertMatch(t, false, "ab/../ac", "**/x")
			// test/globstars.js line 147
			assertMatch(t, false, "../a", "**/x")
			// test/globstars.js line 148
			assertMatch(t, false, "../../b", "**/x")
			// test/globstars.js line 149
			assertMatch(t, false, "../c", "**/x")
			// test/globstars.js line 150
			assertMatch(t, false, "../c/d", "**/x")
			// test/globstars.js line 151
			assertMatch(t, false, "a/../a", "**/x/*")
			// test/globstars.js line 152
			assertMatch(t, false, "ab/../ac", "**/x/*")
			// test/globstars.js line 153
			assertMatch(t, false, "../a", "**/x/*")
			// test/globstars.js line 154
			assertMatch(t, false, "../../b", "**/x/*")
			// test/globstars.js line 155
			assertMatch(t, false, "../c", "**/x/*")
			// test/globstars.js line 156
			assertMatch(t, false, "../c/d", "**/x/*")
			// test/globstars.js line 157
			assertMatch(t, false, "a/../a", "**/x/**")
			// test/globstars.js line 158
			assertMatch(t, false, "ab/../ac", "**/x/**")
			// test/globstars.js line 159
			assertMatch(t, false, "../a", "**/x/**")
			// test/globstars.js line 160
			assertMatch(t, false, "../../b", "**/x/**")
			// test/globstars.js line 161
			assertMatch(t, false, "../c", "**/x/**")
			// test/globstars.js line 162
			assertMatch(t, false, "../c/d", "**/x/**")
			// test/globstars.js line 163
			assertMatch(t, false, "a/../a", "**/x/*/*")
			// test/globstars.js line 164
			assertMatch(t, false, "ab/../ac", "**/x/*/*")
			// test/globstars.js line 165
			assertMatch(t, false, "../a", "**/x/*/*")
			// test/globstars.js line 166
			assertMatch(t, false, "../../b", "**/x/*/*")
			// test/globstars.js line 167
			assertMatch(t, false, "../c", "**/x/*/*")
			// test/globstars.js line 168
			assertMatch(t, false, "../c/d", "**/x/*/*")
			// test/globstars.js line 169
			assertMatch(t, false, "a/../a", "*/x/**")
			// test/globstars.js line 170
			assertMatch(t, false, "ab/../ac", "*/x/**")
			// test/globstars.js line 171
			assertMatch(t, false, "../a", "*/x/**")
			// test/globstars.js line 172
			assertMatch(t, false, "../../b", "*/x/**")
			// test/globstars.js line 173
			assertMatch(t, false, "../c", "*/x/**")
			// test/globstars.js line 174
			assertMatch(t, false, "../c/d", "*/x/**")
			// test/globstars.js line 175
			assertMatch(t, false, "a/../a", "a/**")
			// test/globstars.js line 176
			assertMatch(t, false, "ab/../ac", "a/**")
			// test/globstars.js line 177
			assertMatch(t, false, "../a", "a/**")
			// test/globstars.js line 178
			assertMatch(t, false, "../../b", "a/**")
			// test/globstars.js line 179
			assertMatch(t, false, "../c", "a/**")
			// test/globstars.js line 180
			assertMatch(t, false, "../c/d", "a/**")
			// test/globstars.js line 181
			assertMatch(t, false, "a/../a", "a/**/*")
			// test/globstars.js line 182
			assertMatch(t, false, "ab/../ac", "a/**/*")
			// test/globstars.js line 183
			assertMatch(t, false, "../a", "a/**/*")
			// test/globstars.js line 184
			assertMatch(t, false, "../../b", "a/**/*")
			// test/globstars.js line 185
			assertMatch(t, false, "../c", "a/**/*")
			// test/globstars.js line 186
			assertMatch(t, false, "../c/d", "a/**/*")
			// test/globstars.js line 187
			assertMatch(t, false, "a/../a", "a/**/**/*")
			// test/globstars.js line 188
			assertMatch(t, false, "ab/../ac", "a/**/**/*")
			// test/globstars.js line 189
			assertMatch(t, false, "../a", "a/**/**/*")
			// test/globstars.js line 190
			assertMatch(t, false, "../../b", "a/**/**/*")
			// test/globstars.js line 191
			assertMatch(t, false, "../c", "a/**/**/*")
			// test/globstars.js line 192
			assertMatch(t, false, "../c/d", "a/**/**/*")
			// test/globstars.js line 193
			assertMatch(t, false, "a/../a", "b/**")
			// test/globstars.js line 194
			assertMatch(t, false, "ab/../ac", "b/**")
			// test/globstars.js line 195
			assertMatch(t, false, "../a", "b/**")
			// test/globstars.js line 196
			assertMatch(t, false, "../../b", "b/**")
			// test/globstars.js line 197
			assertMatch(t, false, "../c", "b/**")
			// test/globstars.js line 198
			assertMatch(t, false, "../c/d", "b/**")
		})

		t.Run("should match", func(t *testing.T) {
			// test/globstars.js line 202
			assertMatch(t, false, "a", "**/")
			// test/globstars.js line 203
			assertMatch(t, false, "a", "**/a/*")
			// test/globstars.js line 204
			assertMatch(t, false, "a", "**/a/*/*")
			// test/globstars.js line 205
			assertMatch(t, false, "a", "*/a/**")
			// test/globstars.js line 206
			assertMatch(t, false, "a", "a/**/*")
			// test/globstars.js line 207
			assertMatch(t, false, "a", "a/**/**/*")
			// test/globstars.js line 208
			assertMatch(t, false, "a/b", "**/")
			// test/globstars.js line 209
			assertMatch(t, false, "a/b", "**/b/*")
			// test/globstars.js line 210
			assertMatch(t, false, "a/b", "**/b/*/*")
			// test/globstars.js line 211
			assertMatch(t, false, "a/b", "b/**")
			// test/globstars.js line 212
			assertMatch(t, false, "a/b/c", "**/")
			// test/globstars.js line 213
			assertMatch(t, false, "a/b/c", "**/**/b")
			// test/globstars.js line 214
			assertMatch(t, false, "a/b/c", "**/b")
			// test/globstars.js line 215
			assertMatch(t, false, "a/b/c", "**/b/*/*")
			// test/globstars.js line 216
			assertMatch(t, false, "a/b/c", "b/**")
			// test/globstars.js line 217
			assertMatch(t, false, "a/b/c/d", "**/")
			// test/globstars.js line 218
			assertMatch(t, false, "a/b/c/d", "**/d/*")
			// test/globstars.js line 219
			assertMatch(t, false, "a/b/c/d", "b/**")
			// test/globstars.js line 220
			assertMatch(t, true, "a", "**")
			// test/globstars.js line 221
			assertMatch(t, true, "a", "**/**")
			// test/globstars.js line 222
			assertMatch(t, true, "a", "**/**/*")
			// test/globstars.js line 223
			assertMatch(t, true, "a", "**/**/a")
			// test/globstars.js line 224
			assertMatch(t, true, "a", "**/a")
			// test/globstars.js line 225
			assertMatch(t, true, "a", "**/a/**")
			// test/globstars.js line 226
			assertMatch(t, true, "a", "a/**")
			// test/globstars.js line 227
			assertMatch(t, true, "a/b", "**")
			// test/globstars.js line 228
			assertMatch(t, true, "a/b", "**/**")
			// test/globstars.js line 229
			assertMatch(t, true, "a/b", "**/**/*")
			// test/globstars.js line 230
			assertMatch(t, true, "a/b", "**/**/b")
			// test/globstars.js line 231
			assertMatch(t, true, "a/b", "**/b")
			// test/globstars.js line 232
			assertMatch(t, true, "a/b", "**/b/**")
			// test/globstars.js line 233
			assertMatch(t, true, "a/b", "*/b/**")
			// test/globstars.js line 234
			assertMatch(t, true, "a/b", "a/**")
			// test/globstars.js line 235
			assertMatch(t, true, "a/b", "a/**/*")
			// test/globstars.js line 236
			assertMatch(t, true, "a/b", "a/**/**/*")
			// test/globstars.js line 237
			assertMatch(t, true, "a/b/c", "**")
			// test/globstars.js line 238
			assertMatch(t, true, "a/b/c", "**/**")
			// test/globstars.js line 239
			assertMatch(t, true, "a/b/c", "**/**/*")
			// test/globstars.js line 240
			assertMatch(t, true, "a/b/c", "**/b/*")
			// test/globstars.js line 241
			assertMatch(t, true, "a/b/c", "**/b/**")
			// test/globstars.js line 242
			assertMatch(t, true, "a/b/c", "*/b/**")
			// test/globstars.js line 243
			assertMatch(t, true, "a/b/c", "a/**")
			// test/globstars.js line 244
			assertMatch(t, true, "a/b/c", "a/**/*")
			// test/globstars.js line 245
			assertMatch(t, true, "a/b/c", "a/**/**/*")
			// test/globstars.js line 246
			assertMatch(t, true, "a/b/c/d", "**")
			// test/globstars.js line 247
			assertMatch(t, true, "a/b/c/d", "**/**")
			// test/globstars.js line 248
			assertMatch(t, true, "a/b/c/d", "**/**/*")
			// test/globstars.js line 249
			assertMatch(t, true, "a/b/c/d", "**/**/d")
			// test/globstars.js line 250
			assertMatch(t, true, "a/b/c/d", "**/b/**")
			// test/globstars.js line 251
			assertMatch(t, true, "a/b/c/d", "**/b/*/*")
			// test/globstars.js line 252
			assertMatch(t, true, "a/b/c/d", "**/d")
			// test/globstars.js line 253
			assertMatch(t, true, "a/b/c/d", "*/b/**")
			// test/globstars.js line 254
			assertMatch(t, true, "a/b/c/d", "a/**")
			// test/globstars.js line 255
			assertMatch(t, true, "a/b/c/d", "a/**/*")
			// test/globstars.js line 256
			assertMatch(t, true, "a/b/c/d", "a/**/**/*")
		})

		t.Run("should match nested directories", func(t *testing.T) {
			// test/globstars.js line 260
			assertMatch(t, true, "a/b", "*/*")
			// test/globstars.js line 261
			assertMatch(t, true, "a/b/c/xyz.md", "a/b/c/*.md")
			// test/globstars.js line 262
			assertMatch(t, true, "a/bb.bb/c/xyz.md", "a/*/c/*.md")
			// test/globstars.js line 263
			assertMatch(t, true, "a/bb/c/xyz.md", "a/*/c/*.md")
			// test/globstars.js line 264
			assertMatch(t, true, "a/bbbb/c/xyz.md", "a/*/c/*.md")

			// test/globstars.js line 266
			assertMatch(t, true, "a/b/c", "**/*")
			// test/globstars.js line 267
			assertMatch(t, true, "a/b/c", "**/**")
			// test/globstars.js line 268
			assertMatch(t, true, "a/b/c", "*/**")
			// test/globstars.js line 269
			assertMatch(t, true, "a/b/c/d/e/j/n/p/o/z/c.md", "a/**/j/**/z/*.md")
			// test/globstars.js line 270
			assertMatch(t, true, "a/b/c/d/e/z/c.md", "a/**/z/*.md")
			// test/globstars.js line 271
			assertMatch(t, true, "a/bb.bb/aa/b.b/aa/c/xyz.md", "a/**/c/*.md")
			// test/globstars.js line 272
			assertMatch(t, true, "a/bb.bb/aa/bb/aa/c/xyz.md", "a/**/c/*.md")
			// test/globstars.js line 273
			assertMatch(t, false, "a/b/c/j/e/z/c.txt", "a/**/j/**/z/*.md")
			// test/globstars.js line 274
			assertMatch(t, false, "a/b/c/xyz.md", "a/b/**/c{d,e}/**/xyz.md")
			// test/globstars.js line 275
			assertMatch(t, false, "a/b/d/xyz.md", "a/b/**/c{d,e}/**/xyz.md")
			// test/globstars.js line 276
			assertMatch(t, false, "a/b", "a/**/")
			// test/globstars.js line 277
			assertMatch(t, false, "a/b/.js/c.txt", "**/*")
			// test/globstars.js line 278
			assertMatch(t, false, "a/b/c/d", "a/**/")
			// test/globstars.js line 279
			assertMatch(t, false, "a/bb", "a/**/")
			// test/globstars.js line 280
			assertMatch(t, false, "a/cb", "a/**/")
			// test/globstars.js line 281
			assertMatch(t, true, "/a/b", "/**")
			// test/globstars.js line 282
			assertMatch(t, true, "a.b", "**/*")
			// test/globstars.js line 283
			assertMatch(t, true, "a.js", "**/*")
			// test/globstars.js line 284
			assertMatch(t, true, "a.js", "**/*.js")
			// test/globstars.js line 285
			assertMatch(t, true, "a/", "a/**/")
			// test/globstars.js line 286
			assertMatch(t, true, "a/a.js", "**/*.js")
			// test/globstars.js line 287
			assertMatch(t, true, "a/a/b.js", "**/*.js")
			// test/globstars.js line 288
			assertMatch(t, true, "a/b", "a/**/b")
			// test/globstars.js line 289
			assertMatch(t, true, "a/b", "a/**b")
			// test/globstars.js line 290
			assertMatch(t, true, "a/b.md", "**/*.md")
			// test/globstars.js line 291
			assertMatch(t, true, "a/b/c.js", "**/*")
			// test/globstars.js line 292
			assertMatch(t, true, "a/b/c.txt", "**/*")
			// test/globstars.js line 293
			assertMatch(t, true, "a/b/c/d/", "a/**/")
			// test/globstars.js line 294
			assertMatch(t, true, "a/b/c/d/a.js", "**/*")
			// test/globstars.js line 295
			assertMatch(t, true, "a/b/c/z.js", "a/b/**/*.js")
			// test/globstars.js line 296
			assertMatch(t, true, "a/b/z.js", "a/b/**/*.js")
			// test/globstars.js line 297
			assertMatch(t, true, "ab", "**/*")
			// test/globstars.js line 298
			assertMatch(t, true, "ab/c", "**/*")
			// test/globstars.js line 299
			assertMatch(t, true, "ab/c/d", "**/*")
			// test/globstars.js line 300
			assertMatch(t, true, "abc.js", "**/*")
		})

		t.Run("should not match dotfiles by default", func(t *testing.T) {
			// test/globstars.js line 304
			assertMatch(t, false, "a/.b", "a/**/z/*.md")
			// test/globstars.js line 305
			assertMatch(t, false, "a/b/z/.a", "a/**/z/*.a")
			// test/globstars.js line 306
			assertMatch(t, false, "a/b/z/.a", "a/*/z/*.a")
			// test/globstars.js line 307
			assertMatch(t, false, "a/b/z/.a", "b/a")
			// test/globstars.js line 308
			assertMatch(t, false, "a/foo/z/.b", "a/**/z/*.md")
		})

		t.Run("should match leading dots when defined in pattern", func(t *testing.T) {
			// test/globstars.js line 312 (fixtures declared but only used implicitly)
			// test/globstars.js line 313
			assertMatch(t, false, ".gitignore", "a/**/z/*.md")
			// test/globstars.js line 314
			assertMatch(t, false, "a/b/z/.dotfile", "a/**/z/*.md")
			// test/globstars.js line 315
			assertMatch(t, false, "a/b/z/.dotfile.md", "**/c/.*.md")
			// test/globstars.js line 316
			assertMatch(t, true, "a/.b", "a/.*")
			// test/globstars.js line 317
			assertMatch(t, true, "a/b/z/.a", "a/*/z/.a")
			// test/globstars.js line 318
			assertMatch(t, true, "a/b/z/.dotfile.md", "**/.*.md")
			// test/globstars.js line 319
			assertMatch(t, true, "a/b/z/.dotfile.md", "a/**/z/.*.md")
			// test/globstars.js line 320
			assertMatchList(t, []string{".md", "a.md", "a/b/c.md", ".txt"}, "**/*.md", []string{"a.md", "a/b/c.md"})
			// test/globstars.js line 321
			assertMatchList(t, []string{".md/.md", ".md", "a/.md", "a/b/.md"}, "**/.md", []string{".md", "a/.md", "a/b/.md"})
			// test/globstars.js line 322
			assertMatchList(t, []string{".md/.md", ".md/foo/.md", ".md", "a/.md", "a/b/.md"}, ".md/**/.md", []string{".md/.md", ".md/foo/.md"})
			// test/globstars.js line 323
			assertMatchList(t, []string{".gitignore", "a/b/z/.dotfile", "a/b/z/.dotfile.md", "a/b/z/.dotfile.md", "a/b/z/.dotfile.md"}, "a/**/z/.*.md", []string{"a/b/z/.dotfile.md"})
		})

		t.Run("todo... (micromatch/#24)", func(t *testing.T) {
			// test/globstars.js line 327
			assertMatch(t, true, "foo/bar/baz/one/image.png", "foo/bar/**/one/**/*.*")
			// test/globstars.js line 328
			assertMatch(t, true, "foo/bar/baz/one/two/image.png", "foo/bar/**/one/**/*.*")
			// test/globstars.js line 329
			assertMatch(t, true, "foo/bar/baz/one/two/three/image.png", "foo/bar/**/one/**/*.*")
			// test/globstars.js line 330
			assertMatch(t, false, "a/b/c/d/", "a/b/**/f")
			// test/globstars.js line 331
			assertMatch(t, true, "a", "a/**")
			// test/globstars.js line 332
			assertMatch(t, true, "a", "**")
			// test/globstars.js line 333
			assertMatch(t, true, "a", "a{,/**}")
			// test/globstars.js line 334
			assertMatch(t, true, "a/", "**")
			// test/globstars.js line 335
			assertMatch(t, true, "a/", "a/**")
			// test/globstars.js line 336
			assertMatch(t, true, "a/b/c/d", "**")
			// test/globstars.js line 337
			assertMatch(t, true, "a/b/c/d/", "**")
			// test/globstars.js line 338
			assertMatch(t, true, "a/b/c/d/", "**/**")
			// test/globstars.js line 339
			assertMatch(t, true, "a/b/c/d/", "**/b/**")
			// test/globstars.js line 340
			assertMatch(t, true, "a/b/c/d/", "a/b/**")
			// test/globstars.js line 341
			assertMatch(t, true, "a/b/c/d/", "a/b/**/")
			// test/globstars.js line 342
			assertMatch(t, true, "a/b/c/d/", "a/b/**/c/**/")
			// test/globstars.js line 343
			assertMatch(t, true, "a/b/c/d/", "a/b/**/c/**/d/")
			// test/globstars.js line 344
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/**/*.*")
			// test/globstars.js line 345
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/*.*")
			// test/globstars.js line 346
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/c/**/d/*.*")
			// test/globstars.js line 347
			assertMatch(t, true, "a/b/c/d/e.f", "a/b/**/d/**/*.*")
			// test/globstars.js line 348
			assertMatch(t, true, "a/b/c/d/g/e.f", "a/b/**/d/**/*.*")
			// test/globstars.js line 349
			assertMatch(t, true, "a/b/c/d/g/g/e.f", "a/b/**/d/**/*.*")

			// test/globstars.js line 351
			assertMatch(t, true, "a/b-c/z.js", "a/b-*/**/z.js")
			// test/globstars.js line 352
			assertMatch(t, true, "a/b-c/d/e/z.js", "a/b-*/**/z.js")
		})
	})

	t.Run("globstars", func(t *testing.T) {
		t.Run("should match globstars", func(t *testing.T) {
			// test/globstars.js line 358
			assertMatch(t, true, "a/b/c/d.js", "**/*.js")
			// test/globstars.js line 359
			assertMatch(t, true, "a/b/c.js", "**/*.js")
			// test/globstars.js line 360
			assertMatch(t, true, "a/b.js", "**/*.js")
			// test/globstars.js line 361
			assertMatch(t, true, "a/b/c/d/e/f.js", "a/b/**/*.js")
			// test/globstars.js line 362
			assertMatch(t, true, "a/b/c/d/e.js", "a/b/**/*.js")
			// test/globstars.js line 363
			assertMatch(t, true, "a/b/c/d.js", "a/b/c/**/*.js")
			// test/globstars.js line 364
			assertMatch(t, true, "a/b/c/d.js", "a/b/**/*.js")
			// test/globstars.js line 365
			assertMatch(t, true, "a/b/d.js", "a/b/**/*.js")

			// test/globstars.js line 367
			assertMatch(t, false, "a/d.js", "a/b/**/*.js")
			// test/globstars.js line 368
			assertMatch(t, false, "d.js", "a/b/**/*.js")
		})

		t.Run("should regard non-exclusive double-stars as single stars", func(t *testing.T) {
			// test/globstars.js line 372
			assertMatch(t, false, "a/b/c", "**c")
			// test/globstars.js line 373
			assertMatch(t, false, "a/b/c", "a/**c")
			// test/globstars.js line 374
			assertMatch(t, false, "a/b/c", "a/**z")
			// test/globstars.js line 375
			assertMatch(t, false, "a/b/c/b/c", "a/**b**/c")
			// test/globstars.js line 376
			assertMatch(t, false, "a/b/c/d/e.js", "a/b/c**/*.js")
			// test/globstars.js line 377
			assertMatch(t, true, "a/b/c/b/c", "a/**/b/**/c")
			// test/globstars.js line 378
			assertMatch(t, true, "a/aba/c", "a/**b**/c")
			// test/globstars.js line 379
			assertMatch(t, true, "a/b/c", "a/**b**/c")
			// test/globstars.js line 380
			assertMatch(t, true, "a/b/c/d.js", "a/b/c**/*.js")
		})

		t.Run("should support globstars (**)", func(t *testing.T) {
			// test/globstars.js line 384
			assertMatch(t, false, "a", "a/**/*")
			// test/globstars.js line 385
			assertMatch(t, false, "a", "a/**/**/*")
			// test/globstars.js line 386
			assertMatch(t, false, "a", "a/**/**/**/*")
			// test/globstars.js line 387
			assertMatch(t, false, "a/", "**/a")
			// test/globstars.js line 388
			assertMatch(t, false, "a/", "a/**/*")
			// test/globstars.js line 389
			assertMatch(t, false, "a/", "a/**/**/*")
			// test/globstars.js line 390
			assertMatch(t, false, "a/", "a/**/**/**/*")
			// test/globstars.js line 391
			assertMatch(t, false, "a/b", "**/a")
			// test/globstars.js line 392
			assertMatch(t, false, "a/b/c/j/e/z/c.txt", "a/**/j/**/z/*.md")
			// test/globstars.js line 393
			assertMatch(t, false, "a/bb", "a/**/b")
			// test/globstars.js line 394
			assertMatch(t, false, "a/c", "**/a")
			// test/globstars.js line 395
			assertMatch(t, false, "a/b", "**/a")
			// test/globstars.js line 396
			assertMatch(t, false, "a/x/y", "**/a")
			// test/globstars.js line 397
			assertMatch(t, false, "a/b/c/d", "**/a")
			// test/globstars.js line 398
			assertMatch(t, true, "a", "**")
			// test/globstars.js line 399
			assertMatch(t, true, "a", "**/a")
			// test/globstars.js line 400
			assertMatch(t, true, "a", "a/**")
			// test/globstars.js line 401
			assertMatch(t, true, "a/", "**")
			// test/globstars.js line 402
			assertMatch(t, true, "a/", "**/a/**")
			// test/globstars.js line 403
			assertMatch(t, true, "a/", "a/**")
			// test/globstars.js line 404
			assertMatch(t, true, "a/", "a/**/**")
			// test/globstars.js line 405
			assertMatch(t, true, "a/a", "**/a")
			// test/globstars.js line 406
			assertMatch(t, true, "a/b", "**")
			// test/globstars.js line 407
			assertMatch(t, true, "a/b", "*/*")
			// test/globstars.js line 408
			assertMatch(t, true, "a/b", "a/**")
			// test/globstars.js line 409
			assertMatch(t, true, "a/b", "a/**/*")
			// test/globstars.js line 410
			assertMatch(t, true, "a/b", "a/**/**/*")
			// test/globstars.js line 411
			assertMatch(t, true, "a/b", "a/**/**/**/*")
			// test/globstars.js line 412
			assertMatch(t, true, "a/b", "a/**/b")
			// test/globstars.js line 413
			assertMatch(t, true, "a/b/c", "**")
			// test/globstars.js line 414
			assertMatch(t, true, "a/b/c", "**/*")
			// test/globstars.js line 415
			assertMatch(t, true, "a/b/c", "**/**")
			// test/globstars.js line 416
			assertMatch(t, true, "a/b/c", "*/**")
			// test/globstars.js line 417
			assertMatch(t, true, "a/b/c", "a/**")
			// test/globstars.js line 418
			assertMatch(t, true, "a/b/c", "a/**/*")
			// test/globstars.js line 419
			assertMatch(t, true, "a/b/c", "a/**/**/*")
			// test/globstars.js line 420
			assertMatch(t, true, "a/b/c", "a/**/**/**/*")
			// test/globstars.js line 421
			assertMatch(t, true, "a/b/c/d", "**")
			// test/globstars.js line 422
			assertMatch(t, true, "a/b/c/d", "a/**")
			// test/globstars.js line 423
			assertMatch(t, true, "a/b/c/d", "a/**/*")
			// test/globstars.js line 424
			assertMatch(t, true, "a/b/c/d", "a/**/**/*")
			// test/globstars.js line 425
			assertMatch(t, true, "a/b/c/d", "a/**/**/**/*")
			// test/globstars.js line 426
			assertMatch(t, true, "a/b/c/d.e", "a/b/**/c/**/*.*")
			// test/globstars.js line 427
			assertMatch(t, true, "a/b/c/d/e/f/g.md", "a/**/f/*.md")
			// test/globstars.js line 428
			assertMatch(t, true, "a/b/c/d/e/f/g/h/i/j/k/l.md", "a/**/f/**/k/*.md")
			// test/globstars.js line 429
			assertMatch(t, true, "a/b/c/def.md", "a/b/c/*.md")
			// test/globstars.js line 430
			assertMatch(t, true, "a/bb.bb/c/ddd.md", "a/*/c/*.md")
			// test/globstars.js line 431
			assertMatch(t, true, "a/bb.bb/cc/d.d/ee/f/ggg.md", "a/**/f/*.md")
			// test/globstars.js line 432
			assertMatch(t, true, "a/bb.bb/cc/dd/ee/f/ggg.md", "a/**/f/*.md")
			// test/globstars.js line 433
			assertMatch(t, true, "a/bb/c/ddd.md", "a/*/c/*.md")
			// test/globstars.js line 434
			assertMatch(t, true, "a/bbbb/c/ddd.md", "a/*/c/*.md")
		})
	})
}
