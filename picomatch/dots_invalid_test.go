package picomatch

// dots_invalid_test.go — Faithful 1:1 port of picomatch/test/dots-invalid.js
// Tests that dot segments (. and ..) in paths are never matched by glob patterns.
//
// Source: https://github.com/micromatch/picomatch/blob/master/test/dots-invalid.js

import (
	"testing"
)

func TestDotsInvalid(t *testing.T) {
	// dots-invalid.js line 6
	t.Run("invalid_exclusive_dots", func(t *testing.T) {
		// dots-invalid.js line 7
		t.Run("double_dots", func(t *testing.T) {
			// dots-invalid.js line 8
			t.Run("no_options", func(t *testing.T) {
				// dots-invalid.js line 9
				t.Run("should_not_match_leading_double-dots", func(t *testing.T) {
					// dots-invalid.js line 10
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*/*") // dots-invalid.js line 11
						assertMatch(t, false, "../abc", "*/abc") // dots-invalid.js line 12
						assertMatch(t, false, "../abc", "*/abc/*") // dots-invalid.js line 13
					})
					// dots-invalid.js line 16
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".*/*") // dots-invalid.js line 17
						assertMatch(t, false, "../abc", ".*/abc") // dots-invalid.js line 18
						assertMatch(t, false, "../abc", "*./*") // dots-invalid.js line 20
						assertMatch(t, false, "../abc", "*./abc") // dots-invalid.js line 21
					})
					// dots-invalid.js line 24
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**") // dots-invalid.js line 25
						assertMatch(t, false, "../abc", "**/**") // dots-invalid.js line 26
						assertMatch(t, false, "../abc", "**/**/**") // dots-invalid.js line 27
						assertMatch(t, false, "../abc", "**/abc") // dots-invalid.js line 29
						assertMatch(t, false, "../abc", "**/abc/**") // dots-invalid.js line 30
						assertMatch(t, false, "../abc", "abc/**") // dots-invalid.js line 32
						assertMatch(t, false, "../abc", "abc/**/**") // dots-invalid.js line 33
						assertMatch(t, false, "../abc", "abc/**/**/**") // dots-invalid.js line 34
						assertMatch(t, false, "../abc", "**/abc") // dots-invalid.js line 36
						assertMatch(t, false, "../abc", "**/abc/**") // dots-invalid.js line 37
						assertMatch(t, false, "../abc", "**/abc/**/**") // dots-invalid.js line 38
						assertMatch(t, false, "../abc", "**/**/abc/**") // dots-invalid.js line 40
						assertMatch(t, false, "../abc", "**/**/abc/**/**") // dots-invalid.js line 41
					})
					// dots-invalid.js line 44
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".**") // dots-invalid.js line 45
						assertMatch(t, false, "../abc", ".**/**") // dots-invalid.js line 46
						assertMatch(t, false, "../abc", ".**/abc") // dots-invalid.js line 47
					})
					// dots-invalid.js line 50
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*.*/**") // dots-invalid.js line 51
						assertMatch(t, false, "../abc", "*.*/abc") // dots-invalid.js line 52
					})
					// dots-invalid.js line 55
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**./**") // dots-invalid.js line 56
						assertMatch(t, false, "../abc", "**./abc") // dots-invalid.js line 57
					})
				})
				// dots-invalid.js line 61
				t.Run("should_not_match_nested_double-dots", func(t *testing.T) {
					// dots-invalid.js line 62
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/*") // dots-invalid.js line 63
						assertMatch(t, false, "/../abc", "/*/*") // dots-invalid.js line 64
						assertMatch(t, false, "/../abc", "*/*/*") // dots-invalid.js line 65
						assertMatch(t, false, "abc/../abc", "*/*/*") // dots-invalid.js line 67
						assertMatch(t, false, "abc/../abc/abc", "*/*/*/*") // dots-invalid.js line 68
					})
					// dots-invalid.js line 71
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/.*/*") // dots-invalid.js line 72
						assertMatch(t, false, "/../abc", "/.*/*") // dots-invalid.js line 73
						assertMatch(t, false, "/../abc", "*/*.*/*") // dots-invalid.js line 75
						assertMatch(t, false, "/../abc", "/*.*/*") // dots-invalid.js line 76
						assertMatch(t, false, "/../abc", "*/*./*") // dots-invalid.js line 78
						assertMatch(t, false, "/../abc", "/*./*") // dots-invalid.js line 79
						assertMatch(t, false, "abc/../abc", "*/.*/*") // dots-invalid.js line 81
						assertMatch(t, false, "abc/../abc", "*/*.*/*") // dots-invalid.js line 82
						assertMatch(t, false, "abc/../abc", "*/*./*") // dots-invalid.js line 83
					})
					// dots-invalid.js line 86
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**") // dots-invalid.js line 87
						assertMatch(t, false, "/../abc", "**/**") // dots-invalid.js line 88
						assertMatch(t, false, "/../abc", "/**/**") // dots-invalid.js line 89
						assertMatch(t, false, "/../abc", "**/**/**") // dots-invalid.js line 90
						assertMatch(t, false, "abc/../abc", "**/**/**") // dots-invalid.js line 92
						assertMatch(t, false, "abc/../abc/abc", "**/**/**/**") // dots-invalid.js line 93
					})
					// dots-invalid.js line 96
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/.**/**") // dots-invalid.js line 97
						assertMatch(t, false, "/../abc", "/.**/**") // dots-invalid.js line 98
						assertMatch(t, false, "abc/../abc", "**/.**/**") // dots-invalid.js line 100
						assertMatch(t, false, "abc/../abc", "/.**/**") // dots-invalid.js line 101
					})
					// dots-invalid.js line 104
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**./**") // dots-invalid.js line 105
						assertMatch(t, false, "/../abc", "/**./**") // dots-invalid.js line 106
						assertMatch(t, false, "abc/../abc", "**/**./**") // dots-invalid.js line 108
						assertMatch(t, false, "abc/../abc", "/**./**") // dots-invalid.js line 109
					})
					// dots-invalid.js line 112
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**.**/**") // dots-invalid.js line 113
						assertMatch(t, false, "/../abc", "**/*.*/**") // dots-invalid.js line 114
						assertMatch(t, false, "/../abc", "/**.**/**") // dots-invalid.js line 116
						assertMatch(t, false, "/../abc", "/*.*/**") // dots-invalid.js line 117
						assertMatch(t, false, "abc/../abc", "**/**.**/**") // dots-invalid.js line 119
						assertMatch(t, false, "abc/../abc", "**/*.*/**") // dots-invalid.js line 120
						assertMatch(t, false, "abc/../abc", "/**.**/**") // dots-invalid.js line 122
						assertMatch(t, false, "abc/../abc", "/*.*/**") // dots-invalid.js line 123
					})
				})
				// dots-invalid.js line 127
				t.Run("should_not_match_trailing_double-dots", func(t *testing.T) {
					// dots-invalid.js line 128
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*") // dots-invalid.js line 129
						assertMatch(t, false, "abc/..", "*/*/") // dots-invalid.js line 130
						assertMatch(t, false, "abc/..", "*/*/*") // dots-invalid.js line 131
						assertMatch(t, false, "abc/../", "*/*") // dots-invalid.js line 133
						assertMatch(t, false, "abc/../", "*/*/") // dots-invalid.js line 134
						assertMatch(t, false, "abc/../", "*/*/*") // dots-invalid.js line 135
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*") // dots-invalid.js line 137
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*/") // dots-invalid.js line 138
						assertMatch(t, false, "abc/../abc/abc/../", "*/*/*/*/*") // dots-invalid.js line 139
					})
					// dots-invalid.js line 142
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/.*") // dots-invalid.js line 143
						assertMatch(t, false, "abc/..", "*/.*/") // dots-invalid.js line 144
						assertMatch(t, false, "abc/..", "*/.*/*") // dots-invalid.js line 145
						assertMatch(t, false, "abc/../", "*/.*") // dots-invalid.js line 147
						assertMatch(t, false, "abc/../", "*/.*/") // dots-invalid.js line 148
						assertMatch(t, false, "abc/../", "*/.*/*") // dots-invalid.js line 149
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*") // dots-invalid.js line 151
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*/") // dots-invalid.js line 152
						assertMatch(t, false, "abc/../abc/abc/../", "*/.*/*/.*/*") // dots-invalid.js line 153
					})
					// dots-invalid.js line 156
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*.") // dots-invalid.js line 157
						assertMatch(t, false, "abc/..", "*/*./") // dots-invalid.js line 158
						assertMatch(t, false, "abc/..", "*/*./*") // dots-invalid.js line 159
						assertMatch(t, false, "abc/../", "*/*.") // dots-invalid.js line 161
						assertMatch(t, false, "abc/../", "*/*./") // dots-invalid.js line 162
						assertMatch(t, false, "abc/../", "*/*./*") // dots-invalid.js line 163
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*.") // dots-invalid.js line 165
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*./") // dots-invalid.js line 166
						assertMatch(t, false, "abc/../abc/abc/../", "*/*./*/*./*") // dots-invalid.js line 167
					})
					// dots-invalid.js line 170
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**") // dots-invalid.js line 171
						assertMatch(t, false, "abc/..", "**/**/") // dots-invalid.js line 172
						assertMatch(t, false, "abc/..", "**/**/**") // dots-invalid.js line 173
						assertMatch(t, false, "abc/../", "**/**") // dots-invalid.js line 175
						assertMatch(t, false, "abc/../", "**/**/") // dots-invalid.js line 176
						assertMatch(t, false, "abc/../", "**/**/**") // dots-invalid.js line 177
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**") // dots-invalid.js line 179
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**/") // dots-invalid.js line 180
						assertMatch(t, false, "abc/../abc/abc/../", "**/**/**/**/**") // dots-invalid.js line 181
					})
					// dots-invalid.js line 184
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/.**") // dots-invalid.js line 185
						assertMatch(t, false, "abc/..", "**/.**/") // dots-invalid.js line 186
						assertMatch(t, false, "abc/..", "**/.**/**") // dots-invalid.js line 187
						assertMatch(t, false, "abc/../", "**/.**") // dots-invalid.js line 189
						assertMatch(t, false, "abc/../", "**/.**/") // dots-invalid.js line 190
						assertMatch(t, false, "abc/../", "**/.**/**") // dots-invalid.js line 191
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**") // dots-invalid.js line 193
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**/") // dots-invalid.js line 194
						assertMatch(t, false, "abc/../abc/abc/../", "**/.**/**/.**/**") // dots-invalid.js line 195
					})
					// dots-invalid.js line 198
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.**") // dots-invalid.js line 199
						assertMatch(t, false, "abc/..", "**/**.**/") // dots-invalid.js line 200
						assertMatch(t, false, "abc/..", "**/**.**/**") // dots-invalid.js line 201
						assertMatch(t, false, "abc/../", "**/**.**") // dots-invalid.js line 203
						assertMatch(t, false, "abc/../", "**/**.**/") // dots-invalid.js line 204
						assertMatch(t, false, "abc/../", "**/**.**/**") // dots-invalid.js line 205
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**") // dots-invalid.js line 207
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**/") // dots-invalid.js line 208
						assertMatch(t, false, "abc/../abc/abc/../", "**/**.**/**/.**/**") // dots-invalid.js line 209
					})
					// dots-invalid.js line 212
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.") // dots-invalid.js line 213
						assertMatch(t, false, "abc/..", "**/**./") // dots-invalid.js line 214
						assertMatch(t, false, "abc/..", "**/**./**") // dots-invalid.js line 215
						assertMatch(t, false, "abc/../", "**/**.") // dots-invalid.js line 217
						assertMatch(t, false, "abc/../", "**/**./") // dots-invalid.js line 218
						assertMatch(t, false, "abc/../", "**/**./**") // dots-invalid.js line 219
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**.") // dots-invalid.js line 221
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**./") // dots-invalid.js line 222
						assertMatch(t, false, "abc/../abc/abc/../", "**/**./**/**./**") // dots-invalid.js line 223
					})
				})
			})
			// dots-invalid.js line 228
			t.Run("options_dot_true", func(t *testing.T) {
				// dots-invalid.js line 229
				t.Run("should_not_match_leading_double-dots", func(t *testing.T) {
					// dots-invalid.js line 230
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*/*", &Options{Dot: true}) // dots-invalid.js line 231
						assertMatch(t, false, "../abc", "*/abc", &Options{Dot: true}) // dots-invalid.js line 232
						assertMatch(t, false, "../abc", "*/abc/*", &Options{Dot: true}) // dots-invalid.js line 233
					})
					// dots-invalid.js line 236
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".*/*", &Options{Dot: true}) // dots-invalid.js line 237
						assertMatch(t, false, "../abc", ".*/abc", &Options{Dot: true}) // dots-invalid.js line 238
						assertMatch(t, false, "../abc", "*./*", &Options{Dot: true}) // dots-invalid.js line 240
						assertMatch(t, false, "../abc", "*./abc", &Options{Dot: true}) // dots-invalid.js line 241
					})
					// dots-invalid.js line 244
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**", &Options{Dot: true}) // dots-invalid.js line 245
						assertMatch(t, false, "../abc", "**/**", &Options{Dot: true}) // dots-invalid.js line 246
						assertMatch(t, false, "../abc", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 247
						assertMatch(t, false, "../abc", "**/abc", &Options{Dot: true}) // dots-invalid.js line 249
						assertMatch(t, false, "../abc", "**/abc/**", &Options{Dot: true}) // dots-invalid.js line 250
						assertMatch(t, false, "../abc", "abc/**", &Options{Dot: true}) // dots-invalid.js line 252
						assertMatch(t, false, "../abc", "abc/**/**", &Options{Dot: true}) // dots-invalid.js line 253
						assertMatch(t, false, "../abc", "abc/**/**/**", &Options{Dot: true}) // dots-invalid.js line 254
						assertMatch(t, false, "../abc", "**/abc", &Options{Dot: true}) // dots-invalid.js line 256
						assertMatch(t, false, "../abc", "**/abc/**", &Options{Dot: true}) // dots-invalid.js line 257
						assertMatch(t, false, "../abc", "**/abc/**/**", &Options{Dot: true}) // dots-invalid.js line 258
						assertMatch(t, false, "../abc", "**/**/abc/**", &Options{Dot: true}) // dots-invalid.js line 260
						assertMatch(t, false, "../abc", "**/**/abc/**/**", &Options{Dot: true}) // dots-invalid.js line 261
					})
					// dots-invalid.js line 264
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".**", &Options{Dot: true}) // dots-invalid.js line 265
						assertMatch(t, false, "../abc", ".**/**", &Options{Dot: true}) // dots-invalid.js line 266
						assertMatch(t, false, "../abc", ".**/abc", &Options{Dot: true}) // dots-invalid.js line 267
					})
					// dots-invalid.js line 270
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*.*/**", &Options{Dot: true}) // dots-invalid.js line 271
						assertMatch(t, false, "../abc", "*.*/abc", &Options{Dot: true}) // dots-invalid.js line 272
					})
					// dots-invalid.js line 275
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**./**", &Options{Dot: true}) // dots-invalid.js line 276
						assertMatch(t, false, "../abc", "**./abc", &Options{Dot: true}) // dots-invalid.js line 277
					})
				})
				// dots-invalid.js line 281
				t.Run("should_not_match_nested_double-dots", func(t *testing.T) {
					// dots-invalid.js line 282
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/*", &Options{Dot: true}) // dots-invalid.js line 283
						assertMatch(t, false, "/../abc", "/*/*", &Options{Dot: true}) // dots-invalid.js line 284
						assertMatch(t, false, "/../abc", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 285
						assertMatch(t, false, "abc/../abc", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 287
						assertMatch(t, false, "abc/../abc/abc", "*/*/*/*", &Options{Dot: true}) // dots-invalid.js line 288
					})
					// dots-invalid.js line 291
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 292
						assertMatch(t, false, "/../abc", "/.*/*", &Options{Dot: true}) // dots-invalid.js line 293
						assertMatch(t, false, "/../abc", "*/*.*/*", &Options{Dot: true}) // dots-invalid.js line 295
						assertMatch(t, false, "/../abc", "/*.*/*", &Options{Dot: true}) // dots-invalid.js line 296
						assertMatch(t, false, "/../abc", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 298
						assertMatch(t, false, "/../abc", "/*./*", &Options{Dot: true}) // dots-invalid.js line 299
						assertMatch(t, false, "abc/../abc", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 301
						assertMatch(t, false, "abc/../abc", "*/*.*/*", &Options{Dot: true}) // dots-invalid.js line 302
						assertMatch(t, false, "abc/../abc", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 303
					})
					// dots-invalid.js line 306
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**", &Options{Dot: true}) // dots-invalid.js line 307
						assertMatch(t, false, "/../abc", "**/**", &Options{Dot: true}) // dots-invalid.js line 308
						assertMatch(t, false, "/../abc", "/**/**", &Options{Dot: true}) // dots-invalid.js line 309
						assertMatch(t, false, "/../abc", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 310
						assertMatch(t, false, "abc/../abc", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 312
						assertMatch(t, false, "abc/../abc/abc", "**/**/**/**", &Options{Dot: true}) // dots-invalid.js line 313
					})
					// dots-invalid.js line 316
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 317
						assertMatch(t, false, "/../abc", "/.**/**", &Options{Dot: true}) // dots-invalid.js line 318
						assertMatch(t, false, "abc/../abc", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 320
						assertMatch(t, false, "abc/../abc", "/.**/**", &Options{Dot: true}) // dots-invalid.js line 321
					})
					// dots-invalid.js line 324
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 325
						assertMatch(t, false, "/../abc", "/**./**", &Options{Dot: true}) // dots-invalid.js line 326
						assertMatch(t, false, "abc/../abc", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 328
						assertMatch(t, false, "abc/../abc", "/**./**", &Options{Dot: true}) // dots-invalid.js line 329
					})
					// dots-invalid.js line 332
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 333
						assertMatch(t, false, "/../abc", "**/*.*/**", &Options{Dot: true}) // dots-invalid.js line 334
						assertMatch(t, false, "/../abc", "/**.**/**", &Options{Dot: true}) // dots-invalid.js line 336
						assertMatch(t, false, "/../abc", "/*.*/**", &Options{Dot: true}) // dots-invalid.js line 337
						assertMatch(t, false, "abc/../abc", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 339
						assertMatch(t, false, "abc/../abc", "**/*.*/**", &Options{Dot: true}) // dots-invalid.js line 340
						assertMatch(t, false, "abc/../abc", "/**.**/**", &Options{Dot: true}) // dots-invalid.js line 342
						assertMatch(t, false, "abc/../abc", "/*.*/**", &Options{Dot: true}) // dots-invalid.js line 343
					})
				})
				// dots-invalid.js line 347
				t.Run("should_not_match_trailing_double-dots", func(t *testing.T) {
					// dots-invalid.js line 348
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*", &Options{Dot: true}) // dots-invalid.js line 349
						assertMatch(t, false, "abc/..", "*/*/", &Options{Dot: true}) // dots-invalid.js line 350
						assertMatch(t, false, "abc/..", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 351
						assertMatch(t, false, "abc/../", "*/*", &Options{Dot: true}) // dots-invalid.js line 353
						assertMatch(t, false, "abc/../", "*/*/", &Options{Dot: true}) // dots-invalid.js line 354
						assertMatch(t, false, "abc/../", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 355
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*", &Options{Dot: true}) // dots-invalid.js line 357
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*/", &Options{Dot: true}) // dots-invalid.js line 358
						assertMatch(t, false, "abc/../abc/abc/../", "*/*/*/*/*", &Options{Dot: true}) // dots-invalid.js line 359
					})
					// dots-invalid.js line 362
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/.*", &Options{Dot: true}) // dots-invalid.js line 363
						assertMatch(t, false, "abc/..", "*/.*/", &Options{Dot: true}) // dots-invalid.js line 364
						assertMatch(t, false, "abc/..", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 365
						assertMatch(t, false, "abc/../", "*/.*", &Options{Dot: true}) // dots-invalid.js line 367
						assertMatch(t, false, "abc/../", "*/.*/", &Options{Dot: true}) // dots-invalid.js line 368
						assertMatch(t, false, "abc/../", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 369
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*", &Options{Dot: true}) // dots-invalid.js line 371
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*/", &Options{Dot: true}) // dots-invalid.js line 372
						assertMatch(t, false, "abc/../abc/abc/../", "*/.*/*/.*/*", &Options{Dot: true}) // dots-invalid.js line 373
					})
					// dots-invalid.js line 376
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*.", &Options{Dot: true}) // dots-invalid.js line 377
						assertMatch(t, false, "abc/..", "*/*./", &Options{Dot: true}) // dots-invalid.js line 378
						assertMatch(t, false, "abc/..", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 379
						assertMatch(t, false, "abc/../", "*/*.", &Options{Dot: true}) // dots-invalid.js line 381
						assertMatch(t, false, "abc/../", "*/*./", &Options{Dot: true}) // dots-invalid.js line 382
						assertMatch(t, false, "abc/../", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 383
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*.", &Options{Dot: true}) // dots-invalid.js line 385
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*./", &Options{Dot: true}) // dots-invalid.js line 386
						assertMatch(t, false, "abc/../abc/abc/../", "*/*./*/*./*", &Options{Dot: true}) // dots-invalid.js line 387
					})
					// dots-invalid.js line 390
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**", &Options{Dot: true}) // dots-invalid.js line 391
						assertMatch(t, false, "abc/..", "**/**/", &Options{Dot: true}) // dots-invalid.js line 392
						assertMatch(t, false, "abc/..", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 393
						assertMatch(t, false, "abc/../", "**/**", &Options{Dot: true}) // dots-invalid.js line 395
						assertMatch(t, false, "abc/../", "**/**/", &Options{Dot: true}) // dots-invalid.js line 396
						assertMatch(t, false, "abc/../", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 397
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**", &Options{Dot: true}) // dots-invalid.js line 399
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**/", &Options{Dot: true}) // dots-invalid.js line 400
						assertMatch(t, false, "abc/../abc/abc/../", "**/**/**/**/**", &Options{Dot: true}) // dots-invalid.js line 401
					})
					// dots-invalid.js line 404
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/.**", &Options{Dot: true}) // dots-invalid.js line 405
						assertMatch(t, false, "abc/..", "**/.**/", &Options{Dot: true}) // dots-invalid.js line 406
						assertMatch(t, false, "abc/..", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 407
						assertMatch(t, false, "abc/../", "**/.**", &Options{Dot: true}) // dots-invalid.js line 409
						assertMatch(t, false, "abc/../", "**/.**/", &Options{Dot: true}) // dots-invalid.js line 410
						assertMatch(t, false, "abc/../", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 411
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**", &Options{Dot: true}) // dots-invalid.js line 413
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**/", &Options{Dot: true}) // dots-invalid.js line 414
						assertMatch(t, false, "abc/../abc/abc/../", "**/.**/**/.**/**", &Options{Dot: true}) // dots-invalid.js line 415
					})
					// dots-invalid.js line 418
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.**", &Options{Dot: true}) // dots-invalid.js line 419
						assertMatch(t, false, "abc/..", "**/**.**/", &Options{Dot: true}) // dots-invalid.js line 420
						assertMatch(t, false, "abc/..", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 421
						assertMatch(t, false, "abc/../", "**/**.**", &Options{Dot: true}) // dots-invalid.js line 423
						assertMatch(t, false, "abc/../", "**/**.**/", &Options{Dot: true}) // dots-invalid.js line 424
						assertMatch(t, false, "abc/../", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 425
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**", &Options{Dot: true}) // dots-invalid.js line 427
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**/", &Options{Dot: true}) // dots-invalid.js line 428
						assertMatch(t, false, "abc/../abc/abc/../", "**/**.**/**/.**/**", &Options{Dot: true}) // dots-invalid.js line 429
					})
					// dots-invalid.js line 432
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.", &Options{Dot: true}) // dots-invalid.js line 433
						assertMatch(t, false, "abc/..", "**/**./", &Options{Dot: true}) // dots-invalid.js line 434
						assertMatch(t, false, "abc/..", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 435
						assertMatch(t, false, "abc/../", "**/**.", &Options{Dot: true}) // dots-invalid.js line 437
						assertMatch(t, false, "abc/../", "**/**./", &Options{Dot: true}) // dots-invalid.js line 438
						assertMatch(t, false, "abc/../", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 439
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**.", &Options{Dot: true}) // dots-invalid.js line 441
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**./", &Options{Dot: true}) // dots-invalid.js line 442
						assertMatch(t, false, "abc/../abc/abc/../", "**/**./**/**./**", &Options{Dot: true}) // dots-invalid.js line 443
					})
				})
			})
			// dots-invalid.js line 448
			t.Run("options_strictSlashes_true", func(t *testing.T) {
				// dots-invalid.js line 449
				t.Run("should_not_match_leading_double-dots", func(t *testing.T) {
					// dots-invalid.js line 450
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 451
						assertMatch(t, false, "../abc", "*/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 452
						assertMatch(t, false, "../abc", "*/abc/*", &Options{StrictSlashes: true}) // dots-invalid.js line 453
					})
					// dots-invalid.js line 456
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 457
						assertMatch(t, false, "../abc", ".*/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 458
						assertMatch(t, false, "../abc", "*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 460
						assertMatch(t, false, "../abc", "*./abc", &Options{StrictSlashes: true}) // dots-invalid.js line 461
					})
					// dots-invalid.js line 464
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**", &Options{StrictSlashes: true}) // dots-invalid.js line 465
						assertMatch(t, false, "../abc", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 466
						assertMatch(t, false, "../abc", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 467
						assertMatch(t, false, "../abc", "**/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 469
						assertMatch(t, false, "../abc", "**/abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 470
						assertMatch(t, false, "../abc", "abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 472
						assertMatch(t, false, "../abc", "abc/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 473
						assertMatch(t, false, "../abc", "abc/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 474
						assertMatch(t, false, "../abc", "**/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 476
						assertMatch(t, false, "../abc", "**/abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 477
						assertMatch(t, false, "../abc", "**/abc/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 478
						assertMatch(t, false, "../abc", "**/**/abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 480
						assertMatch(t, false, "../abc", "**/**/abc/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 481
					})
					// dots-invalid.js line 484
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".**", &Options{StrictSlashes: true}) // dots-invalid.js line 485
						assertMatch(t, false, "../abc", ".**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 486
						assertMatch(t, false, "../abc", ".**/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 487
					})
					// dots-invalid.js line 490
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 491
						assertMatch(t, false, "../abc", "*.*/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 492
					})
					// dots-invalid.js line 495
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 496
						assertMatch(t, false, "../abc", "**./abc", &Options{StrictSlashes: true}) // dots-invalid.js line 497
					})
				})
				// dots-invalid.js line 501
				t.Run("should_not_match_nested_double-dots", func(t *testing.T) {
					// dots-invalid.js line 502
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 503
						assertMatch(t, false, "/../abc", "/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 504
						assertMatch(t, false, "/../abc", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 505
						assertMatch(t, false, "abc/../abc", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 507
						assertMatch(t, false, "abc/../abc/abc", "*/*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 508
					})
					// dots-invalid.js line 511
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 512
						assertMatch(t, false, "/../abc", "/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 513
						assertMatch(t, false, "/../abc", "*/*.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 515
						assertMatch(t, false, "/../abc", "/*.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 516
						assertMatch(t, false, "/../abc", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 518
						assertMatch(t, false, "/../abc", "/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 519
						assertMatch(t, false, "abc/../abc", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 521
						assertMatch(t, false, "abc/../abc", "*/*.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 522
						assertMatch(t, false, "abc/../abc", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 523
					})
					// dots-invalid.js line 526
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**", &Options{StrictSlashes: true}) // dots-invalid.js line 527
						assertMatch(t, false, "/../abc", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 528
						assertMatch(t, false, "/../abc", "/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 529
						assertMatch(t, false, "/../abc", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 530
						assertMatch(t, false, "abc/../abc", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 532
						assertMatch(t, false, "abc/../abc/abc", "**/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 533
					})
					// dots-invalid.js line 536
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 537
						assertMatch(t, false, "/../abc", "/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 538
						assertMatch(t, false, "abc/../abc", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 540
						assertMatch(t, false, "abc/../abc", "/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 541
					})
					// dots-invalid.js line 544
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 545
						assertMatch(t, false, "/../abc", "/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 546
						assertMatch(t, false, "abc/../abc", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 548
						assertMatch(t, false, "abc/../abc", "/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 549
					})
					// dots-invalid.js line 552
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 553
						assertMatch(t, false, "/../abc", "**/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 554
						assertMatch(t, false, "/../abc", "/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 556
						assertMatch(t, false, "/../abc", "/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 557
						assertMatch(t, false, "abc/../abc", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 559
						assertMatch(t, false, "abc/../abc", "**/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 560
						assertMatch(t, false, "abc/../abc", "/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 562
						assertMatch(t, false, "abc/../abc", "/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 563
					})
				})
				// dots-invalid.js line 567
				t.Run("should_not_match_trailing_double-dots", func(t *testing.T) {
					// dots-invalid.js line 568
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 569
						assertMatch(t, false, "abc/..", "*/*/", &Options{StrictSlashes: true}) // dots-invalid.js line 570
						assertMatch(t, false, "abc/..", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 571
						assertMatch(t, false, "abc/../", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 573
						assertMatch(t, false, "abc/../", "*/*/", &Options{StrictSlashes: true}) // dots-invalid.js line 574
						assertMatch(t, false, "abc/../", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 575
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 577
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*/", &Options{StrictSlashes: true}) // dots-invalid.js line 578
						assertMatch(t, false, "abc/../abc/abc/../", "*/*/*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 579
					})
					// dots-invalid.js line 582
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/.*", &Options{StrictSlashes: true}) // dots-invalid.js line 583
						assertMatch(t, false, "abc/..", "*/.*/", &Options{StrictSlashes: true}) // dots-invalid.js line 584
						assertMatch(t, false, "abc/..", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 585
						assertMatch(t, false, "abc/../", "*/.*", &Options{StrictSlashes: true}) // dots-invalid.js line 587
						assertMatch(t, false, "abc/../", "*/.*/", &Options{StrictSlashes: true}) // dots-invalid.js line 588
						assertMatch(t, false, "abc/../", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 589
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*", &Options{StrictSlashes: true}) // dots-invalid.js line 591
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*/", &Options{StrictSlashes: true}) // dots-invalid.js line 592
						assertMatch(t, false, "abc/../abc/abc/../", "*/.*/*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 593
					})
					// dots-invalid.js line 596
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*.", &Options{StrictSlashes: true}) // dots-invalid.js line 597
						assertMatch(t, false, "abc/..", "*/*./", &Options{StrictSlashes: true}) // dots-invalid.js line 598
						assertMatch(t, false, "abc/..", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 599
						assertMatch(t, false, "abc/../", "*/*.", &Options{StrictSlashes: true}) // dots-invalid.js line 601
						assertMatch(t, false, "abc/../", "*/*./", &Options{StrictSlashes: true}) // dots-invalid.js line 602
						assertMatch(t, false, "abc/../", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 603
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*.", &Options{StrictSlashes: true}) // dots-invalid.js line 605
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*./", &Options{StrictSlashes: true}) // dots-invalid.js line 606
						assertMatch(t, false, "abc/../abc/abc/../", "*/*./*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 607
					})
					// dots-invalid.js line 610
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 611
						assertMatch(t, false, "abc/..", "**/**/", &Options{StrictSlashes: true}) // dots-invalid.js line 612
						assertMatch(t, false, "abc/..", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 613
						assertMatch(t, false, "abc/../", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 615
						assertMatch(t, false, "abc/../", "**/**/", &Options{StrictSlashes: true}) // dots-invalid.js line 616
						assertMatch(t, false, "abc/../", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 617
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 619
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**/", &Options{StrictSlashes: true}) // dots-invalid.js line 620
						assertMatch(t, false, "abc/../abc/abc/../", "**/**/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 621
					})
					// dots-invalid.js line 624
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/.**", &Options{StrictSlashes: true}) // dots-invalid.js line 625
						assertMatch(t, false, "abc/..", "**/.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 626
						assertMatch(t, false, "abc/..", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 627
						assertMatch(t, false, "abc/../", "**/.**", &Options{StrictSlashes: true}) // dots-invalid.js line 629
						assertMatch(t, false, "abc/../", "**/.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 630
						assertMatch(t, false, "abc/../", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 631
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**", &Options{StrictSlashes: true}) // dots-invalid.js line 633
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 634
						assertMatch(t, false, "abc/../abc/abc/../", "**/.**/**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 635
					})
					// dots-invalid.js line 638
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.**", &Options{StrictSlashes: true}) // dots-invalid.js line 639
						assertMatch(t, false, "abc/..", "**/**.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 640
						assertMatch(t, false, "abc/..", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 641
						assertMatch(t, false, "abc/../", "**/**.**", &Options{StrictSlashes: true}) // dots-invalid.js line 643
						assertMatch(t, false, "abc/../", "**/**.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 644
						assertMatch(t, false, "abc/../", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 645
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**", &Options{StrictSlashes: true}) // dots-invalid.js line 647
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 648
						assertMatch(t, false, "abc/../abc/abc/../", "**/**.**/**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 649
					})
					// dots-invalid.js line 652
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.", &Options{StrictSlashes: true}) // dots-invalid.js line 653
						assertMatch(t, false, "abc/..", "**/**./", &Options{StrictSlashes: true}) // dots-invalid.js line 654
						assertMatch(t, false, "abc/..", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 655
						assertMatch(t, false, "abc/../", "**/**.", &Options{StrictSlashes: true}) // dots-invalid.js line 657
						assertMatch(t, false, "abc/../", "**/**./", &Options{StrictSlashes: true}) // dots-invalid.js line 658
						assertMatch(t, false, "abc/../", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 659
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**.", &Options{StrictSlashes: true}) // dots-invalid.js line 661
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**./", &Options{StrictSlashes: true}) // dots-invalid.js line 662
						assertMatch(t, false, "abc/../abc/abc/../", "**/**./**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 663
					})
				})
			})
			// dots-invalid.js line 668
			t.Run("options_dot_true_strictSlashes_true", func(t *testing.T) {
				// dots-invalid.js line 669
				t.Run("should_not_match_leading_double-dots", func(t *testing.T) {
					// dots-invalid.js line 670
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 671
						assertMatch(t, false, "../abc", "*/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 672
						assertMatch(t, false, "../abc", "*/abc/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 673
					})
					// dots-invalid.js line 676
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 677
						assertMatch(t, false, "../abc", ".*/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 678
						assertMatch(t, false, "../abc", "*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 680
						assertMatch(t, false, "../abc", "*./abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 681
					})
					// dots-invalid.js line 684
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 685
						assertMatch(t, false, "../abc", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 686
						assertMatch(t, false, "../abc", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 687
						assertMatch(t, false, "../abc", "**/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 689
						assertMatch(t, false, "../abc", "**/abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 690
						assertMatch(t, false, "../abc", "abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 692
						assertMatch(t, false, "../abc", "abc/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 693
						assertMatch(t, false, "../abc", "abc/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 694
						assertMatch(t, false, "../abc", "**/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 696
						assertMatch(t, false, "../abc", "**/abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 697
						assertMatch(t, false, "../abc", "**/abc/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 698
						assertMatch(t, false, "../abc", "**/**/abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 700
						assertMatch(t, false, "../abc", "**/**/abc/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 701
					})
					// dots-invalid.js line 704
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", ".**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 705
						assertMatch(t, false, "../abc", ".**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 706
						assertMatch(t, false, "../abc", ".**/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 707
					})
					// dots-invalid.js line 710
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "../abc", "*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 711
						assertMatch(t, false, "../abc", "*.*/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 712
					})
					// dots-invalid.js line 715
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "../abc", "**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 716
						assertMatch(t, false, "../abc", "**./abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 717
					})
				})
				// dots-invalid.js line 721
				t.Run("should_not_match_nested_double-dots", func(t *testing.T) {
					// dots-invalid.js line 722
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 723
						assertMatch(t, false, "/../abc", "/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 724
						assertMatch(t, false, "/../abc", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 725
						assertMatch(t, false, "abc/../abc", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 727
						assertMatch(t, false, "abc/../abc/abc", "*/*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 728
					})
					// dots-invalid.js line 731
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 732
						assertMatch(t, false, "/../abc", "/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 733
						assertMatch(t, false, "/../abc", "*/*.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 735
						assertMatch(t, false, "/../abc", "/*.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 736
						assertMatch(t, false, "/../abc", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 738
						assertMatch(t, false, "/../abc", "/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 739
						assertMatch(t, false, "abc/../abc", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 741
						assertMatch(t, false, "abc/../abc", "*/*.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 742
						assertMatch(t, false, "abc/../abc", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 743
					})
					// dots-invalid.js line 746
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 747
						assertMatch(t, false, "/../abc", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 748
						assertMatch(t, false, "/../abc", "/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 749
						assertMatch(t, false, "/../abc", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 750
						assertMatch(t, false, "abc/../abc", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 752
						assertMatch(t, false, "abc/../abc/abc", "**/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 753
					})
					// dots-invalid.js line 756
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 757
						assertMatch(t, false, "/../abc", "/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 758
						assertMatch(t, false, "abc/../abc", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 760
						assertMatch(t, false, "abc/../abc", "/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 761
					})
					// dots-invalid.js line 764
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 765
						assertMatch(t, false, "/../abc", "/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 766
						assertMatch(t, false, "abc/../abc", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 768
						assertMatch(t, false, "abc/../abc", "/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 769
					})
					// dots-invalid.js line 772
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/../abc", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 773
						assertMatch(t, false, "/../abc", "**/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 774
						assertMatch(t, false, "/../abc", "/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 776
						assertMatch(t, false, "/../abc", "/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 777
						assertMatch(t, false, "abc/../abc", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 779
						assertMatch(t, false, "abc/../abc", "**/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 780
						assertMatch(t, false, "abc/../abc", "/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 782
						assertMatch(t, false, "abc/../abc", "/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 783
					})
				})
				// dots-invalid.js line 787
				t.Run("should_not_match_trailing_double-dots", func(t *testing.T) {
					// dots-invalid.js line 788
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 789
						assertMatch(t, false, "abc/..", "*/*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 790
						assertMatch(t, false, "abc/..", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 791
						assertMatch(t, false, "abc/../", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 793
						assertMatch(t, false, "abc/../", "*/*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 794
						assertMatch(t, false, "abc/../", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 795
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 797
						assertMatch(t, false, "abc/../abc/../", "*/*/*/*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 798
						assertMatch(t, false, "abc/../abc/abc/../", "*/*/*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 799
					})
					// dots-invalid.js line 802
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/.*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 803
						assertMatch(t, false, "abc/..", "*/.*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 804
						assertMatch(t, false, "abc/..", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 805
						assertMatch(t, false, "abc/../", "*/.*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 807
						assertMatch(t, false, "abc/../", "*/.*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 808
						assertMatch(t, false, "abc/../", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 809
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 811
						assertMatch(t, false, "abc/../abc/../", "*/.*/*/.*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 812
						assertMatch(t, false, "abc/../abc/abc/../", "*/.*/*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 813
					})
					// dots-invalid.js line 816
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "*/*.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 817
						assertMatch(t, false, "abc/..", "*/*./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 818
						assertMatch(t, false, "abc/..", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 819
						assertMatch(t, false, "abc/../", "*/*.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 821
						assertMatch(t, false, "abc/../", "*/*./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 822
						assertMatch(t, false, "abc/../", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 823
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 825
						assertMatch(t, false, "abc/../abc/../", "*/*./*/*./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 826
						assertMatch(t, false, "abc/../abc/abc/../", "*/*./*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 827
					})
					// dots-invalid.js line 830
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 831
						assertMatch(t, false, "abc/..", "**/**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 832
						assertMatch(t, false, "abc/..", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 833
						assertMatch(t, false, "abc/../", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 835
						assertMatch(t, false, "abc/../", "**/**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 836
						assertMatch(t, false, "abc/../", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 837
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 839
						assertMatch(t, false, "abc/../abc/../", "**/**/**/**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 840
						assertMatch(t, false, "abc/../abc/abc/../", "**/**/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 841
					})
					// dots-invalid.js line 844
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 845
						assertMatch(t, false, "abc/..", "**/.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 846
						assertMatch(t, false, "abc/..", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 847
						assertMatch(t, false, "abc/../", "**/.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 849
						assertMatch(t, false, "abc/../", "**/.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 850
						assertMatch(t, false, "abc/../", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 851
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 853
						assertMatch(t, false, "abc/../abc/../", "**/.**/**/.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 854
						assertMatch(t, false, "abc/../abc/abc/../", "**/.**/**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 855
					})
					// dots-invalid.js line 858
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 859
						assertMatch(t, false, "abc/..", "**/**.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 860
						assertMatch(t, false, "abc/..", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 861
						assertMatch(t, false, "abc/../", "**/**.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 863
						assertMatch(t, false, "abc/../", "**/**.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 864
						assertMatch(t, false, "abc/../", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 865
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 867
						assertMatch(t, false, "abc/../abc/../", "**/**.**/**/**.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 868
						assertMatch(t, false, "abc/../abc/abc/../", "**/**.**/**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 869
					})
					// dots-invalid.js line 872
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/..", "**/**.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 873
						assertMatch(t, false, "abc/..", "**/**./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 874
						assertMatch(t, false, "abc/..", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 875
						assertMatch(t, false, "abc/../", "**/**.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 877
						assertMatch(t, false, "abc/../", "**/**./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 878
						assertMatch(t, false, "abc/../", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 879
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 881
						assertMatch(t, false, "abc/../abc/../", "**/**./**/**./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 882
						assertMatch(t, false, "abc/../abc/abc/../", "**/**./**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 883
					})
				})
			})
		})
		// dots-invalid.js line 889
		t.Run("single_dots", func(t *testing.T) {
			// dots-invalid.js line 890
			t.Run("no_options", func(t *testing.T) {
				// dots-invalid.js line 891
				t.Run("should_not_match_leading_single-dots", func(t *testing.T) {
					// dots-invalid.js line 892
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*") // dots-invalid.js line 893
						assertMatch(t, false, "./abc", "*/*") // dots-invalid.js line 894
						assertMatch(t, false, "./abc", "*/abc") // dots-invalid.js line 895
						assertMatch(t, false, "./abc", "*/abc/*") // dots-invalid.js line 896
					})
					// dots-invalid.js line 899
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".*/*") // dots-invalid.js line 900
						assertMatch(t, false, "./abc", ".*/abc") // dots-invalid.js line 901
						assertMatch(t, false, "./abc", "*./*") // dots-invalid.js line 903
						assertMatch(t, false, "./abc", "*./abc") // dots-invalid.js line 904
					})
					// dots-invalid.js line 907
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**") // dots-invalid.js line 908
						assertMatch(t, false, "./abc", "**/**") // dots-invalid.js line 909
						assertMatch(t, false, "./abc", "**/**/**") // dots-invalid.js line 910
						assertMatch(t, false, "./abc", "**/abc") // dots-invalid.js line 912
						assertMatch(t, false, "./abc", "**/abc/**") // dots-invalid.js line 913
						assertMatch(t, false, "./abc", "abc/**") // dots-invalid.js line 915
						assertMatch(t, false, "./abc", "abc/**/**") // dots-invalid.js line 916
						assertMatch(t, false, "./abc", "abc/**/**/**") // dots-invalid.js line 917
						assertMatch(t, false, "./abc", "**/abc") // dots-invalid.js line 919
						assertMatch(t, false, "./abc", "**/abc/**") // dots-invalid.js line 920
						assertMatch(t, false, "./abc", "**/abc/**/**") // dots-invalid.js line 921
						assertMatch(t, false, "./abc", "**/**/abc/**") // dots-invalid.js line 923
						assertMatch(t, false, "./abc", "**/**/abc/**/**") // dots-invalid.js line 924
					})
					// dots-invalid.js line 927
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".**") // dots-invalid.js line 928
						assertMatch(t, false, "./abc", ".**/**") // dots-invalid.js line 929
						assertMatch(t, false, "./abc", ".**/abc") // dots-invalid.js line 930
					})
					// dots-invalid.js line 933
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*.*/**") // dots-invalid.js line 934
						assertMatch(t, false, "./abc", "*.*/abc") // dots-invalid.js line 935
					})
					// dots-invalid.js line 938
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**./**") // dots-invalid.js line 939
						assertMatch(t, false, "./abc", "**./abc") // dots-invalid.js line 940
					})
				})
				// dots-invalid.js line 944
				t.Run("should_not_match_nested_single-dots", func(t *testing.T) {
					// dots-invalid.js line 945
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/*") // dots-invalid.js line 946
						assertMatch(t, false, "/./abc", "/*/*") // dots-invalid.js line 947
						assertMatch(t, false, "/./abc", "*/*/*") // dots-invalid.js line 948
						assertMatch(t, false, "abc/./abc", "*/*/*") // dots-invalid.js line 950
						assertMatch(t, false, "abc/./abc/abc", "*/*/*/*") // dots-invalid.js line 951
					})
					// dots-invalid.js line 954
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/.*/*") // dots-invalid.js line 955
						assertMatch(t, false, "/./abc", "/.*/*") // dots-invalid.js line 956
						assertMatch(t, false, "/./abc", "*/*.*/*") // dots-invalid.js line 958
						assertMatch(t, false, "/./abc", "/*.*/*") // dots-invalid.js line 959
						assertMatch(t, false, "/./abc", "*/*./*") // dots-invalid.js line 961
						assertMatch(t, false, "/./abc", "/*./*") // dots-invalid.js line 962
						assertMatch(t, false, "abc/./abc", "*/.*/*") // dots-invalid.js line 964
						assertMatch(t, false, "abc/./abc", "*/*.*/*") // dots-invalid.js line 965
						assertMatch(t, false, "abc/./abc", "*/*./*") // dots-invalid.js line 966
					})
					// dots-invalid.js line 969
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**") // dots-invalid.js line 970
						assertMatch(t, false, "/./abc", "**/**") // dots-invalid.js line 971
						assertMatch(t, false, "/./abc", "/**/**") // dots-invalid.js line 972
						assertMatch(t, false, "/./abc", "**/**/**") // dots-invalid.js line 973
						assertMatch(t, false, "abc/./abc", "**/**/**") // dots-invalid.js line 975
						assertMatch(t, false, "abc/./abc/abc", "**/**/**/**") // dots-invalid.js line 976
					})
					// dots-invalid.js line 979
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/.**/**") // dots-invalid.js line 980
						assertMatch(t, false, "/./abc", "/.**/**") // dots-invalid.js line 981
						assertMatch(t, false, "abc/./abc", "**/.**/**") // dots-invalid.js line 983
						assertMatch(t, false, "abc/./abc", "/.**/**") // dots-invalid.js line 984
					})
					// dots-invalid.js line 987
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**./**") // dots-invalid.js line 988
						assertMatch(t, false, "/./abc", "/**./**") // dots-invalid.js line 989
						assertMatch(t, false, "abc/./abc", "**/**./**") // dots-invalid.js line 991
						assertMatch(t, false, "abc/./abc", "/**./**") // dots-invalid.js line 992
					})
					// dots-invalid.js line 995
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**.**/**") // dots-invalid.js line 996
						assertMatch(t, false, "/./abc", "**/*.*/**") // dots-invalid.js line 997
						assertMatch(t, false, "/./abc", "/**.**/**") // dots-invalid.js line 999
						assertMatch(t, false, "/./abc", "/*.*/**") // dots-invalid.js line 1000
						assertMatch(t, false, "abc/./abc", "**/**.**/**") // dots-invalid.js line 1002
						assertMatch(t, false, "abc/./abc", "**/*.*/**") // dots-invalid.js line 1003
						assertMatch(t, false, "abc/./abc", "/**.**/**") // dots-invalid.js line 1005
						assertMatch(t, false, "abc/./abc", "/*.*/**") // dots-invalid.js line 1006
					})
				})
				// dots-invalid.js line 1010
				t.Run("should_not_match_trailing_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1011
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*") // dots-invalid.js line 1012
						assertMatch(t, false, "abc/.", "*/*/") // dots-invalid.js line 1013
						assertMatch(t, false, "abc/.", "*/*/*") // dots-invalid.js line 1014
						assertMatch(t, false, "abc/./", "*/*") // dots-invalid.js line 1016
						assertMatch(t, false, "abc/./", "*/*/") // dots-invalid.js line 1017
						assertMatch(t, false, "abc/./", "*/*/*") // dots-invalid.js line 1018
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*") // dots-invalid.js line 1020
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*/") // dots-invalid.js line 1021
						assertMatch(t, false, "abc/./abc/abc/./", "*/*/*/*/*") // dots-invalid.js line 1022
					})
					// dots-invalid.js line 1025
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/.*") // dots-invalid.js line 1026
						assertMatch(t, false, "abc/.", "*/.*/") // dots-invalid.js line 1027
						assertMatch(t, false, "abc/.", "*/.*/*") // dots-invalid.js line 1028
						assertMatch(t, false, "abc/./", "*/.*") // dots-invalid.js line 1030
						assertMatch(t, false, "abc/./", "*/.*/") // dots-invalid.js line 1031
						assertMatch(t, false, "abc/./", "*/.*/*") // dots-invalid.js line 1032
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*") // dots-invalid.js line 1034
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*/") // dots-invalid.js line 1035
						assertMatch(t, false, "abc/./abc/abc/./", "*/.*/*/.*/*") // dots-invalid.js line 1036
					})
					// dots-invalid.js line 1039
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*.") // dots-invalid.js line 1040
						assertMatch(t, false, "abc/.", "*/*./") // dots-invalid.js line 1041
						assertMatch(t, false, "abc/.", "*/*./*") // dots-invalid.js line 1042
						assertMatch(t, false, "abc/./", "*/*.") // dots-invalid.js line 1044
						assertMatch(t, false, "abc/./", "*/*./") // dots-invalid.js line 1045
						assertMatch(t, false, "abc/./", "*/*./*") // dots-invalid.js line 1046
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*.") // dots-invalid.js line 1048
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*./") // dots-invalid.js line 1049
						assertMatch(t, false, "abc/./abc/abc/./", "*/*./*/*./*") // dots-invalid.js line 1050
					})
					// dots-invalid.js line 1053
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**") // dots-invalid.js line 1054
						assertMatch(t, false, "abc/.", "**/**/") // dots-invalid.js line 1055
						assertMatch(t, false, "abc/.", "**/**/**") // dots-invalid.js line 1056
						assertMatch(t, false, "abc/./", "**/**") // dots-invalid.js line 1058
						assertMatch(t, false, "abc/./", "**/**/") // dots-invalid.js line 1059
						assertMatch(t, false, "abc/./", "**/**/**") // dots-invalid.js line 1060
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**") // dots-invalid.js line 1062
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**/") // dots-invalid.js line 1063
						assertMatch(t, false, "abc/./abc/abc/./", "**/**/**/**/**") // dots-invalid.js line 1064
					})
					// dots-invalid.js line 1067
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/.**") // dots-invalid.js line 1068
						assertMatch(t, false, "abc/.", "**/.**/") // dots-invalid.js line 1069
						assertMatch(t, false, "abc/.", "**/.**/**") // dots-invalid.js line 1070
						assertMatch(t, false, "abc/./", "**/.**") // dots-invalid.js line 1072
						assertMatch(t, false, "abc/./", "**/.**/") // dots-invalid.js line 1073
						assertMatch(t, false, "abc/./", "**/.**/**") // dots-invalid.js line 1074
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**") // dots-invalid.js line 1076
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**/") // dots-invalid.js line 1077
						assertMatch(t, false, "abc/./abc/abc/./", "**/.**/**/.**/**") // dots-invalid.js line 1078
					})
					// dots-invalid.js line 1081
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.**") // dots-invalid.js line 1082
						assertMatch(t, false, "abc/.", "**/**.**/") // dots-invalid.js line 1083
						assertMatch(t, false, "abc/.", "**/**.**/**") // dots-invalid.js line 1084
						assertMatch(t, false, "abc/./", "**/**.**") // dots-invalid.js line 1086
						assertMatch(t, false, "abc/./", "**/**.**/") // dots-invalid.js line 1087
						assertMatch(t, false, "abc/./", "**/**.**/**") // dots-invalid.js line 1088
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**") // dots-invalid.js line 1090
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**/") // dots-invalid.js line 1091
						assertMatch(t, false, "abc/./abc/abc/./", "**/**.**/**/.**/**") // dots-invalid.js line 1092
					})
					// dots-invalid.js line 1095
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.") // dots-invalid.js line 1096
						assertMatch(t, false, "abc/.", "**/**./") // dots-invalid.js line 1097
						assertMatch(t, false, "abc/.", "**/**./**") // dots-invalid.js line 1098
						assertMatch(t, false, "abc/./", "**/**.") // dots-invalid.js line 1100
						assertMatch(t, false, "abc/./", "**/**./") // dots-invalid.js line 1101
						assertMatch(t, false, "abc/./", "**/**./**") // dots-invalid.js line 1102
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**.") // dots-invalid.js line 1104
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**./") // dots-invalid.js line 1105
						assertMatch(t, false, "abc/./abc/abc/./", "**/**./**/**./**") // dots-invalid.js line 1106
					})
				})
			})
			// dots-invalid.js line 1111
			t.Run("options_dot_true", func(t *testing.T) {
				// dots-invalid.js line 1112
				t.Run("should_not_match_leading_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1113
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*/*", &Options{Dot: true}) // dots-invalid.js line 1114
						assertMatch(t, false, "./abc", "*/abc", &Options{Dot: true}) // dots-invalid.js line 1115
						assertMatch(t, false, "./abc", "*/abc/*", &Options{Dot: true}) // dots-invalid.js line 1116
					})
					// dots-invalid.js line 1119
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".*/*", &Options{Dot: true}) // dots-invalid.js line 1120
						assertMatch(t, false, "./abc", ".*/abc", &Options{Dot: true}) // dots-invalid.js line 1121
						assertMatch(t, false, "./abc", "*./*", &Options{Dot: true}) // dots-invalid.js line 1123
						assertMatch(t, false, "./abc", "*./abc", &Options{Dot: true}) // dots-invalid.js line 1124
					})
					// dots-invalid.js line 1127
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**", &Options{Dot: true}) // dots-invalid.js line 1128
						assertMatch(t, false, "./abc", "**/**", &Options{Dot: true}) // dots-invalid.js line 1129
						assertMatch(t, false, "./abc", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 1130
						assertMatch(t, false, "./abc", "**/abc", &Options{Dot: true}) // dots-invalid.js line 1132
						assertMatch(t, false, "./abc", "**/abc/**", &Options{Dot: true}) // dots-invalid.js line 1133
						assertMatch(t, false, "./abc", "abc/**", &Options{Dot: true}) // dots-invalid.js line 1135
						assertMatch(t, false, "./abc", "abc/**/**", &Options{Dot: true}) // dots-invalid.js line 1136
						assertMatch(t, false, "./abc", "abc/**/**/**", &Options{Dot: true}) // dots-invalid.js line 1137
						assertMatch(t, false, "./abc", "**/abc", &Options{Dot: true}) // dots-invalid.js line 1139
						assertMatch(t, false, "./abc", "**/abc/**", &Options{Dot: true}) // dots-invalid.js line 1140
						assertMatch(t, false, "./abc", "**/abc/**/**", &Options{Dot: true}) // dots-invalid.js line 1141
						assertMatch(t, false, "./abc", "**/**/abc/**", &Options{Dot: true}) // dots-invalid.js line 1143
						assertMatch(t, false, "./abc", "**/**/abc/**/**", &Options{Dot: true}) // dots-invalid.js line 1144
					})
					// dots-invalid.js line 1147
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".**", &Options{Dot: true}) // dots-invalid.js line 1148
						assertMatch(t, false, "./abc", ".**/**", &Options{Dot: true}) // dots-invalid.js line 1149
						assertMatch(t, false, "./abc", ".**/abc", &Options{Dot: true}) // dots-invalid.js line 1150
					})
					// dots-invalid.js line 1153
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*.*/**", &Options{Dot: true}) // dots-invalid.js line 1154
						assertMatch(t, false, "./abc", "*.*/abc", &Options{Dot: true}) // dots-invalid.js line 1155
					})
					// dots-invalid.js line 1158
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**./**", &Options{Dot: true}) // dots-invalid.js line 1159
						assertMatch(t, false, "./abc", "**./abc", &Options{Dot: true}) // dots-invalid.js line 1160
					})
				})
				// dots-invalid.js line 1164
				t.Run("should_not_match_nested_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1165
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/*", &Options{Dot: true}) // dots-invalid.js line 1166
						assertMatch(t, false, "/./abc", "/*/*", &Options{Dot: true}) // dots-invalid.js line 1167
						assertMatch(t, false, "/./abc", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 1168
						assertMatch(t, false, "abc/./abc", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 1170
						assertMatch(t, false, "abc/./abc/abc", "*/*/*/*", &Options{Dot: true}) // dots-invalid.js line 1171
					})
					// dots-invalid.js line 1174
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 1175
						assertMatch(t, false, "/./abc", "/.*/*", &Options{Dot: true}) // dots-invalid.js line 1176
						assertMatch(t, false, "/./abc", "*/*.*/*", &Options{Dot: true}) // dots-invalid.js line 1178
						assertMatch(t, false, "/./abc", "/*.*/*", &Options{Dot: true}) // dots-invalid.js line 1179
						assertMatch(t, false, "/./abc", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 1181
						assertMatch(t, false, "/./abc", "/*./*", &Options{Dot: true}) // dots-invalid.js line 1182
						assertMatch(t, false, "abc/./abc", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 1184
						assertMatch(t, false, "abc/./abc", "*/*.*/*", &Options{Dot: true}) // dots-invalid.js line 1185
						assertMatch(t, false, "abc/./abc", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 1186
					})
					// dots-invalid.js line 1189
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**", &Options{Dot: true}) // dots-invalid.js line 1190
						assertMatch(t, false, "/./abc", "**/**", &Options{Dot: true}) // dots-invalid.js line 1191
						assertMatch(t, false, "/./abc", "/**/**", &Options{Dot: true}) // dots-invalid.js line 1192
						assertMatch(t, false, "/./abc", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 1193
						assertMatch(t, false, "abc/./abc", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 1195
						assertMatch(t, false, "abc/./abc/abc", "**/**/**/**", &Options{Dot: true}) // dots-invalid.js line 1196
					})
					// dots-invalid.js line 1199
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 1200
						assertMatch(t, false, "/./abc", "/.**/**", &Options{Dot: true}) // dots-invalid.js line 1201
						assertMatch(t, false, "abc/./abc", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 1203
						assertMatch(t, false, "abc/./abc", "/.**/**", &Options{Dot: true}) // dots-invalid.js line 1204
					})
					// dots-invalid.js line 1207
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 1208
						assertMatch(t, false, "/./abc", "/**./**", &Options{Dot: true}) // dots-invalid.js line 1209
						assertMatch(t, false, "abc/./abc", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 1211
						assertMatch(t, false, "abc/./abc", "/**./**", &Options{Dot: true}) // dots-invalid.js line 1212
					})
					// dots-invalid.js line 1215
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 1216
						assertMatch(t, false, "/./abc", "**/*.*/**", &Options{Dot: true}) // dots-invalid.js line 1217
						assertMatch(t, false, "/./abc", "/**.**/**", &Options{Dot: true}) // dots-invalid.js line 1219
						assertMatch(t, false, "/./abc", "/*.*/**", &Options{Dot: true}) // dots-invalid.js line 1220
						assertMatch(t, false, "abc/./abc", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 1222
						assertMatch(t, false, "abc/./abc", "**/*.*/**", &Options{Dot: true}) // dots-invalid.js line 1223
						assertMatch(t, false, "abc/./abc", "/**.**/**", &Options{Dot: true}) // dots-invalid.js line 1225
						assertMatch(t, false, "abc/./abc", "/*.*/**", &Options{Dot: true}) // dots-invalid.js line 1226
					})
				})
				// dots-invalid.js line 1230
				t.Run("should_not_match_trailing_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1231
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*", &Options{Dot: true}) // dots-invalid.js line 1232
						assertMatch(t, false, "abc/.", "*/*/", &Options{Dot: true}) // dots-invalid.js line 1233
						assertMatch(t, false, "abc/.", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 1234
						assertMatch(t, false, "abc/./", "*/*", &Options{Dot: true}) // dots-invalid.js line 1236
						assertMatch(t, false, "abc/./", "*/*/", &Options{Dot: true}) // dots-invalid.js line 1237
						assertMatch(t, false, "abc/./", "*/*/*", &Options{Dot: true}) // dots-invalid.js line 1238
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*", &Options{Dot: true}) // dots-invalid.js line 1240
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*/", &Options{Dot: true}) // dots-invalid.js line 1241
						assertMatch(t, false, "abc/./abc/abc/./", "*/*/*/*/*", &Options{Dot: true}) // dots-invalid.js line 1242
					})
					// dots-invalid.js line 1245
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/.*", &Options{Dot: true}) // dots-invalid.js line 1246
						assertMatch(t, false, "abc/.", "*/.*/", &Options{Dot: true}) // dots-invalid.js line 1247
						assertMatch(t, false, "abc/.", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 1248
						assertMatch(t, false, "abc/./", "*/.*", &Options{Dot: true}) // dots-invalid.js line 1250
						assertMatch(t, false, "abc/./", "*/.*/", &Options{Dot: true}) // dots-invalid.js line 1251
						assertMatch(t, false, "abc/./", "*/.*/*", &Options{Dot: true}) // dots-invalid.js line 1252
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*", &Options{Dot: true}) // dots-invalid.js line 1254
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*/", &Options{Dot: true}) // dots-invalid.js line 1255
						assertMatch(t, false, "abc/./abc/abc/./", "*/.*/*/.*/*", &Options{Dot: true}) // dots-invalid.js line 1256
					})
					// dots-invalid.js line 1259
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*.", &Options{Dot: true}) // dots-invalid.js line 1260
						assertMatch(t, false, "abc/.", "*/*./", &Options{Dot: true}) // dots-invalid.js line 1261
						assertMatch(t, false, "abc/.", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 1262
						assertMatch(t, false, "abc/./", "*/*.", &Options{Dot: true}) // dots-invalid.js line 1264
						assertMatch(t, false, "abc/./", "*/*./", &Options{Dot: true}) // dots-invalid.js line 1265
						assertMatch(t, false, "abc/./", "*/*./*", &Options{Dot: true}) // dots-invalid.js line 1266
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*.", &Options{Dot: true}) // dots-invalid.js line 1268
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*./", &Options{Dot: true}) // dots-invalid.js line 1269
						assertMatch(t, false, "abc/./abc/abc/./", "*/*./*/*./*", &Options{Dot: true}) // dots-invalid.js line 1270
					})
					// dots-invalid.js line 1273
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**", &Options{Dot: true}) // dots-invalid.js line 1274
						assertMatch(t, false, "abc/.", "**/**/", &Options{Dot: true}) // dots-invalid.js line 1275
						assertMatch(t, false, "abc/.", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 1276
						assertMatch(t, false, "abc/./", "**/**", &Options{Dot: true}) // dots-invalid.js line 1278
						assertMatch(t, false, "abc/./", "**/**/", &Options{Dot: true}) // dots-invalid.js line 1279
						assertMatch(t, false, "abc/./", "**/**/**", &Options{Dot: true}) // dots-invalid.js line 1280
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**", &Options{Dot: true}) // dots-invalid.js line 1282
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**/", &Options{Dot: true}) // dots-invalid.js line 1283
						assertMatch(t, false, "abc/./abc/abc/./", "**/**/**/**/**", &Options{Dot: true}) // dots-invalid.js line 1284
					})
					// dots-invalid.js line 1287
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/.**", &Options{Dot: true}) // dots-invalid.js line 1288
						assertMatch(t, false, "abc/.", "**/.**/", &Options{Dot: true}) // dots-invalid.js line 1289
						assertMatch(t, false, "abc/.", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 1290
						assertMatch(t, false, "abc/./", "**/.**", &Options{Dot: true}) // dots-invalid.js line 1292
						assertMatch(t, false, "abc/./", "**/.**/", &Options{Dot: true}) // dots-invalid.js line 1293
						assertMatch(t, false, "abc/./", "**/.**/**", &Options{Dot: true}) // dots-invalid.js line 1294
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**", &Options{Dot: true}) // dots-invalid.js line 1296
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**/", &Options{Dot: true}) // dots-invalid.js line 1297
						assertMatch(t, false, "abc/./abc/abc/./", "**/.**/**/.**/**", &Options{Dot: true}) // dots-invalid.js line 1298
					})
					// dots-invalid.js line 1301
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.**", &Options{Dot: true}) // dots-invalid.js line 1302
						assertMatch(t, false, "abc/.", "**/**.**/", &Options{Dot: true}) // dots-invalid.js line 1303
						assertMatch(t, false, "abc/.", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 1304
						assertMatch(t, false, "abc/./", "**/**.**", &Options{Dot: true}) // dots-invalid.js line 1306
						assertMatch(t, false, "abc/./", "**/**.**/", &Options{Dot: true}) // dots-invalid.js line 1307
						assertMatch(t, false, "abc/./", "**/**.**/**", &Options{Dot: true}) // dots-invalid.js line 1308
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**", &Options{Dot: true}) // dots-invalid.js line 1310
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**/", &Options{Dot: true}) // dots-invalid.js line 1311
						assertMatch(t, false, "abc/./abc/abc/./", "**/**.**/**/.**/**", &Options{Dot: true}) // dots-invalid.js line 1312
					})
					// dots-invalid.js line 1315
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.", &Options{Dot: true}) // dots-invalid.js line 1316
						assertMatch(t, false, "abc/.", "**/**./", &Options{Dot: true}) // dots-invalid.js line 1317
						assertMatch(t, false, "abc/.", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 1318
						assertMatch(t, false, "abc/./", "**/**.", &Options{Dot: true}) // dots-invalid.js line 1320
						assertMatch(t, false, "abc/./", "**/**./", &Options{Dot: true}) // dots-invalid.js line 1321
						assertMatch(t, false, "abc/./", "**/**./**", &Options{Dot: true}) // dots-invalid.js line 1322
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**.", &Options{Dot: true}) // dots-invalid.js line 1324
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**./", &Options{Dot: true}) // dots-invalid.js line 1325
						assertMatch(t, false, "abc/./abc/abc/./", "**/**./**/**./**", &Options{Dot: true}) // dots-invalid.js line 1326
					})
				})
			})
			// dots-invalid.js line 1331
			t.Run("options_strictSlashes_true", func(t *testing.T) {
				// dots-invalid.js line 1332
				t.Run("should_not_match_leading_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1333
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1334
						assertMatch(t, false, "./abc", "*/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1335
						assertMatch(t, false, "./abc", "*/abc/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1336
					})
					// dots-invalid.js line 1339
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1340
						assertMatch(t, false, "./abc", ".*/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1341
						assertMatch(t, false, "./abc", "*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 1343
						assertMatch(t, false, "./abc", "*./abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1344
					})
					// dots-invalid.js line 1347
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**", &Options{StrictSlashes: true}) // dots-invalid.js line 1348
						assertMatch(t, false, "./abc", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1349
						assertMatch(t, false, "./abc", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1350
						assertMatch(t, false, "./abc", "**/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1352
						assertMatch(t, false, "./abc", "**/abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1353
						assertMatch(t, false, "./abc", "abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1355
						assertMatch(t, false, "./abc", "abc/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1356
						assertMatch(t, false, "./abc", "abc/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1357
						assertMatch(t, false, "./abc", "**/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1359
						assertMatch(t, false, "./abc", "**/abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1360
						assertMatch(t, false, "./abc", "**/abc/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1361
						assertMatch(t, false, "./abc", "**/**/abc/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1363
						assertMatch(t, false, "./abc", "**/**/abc/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1364
					})
					// dots-invalid.js line 1367
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".**", &Options{StrictSlashes: true}) // dots-invalid.js line 1368
						assertMatch(t, false, "./abc", ".**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1369
						assertMatch(t, false, "./abc", ".**/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1370
					})
					// dots-invalid.js line 1373
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1374
						assertMatch(t, false, "./abc", "*.*/abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1375
					})
					// dots-invalid.js line 1378
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1379
						assertMatch(t, false, "./abc", "**./abc", &Options{StrictSlashes: true}) // dots-invalid.js line 1380
					})
				})
				// dots-invalid.js line 1384
				t.Run("should_not_match_nested_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1385
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1386
						assertMatch(t, false, "/./abc", "/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1387
						assertMatch(t, false, "/./abc", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1388
						assertMatch(t, false, "abc/./abc", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1390
						assertMatch(t, false, "abc/./abc/abc", "*/*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1391
					})
					// dots-invalid.js line 1394
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1395
						assertMatch(t, false, "/./abc", "/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1396
						assertMatch(t, false, "/./abc", "*/*.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1398
						assertMatch(t, false, "/./abc", "/*.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1399
						assertMatch(t, false, "/./abc", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 1401
						assertMatch(t, false, "/./abc", "/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 1402
						assertMatch(t, false, "abc/./abc", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1404
						assertMatch(t, false, "abc/./abc", "*/*.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1405
						assertMatch(t, false, "abc/./abc", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 1406
					})
					// dots-invalid.js line 1409
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**", &Options{StrictSlashes: true}) // dots-invalid.js line 1410
						assertMatch(t, false, "/./abc", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1411
						assertMatch(t, false, "/./abc", "/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1412
						assertMatch(t, false, "/./abc", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1413
						assertMatch(t, false, "abc/./abc", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1415
						assertMatch(t, false, "abc/./abc/abc", "**/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1416
					})
					// dots-invalid.js line 1419
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1420
						assertMatch(t, false, "/./abc", "/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1421
						assertMatch(t, false, "abc/./abc", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1423
						assertMatch(t, false, "abc/./abc", "/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1424
					})
					// dots-invalid.js line 1427
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1428
						assertMatch(t, false, "/./abc", "/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1429
						assertMatch(t, false, "abc/./abc", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1431
						assertMatch(t, false, "abc/./abc", "/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1432
					})
					// dots-invalid.js line 1435
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1436
						assertMatch(t, false, "/./abc", "**/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1437
						assertMatch(t, false, "/./abc", "/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1439
						assertMatch(t, false, "/./abc", "/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1440
						assertMatch(t, false, "abc/./abc", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1442
						assertMatch(t, false, "abc/./abc", "**/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1443
						assertMatch(t, false, "abc/./abc", "/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1445
						assertMatch(t, false, "abc/./abc", "/*.*/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1446
					})
				})
				// dots-invalid.js line 1450
				t.Run("should_not_match_trailing_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1451
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1452
						assertMatch(t, false, "abc/.", "*/*/", &Options{StrictSlashes: true}) // dots-invalid.js line 1453
						assertMatch(t, false, "abc/.", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1454
						assertMatch(t, false, "abc/./", "*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1456
						assertMatch(t, false, "abc/./", "*/*/", &Options{StrictSlashes: true}) // dots-invalid.js line 1457
						assertMatch(t, false, "abc/./", "*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1458
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1460
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*/", &Options{StrictSlashes: true}) // dots-invalid.js line 1461
						assertMatch(t, false, "abc/./abc/abc/./", "*/*/*/*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1462
					})
					// dots-invalid.js line 1465
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/.*", &Options{StrictSlashes: true}) // dots-invalid.js line 1466
						assertMatch(t, false, "abc/.", "*/.*/", &Options{StrictSlashes: true}) // dots-invalid.js line 1467
						assertMatch(t, false, "abc/.", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1468
						assertMatch(t, false, "abc/./", "*/.*", &Options{StrictSlashes: true}) // dots-invalid.js line 1470
						assertMatch(t, false, "abc/./", "*/.*/", &Options{StrictSlashes: true}) // dots-invalid.js line 1471
						assertMatch(t, false, "abc/./", "*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1472
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*", &Options{StrictSlashes: true}) // dots-invalid.js line 1474
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*/", &Options{StrictSlashes: true}) // dots-invalid.js line 1475
						assertMatch(t, false, "abc/./abc/abc/./", "*/.*/*/.*/*", &Options{StrictSlashes: true}) // dots-invalid.js line 1476
					})
					// dots-invalid.js line 1479
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*.", &Options{StrictSlashes: true}) // dots-invalid.js line 1480
						assertMatch(t, false, "abc/.", "*/*./", &Options{StrictSlashes: true}) // dots-invalid.js line 1481
						assertMatch(t, false, "abc/.", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 1482
						assertMatch(t, false, "abc/./", "*/*.", &Options{StrictSlashes: true}) // dots-invalid.js line 1484
						assertMatch(t, false, "abc/./", "*/*./", &Options{StrictSlashes: true}) // dots-invalid.js line 1485
						assertMatch(t, false, "abc/./", "*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 1486
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*.", &Options{StrictSlashes: true}) // dots-invalid.js line 1488
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*./", &Options{StrictSlashes: true}) // dots-invalid.js line 1489
						assertMatch(t, false, "abc/./abc/abc/./", "*/*./*/*./*", &Options{StrictSlashes: true}) // dots-invalid.js line 1490
					})
					// dots-invalid.js line 1493
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1494
						assertMatch(t, false, "abc/.", "**/**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1495
						assertMatch(t, false, "abc/.", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1496
						assertMatch(t, false, "abc/./", "**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1498
						assertMatch(t, false, "abc/./", "**/**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1499
						assertMatch(t, false, "abc/./", "**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1500
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1502
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1503
						assertMatch(t, false, "abc/./abc/abc/./", "**/**/**/**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1504
					})
					// dots-invalid.js line 1507
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/.**", &Options{StrictSlashes: true}) // dots-invalid.js line 1508
						assertMatch(t, false, "abc/.", "**/.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1509
						assertMatch(t, false, "abc/.", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1510
						assertMatch(t, false, "abc/./", "**/.**", &Options{StrictSlashes: true}) // dots-invalid.js line 1512
						assertMatch(t, false, "abc/./", "**/.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1513
						assertMatch(t, false, "abc/./", "**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1514
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**", &Options{StrictSlashes: true}) // dots-invalid.js line 1516
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1517
						assertMatch(t, false, "abc/./abc/abc/./", "**/.**/**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1518
					})
					// dots-invalid.js line 1521
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.**", &Options{StrictSlashes: true}) // dots-invalid.js line 1522
						assertMatch(t, false, "abc/.", "**/**.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1523
						assertMatch(t, false, "abc/.", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1524
						assertMatch(t, false, "abc/./", "**/**.**", &Options{StrictSlashes: true}) // dots-invalid.js line 1526
						assertMatch(t, false, "abc/./", "**/**.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1527
						assertMatch(t, false, "abc/./", "**/**.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1528
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**", &Options{StrictSlashes: true}) // dots-invalid.js line 1530
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**/", &Options{StrictSlashes: true}) // dots-invalid.js line 1531
						assertMatch(t, false, "abc/./abc/abc/./", "**/**.**/**/.**/**", &Options{StrictSlashes: true}) // dots-invalid.js line 1532
					})
					// dots-invalid.js line 1535
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.", &Options{StrictSlashes: true}) // dots-invalid.js line 1536
						assertMatch(t, false, "abc/.", "**/**./", &Options{StrictSlashes: true}) // dots-invalid.js line 1537
						assertMatch(t, false, "abc/.", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1538
						assertMatch(t, false, "abc/./", "**/**.", &Options{StrictSlashes: true}) // dots-invalid.js line 1540
						assertMatch(t, false, "abc/./", "**/**./", &Options{StrictSlashes: true}) // dots-invalid.js line 1541
						assertMatch(t, false, "abc/./", "**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1542
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**.", &Options{StrictSlashes: true}) // dots-invalid.js line 1544
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**./", &Options{StrictSlashes: true}) // dots-invalid.js line 1545
						assertMatch(t, false, "abc/./abc/abc/./", "**/**./**/**./**", &Options{StrictSlashes: true}) // dots-invalid.js line 1546
					})
				})
			})
			// dots-invalid.js line 1551
			t.Run("options_dot_true_strictSlashes_true", func(t *testing.T) {
				// dots-invalid.js line 1552
				t.Run("should_not_match_leading_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1553
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1554
						assertMatch(t, false, "./abc", "*/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1555
						assertMatch(t, false, "./abc", "*/abc/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1556
					})
					// dots-invalid.js line 1559
					t.Run("with_dot_plus_single_star", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1560
						assertMatch(t, false, "./abc", ".*/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1561
						assertMatch(t, false, "./abc", "*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1563
						assertMatch(t, false, "./abc", "*./abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1564
					})
					// dots-invalid.js line 1567
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1568
						assertMatch(t, false, "./abc", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1569
						assertMatch(t, false, "./abc", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1570
						assertMatch(t, false, "./abc", "**/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1572
						assertMatch(t, false, "./abc", "**/abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1573
						assertMatch(t, false, "./abc", "abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1575
						assertMatch(t, false, "./abc", "abc/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1576
						assertMatch(t, false, "./abc", "abc/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1577
						assertMatch(t, false, "./abc", "**/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1579
						assertMatch(t, false, "./abc", "**/abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1580
						assertMatch(t, false, "./abc", "**/abc/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1581
						assertMatch(t, false, "./abc", "**/**/abc/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1583
						assertMatch(t, false, "./abc", "**/**/abc/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1584
					})
					// dots-invalid.js line 1587
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", ".**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1588
						assertMatch(t, false, "./abc", ".**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1589
						assertMatch(t, false, "./abc", ".**/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1590
					})
					// dots-invalid.js line 1593
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "./abc", "*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1594
						assertMatch(t, false, "./abc", "*.*/abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1595
					})
					// dots-invalid.js line 1598
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "./abc", "**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1599
						assertMatch(t, false, "./abc", "**./abc", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1600
					})
				})
				// dots-invalid.js line 1604
				t.Run("should_not_match_nested_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1605
					t.Run("with_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1606
						assertMatch(t, false, "/./abc", "/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1607
						assertMatch(t, false, "/./abc", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1608
						assertMatch(t, false, "abc/./abc", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1610
						assertMatch(t, false, "abc/./abc/abc", "*/*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1611
					})
					// dots-invalid.js line 1614
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1615
						assertMatch(t, false, "/./abc", "/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1616
						assertMatch(t, false, "/./abc", "*/*.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1618
						assertMatch(t, false, "/./abc", "/*.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1619
						assertMatch(t, false, "/./abc", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1621
						assertMatch(t, false, "/./abc", "/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1622
						assertMatch(t, false, "abc/./abc", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1624
						assertMatch(t, false, "abc/./abc", "*/*.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1625
						assertMatch(t, false, "abc/./abc", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1626
					})
					// dots-invalid.js line 1629
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1630
						assertMatch(t, false, "/./abc", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1631
						assertMatch(t, false, "/./abc", "/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1632
						assertMatch(t, false, "/./abc", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1633
						assertMatch(t, false, "abc/./abc", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1635
						assertMatch(t, false, "abc/./abc/abc", "**/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1636
					})
					// dots-invalid.js line 1639
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1640
						assertMatch(t, false, "/./abc", "/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1641
						assertMatch(t, false, "abc/./abc", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1643
						assertMatch(t, false, "abc/./abc", "/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1644
					})
					// dots-invalid.js line 1647
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1648
						assertMatch(t, false, "/./abc", "/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1649
						assertMatch(t, false, "abc/./abc", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1651
						assertMatch(t, false, "abc/./abc", "/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1652
					})
					// dots-invalid.js line 1655
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "/./abc", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1656
						assertMatch(t, false, "/./abc", "**/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1657
						assertMatch(t, false, "/./abc", "/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1659
						assertMatch(t, false, "/./abc", "/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1660
						assertMatch(t, false, "abc/./abc", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1662
						assertMatch(t, false, "abc/./abc", "**/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1663
						assertMatch(t, false, "abc/./abc", "/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1665
						assertMatch(t, false, "abc/./abc", "/*.*/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1666
					})
				})
				// dots-invalid.js line 1670
				t.Run("should_not_match_trailing_single-dots", func(t *testing.T) {
					// dots-invalid.js line 1671
					t.Run("with_single_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1672
						assertMatch(t, false, "abc/.", "*/*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1673
						assertMatch(t, false, "abc/.", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1674
						assertMatch(t, false, "abc/./", "*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1676
						assertMatch(t, false, "abc/./", "*/*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1677
						assertMatch(t, false, "abc/./", "*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1678
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1680
						assertMatch(t, false, "abc/./abc/./", "*/*/*/*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1681
						assertMatch(t, false, "abc/./abc/abc/./", "*/*/*/*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1682
					})
					// dots-invalid.js line 1685
					t.Run("with_dot_plus_star", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/.*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1686
						assertMatch(t, false, "abc/.", "*/.*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1687
						assertMatch(t, false, "abc/.", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1688
						assertMatch(t, false, "abc/./", "*/.*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1690
						assertMatch(t, false, "abc/./", "*/.*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1691
						assertMatch(t, false, "abc/./", "*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1692
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1694
						assertMatch(t, false, "abc/./abc/./", "*/.*/*/.*/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1695
						assertMatch(t, false, "abc/./abc/abc/./", "*/.*/*/.*/*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1696
					})
					// dots-invalid.js line 1699
					t.Run("with_star_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "*/*.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1700
						assertMatch(t, false, "abc/.", "*/*./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1701
						assertMatch(t, false, "abc/.", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1702
						assertMatch(t, false, "abc/./", "*/*.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1704
						assertMatch(t, false, "abc/./", "*/*./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1705
						assertMatch(t, false, "abc/./", "*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1706
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1708
						assertMatch(t, false, "abc/./abc/./", "*/*./*/*./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1709
						assertMatch(t, false, "abc/./abc/abc/./", "*/*./*/*./*", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1710
					})
					// dots-invalid.js line 1713
					t.Run("with_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1714
						assertMatch(t, false, "abc/.", "**/**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1715
						assertMatch(t, false, "abc/.", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1716
						assertMatch(t, false, "abc/./", "**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1718
						assertMatch(t, false, "abc/./", "**/**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1719
						assertMatch(t, false, "abc/./", "**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1720
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1722
						assertMatch(t, false, "abc/./abc/./", "**/**/**/**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1723
						assertMatch(t, false, "abc/./abc/abc/./", "**/**/**/**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1724
					})
					// dots-invalid.js line 1727
					t.Run("with_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1728
						assertMatch(t, false, "abc/.", "**/.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1729
						assertMatch(t, false, "abc/.", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1730
						assertMatch(t, false, "abc/./", "**/.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1732
						assertMatch(t, false, "abc/./", "**/.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1733
						assertMatch(t, false, "abc/./", "**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1734
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1736
						assertMatch(t, false, "abc/./abc/./", "**/.**/**/.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1737
						assertMatch(t, false, "abc/./abc/abc/./", "**/.**/**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1738
					})
					// dots-invalid.js line 1741
					t.Run("with_globstar_plus_dot_plus_globstar", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1742
						assertMatch(t, false, "abc/.", "**/**.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1743
						assertMatch(t, false, "abc/.", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1744
						assertMatch(t, false, "abc/./", "**/**.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1746
						assertMatch(t, false, "abc/./", "**/**.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1747
						assertMatch(t, false, "abc/./", "**/**.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1748
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1750
						assertMatch(t, false, "abc/./abc/./", "**/**.**/**/**.**/", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1751
						assertMatch(t, false, "abc/./abc/abc/./", "**/**.**/**/.**/**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1752
					})
					// dots-invalid.js line 1755
					t.Run("with_globstar_plus_dot", func(t *testing.T) {
						assertMatch(t, false, "abc/.", "**/**.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1756
						assertMatch(t, false, "abc/.", "**/**./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1757
						assertMatch(t, false, "abc/.", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1758
						assertMatch(t, false, "abc/./", "**/**.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1760
						assertMatch(t, false, "abc/./", "**/**./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1761
						assertMatch(t, false, "abc/./", "**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1762
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**.", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1764
						assertMatch(t, false, "abc/./abc/./", "**/**./**/**./", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1765
						assertMatch(t, false, "abc/./abc/abc/./", "**/**./**/**./**", &Options{Dot: true, StrictSlashes: true}) // dots-invalid.js line 1766
					})
				})
			})
		})
	})
}
