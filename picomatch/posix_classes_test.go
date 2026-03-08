// posix_classes_test.go — Faithful 1:1 port of picomatch/test/posix-classes.js
package picomatch

import "testing"

// Note: The JS source defines base options as { strictSlashes: true, posix: true, regex: true }
// and wraps isMatch to always pass these. We replicate that here.
var posixOpts = &Options{StrictSlashes: true, Posix: true}

func TestPosixClasses(t *testing.T) {

	// posix-classes.js lines 14-28: "posix bracket type conversion"
	// These tests use parse().output (convert) to verify the regex output.
	// Since we are testing matching behavior (not internal regex strings),
	// we skip the convert() assertions that only test internal regex output
	// and focus on the isMatch behavioral tests below.

	t.Run("isMatch", func(t *testing.T) {
		t.Run("should support POSIX.2 character classes", func(t *testing.T) {
			// posix-classes.js line 33
			assertMatch(t, true, "e", "[[:xdigit:]]", posixOpts)

			// posix-classes.js line 35
			assertMatch(t, true, "a", "[[:alpha:]123]", posixOpts)
			// posix-classes.js line 36
			assertMatch(t, true, "1", "[[:alpha:]123]", posixOpts)
			// posix-classes.js line 37
			assertMatch(t, false, "5", "[[:alpha:]123]", posixOpts)
			// posix-classes.js line 38
			assertMatch(t, true, "A", "[[:alpha:]123]", posixOpts)

			// posix-classes.js line 40
			assertMatch(t, true, "A", "[[:alpha:]]", posixOpts)
			// posix-classes.js line 41
			assertMatch(t, false, "9", "[[:alpha:]]", posixOpts)
			// posix-classes.js line 42
			assertMatch(t, true, "b", "[[:alpha:]]", posixOpts)

			// posix-classes.js line 44
			assertMatch(t, false, "A", "[![:alpha:]]", posixOpts)
			// posix-classes.js line 45
			assertMatch(t, true, "9", "[![:alpha:]]", posixOpts)
			// posix-classes.js line 46
			assertMatch(t, false, "b", "[![:alpha:]]", posixOpts)

			// posix-classes.js line 48
			assertMatch(t, false, "A", "[^[:alpha:]]", posixOpts)
			// posix-classes.js line 49
			assertMatch(t, true, "9", "[^[:alpha:]]", posixOpts)
			// posix-classes.js line 50
			assertMatch(t, false, "b", "[^[:alpha:]]", posixOpts)

			// posix-classes.js line 52
			assertMatch(t, false, "A", "[[:digit:]]", posixOpts)
			// posix-classes.js line 53
			assertMatch(t, true, "9", "[[:digit:]]", posixOpts)
			// posix-classes.js line 54
			assertMatch(t, false, "b", "[[:digit:]]", posixOpts)

			// posix-classes.js line 56
			assertMatch(t, true, "A", "[^[:digit:]]", posixOpts)
			// posix-classes.js line 57
			assertMatch(t, false, "9", "[^[:digit:]]", posixOpts)
			// posix-classes.js line 58
			assertMatch(t, true, "b", "[^[:digit:]]", posixOpts)

			// posix-classes.js line 60
			assertMatch(t, true, "A", "[![:digit:]]", posixOpts)
			// posix-classes.js line 61
			assertMatch(t, false, "9", "[![:digit:]]", posixOpts)
			// posix-classes.js line 62
			assertMatch(t, true, "b", "[![:digit:]]", posixOpts)

			// posix-classes.js line 64
			assertMatch(t, true, "a", "[[:lower:]]", posixOpts)
			// posix-classes.js line 65
			assertMatch(t, false, "A", "[[:lower:]]", posixOpts)
			// posix-classes.js line 66
			assertMatch(t, false, "9", "[[:lower:]]", posixOpts)

			// posix-classes.js line 68 — invalid posix bracket, but valid char class
			assertMatch(t, true, "a", "[:alpha:]", posixOpts)
			// posix-classes.js line 69 — invalid posix bracket, but valid char class
			assertMatch(t, true, "l", "[:alpha:]", posixOpts)
			// posix-classes.js line 70 — invalid posix bracket, but valid char class
			assertMatch(t, true, "p", "[:alpha:]", posixOpts)
			// posix-classes.js line 71 — invalid posix bracket, but valid char class
			assertMatch(t, true, "h", "[:alpha:]", posixOpts)
			// posix-classes.js line 72 — invalid posix bracket, but valid char class
			assertMatch(t, true, ":", "[:alpha:]", posixOpts)
			// posix-classes.js line 73 — invalid posix bracket, but valid char class
			assertMatch(t, false, "b", "[:alpha:]", posixOpts)
		})

		t.Run("should support multiple posix brackets in one character class", func(t *testing.T) {
			// posix-classes.js line 77
			assertMatch(t, true, "9", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 78
			assertMatch(t, true, "a", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 79
			assertMatch(t, false, "A", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 80
			assertMatch(t, false, "aa", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 81
			assertMatch(t, false, "99", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 82
			assertMatch(t, false, "a9", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 83
			assertMatch(t, false, "9a", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 84
			assertMatch(t, false, "aA", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 85
			assertMatch(t, false, "9A", "[[:lower:][:digit:]]", posixOpts)
			// posix-classes.js line 86
			assertMatch(t, true, "aa", "[[:lower:][:digit:]]+", posixOpts)
			// posix-classes.js line 87
			assertMatch(t, true, "99", "[[:lower:][:digit:]]+", posixOpts)
			// posix-classes.js line 88
			assertMatch(t, true, "a9", "[[:lower:][:digit:]]+", posixOpts)
			// posix-classes.js line 89
			assertMatch(t, true, "9a", "[[:lower:][:digit:]]+", posixOpts)
			// posix-classes.js line 90
			assertMatch(t, false, "aA", "[[:lower:][:digit:]]+", posixOpts)
			// posix-classes.js line 91
			assertMatch(t, false, "9A", "[[:lower:][:digit:]]+", posixOpts)
			// posix-classes.js line 92
			assertMatch(t, true, "a", "[[:lower:][:digit:]]*", posixOpts)
			// posix-classes.js line 93
			assertMatch(t, false, "A", "[[:lower:][:digit:]]*", posixOpts)
			// posix-classes.js line 94
			assertMatch(t, false, "AA", "[[:lower:][:digit:]]*", posixOpts)
			// posix-classes.js line 95
			assertMatch(t, true, "aa", "[[:lower:][:digit:]]*", posixOpts)
			// posix-classes.js line 96
			assertMatch(t, true, "aaa", "[[:lower:][:digit:]]*", posixOpts)
			// posix-classes.js line 97
			assertMatch(t, true, "999", "[[:lower:][:digit:]]*", posixOpts)
		})

		t.Run("should match word characters", func(t *testing.T) {
			// posix-classes.js line 101
			assertMatch(t, false, "a c", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 102
			assertMatch(t, false, "a.c", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 103
			assertMatch(t, false, "a.xy.zc", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 104
			assertMatch(t, false, "a.zc", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 105
			assertMatch(t, false, "abq", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 106
			assertMatch(t, false, "axy zc", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 107
			assertMatch(t, false, "axy", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 108
			assertMatch(t, false, "axy.zc", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 109
			assertMatch(t, true, "a123c", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 110
			assertMatch(t, true, "a1c", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 111
			assertMatch(t, true, "abbbbc", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 112
			assertMatch(t, true, "abbbc", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 113
			assertMatch(t, true, "abbc", "a[[:word:]]+c", posixOpts)
			// posix-classes.js line 114
			assertMatch(t, true, "abc", "a[[:word:]]+c", posixOpts)

			// posix-classes.js line 116
			assertMatch(t, false, "a c", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 117
			assertMatch(t, false, "a.c", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 118
			assertMatch(t, false, "a.xy.zc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 119
			assertMatch(t, false, "a.zc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 120
			assertMatch(t, false, "axy zc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 121
			assertMatch(t, false, "axy.zc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 122
			assertMatch(t, true, "a123c", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 123
			assertMatch(t, true, "a1c", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 124
			assertMatch(t, true, "abbbbc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 125
			assertMatch(t, true, "abbbc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 126
			assertMatch(t, true, "abbc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 127
			assertMatch(t, true, "abc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 128
			assertMatch(t, true, "abq", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 129
			assertMatch(t, true, "axy", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 130
			assertMatch(t, true, "axyzc", "a[[:word:]]+", posixOpts)
			// posix-classes.js line 131
			assertMatch(t, true, "axyzc", "a[[:word:]]+", posixOpts)
		})

		// posix-classes.js lines 134-137: "should not create an invalid posix character class"
		// These are convert() tests that check internal regex output, not matching behavior.
		// Skipped because they test parse().output, not isMatch.

		t.Run("should return true when the pattern matches", func(t *testing.T) {
			// posix-classes.js line 140
			assertMatch(t, true, "a", "[[:lower:]]", posixOpts)
			// posix-classes.js line 141
			assertMatch(t, true, "A", "[[:upper:]]", posixOpts)
			// posix-classes.js line 142
			assertMatch(t, true, "A", "[[:digit:][:upper:][:space:]]", posixOpts)
			// posix-classes.js line 143
			assertMatch(t, true, "1", "[[:digit:][:upper:][:space:]]", posixOpts)
			// posix-classes.js line 144
			assertMatch(t, true, " ", "[[:digit:][:upper:][:space:]]", posixOpts)
			// posix-classes.js line 145
			assertMatch(t, true, "5", "[[:xdigit:]]", posixOpts)
			// posix-classes.js line 146
			assertMatch(t, true, "f", "[[:xdigit:]]", posixOpts)
			// posix-classes.js line 147
			assertMatch(t, true, "D", "[[:xdigit:]]", posixOpts)
			// posix-classes.js line 148
			assertMatch(t, true, "_", "[[:alnum:][:alpha:][:blank:][:cntrl:][:digit:][:graph:][:lower:][:print:][:punct:][:space:][:upper:][:xdigit:]]", posixOpts)
			// posix-classes.js line 149
			assertMatch(t, true, "_", "[[:alnum:][:alpha:][:blank:][:cntrl:][:digit:][:graph:][:lower:][:print:][:punct:][:space:][:upper:][:xdigit:]]", posixOpts)
			// posix-classes.js line 150
			assertMatch(t, true, ".", "[^[:alnum:][:alpha:][:blank:][:cntrl:][:digit:][:lower:][:space:][:upper:][:xdigit:]]", posixOpts)
			// posix-classes.js line 151
			assertMatch(t, true, "5", "[a-c[:digit:]x-z]", posixOpts)
			// posix-classes.js line 152
			assertMatch(t, true, "b", "[a-c[:digit:]x-z]", posixOpts)
			// posix-classes.js line 153
			assertMatch(t, true, "y", "[a-c[:digit:]x-z]", posixOpts)
		})

		t.Run("should return false when the pattern does not match", func(t *testing.T) {
			// posix-classes.js line 157
			assertMatch(t, false, "A", "[[:lower:]]", posixOpts)
			// posix-classes.js line 158
			assertMatch(t, true, "A", "[![:lower:]]", posixOpts)
			// posix-classes.js line 159
			assertMatch(t, false, "a", "[[:upper:]]", posixOpts)
			// posix-classes.js line 160
			assertMatch(t, false, "a", "[[:digit:][:upper:][:space:]]", posixOpts)
			// posix-classes.js line 161
			assertMatch(t, false, ".", "[[:digit:][:upper:][:space:]]", posixOpts)
			// posix-classes.js line 162
			assertMatch(t, false, ".", "[[:alnum:][:alpha:][:blank:][:cntrl:][:digit:][:lower:][:space:][:upper:][:xdigit:]]", posixOpts)
			// posix-classes.js line 163
			assertMatch(t, false, "q", "[a-c[:digit:]x-z]", posixOpts)
		})
	})

	t.Run("literals", func(t *testing.T) {
		t.Run("should match literal brackets when escaped", func(t *testing.T) {
			// posix-classes.js line 169
			assertMatch(t, true, "a [b]", "a [b]", posixOpts)
			// posix-classes.js line 170
			assertMatch(t, true, "a b", "a [b]", posixOpts)

			// posix-classes.js line 172
			assertMatch(t, true, "a [b] c", "a [b] c", posixOpts)
			// posix-classes.js line 173
			assertMatch(t, true, "a b c", "a [b] c", posixOpts)

			// posix-classes.js line 175
			assertMatch(t, true, "a [b]", "a \\[b\\]", posixOpts)
			// posix-classes.js line 176
			assertMatch(t, false, "a b", "a \\[b\\]", posixOpts)

			// posix-classes.js line 178
			assertMatch(t, true, "a [b]", "a ([b])", posixOpts)
			// posix-classes.js line 179
			assertMatch(t, true, "a b", "a ([b])", posixOpts)

			// posix-classes.js line 181
			assertMatch(t, true, "a b", "a (\\[b\\]|[b])", posixOpts)
			// posix-classes.js line 182
			assertMatch(t, true, "a [b]", "a (\\[b\\]|[b])", posixOpts)
		})
	})

	// posix-classes.js lines 186-191: ".makeRe()" tests
	// These test makeRe() output (regex object equality), not matching behavior.
	// Skipped because our Go API doesn't expose makeRe in the same way.

	t.Run("POSIX: From the test suite for the POSIX.2 (BRE) pattern matching code", func(t *testing.T) {
		t.Run("First, test POSIX.2 character classes", func(t *testing.T) {
			// posix-classes.js line 195
			assertMatch(t, true, "e", "[[:xdigit:]]", posixOpts)
			// posix-classes.js line 196
			assertMatch(t, true, "1", "[[:xdigit:]]", posixOpts)
			// posix-classes.js line 197
			assertMatch(t, true, "a", "[[:alpha:]123]", posixOpts)
			// posix-classes.js line 198
			assertMatch(t, true, "1", "[[:alpha:]123]", posixOpts)
		})

		t.Run("should match using POSIX.2 negation patterns", func(t *testing.T) {
			// posix-classes.js line 202
			assertMatch(t, true, "9", "[![:alpha:]]", posixOpts)
			// posix-classes.js line 203
			assertMatch(t, true, "9", "[^[:alpha:]]", posixOpts)
		})

		t.Run("should match word characters", func(t *testing.T) {
			// posix-classes.js line 207
			assertMatch(t, true, "A", "[[:word:]]", posixOpts)
			// posix-classes.js line 208
			assertMatch(t, true, "B", "[[:word:]]", posixOpts)
			// posix-classes.js line 209
			assertMatch(t, true, "a", "[[:word:]]", posixOpts)
			// posix-classes.js line 210
			assertMatch(t, true, "b", "[[:word:]]", posixOpts)
		})

		t.Run("should match digits with word class", func(t *testing.T) {
			// posix-classes.js line 214
			assertMatch(t, true, "1", "[[:word:]]", posixOpts)
			// posix-classes.js line 215
			assertMatch(t, true, "2", "[[:word:]]", posixOpts)
		})

		t.Run("should not digits", func(t *testing.T) {
			// posix-classes.js line 219
			assertMatch(t, true, "1", "[[:digit:]]", posixOpts)
			// posix-classes.js line 220
			assertMatch(t, true, "2", "[[:digit:]]", posixOpts)
		})

		t.Run("should not match word characters with digit class", func(t *testing.T) {
			// posix-classes.js line 224
			assertMatch(t, false, "a", "[[:digit:]]", posixOpts)
			// posix-classes.js line 225
			assertMatch(t, false, "A", "[[:digit:]]", posixOpts)
		})

		t.Run("should match uppercase alpha characters", func(t *testing.T) {
			// posix-classes.js line 229
			assertMatch(t, true, "A", "[[:upper:]]", posixOpts)
			// posix-classes.js line 230
			assertMatch(t, true, "B", "[[:upper:]]", posixOpts)
		})

		t.Run("should not match lowercase alpha characters", func(t *testing.T) {
			// posix-classes.js line 234
			assertMatch(t, false, "a", "[[:upper:]]", posixOpts)
			// posix-classes.js line 235
			assertMatch(t, false, "b", "[[:upper:]]", posixOpts)
		})

		t.Run("should not match digits with upper class", func(t *testing.T) {
			// posix-classes.js line 239
			assertMatch(t, false, "1", "[[:upper:]]", posixOpts)
			// posix-classes.js line 240
			assertMatch(t, false, "2", "[[:upper:]]", posixOpts)
		})

		t.Run("should match lowercase alpha characters", func(t *testing.T) {
			// posix-classes.js line 244
			assertMatch(t, true, "a", "[[:lower:]]", posixOpts)
			// posix-classes.js line 245
			assertMatch(t, true, "b", "[[:lower:]]", posixOpts)
		})

		t.Run("should not match uppercase alpha characters", func(t *testing.T) {
			// posix-classes.js line 249
			assertMatch(t, false, "A", "[[:lower:]]", posixOpts)
			// posix-classes.js line 250
			assertMatch(t, false, "B", "[[:lower:]]", posixOpts)
		})

		t.Run("should match one lower and one upper character", func(t *testing.T) {
			// posix-classes.js line 254
			assertMatch(t, true, "aA", "[[:lower:]][[:upper:]]", posixOpts)
			// posix-classes.js line 255
			assertMatch(t, false, "AA", "[[:lower:]][[:upper:]]", posixOpts)
			// posix-classes.js line 256
			assertMatch(t, false, "Aa", "[[:lower:]][[:upper:]]", posixOpts)
		})

		t.Run("should match hexadecimal digits", func(t *testing.T) {
			// posix-classes.js line 260
			assertMatch(t, true, "ababab", "[[:xdigit:]]*", posixOpts)
			// posix-classes.js line 261
			assertMatch(t, true, "020202", "[[:xdigit:]]*", posixOpts)
			// posix-classes.js line 262
			assertMatch(t, true, "900", "[[:xdigit:]]*", posixOpts)
		})

		t.Run("should match punctuation characters", func(t *testing.T) {
			// posix-classes.js line 266
			assertMatch(t, true, "!", "[[:punct:]]", posixOpts)
			// posix-classes.js line 267
			assertMatch(t, true, "?", "[[:punct:]]", posixOpts)
			// posix-classes.js line 268
			assertMatch(t, true, "#", "[[:punct:]]", posixOpts)
			// posix-classes.js line 269
			assertMatch(t, true, "&", "[[:punct:]]", posixOpts)
			// posix-classes.js line 270
			assertMatch(t, true, "@", "[[:punct:]]", posixOpts)
			// posix-classes.js line 271
			assertMatch(t, true, "+", "[[:punct:]]", posixOpts)
			// posix-classes.js line 272
			assertMatch(t, true, "*", "[[:punct:]]", posixOpts)
			// posix-classes.js line 273
			assertMatch(t, true, ":", "[[:punct:]]", posixOpts)
			// posix-classes.js line 274
			assertMatch(t, true, "=", "[[:punct:]]", posixOpts)
			// posix-classes.js line 275
			assertMatch(t, true, "|", "[[:punct:]]", posixOpts)
			// posix-classes.js line 276
			assertMatch(t, true, "|++", "[[:punct:]]*", posixOpts)
		})

		t.Run("should only match one character", func(t *testing.T) {
			// posix-classes.js line 280
			assertMatch(t, false, "?*+", "[[:punct:]]", posixOpts)
		})

		t.Run("should only match zero or more punctuation characters", func(t *testing.T) {
			// posix-classes.js line 284
			assertMatch(t, true, "?*+", "[[:punct:]]*", posixOpts)
			// posix-classes.js line 285
			assertMatch(t, true, "foo", "foo[[:punct:]]*", posixOpts)
			// posix-classes.js line 286
			assertMatch(t, true, "foo?*+", "foo[[:punct:]]*", posixOpts)
		})

		t.Run("invalid character class expressions are just characters to be matched", func(t *testing.T) {
			// posix-classes.js line 290
			assertMatch(t, true, "a", "[:al:]", posixOpts)
			// posix-classes.js line 291
			assertMatch(t, true, "a", "[[:al:]", posixOpts)
			// posix-classes.js line 292
			assertMatch(t, true, "!", "[abc[:punct:][0-9]", posixOpts)
		})

		t.Run("should match the start of a valid sh identifier", func(t *testing.T) {
			// posix-classes.js line 296
			assertMatch(t, true, "PATH", "[_[:alpha:]]*", posixOpts)
		})

		t.Run("should match the first two characters of a valid sh identifier", func(t *testing.T) {
			// posix-classes.js line 300
			assertMatch(t, true, "PATH", "[_[:alpha:]][_[:alnum:]]*", posixOpts)
		})

		t.Run("should match multiple posix classes", func(t *testing.T) {
			// posix-classes.js line 304
			assertMatch(t, true, "a1B", "[[:alpha:]][[:digit:]][[:upper:]]", posixOpts)
			// posix-classes.js line 305
			assertMatch(t, false, "a1b", "[[:alpha:]][[:digit:]][[:upper:]]", posixOpts)
			// posix-classes.js line 306
			assertMatch(t, true, ".", "[[:digit:][:punct:][:space:]]", posixOpts)
			// posix-classes.js line 307
			assertMatch(t, false, "a", "[[:digit:][:punct:][:space:]]", posixOpts)
			// posix-classes.js line 308
			assertMatch(t, true, "!", "[[:digit:][:punct:][:space:]]", posixOpts)
			// posix-classes.js line 309
			assertMatch(t, false, "!", "[[:digit:]][[:punct:]][[:space:]]", posixOpts)
			// posix-classes.js line 310
			assertMatch(t, true, "1! ", "[[:digit:]][[:punct:]][[:space:]]", posixOpts)
			// posix-classes.js line 311
			assertMatch(t, false, "1!  ", "[[:digit:]][[:punct:]][[:space:]]", posixOpts)
		})

		// posix-classes.js line 314-317: comment about tests ported from Bash 4.3

		t.Run("how about A?", func(t *testing.T) {
			// posix-classes.js line 320
			assertMatch(t, true, "9", "[[:digit:]]", posixOpts)
			// posix-classes.js line 321
			assertMatch(t, false, "X", "[[:digit:]]", posixOpts)
			// posix-classes.js line 322
			assertMatch(t, true, "aB", "[[:lower:]][[:upper:]]", posixOpts)
			// posix-classes.js line 323
			assertMatch(t, true, "a", "[[:alpha:][:digit:]]", posixOpts)
			// posix-classes.js line 324
			assertMatch(t, true, "3", "[[:alpha:][:digit:]]", posixOpts)
			// posix-classes.js line 325
			assertMatch(t, false, "aa", "[[:alpha:][:digit:]]", posixOpts)
			// posix-classes.js line 326
			assertMatch(t, false, "a3", "[[:alpha:][:digit:]]", posixOpts)
			// posix-classes.js line 327
			assertMatch(t, false, "a", "[[:alpha:]\\]", posixOpts)
			// posix-classes.js line 328
			assertMatch(t, false, "b", "[[:alpha:]\\]", posixOpts)
		})

		t.Run("OK, what's a tab?  is it a blank? a space?", func(t *testing.T) {
			// posix-classes.js line 332
			assertMatch(t, true, "\t", "[[:blank:]]", posixOpts)
			// posix-classes.js line 333
			assertMatch(t, true, "\t", "[[:space:]]", posixOpts)
			// posix-classes.js line 334
			assertMatch(t, true, " ", "[[:space:]]", posixOpts)
		})

		t.Run("let's check out characters in the ASCII range", func(t *testing.T) {
			// posix-classes.js line 338
			assertMatch(t, false, "\\377", "[[:ascii:]]", posixOpts)
			// posix-classes.js line 339
			assertMatch(t, false, "9", "[1[:alpha:]123]", posixOpts)
		})

		t.Run("punctuation", func(t *testing.T) {
			// posix-classes.js line 343
			assertMatch(t, false, " ", "[[:punct:]]", posixOpts)
		})

		t.Run("graph", func(t *testing.T) {
			// posix-classes.js line 347
			assertMatch(t, true, "A", "[[:graph:]]", posixOpts)
			// posix-classes.js line 348
			assertMatch(t, false, "\\b", "[[:graph:]]", posixOpts)
			// posix-classes.js line 349
			assertMatch(t, false, "\\n", "[[:graph:]]", posixOpts)
			// posix-classes.js line 350
			assertMatch(t, false, "\\s", "[[:graph:]]", posixOpts)
		})
	})
}
