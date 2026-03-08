// special_characters_test.go — Faithful 1:1 port of picomatch/test/special-characters.js
package picomatch

import (
	"testing"
)

func TestSpecialCharacters(t *testing.T) {
	t.Run("numbers", func(t *testing.T) {
		t.Run("should match numbers in the input string", func(t *testing.T) {
			// special-characters.js line 9
			assertMatch(t, false, "1", "*/*")
			// special-characters.js line 10
			assertMatch(t, true, "1/1", "*/*")
			// special-characters.js line 11
			assertMatch(t, true, "1/2", "*/*")
			// special-characters.js line 12
			assertMatch(t, false, "1/1/1", "*/*")
			// special-characters.js line 13
			assertMatch(t, false, "1/1/2", "*/*")

			// special-characters.js line 15
			assertMatch(t, false, "1", "*/*/1")
			// special-characters.js line 16
			assertMatch(t, false, "1/1", "*/*/1")
			// special-characters.js line 17
			assertMatch(t, false, "1/2", "*/*/1")
			// special-characters.js line 18
			assertMatch(t, true, "1/1/1", "*/*/1")
			// special-characters.js line 19
			assertMatch(t, false, "1/1/2", "*/*/1")

			// special-characters.js line 21
			assertMatch(t, false, "1", "*/*/2")
			// special-characters.js line 22
			assertMatch(t, false, "1/1", "*/*/2")
			// special-characters.js line 23
			assertMatch(t, false, "1/2", "*/*/2")
			// special-characters.js line 24
			assertMatch(t, false, "1/1/1", "*/*/2")
			// special-characters.js line 25
			assertMatch(t, true, "1/1/2", "*/*/2")
		})
	})

	t.Run("qmarks", func(t *testing.T) {
		t.Run("should match literal ? in the input string", func(t *testing.T) {
			// special-characters.js line 31
			assertMatch(t, true, "?", "*")
			// special-characters.js line 32
			assertMatch(t, true, "/?", "/*")
			// special-characters.js line 33
			assertMatch(t, true, "?/?", "*/*")
			// special-characters.js line 34
			assertMatch(t, true, "?/?/", "*/*/")
			// special-characters.js line 35
			assertMatch(t, true, "/?", "/?")
			// special-characters.js line 36
			assertMatch(t, true, "?/?", "?/?")
			// special-characters.js line 37
			assertMatch(t, true, "foo?/bar?", "*/*")
		})

		t.Run("should not match slashes with qmarks", func(t *testing.T) {
			// special-characters.js line 41
			assertMatch(t, false, "aaa/bbb", "aaa?bbb")
		})

		t.Run("should match literal ? with qmarks", func(t *testing.T) {
			// special-characters.js line 45
			assertMatch(t, false, "?", "??")
			// special-characters.js line 46
			assertMatch(t, false, "?", "???")
			// special-characters.js line 47
			assertMatch(t, false, "??", "?")
			// special-characters.js line 48
			assertMatch(t, false, "??", "???")
			// special-characters.js line 49
			assertMatch(t, false, "???", "?")
			// special-characters.js line 50
			assertMatch(t, false, "???", "??")
			// special-characters.js line 51
			assertMatch(t, false, "ac?", "ab?")
			// special-characters.js line 52
			assertMatch(t, true, "?", "?*")
			// special-characters.js line 53
			assertMatch(t, true, "??", "?*")
			// special-characters.js line 54
			assertMatch(t, true, "???", "?*")
			// special-characters.js line 55
			assertMatch(t, true, "????", "?*")
			// special-characters.js line 56
			assertMatch(t, true, "?", "?")
			// special-characters.js line 57
			assertMatch(t, true, "??", "??")
			// special-characters.js line 58
			assertMatch(t, true, "???", "???")
			// special-characters.js line 59
			assertMatch(t, true, "ab?", "ab?")
		})

		t.Run("should match other non-slash characters with qmarks", func(t *testing.T) {
			// special-characters.js line 63
			assertMatch(t, false, "/a/", "?")
			// special-characters.js line 64
			assertMatch(t, false, "/a/", "??")
			// special-characters.js line 65
			assertMatch(t, false, "/a/", "???")
			// special-characters.js line 66
			assertMatch(t, false, "/a/b/", "??")
			// special-characters.js line 67
			assertMatch(t, false, "aaa/bbb", "aaa?bbb")
			// special-characters.js line 68
			assertMatch(t, false, "aaa//bbb", "aaa?bbb")
			// special-characters.js line 69
			assertMatch(t, false, "aaa\\\\bbb", "aaa?bbb")
			// special-characters.js line 70
			assertMatch(t, true, "acb/", "a?b/")
			// special-characters.js line 71
			assertMatch(t, true, "acdb/", "a??b/")
			// special-characters.js line 72
			assertMatch(t, true, "/acb", "/a?b")
		})

		t.Run("should match non-slash characters when ? is escaped", func(t *testing.T) {
			// special-characters.js line 76
			assertMatch(t, false, "acb/", "a\\?b/")
			// special-characters.js line 77
			assertMatch(t, false, "acdb/", "a\\?\\?b/")
			// special-characters.js line 78
			assertMatch(t, false, "/acb", "/a\\?b")
		})

		t.Run("should match one character per question mark", func(t *testing.T) {
			// special-characters.js line 82
			assertMatch(t, true, "a", "?")
			// special-characters.js line 83
			assertMatch(t, false, "aa", "?")
			// special-characters.js line 84
			assertMatch(t, false, "ab", "?")
			// special-characters.js line 85
			assertMatch(t, false, "aaa", "?")
			// special-characters.js line 86
			assertMatch(t, false, "abcdefg", "?")

			// special-characters.js line 88
			assertMatch(t, false, "a", "??")
			// special-characters.js line 89
			assertMatch(t, true, "aa", "??")
			// special-characters.js line 90
			assertMatch(t, true, "ab", "??")
			// special-characters.js line 91
			assertMatch(t, false, "aaa", "??")
			// special-characters.js line 92
			assertMatch(t, false, "abcdefg", "??")

			// special-characters.js line 94
			assertMatch(t, false, "a", "???")
			// special-characters.js line 95
			assertMatch(t, false, "aa", "???")
			// special-characters.js line 96
			assertMatch(t, false, "ab", "???")
			// special-characters.js line 97
			assertMatch(t, true, "aaa", "???")
			// special-characters.js line 98
			assertMatch(t, false, "abcdefg", "???")

			// special-characters.js line 100
			assertMatch(t, false, "aaa", "a?c")
			// special-characters.js line 101
			assertMatch(t, true, "aac", "a?c")
			// special-characters.js line 102
			assertMatch(t, true, "abc", "a?c")
			// special-characters.js line 103
			assertMatch(t, false, "a", "ab?")
			// special-characters.js line 104
			assertMatch(t, false, "aa", "ab?")
			// special-characters.js line 105
			assertMatch(t, false, "ab", "ab?")
			// special-characters.js line 106
			assertMatch(t, false, "ac", "ab?")
			// special-characters.js line 107
			assertMatch(t, false, "abcd", "ab?")
			// special-characters.js line 108
			assertMatch(t, false, "abbb", "ab?")
			// special-characters.js line 109
			assertMatch(t, true, "acb", "a?b")

			// special-characters.js line 111
			assertMatch(t, false, "a/bb/c/dd/e.md", "a/?/c/?/e.md")
			// special-characters.js line 112
			assertMatch(t, true, "a/bb/c/dd/e.md", "a/??/c/??/e.md")
			// special-characters.js line 113
			assertMatch(t, false, "a/bbb/c.md", "a/??/c.md")
			// special-characters.js line 114
			assertMatch(t, true, "a/b/c.md", "a/?/c.md")
			// special-characters.js line 115
			assertMatch(t, true, "a/b/c/d/e.md", "a/?/c/?/e.md")
			// special-characters.js line 116
			assertMatch(t, false, "a/b/c/d/e.md", "a/?/c/???/e.md")
			// special-characters.js line 117
			assertMatch(t, true, "a/b/c/zzz/e.md", "a/?/c/???/e.md")
			// special-characters.js line 118
			assertMatch(t, false, "a/bb/c.md", "a/?/c.md")
			// special-characters.js line 119
			assertMatch(t, true, "a/bb/c.md", "a/??/c.md")
			// special-characters.js line 120
			assertMatch(t, true, "a/bbb/c.md", "a/???/c.md")
			// special-characters.js line 121
			assertMatch(t, true, "a/bbbb/c.md", "a/????/c.md")
		})

		t.Run("should enforce one character per qmark even when preceded by stars", func(t *testing.T) {
			// special-characters.js line 125
			assertMatch(t, false, "a", "*??")
			// special-characters.js line 126
			assertMatch(t, false, "aa", "*???")
			// special-characters.js line 127
			assertMatch(t, true, "aaa", "*???")
			// special-characters.js line 128
			assertMatch(t, false, "a", "*****??")
			// special-characters.js line 129
			assertMatch(t, false, "aa", "*****???")
			// special-characters.js line 130
			assertMatch(t, true, "aaa", "*****???")
		})

		t.Run("should support qmarks and stars", func(t *testing.T) {
			// special-characters.js line 134
			assertMatch(t, false, "aaa", "a*?c")
			// special-characters.js line 135
			assertMatch(t, true, "aac", "a*?c")
			// special-characters.js line 136
			assertMatch(t, true, "abc", "a*?c")

			// special-characters.js line 138
			assertMatch(t, true, "abc", "a**?c")
			// special-characters.js line 139
			assertMatch(t, false, "abb", "a**?c")
			// special-characters.js line 140
			assertMatch(t, true, "acc", "a**?c")
			// special-characters.js line 141
			assertMatch(t, true, "abc", "a*****?c")

			// special-characters.js line 143
			assertMatch(t, true, "a", "*****?")
			// special-characters.js line 144
			assertMatch(t, true, "aa", "*****?")
			// special-characters.js line 145
			assertMatch(t, true, "abc", "*****?")
			// special-characters.js line 146
			assertMatch(t, true, "zzz", "*****?")
			// special-characters.js line 147
			assertMatch(t, true, "bbb", "*****?")
			// special-characters.js line 148
			assertMatch(t, true, "aaaa", "*****?")

			// special-characters.js line 150
			assertMatch(t, false, "a", "*****??")
			// special-characters.js line 151
			assertMatch(t, true, "aa", "*****??")
			// special-characters.js line 152
			assertMatch(t, true, "abc", "*****??")
			// special-characters.js line 153
			assertMatch(t, true, "zzz", "*****??")
			// special-characters.js line 154
			assertMatch(t, true, "bbb", "*****??")
			// special-characters.js line 155
			assertMatch(t, true, "aaaa", "*****??")

			// special-characters.js line 157
			assertMatch(t, false, "a", "?*****??")
			// special-characters.js line 158
			assertMatch(t, false, "aa", "?*****??")
			// special-characters.js line 159
			assertMatch(t, true, "abc", "?*****??")
			// special-characters.js line 160
			assertMatch(t, true, "zzz", "?*****??")
			// special-characters.js line 161
			assertMatch(t, true, "bbb", "?*****??")
			// special-characters.js line 162
			assertMatch(t, true, "aaaa", "?*****??")

			// special-characters.js line 164
			assertMatch(t, true, "abc", "?*****?c")
			// special-characters.js line 165
			assertMatch(t, false, "abb", "?*****?c")
			// special-characters.js line 166
			assertMatch(t, false, "zzz", "?*****?c")

			// special-characters.js line 168
			assertMatch(t, true, "abc", "?***?****c")
			// special-characters.js line 169
			assertMatch(t, false, "bbb", "?***?****c")
			// special-characters.js line 170
			assertMatch(t, false, "zzz", "?***?****c")

			// special-characters.js line 172
			assertMatch(t, true, "abc", "?***?****?")
			// special-characters.js line 173
			assertMatch(t, true, "bbb", "?***?****?")
			// special-characters.js line 174
			assertMatch(t, true, "zzz", "?***?****?")

			// special-characters.js line 176
			assertMatch(t, true, "abc", "?***?****")
			// special-characters.js line 177
			assertMatch(t, true, "abc", "*******c")
			// special-characters.js line 178
			assertMatch(t, true, "abc", "*******?")
			// special-characters.js line 179
			assertMatch(t, true, "abcdecdhjk", "a*cd**?**??k")
			// special-characters.js line 180
			assertMatch(t, true, "abcdecdhjk", "a**?**cd**?**??k")
			// special-characters.js line 181
			assertMatch(t, true, "abcdecdhjk", "a**?**cd**?**??k***")
			// special-characters.js line 182
			assertMatch(t, true, "abcdecdhjk", "a**?**cd**?**??***k")
			// special-characters.js line 183
			assertMatch(t, true, "abcdecdhjk", "a**?**cd**?**??***k**")
			// special-characters.js line 184
			assertMatch(t, true, "abcdecdhjk", "a****c**?**??*****")
		})

		t.Run("should support qmarks, stars and slashes", func(t *testing.T) {
			// special-characters.js line 188
			assertMatch(t, false, "a/b/c/d/e.md", "a/?/c/?/*/e.md")
			// special-characters.js line 189
			assertMatch(t, true, "a/b/c/d/e/e.md", "a/?/c/?/*/e.md")
			// special-characters.js line 190
			assertMatch(t, true, "a/b/c/d/efghijk/e.md", "a/?/c/?/*/e.md")
			// special-characters.js line 191
			assertMatch(t, true, "a/b/c/d/efghijk/e.md", "a/?/**/e.md")
			// special-characters.js line 192
			assertMatch(t, false, "a/bb/e.md", "a/?/e.md")
			// special-characters.js line 193
			assertMatch(t, true, "a/bb/e.md", "a/??/e.md")
			// special-characters.js line 194
			assertMatch(t, false, "a/bb/e.md", "a/?/**/e.md")
			// special-characters.js line 195
			assertMatch(t, true, "a/b/ccc/e.md", "a/?/**/e.md")
			// special-characters.js line 196
			assertMatch(t, true, "a/b/c/d/efghijk/e.md", "a/*/?/**/e.md")
			// special-characters.js line 197
			assertMatch(t, true, "a/b/c/d/efgh.ijk/e.md", "a/*/?/**/e.md")
			// special-characters.js line 198
			assertMatch(t, true, "a/b.bb/c/d/efgh.ijk/e.md", "a/*/?/**/e.md")
			// special-characters.js line 199
			assertMatch(t, true, "a/bbb/c/d/efgh.ijk/e.md", "a/*/?/**/e.md")
		})

		t.Run("should match non-leading dots", func(t *testing.T) {
			// special-characters.js line 203
			assertMatch(t, true, "aaa.bbb", "aaa?bbb")
		})

		t.Run("should not match leading dots", func(t *testing.T) {
			// special-characters.js line 207
			assertMatch(t, false, ".aaa/bbb", "?aaa/bbb")
			// special-characters.js line 208
			assertMatch(t, false, "aaa/.bbb", "aaa/?bbb")
		})

		t.Run("should match characters preceding a dot", func(t *testing.T) {
			// special-characters.js line 212
			assertMatch(t, true, "a/bbb/abcd.md", "a/*/ab??.md")
			// special-characters.js line 213
			assertMatch(t, true, "a/bbb/abcd.md", "a/bbb/ab??.md")
			// special-characters.js line 214
			assertMatch(t, true, "a/bbb/abcd.md", "a/bbb/ab???md")
		})
	})

	t.Run("parentheses ()", func(t *testing.T) {
		t.Run("should match literal parentheses in the input string", func(t *testing.T) {
			// special-characters.js line 220
			assertMatch(t, false, "my/folder (Work, Accts)", "/*")
			// special-characters.js line 221
			assertMatch(t, true, "my/folder (Work, Accts)", "*/*")
			// special-characters.js line 222
			assertMatch(t, true, "my/folder (Work, Accts)", "*/*,*")
			// special-characters.js line 223
			assertMatch(t, true, "my/folder (Work, Accts)", "*/*(W*, *)*")
			// special-characters.js line 224
			assertMatch(t, true, "my/folder/(Work, Accts)", "**/*(W*, *)*")
			// special-characters.js line 225
			assertMatch(t, false, "my/folder/(Work, Accts)", "*/*(W*, *)*")
			// special-characters.js line 226
			assertMatch(t, true, "foo(bar)baz", "foo*baz")
		})

		t.Run("should match literal parens with brackets", func(t *testing.T) {
			// special-characters.js line 230
			assertMatch(t, true, "foo(bar)baz", "foo[bar()]+baz")
		})

		t.Run("should throw an error on imbalanced, unescaped parens", func(t *testing.T) {
			// special-characters.js line 234-236
			opts := &Options{StrictBrackets: true}

			// MakeRe("*)", {strictBrackets: true}) should panic with "Missing opening"
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected MakeRe(%q, {StrictBrackets:true}) to panic", "*)")
					}
					// special-characters.js line 235: /Missing opening: "\("/
					s, ok := r.(string)
					if !ok || !contains(s, `Missing opening: "("`) {
						t.Errorf("expected panic message to contain %q, got %v", `Missing opening: "("`, r)
					}
				}()
				MakeRe("*)", opts)
			}()

			// MakeRe("*(", {strictBrackets: true}) should panic with "Missing closing"
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected MakeRe(%q, {StrictBrackets:true}) to panic", "*(")
					}
					// special-characters.js line 236: /Missing closing: "\)"/
					s, ok := r.(string)
					if !ok || !contains(s, `Missing closing: ")"`) {
						t.Errorf("expected panic message to contain %q, got %v", `Missing closing: ")"`, r)
					}
				}()
				MakeRe("*(", opts)
			}()
		})

		t.Run("should throw an error on imbalanced, unescaped brackets", func(t *testing.T) {
			// special-characters.js line 239-243
			opts := &Options{StrictBrackets: true}

			// MakeRe("*]", {strictBrackets: true}) should panic with "Missing opening"
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected MakeRe(%q, {StrictBrackets:true}) to panic", "*]")
					}
					// special-characters.js line 241: /Missing opening: "\["/
					s, ok := r.(string)
					if !ok || !contains(s, `Missing opening: "["`) {
						t.Errorf("expected panic message to contain %q, got %v", `Missing opening: "["`, r)
					}
				}()
				MakeRe("*]", opts)
			}()

			// MakeRe("*[", {strictBrackets: true}) should panic with "Missing closing"
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected MakeRe(%q, {StrictBrackets:true}) to panic", "*[")
					}
					// special-characters.js line 242: /Missing closing: "\]"/
					s, ok := r.(string)
					if !ok || !contains(s, `Missing closing: "]"`) {
						t.Errorf("expected panic message to contain %q, got %v", `Missing closing: "]"`, r)
					}
				}()
				MakeRe("*[", opts)
			}()
		})
	})

	t.Run("path characters", func(t *testing.T) {
		t.Run("should match windows drives with globstars", func(t *testing.T) {
			// special-characters.js line 248
			assertMatch(t, true, "bar/", "**")
			// special-characters.js line 249
			assertMatch(t, true, "A://", "**")
			// special-characters.js line 250
			assertMatch(t, true, "B:foo/a/b/c/d", "**")
			// special-characters.js line 251
			assertMatch(t, true, "C:/Users/", "**")
			// special-characters.js line 252
			assertMatch(t, true, "c:\\", "**")
			// special-characters.js line 253
			assertMatch(t, true, "C:\\Users\\", "**")
			// special-characters.js line 254
			assertMatch(t, true, "C:cwd/another", "**")
			// special-characters.js line 255
			assertMatch(t, true, "C:cwd\\another", "**")
		})

		t.Run("should not match multiple windows directories with a single star", func(t *testing.T) {
			// special-characters.js line 259
			assertMatch(t, true, "c:\\", "*{,/}", &Options{Windows: true})
			// special-characters.js line 260
			assertMatch(t, false, "C:\\Users\\", "*", &Options{Windows: true})
			// special-characters.js line 261
			assertMatch(t, false, "C:cwd\\another", "*", &Options{Windows: true})
		})

		t.Run("should match mixed slashes on windows", func(t *testing.T) {
			// special-characters.js line 265
			assertMatch(t, true, "//C://user\\docs\\Letter.txt", "**", &Options{Windows: true})
			// special-characters.js line 266
			assertMatch(t, true, "//C:\\\\user/docs/Letter.txt", "**", &Options{Windows: true})
			// special-characters.js line 267
			assertMatch(t, true, ":\\", "*{,/}", &Options{Windows: true})
			// special-characters.js line 268
			assertMatch(t, true, ":\\", ":*{,/}", &Options{Windows: true})
			// special-characters.js line 269
			assertMatch(t, true, "\\\\foo/bar", "**", &Options{Windows: true})
			// special-characters.js line 270
			assertMatch(t, true, "\\\\foo/bar", "//*/*", &Options{Windows: true})
			// special-characters.js line 271
			assertMatch(t, true, "\\\\unc\\admin$", "**", &Options{Windows: true})
			// special-characters.js line 272
			assertMatch(t, true, "\\\\unc\\admin$", "//*/*$", &Options{Windows: true})
			// special-characters.js line 273
			assertMatch(t, true, "\\\\unc\\admin$\\system32", "//*/*$/*32", &Options{Windows: true})
			// special-characters.js line 274
			assertMatch(t, true, "\\\\unc\\share\\foo", "//u*/s*/f*", &Options{Windows: true})
			// special-characters.js line 275
			assertMatch(t, true, "foo\\bar\\baz", "f*/*/*", &Options{Windows: true})
		})

		t.Run("should match mixed slashes when options.windows is true", func(t *testing.T) {
			// special-characters.js line 279
			assertMatch(t, true, "//C://user\\docs\\Letter.txt", "**", &Options{Windows: true})
			// special-characters.js line 280
			assertMatch(t, true, "//C:\\\\user/docs/Letter.txt", "**", &Options{Windows: true})
			// special-characters.js line 281
			assertMatch(t, true, ":\\", "*{,/}", &Options{Windows: true})
			// special-characters.js line 282
			assertMatch(t, true, ":\\", ":*{,/}", &Options{Windows: true})
			// special-characters.js line 283
			assertMatch(t, true, "\\\\foo/bar", "**", &Options{Windows: true})
			// special-characters.js line 284
			assertMatch(t, true, "\\\\foo/bar", "//*/*", &Options{Windows: true})
			// special-characters.js line 285
			assertMatch(t, true, "\\\\unc\\admin$", "//**", &Options{Windows: true})
			// special-characters.js line 286
			assertMatch(t, true, "\\\\unc\\admin$", "//*/*$", &Options{Windows: true})
			// special-characters.js line 287
			assertMatch(t, true, "\\\\unc\\admin$\\system32", "//*/*$/*32", &Options{Windows: true})
			// special-characters.js line 288
			assertMatch(t, true, "\\\\unc\\share\\foo", "//u*/s*/f*", &Options{Windows: true})
			// special-characters.js line 289
			assertMatch(t, true, "\\\\\\\\\\\\unc\\share\\foo", "/\\{1,\\}u*/s*/f*", &Options{Windows: true, Unescape: true})
			// special-characters.js line 290
			assertMatch(t, true, "foo\\bar\\baz", "f*/*/*", &Options{Windows: true})
			// special-characters.js line 291
			assertMatch(t, true, "//*:/**", "**")
			// special-characters.js line 292
			assertMatch(t, false, "//server/file", "//*")
			// special-characters.js line 293
			assertMatch(t, true, "//server/file", "/**")
			// special-characters.js line 294
			assertMatch(t, true, "//server/file", "//**")
			// special-characters.js line 295
			assertMatch(t, true, "//server/file", "**")
			// special-characters.js line 296
			assertMatch(t, true, "//UNC//Server01//user//docs//Letter.txt", "**")
			// special-characters.js line 297
			assertMatch(t, true, "/foo", "**")
			// special-characters.js line 298
			assertMatch(t, true, "/foo/a/b/c/d", "**")
			// special-characters.js line 299
			assertMatch(t, true, "/foo/bar", "**")
			// special-characters.js line 300
			assertMatch(t, true, "/home/foo", "**")
			// special-characters.js line 301
			assertMatch(t, true, "/home/foo/..", "**/..") // NOTE: JS has isMatch('/home/foo/..', '**/..')
			// special-characters.js line 302
			assertMatch(t, true, "/user/docs/Letter.txt", "**")
			// special-characters.js line 303
			assertMatch(t, true, "directory\\directory", "**")
			// special-characters.js line 304
			assertMatch(t, true, "a/b/c.js", "**")
			// special-characters.js line 305
			assertMatch(t, true, "directory/directory", "**")
			// special-characters.js line 306
			assertMatch(t, true, "foo/bar", "**")
		})

		t.Run("should match any character zero or more times, except for /", func(t *testing.T) {
			// special-characters.js line 310
			assertMatch(t, false, "foo", "*a*")
			// special-characters.js line 311
			assertMatch(t, false, "foo", "*r")
			// special-characters.js line 312
			assertMatch(t, false, "foo", "b*")
			// special-characters.js line 313
			assertMatch(t, false, "foo/bar", "*")
			// special-characters.js line 314
			assertMatch(t, true, "foo/bar", "*/*")
			// special-characters.js line 315
			assertMatch(t, false, "foo/bar/baz", "*/*")
			// special-characters.js line 316
			assertMatch(t, true, "bar", "*a*")
			// special-characters.js line 317
			assertMatch(t, true, "bar", "*r")
			// special-characters.js line 318
			assertMatch(t, true, "bar", "b*")
			// special-characters.js line 319
			assertMatch(t, true, "foo/bar/baz", "*/*/*")
		})

		t.Run("should match dashes surrounded by spaces", func(t *testing.T) {
			// special-characters.js line 323
			assertMatch(t, true, "my/folder - 1", "*/*")
			// special-characters.js line 324
			assertMatch(t, true, "my/folder - copy (1)", "*/*")
			// special-characters.js line 325
			assertMatch(t, true, "my/folder - copy [1]", "*/*")
			// special-characters.js line 326
			assertMatch(t, true, "my/folder - foo + bar - copy [1]", "*/*")
			// special-characters.js line 327
			assertMatch(t, false, "my/folder - foo + bar - copy [1]", "*")

			// special-characters.js line 329
			assertMatch(t, true, "my/folder - 1", "*/*-*")
			// special-characters.js line 330
			assertMatch(t, true, "my/folder - copy (1)", "*/*-*")
			// special-characters.js line 331
			assertMatch(t, true, "my/folder - copy [1]", "*/*-*")
			// special-characters.js line 332
			assertMatch(t, true, "my/folder - foo + bar - copy [1]", "*/*-*")

			// special-characters.js line 334
			assertMatch(t, true, "my/folder - 1", "*/*1")
			// special-characters.js line 335
			assertMatch(t, false, "my/folder - copy (1)", "*/*1")
		})
	})

	t.Run("brackets", func(t *testing.T) {
		t.Run("should support square brackets in globs", func(t *testing.T) {
			// special-characters.js line 341
			assertMatch(t, true, "foo/bar - 1", "**/*[1]")
			// special-characters.js line 342
			assertMatch(t, false, "foo/bar - copy (1)", "**/*[1]")
			// special-characters.js line 343
			assertMatch(t, false, "foo/bar (1)", "**/*[1]")
			// special-characters.js line 344
			assertMatch(t, false, "foo/bar (4)", "**/*[1]")
			// special-characters.js line 345
			assertMatch(t, false, "foo/bar (7)", "**/*[1]")
			// special-characters.js line 346
			assertMatch(t, false, "foo/bar (42)", "**/*[1]")
			// special-characters.js line 347
			assertMatch(t, true, "foo/bar - copy [1]", "**/*[1]")
			// special-characters.js line 348
			assertMatch(t, true, "foo/bar - foo + bar - copy [1]", "**/*[1]")
		})

		t.Run("should match (escaped) bracket literals", func(t *testing.T) {
			// special-characters.js line 352
			assertMatch(t, true, "a [b]", "a \\[b\\]")
			// special-characters.js line 353
			assertMatch(t, true, "a [b] c", "a [b] c")
			// special-characters.js line 354
			assertMatch(t, true, "a [b]", "a \\[b\\]*")
			// special-characters.js line 355
			assertMatch(t, true, "a [bc]", "a \\[bc\\]*")
			// special-characters.js line 356
			assertMatch(t, false, "a [b]", "a \\[b\\].*")
			// special-characters.js line 357
			assertMatch(t, true, "a [b].js", "a \\[b\\].*")
			// special-characters.js line 358
			assertMatch(t, false, "foo/bar - 1", "**/*\\[*\\]")
			// special-characters.js line 359
			assertMatch(t, false, "foo/bar - copy (1)", "**/*\\[*\\]")
			// special-characters.js line 360
			assertMatch(t, false, "foo/bar (1)", "**/*\\[*\\]")
			// special-characters.js line 361
			assertMatch(t, false, "foo/bar (4)", "**/*\\[*\\]")
			// special-characters.js line 362
			assertMatch(t, false, "foo/bar (7)", "**/*\\[*\\]")
			// special-characters.js line 363
			assertMatch(t, false, "foo/bar (42)", "**/*\\[*\\]")
			// special-characters.js line 364
			assertMatch(t, true, "foo/bar - copy [1]", "**/*\\[*\\]")
			// special-characters.js line 365
			assertMatch(t, true, "foo/bar - foo + bar - copy [1]", "**/*\\[*\\]")

			// special-characters.js line 367
			assertMatch(t, false, "foo/bar - 1", "**/*\\[1\\]")
			// special-characters.js line 368
			assertMatch(t, false, "foo/bar - copy (1)", "**/*\\[1\\]")
			// special-characters.js line 369
			assertMatch(t, false, "foo/bar (1)", "**/*\\[1\\]")
			// special-characters.js line 370
			assertMatch(t, false, "foo/bar (4)", "**/*\\[1\\]")
			// special-characters.js line 371
			assertMatch(t, false, "foo/bar (7)", "**/*\\[1\\]")
			// special-characters.js line 372
			assertMatch(t, false, "foo/bar (42)", "**/*\\[1\\]")
			// special-characters.js line 373
			assertMatch(t, true, "foo/bar - copy [1]", "**/*\\[1\\]")
			// special-characters.js line 374
			assertMatch(t, true, "foo/bar - foo + bar - copy [1]", "**/*\\[1\\]")

			// special-characters.js line 376
			assertMatch(t, false, "foo/bar - 1", "*/*\\[*\\]")
			// special-characters.js line 377
			assertMatch(t, false, "foo/bar - copy (1)", "*/*\\[*\\]")
			// special-characters.js line 378
			assertMatch(t, false, "foo/bar (1)", "*/*\\[*\\]")
			// special-characters.js line 379
			assertMatch(t, false, "foo/bar (4)", "*/*\\[*\\]")
			// special-characters.js line 380
			assertMatch(t, false, "foo/bar (7)", "*/*\\[*\\]")
			// special-characters.js line 381
			assertMatch(t, false, "foo/bar (42)", "*/*\\[*\\]")
			// special-characters.js line 382
			assertMatch(t, true, "foo/bar - copy [1]", "*/*\\[*\\]")
			// special-characters.js line 383
			assertMatch(t, true, "foo/bar - foo + bar - copy [1]", "*/*\\[*\\]")

			// special-characters.js line 385
			assertMatch(t, true, "a [b]", "a \\[b\\]")
			// special-characters.js line 386
			assertMatch(t, true, "a [b] c", "a [b] c")
			// special-characters.js line 387
			assertMatch(t, true, "a [b]", "a \\[b\\]*")
			// special-characters.js line 388
			assertMatch(t, true, "a [bc]", "a \\[bc\\]*")
			// special-characters.js line 389
			assertMatch(t, false, "a [b]", "a \\[b\\].*")
			// special-characters.js line 390
			assertMatch(t, true, "a [b].js", "a \\[b\\].*")
		})
	})

	t.Run("star - \"*\"", func(t *testing.T) {
		t.Run("should match literal *", func(t *testing.T) {
			// special-characters.js line 396
			assertMatch(t, true, "*", "*")
			// special-characters.js line 397
			assertMatch(t, true, "*/*", "*/*")
			// special-characters.js line 398
			assertMatch(t, true, "*/*", "?/?")
			// special-characters.js line 399
			assertMatch(t, true, "*/*/", "*/*/")
			// special-characters.js line 400
			assertMatch(t, true, "/*", "/*")
			// special-characters.js line 401
			assertMatch(t, true, "/*", "/?")
			// special-characters.js line 402
			assertMatch(t, true, "foo*/bar*", "*/*")
		})

		t.Run("should support stars following brackets", func(t *testing.T) {
			// special-characters.js line 406
			assertMatch(t, true, "a", "[a]*")
			// special-characters.js line 407
			assertMatch(t, true, "aa", "[a]*")
			// special-characters.js line 408
			assertMatch(t, true, "aaa", "[a]*")
			// special-characters.js line 409
			assertMatch(t, true, "az", "[a-z]*")
			// special-characters.js line 410
			assertMatch(t, true, "zzz", "[a-z]*")
		})

		t.Run("should support stars following parens", func(t *testing.T) {
			// special-characters.js line 414
			assertMatch(t, true, "a", "(a)*")
			// special-characters.js line 415
			assertMatch(t, true, "ab", "(a|b)*")
			// special-characters.js line 416
			assertMatch(t, true, "aa", "(a)*")
			// special-characters.js line 417
			assertMatch(t, true, "aaab", "(a|b)*")
			// special-characters.js line 418
			assertMatch(t, true, "aaabbb", "(a|b)*")
		})

		t.Run("should not match slashes with single stars", func(t *testing.T) {
			// special-characters.js line 422
			assertMatch(t, false, "a/b", "(a)*")
			// special-characters.js line 423
			assertMatch(t, false, "a/b", "[a]*")
			// special-characters.js line 424
			assertMatch(t, false, "a/b", "a*")
			// special-characters.js line 425
			assertMatch(t, false, "a/b", "(a|b)*")
		})

		t.Run("should not match dots with stars by default", func(t *testing.T) {
			// special-characters.js line 429
			assertMatch(t, false, ".a", "(a)*")
			// special-characters.js line 430
			assertMatch(t, false, ".a", "*[a]*")
			// special-characters.js line 431
			assertMatch(t, false, ".a", "*[a]")
			// special-characters.js line 432
			assertMatch(t, false, ".a", "*a*")
			// special-characters.js line 433
			assertMatch(t, false, ".a", "*a")
			// special-characters.js line 434
			assertMatch(t, false, ".a", "*(a|b)")
		})
	})

	t.Run("plus - \"+\"", func(t *testing.T) {
		t.Run("should match literal +", func(t *testing.T) {
			// special-characters.js line 440
			assertMatch(t, true, "+", "*")
			// special-characters.js line 441
			assertMatch(t, true, "/+", "/*")
			// special-characters.js line 442
			assertMatch(t, true, "+/+", "*/*")
			// special-characters.js line 443
			assertMatch(t, true, "+/+/", "*/*/")
			// special-characters.js line 444
			assertMatch(t, true, "/+", "/+")
			// special-characters.js line 445
			assertMatch(t, true, "/+", "/?")
			// special-characters.js line 446
			assertMatch(t, true, "+/+", "?/?")
			// special-characters.js line 447
			assertMatch(t, true, "+/+", "+/+")
			// special-characters.js line 448
			assertMatch(t, true, "foo+/bar+", "*/*")
		})

		t.Run("should support plus signs that follow brackets (and not escape them)", func(t *testing.T) {
			// special-characters.js line 452
			assertMatch(t, true, "a", "[a]+")
			// special-characters.js line 453
			assertMatch(t, true, "aa", "[a]+")
			// special-characters.js line 454
			assertMatch(t, true, "aaa", "[a]+")
			// special-characters.js line 455
			assertMatch(t, true, "az", "[a-z]+")
			// special-characters.js line 456
			assertMatch(t, true, "zzz", "[a-z]+")
		})

		t.Run("should not escape plus signs that follow parens", func(t *testing.T) {
			// special-characters.js line 460
			assertMatch(t, true, "a", "(a)+")
			// special-characters.js line 461
			assertMatch(t, true, "ab", "(a|b)+")
			// special-characters.js line 462
			assertMatch(t, true, "aa", "(a)+")
			// special-characters.js line 463
			assertMatch(t, true, "aaab", "(a|b)+")
			// special-characters.js line 464
			assertMatch(t, true, "aaabbb", "(a|b)+")
		})

		t.Run("should escape plus signs to match string literals", func(t *testing.T) {
			// special-characters.js line 468
			assertMatch(t, true, "a+b/src/glimini.js", "a+b/src/*.js")
			// special-characters.js line 469
			assertMatch(t, true, "+b/src/glimini.js", "+b/src/*.js")
			// special-characters.js line 470
			assertMatch(t, true, "coffee+/src/glimini.js", "coffee+/src/*.js")
			// special-characters.js line 471
			assertMatch(t, true, "coffee+/src/glimini.js", "coffee+/src/*.js")
			// special-characters.js line 472
			assertMatch(t, true, "coffee+/src/glimini.js", "coffee+/src/*")
		})

		t.Run("should not escape + following brackets", func(t *testing.T) {
			// special-characters.js line 476
			assertMatch(t, true, "a", "[a]+")
			// special-characters.js line 477
			assertMatch(t, true, "aa", "[a]+")
			// special-characters.js line 478
			assertMatch(t, true, "aaa", "[a]+")
			// special-characters.js line 479
			assertMatch(t, true, "az", "[a-z]+")
			// special-characters.js line 480
			assertMatch(t, true, "zzz", "[a-z]+")
		})

		t.Run("should not escape + following parens", func(t *testing.T) {
			// special-characters.js line 484
			assertMatch(t, true, "a", "(a)+")
			// special-characters.js line 485
			assertMatch(t, true, "ab", "(a|b)+")
			// special-characters.js line 486
			assertMatch(t, true, "aa", "(a)+")
			// special-characters.js line 487
			assertMatch(t, true, "aaab", "(a|b)+")
			// special-characters.js line 488
			assertMatch(t, true, "aaabbb", "(a|b)+")
		})
	})

	t.Run("dollar $", func(t *testing.T) {
		t.Run("should match dollar signs", func(t *testing.T) {
			// special-characters.js line 494
			assertMatch(t, false, "$", "!($)")
			// special-characters.js line 495
			assertMatch(t, false, "$", "!$")
			// special-characters.js line 496
			assertMatch(t, true, "$$", "!$")
			// special-characters.js line 497
			assertMatch(t, true, "$$", "!($)")
			// special-characters.js line 498
			assertMatch(t, true, "$$$", "!($)")
			// special-characters.js line 499
			assertMatch(t, true, "^", "!($)")

			// special-characters.js line 501
			assertMatch(t, true, "$", "!($$)")
			// special-characters.js line 502
			assertMatch(t, false, "$$", "!($$)")
			// special-characters.js line 503
			assertMatch(t, true, "$$$", "!($$)")
			// special-characters.js line 504
			assertMatch(t, true, "^", "!($$)")

			// special-characters.js line 506
			assertMatch(t, false, "$", "!($*)")
			// special-characters.js line 507
			assertMatch(t, false, "$$", "!($*)")
			// special-characters.js line 508
			assertMatch(t, false, "$$$", "!($*)")
			// special-characters.js line 509
			assertMatch(t, true, "^", "!($*)")

			// special-characters.js line 511
			assertMatch(t, true, "$", "*")
			// special-characters.js line 512
			assertMatch(t, true, "$$", "*")
			// special-characters.js line 513
			assertMatch(t, true, "$$$", "*")
			// special-characters.js line 514
			assertMatch(t, true, "^", "*")

			// special-characters.js line 516
			assertMatch(t, true, "$", "$*")
			// special-characters.js line 517
			assertMatch(t, true, "$$", "$*")
			// special-characters.js line 518
			assertMatch(t, true, "$$$", "$*")
			// special-characters.js line 519
			assertMatch(t, false, "^", "$*")

			// special-characters.js line 521
			assertMatch(t, true, "$", "*$*")
			// special-characters.js line 522
			assertMatch(t, true, "$$", "*$*")
			// special-characters.js line 523
			assertMatch(t, true, "$$$", "*$*")
			// special-characters.js line 524
			assertMatch(t, false, "^", "*$*")

			// special-characters.js line 526
			assertMatch(t, true, "$", "*$")
			// special-characters.js line 527
			assertMatch(t, true, "$$", "*$")
			// special-characters.js line 528
			assertMatch(t, true, "$$$", "*$")
			// special-characters.js line 529
			assertMatch(t, false, "^", "*$")

			// special-characters.js line 531
			assertMatch(t, false, "$", "?$")
			// special-characters.js line 532
			assertMatch(t, true, "$$", "?$")
			// special-characters.js line 533
			assertMatch(t, false, "$$$", "?$")
			// special-characters.js line 534
			assertMatch(t, false, "^", "?$")
		})
	})

	t.Run("caret ^", func(t *testing.T) {
		t.Run("should match carets", func(t *testing.T) {
			// special-characters.js line 540
			assertMatch(t, true, "^", "^")
			// special-characters.js line 541
			assertMatch(t, true, "^/foo", "^/*")
			// special-characters.js line 542
			assertMatch(t, true, "^/foo", "^/*")
			// special-characters.js line 543
			assertMatch(t, true, "foo^", "*^")
			// special-characters.js line 544
			assertMatch(t, true, "^foo/foo", "^foo/*")
			// special-characters.js line 545
			assertMatch(t, true, "foo^/foo", "foo^/*")

			// special-characters.js line 547
			assertMatch(t, false, "^", "!(^)")
			// special-characters.js line 548
			assertMatch(t, true, "^^", "!(^)")
			// special-characters.js line 549
			assertMatch(t, true, "^^^", "!(^)")
			// special-characters.js line 550
			assertMatch(t, true, "&", "!(^)")

			// special-characters.js line 552
			assertMatch(t, true, "^", "!(^^)")
			// special-characters.js line 553
			assertMatch(t, false, "^^", "!(^^)")
			// special-characters.js line 554
			assertMatch(t, true, "^^^", "!(^^)")
			// special-characters.js line 555
			assertMatch(t, true, "&", "!(^^)")

			// special-characters.js line 557
			assertMatch(t, false, "^", "!(^*)")
			// special-characters.js line 558
			assertMatch(t, false, "^^", "!(^*)")
			// special-characters.js line 559
			assertMatch(t, false, "^^^", "!(^*)")
			// special-characters.js line 560
			assertMatch(t, true, "&", "!(^*)")

			// special-characters.js line 562
			assertMatch(t, true, "^", "*")
			// special-characters.js line 563
			assertMatch(t, true, "^^", "*")
			// special-characters.js line 564
			assertMatch(t, true, "^^^", "*")
			// special-characters.js line 565
			assertMatch(t, true, "&", "*")

			// special-characters.js line 567
			assertMatch(t, true, "^", "^*")
			// special-characters.js line 568
			assertMatch(t, true, "^^", "^*")
			// special-characters.js line 569
			assertMatch(t, true, "^^^", "^*")
			// special-characters.js line 570
			assertMatch(t, false, "&", "^*")

			// special-characters.js line 572
			assertMatch(t, true, "^", "*^*")
			// special-characters.js line 573
			assertMatch(t, true, "^^", "*^*")
			// special-characters.js line 574
			assertMatch(t, true, "^^^", "*^*")
			// special-characters.js line 575
			assertMatch(t, false, "&", "*^*")

			// special-characters.js line 577
			assertMatch(t, true, "^", "*^")
			// special-characters.js line 578
			assertMatch(t, true, "^^", "*^")
			// special-characters.js line 579
			assertMatch(t, true, "^^^", "*^")
			// special-characters.js line 580
			assertMatch(t, false, "&", "*^")

			// special-characters.js line 582
			assertMatch(t, false, "^", "?^")
			// special-characters.js line 583
			assertMatch(t, true, "^^", "?^")
			// special-characters.js line 584
			assertMatch(t, false, "^^^", "?^")
			// special-characters.js line 585
			assertMatch(t, false, "&", "?^")
		})
	})

	t.Run("mixed special characters", func(t *testing.T) {
		t.Run("should match special characters in paths", func(t *testing.T) {
			// special-characters.js line 591
			assertMatch(t, true, "my/folder +1", "*/*")
			// special-characters.js line 592
			assertMatch(t, true, "my/folder -1", "*/*")
			// special-characters.js line 593
			assertMatch(t, true, "my/folder *1", "*/*")
			// special-characters.js line 594
			assertMatch(t, true, "my/folder", "*/*")
			// special-characters.js line 595
			assertMatch(t, true, "my/folder+foo+bar&baz", "*/*")
			// special-characters.js line 596
			assertMatch(t, true, "my/folder - $1.00", "*/*")
			// special-characters.js line 597
			assertMatch(t, true, "my/folder - ^1.00", "*/*")
			// special-characters.js line 598
			assertMatch(t, true, "my/folder - %1.00", "*/*")

			// special-characters.js line 600
			assertMatch(t, true, "my/folder +1", "*/!(*%)*")
			// special-characters.js line 601
			assertMatch(t, true, "my/folder -1", "*/!(*%)*")
			// special-characters.js line 602
			assertMatch(t, true, "my/folder *1", "*/!(*%)*")
			// special-characters.js line 603
			assertMatch(t, true, "my/folder", "*/!(*%)*")
			// special-characters.js line 604
			assertMatch(t, true, "my/folder+foo+bar&baz", "*/!(*%)*")
			// special-characters.js line 605
			assertMatch(t, true, "my/folder - $1.00", "*/!(*%)*")
			// special-characters.js line 606
			assertMatch(t, true, "my/folder - ^1.00", "*/!(*%)*")
			// special-characters.js line 607
			assertMatch(t, false, "my/folder - %1.00", "*/!(*%)*")

			// special-characters.js line 609
			assertMatch(t, false, "my/folder +1", "*/*$*")
			// special-characters.js line 610
			assertMatch(t, false, "my/folder -1", "*/*$*")
			// special-characters.js line 611
			assertMatch(t, false, "my/folder *1", "*/*$*")
			// special-characters.js line 612
			assertMatch(t, false, "my/folder", "*/*$*")
			// special-characters.js line 613
			assertMatch(t, false, "my/folder+foo+bar&baz", "*/*$*")
			// special-characters.js line 614
			assertMatch(t, true, "my/folder - $1.00", "*/*$*")
			// special-characters.js line 615
			assertMatch(t, false, "my/folder - ^1.00", "*/*$*")
			// special-characters.js line 616
			assertMatch(t, false, "my/folder - %1.00", "*/*$*")

			// special-characters.js line 618
			assertMatch(t, false, "my/folder +1", "*/*^*")
			// special-characters.js line 619
			assertMatch(t, false, "my/folder -1", "*/*^*")
			// special-characters.js line 620
			assertMatch(t, false, "my/folder *1", "*/*^*")
			// special-characters.js line 621
			assertMatch(t, false, "my/folder", "*/*^*")
			// special-characters.js line 622
			assertMatch(t, false, "my/folder+foo+bar&baz", "*/*^*")
			// special-characters.js line 623
			assertMatch(t, false, "my/folder - $1.00", "*/*^*")
			// special-characters.js line 624
			assertMatch(t, true, "my/folder - ^1.00", "*/*^*")
			// special-characters.js line 625
			assertMatch(t, false, "my/folder - %1.00", "*/*^*")

			// special-characters.js line 627
			assertMatch(t, false, "my/folder +1", "*/*&*")
			// special-characters.js line 628
			assertMatch(t, false, "my/folder -1", "*/*&*")
			// special-characters.js line 629
			assertMatch(t, false, "my/folder *1", "*/*&*")
			// special-characters.js line 630
			assertMatch(t, false, "my/folder", "*/*&*")
			// special-characters.js line 631
			assertMatch(t, true, "my/folder+foo+bar&baz", "*/*&*")
			// special-characters.js line 632
			assertMatch(t, false, "my/folder - $1.00", "*/*&*")
			// special-characters.js line 633
			assertMatch(t, false, "my/folder - ^1.00", "*/*&*")
			// special-characters.js line 634
			assertMatch(t, false, "my/folder - %1.00", "*/*&*")

			// special-characters.js line 636
			assertMatch(t, true, "my/folder +1", "*/*+*")
			// special-characters.js line 637
			assertMatch(t, false, "my/folder -1", "*/*+*")
			// special-characters.js line 638
			assertMatch(t, false, "my/folder *1", "*/*+*")
			// special-characters.js line 639
			assertMatch(t, false, "my/folder", "*/*+*")
			// special-characters.js line 640
			assertMatch(t, true, "my/folder+foo+bar&baz", "*/*+*")
			// special-characters.js line 641
			assertMatch(t, false, "my/folder - $1.00", "*/*+*")
			// special-characters.js line 642
			assertMatch(t, false, "my/folder - ^1.00", "*/*+*")
			// special-characters.js line 643
			assertMatch(t, false, "my/folder - %1.00", "*/*+*")

			// special-characters.js line 645
			assertMatch(t, false, "my/folder +1", "*/*-*")
			// special-characters.js line 646
			assertMatch(t, true, "my/folder -1", "*/*-*")
			// special-characters.js line 647
			assertMatch(t, false, "my/folder *1", "*/*-*")
			// special-characters.js line 648
			assertMatch(t, false, "my/folder", "*/*-*")
			// special-characters.js line 649
			assertMatch(t, false, "my/folder+foo+bar&baz", "*/*-*")
			// special-characters.js line 650
			assertMatch(t, true, "my/folder - $1.00", "*/*-*")
			// special-characters.js line 651
			assertMatch(t, true, "my/folder - ^1.00", "*/*-*")
			// special-characters.js line 652
			assertMatch(t, true, "my/folder - %1.00", "*/*-*")

			// special-characters.js line 654
			assertMatch(t, false, "my/folder +1", "*/*\\**")
			// special-characters.js line 655
			assertMatch(t, false, "my/folder -1", "*/*\\**")
			// special-characters.js line 656
			assertMatch(t, true, "my/folder *1", "*/*\\**")
			// special-characters.js line 657
			assertMatch(t, false, "my/folder", "*/*\\**")
			// special-characters.js line 658
			assertMatch(t, false, "my/folder+foo+bar&baz", "*/*\\**")
			// special-characters.js line 659
			assertMatch(t, false, "my/folder - $1.00", "*/*\\**")
			// special-characters.js line 660
			assertMatch(t, false, "my/folder - ^1.00", "*/*\\**")
			// special-characters.js line 661
			assertMatch(t, false, "my/folder - %1.00", "*/*\\**")
		})
	})
}

// contains is a simple helper for checking substring presence in panic messages.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

// searchSubstring does a naive substring search.
func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
