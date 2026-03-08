// dotfiles_test.go — Faithful 1:1 port of picomatch/test/dotfiles.js
package picomatch

import "testing"

func TestDotfiles(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		t.Run("should not match dotfiles by default", func(t *testing.T) {
			// dotfiles.js line 10
			assertMatchList(t, []string{".dotfile"}, "*", []string{})
			// dotfiles.js line 11
			assertMatchList(t, []string{".dotfile"}, "**", []string{})
			// dotfiles.js line 12
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "*.md", []string{})
			// dotfiles.js line 13
			assertMatchList(t, []string{"a/b", "a/.b", ".a/b", ".a/.b"}, "**", []string{"a/b"})
			// dotfiles.js line 14
			assertMatchList(t, []string{"a/b/c/.dotfile"}, "*.*", []string{})
		})
	})

	t.Run("leading dot", func(t *testing.T) {
		t.Run("should match dotfiles when a leading dot is defined in the path", func(t *testing.T) {
			// dotfiles.js line 20
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "**/.*", []string{"a/b/c/.dotfile.md"})
			// dotfiles.js line 21
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "**/.*.md", []string{"a/b/c/.dotfile.md"})
		})

		t.Run("should use negation patterns on dotfiles", func(t *testing.T) {
			// dotfiles.js line 25
			assertMatchList(t, []string{".a", ".b", "c", "c.md"}, "!.*", []string{"c", "c.md"})
			// dotfiles.js line 26
			assertMatchList(t, []string{".a", ".b", "c", "c.md"}, "!.b", []string{".a", "c", "c.md"})
		})

		t.Run("should match dotfiles when there is a leading dot", func(t *testing.T) {
			opts := &Options{Dot: true}
			// dotfiles.js line 31
			assertMatchList(t, []string{".dotfile"}, "*", []string{".dotfile"}, opts)
			// dotfiles.js line 32
			assertMatchList(t, []string{".dotfile"}, "**", []string{".dotfile"}, opts)
			// dotfiles.js line 33
			assertMatchList(t, []string{"a/b", "a/.b", ".a/b", ".a/.b"}, "**", []string{"a/b", "a/.b", ".a/b", ".a/.b"}, opts)
			// dotfiles.js line 34
			assertMatchList(t, []string{"a/b", "a/.b", "a/.b", ".a/.b"}, "a/{.*,**}", []string{"a/b", "a/.b"}, opts)
			// dotfiles.js line 35
			assertMatchList(t, []string{"a/b", "a/.b", "a/.b", ".a/.b"}, "{.*,**}", []string{"a/b"}, &Options{})
			// dotfiles.js line 36
			assertMatchList(t, []string{"a/b", "a/.b", "a/.b", ".a/.b"}, "{.*,**}", []string{"a/b", "a/.b", ".a/.b"}, opts)
			// dotfiles.js line 37
			assertMatchList(t, []string{".dotfile"}, ".dotfile", []string{".dotfile"}, opts)
			// dotfiles.js line 38
			assertMatchList(t, []string{".dotfile.md"}, ".*.md", []string{".dotfile.md"}, opts)
		})

		t.Run("should match dotfiles when there is not a leading dot", func(t *testing.T) {
			opts := &Options{Dot: true}
			// dotfiles.js line 43
			assertMatchList(t, []string{".dotfile"}, "*.*", []string{".dotfile"}, opts)
			// dotfiles.js line 44
			assertMatchList(t, []string{".a", ".b", "c", "c.md"}, "*.*", []string{".a", ".b", "c.md"}, opts)
			// dotfiles.js line 45
			assertMatchList(t, []string{".dotfile"}, "*.md", []string{}, opts)
			// dotfiles.js line 46
			assertMatchList(t, []string{".verb.txt"}, "*.md", []string{}, opts)
			// dotfiles.js line 47
			assertMatchList(t, []string{"a/b/c/.dotfile"}, "*.md", []string{}, opts)
			// dotfiles.js line 48
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "*.md", []string{}, opts)
			// dotfiles.js line 49
			assertMatchList(t, []string{"a/b/c/.verb.md"}, "**/*.md", []string{"a/b/c/.verb.md"}, opts)
			// dotfiles.js line 50
			assertMatchList(t, []string{"foo.md"}, "*.md", []string{"foo.md"}, opts)
		})

		t.Run("should use negation patterns on dotfiles (with dot option)", func(t *testing.T) {
			// dotfiles.js line 54
			assertMatchList(t, []string{".a", ".b", "c", "c.md"}, "!.*", []string{"c", "c.md"})
			// dotfiles.js line 55
			assertMatchList(t, []string{".a", ".b", "c", "c.md"}, "!(.*)", []string{"c", "c.md"})
			// dotfiles.js line 56
			assertMatchList(t, []string{".a", ".b", "c", "c.md"}, "!(.*)*", []string{"c", "c.md"})
			// dotfiles.js line 57
			assertMatchList(t, []string{".a", ".b", "c", "c.md"}, "!*.*", []string{".a", ".b", "c"})
		})
	})

	t.Run("options.dot", func(t *testing.T) {
		t.Run("should match dotfiles when options.dot is true", func(t *testing.T) {
			fixtures := []string{"a/./b", "a/../b", "a/c/b", "a/.d/b"}
			dotTrue := &Options{Dot: true}
			dotFalse := &Options{Dot: false}
			// dotfiles.js line 64
			assertMatchList(t, []string{".dotfile"}, "*.*", []string{".dotfile"}, dotTrue)
			// dotfiles.js line 65
			assertMatchList(t, []string{".dotfile"}, "*.md", []string{}, dotTrue)
			// dotfiles.js line 66
			assertMatchList(t, []string{".dotfile"}, ".dotfile", []string{".dotfile"}, dotTrue)
			// dotfiles.js line 67
			assertMatchList(t, []string{".dotfile.md"}, ".*.md", []string{".dotfile.md"}, dotTrue)
			// dotfiles.js line 68
			assertMatchList(t, []string{".verb.txt"}, "*.md", []string{}, dotTrue)
			// dotfiles.js line 69
			assertMatchList(t, []string{".verb.txt"}, "*.md", []string{}, dotTrue)
			// dotfiles.js line 70
			assertMatchList(t, []string{"a/b/c/.dotfile"}, "*.md", []string{}, dotTrue)
			// dotfiles.js line 71
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "**/*.md", []string{"a/b/c/.dotfile.md"}, dotTrue)
			// dotfiles.js line 72
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "**/.*", []string{"a/b/c/.dotfile.md"}, dotFalse)
			// dotfiles.js line 73
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "**/.*.md", []string{"a/b/c/.dotfile.md"}, dotFalse)
			// dotfiles.js line 74
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "*.md", []string{}, dotFalse)
			// dotfiles.js line 75
			assertMatchList(t, []string{"a/b/c/.dotfile.md"}, "*.md", []string{}, dotTrue)
			// dotfiles.js line 76
			assertMatchList(t, []string{"a/b/c/.verb.md"}, "**/*.md", []string{"a/b/c/.verb.md"}, dotTrue)
			// dotfiles.js line 77
			assertMatchList(t, []string{"d.md"}, "*.md", []string{"d.md"}, dotTrue)
			// dotfiles.js line 78
			assertMatchList(t, fixtures, "a/*/b", []string{"a/c/b", "a/.d/b"}, dotTrue)
			// dotfiles.js line 79
			assertMatchList(t, fixtures, "a/.*/b", []string{"a/.d/b"})
			// dotfiles.js line 80
			assertMatchList(t, fixtures, "a/.*/b", []string{"a/.d/b"}, dotTrue)
		})

		t.Run("should match dotfiles when options.dot is true (isMatch)", func(t *testing.T) {
			dotTrue := &Options{Dot: true}
			// dotfiles.js line 84
			assertMatch(t, true, ".dot", "**/*dot", dotTrue)
			// dotfiles.js line 85
			assertMatch(t, true, ".dot", "*dot", dotTrue)
			// dotfiles.js line 86
			assertMatch(t, true, ".dot", "?dot", dotTrue)
			// dotfiles.js line 87
			assertMatch(t, true, ".dotfile.js", ".*.js", dotTrue)
			// dotfiles.js line 88
			assertMatch(t, true, "/a/b/.dot", "/**/*dot", dotTrue)
			// dotfiles.js line 89
			assertMatch(t, true, "/a/b/.dot", "**/*dot", dotTrue)
			// dotfiles.js line 90
			assertMatch(t, true, "/a/b/.dot", "**/.[d]ot", dotTrue)
			// dotfiles.js line 91
			assertMatch(t, true, "/a/b/.dot", "**/?dot", dotTrue)
			// dotfiles.js line 92
			assertMatch(t, true, "/a/b/.dot", "/**/.[d]ot", dotTrue)
			// dotfiles.js line 93
			assertMatch(t, true, "/a/b/.dot", "/**/?dot", dotTrue)
			// dotfiles.js line 94
			assertMatch(t, true, "a/b/.dot", "**/*dot", dotTrue)
			// dotfiles.js line 95
			assertMatch(t, true, "a/b/.dot", "**/.[d]ot", dotTrue)
			// dotfiles.js line 96
			assertMatch(t, true, "a/b/.dot", "**/?dot", dotTrue)
		})

		t.Run("should not match dotfiles when options.dot is false", func(t *testing.T) {
			dotFalse := &Options{Dot: false}
			// dotfiles.js line 100
			assertMatch(t, false, "a/b/.dot", "**/*dot", dotFalse)
			// dotfiles.js line 101
			assertMatch(t, false, "a/b/.dot", "**/?dot", dotFalse)
		})

		t.Run("should not match dotfiles when .dot is not defined and a dot is not in the glob pattern", func(t *testing.T) {
			// dotfiles.js line 105
			assertMatch(t, false, "a/b/.dot", "**/*dot")
			// dotfiles.js line 106
			assertMatch(t, false, "a/b/.dot", "**/?dot")
		})
	})

	t.Run("valid dotfiles", func(t *testing.T) {
		t.Run("micromatch issue#63 (dots)", func(t *testing.T) {
			// dotfiles.js line 112
			assertMatch(t, false, "/aaa/.git/foo", "/aaa/**/*")
			// dotfiles.js line 113
			assertMatch(t, false, "/aaa/bbb/.git", "/aaa/bbb/*")
			// dotfiles.js line 114
			assertMatch(t, false, "/aaa/bbb/.git", "/aaa/bbb/**")
			// dotfiles.js line 115
			assertMatch(t, false, "/aaa/bbb/ccc/.git", "/aaa/bbb/**")
			// dotfiles.js line 116
			assertMatch(t, false, "aaa/bbb/.git", "aaa/bbb/**")
			// dotfiles.js line 117
			assertMatch(t, true, "/aaa/bbb/", "/aaa/bbb/**")
			// dotfiles.js line 118
			assertMatch(t, true, "/aaa/bbb/foo", "/aaa/bbb/**")

			dotTrue := &Options{Dot: true}
			// dotfiles.js line 120
			assertMatch(t, true, "/aaa/.git/foo", "/aaa/**/*", dotTrue)
			// dotfiles.js line 121
			assertMatch(t, true, "/aaa/bbb/.git", "/aaa/bbb/*", dotTrue)
			// dotfiles.js line 122
			assertMatch(t, true, "/aaa/bbb/.git", "/aaa/bbb/**", dotTrue)
			// dotfiles.js line 123
			assertMatch(t, true, "/aaa/bbb/ccc/.git", "/aaa/bbb/**", dotTrue)
			// dotfiles.js line 124
			assertMatch(t, true, "aaa/bbb/.git", "aaa/bbb/**", dotTrue)
		})

		t.Run("should not match dotfiles with single stars by default", func(t *testing.T) {
			// dotfiles.js line 128
			assertMatch(t, true, "foo", "*")
			// dotfiles.js line 129
			assertMatch(t, true, "foo/bar", "*/*")
			// dotfiles.js line 130
			assertMatch(t, false, ".foo", "*")
			// dotfiles.js line 131
			assertMatch(t, false, ".foo/bar", "*/*")
			// dotfiles.js line 132
			assertMatch(t, false, ".foo/.bar", "*/*")
			// dotfiles.js line 133
			assertMatch(t, false, "foo/.bar", "*/*")
			// dotfiles.js line 134
			assertMatch(t, false, "foo/.bar/baz", "*/*/*")
		})

		t.Run("should work with dots in the path", func(t *testing.T) {
			// dotfiles.js line 138
			assertMatch(t, true, "../test.js", "../*.js")
			// dotfiles.js line 139
			assertMatch(t, true, "../.test.js", "../*.js", &Options{Dot: true})
			// dotfiles.js line 140
			assertMatch(t, false, "../.test.js", "../*.js")
		})

		t.Run("should not match dotfiles with globstar by default", func(t *testing.T) {
			// dotfiles.js line 144
			assertMatch(t, false, ".foo", "**/**")
			// dotfiles.js line 145
			assertMatch(t, false, ".foo", "**")
			// dotfiles.js line 146
			assertMatch(t, false, ".foo", "**/*")
			// dotfiles.js line 147
			assertMatch(t, false, "bar/.foo", "**/*")
			// dotfiles.js line 148
			assertMatch(t, false, ".bar", "**/*")
			// dotfiles.js line 149
			assertMatch(t, false, "foo/.bar", "**/*")
			// dotfiles.js line 150
			assertMatch(t, false, "foo/.bar", "**/*a*")
		})

		t.Run("should match dotfiles when a leading dot is in the pattern", func(t *testing.T) {
			// dotfiles.js line 154
			assertMatch(t, false, "foo", "**/.*a*")
			// dotfiles.js line 155
			assertMatch(t, true, ".bar", "**/.*a*")
			// dotfiles.js line 156
			assertMatch(t, true, "foo/.bar", "**/.*a*")
			// dotfiles.js line 157
			assertMatch(t, true, ".foo", "**/.*")

			// dotfiles.js line 159
			assertMatch(t, false, "foo", ".*a*")
			// dotfiles.js line 160
			assertMatch(t, true, ".bar", ".*a*")
			// dotfiles.js line 161
			assertMatch(t, false, "bar", ".*a*")

			// dotfiles.js line 163
			assertMatch(t, false, "foo", ".b*")
			// dotfiles.js line 164
			assertMatch(t, true, ".bar", ".b*")
			// dotfiles.js line 165
			assertMatch(t, false, "bar", ".b*")

			// dotfiles.js line 167
			assertMatch(t, false, "foo", ".*r")
			// dotfiles.js line 168
			assertMatch(t, true, ".bar", ".*r")
			// dotfiles.js line 169
			assertMatch(t, false, "bar", ".*r")
		})

		t.Run("should not match a dot when the dot is not explicitly defined", func(t *testing.T) {
			// dotfiles.js line 173
			assertMatch(t, false, ".dot", "**/*dot")
			// dotfiles.js line 174
			assertMatch(t, false, ".dot", "**/?dot")
			// dotfiles.js line 175
			assertMatch(t, false, ".dot", "*/*dot")
			// dotfiles.js line 176
			assertMatch(t, false, ".dot", "*/?dot")
			// dotfiles.js line 177
			assertMatch(t, false, ".dot", "*dot")
			// dotfiles.js line 178
			assertMatch(t, false, ".dot", "/*dot")
			// dotfiles.js line 179
			assertMatch(t, false, ".dot", "/?dot")
			// dotfiles.js line 180
			assertMatch(t, false, "/.dot", "**/*dot")
			// dotfiles.js line 181
			assertMatch(t, false, "/.dot", "**/?dot")
			// dotfiles.js line 182
			assertMatch(t, false, "/.dot", "*/*dot")
			// dotfiles.js line 183
			assertMatch(t, false, "/.dot", "*/?dot")
			// dotfiles.js line 184
			assertMatch(t, false, "/.dot", "/*dot")
			// dotfiles.js line 185
			assertMatch(t, false, "/.dot", "/?dot")
			// dotfiles.js line 186
			assertMatch(t, false, "abc/.dot", "*/*dot")
			// dotfiles.js line 187
			assertMatch(t, false, "abc/.dot", "*/?dot")
			// dotfiles.js line 188
			assertMatch(t, false, "abc/.dot", "abc/*dot")
			// dotfiles.js line 189
			assertMatch(t, false, "abc/abc/.dot", "**/*dot")
			// dotfiles.js line 190
			assertMatch(t, false, "abc/abc/.dot", "**/?dot")
		})

		t.Run("should not match leading dots with question marks", func(t *testing.T) {
			// dotfiles.js line 194
			assertMatch(t, false, ".dot", "?dot")
			// dotfiles.js line 195
			assertMatch(t, false, "/.dot", "/?dot")
			// dotfiles.js line 196
			assertMatch(t, false, "abc/.dot", "abc/?dot")
		})

		t.Run("should match double dots when defined in pattern", func(t *testing.T) {
			// dotfiles.js line 200
			assertMatch(t, false, "../../b", "**/../*")
			// dotfiles.js line 201
			assertMatch(t, false, "../../b", "*/../*")
			// dotfiles.js line 202
			assertMatch(t, false, "../../b", "../*")
			// dotfiles.js line 203
			assertMatch(t, false, "../abc", "*/../*")
			// dotfiles.js line 204
			assertMatch(t, false, "../abc", "*/../*")
			// dotfiles.js line 205
			assertMatch(t, false, "../c/d", "**/../*")
			// dotfiles.js line 206
			assertMatch(t, false, "../c/d", "*/../*")
			// dotfiles.js line 207
			assertMatch(t, false, "../c/d", "../*")
			// dotfiles.js line 208
			assertMatch(t, false, "abc", "**/../*")
			// dotfiles.js line 209
			assertMatch(t, false, "abc", "*/../*")
			// dotfiles.js line 210
			assertMatch(t, false, "abc", "../*")
			// dotfiles.js line 211
			assertMatch(t, false, "abc/../abc", "../*")
			// dotfiles.js line 212
			assertMatch(t, false, "abc/../abc", "../*")
			// dotfiles.js line 213
			assertMatch(t, false, "abc/../", "**/../*")

			// dotfiles.js line 215
			assertMatch(t, true, "..", "..")
			// dotfiles.js line 216
			assertMatch(t, true, "../b", "../*")
			// dotfiles.js line 217
			assertMatch(t, true, "../../b", "../../*")
			// dotfiles.js line 218
			assertMatch(t, true, "../../..", "../../..")
			// dotfiles.js line 219
			assertMatch(t, true, "../abc", "**/../*")
			// dotfiles.js line 220
			assertMatch(t, true, "../abc", "../*")
			// dotfiles.js line 221
			assertMatch(t, true, "abc/../abc", "**/../*")
			// dotfiles.js line 222
			assertMatch(t, true, "abc/../abc", "*/../*")
			// dotfiles.js line 223
			assertMatch(t, true, "abc/../abc", "**/../*")
			// dotfiles.js line 224
			assertMatch(t, true, "abc/../abc", "*/../*")
		})

		t.Run("should not match double dots when not defined in pattern", func(t *testing.T) {
			// dotfiles.js line 228
			assertMatch(t, false, "../abc", "**/*")
			// dotfiles.js line 229
			assertMatch(t, false, "../abc", "**/**/**")
			// dotfiles.js line 230
			assertMatch(t, false, "../abc", "**/**/abc")
			// dotfiles.js line 231
			assertMatch(t, false, "../abc", "**/**/abc/**")
			// dotfiles.js line 232
			assertMatch(t, false, "../abc", "**/*/*")
			// dotfiles.js line 233
			assertMatch(t, false, "../abc", "**/abc/**")
			// dotfiles.js line 234
			assertMatch(t, false, "../abc", "*/*")
			// dotfiles.js line 235
			assertMatch(t, false, "../abc", "*/abc/**")
			// dotfiles.js line 236
			assertMatch(t, false, "abc/../abc", "**/*")
			// dotfiles.js line 237
			assertMatch(t, false, "abc/../abc", "**/*/*")
			// dotfiles.js line 238
			assertMatch(t, false, "abc/../abc", "**/*/abc")
			// dotfiles.js line 239
			assertMatch(t, false, "abc/../abc", "*/**/*")
			// dotfiles.js line 240
			assertMatch(t, false, "abc/../abc", "*/*/*")
			// dotfiles.js line 241
			assertMatch(t, false, "abc/../abc", "abc/**/*")
			// dotfiles.js line 242
			assertMatch(t, false, "abc/../abc", "**/**/*")
			// dotfiles.js line 243
			assertMatch(t, false, "abc/../abc", "**/*/*")
			// dotfiles.js line 244
			assertMatch(t, false, "abc/../abc", "*/**/*")
			// dotfiles.js line 245
			assertMatch(t, false, "abc/../abc", "*/*/*")

			dotTrue := &Options{Dot: true}
			// dotfiles.js line 247
			assertMatch(t, false, "../abc", "**/**/**", dotTrue)
			// dotfiles.js line 248
			assertMatch(t, false, "../abc", "**/**/abc", dotTrue)
			// dotfiles.js line 249
			assertMatch(t, false, "../abc", "**/**/abc/**", dotTrue)
			// dotfiles.js line 250
			assertMatch(t, false, "../abc", "**/abc/**", dotTrue)
			// dotfiles.js line 251
			assertMatch(t, false, "../abc", "*/abc/**", dotTrue)

			// dotfiles.js line 253
			assertMatch(t, false, "../abc", "**/*/*", dotTrue)
			// dotfiles.js line 254
			assertMatch(t, false, "../abc", "*/*", dotTrue)
			// dotfiles.js line 255
			assertMatch(t, false, "abc/../abc", "**/*/*", dotTrue)
			// dotfiles.js line 256
			assertMatch(t, false, "abc/../abc", "*/*/*", dotTrue)
			// dotfiles.js line 257
			assertMatch(t, false, "abc/../abc", "**/*/*", dotTrue)
			// dotfiles.js line 258
			assertMatch(t, false, "abc/../abc", "*/*/*", dotTrue)
			// dotfiles.js line 259
			assertMatch(t, false, "abc/..", "**/*", dotTrue)
			// dotfiles.js line 260
			assertMatch(t, false, "abc/..", "*/*", dotTrue)
			// dotfiles.js line 261
			assertMatch(t, false, "abc/abc/..", "*/**/*", dotTrue)

			// dotfiles.js line 263
			assertMatch(t, false, "abc/../abc", "abc/**/*")
			// dotfiles.js line 264
			assertMatch(t, false, "abc/../abc", "abc/**/*", dotTrue)
			// dotfiles.js line 265
			assertMatch(t, false, "abc/../abc", "abc/**/*/*", dotTrue)
			// dotfiles.js line 266
			assertMatch(t, false, "abc/../abc", "abc/*/*/*", dotTrue)
			// dotfiles.js line 267
			assertMatch(t, false, "abc/../abc", "abc/**/*/*", dotTrue)
			// dotfiles.js line 268
			assertMatch(t, false, "abc/../abc", "abc/*/*/*", dotTrue)
			// dotfiles.js line 269
			assertMatch(t, false, "abc/..", "abc/**/*", dotTrue)
			// dotfiles.js line 270
			assertMatch(t, false, "abc/..", "abc/*/*", dotTrue)
			// dotfiles.js line 271
			assertMatch(t, false, "abc/abc/..", "abc/*/**/*", dotTrue)

			// dotfiles.js line 273
			assertMatch(t, false, "../abc", "**/*/*", dotTrue)
			// dotfiles.js line 274
			assertMatch(t, false, "../abc", "*/*", dotTrue)
			// dotfiles.js line 275
			assertMatch(t, false, "abc/../abc", "**/*/*", dotTrue)
			// dotfiles.js line 276
			assertMatch(t, false, "abc/../abc", "*/*/*", dotTrue)
			// dotfiles.js line 277
			assertMatch(t, false, "abc/../abc", "**/*/*", dotTrue)
			// dotfiles.js line 278
			assertMatch(t, false, "abc/../abc", "*/*/*", dotTrue)
			// dotfiles.js line 279
			assertMatch(t, false, "abc/..", "**/*", dotTrue)
			// dotfiles.js line 280
			assertMatch(t, false, "abc/..", "*/*", dotTrue)
			// dotfiles.js line 281
			assertMatch(t, false, "abc/abc/..", "*/**/*", dotTrue)

			strictSlashes := &Options{StrictSlashes: true}
			// dotfiles.js line 283
			assertMatch(t, false, "abc/../abc", "abc/**/*", strictSlashes)
			// dotfiles.js line 284
			assertMatch(t, false, "abc/../abc", "abc/**/*/*", strictSlashes)
			// dotfiles.js line 285
			assertMatch(t, false, "abc/../abc", "abc/**/*/*", strictSlashes)
			// dotfiles.js line 286
			assertMatch(t, false, "abc/../abc", "abc/*/*/*", strictSlashes)
			// dotfiles.js line 287
			assertMatch(t, false, "abc/../abc", "abc/**/*/*", strictSlashes)
			// dotfiles.js line 288
			assertMatch(t, false, "abc/../abc", "abc/*/*/*", strictSlashes)
			// dotfiles.js line 289
			assertMatch(t, false, "abc/..", "abc/**/*", strictSlashes)
			// dotfiles.js line 290
			assertMatch(t, false, "abc/..", "abc/*/*", strictSlashes)
			// dotfiles.js line 291
			assertMatch(t, false, "abc/abc/..", "abc/*/**/*", strictSlashes)
		})

		t.Run("should not match single exclusive dots when not defined in pattern", func(t *testing.T) {
			// dotfiles.js line 295
			assertMatch(t, false, ".", "**")
			// dotfiles.js line 296
			assertMatch(t, false, "abc/./abc", "**")
			// dotfiles.js line 297
			assertMatch(t, false, "abc/abc/.", "**")
			// dotfiles.js line 298
			assertMatch(t, false, "abc/abc/./abc", "**")

			dotTrue := &Options{Dot: true}
			// dotfiles.js line 300
			assertMatch(t, false, ".", "**", dotTrue)
			// dotfiles.js line 301
			assertMatch(t, false, "..", "**", dotTrue)
			// dotfiles.js line 302
			assertMatch(t, false, "../", "**", dotTrue)
			// dotfiles.js line 303
			assertMatch(t, false, "/../", "**", dotTrue)
			// dotfiles.js line 304
			assertMatch(t, false, "/..", "**", dotTrue)
			// dotfiles.js line 305
			assertMatch(t, false, "abc/./abc", "**", dotTrue)
			// dotfiles.js line 306
			assertMatch(t, false, "abc/abc/.", "**", dotTrue)
			// dotfiles.js line 307
			assertMatch(t, false, "abc/abc/./abc", "**", dotTrue)
		})

		t.Run("should match leading dots in root path when glob is prefixed with **/", func(t *testing.T) {
			// dotfiles.js line 311
			assertMatch(t, false, ".abc/.abc", "**/.abc/**")
			// dotfiles.js line 312
			assertMatch(t, true, ".abc", "**/.abc/**")
			// dotfiles.js line 313
			assertMatch(t, true, ".abc/", "**/.abc/**")
			// dotfiles.js line 314
			assertMatch(t, true, ".abc/abc", "**/.abc/**")
			// dotfiles.js line 315
			assertMatch(t, true, ".abc/abc/b", "**/.abc/**")
			// dotfiles.js line 316
			assertMatch(t, true, "abc/.abc/b", "**/.abc/**")
			// dotfiles.js line 317
			assertMatch(t, true, "abc/abc/.abc", "**/.abc")
			// dotfiles.js line 318
			assertMatch(t, true, "abc/abc/.abc", "**/.abc/**")
			// dotfiles.js line 319
			assertMatch(t, true, "abc/abc/.abc/", "**/.abc/**")
			// dotfiles.js line 320
			assertMatch(t, true, "abc/abc/.abc/abc", "**/.abc/**")
			// dotfiles.js line 321
			assertMatch(t, true, "abc/abc/.abc/c/d", "**/.abc/**")
			// dotfiles.js line 322
			assertMatch(t, true, "abc/abc/.abc/c/d/e", "**/.abc/**")
		})

		t.Run("should match a dot when the dot is explicitly defined", func(t *testing.T) {
			// dotfiles.js line 326
			assertMatch(t, true, "/.dot", "**/.dot*")
			// dotfiles.js line 327
			assertMatch(t, true, "aaa/bbb/.dot", "**/.dot*")
			// dotfiles.js line 328
			assertMatch(t, true, "aaa/.dot", "*/.dot*")
			// dotfiles.js line 329
			assertMatch(t, true, ".aaa.bbb", ".*.*")
			// dotfiles.js line 330
			assertMatch(t, true, ".aaa.bbb", ".*.*")
			// dotfiles.js line 331
			assertMatch(t, false, ".aaa.bbb/", ".*.*", &Options{StrictSlashes: true})
			// dotfiles.js line 332
			assertMatch(t, false, ".aaa.bbb", ".*.*/")
			// dotfiles.js line 333
			assertMatch(t, true, ".aaa.bbb/", ".*.*/")
			// dotfiles.js line 334
			assertMatch(t, true, ".aaa.bbb/", ".*.*{,/}")
			// dotfiles.js line 335
			assertMatch(t, true, ".aaa.bbb", ".*.bbb")
			// dotfiles.js line 336
			assertMatch(t, true, ".dotfile.js", ".*.js")
			// dotfiles.js line 337
			assertMatch(t, true, ".dot", ".*ot")
			// dotfiles.js line 338
			assertMatch(t, true, ".dot.bbb.ccc", ".*ot.*.*")
			// dotfiles.js line 339
			assertMatch(t, true, ".dot", ".d?t")
			// dotfiles.js line 340
			assertMatch(t, true, ".dot", ".dot*")
			// dotfiles.js line 341
			assertMatch(t, true, "/.dot", "/.dot*")
		})

		t.Run("should match dots defined in brackets", func(t *testing.T) {
			// dotfiles.js line 345
			assertMatch(t, true, "/.dot", "**/.[d]ot")
			// dotfiles.js line 346
			assertMatch(t, true, "aaa/.dot", "**/.[d]ot")
			// dotfiles.js line 347
			assertMatch(t, true, "aaa/bbb/.dot", "**/.[d]ot")
			// dotfiles.js line 348
			assertMatch(t, true, "aaa/.dot", "*/.[d]ot")
			// dotfiles.js line 349
			assertMatch(t, true, ".dot", ".[d]ot")
			// dotfiles.js line 350
			assertMatch(t, true, ".dot", ".[d]ot")
			// dotfiles.js line 351
			assertMatch(t, true, "/.dot", "/.[d]ot")
		})
	})
}
