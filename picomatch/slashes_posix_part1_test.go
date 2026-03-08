package picomatch

// slashes_posix_part1_test.go — Ported from picomatch/test/slashes-posix.js lines 1-600
// Tests slash handling on POSIX systems.

import (
	"testing"
)

func TestSlashesPosixPart1(t *testing.T) {
	t.Run("should match a literal string", func(t *testing.T) {
		assertMatch(t, false, "a/a", "(a/b)")  // slashes-posix.js:8
		assertMatch(t, true, "a/b", "(a/b)")   // slashes-posix.js:9
		assertMatch(t, false, "a/c", "(a/b)")  // slashes-posix.js:10
		assertMatch(t, false, "b/a", "(a/b)")  // slashes-posix.js:11
		assertMatch(t, false, "b/b", "(a/b)")  // slashes-posix.js:12
		assertMatch(t, false, "b/c", "(a/b)")  // slashes-posix.js:13

		assertMatch(t, false, "a/a", "a/b") // slashes-posix.js:15
		assertMatch(t, true, "a/b", "a/b")  // slashes-posix.js:16
		assertMatch(t, false, "a/c", "a/b") // slashes-posix.js:17
		assertMatch(t, false, "b/a", "a/b") // slashes-posix.js:18
		assertMatch(t, false, "b/b", "a/b") // slashes-posix.js:19
		assertMatch(t, false, "b/c", "a/b") // slashes-posix.js:20
	})

	t.Run("should match an array of literal strings", func(t *testing.T) {
		assertMatch(t, false, "a/a", "a/b") // slashes-posix.js:24
		assertMatch(t, true, "a/b", "a/b")  // slashes-posix.js:25
		assertMatch(t, false, "a/c", "a/b") // slashes-posix.js:26
		assertMatch(t, false, "b/a", "a/b") // slashes-posix.js:27
		assertMatch(t, false, "b/b", "a/b") // slashes-posix.js:28
		assertMatch(t, true, "b/b", "b/b")  // slashes-posix.js:29
		assertMatch(t, false, "b/c", "a/b") // slashes-posix.js:30
	})

	t.Run("should support regex logical or", func(t *testing.T) {
		assertMatch(t, true, "a/a", "a/(a|c)")   // slashes-posix.js:34
		assertMatch(t, false, "a/b", "a/(a|c)")  // slashes-posix.js:35
		assertMatch(t, true, "a/c", "a/(a|c)")   // slashes-posix.js:36

		assertMatch(t, true, "a/a", "a/(a|b|c)") // slashes-posix.js:38
		assertMatch(t, true, "a/b", "a/(a|b|c)") // slashes-posix.js:39
		assertMatch(t, true, "a/c", "a/(a|b|c)") // slashes-posix.js:40
	})

	t.Run("should support regex ranges", func(t *testing.T) {
		assertMatch(t, false, "a/a", "a/[b-c]")   // slashes-posix.js:44
		assertMatch(t, true, "a/b", "a/[b-c]")    // slashes-posix.js:45
		assertMatch(t, true, "a/c", "a/[b-c]")    // slashes-posix.js:46

		assertMatch(t, true, "a/a", "a/[a-z]")    // slashes-posix.js:48
		assertMatch(t, true, "a/b", "a/[a-z]")    // slashes-posix.js:49
		assertMatch(t, true, "a/c", "a/[a-z]")    // slashes-posix.js:50
		assertMatch(t, false, "a/x/y", "a/[a-z]") // slashes-posix.js:51
		assertMatch(t, true, "a/x", "a/[a-z]")    // slashes-posix.js:52
	})

	t.Run("should support single globs (*)", func(t *testing.T) {
		// Pattern: *
		assertMatch(t, true, "a", "*")           // slashes-posix.js:56
		assertMatch(t, true, "b", "*")           // slashes-posix.js:57
		assertMatch(t, false, "a/a", "*")        // slashes-posix.js:58
		assertMatch(t, false, "a/b", "*")        // slashes-posix.js:59
		assertMatch(t, false, "a/c", "*")        // slashes-posix.js:60
		assertMatch(t, false, "a/x", "*")        // slashes-posix.js:61
		assertMatch(t, false, "a/a/a", "*")      // slashes-posix.js:62
		assertMatch(t, false, "a/a/b", "*")      // slashes-posix.js:63
		assertMatch(t, false, "a/a/a/a", "*")    // slashes-posix.js:64
		assertMatch(t, false, "a/a/a/a/a", "*")  // slashes-posix.js:65
		assertMatch(t, false, "x/y", "*")        // slashes-posix.js:66
		assertMatch(t, false, "z/z", "*")        // slashes-posix.js:67

		// Pattern: */*
		assertMatch(t, false, "a", "*/*")          // slashes-posix.js:69
		assertMatch(t, false, "b", "*/*")          // slashes-posix.js:70
		assertMatch(t, true, "a/a", "*/*")         // slashes-posix.js:71
		assertMatch(t, true, "a/b", "*/*")         // slashes-posix.js:72
		assertMatch(t, true, "a/c", "*/*")         // slashes-posix.js:73
		assertMatch(t, true, "a/x", "*/*")         // slashes-posix.js:74
		assertMatch(t, false, "a/a/a", "*/*")      // slashes-posix.js:75
		assertMatch(t, false, "a/a/b", "*/*")      // slashes-posix.js:76
		assertMatch(t, false, "a/a/a/a", "*/*")    // slashes-posix.js:77
		assertMatch(t, false, "a/a/a/a/a", "*/*")  // slashes-posix.js:78
		assertMatch(t, true, "x/y", "*/*")         // slashes-posix.js:79
		assertMatch(t, true, "z/z", "*/*")         // slashes-posix.js:80

		// Pattern: */*/*
		assertMatch(t, false, "a", "*/*/*")          // slashes-posix.js:82
		assertMatch(t, false, "b", "*/*/*")          // slashes-posix.js:83
		assertMatch(t, false, "a/a", "*/*/*")        // slashes-posix.js:84
		assertMatch(t, false, "a/b", "*/*/*")        // slashes-posix.js:85
		assertMatch(t, false, "a/c", "*/*/*")        // slashes-posix.js:86
		assertMatch(t, false, "a/x", "*/*/*")        // slashes-posix.js:87
		assertMatch(t, true, "a/a/a", "*/*/*")       // slashes-posix.js:88
		assertMatch(t, true, "a/a/b", "*/*/*")       // slashes-posix.js:89
		assertMatch(t, false, "a/a/a/a", "*/*/*")    // slashes-posix.js:90
		assertMatch(t, false, "a/a/a/a/a", "*/*/*")  // slashes-posix.js:91
		assertMatch(t, false, "x/y", "*/*/*")        // slashes-posix.js:92
		assertMatch(t, false, "z/z", "*/*/*")        // slashes-posix.js:93

		// Pattern: */*/*/*
		assertMatch(t, false, "a", "*/*/*/*")          // slashes-posix.js:95
		assertMatch(t, false, "b", "*/*/*/*")          // slashes-posix.js:96
		assertMatch(t, false, "a/a", "*/*/*/*")        // slashes-posix.js:97
		assertMatch(t, false, "a/b", "*/*/*/*")        // slashes-posix.js:98
		assertMatch(t, false, "a/c", "*/*/*/*")        // slashes-posix.js:99
		assertMatch(t, false, "a/x", "*/*/*/*")        // slashes-posix.js:100
		assertMatch(t, false, "a/a/a", "*/*/*/*")      // slashes-posix.js:101
		assertMatch(t, false, "a/a/b", "*/*/*/*")      // slashes-posix.js:102
		assertMatch(t, true, "a/a/a/a", "*/*/*/*")     // slashes-posix.js:103
		assertMatch(t, false, "a/a/a/a/a", "*/*/*/*")  // slashes-posix.js:104
		assertMatch(t, false, "x/y", "*/*/*/*")        // slashes-posix.js:105
		assertMatch(t, false, "z/z", "*/*/*/*")        // slashes-posix.js:106

		// Pattern: */*/*/*/*
		assertMatch(t, false, "a", "*/*/*/*/*")          // slashes-posix.js:108
		assertMatch(t, false, "b", "*/*/*/*/*")          // slashes-posix.js:109
		assertMatch(t, false, "a/a", "*/*/*/*/*")        // slashes-posix.js:110
		assertMatch(t, false, "a/b", "*/*/*/*/*")        // slashes-posix.js:111
		assertMatch(t, false, "a/c", "*/*/*/*/*")        // slashes-posix.js:112
		assertMatch(t, false, "a/x", "*/*/*/*/*")        // slashes-posix.js:113
		assertMatch(t, false, "a/a/a", "*/*/*/*/*")      // slashes-posix.js:114
		assertMatch(t, false, "a/a/b", "*/*/*/*/*")      // slashes-posix.js:115
		assertMatch(t, false, "a/a/a/a", "*/*/*/*/*")    // slashes-posix.js:116
		assertMatch(t, true, "a/a/a/a/a", "*/*/*/*/*")   // slashes-posix.js:117
		assertMatch(t, false, "x/y", "*/*/*/*/*")        // slashes-posix.js:118
		assertMatch(t, false, "z/z", "*/*/*/*/*")        // slashes-posix.js:119

		// Pattern: a/*
		assertMatch(t, false, "a", "a/*")          // slashes-posix.js:121
		assertMatch(t, false, "b", "a/*")          // slashes-posix.js:122
		assertMatch(t, true, "a/a", "a/*")         // slashes-posix.js:123
		assertMatch(t, true, "a/b", "a/*")         // slashes-posix.js:124
		assertMatch(t, true, "a/c", "a/*")         // slashes-posix.js:125
		assertMatch(t, true, "a/x", "a/*")         // slashes-posix.js:126
		assertMatch(t, false, "a/a/a", "a/*")      // slashes-posix.js:127
		assertMatch(t, false, "a/a/b", "a/*")      // slashes-posix.js:128
		assertMatch(t, false, "a/a/a/a", "a/*")    // slashes-posix.js:129
		assertMatch(t, false, "a/a/a/a/a", "a/*")  // slashes-posix.js:130
		assertMatch(t, false, "x/y", "a/*")        // slashes-posix.js:131
		assertMatch(t, false, "z/z", "a/*")        // slashes-posix.js:132

		// Pattern: a/*/*
		assertMatch(t, false, "a", "a/*/*")          // slashes-posix.js:134
		assertMatch(t, false, "b", "a/*/*")          // slashes-posix.js:135
		assertMatch(t, false, "a/a", "a/*/*")        // slashes-posix.js:136
		assertMatch(t, false, "a/b", "a/*/*")        // slashes-posix.js:137
		assertMatch(t, false, "a/c", "a/*/*")        // slashes-posix.js:138
		assertMatch(t, false, "a/x", "a/*/*")        // slashes-posix.js:139
		assertMatch(t, true, "a/a/a", "a/*/*")       // slashes-posix.js:140
		assertMatch(t, true, "a/a/b", "a/*/*")       // slashes-posix.js:141
		assertMatch(t, false, "a/a/a/a", "a/*/*")    // slashes-posix.js:142
		assertMatch(t, false, "a/a/a/a/a", "a/*/*")  // slashes-posix.js:143
		assertMatch(t, false, "x/y", "a/*/*")        // slashes-posix.js:144
		assertMatch(t, false, "z/z", "a/*/*")        // slashes-posix.js:145

		// Pattern: a/*/*/*
		assertMatch(t, false, "a", "a/*/*/*")          // slashes-posix.js:147
		assertMatch(t, false, "b", "a/*/*/*")          // slashes-posix.js:148
		assertMatch(t, false, "a/a", "a/*/*/*")        // slashes-posix.js:149
		assertMatch(t, false, "a/b", "a/*/*/*")        // slashes-posix.js:150
		assertMatch(t, false, "a/c", "a/*/*/*")        // slashes-posix.js:151
		assertMatch(t, false, "a/x", "a/*/*/*")        // slashes-posix.js:152
		assertMatch(t, false, "a/a/a", "a/*/*/*")      // slashes-posix.js:153
		assertMatch(t, false, "a/a/b", "a/*/*/*")      // slashes-posix.js:154
		assertMatch(t, true, "a/a/a/a", "a/*/*/*")     // slashes-posix.js:155
		assertMatch(t, false, "a/a/a/a/a", "a/*/*/*")  // slashes-posix.js:156
		assertMatch(t, false, "x/y", "a/*/*/*")        // slashes-posix.js:157
		assertMatch(t, false, "z/z", "a/*/*/*")        // slashes-posix.js:158

		// Pattern: a/*/*/*/*
		assertMatch(t, false, "a", "a/*/*/*/*")          // slashes-posix.js:160
		assertMatch(t, false, "b", "a/*/*/*/*")          // slashes-posix.js:161
		assertMatch(t, false, "a/a", "a/*/*/*/*")        // slashes-posix.js:162
		assertMatch(t, false, "a/b", "a/*/*/*/*")        // slashes-posix.js:163
		assertMatch(t, false, "a/c", "a/*/*/*/*")        // slashes-posix.js:164
		assertMatch(t, false, "a/x", "a/*/*/*/*")        // slashes-posix.js:165
		assertMatch(t, false, "a/a/a", "a/*/*/*/*")      // slashes-posix.js:166
		assertMatch(t, false, "a/a/b", "a/*/*/*/*")      // slashes-posix.js:167
		assertMatch(t, false, "a/a/a/a", "a/*/*/*/*")    // slashes-posix.js:168
		assertMatch(t, true, "a/a/a/a/a", "a/*/*/*/*")   // slashes-posix.js:169
		assertMatch(t, false, "x/y", "a/*/*/*/*")        // slashes-posix.js:170
		assertMatch(t, false, "z/z", "a/*/*/*/*")        // slashes-posix.js:171

		// Pattern: a/*/a
		assertMatch(t, false, "a", "a/*/a")          // slashes-posix.js:173
		assertMatch(t, false, "b", "a/*/a")          // slashes-posix.js:174
		assertMatch(t, false, "a/a", "a/*/a")        // slashes-posix.js:175
		assertMatch(t, false, "a/b", "a/*/a")        // slashes-posix.js:176
		assertMatch(t, false, "a/c", "a/*/a")        // slashes-posix.js:177
		assertMatch(t, false, "a/x", "a/*/a")        // slashes-posix.js:178
		assertMatch(t, true, "a/a/a", "a/*/a")       // slashes-posix.js:179
		assertMatch(t, false, "a/a/b", "a/*/a")      // slashes-posix.js:180
		assertMatch(t, false, "a/a/a/a", "a/*/a")    // slashes-posix.js:181
		assertMatch(t, false, "a/a/a/a/a", "a/*/a")  // slashes-posix.js:182
		assertMatch(t, false, "x/y", "a/*/a")        // slashes-posix.js:183
		assertMatch(t, false, "z/z", "a/*/a")        // slashes-posix.js:184

		// Pattern: a/*/b
		assertMatch(t, false, "a", "a/*/b")          // slashes-posix.js:186
		assertMatch(t, false, "b", "a/*/b")          // slashes-posix.js:187
		assertMatch(t, false, "a/a", "a/*/b")        // slashes-posix.js:188
		assertMatch(t, false, "a/b", "a/*/b")        // slashes-posix.js:189
		assertMatch(t, false, "a/c", "a/*/b")        // slashes-posix.js:190
		assertMatch(t, false, "a/x", "a/*/b")        // slashes-posix.js:191
		assertMatch(t, false, "a/a/a", "a/*/b")      // slashes-posix.js:192
		assertMatch(t, true, "a/a/b", "a/*/b")       // slashes-posix.js:193
		assertMatch(t, false, "a/a/a/a", "a/*/b")    // slashes-posix.js:194
		assertMatch(t, false, "a/a/a/a/a", "a/*/b")  // slashes-posix.js:195
		assertMatch(t, false, "x/y", "a/*/b")        // slashes-posix.js:196
		assertMatch(t, false, "z/z", "a/*/b")        // slashes-posix.js:197
	})

	t.Run("should support globstars (**)", func(t *testing.T) {
		// Pattern: a (literal)
		assertMatch(t, true, "a", "a")       // slashes-posix.js:201
		assertMatch(t, false, "a/", "a")     // slashes-posix.js:202
		assertMatch(t, false, "a/a", "a")    // slashes-posix.js:203
		assertMatch(t, false, "a/b", "a")    // slashes-posix.js:204
		assertMatch(t, false, "a/c", "a")    // slashes-posix.js:205
		assertMatch(t, false, "a/x", "a")    // slashes-posix.js:206
		assertMatch(t, false, "a/x/y", "a")  // slashes-posix.js:207
		assertMatch(t, false, "a/x/y/z", "a") // slashes-posix.js:208

		// Pattern: *
		assertMatch(t, true, "a", "*")         // slashes-posix.js:210
		// line 211: isMatch('a/', '*', { relaxSlashes: true }) — skipped, relaxSlashes not ported
		assertMatch(t, true, "a/", "*{,/}")    // slashes-posix.js:212
		assertMatch(t, false, "a/a", "*")      // slashes-posix.js:213
		assertMatch(t, false, "a/b", "*")      // slashes-posix.js:214
		assertMatch(t, false, "a/c", "*")      // slashes-posix.js:215
		assertMatch(t, false, "a/x", "*")      // slashes-posix.js:216
		assertMatch(t, false, "a/x/y", "*")    // slashes-posix.js:217
		assertMatch(t, false, "a/x/y/z", "*")  // slashes-posix.js:218

		// Pattern: */
		assertMatch(t, false, "a", "*/")         // slashes-posix.js:220
		assertMatch(t, true, "a/", "*/")         // slashes-posix.js:221
		assertMatch(t, false, "a/a", "*/")       // slashes-posix.js:222
		assertMatch(t, false, "a/b", "*/")       // slashes-posix.js:223
		assertMatch(t, false, "a/c", "*/")       // slashes-posix.js:224
		assertMatch(t, false, "a/x", "*/")       // slashes-posix.js:225
		assertMatch(t, false, "a/x/y", "*/")     // slashes-posix.js:226
		assertMatch(t, false, "a/x/y/z", "*/")   // slashes-posix.js:227

		// Pattern: */*
		assertMatch(t, false, "a", "*/*")        // slashes-posix.js:229
		assertMatch(t, false, "a/", "*/*")       // slashes-posix.js:230
		assertMatch(t, true, "a/a", "*/*")       // slashes-posix.js:231
		assertMatch(t, true, "a/b", "*/*")       // slashes-posix.js:232
		assertMatch(t, true, "a/c", "*/*")       // slashes-posix.js:233
		assertMatch(t, true, "a/x", "*/*")       // slashes-posix.js:234
		assertMatch(t, false, "a/x/y", "*/*")    // slashes-posix.js:235
		assertMatch(t, false, "a/x/y/z", "*/*")  // slashes-posix.js:236

		// Pattern: **
		assertMatch(t, true, "a", "**")         // slashes-posix.js:238
		assertMatch(t, true, "a/", "**")        // slashes-posix.js:239
		assertMatch(t, true, "a/a", "**")       // slashes-posix.js:240
		assertMatch(t, true, "a/b", "**")       // slashes-posix.js:241
		assertMatch(t, true, "a/c", "**")       // slashes-posix.js:242
		assertMatch(t, true, "a/x", "**")       // slashes-posix.js:243
		assertMatch(t, true, "a/x/y", "**")     // slashes-posix.js:244
		assertMatch(t, true, "a/x/y/z", "**")   // slashes-posix.js:245

		// Pattern: **/a
		assertMatch(t, false, "a/", "**/a")       // slashes-posix.js:247
		assertMatch(t, false, "a/b", "**/a")      // slashes-posix.js:248
		assertMatch(t, false, "a/c", "**/a")      // slashes-posix.js:249
		assertMatch(t, false, "a/x", "**/a")      // slashes-posix.js:250
		assertMatch(t, false, "a/x/y/z", "**/a")  // slashes-posix.js:251
		assertMatch(t, true, "a/x/y/z/a", "**/a") // slashes-posix.js:252
		assertMatch(t, true, "a", "**/a")          // slashes-posix.js:253
		assertMatch(t, true, "a/a", "**/a")        // slashes-posix.js:254

		// Pattern: a/*
		assertMatch(t, false, "a", "a/*")        // slashes-posix.js:256
		assertMatch(t, false, "a/", "a/*")       // slashes-posix.js:257
		assertMatch(t, true, "a/a", "a/*")       // slashes-posix.js:258
		assertMatch(t, true, "a/b", "a/*")       // slashes-posix.js:259
		assertMatch(t, true, "a/c", "a/*")       // slashes-posix.js:260
		assertMatch(t, true, "a/x", "a/*")       // slashes-posix.js:261
		assertMatch(t, false, "a/x/y", "a/*")    // slashes-posix.js:262
		assertMatch(t, false, "a/x/y/z", "a/*")  // slashes-posix.js:263

		// Pattern: a/**
		assertMatch(t, true, "a", "a/**")         // slashes-posix.js:265
		assertMatch(t, true, "a/", "a/**")        // slashes-posix.js:266
		assertMatch(t, true, "a/a", "a/**")       // slashes-posix.js:267
		assertMatch(t, true, "a/b", "a/**")       // slashes-posix.js:268
		assertMatch(t, true, "a/c", "a/**")       // slashes-posix.js:269
		assertMatch(t, true, "a/x", "a/**")       // slashes-posix.js:270
		assertMatch(t, true, "a/x/y", "a/**")     // slashes-posix.js:271
		assertMatch(t, true, "a/x/y/z", "a/**")   // slashes-posix.js:272

		// Pattern: a/**/*
		assertMatch(t, false, "a", "a/**/*")        // slashes-posix.js:274
		assertMatch(t, false, "a/", "a/**/*")       // slashes-posix.js:275
		assertMatch(t, true, "a/a", "a/**/*")       // slashes-posix.js:276
		assertMatch(t, true, "a/b", "a/**/*")       // slashes-posix.js:277
		assertMatch(t, true, "a/c", "a/**/*")       // slashes-posix.js:278
		assertMatch(t, true, "a/x", "a/**/*")       // slashes-posix.js:279
		assertMatch(t, true, "a/x/y", "a/**/*")     // slashes-posix.js:280
		assertMatch(t, true, "a/x/y/z", "a/**/*")   // slashes-posix.js:281

		// Pattern: a/**/**/*
		assertMatch(t, false, "a", "a/**/**/*")        // slashes-posix.js:283
		assertMatch(t, false, "a/", "a/**/**/*")       // slashes-posix.js:284
		assertMatch(t, true, "a/a", "a/**/**/*")       // slashes-posix.js:285
		assertMatch(t, true, "a/b", "a/**/**/*")       // slashes-posix.js:286
		assertMatch(t, true, "a/c", "a/**/**/*")       // slashes-posix.js:287
		assertMatch(t, true, "a/x", "a/**/**/*")       // slashes-posix.js:288
		assertMatch(t, true, "a/x/y", "a/**/**/*")     // slashes-posix.js:289
		assertMatch(t, true, "a/x/y/z", "a/**/**/*")   // slashes-posix.js:290

		// Complex globstar patterns
		assertMatch(t, true, "a/b/foo/bar/baz.qux", "a/b/**/bar/**/*.*") // slashes-posix.js:292
		assertMatch(t, true, "a/b/bar/baz.qux", "a/b/**/bar/**/*.*")    // slashes-posix.js:293
	})

	t.Run("should support negation patterns", func(t *testing.T) {
		assertMatch(t, true, "a/a", "!a/b")   // slashes-posix.js:297
		assertMatch(t, false, "a/b", "!a/b")  // slashes-posix.js:298
		assertMatch(t, true, "a/c", "!a/b")   // slashes-posix.js:299
		assertMatch(t, true, "b/a", "!a/b")   // slashes-posix.js:300
		assertMatch(t, true, "b/b", "!a/b")   // slashes-posix.js:301
		assertMatch(t, true, "b/c", "!a/b")   // slashes-posix.js:302

		// Lines 304-309: isMatch with array patterns ['*/*', '!a/b', '!*/c']
		// assertMatch only takes string patterns, so use IsMatch directly.
		if !IsMatch("a/a", []string{"*/*", "!a/b", "!*/c"}, nil) { // slashes-posix.js:304
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/a", []string{"*/*", "!a/b", "!*/c"})
		}
		if !IsMatch("a/b", []string{"*/*", "!a/b", "!*/c"}, nil) { // slashes-posix.js:305
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/b", []string{"*/*", "!a/b", "!*/c"})
		}
		if !IsMatch("a/c", []string{"*/*", "!a/b", "!*/c"}, nil) { // slashes-posix.js:306
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/c", []string{"*/*", "!a/b", "!*/c"})
		}
		if !IsMatch("b/a", []string{"*/*", "!a/b", "!*/c"}, nil) { // slashes-posix.js:307
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/a", []string{"*/*", "!a/b", "!*/c"})
		}
		if !IsMatch("b/b", []string{"*/*", "!a/b", "!*/c"}, nil) { // slashes-posix.js:308
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/b", []string{"*/*", "!a/b", "!*/c"})
		}
		if !IsMatch("b/c", []string{"*/*", "!a/b", "!*/c"}, nil) { // slashes-posix.js:309
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/c", []string{"*/*", "!a/b", "!*/c"})
		}

		// Lines 311-316: isMatch with array patterns ['!a/b', '!*/c']
		if !IsMatch("a/a", []string{"!a/b", "!*/c"}, nil) { // slashes-posix.js:311
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/a", []string{"!a/b", "!*/c"})
		}
		if !IsMatch("a/b", []string{"!a/b", "!*/c"}, nil) { // slashes-posix.js:312
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/b", []string{"!a/b", "!*/c"})
		}
		if !IsMatch("a/c", []string{"!a/b", "!*/c"}, nil) { // slashes-posix.js:313
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/c", []string{"!a/b", "!*/c"})
		}
		if !IsMatch("b/a", []string{"!a/b", "!*/c"}, nil) { // slashes-posix.js:314
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/a", []string{"!a/b", "!*/c"})
		}
		if !IsMatch("b/b", []string{"!a/b", "!*/c"}, nil) { // slashes-posix.js:315
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/b", []string{"!a/b", "!*/c"})
		}
		if !IsMatch("b/c", []string{"!a/b", "!*/c"}, nil) { // slashes-posix.js:316
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/c", []string{"!a/b", "!*/c"})
		}

		// Lines 318-323: isMatch with array patterns ['!a/b', '!a/c']
		if !IsMatch("a/a", []string{"!a/b", "!a/c"}, nil) { // slashes-posix.js:318
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/a", []string{"!a/b", "!a/c"})
		}
		if !IsMatch("a/b", []string{"!a/b", "!a/c"}, nil) { // slashes-posix.js:319
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/b", []string{"!a/b", "!a/c"})
		}
		if !IsMatch("a/c", []string{"!a/b", "!a/c"}, nil) { // slashes-posix.js:320
			t.Errorf("expected IsMatch(%q, %v) to be true", "a/c", []string{"!a/b", "!a/c"})
		}
		if !IsMatch("b/a", []string{"!a/b", "!a/c"}, nil) { // slashes-posix.js:321
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/a", []string{"!a/b", "!a/c"})
		}
		if !IsMatch("b/b", []string{"!a/b", "!a/c"}, nil) { // slashes-posix.js:322
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/b", []string{"!a/b", "!a/c"})
		}
		if !IsMatch("b/c", []string{"!a/b", "!a/c"}, nil) { // slashes-posix.js:323
			t.Errorf("expected IsMatch(%q, %v) to be true", "b/c", []string{"!a/b", "!a/c"})
		}

		// Negation with parens: !a/(b)
		assertMatch(t, true, "a/a", "!a/(b)")   // slashes-posix.js:325
		assertMatch(t, false, "a/b", "!a/(b)")  // slashes-posix.js:326
		assertMatch(t, true, "a/c", "!a/(b)")   // slashes-posix.js:327
		assertMatch(t, true, "b/a", "!a/(b)")   // slashes-posix.js:328
		assertMatch(t, true, "b/b", "!a/(b)")   // slashes-posix.js:329
		assertMatch(t, true, "b/c", "!a/(b)")   // slashes-posix.js:330

		// Negation with extglob: !(a/b)
		assertMatch(t, true, "a/a", "!(a/b)")   // slashes-posix.js:332
		assertMatch(t, false, "a/b", "!(a/b)")  // slashes-posix.js:333
		assertMatch(t, true, "a/c", "!(a/b)")   // slashes-posix.js:334
		assertMatch(t, true, "b/a", "!(a/b)")   // slashes-posix.js:335
		assertMatch(t, true, "b/b", "!(a/b)")   // slashes-posix.js:336
		assertMatch(t, true, "b/c", "!(a/b)")   // slashes-posix.js:337
	})

	t.Run("should work with file extensions", func(t *testing.T) {
		assertMatch(t, false, "a.txt", "a/**/*.txt")   // slashes-posix.js:341
		assertMatch(t, true, "a/b.txt", "a/**/*.txt")  // slashes-posix.js:342
		assertMatch(t, true, "a/x/y.txt", "a/**/*.txt") // slashes-posix.js:343
		// Line 344: isMatch('a/x/y/z', ['a/**/*.txt']) — array with single pattern, equivalent to string
		assertMatch(t, false, "a/x/y/z", "a/**/*.txt") // slashes-posix.js:344

		assertMatch(t, false, "a.txt", "a/*.txt")      // slashes-posix.js:346
		assertMatch(t, true, "a/b.txt", "a/*.txt")     // slashes-posix.js:347
		assertMatch(t, false, "a/x/y.txt", "a/*.txt")  // slashes-posix.js:348
		assertMatch(t, false, "a/x/y/z", "a/*.txt")    // slashes-posix.js:349

		assertMatch(t, true, "a.txt", "a*.txt")        // slashes-posix.js:351
		assertMatch(t, false, "a/b.txt", "a*.txt")     // slashes-posix.js:352
		assertMatch(t, false, "a/x/y.txt", "a*.txt")   // slashes-posix.js:353
		assertMatch(t, false, "a/x/y/z", "a*.txt")     // slashes-posix.js:354

		assertMatch(t, true, "a.txt", "*.txt")         // slashes-posix.js:356
		assertMatch(t, false, "a/b.txt", "*.txt")      // slashes-posix.js:357
		assertMatch(t, false, "a/x/y.txt", "*.txt")    // slashes-posix.js:358
		assertMatch(t, false, "a/x/y/z", "*.txt")      // slashes-posix.js:359
	})

	t.Run("should match one directory level with a single star (*)", func(t *testing.T) {
		ss := &Options{StrictSlashes: true}

		assertMatch(t, false, "/a", "*/")           // slashes-posix.js:363
		assertMatch(t, false, "/a", "*/*/*")         // slashes-posix.js:364
		assertMatch(t, false, "/a", "*/*/*/*")       // slashes-posix.js:365
		assertMatch(t, false, "/a", "*/*/*/*/*")     // slashes-posix.js:366
		assertMatch(t, false, "/a", "/*/")           // slashes-posix.js:367
		assertMatch(t, false, "/a", "a/*")           // slashes-posix.js:368
		assertMatch(t, false, "/a", "a/*/*")         // slashes-posix.js:369
		assertMatch(t, false, "/a", "a/*/*/*")       // slashes-posix.js:370
		assertMatch(t, false, "/a", "a/*/*/*/*")     // slashes-posix.js:371
		assertMatch(t, false, "/a", "a/*/a")         // slashes-posix.js:372
		assertMatch(t, false, "/a", "a/*/b")         // slashes-posix.js:373
		assertMatch(t, false, "/a/", "*")            // slashes-posix.js:374
		assertMatch(t, false, "/a/", "**/*", ss)     // slashes-posix.js:375
		assertMatch(t, false, "/a/", "*/")           // slashes-posix.js:376
		assertMatch(t, false, "/a/", "*/*", ss)      // slashes-posix.js:377
		assertMatch(t, false, "/a/", "*/*/*")        // slashes-posix.js:378
		assertMatch(t, false, "/a/", "*/*/*/*")      // slashes-posix.js:379
		assertMatch(t, false, "/a/", "*/*/*/*/*")    // slashes-posix.js:380
		assertMatch(t, false, "/a/", "/*", ss)       // slashes-posix.js:381
		assertMatch(t, false, "/a/", "a/*")          // slashes-posix.js:382
		assertMatch(t, false, "/a/", "a/*/*")        // slashes-posix.js:383
		assertMatch(t, false, "/a/", "a/*/*/*")      // slashes-posix.js:384
		assertMatch(t, false, "/a/", "a/*/*/*/*")    // slashes-posix.js:385
		assertMatch(t, false, "/a/", "a/*/a")        // slashes-posix.js:386
		assertMatch(t, false, "/a/", "a/*/b")        // slashes-posix.js:387
		assertMatch(t, false, "/ab", "*")            // slashes-posix.js:388
		assertMatch(t, false, "/abc", "*")           // slashes-posix.js:389
		assertMatch(t, false, "/b", "*")             // slashes-posix.js:390
		assertMatch(t, false, "/b", "*/")            // slashes-posix.js:391
		assertMatch(t, false, "/b", "*/*/*")         // slashes-posix.js:392
		assertMatch(t, false, "/b", "*/*/*/*")       // slashes-posix.js:393
		assertMatch(t, false, "/b", "*/*/*/*/*")     // slashes-posix.js:394
		assertMatch(t, false, "/b", "/*/")           // slashes-posix.js:395
		assertMatch(t, false, "/b", "a/*")           // slashes-posix.js:396
		assertMatch(t, false, "/b", "a/*/*")         // slashes-posix.js:397
		assertMatch(t, false, "/b", "a/*/*/*")       // slashes-posix.js:398
		assertMatch(t, false, "/b", "a/*/*/*/*")     // slashes-posix.js:399
		assertMatch(t, false, "/b", "a/*/a")         // slashes-posix.js:400
		assertMatch(t, false, "/b", "a/*/b")         // slashes-posix.js:401
		assertMatch(t, false, "a", "*/")             // slashes-posix.js:402
		assertMatch(t, false, "a", "*/*")            // slashes-posix.js:403
		assertMatch(t, false, "a", "*/*/*")          // slashes-posix.js:404
		assertMatch(t, false, "a", "*/*/*/*")        // slashes-posix.js:405
		assertMatch(t, false, "a", "*/*/*/*/*")      // slashes-posix.js:406
		assertMatch(t, false, "a", "/*")             // slashes-posix.js:407
		assertMatch(t, false, "a", "/*/")            // slashes-posix.js:408
		assertMatch(t, false, "a", "a/*")            // slashes-posix.js:409
		assertMatch(t, false, "a", "a/*/*")          // slashes-posix.js:410
		assertMatch(t, false, "a", "a/*/*/*")        // slashes-posix.js:411
		assertMatch(t, false, "a", "a/*/*/*/*")      // slashes-posix.js:412
		assertMatch(t, false, "a", "a/*/a")          // slashes-posix.js:413
		assertMatch(t, false, "a", "a/*/b")          // slashes-posix.js:414
		assertMatch(t, false, "a/", "*", ss)         // slashes-posix.js:415
		assertMatch(t, false, "a/", "**/*", ss)      // slashes-posix.js:416
		assertMatch(t, false, "a/", "*/*", ss)       // slashes-posix.js:417
		assertMatch(t, false, "a/", "*/*/*/*", ss)   // slashes-posix.js:418
		assertMatch(t, false, "a/", "*/*/*/*/*", ss) // slashes-posix.js:419
		assertMatch(t, false, "a/", "/*", ss)        // slashes-posix.js:420
		assertMatch(t, false, "a/", "/*/", ss)       // slashes-posix.js:421
		assertMatch(t, false, "a/", "a/*", ss)       // slashes-posix.js:422
		assertMatch(t, false, "a/", "a/*/*", ss)     // slashes-posix.js:423
		assertMatch(t, false, "a/", "a/*/*/*", ss)   // slashes-posix.js:424
		assertMatch(t, false, "a/", "a/*/*/*/*", ss) // slashes-posix.js:425
		assertMatch(t, false, "a/", "a/*/a", ss)     // slashes-posix.js:426
		assertMatch(t, false, "a/", "a/*/b", ss)     // slashes-posix.js:427
		assertMatch(t, false, "a/a", "*")            // slashes-posix.js:428
		assertMatch(t, false, "a/a", "*/")           // slashes-posix.js:429
		assertMatch(t, false, "a/a", "*/*/*")        // slashes-posix.js:430
		assertMatch(t, false, "a/a", "*/*/*/*")      // slashes-posix.js:431
		assertMatch(t, false, "a/a", "*/*/*/*/*")    // slashes-posix.js:432
		assertMatch(t, false, "a/a", "/*")           // slashes-posix.js:433
		assertMatch(t, false, "a/a", "/*/")          // slashes-posix.js:434
		assertMatch(t, false, "a/a", "a/*/*")        // slashes-posix.js:435
		assertMatch(t, false, "a/a", "a/*/*/*")      // slashes-posix.js:436
		assertMatch(t, false, "a/a", "a/*/*/*/*")    // slashes-posix.js:437
		assertMatch(t, false, "a/a", "a/*/a")        // slashes-posix.js:438
		assertMatch(t, false, "a/a", "a/*/b")        // slashes-posix.js:439
		assertMatch(t, false, "a/a/a", "*")          // slashes-posix.js:440
		assertMatch(t, false, "a/a/a", "*/")         // slashes-posix.js:441
		assertMatch(t, false, "a/a/a", "*/*")        // slashes-posix.js:442
		assertMatch(t, false, "a/a/a", "*/*/*/*")    // slashes-posix.js:443
		assertMatch(t, false, "a/a/a", "*/*/*/*/*")  // slashes-posix.js:444
		assertMatch(t, false, "a/a/a", "/*")         // slashes-posix.js:445
		assertMatch(t, false, "a/a/a", "/*/")        // slashes-posix.js:446
		assertMatch(t, false, "a/a/a", "a/*")        // slashes-posix.js:447
		assertMatch(t, false, "a/a/a", "a/*/*/*")    // slashes-posix.js:448
		assertMatch(t, false, "a/a/a", "a/*/*/*/*")  // slashes-posix.js:449
		assertMatch(t, false, "a/a/a", "a/*/b")      // slashes-posix.js:450
		assertMatch(t, false, "a/a/a/a", "*")         // slashes-posix.js:451
		assertMatch(t, false, "a/a/a/a", "*/")        // slashes-posix.js:452
		assertMatch(t, false, "a/a/a/a", "*/*")       // slashes-posix.js:453
		assertMatch(t, false, "a/a/a/a", "*/*/*")     // slashes-posix.js:454
		assertMatch(t, false, "a/a/a/a", "*/*/*/*/*") // slashes-posix.js:455
		assertMatch(t, false, "a/a/a/a", "/*")        // slashes-posix.js:456
		assertMatch(t, false, "a/a/a/a", "/*/")       // slashes-posix.js:457
		assertMatch(t, false, "a/a/a/a", "a/*")       // slashes-posix.js:458
		assertMatch(t, false, "a/a/a/a", "a/*/*")     // slashes-posix.js:459
		assertMatch(t, false, "a/a/a/a", "a/*/*/*/*") // slashes-posix.js:460
		assertMatch(t, false, "a/a/a/a", "a/*/a")     // slashes-posix.js:461
		assertMatch(t, false, "a/a/a/a", "a/*/b")     // slashes-posix.js:462
		assertMatch(t, false, "a/a/a/a/a", "*")         // slashes-posix.js:463
		assertMatch(t, false, "a/a/a/a/a", "*/")        // slashes-posix.js:464
		assertMatch(t, false, "a/a/a/a/a", "*/*")       // slashes-posix.js:465
		assertMatch(t, false, "a/a/a/a/a", "*/*/*")     // slashes-posix.js:466
		assertMatch(t, false, "a/a/a/a/a", "*/*/*/*")   // slashes-posix.js:467
		assertMatch(t, false, "a/a/a/a/a", "/*")        // slashes-posix.js:468
		assertMatch(t, false, "a/a/a/a/a", "/*/")       // slashes-posix.js:469
		assertMatch(t, false, "a/a/a/a/a", "a/*")       // slashes-posix.js:470
		assertMatch(t, false, "a/a/a/a/a", "a/*/*")     // slashes-posix.js:471
		assertMatch(t, false, "a/a/a/a/a", "a/*/*/*")   // slashes-posix.js:472
		assertMatch(t, false, "a/a/a/a/a", "a/*/a")     // slashes-posix.js:473
		assertMatch(t, false, "a/a/a/a/a", "a/*/b")     // slashes-posix.js:474
		assertMatch(t, false, "a/a/b", "*")            // slashes-posix.js:475
		assertMatch(t, false, "a/a/b", "*/")           // slashes-posix.js:476
		assertMatch(t, false, "a/a/b", "*/*")          // slashes-posix.js:477
		assertMatch(t, false, "a/a/b", "*/*/*/*")      // slashes-posix.js:478
		assertMatch(t, false, "a/a/b", "*/*/*/*/*")    // slashes-posix.js:479
		assertMatch(t, false, "a/a/b", "/*")           // slashes-posix.js:480
		assertMatch(t, false, "a/a/b", "/*/")          // slashes-posix.js:481
		assertMatch(t, false, "a/a/b", "a/*")          // slashes-posix.js:482
		assertMatch(t, false, "a/a/b", "a/*/*/*")      // slashes-posix.js:483
		assertMatch(t, false, "a/a/b", "a/*/*/*/*")    // slashes-posix.js:484
		assertMatch(t, false, "a/a/b", "a/*/a")        // slashes-posix.js:485
		assertMatch(t, false, "a/b", "*")              // slashes-posix.js:486
		assertMatch(t, false, "a/b", "*/")             // slashes-posix.js:487
		assertMatch(t, false, "a/b", "*/*/*/*")        // slashes-posix.js:488
		assertMatch(t, false, "a/b", "*/*/*/*/*")      // slashes-posix.js:489
		assertMatch(t, false, "a/b", "/*")             // slashes-posix.js:490
		assertMatch(t, false, "a/b", "/*/")            // slashes-posix.js:491
		assertMatch(t, false, "a/b", "a/*/*")          // slashes-posix.js:492
		assertMatch(t, false, "a/b", "a/*/*/*")        // slashes-posix.js:493
		assertMatch(t, false, "a/b", "a/*/*/*/*")      // slashes-posix.js:494
		assertMatch(t, false, "a/b", "a/*/a")          // slashes-posix.js:495
		assertMatch(t, false, "a/b", "a/*/b")          // slashes-posix.js:496
		assertMatch(t, false, "a/c", "*")              // slashes-posix.js:497
		assertMatch(t, false, "a/c", "*/")             // slashes-posix.js:498
		assertMatch(t, false, "a/c", "*/*/*/*")        // slashes-posix.js:499
		assertMatch(t, false, "a/c", "*/*/*/*/*")      // slashes-posix.js:500
		assertMatch(t, false, "a/c", "/*")             // slashes-posix.js:501
		assertMatch(t, false, "a/c", "/*/")            // slashes-posix.js:502
		assertMatch(t, false, "a/c", "a/*/*")          // slashes-posix.js:503
		assertMatch(t, false, "a/c", "a/*/*/*")        // slashes-posix.js:504
		assertMatch(t, false, "a/c", "a/*/*/*/*")      // slashes-posix.js:505
		assertMatch(t, false, "a/c", "a/*/a")          // slashes-posix.js:506
		assertMatch(t, false, "a/c", "a/*/b")          // slashes-posix.js:507
		assertMatch(t, false, "a/x", "*")              // slashes-posix.js:508
		assertMatch(t, false, "a/x", "*/")             // slashes-posix.js:509
		assertMatch(t, false, "a/x", "*/*/*/*")        // slashes-posix.js:510
		assertMatch(t, false, "a/x", "*/*/*/*/*")      // slashes-posix.js:511
		assertMatch(t, false, "a/x", "/*")             // slashes-posix.js:512
		assertMatch(t, false, "a/x", "/*/")            // slashes-posix.js:513
		assertMatch(t, false, "a/x", "a/*/*")          // slashes-posix.js:514
		assertMatch(t, false, "a/x", "a/*/*/*")        // slashes-posix.js:515
		assertMatch(t, false, "a/x", "a/*/*/*/*")      // slashes-posix.js:516
		assertMatch(t, false, "a/x", "a/*/a")          // slashes-posix.js:517
		assertMatch(t, false, "a/x", "a/*/b")          // slashes-posix.js:518
		assertMatch(t, false, "b", "*/")               // slashes-posix.js:519
		assertMatch(t, false, "b", "*/*")              // slashes-posix.js:520
		assertMatch(t, false, "b", "*/*/*/*")          // slashes-posix.js:521
		assertMatch(t, false, "b", "*/*/*/*/*")        // slashes-posix.js:522
		assertMatch(t, false, "b", "/*")               // slashes-posix.js:523
		assertMatch(t, false, "b", "/*/")              // slashes-posix.js:524
		assertMatch(t, false, "b", "a/*")              // slashes-posix.js:525
		assertMatch(t, false, "b", "a/*/*")            // slashes-posix.js:526
		assertMatch(t, false, "b", "a/*/*/*")          // slashes-posix.js:527
		assertMatch(t, false, "b", "a/*/*/*/*")        // slashes-posix.js:528
		assertMatch(t, false, "b", "a/*/a")            // slashes-posix.js:529
		assertMatch(t, false, "b", "a/*/b")            // slashes-posix.js:530
		assertMatch(t, false, "x/y", "*")              // slashes-posix.js:531
		assertMatch(t, false, "x/y", "*/")             // slashes-posix.js:532
		assertMatch(t, false, "x/y", "*/*/*")          // slashes-posix.js:533
		assertMatch(t, false, "x/y", "*/*/*/*")        // slashes-posix.js:534
		assertMatch(t, false, "x/y", "*/*/*/*/*")      // slashes-posix.js:535
		assertMatch(t, false, "x/y", "/*")             // slashes-posix.js:536
		assertMatch(t, false, "x/y", "/*/")            // slashes-posix.js:537
		assertMatch(t, false, "x/y", "a/*")            // slashes-posix.js:538
		assertMatch(t, false, "x/y", "a/*/*")          // slashes-posix.js:539
		assertMatch(t, false, "x/y", "a/*/*/*")        // slashes-posix.js:540
		assertMatch(t, false, "x/y", "a/*/*/*/*")      // slashes-posix.js:541
		assertMatch(t, false, "x/y", "a/*/a")          // slashes-posix.js:542
		assertMatch(t, false, "x/y", "a/*/b")          // slashes-posix.js:543
		assertMatch(t, false, "z/z", "*")              // slashes-posix.js:544
		assertMatch(t, false, "z/z", "*/")             // slashes-posix.js:545
		assertMatch(t, false, "z/z", "*/*/*/*")        // slashes-posix.js:546
		assertMatch(t, false, "z/z", "*/*/*/*/*")      // slashes-posix.js:547
		assertMatch(t, false, "z/z", "/*")             // slashes-posix.js:548
		assertMatch(t, false, "z/z", "/*/")            // slashes-posix.js:549
		assertMatch(t, false, "z/z", "a/*")            // slashes-posix.js:550
		assertMatch(t, false, "z/z", "a/*/*")          // slashes-posix.js:551
		assertMatch(t, false, "z/z", "a/*/*/*")        // slashes-posix.js:552
		assertMatch(t, false, "z/z", "a/*/*/*/*")      // slashes-posix.js:553
		assertMatch(t, false, "z/z", "a/*/a")          // slashes-posix.js:554
		assertMatch(t, false, "z/z", "a/*/b")          // slashes-posix.js:555

		// True assertions (lines 556-600)
		assertMatch(t, true, "/a", "**/*")             // slashes-posix.js:556
		assertMatch(t, true, "/a", "*/*")              // slashes-posix.js:557
		assertMatch(t, true, "/a", "/*")               // slashes-posix.js:558
		assertMatch(t, true, "/a/", "**/*{,/}")        // slashes-posix.js:559
		assertMatch(t, true, "/a/", "*/*")             // slashes-posix.js:560
		assertMatch(t, true, "/a/", "*/*{,/}")         // slashes-posix.js:561
		assertMatch(t, true, "/a/", "/*")              // slashes-posix.js:562
		assertMatch(t, true, "/a/", "/*/")             // slashes-posix.js:563
		assertMatch(t, true, "/a/", "/*{,/}")          // slashes-posix.js:564
		assertMatch(t, true, "/b", "**/*")             // slashes-posix.js:565
		assertMatch(t, true, "/b", "*/*")              // slashes-posix.js:566
		assertMatch(t, true, "/b", "/*")               // slashes-posix.js:567
		assertMatch(t, true, "a", "*")                 // slashes-posix.js:568
		assertMatch(t, true, "a", "**/*")              // slashes-posix.js:569
		assertMatch(t, true, "a/", "**/*{,/}")         // slashes-posix.js:570
		assertMatch(t, true, "a/", "*/")               // slashes-posix.js:571
		assertMatch(t, true, "a/", "*{,/}")            // slashes-posix.js:572
		assertMatch(t, true, "a/a", "**/*")            // slashes-posix.js:573
		assertMatch(t, true, "a/a", "*/*")             // slashes-posix.js:574
		assertMatch(t, true, "a/a", "a/*")             // slashes-posix.js:575
		assertMatch(t, true, "a/a/a", "**/*")          // slashes-posix.js:576
		assertMatch(t, true, "a/a/a", "*/*/*")         // slashes-posix.js:577
		assertMatch(t, true, "a/a/a", "a/*/*")         // slashes-posix.js:578
		assertMatch(t, true, "a/a/a", "a/*/a")         // slashes-posix.js:579
		assertMatch(t, true, "a/a/a/a", "**/*")        // slashes-posix.js:580
		assertMatch(t, true, "a/a/a/a", "*/*/*/*")     // slashes-posix.js:581
		assertMatch(t, true, "a/a/a/a", "a/*/*/*")     // slashes-posix.js:582
		assertMatch(t, true, "a/a/a/a/a", "**/*")      // slashes-posix.js:583
		assertMatch(t, true, "a/a/a/a/a", "*/*/*/*/*")  // slashes-posix.js:584
		assertMatch(t, true, "a/a/a/a/a", "a/*/*/*/*")  // slashes-posix.js:585
		assertMatch(t, true, "a/a/b", "**/*")          // slashes-posix.js:586
		assertMatch(t, true, "a/a/b", "a/*/*")         // slashes-posix.js:587
		assertMatch(t, true, "a/a/b", "a/*/b")         // slashes-posix.js:588
		assertMatch(t, true, "a/b", "**/*")            // slashes-posix.js:589
		assertMatch(t, true, "a/b", "*/*")             // slashes-posix.js:590
		assertMatch(t, true, "a/b", "a/*")             // slashes-posix.js:591
		assertMatch(t, true, "a/c", "**/*")            // slashes-posix.js:592
		assertMatch(t, true, "a/c", "*/*")             // slashes-posix.js:593
		assertMatch(t, true, "a/c", "a/*")             // slashes-posix.js:594
		assertMatch(t, true, "a/x", "**/*")            // slashes-posix.js:595
		assertMatch(t, true, "a/x", "*/*")             // slashes-posix.js:596
		assertMatch(t, true, "a/x", "a/*")             // slashes-posix.js:597
		assertMatch(t, true, "b", "*")                 // slashes-posix.js:598
		assertMatch(t, true, "b", "**/*")              // slashes-posix.js:599
		assertMatch(t, true, "x/y", "**/*")            // slashes-posix.js:600
	})
}
