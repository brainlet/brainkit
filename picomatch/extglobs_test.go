// extglobs_test.go — Faithful 1:1 port of picomatch/test/extglobs.js
package picomatch

import (
	"runtime"
	"strings"
	"testing"
)

func TestExtglobs(t *testing.T) {
	// extglobs.js:12-16 — "should throw on imbalanced sets when `strictBrackets` is true"
	t.Run("should throw on imbalanced sets when strictBrackets is true", func(t *testing.T) {
		opts := &Options{StrictBrackets: true}

		// extglobs.js:13 — assert.throws(() => makeRe('a(b', opts), /Missing closing: "\)"/i);
		t.Run("a(b should panic with Missing closing", func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("expected MakeRe(%q, {StrictBrackets:true}) to panic", "a(b")
					return
				}
				s, ok := r.(string)
				if !ok || !strings.Contains(strings.ToLower(s), "missing closing") {
					t.Errorf("expected panic to contain 'Missing closing', got: %v", r)
				}
			}()
			MakeRe("a(b", opts) // extglobs.js:13
		})

		// extglobs.js:14 — assert.throws(() => makeRe('a)b', opts), /Missing opening: "\("/i);
		t.Run("a)b should panic with Missing opening", func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("expected MakeRe(%q, {StrictBrackets:true}) to panic", "a)b")
					return
				}
				s, ok := r.(string)
				if !ok || !strings.Contains(strings.ToLower(s), "missing opening") {
					t.Errorf("expected panic to contain 'Missing opening', got: %v", r)
				}
			}()
			MakeRe("a)b", opts) // extglobs.js:14
		})
	})

	t.Run("should escape special characters immediately following opening parens", func(t *testing.T) {
		// extglobs.js line 19
		assertMatch(t, true, "cbz", "c!(.)z")
		// extglobs.js line 20
		assertMatch(t, false, "cbz", "c!(*)z")
		// extglobs.js line 21
		assertMatch(t, true, "cccz", "c!(b*)z")
		// extglobs.js line 22
		assertMatch(t, true, "cbz", "c!(+)z")
		// extglobs.js line 23
		assertMatch(t, true, "cbz", "c!(?)z")
		// extglobs.js line 24
		assertMatch(t, true, "cbz", "c!(@)z")
	})

	t.Run("should not convert capture groups to extglobs", func(t *testing.T) {
		// extglobs.js:28 — assert.strictEqual(makeRe('c!(?:foo)?z').source, '^(?:c!(?:foo)?z)$');
		re := MakeRe("c!(?:foo)?z", nil)
		expectedSource := "^(?:c!(?:foo)?z)$"
		actualSource := re.re.String()
		if actualSource != expectedSource {
			t.Errorf("MakeRe(%q).re.String(): expected %q, got %q",
				"c!(?:foo)?z", expectedSource, actualSource)
		}
		// extglobs.js:29
		assertMatch(t, false, "c/z", "c!(?:foo)?z")
		// extglobs.js line 30
		assertMatch(t, true, "c!fooz", "c!(?:foo)?z")
		// extglobs.js line 31
		assertMatch(t, true, "c!z", "c!(?:foo)?z")
	})

	t.Run("negation", func(t *testing.T) {
		t.Run("should support negation extglobs as the entire pattern", func(t *testing.T) {
			// extglobs.js line 36
			assertMatch(t, false, "abc", "!(abc)")
			// extglobs.js line 37
			assertMatch(t, false, "a", "!(a)")
			// extglobs.js line 38
			assertMatch(t, true, "aa", "!(a)")
			// extglobs.js line 39
			assertMatch(t, true, "b", "!(a)")
		})

		t.Run("should support negation extglobs as part of a pattern", func(t *testing.T) {
			// extglobs.js line 43
			assertMatch(t, true, "aac", "a!(b)c")
			// extglobs.js line 44
			assertMatch(t, false, "abc", "a!(b)c")
			// extglobs.js line 45
			assertMatch(t, true, "acc", "a!(b)c")
			// extglobs.js line 46
			assertMatch(t, true, "abz", "a!(z)")
			// extglobs.js line 47
			assertMatch(t, false, "az", "a!(z)")
		})

		t.Run("should support excluding dots with negation extglobs", func(t *testing.T) {
			// extglobs.js line 51
			assertMatch(t, false, "a.", "a!(.)")
			// extglobs.js line 52
			assertMatch(t, false, ".a", "!(.)a")
			// extglobs.js line 53
			assertMatch(t, false, "a.c", "a!(.)c")
			// extglobs.js line 54
			assertMatch(t, true, "abc", "a!(.)c")
		})

		// See https://github.com/micromatch/picomatch/issues/83
		t.Run("should support stars in negation extglobs", func(t *testing.T) {
			// extglobs.js line 59
			assertMatch(t, false, "/file.d.ts", "/!(*.d).ts")
			// extglobs.js line 60
			assertMatch(t, true, "/file.ts", "/!(*.d).ts")
			// extglobs.js line 61
			assertMatch(t, true, "/file.something.ts", "/!(*.d).ts")
			// extglobs.js line 62
			assertMatch(t, true, "/file.d.something.ts", "/!(*.d).ts")
			// extglobs.js line 63
			assertMatch(t, true, "/file.dhello.ts", "/!(*.d).ts")

			// extglobs.js line 65
			assertMatch(t, false, "/file.d.ts", "**/!(*.d).ts")
			// extglobs.js line 66
			assertMatch(t, true, "/file.ts", "**/!(*.d).ts")
			// extglobs.js line 67
			assertMatch(t, true, "/file.something.ts", "**/!(*.d).ts")
			// extglobs.js line 68
			assertMatch(t, true, "/file.d.something.ts", "**/!(*.d).ts")
			// extglobs.js line 69
			assertMatch(t, true, "/file.dhello.ts", "**/!(*.d).ts")
		})

		// See https://github.com/micromatch/picomatch/issues/93
		t.Run("should support stars in negation extglobs with expression after closing parenthesis", func(t *testing.T) {
			// Nested expression after closing parenthesis
			// extglobs.js line 75
			assertMatch(t, false, "/file.d.ts", "/!(*.d).{ts,tsx}")
			// extglobs.js line 76
			assertMatch(t, true, "/file.ts", "/!(*.d).{ts,tsx}")
			// extglobs.js line 77
			assertMatch(t, true, "/file.something.ts", "/!(*.d).{ts,tsx}")
			// extglobs.js line 78
			assertMatch(t, true, "/file.d.something.ts", "/!(*.d).{ts,tsx}")
			// extglobs.js line 79
			assertMatch(t, true, "/file.dhello.ts", "/!(*.d).{ts,tsx}")

			// Extglob after closing parenthesis
			// extglobs.js line 82
			assertMatch(t, false, "/file.d.ts", "/!(*.d).@(ts)")
			// extglobs.js line 83
			assertMatch(t, true, "/file.ts", "/!(*.d).@(ts)")
			// extglobs.js line 84
			assertMatch(t, true, "/file.something.ts", "/!(*.d).@(ts)")
			// extglobs.js line 85
			assertMatch(t, true, "/file.d.something.ts", "/!(*.d).@(ts)")
			// extglobs.js line 86
			assertMatch(t, true, "/file.dhello.ts", "/!(*.d).@(ts)")
		})

		t.Run("should support negation extglobs in patterns with slashes", func(t *testing.T) {
			// extglobs.js line 90
			assertMatch(t, false, "foo/abc", "foo/!(abc)")
			// extglobs.js line 91
			assertMatch(t, true, "foo/bar", "foo/!(abc)")

			// extglobs.js line 93
			assertMatch(t, false, "a/z", "a/!(z)")
			// extglobs.js line 94
			assertMatch(t, true, "a/b", "a/!(z)")

			// extglobs.js line 96
			assertMatch(t, false, "c/z/v", "c/!(z)/v")
			// extglobs.js line 97
			assertMatch(t, true, "c/a/v", "c/!(z)/v")

			// extglobs.js line 99
			assertMatch(t, true, "a/a", "!(b/a)")
			// extglobs.js line 100
			assertMatch(t, false, "b/a", "!(b/a)")

			// extglobs.js line 102
			assertMatch(t, false, "foo/bar", "!(!(foo))*")
			// extglobs.js line 103
			assertMatch(t, true, "a/a", "!(b/a)")
			// extglobs.js line 104
			assertMatch(t, false, "b/a", "!(b/a)")

			// extglobs.js line 106
			assertMatch(t, true, "a/a", "(!(b/a))")
			// extglobs.js line 107
			assertMatch(t, true, "a/a", "!((b/a))")
			// extglobs.js line 108
			assertMatch(t, false, "b/a", "!((b/a))")

			// extglobs.js line 110
			assertMatch(t, false, "a/a", "(!(?:b/a))")
			// extglobs.js line 111
			assertMatch(t, false, "b/a", "!((?:b/a))")

			// extglobs.js line 113
			assertMatch(t, true, "a/a", "!(b/(a))")
			// extglobs.js line 114
			assertMatch(t, false, "b/a", "!(b/(a))")

			// extglobs.js line 116
			assertMatch(t, true, "a/a", "!(b/a)")
			// extglobs.js line 117
			assertMatch(t, false, "b/a", "!(b/a)")
		})

		t.Run("should not match slashes with extglobs that do not have slashes", func(t *testing.T) {
			// extglobs.js line 121
			assertMatch(t, false, "c/z", "c!(z)")
			// extglobs.js line 122
			assertMatch(t, false, "c/z", "c!(z)z")
			// extglobs.js line 123
			assertMatch(t, false, "c/z", "c!(.)z")
			// extglobs.js line 124
			assertMatch(t, false, "c/z", "c!(*)z")
			// extglobs.js line 125
			assertMatch(t, false, "c/z", "c!(+)z")
			// extglobs.js line 126
			assertMatch(t, false, "c/z", "c!(?)z")
			// extglobs.js line 127
			assertMatch(t, false, "c/z", "c!(@)z")
		})

		t.Run("should support matching slashes with extglobs that have slashes", func(t *testing.T) {
			// extglobs.js line 131
			assertMatch(t, false, "c/z", "a!(z)")
			// extglobs.js line 132
			assertMatch(t, false, "c/z", "c!(.)z")
			// extglobs.js line 133
			assertMatch(t, false, "c/z", "c!(/)z")
			// extglobs.js line 134
			assertMatch(t, false, "c/z", "c!(/z)z")
			// extglobs.js line 135
			assertMatch(t, false, "c/b", "c!(/z)z")
			// extglobs.js line 136
			assertMatch(t, true, "c/b/z", "c!(/z)z")
		})

		t.Run("should support negation extglobs following !", func(t *testing.T) {
			// extglobs.js line 140
			assertMatch(t, true, "abc", "!!(abc)")
			// extglobs.js line 141
			assertMatch(t, false, "abc", "!!!(abc)")
			// extglobs.js line 142
			assertMatch(t, true, "abc", "!!!!(abc)")
			// extglobs.js line 143
			assertMatch(t, false, "abc", "!!!!!(abc)")
			// extglobs.js line 144
			assertMatch(t, true, "abc", "!!!!!!(abc)")
			// extglobs.js line 145
			assertMatch(t, false, "abc", "!!!!!!!(abc)")
			// extglobs.js line 146
			assertMatch(t, true, "abc", "!!!!!!!!(abc)")
		})

		t.Run("should support nested negation extglobs", func(t *testing.T) {
			// extglobs.js line 150
			assertMatch(t, true, "abc", "!(!(abc))")
			// extglobs.js line 151
			assertMatch(t, false, "abc", "!(!(!(abc)))")
			// extglobs.js line 152
			assertMatch(t, true, "abc", "!(!(!(!(abc))))")
			// extglobs.js line 153
			assertMatch(t, false, "abc", "!(!(!(!(!(abc)))))")
			// extglobs.js line 154
			assertMatch(t, true, "abc", "!(!(!(!(!(!(abc))))))")
			// extglobs.js line 155
			assertMatch(t, false, "abc", "!(!(!(!(!(!(!(abc)))))))")
			// extglobs.js line 156
			assertMatch(t, true, "abc", "!(!(!(!(!(!(!(!(abc))))))))")

			// extglobs.js line 158
			assertMatch(t, true, "foo/abc", "foo/!(!(abc))")
			// extglobs.js line 159
			assertMatch(t, false, "foo/abc", "foo/!(!(!(abc)))")
			// extglobs.js line 160
			assertMatch(t, true, "foo/abc", "foo/!(!(!(!(abc))))")
			// extglobs.js line 161
			assertMatch(t, false, "foo/abc", "foo/!(!(!(!(!(abc)))))")
			// extglobs.js line 162
			assertMatch(t, true, "foo/abc", "foo/!(!(!(!(!(!(abc))))))")
			// extglobs.js line 163
			assertMatch(t, false, "foo/abc", "foo/!(!(!(!(!(!(!(abc)))))))")
			// extglobs.js line 164
			assertMatch(t, true, "foo/abc", "foo/!(!(!(!(!(!(!(!(abc))))))))")
		})

		t.Run("should support multiple !(...) extglobs in a pattern", func(t *testing.T) {
			// extglobs.js line 168
			assertMatch(t, false, "moo.cow", "!(moo).!(cow)")
			// extglobs.js line 169
			assertMatch(t, false, "foo.cow", "!(moo).!(cow)")
			// extglobs.js line 170
			assertMatch(t, false, "moo.bar", "!(moo).!(cow)")
			// extglobs.js line 171
			assertMatch(t, true, "foo.bar", "!(moo).!(cow)")

			// extglobs.js line 173
			assertMatch(t, false, "a   ", "@(!(a) )*")
			// extglobs.js line 174
			assertMatch(t, false, "a   b", "@(!(a) )*")
			// extglobs.js line 175
			assertMatch(t, false, "a  b", "@(!(a) )*")
			// extglobs.js line 176
			assertMatch(t, false, "a  ", "@(!(a) )*")
			// extglobs.js line 177
			assertMatch(t, false, "a ", "@(!(a) )*")
			// extglobs.js line 178
			assertMatch(t, false, "a", "@(!(a) )*")
			// extglobs.js line 179
			assertMatch(t, false, "aa", "@(!(a) )*")
			// extglobs.js line 180
			assertMatch(t, false, "b", "@(!(a) )*")
			// extglobs.js line 181
			assertMatch(t, false, "bb", "@(!(a) )*")
			// extglobs.js line 182
			assertMatch(t, true, " a ", "@(!(a) )*")
			// extglobs.js line 183
			assertMatch(t, true, "b  ", "@(!(a) )*")
			// extglobs.js line 184
			assertMatch(t, true, "b ", "@(!(a) )*")

			// extglobs.js line 186
			assertMatch(t, false, "c/z", "a*!(z)")
			// extglobs.js line 187
			assertMatch(t, true, "abz", "a*!(z)")
			// extglobs.js line 188
			assertMatch(t, true, "az", "a*!(z)")

			// extglobs.js line 190
			assertMatch(t, false, "a", "!(a*)")
			// extglobs.js line 191
			assertMatch(t, false, "aa", "!(a*)")
			// extglobs.js line 192
			assertMatch(t, false, "ab", "!(a*)")
			// extglobs.js line 193
			assertMatch(t, true, "b", "!(a*)")

			// extglobs.js line 195
			assertMatch(t, false, "a", "!(*a*)")
			// extglobs.js line 196
			assertMatch(t, false, "aa", "!(*a*)")
			// extglobs.js line 197
			assertMatch(t, false, "ab", "!(*a*)")
			// extglobs.js line 198
			assertMatch(t, false, "ac", "!(*a*)")
			// extglobs.js line 199
			assertMatch(t, true, "b", "!(*a*)")

			// extglobs.js line 201
			assertMatch(t, false, "a", "!(*a)")
			// extglobs.js line 202
			assertMatch(t, false, "aa", "!(*a)")
			// extglobs.js line 203
			assertMatch(t, false, "bba", "!(*a)")
			// extglobs.js line 204
			assertMatch(t, true, "ab", "!(*a)")
			// extglobs.js line 205
			assertMatch(t, true, "ac", "!(*a)")
			// extglobs.js line 206
			assertMatch(t, true, "b", "!(*a)")

			// extglobs.js line 208
			assertMatch(t, false, "a", "!(*a)*")
			// extglobs.js line 209
			assertMatch(t, false, "aa", "!(*a)*")
			// extglobs.js line 210
			assertMatch(t, false, "bba", "!(*a)*")
			// extglobs.js line 211
			assertMatch(t, false, "ab", "!(*a)*")
			// extglobs.js line 212
			assertMatch(t, false, "ac", "!(*a)*")
			// extglobs.js line 213
			assertMatch(t, true, "b", "!(*a)*")

			// extglobs.js line 215
			assertMatch(t, false, "a", "!(a)*")
			// extglobs.js line 216
			assertMatch(t, false, "abb", "!(a)*")
			// extglobs.js line 217
			assertMatch(t, true, "ba", "!(a)*")

			// extglobs.js line 219
			assertMatch(t, true, "aa", "a!(b)*")
			// extglobs.js line 220
			assertMatch(t, false, "ab", "a!(b)*")
			// extglobs.js line 221
			assertMatch(t, false, "aba", "a!(b)*")
			// extglobs.js line 222
			assertMatch(t, true, "ac", "a!(b)*")
		})

		t.Run("should multiple nested negation extglobs", func(t *testing.T) {
			// extglobs.js line 226
			assertMatch(t, true, "moo.cow", "!(!(moo)).!(!(cow))")
		})

		t.Run("should support logical-or inside negation !(...) extglobs", func(t *testing.T) {
			// extglobs.js line 230
			assertMatch(t, false, "ac", "!(a|b)c")
			// extglobs.js line 231
			assertMatch(t, false, "bc", "!(a|b)c")
			// extglobs.js line 232
			assertMatch(t, true, "cc", "!(a|b)c")
		})

		t.Run("should support multiple logical-ors negation extglobs", func(t *testing.T) {
			// extglobs.js line 236
			assertMatch(t, false, "ac.d", "!(a|b)c.!(d|e)")
			// extglobs.js line 237
			assertMatch(t, false, "bc.d", "!(a|b)c.!(d|e)")
			// extglobs.js line 238
			assertMatch(t, false, "cc.d", "!(a|b)c.!(d|e)")
			// extglobs.js line 239
			assertMatch(t, false, "ac.e", "!(a|b)c.!(d|e)")
			// extglobs.js line 240
			assertMatch(t, false, "bc.e", "!(a|b)c.!(d|e)")
			// extglobs.js line 241
			assertMatch(t, false, "cc.e", "!(a|b)c.!(d|e)")
			// extglobs.js line 242
			assertMatch(t, false, "ac.f", "!(a|b)c.!(d|e)")
			// extglobs.js line 243
			assertMatch(t, false, "bc.f", "!(a|b)c.!(d|e)")
			// extglobs.js line 244
			assertMatch(t, true, "cc.f", "!(a|b)c.!(d|e)")
			// extglobs.js line 245
			assertMatch(t, true, "dc.g", "!(a|b)c.!(d|e)")
		})

		t.Run("should support nested logical-ors inside negation extglobs", func(t *testing.T) {
			// extglobs.js line 249
			assertMatch(t, true, "ac.d", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 250
			assertMatch(t, true, "bc.d", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 251
			assertMatch(t, false, "cc.d", "!(a|b)c.!(d|e)")
			// extglobs.js line 252
			assertMatch(t, true, "cc.d", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 253
			assertMatch(t, true, "cc.d", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 254
			assertMatch(t, true, "ac.e", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 255
			assertMatch(t, true, "bc.e", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 256
			assertMatch(t, true, "cc.e", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 257
			assertMatch(t, true, "ac.f", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 258
			assertMatch(t, true, "bc.f", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 259
			assertMatch(t, false, "cc.f", "!(!(a|b)c.!(d|e))")
			// extglobs.js line 260
			assertMatch(t, false, "dc.g", "!(!(a|b)c.!(d|e))")
		})
	})

	t.Run("file extensions", func(t *testing.T) {
		t.Run("should support matching file extensions with @(...)", func(t *testing.T) {
			// extglobs.js line 266
			assertMatch(t, false, ".md", "@(a|b).md")
			// extglobs.js line 267
			assertMatch(t, false, "a.js", "@(a|b).md")
			// extglobs.js line 268
			assertMatch(t, false, "c.md", "@(a|b).md")
			// extglobs.js line 269
			assertMatch(t, true, "a.md", "@(a|b).md")
			// extglobs.js line 270
			assertMatch(t, true, "b.md", "@(a|b).md")
		})

		t.Run("should support matching file extensions with +(...)", func(t *testing.T) {
			// extglobs.js line 274
			assertMatch(t, false, ".md", "+(a|b).md")
			// extglobs.js line 275
			assertMatch(t, false, "a.js", "+(a|b).md")
			// extglobs.js line 276
			assertMatch(t, false, "c.md", "+(a|b).md")
			// extglobs.js line 277
			assertMatch(t, true, "a.md", "+(a|b).md")
			// extglobs.js line 278
			assertMatch(t, true, "aa.md", "+(a|b).md")
			// extglobs.js line 279
			assertMatch(t, true, "ab.md", "+(a|b).md")
			// extglobs.js line 280
			assertMatch(t, true, "b.md", "+(a|b).md")
			// extglobs.js line 281
			assertMatch(t, true, "bb.md", "+(a|b).md")
		})

		t.Run("should support matching file extensions with *(...)", func(t *testing.T) {
			// extglobs.js line 285
			assertMatch(t, false, "a.js", "*(a|b).md")
			// extglobs.js line 286
			assertMatch(t, false, "c.md", "*(a|b).md")
			// extglobs.js line 287
			assertMatch(t, true, ".md", "*(a|b).md")
			// extglobs.js line 288
			assertMatch(t, true, "a.md", "*(a|b).md")
			// extglobs.js line 289
			assertMatch(t, true, "aa.md", "*(a|b).md")
			// extglobs.js line 290
			assertMatch(t, true, "ab.md", "*(a|b).md")
			// extglobs.js line 291
			assertMatch(t, true, "b.md", "*(a|b).md")
			// extglobs.js line 292
			assertMatch(t, true, "bb.md", "*(a|b).md")
		})

		t.Run("should support matching file extensions with ?(...)", func(t *testing.T) {
			// extglobs.js line 296
			assertMatch(t, false, "a.js", "?(a|b).md")
			// extglobs.js line 297
			assertMatch(t, false, "bb.md", "?(a|b).md")
			// extglobs.js line 298
			assertMatch(t, false, "c.md", "?(a|b).md")
			// extglobs.js line 299
			assertMatch(t, true, ".md", "?(a|b).md")
			// extglobs.js line 300
			assertMatch(t, true, "a.md", "?(a|ab|b).md")
			// extglobs.js line 301
			assertMatch(t, true, "a.md", "?(a|b).md")
			// extglobs.js line 302
			assertMatch(t, true, "aa.md", "?(a|aa|b).md")
			// extglobs.js line 303
			assertMatch(t, true, "ab.md", "?(a|ab|b).md")
			// extglobs.js line 304
			assertMatch(t, true, "b.md", "?(a|ab|b).md")

			// See https://github.com/micromatch/micromatch/issues/186
			// extglobs.js line 307
			assertMatch(t, true, "ab", "+(a)?(b)")
			// extglobs.js line 308
			assertMatch(t, true, "aab", "+(a)?(b)")
			// extglobs.js line 309
			assertMatch(t, true, "aa", "+(a)?(b)")
			// extglobs.js line 310
			assertMatch(t, true, "a", "+(a)?(b)")
		})
	})

	t.Run("statechar", func(t *testing.T) {
		t.Run("should support ?(...) extglobs ending with statechar", func(t *testing.T) {
			// extglobs.js line 316
			assertMatch(t, false, "ax", "a?(b*)")
			// extglobs.js line 317
			assertMatch(t, true, "ax", "?(a*|b)")
		})

		t.Run("should support *(...) extglobs ending with statechar", func(t *testing.T) {
			// extglobs.js line 321
			assertMatch(t, false, "ax", "a*(b*)")
			// extglobs.js line 322
			assertMatch(t, true, "ax", "*(a*|b)")
		})

		t.Run("should support @(...) extglobs ending with statechar", func(t *testing.T) {
			// extglobs.js line 326
			assertMatch(t, false, "ax", "a@(b*)")
			// extglobs.js line 327
			assertMatch(t, true, "ax", "@(a*|b)")
		})

		t.Run("should support ?(...) extglobs ending with statechar (duplicate)", func(t *testing.T) {
			// extglobs.js line 331 (duplicate section in original JS)
			assertMatch(t, false, "ax", "a?(b*)")
			// extglobs.js line 332
			assertMatch(t, true, "ax", "?(a*|b)")
		})

		t.Run("should support !(...) extglobs ending with statechar", func(t *testing.T) {
			// extglobs.js line 336
			assertMatch(t, true, "ax", "a!(b*)")
			// extglobs.js line 337
			assertMatch(t, false, "ax", "!(a*|b)")
		})
	})

	t.Run("should match nested directories with negation extglobs", func(t *testing.T) {
		// extglobs.js line 342
		assertMatch(t, true, "a", "!(a/**)")
		// extglobs.js line 343
		assertMatch(t, false, "a/", "!(a/**)")
		// extglobs.js line 344
		assertMatch(t, false, "a/b", "!(a/**)")
		// extglobs.js line 345
		assertMatch(t, false, "a/b/c", "!(a/**)")
		// extglobs.js line 346
		assertMatch(t, true, "b", "!(a/**)")
		// extglobs.js line 347
		assertMatch(t, true, "b/c", "!(a/**)")

		// extglobs.js line 349
		assertMatch(t, true, "a/a", "a/!(b*)")
		// extglobs.js line 350
		assertMatch(t, false, "a/b", "a/!(b*)")
		// extglobs.js line 351
		assertMatch(t, false, "a/b/c", "a/!(b/*)")
		// extglobs.js line 352
		assertMatch(t, false, "a/b/c", "a/!(b*)")
		// extglobs.js line 353
		assertMatch(t, true, "a/c", "a/!(b*)")

		// extglobs.js line 355
		assertMatch(t, true, "a/a/", "a/!(b*)/**")
		// extglobs.js line 356
		assertMatch(t, true, "a/a", "a/!(b*)")
		// extglobs.js line 357
		assertMatch(t, true, "a/a", "a/!(b*)/**")
		// extglobs.js line 358
		assertMatch(t, false, "a/b", "a/!(b*)/**")
		// extglobs.js line 359
		assertMatch(t, false, "a/b/c", "a/!(b*)/**")
		// extglobs.js line 360
		assertMatch(t, true, "a/c", "a/!(b*)/**")
		// extglobs.js line 361
		assertMatch(t, true, "a/c", "a/!(b*)")
		// extglobs.js line 362
		assertMatch(t, true, "a/c/", "a/!(b*)/**")
	})

	t.Run("should support *(...)", func(t *testing.T) {
		// extglobs.js line 366
		assertMatch(t, true, "a", "a*(z)")
		// extglobs.js line 367
		assertMatch(t, true, "az", "a*(z)")
		// extglobs.js line 368
		assertMatch(t, true, "azz", "a*(z)")
		// extglobs.js line 369
		assertMatch(t, true, "azzz", "a*(z)")
		// extglobs.js line 370
		assertMatch(t, false, "abz", "a*(z)")
		// extglobs.js line 371
		assertMatch(t, false, "cz", "a*(z)")

		// extglobs.js line 373
		assertMatch(t, false, "a/a", "*(b/a)")
		// extglobs.js line 374
		assertMatch(t, false, "a/b", "*(b/a)")
		// extglobs.js line 375
		assertMatch(t, false, "a/c", "*(b/a)")
		// extglobs.js line 376
		assertMatch(t, true, "b/a", "*(b/a)")
		// extglobs.js line 377
		assertMatch(t, false, "b/b", "*(b/a)")
		// extglobs.js line 378
		assertMatch(t, false, "b/c", "*(b/a)")

		// extglobs.js line 380
		assertMatch(t, false, "cz", "a**(z)")
		// extglobs.js line 381
		assertMatch(t, true, "abz", "a**(z)")
		// extglobs.js line 382
		assertMatch(t, true, "az", "a**(z)")

		// extglobs.js line 384
		assertMatch(t, false, "c/z/v", "*(z)")
		// extglobs.js line 385
		assertMatch(t, true, "z", "*(z)")
		// extglobs.js line 386
		assertMatch(t, false, "zf", "*(z)")
		// extglobs.js line 387
		assertMatch(t, false, "fz", "*(z)")

		// extglobs.js line 389
		assertMatch(t, false, "c/a/v", "c/*(z)/v")
		// extglobs.js line 390
		assertMatch(t, true, "c/z/v", "c/*(z)/v")

		// extglobs.js line 392
		assertMatch(t, false, "a.md.js", "*.*(js).js")
		// extglobs.js line 393
		assertMatch(t, true, "a.js.js", "*.*(js).js")
	})

	t.Run("should support +(...) extglobs", func(t *testing.T) {
		// extglobs.js line 397
		assertMatch(t, false, "a", "a+(z)")
		// extglobs.js line 398
		assertMatch(t, true, "az", "a+(z)")
		// extglobs.js line 399
		assertMatch(t, false, "cz", "a+(z)")
		// extglobs.js line 400
		assertMatch(t, false, "abz", "a+(z)")
		// extglobs.js line 401
		assertMatch(t, false, "a+z", "a+(z)")
		// extglobs.js line 402
		assertMatch(t, true, "a+z", "a++(z)")
		// extglobs.js line 403
		assertMatch(t, false, "c+z", "a+(z)")
		// extglobs.js line 404
		assertMatch(t, false, "a+bz", "a+(z)")
		// extglobs.js line 405
		assertMatch(t, false, "az", "+(z)")
		// extglobs.js line 406
		assertMatch(t, false, "cz", "+(z)")
		// extglobs.js line 407
		assertMatch(t, false, "abz", "+(z)")
		// extglobs.js line 408
		assertMatch(t, false, "fz", "+(z)")
		// extglobs.js line 409
		assertMatch(t, true, "z", "+(z)")
		// extglobs.js line 410
		assertMatch(t, true, "zz", "+(z)")
		// extglobs.js line 411
		assertMatch(t, true, "c/z/v", "c/+(z)/v")
		// extglobs.js line 412
		assertMatch(t, true, "c/zz/v", "c/+(z)/v")
		// extglobs.js line 413
		assertMatch(t, false, "c/a/v", "c/+(z)/v")
	})

	t.Run("should support ?(...) extglobs", func(t *testing.T) {
		// extglobs.js line 417
		assertMatch(t, true, "a?z", "a??(z)")
		// extglobs.js line 418
		assertMatch(t, true, "a.z", "a??(z)")
		// extglobs.js line 419
		assertMatch(t, false, "a/z", "a??(z)")
		// extglobs.js line 420
		assertMatch(t, true, "a?", "a??(z)")
		// extglobs.js line 421
		assertMatch(t, true, "ab", "a??(z)")
		// extglobs.js line 422
		assertMatch(t, false, "a/", "a??(z)")

		// extglobs.js line 424
		assertMatch(t, false, "a?z", "a?(z)")
		// extglobs.js line 425
		assertMatch(t, false, "abz", "a?(z)")
		// extglobs.js line 426
		assertMatch(t, false, "z", "a?(z)")
		// extglobs.js line 427
		assertMatch(t, true, "a", "a?(z)")
		// extglobs.js line 428
		assertMatch(t, true, "az", "a?(z)")

		// extglobs.js line 430
		assertMatch(t, false, "abz", "?(z)")
		// extglobs.js line 431
		assertMatch(t, false, "az", "?(z)")
		// extglobs.js line 432
		assertMatch(t, false, "cz", "?(z)")
		// extglobs.js line 433
		assertMatch(t, false, "fz", "?(z)")
		// extglobs.js line 434
		assertMatch(t, false, "zz", "?(z)")
		// extglobs.js line 435
		assertMatch(t, true, "z", "?(z)")

		// extglobs.js line 437
		assertMatch(t, false, "c/a/v", "c/?(z)/v")
		// extglobs.js line 438
		assertMatch(t, false, "c/zz/v", "c/?(z)/v")
		// extglobs.js line 439
		assertMatch(t, true, "c/z/v", "c/?(z)/v")
	})

	t.Run("should support @(...) extglobs", func(t *testing.T) {
		// extglobs.js line 443
		assertMatch(t, true, "c/z/v", "c/@(z)/v")
		// extglobs.js line 444
		assertMatch(t, false, "c/a/v", "c/@(z)/v")
		// extglobs.js line 445
		assertMatch(t, true, "moo.cow", "@(*.*)")

		// extglobs.js line 447
		assertMatch(t, false, "cz", "a*@(z)")
		// extglobs.js line 448
		assertMatch(t, true, "abz", "a*@(z)")
		// extglobs.js line 449
		assertMatch(t, true, "az", "a*@(z)")

		// extglobs.js line 451
		assertMatch(t, false, "cz", "a@(z)")
		// extglobs.js line 452
		assertMatch(t, false, "abz", "a@(z)")
		// extglobs.js line 453
		assertMatch(t, true, "az", "a@(z)")
	})

	t.Run("should match exactly one of the given pattern", func(t *testing.T) {
		// extglobs.js line 457
		assertMatch(t, false, "aa.aa", "(b|a).(a)")
		// extglobs.js line 458
		assertMatch(t, false, "a.bb", "(b|a).(a)")
		// extglobs.js line 459
		assertMatch(t, false, "a.aa.a", "(b|a).(a)")
		// extglobs.js line 460
		assertMatch(t, false, "cc.a", "(b|a).(a)")
		// extglobs.js line 461
		assertMatch(t, true, "a.a", "(b|a).(a)")
		// extglobs.js line 462
		assertMatch(t, false, "c.a", "(b|a).(a)")
		// extglobs.js line 463
		assertMatch(t, false, "dd.aa.d", "(b|a).(a)")
		// extglobs.js line 464
		assertMatch(t, true, "b.a", "(b|a).(a)")

		// extglobs.js line 466
		assertMatch(t, false, "aa.aa", "@(b|a).@(a)")
		// extglobs.js line 467
		assertMatch(t, false, "a.bb", "@(b|a).@(a)")
		// extglobs.js line 468
		assertMatch(t, false, "a.aa.a", "@(b|a).@(a)")
		// extglobs.js line 469
		assertMatch(t, false, "cc.a", "@(b|a).@(a)")
		// extglobs.js line 470
		assertMatch(t, true, "a.a", "@(b|a).@(a)")
		// extglobs.js line 471
		assertMatch(t, false, "c.a", "@(b|a).@(a)")
		// extglobs.js line 472
		assertMatch(t, false, "dd.aa.d", "@(b|a).@(a)")
		// extglobs.js line 473
		assertMatch(t, true, "b.a", "@(b|a).@(a)")
	})

	t.Run("should pass tests from rosenblatt's korn shell book", func(t *testing.T) {
		// This one is the only difference, since picomatch does not match empty strings.
		// extglobs.js line 478
		assertMatch(t, false, "", "*(0|1|3|5|7|9)")

		// extglobs.js line 480
		assertMatch(t, true, "137577991", "*(0|1|3|5|7|9)")
		// extglobs.js line 481
		assertMatch(t, false, "2468", "*(0|1|3|5|7|9)")

		// extglobs.js line 483
		assertMatch(t, true, "file.c", "*.c?(c)")
		// extglobs.js line 484
		assertMatch(t, false, "file.C", "*.c?(c)")
		// extglobs.js line 485
		assertMatch(t, true, "file.cc", "*.c?(c)")
		// extglobs.js line 486
		assertMatch(t, false, "file.ccc", "*.c?(c)")

		// extglobs.js line 488
		assertMatch(t, true, "parse.y", "!(*.c|*.h|Makefile.in|config*|README)")
		// extglobs.js line 489
		assertMatch(t, false, "shell.c", "!(*.c|*.h|Makefile.in|config*|README)")
		// extglobs.js line 490
		assertMatch(t, true, "Makefile", "!(*.c|*.h|Makefile.in|config*|README)")
		// extglobs.js line 491
		assertMatch(t, false, "Makefile.in", "!(*.c|*.h|Makefile.in|config*|README)")

		// extglobs.js line 493
		assertMatch(t, false, "VMS.FILE;", `*\;[1-9]*([0-9])`)
		// extglobs.js line 494
		assertMatch(t, false, "VMS.FILE;0", `*\;[1-9]*([0-9])`)
		// extglobs.js line 495
		assertMatch(t, true, "VMS.FILE;1", `*\;[1-9]*([0-9])`)
		// extglobs.js line 496
		assertMatch(t, true, "VMS.FILE;139", `*\;[1-9]*([0-9])`)
		// extglobs.js line 497
		assertMatch(t, false, "VMS.FILE;1N", `*\;[1-9]*([0-9])`)
	})

	t.Run("tests derived from the pd-ksh test suite", func(t *testing.T) {
		// extglobs.js line 501
		assertMatch(t, true, "abcx", "!([*)*")
		// extglobs.js line 502
		assertMatch(t, true, "abcz", "!([*)*")
		// extglobs.js line 503
		assertMatch(t, true, "bbc", "!([*)*")

		// extglobs.js line 505
		assertMatch(t, true, "abcx", "!([[*])*")
		// extglobs.js line 506
		assertMatch(t, true, "abcz", "!([[*])*")
		// extglobs.js line 507
		assertMatch(t, true, "bbc", "!([[*])*")

		// extglobs.js line 509
		assertMatch(t, true, "abcx", `+(a|b\[)*`)
		// extglobs.js line 510
		assertMatch(t, true, "abcz", `+(a|b\[)*`)
		// extglobs.js line 511
		assertMatch(t, false, "bbc", `+(a|b\[)*`)

		// extglobs.js line 513
		assertMatch(t, true, "abcx", "+(a|b[)*")
		// extglobs.js line 514
		assertMatch(t, true, "abcz", "+(a|b[)*")
		// extglobs.js line 515
		assertMatch(t, false, "bbc", "+(a|b[)*")

		// extglobs.js line 517
		assertMatch(t, false, "abcx", "[a*(]*z")
		// extglobs.js line 518
		assertMatch(t, true, "abcz", "[a*(]*z")
		// extglobs.js line 519
		assertMatch(t, false, "bbc", "[a*(]*z")
		// extglobs.js line 520
		assertMatch(t, true, "aaz", "[a*(]*z")
		// extglobs.js line 521
		assertMatch(t, true, "aaaz", "[a*(]*z")

		// extglobs.js line 523
		assertMatch(t, false, "abcx", "[a*(]*)z")
		// extglobs.js line 524
		assertMatch(t, false, "abcz", "[a*(]*)z")
		// extglobs.js line 525
		assertMatch(t, false, "bbc", "[a*(]*)z")

		// extglobs.js line 527
		assertMatch(t, false, "abc", "+()c")
		// extglobs.js line 528
		assertMatch(t, false, "abc", "+()x")
		// extglobs.js line 529
		assertMatch(t, true, "abc", "+(*)c")
		// extglobs.js line 530
		assertMatch(t, false, "abc", "+(*)x")
		// extglobs.js line 531
		assertMatch(t, false, "abc", "no-file+(a|b)stuff")
		// extglobs.js line 532
		assertMatch(t, false, "abc", "no-file+(a*(c)|b)stuff")

		// extglobs.js line 534
		assertMatch(t, true, "abd", "a+(b|c)d")
		// extglobs.js line 535
		assertMatch(t, true, "acd", "a+(b|c)d")

		// extglobs.js line 537
		assertMatch(t, false, "abc", "a+(b|c)d")

		// extglobs.js line 539
		assertMatch(t, true, "abd", "a!(b|B)")
		// extglobs.js line 540
		assertMatch(t, true, "acd", "a!(@(b|B))")
		// extglobs.js line 541
		assertMatch(t, true, "ac", "a!(@(b|B))")
		// extglobs.js line 542
		assertMatch(t, false, "ab", "a!(@(b|B))")

		// extglobs.js line 544
		assertMatch(t, false, "abc", "a!(@(b|B))d")
		// extglobs.js line 545
		assertMatch(t, false, "abd", "a!(@(b|B))d")
		// extglobs.js line 546
		assertMatch(t, true, "acd", "a!(@(b|B))d")

		// extglobs.js line 548
		assertMatch(t, true, "abd", "a[b*(foo|bar)]d")
		// extglobs.js line 549
		assertMatch(t, false, "abc", "a[b*(foo|bar)]d")
		// extglobs.js line 550
		assertMatch(t, false, "acd", "a[b*(foo|bar)]d")
	})

	t.Run("stuff from korn's book", func(t *testing.T) {
		// extglobs.js line 554
		assertMatch(t, false, "para", "para+([0-9])")
		// extglobs.js line 555
		assertMatch(t, false, "para381", "para?([345]|99)1")
		// extglobs.js line 556
		assertMatch(t, false, "paragraph", "para*([0-9])")
		// extglobs.js line 557
		assertMatch(t, false, "paramour", "para@(chute|graph)")
		// extglobs.js line 558
		assertMatch(t, true, "para", "para*([0-9])")
		// extglobs.js line 559
		assertMatch(t, true, "para.38", "para!(*.[0-9])")
		// extglobs.js line 560
		assertMatch(t, true, "para.38", "para!(*.[00-09])")
		// extglobs.js line 561
		assertMatch(t, true, "para.graph", "para!(*.[0-9])")
		// extglobs.js line 562
		assertMatch(t, true, "para13829383746592", "para*([0-9])")
		// extglobs.js line 563
		assertMatch(t, true, "para39", "para!(*.[0-9])")
		// extglobs.js line 564
		assertMatch(t, true, "para987346523", "para+([0-9])")
		// extglobs.js line 565
		assertMatch(t, true, "para991", "para?([345]|99)1")
		// extglobs.js line 566
		assertMatch(t, true, "paragraph", "para!(*.[0-9])")
		// extglobs.js line 567
		assertMatch(t, true, "paragraph", "para@(chute|graph)")
	})

	t.Run("simple kleene star tests", func(t *testing.T) {
		// extglobs.js line 571
		assertMatch(t, false, "foo", "*(a|b[)")
		// extglobs.js line 572
		assertMatch(t, false, "(", "*(a|b[)")
		// extglobs.js line 573
		assertMatch(t, false, ")", "*(a|b[)")
		// extglobs.js line 574
		assertMatch(t, false, "|", "*(a|b[)")
		// extglobs.js line 575
		assertMatch(t, true, "a", "*(a|b)")
		// extglobs.js line 576
		assertMatch(t, true, "b", "*(a|b)")
		// extglobs.js line 577
		assertMatch(t, true, "b[", `*(a|b\[)`)
		// extglobs.js line 578
		assertMatch(t, true, "ab[", `+(a|b\[)`)
		// extglobs.js line 579
		assertMatch(t, false, "ab[cde", `+(a|b\[)`)
		// extglobs.js line 580
		assertMatch(t, true, "ab[cde", `+(a|b\[)*`)

		// extglobs.js line 582
		assertMatch(t, true, "foo", "*(a|b|f)*")
		// extglobs.js line 583
		assertMatch(t, true, "foo", "*(a|b|o)*")
		// extglobs.js line 584
		assertMatch(t, true, "foo", "*(a|b|f|o)")
		// extglobs.js line 585
		assertMatch(t, true, "*(a|b[)", `\*\(a\|b\[\)`)
		// extglobs.js line 586
		assertMatch(t, false, "foo", "*(a|b)")
		// extglobs.js line 587
		assertMatch(t, false, "foo", `*(a|b\[)`)
		// extglobs.js line 588
		assertMatch(t, true, "foo", `*(a|b\[)|f*`)
	})

	t.Run("should support multiple extglobs", func(t *testing.T) {
		// extglobs.js line 592
		assertMatch(t, true, "moo.cow", "@(*).@(*)")
		// extglobs.js line 593
		assertMatch(t, true, "a.a", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 594
		assertMatch(t, true, "a.b", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 595
		assertMatch(t, false, "a.c", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 596
		assertMatch(t, false, "a.c.d", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 597
		assertMatch(t, false, "c.c", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 598
		assertMatch(t, false, "a.", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 599
		assertMatch(t, false, "d.d", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 600
		assertMatch(t, false, "e.e", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 601
		assertMatch(t, false, "f.f", "*.@(a|b|@(ab|a*@(b))*@(c)d)")
		// extglobs.js line 602
		assertMatch(t, true, "a.abcd", "*.@(a|b|@(ab|a*@(b))*@(c)d)")

		// extglobs.js line 604
		assertMatch(t, false, "a.a", "!(*.a|*.b|*.c)")
		// extglobs.js line 605
		assertMatch(t, false, "a.b", "!(*.a|*.b|*.c)")
		// extglobs.js line 606
		assertMatch(t, false, "a.c", "!(*.a|*.b|*.c)")
		// extglobs.js line 607
		assertMatch(t, true, "a.c.d", "!(*.a|*.b|*.c)")
		// extglobs.js line 608
		assertMatch(t, false, "c.c", "!(*.a|*.b|*.c)")
		// extglobs.js line 609
		assertMatch(t, true, "a.", "!(*.a|*.b|*.c)")
		// extglobs.js line 610
		assertMatch(t, true, "d.d", "!(*.a|*.b|*.c)")
		// extglobs.js line 611
		assertMatch(t, true, "e.e", "!(*.a|*.b|*.c)")
		// extglobs.js line 612
		assertMatch(t, true, "f.f", "!(*.a|*.b|*.c)")
		// extglobs.js line 613
		assertMatch(t, true, "a.abcd", "!(*.a|*.b|*.c)")

		// extglobs.js line 615
		assertMatch(t, true, "a.a", "!(*.[^a-c])")
		// extglobs.js line 616
		assertMatch(t, true, "a.b", "!(*.[^a-c])")
		// extglobs.js line 617
		assertMatch(t, true, "a.c", "!(*.[^a-c])")
		// extglobs.js line 618
		assertMatch(t, false, "a.c.d", "!(*.[^a-c])")
		// extglobs.js line 619
		assertMatch(t, true, "c.c", "!(*.[^a-c])")
		// extglobs.js line 620
		assertMatch(t, true, "a.", "!(*.[^a-c])")
		// extglobs.js line 621
		assertMatch(t, false, "d.d", "!(*.[^a-c])")
		// extglobs.js line 622
		assertMatch(t, false, "e.e", "!(*.[^a-c])")
		// extglobs.js line 623
		assertMatch(t, false, "f.f", "!(*.[^a-c])")
		// extglobs.js line 624
		assertMatch(t, true, "a.abcd", "!(*.[^a-c])")

		// extglobs.js line 626
		assertMatch(t, false, "a.a", "!(*.[a-c])")
		// extglobs.js line 627
		assertMatch(t, false, "a.b", "!(*.[a-c])")
		// extglobs.js line 628
		assertMatch(t, false, "a.c", "!(*.[a-c])")
		// extglobs.js line 629
		assertMatch(t, true, "a.c.d", "!(*.[a-c])")
		// extglobs.js line 630
		assertMatch(t, false, "c.c", "!(*.[a-c])")
		// extglobs.js line 631
		assertMatch(t, true, "a.", "!(*.[a-c])")
		// extglobs.js line 632
		assertMatch(t, true, "d.d", "!(*.[a-c])")
		// extglobs.js line 633
		assertMatch(t, true, "e.e", "!(*.[a-c])")
		// extglobs.js line 634
		assertMatch(t, true, "f.f", "!(*.[a-c])")
		// extglobs.js line 635
		assertMatch(t, true, "a.abcd", "!(*.[a-c])")

		// extglobs.js line 637
		assertMatch(t, false, "a.a", "!(*.[a-c]*)")
		// extglobs.js line 638
		assertMatch(t, false, "a.b", "!(*.[a-c]*)")
		// extglobs.js line 639
		assertMatch(t, false, "a.c", "!(*.[a-c]*)")
		// extglobs.js line 640
		assertMatch(t, false, "a.c.d", "!(*.[a-c]*)")
		// extglobs.js line 641
		assertMatch(t, false, "c.c", "!(*.[a-c]*)")
		// extglobs.js line 642
		assertMatch(t, true, "a.", "!(*.[a-c]*)")
		// extglobs.js line 643
		assertMatch(t, true, "d.d", "!(*.[a-c]*)")
		// extglobs.js line 644
		assertMatch(t, true, "e.e", "!(*.[a-c]*)")
		// extglobs.js line 645
		assertMatch(t, true, "f.f", "!(*.[a-c]*)")
		// extglobs.js line 646
		assertMatch(t, false, "a.abcd", "!(*.[a-c]*)")

		// extglobs.js line 648
		assertMatch(t, false, "a.a", "*.!(a|b|c)")
		// extglobs.js line 649
		assertMatch(t, false, "a.b", "*.!(a|b|c)")
		// extglobs.js line 650
		assertMatch(t, false, "a.c", "*.!(a|b|c)")
		// extglobs.js line 651
		assertMatch(t, true, "a.c.d", "*.!(a|b|c)")
		// extglobs.js line 652
		assertMatch(t, false, "c.c", "*.!(a|b|c)")
		// extglobs.js line 653
		assertMatch(t, true, "a.", "*.!(a|b|c)")
		// extglobs.js line 654
		assertMatch(t, true, "d.d", "*.!(a|b|c)")
		// extglobs.js line 655
		assertMatch(t, true, "e.e", "*.!(a|b|c)")
		// extglobs.js line 656
		assertMatch(t, true, "f.f", "*.!(a|b|c)")
		// extglobs.js line 657
		assertMatch(t, true, "a.abcd", "*.!(a|b|c)")

		// extglobs.js line 659
		assertMatch(t, true, "a.a", "*!(.a|.b|.c)")
		// extglobs.js line 660
		assertMatch(t, true, "a.b", "*!(.a|.b|.c)")
		// extglobs.js line 661
		assertMatch(t, true, "a.c", "*!(.a|.b|.c)")
		// extglobs.js line 662
		assertMatch(t, true, "a.c.d", "*!(.a|.b|.c)")
		// extglobs.js line 663
		assertMatch(t, true, "c.c", "*!(.a|.b|.c)")
		// extglobs.js line 664
		assertMatch(t, true, "a.", "*!(.a|.b|.c)")
		// extglobs.js line 665
		assertMatch(t, true, "d.d", "*!(.a|.b|.c)")
		// extglobs.js line 666
		assertMatch(t, true, "e.e", "*!(.a|.b|.c)")
		// extglobs.js line 667
		assertMatch(t, true, "f.f", "*!(.a|.b|.c)")
		// extglobs.js line 668
		assertMatch(t, true, "a.abcd", "*!(.a|.b|.c)")

		// extglobs.js line 670
		assertMatch(t, false, "a.a", "!(*.[a-c])*")
		// extglobs.js line 671
		assertMatch(t, false, "a.b", "!(*.[a-c])*")
		// extglobs.js line 672
		assertMatch(t, false, "a.c", "!(*.[a-c])*")
		// extglobs.js line 673
		assertMatch(t, false, "a.c.d", "!(*.[a-c])*")
		// extglobs.js line 674
		assertMatch(t, false, "c.c", "!(*.[a-c])*")
		// extglobs.js line 675
		assertMatch(t, true, "a.", "!(*.[a-c])*")
		// extglobs.js line 676
		assertMatch(t, true, "d.d", "!(*.[a-c])*")
		// extglobs.js line 677
		assertMatch(t, true, "e.e", "!(*.[a-c])*")
		// extglobs.js line 678
		assertMatch(t, true, "f.f", "!(*.[a-c])*")
		// extglobs.js line 679
		assertMatch(t, false, "a.abcd", "!(*.[a-c])*")

		// extglobs.js line 681
		assertMatch(t, true, "a.a", "*!(.a|.b|.c)*")
		// extglobs.js line 682
		assertMatch(t, true, "a.b", "*!(.a|.b|.c)*")
		// extglobs.js line 683
		assertMatch(t, true, "a.c", "*!(.a|.b|.c)*")
		// extglobs.js line 684
		assertMatch(t, true, "a.c.d", "*!(.a|.b|.c)*")
		// extglobs.js line 685
		assertMatch(t, true, "c.c", "*!(.a|.b|.c)*")
		// extglobs.js line 686
		assertMatch(t, true, "a.", "*!(.a|.b|.c)*")
		// extglobs.js line 687
		assertMatch(t, true, "d.d", "*!(.a|.b|.c)*")
		// extglobs.js line 688
		assertMatch(t, true, "e.e", "*!(.a|.b|.c)*")
		// extglobs.js line 689
		assertMatch(t, true, "f.f", "*!(.a|.b|.c)*")
		// extglobs.js line 690
		assertMatch(t, true, "a.abcd", "*!(.a|.b|.c)*")

		// extglobs.js line 692
		assertMatch(t, false, "a.a", "*.!(a|b|c)*")
		// extglobs.js line 693
		assertMatch(t, false, "a.b", "*.!(a|b|c)*")
		// extglobs.js line 694
		assertMatch(t, false, "a.c", "*.!(a|b|c)*")
		// extglobs.js line 695
		assertMatch(t, true, "a.c.d", "*.!(a|b|c)*")
		// extglobs.js line 696
		assertMatch(t, false, "c.c", "*.!(a|b|c)*")
		// extglobs.js line 697
		assertMatch(t, true, "a.", "*.!(a|b|c)*")
		// extglobs.js line 698
		assertMatch(t, true, "d.d", "*.!(a|b|c)*")
		// extglobs.js line 699
		assertMatch(t, true, "e.e", "*.!(a|b|c)*")
		// extglobs.js line 700
		assertMatch(t, true, "f.f", "*.!(a|b|c)*")
		// extglobs.js line 701
		assertMatch(t, false, "a.abcd", "*.!(a|b|c)*")
	})

	t.Run("should correctly match empty parens", func(t *testing.T) {
		// extglobs.js line 705
		assertMatch(t, false, "def", "@()ef")
		// extglobs.js line 706
		assertMatch(t, true, "ef", "@()ef")

		// extglobs.js line 708
		assertMatch(t, false, "def", "()ef")
		// extglobs.js line 709
		assertMatch(t, true, "ef", "()ef")
	})

	t.Run("should match escaped parens", func(t *testing.T) {
		// extglobs.js line 713-715: platform-specific test for literal backslash in filename
		// Keep Go string encodings runtime-equivalent to the upstream JS literal bytes.
		// Raw strings are only valid here when they preserve the exact same runtime value.
		if runtime.GOOS != "windows" {
			// extglobs.js line 714
			assertMatch(t, true, `a\(b`, "a\\\\\\"+"(b")
		}
		// extglobs.js line 716
		assertMatch(t, true, "a(b", "a(b")
		// extglobs.js line 717
		assertMatch(t, true, "a(b", `a\(b`)
		// extglobs.js line 718
		assertMatch(t, false, "a((b", "a(b")
		// extglobs.js line 719
		assertMatch(t, false, "a((((b", "a(b")
		// extglobs.js line 720
		assertMatch(t, false, "ab", "a(b")

		// extglobs.js line 722
		assertMatch(t, true, "a(b", `a\(b`)
		// extglobs.js line 723
		assertMatch(t, false, "a((b", `a\(b`)
		// extglobs.js line 724
		assertMatch(t, false, "a((((b", `a\(b`)
		// extglobs.js line 725
		assertMatch(t, false, "ab", `a\(b`)

		// extglobs.js line 727
		assertMatch(t, true, "a(b", "a(*b")
		// extglobs.js line 728
		assertMatch(t, true, "a(ab", `a\(*b`)
		// extglobs.js line 729
		assertMatch(t, true, "a((b", "a(*b")
		// extglobs.js line 730
		assertMatch(t, true, "a((((b", "a(*b")
		// extglobs.js line 731
		assertMatch(t, false, "ab", "a(*b")
	})

	t.Run("should match escaped backslashes", func(t *testing.T) {
		// extglobs.js line 735
		assertMatch(t, true, "a(b", `a\(b`)
		// extglobs.js line 736
		assertMatch(t, true, "a((b", `a\(\(b`)
		// extglobs.js line 737
		assertMatch(t, true, "a((((b", `a\(\(\(\(b`)

		// extglobs.js line 739
		assertMatch(t, false, "a(b", `a\\(b`)
		// extglobs.js line 740
		assertMatch(t, false, "a((b", `a\\(b`)
		// extglobs.js line 741
		assertMatch(t, false, "a((((b", `a\\(b`)
		// extglobs.js line 742
		assertMatch(t, false, "ab", `a\\(b`)

		// extglobs.js line 744
		assertMatch(t, false, "a/b", `a\\b`)
		// extglobs.js line 745
		assertMatch(t, false, "ab", `a\\b`)
	})

	// these are not extglobs, and do not need to pass, but they are included
	// to test integration with other features
	t.Run("should support regex characters", func(t *testing.T) {
		// extglobs.js line 751
		fixtures := []string{"a c", "a.c", "a.xy.zc", "a.zc", "a123c", "a1c", "abbbbc", "abbbc", "abbc", "abc", "abq", "axy zc", "axy", "axy.zc", "axyzc"}

		// extglobs.js line 753-755: platform-specific test for literal backslash in filename
		if runtime.GOOS != "windows" {
			// extglobs.js line 754
			assertMatchList(t, []string{`a\b`, "a/b", "ab"}, "a/b", []string{"a/b"})
		}

		// extglobs.js line 757
		assertMatchList(t, []string{"a/b", "ab"}, "a/b", []string{"a/b"})
		// extglobs.js line 758
		assertMatchList(t, fixtures, "ab?bc", []string{"abbbc"})
		// extglobs.js line 759
		assertMatchList(t, fixtures, "ab*c", []string{"abbbbc", "abbbc", "abbc", "abc"})
		// extglobs.js line 760
		assertMatchList(t, fixtures, "a+(b)bc", []string{"abbbbc", "abbbc", "abbc"})
		// extglobs.js line 761
		assertMatchList(t, fixtures, "^abc$", []string{})
		// extglobs.js line 762
		assertMatchList(t, fixtures, "a.c", []string{"a.c"})
		// extglobs.js line 763
		assertMatchList(t, fixtures, "a.*c", []string{"a.c", "a.xy.zc", "a.zc"})
		// extglobs.js line 764
		assertMatchList(t, fixtures, "a*c", []string{"a c", "a.c", "a.xy.zc", "a.zc", "a123c", "a1c", "abbbbc", "abbbc", "abbc", "abc", "axy zc", "axy.zc", "axyzc"})
		// extglobs.js line 765 — Should match word characters
		assertMatchList(t, fixtures, `a[\w]+c`, []string{"a123c", "a1c", "abbbbc", "abbbc", "abbc", "abc", "axyzc"})
		// extglobs.js line 766 — Should match non-word characters
		assertMatchList(t, fixtures, `a[\W]+c`, []string{"a c", "a.c"})
		// extglobs.js line 767 — Should match numbers
		assertMatchList(t, fixtures, `a[\d]+c`, []string{"a123c", "a1c"})
		// extglobs.js line 768
		assertMatchList(t, []string{"foo@#$%123ASD #$$%^&", "foo!@#$asdfl;", "123"}, `[\d]+`, []string{"123"})
		// extglobs.js line 769 — Should match non-numbers
		assertMatchList(t, []string{"a123c", "abbbc"}, `a[\D]+c`, []string{"abbbc"})
		// extglobs.js line 770 — Should match word boundaries
		assertMatchList(t, []string{"foo", " foo "}, `(f|o)+\b`, []string{"foo"})
	})
}
