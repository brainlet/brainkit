// slashes_win_test.go — Faithful 1:1 port of picomatch/test/slashes-windows.js
// NOTE: Named slashes_win_test.go (not slashes_windows_test.go) to avoid Go's
// implicit GOOS=windows build constraint for files ending in _windows.go.
package picomatch

import (
	"testing"
)

func TestSlashHandlingWindows(t *testing.T) {
	// Shorthand options used throughout this file
	win := &Options{Windows: true}
	nowin := &Options{Windows: false}

	t.Run("should match absolute windows paths with regex from makeRe", func(t *testing.T) {
		// slashes-windows.js line 8-9
		// Note: The JS test uses makeRe + regex.test(). We test via IsMatch which compiles the same regex.
		assertMatch(t, true, "C:\\Users\\user\\Projects\\project\\path\\image.jpg", "**/path/**", win)
	})

	t.Run("should match windows path separators with a string literal", func(t *testing.T) {
		// slashes-windows.js line 13
		assertMatch(t, false, "a\\a", "(a/b)", win)
		// slashes-windows.js line 14
		assertMatch(t, true, "a\\b", "(a/b)", win)
		// slashes-windows.js line 15
		assertMatch(t, false, "a\\c", "(a/b)", win)
		// slashes-windows.js line 16
		assertMatch(t, false, "b\\a", "(a/b)", win)
		// slashes-windows.js line 17
		assertMatch(t, false, "b\\b", "(a/b)", win)
		// slashes-windows.js line 18
		assertMatch(t, false, "b\\c", "(a/b)", win)

		// slashes-windows.js line 20
		assertMatch(t, false, "a\\a", "a/b", win)
		// slashes-windows.js line 21
		assertMatch(t, true, "a\\b", "a/b", win)
		// slashes-windows.js line 22
		assertMatch(t, false, "a\\c", "a/b", win)
		// slashes-windows.js line 23
		assertMatch(t, false, "b\\a", "a/b", win)
		// slashes-windows.js line 24
		assertMatch(t, false, "b\\b", "a/b", win)
		// slashes-windows.js line 25
		assertMatch(t, false, "b\\c", "a/b", win)
	})

	t.Run("should not match literal backslashes with literal forward slashes when windows is disabled", func(t *testing.T) {
		// slashes-windows.js line 29
		assertMatch(t, false, "a\\a", "a\\b", nowin)
		// slashes-windows.js line 30
		assertMatch(t, true, "a\\b", "a\\b", nowin)
		// slashes-windows.js line 31
		assertMatch(t, false, "a\\c", "a\\b", nowin)
		// slashes-windows.js line 32
		assertMatch(t, false, "b\\a", "a\\b", nowin)
		// slashes-windows.js line 33
		assertMatch(t, false, "b\\b", "a\\b", nowin)
		// slashes-windows.js line 34
		assertMatch(t, false, "b\\c", "a\\b", nowin)

		// slashes-windows.js line 36
		assertMatch(t, false, "a\\a", "a/b", nowin)
		// slashes-windows.js line 37
		assertMatch(t, false, "a\\b", "a/b", nowin)
		// slashes-windows.js line 38
		assertMatch(t, false, "a\\c", "a/b", nowin)
		// slashes-windows.js line 39
		assertMatch(t, false, "b\\a", "a/b", nowin)
		// slashes-windows.js line 40
		assertMatch(t, false, "b\\b", "a/b", nowin)
		// slashes-windows.js line 41
		assertMatch(t, false, "b\\c", "a/b", nowin)
	})

	t.Run("should match an array of literal strings", func(t *testing.T) {
		// slashes-windows.js line 45
		assertMatch(t, false, "a\\a", "(a/b)", win)
		// slashes-windows.js line 46
		assertMatch(t, true, "a\\b", "(a/b)", win)
		// slashes-windows.js line 47
		assertMatch(t, false, "a\\c", "(a/b)", win)
		// slashes-windows.js line 48
		assertMatch(t, false, "b\\a", "(a/b)", win)
		// slashes-windows.js line 49
		assertMatch(t, false, "b\\b", "(a/b)", win)
		// slashes-windows.js line 50
		assertMatch(t, false, "b\\c", "(a/b)", win)
	})

	t.Run("should not match backslashes with forward slashes when windows is disabled", func(t *testing.T) {
		// slashes-windows.js line 54
		assertMatch(t, false, "a\\a", "a/(a|c)", nowin)
		// slashes-windows.js line 55
		assertMatch(t, false, "a\\b", "a/(a|c)", nowin)
		// slashes-windows.js line 56
		assertMatch(t, false, "a\\c", "a/(a|c)", nowin)
		// slashes-windows.js line 57
		assertMatch(t, false, "a\\a", "a/(a|b|c)", nowin)
		// slashes-windows.js line 58
		assertMatch(t, false, "a\\b", "a/(a|b|c)", nowin)
		// slashes-windows.js line 59
		assertMatch(t, false, "a\\c", "a/(a|b|c)", nowin)
		// slashes-windows.js line 60
		assertMatch(t, false, "a\\a", "(a\\b)", nowin)
		// slashes-windows.js line 61
		assertMatch(t, true, "a\\b", "(a\\\\b)", nowin)
		// slashes-windows.js line 62
		assertMatch(t, false, "a\\c", "(a\\b)", nowin)
		// slashes-windows.js line 63
		assertMatch(t, false, "b\\a", "(a\\b)", nowin)
		// slashes-windows.js line 64
		assertMatch(t, false, "b\\b", "(a\\b)", nowin)
		// slashes-windows.js line 65
		assertMatch(t, false, "b\\c", "(a\\b)", nowin)
		// slashes-windows.js line 66
		assertMatch(t, false, "a\\a", "(a/b)", nowin)
		// slashes-windows.js line 67
		assertMatch(t, false, "a\\b", "(a/b)", nowin)
		// slashes-windows.js line 68
		assertMatch(t, false, "a\\c", "(a/b)", nowin)
		// slashes-windows.js line 69
		assertMatch(t, false, "b\\a", "(a/b)", nowin)
		// slashes-windows.js line 70
		assertMatch(t, false, "b\\b", "(a/b)", nowin)
		// slashes-windows.js line 71
		assertMatch(t, false, "b\\c", "(a/b)", nowin)

		// slashes-windows.js line 73
		assertMatch(t, false, "a\\a", "a/c", nowin)
		// slashes-windows.js line 74
		assertMatch(t, false, "a\\b", "a/c", nowin)
		// slashes-windows.js line 75
		assertMatch(t, false, "a\\c", "a/c", nowin)
		// slashes-windows.js line 76
		assertMatch(t, false, "b\\a", "a/c", nowin)
		// slashes-windows.js line 77
		assertMatch(t, false, "b\\b", "a/c", nowin)
		// slashes-windows.js line 78
		assertMatch(t, false, "b\\c", "a/c", nowin)
	})

	t.Run("should match backslashes when followed by regex logical or", func(t *testing.T) {
		// slashes-windows.js line 82
		assertMatch(t, true, "a\\a", "a/(a|c)", win)
		// slashes-windows.js line 83
		assertMatch(t, false, "a\\b", "a/(a|c)", win)
		// slashes-windows.js line 84
		assertMatch(t, true, "a\\c", "a/(a|c)", win)

		// slashes-windows.js line 86
		assertMatch(t, true, "a\\a", "a/(a|b|c)", win)
		// slashes-windows.js line 87
		assertMatch(t, true, "a\\b", "a/(a|b|c)", win)
		// slashes-windows.js line 88
		assertMatch(t, true, "a\\c", "a/(a|b|c)", win)
	})

	t.Run("should support matching backslashes with regex ranges", func(t *testing.T) {
		// slashes-windows.js line 92
		assertMatch(t, false, "a\\a", "a/[b-c]", win)
		// slashes-windows.js line 93
		assertMatch(t, true, "a\\b", "a/[b-c]", win)
		// slashes-windows.js line 94
		assertMatch(t, true, "a\\c", "a/[b-c]", win)
		// slashes-windows.js line 95
		assertMatch(t, false, "a\\x\\y", "a/[b-c]", win)
		// slashes-windows.js line 96
		assertMatch(t, false, "a\\x", "a/[b-c]", win)

		// slashes-windows.js line 98
		assertMatch(t, true, "a\\a", "a/[a-z]", win)
		// slashes-windows.js line 99
		assertMatch(t, true, "a\\b", "a/[a-z]", win)
		// slashes-windows.js line 100
		assertMatch(t, true, "a\\c", "a/[a-z]", win)
		// slashes-windows.js line 101
		assertMatch(t, false, "a\\x\\y", "a/[a-z]", win)
		// slashes-windows.js line 102
		assertMatch(t, true, "a\\x\\y", "a/[a-z]/y", win)
		// slashes-windows.js line 103
		assertMatch(t, true, "a\\x", "a/[a-z]", win)

		// slashes-windows.js line 105
		assertMatch(t, false, "a\\a", "a/[b-c]", nowin)
		// slashes-windows.js line 106
		assertMatch(t, false, "a\\b", "a/[b-c]", nowin)
		// slashes-windows.js line 107
		assertMatch(t, false, "a\\c", "a/[b-c]", nowin)
		// slashes-windows.js line 108
		assertMatch(t, false, "a\\x\\y", "a/[b-c]", nowin)
		// slashes-windows.js line 109
		assertMatch(t, false, "a\\x", "a/[b-c]", nowin)

		// slashes-windows.js line 111
		assertMatch(t, false, "a\\a", "a/[a-z]", nowin)
		// slashes-windows.js line 112
		assertMatch(t, false, "a\\b", "a/[a-z]", nowin)
		// slashes-windows.js line 113
		assertMatch(t, false, "a\\c", "a/[a-z]", nowin)
		// slashes-windows.js line 114
		assertMatch(t, false, "a\\x\\y", "a/[a-z]", nowin)
		// slashes-windows.js line 115
		assertMatch(t, false, "a\\x", "a/[a-z]", nowin)
	})

	t.Run("should not match slashes with single stars", func(t *testing.T) {
		// Pattern: *
		// slashes-windows.js line 119
		assertMatch(t, true, "a", "*", win)
		// slashes-windows.js line 120
		assertMatch(t, true, "b", "*", win)
		// slashes-windows.js line 121
		assertMatch(t, false, "a\\a", "*", win)
		// slashes-windows.js line 122
		assertMatch(t, false, "a\\b", "*", win)
		// slashes-windows.js line 123
		assertMatch(t, false, "a\\c", "*", win)
		// slashes-windows.js line 124
		assertMatch(t, false, "a\\x", "*", win)
		// slashes-windows.js line 125
		assertMatch(t, false, "a\\a\\a", "*", win)
		// slashes-windows.js line 126
		assertMatch(t, false, "a\\a\\b", "*", win)
		// slashes-windows.js line 127
		assertMatch(t, false, "a\\a\\a\\a", "*", win)
		// slashes-windows.js line 128
		assertMatch(t, false, "a\\a\\a\\a\\a", "*", win)
		// slashes-windows.js line 129
		assertMatch(t, false, "x\\y", "*", win)
		// slashes-windows.js line 130
		assertMatch(t, false, "z\\z", "*", win)

		// Pattern: */*
		// slashes-windows.js line 132
		assertMatch(t, false, "a", "*/*", win)
		// slashes-windows.js line 133
		assertMatch(t, false, "b", "*/*", win)
		// slashes-windows.js line 134
		assertMatch(t, true, "a\\a", "*/*", win)
		// slashes-windows.js line 135
		assertMatch(t, true, "a\\b", "*/*", win)
		// slashes-windows.js line 136
		assertMatch(t, true, "a\\c", "*/*", win)
		// slashes-windows.js line 137
		assertMatch(t, true, "a\\x", "*/*", win)
		// slashes-windows.js line 138
		assertMatch(t, false, "a\\a\\a", "*/*", win)
		// slashes-windows.js line 139
		assertMatch(t, false, "a\\a\\b", "*/*", win)
		// slashes-windows.js line 140
		assertMatch(t, false, "a\\a\\a\\a", "*/*", win)
		// slashes-windows.js line 141
		assertMatch(t, false, "a\\a\\a\\a\\a", "*/*", win)
		// slashes-windows.js line 142
		assertMatch(t, true, "x\\y", "*/*", win)
		// slashes-windows.js line 143
		assertMatch(t, true, "z\\z", "*/*", win)

		// Pattern: */*/*
		// slashes-windows.js line 145
		assertMatch(t, false, "a", "*/*/*", win)
		// slashes-windows.js line 146
		assertMatch(t, false, "b", "*/*/*", win)
		// slashes-windows.js line 147
		assertMatch(t, false, "a\\a", "*/*/*", win)
		// slashes-windows.js line 148
		assertMatch(t, false, "a\\b", "*/*/*", win)
		// slashes-windows.js line 149
		assertMatch(t, false, "a\\c", "*/*/*", win)
		// slashes-windows.js line 150
		assertMatch(t, false, "a\\x", "*/*/*", win)
		// slashes-windows.js line 151
		assertMatch(t, true, "a\\a\\a", "*/*/*", win)
		// slashes-windows.js line 152
		assertMatch(t, true, "a\\a\\b", "*/*/*", win)
		// slashes-windows.js line 153
		assertMatch(t, false, "a\\a\\a\\a", "*/*/*", win)
		// slashes-windows.js line 154
		assertMatch(t, false, "a\\a\\a\\a\\a", "*/*/*", win)
		// slashes-windows.js line 155
		assertMatch(t, false, "x\\y", "*/*/*", win)
		// slashes-windows.js line 156
		assertMatch(t, false, "z\\z", "*/*/*", win)

		// Pattern: */*/*/*
		// slashes-windows.js line 158
		assertMatch(t, false, "a", "*/*/*/*", win)
		// slashes-windows.js line 159
		assertMatch(t, false, "b", "*/*/*/*", win)
		// slashes-windows.js line 160
		assertMatch(t, false, "a\\a", "*/*/*/*", win)
		// slashes-windows.js line 161
		assertMatch(t, false, "a\\b", "*/*/*/*", win)
		// slashes-windows.js line 162
		assertMatch(t, false, "a\\c", "*/*/*/*", win)
		// slashes-windows.js line 163
		assertMatch(t, false, "a\\x", "*/*/*/*", win)
		// slashes-windows.js line 164
		assertMatch(t, false, "a\\a\\a", "*/*/*/*", win)
		// slashes-windows.js line 165
		assertMatch(t, false, "a\\a\\b", "*/*/*/*", win)
		// slashes-windows.js line 166
		assertMatch(t, true, "a\\a\\a\\a", "*/*/*/*", win)
		// slashes-windows.js line 167
		assertMatch(t, false, "a\\a\\a\\a\\a", "*/*/*/*", win)
		// slashes-windows.js line 168
		assertMatch(t, false, "x\\y", "*/*/*/*", win)
		// slashes-windows.js line 169
		assertMatch(t, false, "z\\z", "*/*/*/*", win)

		// Pattern: */*/*/*/*
		// slashes-windows.js line 171
		assertMatch(t, false, "a", "*/*/*/*/*", win)
		// slashes-windows.js line 172
		assertMatch(t, false, "b", "*/*/*/*/*", win)
		// slashes-windows.js line 173
		assertMatch(t, false, "a\\a", "*/*/*/*/*", win)
		// slashes-windows.js line 174
		assertMatch(t, false, "a\\b", "*/*/*/*/*", win)
		// slashes-windows.js line 175
		assertMatch(t, false, "a\\c", "*/*/*/*/*", win)
		// slashes-windows.js line 176
		assertMatch(t, false, "a\\x", "*/*/*/*/*", win)
		// slashes-windows.js line 177
		assertMatch(t, false, "a\\a\\a", "*/*/*/*/*", win)
		// slashes-windows.js line 178
		assertMatch(t, false, "a\\a\\b", "*/*/*/*/*", win)
		// slashes-windows.js line 179
		assertMatch(t, false, "a\\a\\a\\a", "*/*/*/*/*", win)
		// slashes-windows.js line 180
		assertMatch(t, true, "a\\a\\a\\a\\a", "*/*/*/*/*", win)
		// slashes-windows.js line 181
		assertMatch(t, false, "x\\y", "*/*/*/*/*", win)
		// slashes-windows.js line 182
		assertMatch(t, false, "z\\z", "*/*/*/*/*", win)

		// Pattern: a/*
		// slashes-windows.js line 184
		assertMatch(t, false, "a", "a/*", win)
		// slashes-windows.js line 185
		assertMatch(t, false, "b", "a/*", win)
		// slashes-windows.js line 186
		assertMatch(t, true, "a\\a", "a/*", win)
		// slashes-windows.js line 187
		assertMatch(t, true, "a\\b", "a/*", win)
		// slashes-windows.js line 188
		assertMatch(t, true, "a\\c", "a/*", win)
		// slashes-windows.js line 189
		assertMatch(t, true, "a\\x", "a/*", win)
		// slashes-windows.js line 190
		assertMatch(t, false, "a\\a\\a", "a/*", win)
		// slashes-windows.js line 191
		assertMatch(t, false, "a\\a\\b", "a/*", win)
		// slashes-windows.js line 192
		assertMatch(t, false, "a\\a\\a\\a", "a/*", win)
		// slashes-windows.js line 193
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*", win)
		// slashes-windows.js line 194
		assertMatch(t, false, "x\\y", "a/*", win)
		// slashes-windows.js line 195
		assertMatch(t, false, "z\\z", "a/*", win)

		// Pattern: a/*/*
		// slashes-windows.js line 197
		assertMatch(t, false, "a", "a/*/*", win)
		// slashes-windows.js line 198
		assertMatch(t, false, "b", "a/*/*", win)
		// slashes-windows.js line 199
		assertMatch(t, false, "a\\a", "a/*/*", win)
		// slashes-windows.js line 200
		assertMatch(t, false, "a\\b", "a/*/*", win)
		// slashes-windows.js line 201
		assertMatch(t, false, "a\\c", "a/*/*", win)
		// slashes-windows.js line 202
		assertMatch(t, false, "a\\x", "a/*/*", win)
		// slashes-windows.js line 203
		assertMatch(t, true, "a\\a\\a", "a/*/*", win)
		// slashes-windows.js line 204
		assertMatch(t, true, "a\\a\\b", "a/*/*", win)
		// slashes-windows.js line 205
		assertMatch(t, false, "a\\a\\a\\a", "a/*/*", win)
		// slashes-windows.js line 206
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/*", win)
		// slashes-windows.js line 207
		assertMatch(t, false, "x\\y", "a/*/*", win)
		// slashes-windows.js line 208
		assertMatch(t, false, "z\\z", "a/*/*", win)

		// Pattern: a/*/*/*
		// slashes-windows.js line 210
		assertMatch(t, false, "a", "a/*/*/*", win)
		// slashes-windows.js line 211
		assertMatch(t, false, "b", "a/*/*/*", win)
		// slashes-windows.js line 212
		assertMatch(t, false, "a\\a", "a/*/*/*", win)
		// slashes-windows.js line 213
		assertMatch(t, false, "a\\b", "a/*/*/*", win)
		// slashes-windows.js line 214
		assertMatch(t, false, "a\\c", "a/*/*/*", win)
		// slashes-windows.js line 215
		assertMatch(t, false, "a\\x", "a/*/*/*", win)
		// slashes-windows.js line 216
		assertMatch(t, false, "a\\a\\a", "a/*/*/*", win)
		// slashes-windows.js line 217
		assertMatch(t, false, "a\\a\\b", "a/*/*/*", win)
		// slashes-windows.js line 218
		assertMatch(t, true, "a\\a\\a\\a", "a/*/*/*", win)
		// slashes-windows.js line 219
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/*/*", win)
		// slashes-windows.js line 220
		assertMatch(t, false, "x\\y", "a/*/*/*", win)
		// slashes-windows.js line 221
		assertMatch(t, false, "z\\z", "a/*/*/*", win)

		// Pattern: a/*/*/*/*
		// slashes-windows.js line 223
		assertMatch(t, false, "a", "a/*/*/*/*", win)
		// slashes-windows.js line 224
		assertMatch(t, false, "b", "a/*/*/*/*", win)
		// slashes-windows.js line 225
		assertMatch(t, false, "a\\a", "a/*/*/*/*", win)
		// slashes-windows.js line 226
		assertMatch(t, false, "a\\b", "a/*/*/*/*", win)
		// slashes-windows.js line 227
		assertMatch(t, false, "a\\c", "a/*/*/*/*", win)
		// slashes-windows.js line 228
		assertMatch(t, false, "a\\x", "a/*/*/*/*", win)
		// slashes-windows.js line 229
		assertMatch(t, false, "a\\a\\a", "a/*/*/*/*", win)
		// slashes-windows.js line 230
		assertMatch(t, false, "a\\a\\b", "a/*/*/*/*", win)
		// slashes-windows.js line 231
		assertMatch(t, false, "a\\a\\a\\a", "a/*/*/*/*", win)
		// slashes-windows.js line 232
		assertMatch(t, true, "a\\a\\a\\a\\a", "a/*/*/*/*", win)
		// slashes-windows.js line 233
		assertMatch(t, false, "x\\y", "a/*/*/*/*", win)
		// slashes-windows.js line 234
		assertMatch(t, false, "z\\z", "a/*/*/*/*", win)

		// Pattern: a/*/a
		// slashes-windows.js line 236
		assertMatch(t, false, "a", "a/*/a", win)
		// slashes-windows.js line 237
		assertMatch(t, false, "b", "a/*/a", win)
		// slashes-windows.js line 238
		assertMatch(t, false, "a\\a", "a/*/a", win)
		// slashes-windows.js line 239
		assertMatch(t, false, "a\\b", "a/*/a", win)
		// slashes-windows.js line 240
		assertMatch(t, false, "a\\c", "a/*/a", win)
		// slashes-windows.js line 241
		assertMatch(t, false, "a\\x", "a/*/a", win)
		// slashes-windows.js line 242
		assertMatch(t, true, "a\\a\\a", "a/*/a", win)
		// slashes-windows.js line 243
		assertMatch(t, false, "a\\a\\b", "a/*/a", win)
		// slashes-windows.js line 244
		assertMatch(t, false, "a\\a\\a\\a", "a/*/a", win)
		// slashes-windows.js line 245
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/a", win)
		// slashes-windows.js line 246
		assertMatch(t, false, "x\\y", "a/*/a", win)
		// slashes-windows.js line 247
		assertMatch(t, false, "z\\z", "a/*/a", win)

		// Pattern: a/*/b
		// slashes-windows.js line 249
		assertMatch(t, false, "a", "a/*/b", win)
		// slashes-windows.js line 250
		assertMatch(t, false, "b", "a/*/b", win)
		// slashes-windows.js line 251
		assertMatch(t, false, "a\\a", "a/*/b", win)
		// slashes-windows.js line 252
		assertMatch(t, false, "a\\b", "a/*/b", win)
		// slashes-windows.js line 253
		assertMatch(t, false, "a\\c", "a/*/b", win)
		// slashes-windows.js line 254
		assertMatch(t, false, "a\\x", "a/*/b", win)
		// slashes-windows.js line 255
		assertMatch(t, false, "a\\a\\a", "a/*/b", win)
		// slashes-windows.js line 256
		assertMatch(t, true, "a\\a\\b", "a/*/b", win)
		// slashes-windows.js line 257
		assertMatch(t, false, "a\\a\\a\\a", "a/*/b", win)
		// slashes-windows.js line 258
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/b", win)
		// slashes-windows.js line 259
		assertMatch(t, false, "x\\y", "a/*/b", win)
		// slashes-windows.js line 260
		assertMatch(t, false, "z\\z", "a/*/b", win)

		// Same patterns with windows: false — all should be false because backslashes are not path separators
		// Pattern: */* with windows: false
		// slashes-windows.js line 262
		assertMatch(t, false, "a", "*/*", nowin)
		// slashes-windows.js line 263
		assertMatch(t, false, "b", "*/*", nowin)
		// slashes-windows.js line 264
		assertMatch(t, false, "a\\a", "*/*", nowin)
		// slashes-windows.js line 265
		assertMatch(t, false, "a\\b", "*/*", nowin)
		// slashes-windows.js line 266
		assertMatch(t, false, "a\\c", "*/*", nowin)
		// slashes-windows.js line 267
		assertMatch(t, false, "a\\x", "*/*", nowin)
		// slashes-windows.js line 268
		assertMatch(t, false, "a\\a\\a", "*/*", nowin)
		// slashes-windows.js line 269
		assertMatch(t, false, "a\\a\\b", "*/*", nowin)
		// slashes-windows.js line 270
		assertMatch(t, false, "a\\a\\a\\a", "*/*", nowin)
		// slashes-windows.js line 271
		assertMatch(t, false, "a\\a\\a\\a\\a", "*/*", nowin)
		// slashes-windows.js line 272
		assertMatch(t, false, "x\\y", "*/*", nowin)
		// slashes-windows.js line 273
		assertMatch(t, false, "z\\z", "*/*", nowin)

		// Pattern: */*/* with windows: false
		// slashes-windows.js line 275
		assertMatch(t, false, "a", "*/*/*", nowin)
		// slashes-windows.js line 276
		assertMatch(t, false, "b", "*/*/*", nowin)
		// slashes-windows.js line 277
		assertMatch(t, false, "a\\a", "*/*/*", nowin)
		// slashes-windows.js line 278
		assertMatch(t, false, "a\\b", "*/*/*", nowin)
		// slashes-windows.js line 279
		assertMatch(t, false, "a\\c", "*/*/*", nowin)
		// slashes-windows.js line 280
		assertMatch(t, false, "a\\x", "*/*/*", nowin)
		// slashes-windows.js line 281
		assertMatch(t, false, "a\\a\\a", "*/*/*", nowin)
		// slashes-windows.js line 282
		assertMatch(t, false, "a\\a\\b", "*/*/*", nowin)
		// slashes-windows.js line 283
		assertMatch(t, false, "a\\a\\a\\a", "*/*/*", nowin)
		// slashes-windows.js line 284
		assertMatch(t, false, "a\\a\\a\\a\\a", "*/*/*", nowin)
		// slashes-windows.js line 285
		assertMatch(t, false, "x\\y", "*/*/*", nowin)
		// slashes-windows.js line 286
		assertMatch(t, false, "z\\z", "*/*/*", nowin)

		// Pattern: */*/*/* with windows: false
		// slashes-windows.js line 288
		assertMatch(t, false, "a", "*/*/*/*", nowin)
		// slashes-windows.js line 289
		assertMatch(t, false, "b", "*/*/*/*", nowin)
		// slashes-windows.js line 290
		assertMatch(t, false, "a\\a", "*/*/*/*", nowin)
		// slashes-windows.js line 291
		assertMatch(t, false, "a\\b", "*/*/*/*", nowin)
		// slashes-windows.js line 292
		assertMatch(t, false, "a\\c", "*/*/*/*", nowin)
		// slashes-windows.js line 293
		assertMatch(t, false, "a\\x", "*/*/*/*", nowin)
		// slashes-windows.js line 294
		assertMatch(t, false, "a\\a\\a", "*/*/*/*", nowin)
		// slashes-windows.js line 295
		assertMatch(t, false, "a\\a\\b", "*/*/*/*", nowin)
		// slashes-windows.js line 296
		assertMatch(t, false, "a\\a\\a\\a", "*/*/*/*", nowin)
		// slashes-windows.js line 297
		assertMatch(t, false, "a\\a\\a\\a\\a", "*/*/*/*", nowin)
		// slashes-windows.js line 298
		assertMatch(t, false, "x\\y", "*/*/*/*", nowin)
		// slashes-windows.js line 299
		assertMatch(t, false, "z\\z", "*/*/*/*", nowin)

		// Pattern: */*/*/*/* with windows: false
		// slashes-windows.js line 301
		assertMatch(t, false, "a", "*/*/*/*/*", nowin)
		// slashes-windows.js line 302
		assertMatch(t, false, "b", "*/*/*/*/*", nowin)
		// slashes-windows.js line 303
		assertMatch(t, false, "a\\a", "*/*/*/*/*", nowin)
		// slashes-windows.js line 304
		assertMatch(t, false, "a\\b", "*/*/*/*/*", nowin)
		// slashes-windows.js line 305
		assertMatch(t, false, "a\\c", "*/*/*/*/*", nowin)
		// slashes-windows.js line 306
		assertMatch(t, false, "a\\x", "*/*/*/*/*", nowin)
		// slashes-windows.js line 307
		assertMatch(t, false, "a\\a\\a", "*/*/*/*/*", nowin)
		// slashes-windows.js line 308
		assertMatch(t, false, "a\\a\\b", "*/*/*/*/*", nowin)
		// slashes-windows.js line 309
		assertMatch(t, false, "a\\a\\a\\a", "*/*/*/*/*", nowin)
		// slashes-windows.js line 310
		assertMatch(t, false, "a\\a\\a\\a\\a", "*/*/*/*/*", nowin)
		// slashes-windows.js line 311
		assertMatch(t, false, "x\\y", "*/*/*/*/*", nowin)
		// slashes-windows.js line 312
		assertMatch(t, false, "z\\z", "*/*/*/*/*", nowin)

		// Pattern: a/* with windows: false
		// slashes-windows.js line 314
		assertMatch(t, false, "a", "a/*", nowin)
		// slashes-windows.js line 315
		assertMatch(t, false, "b", "a/*", nowin)
		// slashes-windows.js line 316
		assertMatch(t, false, "a\\a", "a/*", nowin)
		// slashes-windows.js line 317
		assertMatch(t, false, "a\\b", "a/*", nowin)
		// slashes-windows.js line 318
		assertMatch(t, false, "a\\c", "a/*", nowin)
		// slashes-windows.js line 319
		assertMatch(t, false, "a\\x", "a/*", nowin)
		// slashes-windows.js line 320
		assertMatch(t, false, "a\\a\\a", "a/*", nowin)
		// slashes-windows.js line 321
		assertMatch(t, false, "a\\a\\b", "a/*", nowin)
		// slashes-windows.js line 322
		assertMatch(t, false, "a\\a\\a\\a", "a/*", nowin)
		// slashes-windows.js line 323
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*", nowin)
		// slashes-windows.js line 324
		assertMatch(t, false, "x\\y", "a/*", nowin)
		// slashes-windows.js line 325
		assertMatch(t, false, "z\\z", "a/*", nowin)

		// Pattern: a/*/* with windows: false
		// slashes-windows.js line 327
		assertMatch(t, false, "a", "a/*/*", nowin)
		// slashes-windows.js line 328
		assertMatch(t, false, "b", "a/*/*", nowin)
		// slashes-windows.js line 329
		assertMatch(t, false, "a\\a", "a/*/*", nowin)
		// slashes-windows.js line 330
		assertMatch(t, false, "a\\b", "a/*/*", nowin)
		// slashes-windows.js line 331
		assertMatch(t, false, "a\\c", "a/*/*", nowin)
		// slashes-windows.js line 332
		assertMatch(t, false, "a\\x", "a/*/*", nowin)
		// slashes-windows.js line 333
		assertMatch(t, false, "a\\a\\a", "a/*/*", nowin)
		// slashes-windows.js line 334
		assertMatch(t, false, "a\\a\\b", "a/*/*", nowin)
		// slashes-windows.js line 335
		assertMatch(t, false, "a\\a\\a\\a", "a/*/*", nowin)
		// slashes-windows.js line 336
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/*", nowin)
		// slashes-windows.js line 337
		assertMatch(t, false, "x\\y", "a/*/*", nowin)
		// slashes-windows.js line 338
		assertMatch(t, false, "z\\z", "a/*/*", nowin)

		// Pattern: a/*/*/* with windows: false
		// slashes-windows.js line 340
		assertMatch(t, false, "a", "a/*/*/*", nowin)
		// slashes-windows.js line 341
		assertMatch(t, false, "b", "a/*/*/*", nowin)
		// slashes-windows.js line 342
		assertMatch(t, false, "a\\a", "a/*/*/*", nowin)
		// slashes-windows.js line 343
		assertMatch(t, false, "a\\b", "a/*/*/*", nowin)
		// slashes-windows.js line 344
		assertMatch(t, false, "a\\c", "a/*/*/*", nowin)
		// slashes-windows.js line 345
		assertMatch(t, false, "a\\x", "a/*/*/*", nowin)
		// slashes-windows.js line 346
		assertMatch(t, false, "a\\a\\a", "a/*/*/*", nowin)
		// slashes-windows.js line 347
		assertMatch(t, false, "a\\a\\b", "a/*/*/*", nowin)
		// slashes-windows.js line 348
		assertMatch(t, false, "a\\a\\a\\a", "a/*/*/*", nowin)
		// slashes-windows.js line 349
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/*/*", nowin)
		// slashes-windows.js line 350
		assertMatch(t, false, "x\\y", "a/*/*/*", nowin)
		// slashes-windows.js line 351
		assertMatch(t, false, "z\\z", "a/*/*/*", nowin)

		// Pattern: a/*/*/*/* with windows: false
		// slashes-windows.js line 353
		assertMatch(t, false, "a", "a/*/*/*/*", nowin)
		// slashes-windows.js line 354
		assertMatch(t, false, "b", "a/*/*/*/*", nowin)
		// slashes-windows.js line 355
		assertMatch(t, false, "a\\a", "a/*/*/*/*", nowin)
		// slashes-windows.js line 356
		assertMatch(t, false, "a\\b", "a/*/*/*/*", nowin)
		// slashes-windows.js line 357
		assertMatch(t, false, "a\\c", "a/*/*/*/*", nowin)
		// slashes-windows.js line 358
		assertMatch(t, false, "a\\x", "a/*/*/*/*", nowin)
		// slashes-windows.js line 359
		assertMatch(t, false, "a\\a\\a", "a/*/*/*/*", nowin)
		// slashes-windows.js line 360
		assertMatch(t, false, "a\\a\\b", "a/*/*/*/*", nowin)
		// slashes-windows.js line 361
		assertMatch(t, false, "a\\a\\a\\a", "a/*/*/*/*", nowin)
		// slashes-windows.js line 362
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/*/*/*", nowin)
		// slashes-windows.js line 363
		assertMatch(t, false, "x\\y", "a/*/*/*/*", nowin)
		// slashes-windows.js line 364
		assertMatch(t, false, "z\\z", "a/*/*/*/*", nowin)

		// Pattern: a/*/a with windows: false
		// slashes-windows.js line 366
		assertMatch(t, false, "a", "a/*/a", nowin)
		// slashes-windows.js line 367
		assertMatch(t, false, "b", "a/*/a", nowin)
		// slashes-windows.js line 368
		assertMatch(t, false, "a\\a", "a/*/a", nowin)
		// slashes-windows.js line 369
		assertMatch(t, false, "a\\b", "a/*/a", nowin)
		// slashes-windows.js line 370
		assertMatch(t, false, "a\\c", "a/*/a", nowin)
		// slashes-windows.js line 371
		assertMatch(t, false, "a\\x", "a/*/a", nowin)
		// slashes-windows.js line 372
		assertMatch(t, false, "a\\a\\a", "a/*/a", nowin)
		// slashes-windows.js line 373
		assertMatch(t, false, "a\\a\\b", "a/*/a", nowin)
		// slashes-windows.js line 374
		assertMatch(t, false, "a\\a\\a\\a", "a/*/a", nowin)
		// slashes-windows.js line 375
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/a", nowin)
		// slashes-windows.js line 376
		assertMatch(t, false, "x\\y", "a/*/a", nowin)
		// slashes-windows.js line 377
		assertMatch(t, false, "z\\z", "a/*/a", nowin)

		// Pattern: a/*/b with windows: false
		// slashes-windows.js line 379
		assertMatch(t, false, "a", "a/*/b", nowin)
		// slashes-windows.js line 380
		assertMatch(t, false, "b", "a/*/b", nowin)
		// slashes-windows.js line 381
		assertMatch(t, false, "a\\a", "a/*/b", nowin)
		// slashes-windows.js line 382
		assertMatch(t, false, "a\\b", "a/*/b", nowin)
		// slashes-windows.js line 383
		assertMatch(t, false, "a\\c", "a/*/b", nowin)
		// slashes-windows.js line 384
		assertMatch(t, false, "a\\x", "a/*/b", nowin)
		// slashes-windows.js line 385
		assertMatch(t, false, "a\\a\\a", "a/*/b", nowin)
		// slashes-windows.js line 386
		assertMatch(t, false, "a\\a\\b", "a/*/b", nowin)
		// slashes-windows.js line 387
		assertMatch(t, false, "a\\a\\a\\a", "a/*/b", nowin)
		// slashes-windows.js line 388
		assertMatch(t, false, "a\\a\\a\\a\\a", "a/*/b", nowin)
		// slashes-windows.js line 389
		assertMatch(t, false, "x\\y", "a/*/b", nowin)
		// slashes-windows.js line 390
		assertMatch(t, false, "z\\z", "a/*/b", nowin)
	})

	t.Run("should support globstars", func(t *testing.T) {
		// slashes-windows.js line 394
		assertMatch(t, true, "a\\a", "a/**", win)
		// slashes-windows.js line 395
		assertMatch(t, true, "a\\b", "a/**", win)
		// slashes-windows.js line 396
		assertMatch(t, true, "a\\c", "a/**", win)
		// slashes-windows.js line 397
		assertMatch(t, true, "a\\x", "a/**", win)
		// slashes-windows.js line 398
		assertMatch(t, true, "a\\x\\y", "a/**", win)
		// slashes-windows.js line 399
		assertMatch(t, true, "a\\x\\y\\z", "a/**", win)

		// slashes-windows.js line 401
		assertMatch(t, true, "a\\a", "a/**/*", win)
		// slashes-windows.js line 402
		assertMatch(t, true, "a\\b", "a/**/*", win)
		// slashes-windows.js line 403
		assertMatch(t, true, "a\\c", "a/**/*", win)
		// slashes-windows.js line 404
		assertMatch(t, true, "a\\x", "a/**/*", win)
		// slashes-windows.js line 405
		assertMatch(t, true, "a\\x\\y", "a/**/*", win)
		// slashes-windows.js line 406
		assertMatch(t, true, "a\\x\\y\\z", "a/**/*", win)

		// slashes-windows.js line 408
		assertMatch(t, true, "a\\a", "a/**/**/*", win)
		// slashes-windows.js line 409
		assertMatch(t, true, "a\\b", "a/**/**/*", win)
		// slashes-windows.js line 410
		assertMatch(t, true, "a\\c", "a/**/**/*", win)
		// slashes-windows.js line 411
		assertMatch(t, true, "a\\x", "a/**/**/*", win)
		// slashes-windows.js line 412
		assertMatch(t, true, "a\\x\\y", "a/**/**/*", win)
		// slashes-windows.js line 413
		assertMatch(t, true, "a\\x\\y\\z", "a/**/**/*", win)
	})

	t.Run("should not match backslashes with globstars when disabled", func(t *testing.T) {
		// slashes-windows.js line 417
		assertMatch(t, false, "a\\a", "a/**", nowin)
		// slashes-windows.js line 418
		assertMatch(t, false, "a\\b", "a/**", nowin)
		// slashes-windows.js line 419
		assertMatch(t, false, "a\\c", "a/**", nowin)
		// slashes-windows.js line 420
		assertMatch(t, false, "a\\x", "a/**", nowin)
		// slashes-windows.js line 421
		assertMatch(t, false, "a\\x\\y", "a/**", nowin)
		// slashes-windows.js line 422
		assertMatch(t, false, "a\\x\\y\\z", "a/**", nowin)

		// slashes-windows.js line 424
		assertMatch(t, false, "a\\a", "a/**/*", nowin)
		// slashes-windows.js line 425
		assertMatch(t, false, "a\\b", "a/**/*", nowin)
		// slashes-windows.js line 426
		assertMatch(t, false, "a\\c", "a/**/*", nowin)
		// slashes-windows.js line 427
		assertMatch(t, false, "a\\x", "a/**/*", nowin)
		// slashes-windows.js line 428
		assertMatch(t, false, "a\\x\\y", "a/**/*", nowin)
		// slashes-windows.js line 429
		assertMatch(t, false, "a\\x\\y\\z", "a/**/*", nowin)

		// slashes-windows.js line 431
		assertMatch(t, false, "a\\a", "a/**/**/*", nowin)
		// slashes-windows.js line 432
		assertMatch(t, false, "a\\b", "a/**/**/*", nowin)
		// slashes-windows.js line 433
		assertMatch(t, false, "a\\c", "a/**/**/*", nowin)
		// slashes-windows.js line 434
		assertMatch(t, false, "a\\x", "a/**/**/*", nowin)
		// slashes-windows.js line 435
		assertMatch(t, false, "a\\x\\y", "a/**/**/*", nowin)
		// slashes-windows.js line 436
		assertMatch(t, false, "a\\x\\y\\z", "a/**/**/*", nowin)
	})

	t.Run("should work with file extensions", func(t *testing.T) {
		// slashes-windows.js line 440
		assertMatch(t, true, "a.txt", "a*.txt", win)
		// slashes-windows.js line 441
		assertMatch(t, false, "a\\b.txt", "a*.txt", win)
		// slashes-windows.js line 442
		assertMatch(t, false, "a\\x\\y.txt", "a*.txt", win)
		// slashes-windows.js line 443
		assertMatch(t, false, "a\\x\\y\\z", "a*.txt", win)

		// slashes-windows.js line 445
		assertMatch(t, true, "a.txt", "a.txt", win)
		// slashes-windows.js line 446
		assertMatch(t, false, "a\\b.txt", "a.txt", win)
		// slashes-windows.js line 447
		assertMatch(t, false, "a\\x\\y.txt", "a.txt", win)
		// slashes-windows.js line 448
		assertMatch(t, false, "a\\x\\y\\z", "a.txt", win)

		// slashes-windows.js line 450
		assertMatch(t, false, "a.txt", "a/**/*.txt", win)
		// slashes-windows.js line 451
		assertMatch(t, true, "a\\b.txt", "a/**/*.txt", win)
		// slashes-windows.js line 452
		assertMatch(t, true, "a\\x\\y.txt", "a/**/*.txt", win)
		// slashes-windows.js line 453
		assertMatch(t, false, "a\\x\\y\\z", "a/**/*.txt", win)

		// slashes-windows.js line 455
		assertMatch(t, false, "a.txt", "a/**/*.txt", nowin)
		// slashes-windows.js line 456
		assertMatch(t, false, "a\\b.txt", "a/**/*.txt", nowin)
		// slashes-windows.js line 457
		assertMatch(t, false, "a\\x\\y.txt", "a/**/*.txt", nowin)
		// slashes-windows.js line 458
		assertMatch(t, false, "a\\x\\y\\z", "a/**/*.txt", nowin)

		// slashes-windows.js line 460
		assertMatch(t, false, "a.txt", "a/*.txt", win)
		// slashes-windows.js line 461
		assertMatch(t, true, "a\\b.txt", "a/*.txt", win)
		// slashes-windows.js line 462
		assertMatch(t, false, "a\\x\\y.txt", "a/*.txt", win)
		// slashes-windows.js line 463
		assertMatch(t, false, "a\\x\\y\\z", "a/*.txt", win)

		// slashes-windows.js line 465
		assertMatch(t, false, "a.txt", "a/*.txt", nowin)
		// slashes-windows.js line 466
		assertMatch(t, false, "a\\b.txt", "a/*.txt", nowin)
		// slashes-windows.js line 467
		assertMatch(t, false, "a\\x\\y.txt", "a/*.txt", nowin)
		// slashes-windows.js line 468
		assertMatch(t, false, "a\\x\\y\\z", "a/*.txt", nowin)

		// slashes-windows.js line 470
		assertMatch(t, false, "a.txt", "a/*/*.txt", win)
		// slashes-windows.js line 471
		assertMatch(t, false, "a\\b.txt", "a/*/*.txt", win)
		// slashes-windows.js line 472
		assertMatch(t, true, "a\\x\\y.txt", "a/*/*.txt", win)
		// slashes-windows.js line 473
		assertMatch(t, false, "a\\x\\y\\z", "a/*/*.txt", win)

		// slashes-windows.js line 475
		assertMatch(t, false, "a.txt", "a/*/*.txt", nowin)
		// slashes-windows.js line 476
		assertMatch(t, false, "a\\b.txt", "a/*/*.txt", nowin)
		// slashes-windows.js line 477
		assertMatch(t, false, "a\\x\\y.txt", "a/*/*.txt", nowin)
		// slashes-windows.js line 478
		assertMatch(t, false, "a\\x\\y\\z", "a/*/*.txt", nowin)
	})

	t.Run("should support negation patterns", func(t *testing.T) {
		// slashes-windows.js line 482
		assertMatch(t, true, "a", "!a/b", win)
		// slashes-windows.js line 483
		assertMatch(t, true, "a\\a", "!a/b", win)
		// slashes-windows.js line 484
		assertMatch(t, false, "a\\b", "!a/b", win)
		// slashes-windows.js line 485
		assertMatch(t, true, "a\\c", "!a/b", win)
		// slashes-windows.js line 486
		assertMatch(t, true, "b\\a", "!a/b", win)
		// slashes-windows.js line 487
		assertMatch(t, true, "b\\b", "!a/b", win)
		// slashes-windows.js line 488
		assertMatch(t, true, "b\\c", "!a/b", win)

		// slashes-windows.js line 490
		assertMatch(t, true, "a", "!*/c", win)
		// slashes-windows.js line 491
		assertMatch(t, true, "a\\a", "!*/c", win)
		// slashes-windows.js line 492
		assertMatch(t, true, "a\\b", "!*/c", win)
		// slashes-windows.js line 493
		assertMatch(t, false, "a\\c", "!*/c", win)
		// slashes-windows.js line 494
		assertMatch(t, true, "b\\a", "!*/c", win)
		// slashes-windows.js line 495
		assertMatch(t, true, "b\\b", "!*/c", win)
		// slashes-windows.js line 496
		assertMatch(t, false, "b\\c", "!*/c", win)

		// slashes-windows.js line 498
		assertMatch(t, true, "a", "!a/b", win)
		// slashes-windows.js line 499
		assertMatch(t, true, "a\\a", "!a/b", win)
		// slashes-windows.js line 500
		assertMatch(t, false, "a\\b", "!a/b", win)
		// slashes-windows.js line 501
		assertMatch(t, true, "a\\c", "!a/b", win)
		// slashes-windows.js line 502
		assertMatch(t, true, "b\\a", "!a/b", win)
		// slashes-windows.js line 503
		assertMatch(t, true, "b\\b", "!a/b", win)
		// slashes-windows.js line 504
		assertMatch(t, true, "b\\c", "!a/b", win)

		// slashes-windows.js line 506
		assertMatch(t, true, "a", "!*/c", win)
		// slashes-windows.js line 507
		assertMatch(t, true, "a\\a", "!*/c", win)
		// slashes-windows.js line 508
		assertMatch(t, true, "a\\b", "!*/c", win)
		// slashes-windows.js line 509
		assertMatch(t, false, "a\\c", "!*/c", win)
		// slashes-windows.js line 510
		assertMatch(t, true, "b\\a", "!*/c", win)
		// slashes-windows.js line 511
		assertMatch(t, true, "b\\b", "!*/c", win)
		// slashes-windows.js line 512
		assertMatch(t, false, "b\\c", "!*/c", win)

		// slashes-windows.js line 514
		assertMatch(t, true, "a", "!a/(b)", win)
		// slashes-windows.js line 515
		assertMatch(t, true, "a\\a", "!a/(b)", win)
		// slashes-windows.js line 516
		assertMatch(t, false, "a\\b", "!a/(b)", win)
		// slashes-windows.js line 517
		assertMatch(t, true, "a\\c", "!a/(b)", win)
		// slashes-windows.js line 518
		assertMatch(t, true, "b\\a", "!a/(b)", win)
		// slashes-windows.js line 519
		assertMatch(t, true, "b\\b", "!a/(b)", win)
		// slashes-windows.js line 520
		assertMatch(t, true, "b\\c", "!a/(b)", win)

		// slashes-windows.js line 522
		assertMatch(t, true, "a", "!(a/b)", win)
		// slashes-windows.js line 523
		assertMatch(t, true, "a\\a", "!(a/b)", win)
		// slashes-windows.js line 524
		assertMatch(t, false, "a\\b", "!(a/b)", win)
		// slashes-windows.js line 525
		assertMatch(t, true, "a\\c", "!(a/b)", win)
		// slashes-windows.js line 526
		assertMatch(t, true, "b\\a", "!(a/b)", win)
		// slashes-windows.js line 527
		assertMatch(t, true, "b\\b", "!(a/b)", win)
		// slashes-windows.js line 528
		assertMatch(t, true, "b\\c", "!(a/b)", win)
	})
}
