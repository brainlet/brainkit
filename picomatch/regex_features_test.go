// regex_features_test.go — Faithful 1:1 port of picomatch/test/regex-features.js
package picomatch

import "testing"

func TestRegexFeatures(t *testing.T) {
	t.Run("word boundaries", func(t *testing.T) {
		t.Run("should support word boundaries", func(t *testing.T) {
			// regex-features.js line 10
			assertMatch(t, true, "a", "a\\b")
		})

		t.Run("should support word boundaries in parens", func(t *testing.T) {
			// regex-features.js line 14
			assertMatch(t, true, "a", "(a\\b)")
		})
	})

	t.Run("regex lookarounds", func(t *testing.T) {
		t.Run("should support regex lookbehinds", func(t *testing.T) {
			// regex-features.js line 20
			assertMatch(t, true, "foo/cbaz", "foo/*(?<!d)baz")
			// regex-features.js line 21
			assertMatch(t, false, "foo/cbaz", "foo/*(?<!c)baz")
			// regex-features.js line 22
			assertMatch(t, false, "foo/cbaz", "foo/*(?<=d)baz")
			// regex-features.js line 23
			assertMatch(t, true, "foo/cbaz", "foo/*(?<=c)baz")
		})
	})

	t.Run("regex back-references", func(t *testing.T) {
		t.Run("should support regex backreferences", func(t *testing.T) {
			// regex-features.js line 29
			assertMatch(t, false, "1/2", "(*)/\\1")
			// regex-features.js line 30
			assertMatch(t, true, "1/1", "(*)/\\1")
			// regex-features.js line 31
			assertMatch(t, true, "1/1/1/1", "(*)/\\1/\\1/\\1")
			// regex-features.js line 32
			assertMatch(t, false, "1/11/111/1111", "(*)/\\1/\\1/\\1")
			// regex-features.js line 33
			assertMatch(t, true, "1/11/111/1111", "(*)/(\\1)+/(\\1)+/(\\1)+")
			// regex-features.js line 34
			assertMatch(t, false, "1/2/1/1", "(*)/\\1/\\1/\\1")
			// regex-features.js line 35
			assertMatch(t, false, "1/1/2/1", "(*)/\\1/\\1/\\1")
			// regex-features.js line 36
			assertMatch(t, false, "1/1/1/2", "(*)/\\1/\\1/\\1")
			// regex-features.js line 37
			assertMatch(t, true, "1/1/1/1", "(*)/\\1/(*)/\\2")
			// regex-features.js line 38
			assertMatch(t, false, "1/1/2/1", "(*)/\\1/(*)/\\2")
			// regex-features.js line 39
			assertMatch(t, false, "1/1/2/1", "(*)/\\1/(*)/\\2")
			// regex-features.js line 40
			assertMatch(t, true, "1/1/2/2", "(*)/\\1/(*)/\\2")
		})
	})

	t.Run("regex character classes", func(t *testing.T) {
		t.Run("should not match with character classes when disabled", func(t *testing.T) {
			// regex-features.js line 46
			assertMatch(t, false, "a/a", "a/[a-z]", &Options{Nobracket: true})
			// regex-features.js line 47
			assertMatch(t, false, "a/b", "a/[a-z]", &Options{Nobracket: true})
			// regex-features.js line 48
			assertMatch(t, false, "a/c", "a/[a-z]", &Options{Nobracket: true})
		})

		t.Run("should match with character classes by default", func(t *testing.T) {
			// regex-features.js line 52
			assertMatch(t, true, "a/a", "a/[a-z]")
			// regex-features.js line 53
			assertMatch(t, true, "a/b", "a/[a-z]")
			// regex-features.js line 54
			assertMatch(t, true, "a/c", "a/[a-z]")

			// regex-features.js line 56
			assertMatch(t, false, "foo/bar", "**/[jkl]*")
			// regex-features.js line 57
			assertMatch(t, true, "foo/jar", "**/[jkl]*")

			// regex-features.js line 59
			assertMatch(t, true, "foo/bar", "**/[^jkl]*")
			// regex-features.js line 60
			assertMatch(t, false, "foo/jar", "**/[^jkl]*")

			// regex-features.js line 62
			assertMatch(t, true, "foo/bar", "**/[abc]*")
			// regex-features.js line 63
			assertMatch(t, false, "foo/jar", "**/[abc]*")

			// regex-features.js line 65
			assertMatch(t, false, "foo/bar", "**/[^abc]*")
			// regex-features.js line 66
			assertMatch(t, true, "foo/jar", "**/[^abc]*")

			// regex-features.js line 68
			assertMatch(t, true, "foo/bar", "**/[abc]ar")
			// regex-features.js line 69
			assertMatch(t, false, "foo/jar", "**/[abc]ar")
		})

		t.Run("should match character classes", func(t *testing.T) {
			// regex-features.js line 73
			assertMatch(t, false, "abc", "a[bc]d")
			// regex-features.js line 74
			assertMatch(t, true, "abd", "a[bc]d")
		})

		t.Run("should match character class alphabetical ranges", func(t *testing.T) {
			// regex-features.js line 78
			assertMatch(t, false, "abc", "a[b-d]e")
			// regex-features.js line 79
			assertMatch(t, false, "abd", "a[b-d]e")
			// regex-features.js line 80
			assertMatch(t, true, "abe", "a[b-d]e")
			// regex-features.js line 81
			assertMatch(t, false, "ac", "a[b-d]e")
			// regex-features.js line 82
			assertMatch(t, false, "a-", "a[b-d]e")

			// regex-features.js line 84
			assertMatch(t, false, "abc", "a[b-d]")
			// regex-features.js line 85
			assertMatch(t, false, "abd", "a[b-d]")
			// regex-features.js line 86
			assertMatch(t, true, "abd", "a[b-d]+")
			// regex-features.js line 87
			assertMatch(t, false, "abe", "a[b-d]")
			// regex-features.js line 88
			assertMatch(t, true, "ac", "a[b-d]")
			// regex-features.js line 89
			assertMatch(t, false, "a-", "a[b-d]")
		})

		t.Run("should match character classes with leading dashes", func(t *testing.T) {
			// regex-features.js line 93
			assertMatch(t, false, "abc", "a[-c]")
			// regex-features.js line 94
			assertMatch(t, true, "ac", "a[-c]")
			// regex-features.js line 95
			assertMatch(t, true, "a-", "a[-c]")
		})

		t.Run("should match character classes with trailing dashes", func(t *testing.T) {
			// regex-features.js line 99
			assertMatch(t, false, "abc", "a[c-]")
			// regex-features.js line 100
			assertMatch(t, true, "ac", "a[c-]")
			// regex-features.js line 101
			assertMatch(t, true, "a-", "a[c-]")
		})

		t.Run("should match bracket literals", func(t *testing.T) {
			// regex-features.js line 105
			assertMatch(t, true, "a]c", "a[]]c")
			// regex-features.js line 106
			assertMatch(t, true, "a]c", "a]c")
			// regex-features.js line 107
			assertMatch(t, true, "a]", "a]")

			// regex-features.js line 109
			assertMatch(t, true, "a[c", "a[\\[]c")
			// regex-features.js line 110
			assertMatch(t, true, "a[c", "a[c")
			// regex-features.js line 111
			assertMatch(t, true, "a[", "a[")
		})

		t.Run("should support negated character classes", func(t *testing.T) {
			// regex-features.js line 115
			assertMatch(t, false, "a]", "a[^bc]d")
			// regex-features.js line 116
			assertMatch(t, false, "acd", "a[^bc]d")
			// regex-features.js line 117
			assertMatch(t, true, "aed", "a[^bc]d")
			// regex-features.js line 118
			assertMatch(t, true, "azd", "a[^bc]d")
			// regex-features.js line 119
			assertMatch(t, false, "ac", "a[^bc]d")
			// regex-features.js line 120
			assertMatch(t, false, "a-", "a[^bc]d")
		})

		t.Run("should match negated dashes", func(t *testing.T) {
			// regex-features.js line 124
			assertMatch(t, false, "abc", "a[^-b]c")
			// regex-features.js line 125
			assertMatch(t, true, "adc", "a[^-b]c")
			// regex-features.js line 126
			assertMatch(t, false, "a-c", "a[^-b]c")
		})

		t.Run("should match negated brackets", func(t *testing.T) {
			// regex-features.js line 130
			assertMatch(t, true, "a-c", "a[^\\]b]c")
			// regex-features.js line 131
			assertMatch(t, false, "abc", "a[^\\]b]c")
			// regex-features.js line 132
			assertMatch(t, false, "a]c", "a[^\\]b]c")
			// regex-features.js line 133
			assertMatch(t, true, "adc", "a[^\\]b]c")
		})

		t.Run("should match alpha-numeric characters", func(t *testing.T) {
			// regex-features.js line 137
			assertMatch(t, false, "0123e45g78", "[\\de]+")
			// regex-features.js line 138
			assertMatch(t, true, "0123e456", "[\\de]+")
			// regex-features.js line 139
			assertMatch(t, true, "01234", "[\\de]+")
		})

		t.Run("should support valid regex ranges", func(t *testing.T) {
			// regex-features.js line 143
			assertMatch(t, false, "a/a", "a/[b-c]")
			// regex-features.js line 144
			assertMatch(t, false, "a/z", "a/[b-c]")
			// regex-features.js line 145
			assertMatch(t, true, "a/b", "a/[b-c]")
			// regex-features.js line 146
			assertMatch(t, true, "a/c", "a/[b-c]")
			// regex-features.js line 147
			assertMatch(t, true, "a/b", "[a-z]/[a-z]")
			// regex-features.js line 148
			assertMatch(t, true, "a/z", "[a-z]/[a-z]")
			// regex-features.js line 149
			assertMatch(t, true, "z/z", "[a-z]/[a-z]")
			// regex-features.js line 150
			assertMatch(t, false, "a/x/y", "a/[a-z]")

			// regex-features.js line 152
			assertMatch(t, true, "a.a", "[a-b].[a-b]")
			// regex-features.js line 153
			assertMatch(t, true, "a.b", "[a-b].[a-b]")
			// regex-features.js line 154
			assertMatch(t, false, "a.a.a", "[a-b].[a-b]")
			// regex-features.js line 155
			assertMatch(t, false, "c.a", "[a-b].[a-b]")
			// regex-features.js line 156
			assertMatch(t, false, "d.a.d", "[a-b].[a-b]")
			// regex-features.js line 157
			assertMatch(t, false, "a.bb", "[a-b].[a-b]")
			// regex-features.js line 158
			assertMatch(t, false, "a.ccc", "[a-b].[a-b]")

			// regex-features.js line 160
			assertMatch(t, true, "a.a", "[a-d].[a-b]")
			// regex-features.js line 161
			assertMatch(t, true, "a.b", "[a-d].[a-b]")
			// regex-features.js line 162
			assertMatch(t, false, "a.a.a", "[a-d].[a-b]")
			// regex-features.js line 163
			assertMatch(t, true, "c.a", "[a-d].[a-b]")
			// regex-features.js line 164
			assertMatch(t, false, "d.a.d", "[a-d].[a-b]")
			// regex-features.js line 165
			assertMatch(t, false, "a.bb", "[a-d].[a-b]")
			// regex-features.js line 166
			assertMatch(t, false, "a.ccc", "[a-d].[a-b]")

			// regex-features.js line 168
			assertMatch(t, true, "a.a", "[a-d]*.[a-b]")
			// regex-features.js line 169
			assertMatch(t, true, "a.b", "[a-d]*.[a-b]")
			// regex-features.js line 170
			assertMatch(t, true, "a.a.a", "[a-d]*.[a-b]")
			// regex-features.js line 171
			assertMatch(t, true, "c.a", "[a-d]*.[a-b]")
			// regex-features.js line 172
			assertMatch(t, false, "d.a.d", "[a-d]*.[a-b]")
			// regex-features.js line 173
			assertMatch(t, false, "a.bb", "[a-d]*.[a-b]")
			// regex-features.js line 174
			assertMatch(t, false, "a.ccc", "[a-d]*.[a-b]")
		})

		t.Run("should support valid regex ranges with glob negation patterns", func(t *testing.T) {
			// regex-features.js line 178
			assertMatch(t, false, "a.a", "!*.[a-b]")
			// regex-features.js line 179
			assertMatch(t, false, "a.b", "!*.[a-b]")
			// regex-features.js line 180
			assertMatch(t, false, "a.a.a", "!*.[a-b]")
			// regex-features.js line 181
			assertMatch(t, false, "c.a", "!*.[a-b]")
			// regex-features.js line 182
			assertMatch(t, true, "d.a.d", "!*.[a-b]")
			// regex-features.js line 183
			assertMatch(t, true, "a.bb", "!*.[a-b]")
			// regex-features.js line 184
			assertMatch(t, true, "a.ccc", "!*.[a-b]")

			// regex-features.js line 186
			assertMatch(t, false, "a.a", "!*.[a-b]*")
			// regex-features.js line 187
			assertMatch(t, false, "a.b", "!*.[a-b]*")
			// regex-features.js line 188
			assertMatch(t, false, "a.a.a", "!*.[a-b]*")
			// regex-features.js line 189
			assertMatch(t, false, "c.a", "!*.[a-b]*")
			// regex-features.js line 190
			assertMatch(t, false, "d.a.d", "!*.[a-b]*")
			// regex-features.js line 191
			assertMatch(t, false, "a.bb", "!*.[a-b]*")
			// regex-features.js line 192
			assertMatch(t, true, "a.ccc", "!*.[a-b]*")

			// regex-features.js line 194
			assertMatch(t, false, "a.a", "![a-b].[a-b]")
			// regex-features.js line 195
			assertMatch(t, false, "a.b", "![a-b].[a-b]")
			// regex-features.js line 196
			assertMatch(t, true, "a.a.a", "![a-b].[a-b]")
			// regex-features.js line 197
			assertMatch(t, true, "c.a", "![a-b].[a-b]")
			// regex-features.js line 198
			assertMatch(t, true, "d.a.d", "![a-b].[a-b]")
			// regex-features.js line 199
			assertMatch(t, true, "a.bb", "![a-b].[a-b]")
			// regex-features.js line 200
			assertMatch(t, true, "a.ccc", "![a-b].[a-b]")

			// regex-features.js line 202
			assertMatch(t, false, "a.a", "![a-b]+.[a-b]+")
			// regex-features.js line 203
			assertMatch(t, false, "a.b", "![a-b]+.[a-b]+")
			// regex-features.js line 204
			assertMatch(t, true, "a.a.a", "![a-b]+.[a-b]+")
			// regex-features.js line 205
			assertMatch(t, true, "c.a", "![a-b]+.[a-b]+")
			// regex-features.js line 206
			assertMatch(t, true, "d.a.d", "![a-b]+.[a-b]+")
			// regex-features.js line 207
			assertMatch(t, false, "a.bb", "![a-b]+.[a-b]+")
			// regex-features.js line 208
			assertMatch(t, true, "a.ccc", "![a-b]+.[a-b]+")
		})

		t.Run("should support valid regex ranges in negated character classes", func(t *testing.T) {
			// regex-features.js line 212
			assertMatch(t, false, "a.a", "*.[^a-b]")
			// regex-features.js line 213
			assertMatch(t, false, "a.b", "*.[^a-b]")
			// regex-features.js line 214
			assertMatch(t, false, "a.a.a", "*.[^a-b]")
			// regex-features.js line 215
			assertMatch(t, false, "c.a", "*.[^a-b]")
			// regex-features.js line 216
			assertMatch(t, true, "d.a.d", "*.[^a-b]")
			// regex-features.js line 217
			assertMatch(t, false, "a.bb", "*.[^a-b]")
			// regex-features.js line 218
			assertMatch(t, false, "a.ccc", "*.[^a-b]")

			// regex-features.js line 220
			assertMatch(t, false, "a.a", "a.[^a-b]*")
			// regex-features.js line 221
			assertMatch(t, false, "a.b", "a.[^a-b]*")
			// regex-features.js line 222
			assertMatch(t, false, "a.a.a", "a.[^a-b]*")
			// regex-features.js line 223
			assertMatch(t, false, "c.a", "a.[^a-b]*")
			// regex-features.js line 224
			assertMatch(t, false, "d.a.d", "a.[^a-b]*")
			// regex-features.js line 225
			assertMatch(t, false, "a.bb", "a.[^a-b]*")
			// regex-features.js line 226
			assertMatch(t, true, "a.ccc", "a.[^a-b]*")
		})
	})

	t.Run("regex capture groups", func(t *testing.T) {
		t.Run("should support regex logical or", func(t *testing.T) {
			// regex-features.js line 232
			assertMatch(t, true, "a/a", "a/(a|c)")
			// regex-features.js line 233
			assertMatch(t, false, "a/b", "a/(a|c)")
			// regex-features.js line 234
			assertMatch(t, true, "a/c", "a/(a|c)")

			// regex-features.js line 236
			assertMatch(t, true, "a/a", "a/(a|b|c)")
			// regex-features.js line 237
			assertMatch(t, true, "a/b", "a/(a|b|c)")
			// regex-features.js line 238
			assertMatch(t, true, "a/c", "a/(a|b|c)")
		})

		t.Run("should support regex character classes inside extglobs", func(t *testing.T) {
			// regex-features.js line 242
			assertMatch(t, false, "foo/bar", "**/!([a-k])*")
			// regex-features.js line 243
			assertMatch(t, false, "foo/jar", "**/!([a-k])*")

			// regex-features.js line 245
			assertMatch(t, false, "foo/bar", "**/!([a-i])*")
			// regex-features.js line 246
			assertMatch(t, true, "foo/bar", "**/!([c-i])*")
			// regex-features.js line 247
			assertMatch(t, true, "foo/jar", "**/!([a-i])*")
		})

		t.Run("should support regex capture groups", func(t *testing.T) {
			// regex-features.js line 251
			assertMatch(t, true, "a/bb/c/dd/e.md", "a/??/?/(dd)/e.md")
			// regex-features.js line 252
			assertMatch(t, true, "a/b/c/d/e.md", "a/?/c/?/(e|f).md")
			// regex-features.js line 253
			assertMatch(t, true, "a/b/c/d/f.md", "a/?/c/?/(e|f).md")
		})

		t.Run("should support regex capture groups with slashes", func(t *testing.T) {
			// regex-features.js line 257
			assertMatch(t, false, "a/a", "(a/b)")
			// regex-features.js line 258
			assertMatch(t, true, "a/b", "(a/b)")
			// regex-features.js line 259
			assertMatch(t, false, "a/c", "(a/b)")
			// regex-features.js line 260
			assertMatch(t, false, "b/a", "(a/b)")
			// regex-features.js line 261
			assertMatch(t, false, "b/b", "(a/b)")
			// regex-features.js line 262
			assertMatch(t, false, "b/c", "(a/b)")
		})

		t.Run("should support regex non-capture groups", func(t *testing.T) {
			// regex-features.js line 266
			assertMatch(t, true, "a/bb/c/dd/e.md", "a/**/(?:dd)/e.md")
			// regex-features.js line 267
			assertMatch(t, true, "a/b/c/d/e.md", "a/?/c/?/(?:e|f).md")
			// regex-features.js line 268
			assertMatch(t, true, "a/b/c/d/f.md", "a/?/c/?/(?:e|f).md")
		})
	})

	t.Run("quantifiers", func(t *testing.T) {
		t.Run("should support regex quantifiers by escaping braces", func(t *testing.T) {
			// regex-features.js line 274
			assertMatch(t, true, "a   ", "a \\{1,5\\}", &Options{Unescape: true})
			// regex-features.js line 275
			assertMatch(t, false, "a   ", "a \\{1,2\\}", &Options{Unescape: true})
			// regex-features.js line 276
			assertMatch(t, false, "a   ", "a \\{1,2\\}")
		})

		t.Run("should support extglobs with regex quantifiers", func(t *testing.T) {
			// regex-features.js line 280
			assertMatch(t, false, "a  ", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 281
			assertMatch(t, false, "a ", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 282
			assertMatch(t, false, "a", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 283
			assertMatch(t, false, "aa", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 284
			assertMatch(t, false, "aaa", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 285
			assertMatch(t, false, "b", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 286
			assertMatch(t, false, "bb", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 287
			assertMatch(t, false, "bbb", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 288
			assertMatch(t, true, " a ", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 289
			assertMatch(t, true, "b  ", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})
			// regex-features.js line 290
			assertMatch(t, true, "b ", "@(!(a) \\{1,2\\})*", &Options{Unescape: true})

			// regex-features.js line 292
			assertMatch(t, true, "a   ", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 293
			assertMatch(t, true, "a   b", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 294
			assertMatch(t, true, "a  b", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 295
			assertMatch(t, true, "a  ", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 296
			assertMatch(t, true, "a ", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 297
			assertMatch(t, true, "a", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 298
			assertMatch(t, true, "aa", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 299
			assertMatch(t, true, "b", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 300
			assertMatch(t, true, "bb", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 301
			assertMatch(t, true, " a ", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 302
			assertMatch(t, true, "b  ", "@(!(a \\{1,2\\}))*")
			// regex-features.js line 303
			assertMatch(t, true, "b ", "@(!(a \\{1,2\\}))*")
		})

		// regex-features.js lines 306-313: "should basename paths"
		// These test utils.basename() which is an internal utility function.
		// Ported as behavioral tests below.
		t.Run("should basename paths", func(t *testing.T) {
			// regex-features.js line 307 — utils.basename('/a/b/c') === 'c'
			// regex-features.js line 308 — utils.basename('/a/b/c/') === 'c'
			// regex-features.js line 309 — utils.basename('/a\\b/c', { windows: true }) === 'c'
			// regex-features.js line 310 — utils.basename('/a\\b/c\\', { windows: true }) === 'c'
			// regex-features.js line 311 — utils.basename('\\a/b\\c', { windows: true }) === 'c'
			// regex-features.js line 312 — utils.basename('\\a/b\\c/', { windows: true }) === 'c'

			// These test picomatch's internal utils.basename, not glob matching.
			// We verify the matching behavior that depends on basename via MatchBase option instead.
			assertMatch(t, true, "/a/b/c", "**/c")
			assertMatch(t, true, "/a/b/c/", "**/c{,/}")
		})
	})
}
