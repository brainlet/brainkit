// api_scan_test.go — Faithful 1:1 port of picomatch/test/api.scan.js
package picomatch

import (
	"testing"
)

// scanBase is a helper that returns only the Base from Scan.
// Ported from: api.scan.js line 5
func scanBase(input string, opts ...*ScanOptions) string {
	var o *ScanOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return Scan(input, o).Base
}

// scanBoth is a helper that returns [Base, Glob] from Scan.
// Ported from: api.scan.js lines 6-9
func scanBoth(input string, opts ...*ScanOptions) [2]string {
	var o *ScanOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := Scan(input, o)
	return [2]string{s.Base, s.Glob}
}

// assertScanState compares a ScanState against expected values (without optional fields).
func assertScanState(t *testing.T, input string, opts *ScanOptions, expected ScanState) {
	t.Helper()
	got := Scan(input, opts)
	if got.Input != expected.Input {
		t.Errorf("Scan(%q).Input = %q, want %q", input, got.Input, expected.Input)
	}
	if got.Prefix != expected.Prefix {
		t.Errorf("Scan(%q).Prefix = %q, want %q", input, got.Prefix, expected.Prefix)
	}
	if got.Start != expected.Start {
		t.Errorf("Scan(%q).Start = %d, want %d", input, got.Start, expected.Start)
	}
	if got.Base != expected.Base {
		t.Errorf("Scan(%q).Base = %q, want %q", input, got.Base, expected.Base)
	}
	if got.Glob != expected.Glob {
		t.Errorf("Scan(%q).Glob = %q, want %q", input, got.Glob, expected.Glob)
	}
	if got.IsBrace != expected.IsBrace {
		t.Errorf("Scan(%q).IsBrace = %v, want %v", input, got.IsBrace, expected.IsBrace)
	}
	if got.IsBracket != expected.IsBracket {
		t.Errorf("Scan(%q).IsBracket = %v, want %v", input, got.IsBracket, expected.IsBracket)
	}
	if got.IsGlob != expected.IsGlob {
		t.Errorf("Scan(%q).IsGlob = %v, want %v", input, got.IsGlob, expected.IsGlob)
	}
	if got.IsExtglob != expected.IsExtglob {
		t.Errorf("Scan(%q).IsExtglob = %v, want %v", input, got.IsExtglob, expected.IsExtglob)
	}
	if got.IsGlobstar != expected.IsGlobstar {
		t.Errorf("Scan(%q).IsGlobstar = %v, want %v", input, got.IsGlobstar, expected.IsGlobstar)
	}
	if got.Negated != expected.Negated {
		t.Errorf("Scan(%q).Negated = %v, want %v", input, got.Negated, expected.Negated)
	}
	if got.NegatedExtglob != expected.NegatedExtglob {
		t.Errorf("Scan(%q).NegatedExtglob = %v, want %v", input, got.NegatedExtglob, expected.NegatedExtglob)
	}
}

// assertScanStateWithParts compares a ScanState including Slashes and Parts.
func assertScanStateWithParts(t *testing.T, input string, opts *ScanOptions, expected ScanState) {
	t.Helper()
	got := Scan(input, opts)
	assertScanState(t, input, opts, expected)
	// Check Slashes
	if len(got.Slashes) != len(expected.Slashes) {
		t.Errorf("Scan(%q).Slashes = %v, want %v", input, got.Slashes, expected.Slashes)
	} else {
		for i := range expected.Slashes {
			if got.Slashes[i] != expected.Slashes[i] {
				t.Errorf("Scan(%q).Slashes[%d] = %d, want %d", input, i, got.Slashes[i], expected.Slashes[i])
			}
		}
	}
	// Check Parts
	if len(got.Parts) != len(expected.Parts) {
		t.Errorf("Scan(%q).Parts = %v, want %v", input, got.Parts, expected.Parts)
	} else {
		for i := range expected.Parts {
			if got.Parts[i] != expected.Parts[i] {
				t.Errorf("Scan(%q).Parts[%d] = %q, want %q", input, i, got.Parts[i], expected.Parts[i])
			}
		}
	}
}

// assertParts checks that Scan(pattern, {parts:true}).Parts matches expected.
// Ported from: api.scan.js lines 15-19
func assertParts(t *testing.T, pattern string, parts []string) {
	t.Helper()
	info := Scan(pattern, &ScanOptions{Parts: true})
	if len(info.Parts) != len(parts) {
		t.Errorf("Scan(%q, {parts:true}).Parts = %v (len %d), want %v (len %d)",
			pattern, info.Parts, len(info.Parts), parts, len(parts))
		return
	}
	for i := range parts {
		if info.Parts[i] != parts[i] {
			t.Errorf("Scan(%q, {parts:true}).Parts[%d] = %q, want %q",
				pattern, i, info.Parts[i], parts[i])
		}
	}
}

func TestScan(t *testing.T) {
	t.Run(".scan", func(t *testing.T) {
		t.Run("should get the base and glob from a pattern", func(t *testing.T) {
			// api.scan.js line 31
			got := scanBoth("foo/bar")
			if got != [2]string{"foo/bar", ""} {
				t.Errorf("both(\"foo/bar\") = %v, want [\"foo/bar\", \"\"]", got)
			}
			// api.scan.js line 32
			got = scanBoth("foo/@bar")
			if got != [2]string{"foo/@bar", ""} {
				t.Errorf("both(\"foo/@bar\") = %v, want [\"foo/@bar\", \"\"]", got)
			}
			// api.scan.js line 33
			got = scanBoth("foo/@bar\\+")
			if got != [2]string{"foo/@bar\\+", ""} {
				t.Errorf("both(\"foo/@bar\\\\+\") = %v, want [\"foo/@bar\\\\+\", \"\"]", got)
			}
			// api.scan.js line 34
			got = scanBoth("foo/bar+")
			if got != [2]string{"foo/bar+", ""} {
				t.Errorf("both(\"foo/bar+\") = %v, want [\"foo/bar+\", \"\"]", got)
			}
			// api.scan.js line 35
			got = scanBoth("foo/bar*")
			if got != [2]string{"foo", "bar*"} {
				t.Errorf("both(\"foo/bar*\") = %v, want [\"foo\", \"bar*\"]", got)
			}
		})

		t.Run("should handle leading ./", func(t *testing.T) {
			// api.scan.js line 39-52
			assertScanState(t, "./foo/bar/*.js", nil, ScanState{
				Input:          "./foo/bar/*.js",
				Prefix:         "./",
				Start:          2,
				Base:           "foo/bar",
				Glob:           "*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      false,
				Negated:        false,
				NegatedExtglob: false,
			})
		})

		t.Run("should detect braces", func(t *testing.T) {
			// api.scan.js line 56-69
			assertScanState(t, "foo/{a,b,c}/*.js", &ScanOptions{ScanToEnd: true}, ScanState{
				Input:          "foo/{a,b,c}/*.js",
				Prefix:         "",
				Start:          0,
				Base:           "foo",
				Glob:           "{a,b,c}/*.js",
				IsBrace:        true,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      false,
				Negated:        false,
				NegatedExtglob: false,
			})
		})

		t.Run("should detect globstars", func(t *testing.T) {
			// api.scan.js line 73-86
			assertScanState(t, "./foo/**/*.js", &ScanOptions{ScanToEnd: true}, ScanState{
				Input:          "./foo/**/*.js",
				Prefix:         "./",
				Start:          2,
				Base:           "foo",
				Glob:           "**/*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     true,
				IsExtglob:      false,
				Negated:        false,
				NegatedExtglob: false,
			})
		})

		t.Run("should detect extglobs", func(t *testing.T) {
			// api.scan.js line 90-103
			assertScanState(t, "./foo/@(foo)/*.js", nil, ScanState{
				Input:          "./foo/@(foo)/*.js",
				Prefix:         "./",
				Start:          2,
				Base:           "foo",
				Glob:           "@(foo)/*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      true,
				Negated:        false,
				NegatedExtglob: false,
			})
		})

		t.Run("should detect extglobs and globstars", func(t *testing.T) {
			// api.scan.js line 107-122
			assertScanStateWithParts(t, "./foo/@(bar)/**/*.js", &ScanOptions{Parts: true}, ScanState{
				Input:          "./foo/@(bar)/**/*.js",
				Prefix:         "./",
				Start:          2,
				Base:           "foo",
				Glob:           "@(bar)/**/*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     true,
				IsExtglob:      true,
				Negated:        false,
				NegatedExtglob: false,
				Slashes:        []int{1, 5, 12, 15},
				Parts:          []string{"foo", "@(bar)", "**", "*.js"},
			})
		})

		t.Run("should handle leading !", func(t *testing.T) {
			// api.scan.js line 126-139
			assertScanState(t, "!foo/bar/*.js", nil, ScanState{
				Input:          "!foo/bar/*.js",
				Prefix:         "!",
				Start:          1,
				Base:           "foo/bar",
				Glob:           "*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      false,
				Negated:        true,
				NegatedExtglob: false,
			})
		})

		t.Run("should detect negated extglobs at the beginning", func(t *testing.T) {
			// api.scan.js line 143-156
			assertScanState(t, "!(foo)*", nil, ScanState{
				Input:          "!(foo)*",
				Prefix:         "",
				Start:          0,
				Base:           "",
				Glob:           "!(foo)*",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      true,
				Negated:        false,
				NegatedExtglob: true,
			})

			// api.scan.js line 158-171
			assertScanState(t, "!(foo)", nil, ScanState{
				Input:          "!(foo)",
				Prefix:         "",
				Start:          0,
				Base:           "",
				Glob:           "!(foo)",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      true,
				Negated:        false,
				NegatedExtglob: true,
			})
		})

		t.Run("should not detect negated extglobs in the middle", func(t *testing.T) {
			// api.scan.js line 175-188
			assertScanState(t, "test/!(foo)/*", nil, ScanState{
				Input:          "test/!(foo)/*",
				Prefix:         "",
				Start:          0,
				Base:           "test",
				Glob:           "!(foo)/*",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      true,
				Negated:        false,
				NegatedExtglob: false,
			})
		})

		t.Run("should handle leading ./ when negated", func(t *testing.T) {
			// api.scan.js line 192-205
			assertScanState(t, "./!foo/bar/*.js", nil, ScanState{
				Input:          "./!foo/bar/*.js",
				Prefix:         "./!",
				Start:          3,
				Base:           "foo/bar",
				Glob:           "*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      false,
				Negated:        true,
				NegatedExtglob: false,
			})

			// api.scan.js line 207-220
			assertScanState(t, "!./foo/bar/*.js", nil, ScanState{
				Input:          "!./foo/bar/*.js",
				Prefix:         "!./",
				Start:          3,
				Base:           "foo/bar",
				Glob:           "*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      false,
				Negated:        true,
				NegatedExtglob: false,
			})
		})

		t.Run("should recognize leading ./", func(t *testing.T) {
			// api.scan.js line 224
			b := scanBase("./(a|b)")
			if b != "" {
				t.Errorf("base(\"./(a|b)\") = %q, want \"\"", b)
			}
		})

		t.Run("should strip glob magic to return base path", func(t *testing.T) {
			// api.scan.js line 228
			assertEqual(t, scanBase("."), ".")
			// api.scan.js line 229
			assertEqual(t, scanBase(".*"), "")
			// api.scan.js line 230
			assertEqual(t, scanBase("/.*"), "/")
			// api.scan.js line 231
			assertEqual(t, scanBase("/.*/"), "/")
			// api.scan.js line 232
			assertEqual(t, scanBase("a/.*/b"), "a")
			// api.scan.js line 233
			assertEqual(t, scanBase("a*/.*/b"), "")
			// api.scan.js line 234
			assertEqual(t, scanBase("*/a/b/c"), "")
			// api.scan.js line 235
			assertEqual(t, scanBase("*"), "")
			// api.scan.js line 236
			assertEqual(t, scanBase("*/"), "")
			// api.scan.js line 237
			assertEqual(t, scanBase("*/*"), "")
			// api.scan.js line 238
			assertEqual(t, scanBase("*/*/"), "")
			// api.scan.js line 239
			assertEqual(t, scanBase("**"), "")
			// api.scan.js line 240
			assertEqual(t, scanBase("**/"), "")
			// api.scan.js line 241
			assertEqual(t, scanBase("**/*"), "")
			// api.scan.js line 242
			assertEqual(t, scanBase("**/*/"), "")
			// api.scan.js line 243
			assertEqual(t, scanBase("/*.js"), "/")
			// api.scan.js line 244
			assertEqual(t, scanBase("*.js"), "")
			// api.scan.js line 245
			assertEqual(t, scanBase("**/*.js"), "")
			// api.scan.js line 246
			assertEqual(t, scanBase("/root/path/to/*.js"), "/root/path/to")
			// api.scan.js line 247
			assertEqual(t, scanBase("[a-z]"), "")
			// api.scan.js line 248
			assertEqual(t, scanBase("chapter/foo [bar]/"), "chapter")
			// api.scan.js line 249
			assertEqual(t, scanBase("path/!/foo"), "path/!/foo")
			// api.scan.js line 250
			assertEqual(t, scanBase("path/!/foo/"), "path/!/foo/")
			// api.scan.js line 251
			assertEqual(t, scanBase("path/!subdir/foo.js"), "path/!subdir/foo.js")
			// api.scan.js line 252
			assertEqual(t, scanBase("path/**/*"), "path")
			// api.scan.js line 253
			assertEqual(t, scanBase("path/**/subdir/foo.*"), "path")
			// api.scan.js line 254
			assertEqual(t, scanBase("path/*/foo"), "path")
			// api.scan.js line 255
			assertEqual(t, scanBase("path/*/foo/"), "path")
			// api.scan.js line 256 — plus sign must be escaped
			assertEqual(t, scanBase("path/+/foo"), "path/+/foo")
			// api.scan.js line 257 — plus sign must be escaped
			assertEqual(t, scanBase("path/+/foo/"), "path/+/foo/")
			// api.scan.js line 258 — qmarks must be escaped
			assertEqual(t, scanBase("path/?/foo"), "path")
			// api.scan.js line 259 — qmarks must be escaped
			assertEqual(t, scanBase("path/?/foo/"), "path")
			// api.scan.js line 260
			assertEqual(t, scanBase("path/@/foo"), "path/@/foo")
			// api.scan.js line 261
			assertEqual(t, scanBase("path/@/foo/"), "path/@/foo/")
			// api.scan.js line 262
			assertEqual(t, scanBase("path/[a-z]"), "path")
			// api.scan.js line 263
			assertEqual(t, scanBase("path/subdir/**/foo.js"), "path/subdir")
			// api.scan.js line 264
			assertEqual(t, scanBase("path/to/*.js"), "path/to")
		})

		t.Run("should respect escaped characters", func(t *testing.T) {
			// api.scan.js line 268
			assertEqual(t, scanBase("path/\\*\\*/subdir/foo.*"), "path/\\*\\*/subdir")
			// api.scan.js line 269
			assertEqual(t, scanBase("path/\\[\\*\\]/subdir/foo.*"), "path/\\[\\*\\]/subdir")
			// api.scan.js line 270
			assertEqual(t, scanBase("path/\\[foo bar\\]/subdir/foo.*"), "path/\\[foo bar\\]/subdir")
			// api.scan.js line 271
			assertEqual(t, scanBase("path/\\[bar]/"), "path/\\[bar]/")
			// api.scan.js line 272
			assertEqual(t, scanBase("path/\\[bar]"), "path/\\[bar]")
			// api.scan.js line 273
			assertEqual(t, scanBase("[bar]"), "")
			// api.scan.js line 274
			assertEqual(t, scanBase("[bar]/"), "")
			// api.scan.js line 275
			assertEqual(t, scanBase("./\\[bar]"), "\\[bar]")
			// api.scan.js line 276
			assertEqual(t, scanBase("\\[bar]/"), "\\[bar]/")
			// api.scan.js line 277
			assertEqual(t, scanBase("\\[bar\\]/"), "\\[bar\\]/")
			// api.scan.js line 278
			assertEqual(t, scanBase("[bar\\]/"), "[bar\\]/")
			// api.scan.js line 279
			assertEqual(t, scanBase("path/foo \\[bar]/"), "path/foo \\[bar]/")
			// api.scan.js line 280
			assertEqual(t, scanBase("\\[bar]"), "\\[bar]")
			// api.scan.js line 281
			assertEqual(t, scanBase("[bar\\]"), "[bar\\]")
		})

		t.Run("should return full non-glob paths", func(t *testing.T) {
			// api.scan.js line 285
			assertEqual(t, scanBase("path"), "path")
			// api.scan.js line 286
			assertEqual(t, scanBase("path/foo"), "path/foo")
			// api.scan.js line 287
			assertEqual(t, scanBase("path/foo/"), "path/foo/")
			// api.scan.js line 288
			assertEqual(t, scanBase("path/foo/bar.js"), "path/foo/bar.js")
		})

		t.Run("should not return glob when noext is true", func(t *testing.T) {
			// api.scan.js line 292-305
			assertScanState(t, "./foo/bar/*.js", &ScanOptions{Noext: true}, ScanState{
				Input:          "./foo/bar/*.js",
				Prefix:         "./",
				Start:          2,
				Base:           "foo/bar/*.js",
				Glob:           "",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         false,
				IsGlobstar:     false,
				IsExtglob:      false,
				Negated:        false,
				NegatedExtglob: false,
			})
		})

		t.Run("should respect nonegate opts", func(t *testing.T) {
			// api.scan.js line 309-322
			assertScanState(t, "!foo/bar/*.js", &ScanOptions{Nonegate: true}, ScanState{
				Input:          "!foo/bar/*.js",
				Prefix:         "",
				Start:          0,
				Base:           "!foo/bar",
				Glob:           "*.js",
				IsBrace:        false,
				IsBracket:      false,
				IsGlob:         true,
				IsGlobstar:     false,
				IsExtglob:      false,
				Negated:        false,
				NegatedExtglob: false,
			})
		})

		t.Run("should return parts of the pattern", func(t *testing.T) {
			// api.scan.js line 353
			assertParts(t, "./foo", []string{"foo"})
			// api.scan.js line 354
			assertParts(t, "../foo", []string{"..", "foo"})

			// api.scan.js line 356
			assertParts(t, "foo/bar", []string{"foo", "bar"})
			// api.scan.js line 357
			assertParts(t, "foo/*", []string{"foo", "*"})
			// api.scan.js line 358
			assertParts(t, "foo/**", []string{"foo", "**"})
			// api.scan.js line 359
			assertParts(t, "foo/**/*", []string{"foo", "**", "*"})
			// api.scan.js line 360 — Unicode path segments
			assertParts(t, "\u30D5\u30A9\u30EB\u30C0/**/*", []string{"\u30D5\u30A9\u30EB\u30C0", "**", "*"})

			// api.scan.js line 362
			assertParts(t, "foo/!(abc)", []string{"foo", "!(abc)"})
			// api.scan.js line 363
			assertParts(t, "c/!(z)/v", []string{"c", "!(z)", "v"})
			// api.scan.js line 364
			assertParts(t, "c/@(z)/v", []string{"c", "@(z)", "v"})
			// api.scan.js line 365
			assertParts(t, "foo/(bar|baz)", []string{"foo", "(bar|baz)"})
			// api.scan.js line 366
			assertParts(t, "foo/(bar|baz)*", []string{"foo", "(bar|baz)*"})
			// api.scan.js line 367
			assertParts(t, "**/*(W*, *)*", []string{"**", "*(W*, *)*"})
			// api.scan.js line 368
			assertParts(t, "a/**@(/x|/z)/*.md", []string{"a", "**@(/x|/z)", "*.md"})
			// api.scan.js line 369
			assertParts(t, "foo/(bar|baz)/*.js", []string{"foo", "(bar|baz)", "*.js"})

			// api.scan.js line 371
			assertParts(t, "XXX/*/*/12/*/*/m/*/*", []string{"XXX", "*", "*", "12", "*", "*", "m", "*", "*"})
			// api.scan.js line 372
			assertParts(t, "foo/\\\"**\\\"/bar", []string{"foo", "\\\"**\\\"", "bar"})

			// api.scan.js line 374
			assertParts(t, "[0-9]/[0-9]", []string{"[0-9]", "[0-9]"})
			// api.scan.js line 375
			assertParts(t, "foo/[0-9]/[0-9]", []string{"foo", "[0-9]", "[0-9]"})
			// api.scan.js line 376
			assertParts(t, "foo[0-9]/bar[0-9]", []string{"foo[0-9]", "bar[0-9]"})
		})
	})

	t.Run(".base (glob2base test patterns)", func(t *testing.T) {
		t.Run("should get a base name", func(t *testing.T) {
			// api.scan.js line 382
			assertEqual(t, scanBase("js/*.js"), "js")
		})

		t.Run("should get a base name from a nested glob", func(t *testing.T) {
			// api.scan.js line 386
			assertEqual(t, scanBase("js/**/test/*.js"), "js")
		})

		t.Run("should get a base name from a flat file", func(t *testing.T) {
			// api.scan.js line 390
			assertEqual(t, scanBase("js/test/wow.js"), "js/test/wow.js")
		})

		t.Run("should get a base name from character class pattern", func(t *testing.T) {
			// api.scan.js line 394
			assertEqual(t, scanBase("js/t[a-z]st}/*.js"), "js")
		})

		t.Run("should get a base name from extglob", func(t *testing.T) {
			// api.scan.js line 398
			assertEqual(t, scanBase("js/t+(wo|est)/*.js"), "js")
		})

		t.Run("should get a base name from a path with non-extglob parens", func(t *testing.T) {
			// api.scan.js line 402
			assertEqual(t, scanBase("(a|b)"), "")
			// api.scan.js line 403
			assertEqual(t, scanBase("foo/(a|b)"), "foo")
			// api.scan.js line 404
			assertEqual(t, scanBase("/(a|b)"), "/")
			// api.scan.js line 405
			assertEqual(t, scanBase("a/(b c)"), "a")
			// api.scan.js line 406
			assertEqual(t, scanBase("foo/(b c)/baz"), "foo")
			// api.scan.js line 407
			assertEqual(t, scanBase("a/(b c)/"), "a")
			// api.scan.js line 408
			assertEqual(t, scanBase("a/(b c)/d"), "a")
			// api.scan.js line 409
			assertEqual(t, scanBase("a/(b c)", &ScanOptions{Noparen: true}), "a/(b c)")
			// api.scan.js line 410
			assertEqual(t, scanBase("a/(b c)/", &ScanOptions{Noparen: true}), "a/(b c)/")
			// api.scan.js line 411
			assertEqual(t, scanBase("a/(b c)/d", &ScanOptions{Noparen: true}), "a/(b c)/d")
			// api.scan.js line 412
			assertEqual(t, scanBase("foo/(b c)/baz", &ScanOptions{Noparen: true}), "foo/(b c)/baz")
			// api.scan.js line 413
			assertEqual(t, scanBase("path/(foo bar)/subdir/foo.*", &ScanOptions{Noparen: true}), "path/(foo bar)/subdir")
			// api.scan.js line 414 — parens must be escaped
			assertEqual(t, scanBase("a/\\(b c)"), "a/\\(b c)")
			// api.scan.js line 415 — parens must be escaped
			assertEqual(t, scanBase("a/\\+\\(b c)/foo"), "a/\\+\\(b c)/foo")
			// api.scan.js line 416
			assertEqual(t, scanBase("js/t(wo|est)/*.js"), "js")
			// api.scan.js line 417
			assertEqual(t, scanBase("js/t/(wo|est)/*.js"), "js/t")
			// api.scan.js line 418 — parens must be escaped
			assertEqual(t, scanBase("path/(foo bar)/subdir/foo.*"), "path")
			// api.scan.js line 419
			assertEqual(t, scanBase("path/(foo/bar|baz)"), "path")
			// api.scan.js line 420
			assertEqual(t, scanBase("path/(foo/bar|baz)/"), "path")
			// api.scan.js line 421
			assertEqual(t, scanBase("path/(to|from)"), "path")
			// api.scan.js line 422
			assertEqual(t, scanBase("path/\\(foo/bar|baz)/"), "path/\\(foo/bar|baz)/")
			// api.scan.js line 423
			assertEqual(t, scanBase("path/\\*(a|b)"), "path")
			// api.scan.js line 424
			assertEqual(t, scanBase("path/\\*(a|b)/subdir/foo.*"), "path")
			// api.scan.js line 425
			assertEqual(t, scanBase("path/\\*/(a|b)/subdir/foo.*"), "path/\\*")
			// api.scan.js line 426
			assertEqual(t, scanBase("path/\\*\\(a\\|b\\)/subdir/foo.*"), "path/\\*\\(a\\|b\\)/subdir")
		})
	})

	t.Run("technically invalid windows globs", func(t *testing.T) {
		t.Run("should support simple globs with backslash path separator", func(t *testing.T) {
			// api.scan.js line 432
			assertEqual(t, scanBase("C:\\path\\*.js"), "C:\\path\\*.js")
			// api.scan.js line 433
			assertEqual(t, scanBase("C:\\\\path\\\\*.js"), "")
			// api.scan.js line 434
			assertEqual(t, scanBase("C:\\\\path\\*.js"), "C:\\\\path\\*.js")
		})
	})

	t.Run("glob base >", func(t *testing.T) {
		t.Run("should parse globs", func(t *testing.T) {
			// api.scan.js line 440
			assertBoth(t, "!foo", [2]string{"foo", ""})
			// api.scan.js line 441
			assertBoth(t, "*", [2]string{"", "*"})
			// api.scan.js line 442
			assertBoth(t, "**", [2]string{"", "**"})
			// api.scan.js line 443
			assertBoth(t, "**/*.md", [2]string{"", "**/*.md"})
			// api.scan.js line 444
			assertBoth(t, "**/*.min.js", [2]string{"", "**/*.min.js"})
			// api.scan.js line 445
			assertBoth(t, "**/*foo.js", [2]string{"", "**/*foo.js"})
			// api.scan.js line 446
			assertBoth(t, "**/.*", [2]string{"", "**/.*"})
			// api.scan.js line 447
			assertBoth(t, "**/d", [2]string{"", "**/d"})
			// api.scan.js line 448
			assertBoth(t, "*.*", [2]string{"", "*.*"})
			// api.scan.js line 449
			assertBoth(t, "*.js", [2]string{"", "*.js"})
			// api.scan.js line 450
			assertBoth(t, "*.md", [2]string{"", "*.md"})
			// api.scan.js line 451
			assertBoth(t, "*.min.js", [2]string{"", "*.min.js"})
			// api.scan.js line 452
			assertBoth(t, "*/*", [2]string{"", "*/*"})
			// api.scan.js line 453
			assertBoth(t, "*/*/*/*", [2]string{"", "*/*/*/*"})
			// api.scan.js line 454
			assertBoth(t, "*/*/*/e", [2]string{"", "*/*/*/e"})
			// api.scan.js line 455
			assertBoth(t, "*/b/*/e", [2]string{"", "*/b/*/e"})
			// api.scan.js line 456
			assertBoth(t, "*b", [2]string{"", "*b"})
			// api.scan.js line 457
			assertBoth(t, ".*", [2]string{"", ".*"})
			// api.scan.js line 458
			assertBoth(t, "*", [2]string{"", "*"})
			// api.scan.js line 459
			assertBoth(t, "a/**/j/**/z/*.md", [2]string{"a", "**/j/**/z/*.md"})
			// api.scan.js line 460
			assertBoth(t, "a/**/z/*.md", [2]string{"a", "**/z/*.md"})
			// api.scan.js line 461
			assertBoth(t, "node_modules/*-glob/**/*.js", [2]string{"node_modules", "*-glob/**/*.js"})
			// api.scan.js line 462
			assertBoth(t, "{a/b/{c,/foo.js}/e.f.g}", [2]string{"", "{a/b/{c,/foo.js}/e.f.g}"})
			// api.scan.js line 463
			assertBoth(t, ".a*", [2]string{"", ".a*"})
			// api.scan.js line 464
			assertBoth(t, ".b*", [2]string{"", ".b*"})
			// api.scan.js line 465
			assertBoth(t, "/*", [2]string{"/", "*"})
			// api.scan.js line 466
			assertBoth(t, "a/***", [2]string{"a", "***"})
			// api.scan.js line 467
			assertBoth(t, "a/**/b/*.{foo,bar}", [2]string{"a", "**/b/*.{foo,bar}"})
			// api.scan.js line 468
			assertBoth(t, "a/**/c/*", [2]string{"a", "**/c/*"})
			// api.scan.js line 469
			assertBoth(t, "a/**/c/*.md", [2]string{"a", "**/c/*.md"})
			// api.scan.js line 470
			assertBoth(t, "a/**/e", [2]string{"a", "**/e"})
			// api.scan.js line 471
			assertBoth(t, "a/**/j/**/z/*.md", [2]string{"a", "**/j/**/z/*.md"})
			// api.scan.js line 472
			assertBoth(t, "a/**/z/*.md", [2]string{"a", "**/z/*.md"})
			// api.scan.js line 473
			assertBoth(t, "a/**c*", [2]string{"a", "**c*"})
			// api.scan.js line 474
			assertBoth(t, "a/**c/*", [2]string{"a", "**c/*"})
			// api.scan.js line 475
			assertBoth(t, "a/*/*/e", [2]string{"a", "*/*/e"})
			// api.scan.js line 476
			assertBoth(t, "a/*/c/*.md", [2]string{"a", "*/c/*.md"})
			// api.scan.js line 477
			assertBoth(t, "a/b/**/c{d,e}/**/xyz.md", [2]string{"a/b", "**/c{d,e}/**/xyz.md"})
			// api.scan.js line 478
			assertBoth(t, "a/b/**/e", [2]string{"a/b", "**/e"})
			// api.scan.js line 479
			assertBoth(t, "a/b/*.{foo,bar}", [2]string{"a/b", "*.{foo,bar}"})
			// api.scan.js line 480
			assertBoth(t, "a/b/*/e", [2]string{"a/b", "*/e"})
			// api.scan.js line 481
			assertBoth(t, "a/b/.git/", [2]string{"a/b/.git/", ""})
			// api.scan.js line 482
			assertBoth(t, "a/b/.git/**", [2]string{"a/b/.git", "**"})
			// api.scan.js line 483
			assertBoth(t, "a/b/.{foo,bar}", [2]string{"a/b", ".{foo,bar}"})
			// api.scan.js line 484
			assertBoth(t, "a/b/c/*", [2]string{"a/b/c", "*"})
			// api.scan.js line 485
			assertBoth(t, "a/b/c/**/*.min.js", [2]string{"a/b/c", "**/*.min.js"})
			// api.scan.js line 486
			assertBoth(t, "a/b/c/*.md", [2]string{"a/b/c", "*.md"})
			// api.scan.js line 487
			assertBoth(t, "a/b/c/.*.md", [2]string{"a/b/c", ".*.md"})
			// api.scan.js line 488
			assertBoth(t, "a/b/{c,.gitignore,{a,b}}/{a,b}/abc.foo.js", [2]string{"a/b", "{c,.gitignore,{a,b}}/{a,b}/abc.foo.js"})
			// api.scan.js line 489
			assertBoth(t, "a/b/{c,/.gitignore}", [2]string{"a/b", "{c,/.gitignore}"})
			// api.scan.js line 490
			assertBoth(t, "a/b/{c,d}/", [2]string{"a/b", "{c,d}/"})
			// api.scan.js line 491
			assertBoth(t, "a/b/{c,d}/e/f.g", [2]string{"a/b", "{c,d}/e/f.g"})
			// api.scan.js line 492
			assertBoth(t, "b/*/*/*", [2]string{"b", "*/*/*"})
		})

		t.Run("should support file extensions", func(t *testing.T) {
			// api.scan.js line 496
			assertBoth(t, ".md", [2]string{".md", ""})
		})

		t.Run("should support negation pattern", func(t *testing.T) {
			// api.scan.js line 500
			assertBoth(t, "!*.min.js", [2]string{"", "*.min.js"})
			// api.scan.js line 501
			assertBoth(t, "!foo", [2]string{"foo", ""})
			// api.scan.js line 502
			assertBoth(t, "!foo/*.js", [2]string{"foo", "*.js"})
			// api.scan.js line 503
			assertBoth(t, "!foo/(a|b).min.js", [2]string{"foo", "(a|b).min.js"})
			// api.scan.js line 504
			assertBoth(t, "!foo/[a-b].min.js", [2]string{"foo", "[a-b].min.js"})
			// api.scan.js line 505
			assertBoth(t, "!foo/{a,b}.min.js", [2]string{"foo", "{a,b}.min.js"})
			// api.scan.js line 506
			assertBoth(t, "a/b/c/!foo", [2]string{"a/b/c/!foo", ""})
		})

		t.Run("should support extglobs", func(t *testing.T) {
			// api.scan.js line 510
			assertBoth(t, "/a/b/!(a|b)/e.f.g/", [2]string{"/a/b", "!(a|b)/e.f.g/"})
			// api.scan.js line 511
			assertBoth(t, "/a/b/@(a|b)/e.f.g/", [2]string{"/a/b", "@(a|b)/e.f.g/"})
			// api.scan.js line 512
			assertBoth(t, "@(a|b)/e.f.g/", [2]string{"", "@(a|b)/e.f.g/"})
			// api.scan.js line 513
			assertEqual(t, scanBase("path/!(to|from)"), "path")
			// api.scan.js line 514
			assertEqual(t, scanBase("path/*(to|from)"), "path")
			// api.scan.js line 515
			assertEqual(t, scanBase("path/+(to|from)"), "path")
			// api.scan.js line 516
			assertEqual(t, scanBase("path/?(to|from)"), "path")
			// api.scan.js line 517
			assertEqual(t, scanBase("path/@(to|from)"), "path")
		})

		t.Run("should support regex character classes", func(t *testing.T) {
			unescape := &ScanOptions{Unescape: true}
			// api.scan.js line 522
			assertBoth(t, "[a-c]b*", [2]string{"", "[a-c]b*"})
			// api.scan.js line 523
			assertBoth(t, "[a-j]*[^c]", [2]string{"", "[a-j]*[^c]"})
			// api.scan.js line 524
			assertBoth(t, "[a-j]*[^c]b/c", [2]string{"", "[a-j]*[^c]b/c"})
			// api.scan.js line 525
			assertBoth(t, "[a-j]*[^c]bc", [2]string{"", "[a-j]*[^c]bc"})
			// api.scan.js line 526
			assertBoth(t, "[ab][ab]", [2]string{"", "[ab][ab]"})
			// api.scan.js line 527
			assertBoth(t, "foo/[a-b].min.js", [2]string{"foo", "[a-b].min.js"})
			// api.scan.js line 528
			assertEqual(t, scanBase("path/foo[a\\/]/", unescape), "path")
			// api.scan.js line 529
			assertEqual(t, scanBase("path/foo\\[a\\/]/", unescape), "path/foo[a\\/]/")
			// api.scan.js line 530
			assertEqual(t, scanBase("foo[a\\/]", unescape), "")
			// api.scan.js line 531
			assertEqual(t, scanBase("foo\\[a\\/]", unescape), "foo[a\\/]")
		})

		t.Run("should support qmarks", func(t *testing.T) {
			// api.scan.js line 535
			assertBoth(t, "?", [2]string{"", "?"})
			// api.scan.js line 536
			assertBoth(t, "?/?", [2]string{"", "?/?"})
			// api.scan.js line 537
			assertBoth(t, "??", [2]string{"", "??"})
			// api.scan.js line 538
			assertBoth(t, "???", [2]string{"", "???"})
			// api.scan.js line 539
			assertBoth(t, "?a", [2]string{"", "?a"})
			// api.scan.js line 540
			assertBoth(t, "?b", [2]string{"", "?b"})
			// api.scan.js line 541
			assertBoth(t, "a?b", [2]string{"", "a?b"})
			// api.scan.js line 542
			assertBoth(t, "a/?/c.js", [2]string{"a", "?/c.js"})
			// api.scan.js line 543
			assertBoth(t, "a/?/c.md", [2]string{"a", "?/c.md"})
			// api.scan.js line 544
			assertBoth(t, "a/?/c/?/*/f.js", [2]string{"a", "?/c/?/*/f.js"})
			// api.scan.js line 545
			assertBoth(t, "a/?/c/?/*/f.md", [2]string{"a", "?/c/?/*/f.md"})
			// api.scan.js line 546
			assertBoth(t, "a/?/c/?/e.js", [2]string{"a", "?/c/?/e.js"})
			// api.scan.js line 547
			assertBoth(t, "a/?/c/?/e.md", [2]string{"a", "?/c/?/e.md"})
			// api.scan.js line 548
			assertBoth(t, "a/?/c/???/e.js", [2]string{"a", "?/c/???/e.js"})
			// api.scan.js line 549
			assertBoth(t, "a/?/c/???/e.md", [2]string{"a", "?/c/???/e.md"})
			// api.scan.js line 550
			assertBoth(t, "a/??/c.js", [2]string{"a", "??/c.js"})
			// api.scan.js line 551
			assertBoth(t, "a/??/c.md", [2]string{"a", "??/c.md"})
			// api.scan.js line 552
			assertBoth(t, "a/???/c.js", [2]string{"a", "???/c.js"})
			// api.scan.js line 553
			assertBoth(t, "a/???/c.md", [2]string{"a", "???/c.md"})
			// api.scan.js line 554
			assertBoth(t, "a/????/c.js", [2]string{"a", "????/c.js"})
		})

		t.Run("should support non-glob patterns", func(t *testing.T) {
			// api.scan.js line 558
			assertBoth(t, "", [2]string{"", ""})
			// api.scan.js line 559
			assertBoth(t, ".", [2]string{".", ""})
			// api.scan.js line 560
			assertBoth(t, "a", [2]string{"a", ""})
			// api.scan.js line 561
			assertBoth(t, ".a", [2]string{".a", ""})
			// api.scan.js line 562
			assertBoth(t, "/a", [2]string{"/a", ""})
			// api.scan.js line 563
			assertBoth(t, "a/", [2]string{"a/", ""})
			// api.scan.js line 564
			assertBoth(t, "/a/", [2]string{"/a/", ""})
			// api.scan.js line 565
			assertBoth(t, "/a/b/c", [2]string{"/a/b/c", ""})
			// api.scan.js line 566
			assertBoth(t, "/a/b/c/", [2]string{"/a/b/c/", ""})
			// api.scan.js line 567
			assertBoth(t, "a/b/c/", [2]string{"a/b/c/", ""})
			// api.scan.js line 568
			assertBoth(t, "a.min.js", [2]string{"a.min.js", ""})
			// api.scan.js line 569
			assertBoth(t, "a/.x.md", [2]string{"a/.x.md", ""})
			// api.scan.js line 570
			assertBoth(t, "a/b/.gitignore", [2]string{"a/b/.gitignore", ""})
			// api.scan.js line 571
			assertBoth(t, "a/b/c/d.md", [2]string{"a/b/c/d.md", ""})
			// api.scan.js line 572
			assertBoth(t, "a/b/c/d.e.f/g.min.js", [2]string{"a/b/c/d.e.f/g.min.js", ""})
			// api.scan.js line 573
			assertBoth(t, "a/b/.git", [2]string{"a/b/.git", ""})
			// api.scan.js line 574
			assertBoth(t, "a/b/.git/", [2]string{"a/b/.git/", ""})
			// api.scan.js line 575
			assertBoth(t, "a/b/c", [2]string{"a/b/c", ""})
			// api.scan.js line 576
			assertBoth(t, "a/b/c.d/e.md", [2]string{"a/b/c.d/e.md", ""})
			// api.scan.js line 577
			assertBoth(t, "a/b/c.md", [2]string{"a/b/c.md", ""})
			// api.scan.js line 578
			assertBoth(t, "a/b/c.min.js", [2]string{"a/b/c.min.js", ""})
			// api.scan.js line 579
			assertBoth(t, "a/b/git/", [2]string{"a/b/git/", ""})
			// api.scan.js line 580
			assertBoth(t, "aa", [2]string{"aa", ""})
			// api.scan.js line 581
			assertBoth(t, "ab", [2]string{"ab", ""})
			// api.scan.js line 582
			assertBoth(t, "bb", [2]string{"bb", ""})
			// api.scan.js line 583
			assertBoth(t, "c.md", [2]string{"c.md", ""})
			// api.scan.js line 584
			assertBoth(t, "foo", [2]string{"foo", ""})
		})
	})

	t.Run("braces", func(t *testing.T) {
		t.Run("should recognize brace sets", func(t *testing.T) {
			// api.scan.js line 590
			assertEqual(t, scanBase("path/{to,from}"), "path")
			// api.scan.js line 591
			assertEqual(t, scanBase("path/{foo,bar}/"), "path")
			// api.scan.js line 592
			assertEqual(t, scanBase("js/{src,test}/*.js"), "js")
			// api.scan.js line 593
			assertEqual(t, scanBase("{a,b}"), "")
			// api.scan.js line 594
			assertEqual(t, scanBase("/{a,b}"), "/")
			// api.scan.js line 595
			assertEqual(t, scanBase("/{a,b}/"), "/")
		})

		t.Run("should recognize brace ranges", func(t *testing.T) {
			// api.scan.js line 599
			assertEqual(t, scanBase("js/test{0..9}/*.js"), "js")
		})

		t.Run("should respect brace enclosures with embedded separators", func(t *testing.T) {
			unescape := &ScanOptions{Unescape: true}
			// api.scan.js line 604
			assertEqual(t, scanBase("path/{,/,bar/baz,qux}/", unescape), "path")
			// api.scan.js line 605
			assertEqual(t, scanBase("path/\\{,/,bar/baz,qux}/", unescape), "path/{,/,bar/baz,qux}/")
			// api.scan.js line 606
			assertEqual(t, scanBase("path/\\{,/,bar/baz,qux\\}/", unescape), "path/{,/,bar/baz,qux}/")
			// api.scan.js line 607
			assertEqual(t, scanBase("/{,/,bar/baz,qux}/", unescape), "/")
			// api.scan.js line 608
			assertEqual(t, scanBase("/\\{,/,bar/baz,qux}/", unescape), "/{,/,bar/baz,qux}/")
			// api.scan.js line 609
			assertEqual(t, scanBase("{,/,bar/baz,qux}", unescape), "")
			// api.scan.js line 610
			assertEqual(t, scanBase("\\{,/,bar/baz,qux\\}", unescape), "{,/,bar/baz,qux}")
			// api.scan.js line 611
			assertEqual(t, scanBase("\\{,/,bar/baz,qux}/", unescape), "{,/,bar/baz,qux}/")
		})

		t.Run("should handle escaped nested braces", func(t *testing.T) {
			unescape := &ScanOptions{Unescape: true}
			// api.scan.js line 616
			assertEqual(t, scanBase("\\{../,./,\\{bar,/baz},qux}", unescape), "{../,./,{bar,/baz},qux}")
			// api.scan.js line 617
			assertEqual(t, scanBase("\\{../,./,\\{bar,/baz},qux}/", unescape), "{../,./,{bar,/baz},qux}/")
			// api.scan.js line 618
			assertEqual(t, scanBase("path/\\{,/,bar/{baz,qux}}/", unescape), "path/{,/,bar/{baz,qux}}/")
			// api.scan.js line 619
			assertEqual(t, scanBase("path/\\{../,./,\\{bar,/baz},qux}/", unescape), "path/{../,./,{bar,/baz},qux}/")
			// api.scan.js line 620
			assertEqual(t, scanBase("path/\\{../,./,\\{bar,/baz},qux}/", unescape), "path/{../,./,{bar,/baz},qux}/")
			// api.scan.js line 621
			assertEqual(t, scanBase("path/\\{../,./,{bar,/baz},qux}/", unescape), "path/{../,./,{bar,/baz},qux}/")
			// api.scan.js line 622
			assertEqual(t, scanBase("path/{,/,bar/\\{baz,qux}}/", unescape), "path")
		})

		t.Run("should recognize escaped braces", func(t *testing.T) {
			unescape := &ScanOptions{Unescape: true}
			// api.scan.js line 627
			assertEqual(t, scanBase("\\{foo,bar\\}", unescape), "{foo,bar}")
			// api.scan.js line 628
			assertEqual(t, scanBase("\\{foo,bar\\}/", unescape), "{foo,bar}/")
			// api.scan.js line 629
			assertEqual(t, scanBase("\\{foo,bar}/", unescape), "{foo,bar}/")
			// api.scan.js line 630
			assertEqual(t, scanBase("path/\\{foo,bar}/", unescape), "path/{foo,bar}/")
		})

		t.Run("should get a base name from a complex brace glob", func(t *testing.T) {
			// api.scan.js line 634
			assertEqual(t, scanBase("one/{foo,bar}/**/{baz,qux}/*.txt"), "one")
			// api.scan.js line 635
			assertEqual(t, scanBase("two/baz/**/{abc,xyz}/*.js"), "two/baz")
			// api.scan.js line 636
			assertEqual(t, scanBase("foo/{bar,baz}/**/aaa/{bbb,ccc}"), "foo")
		})

		t.Run("should support braces: no path", func(t *testing.T) {
			// api.scan.js line 640
			assertBoth(t, "/a/b/{c,/foo.js}/e.f.g/", [2]string{"/a/b", "{c,/foo.js}/e.f.g/"})
			// api.scan.js line 641
			assertBoth(t, "{a/b/c.js,/a/b/{c,/foo.js}/e.f.g/}", [2]string{"", "{a/b/c.js,/a/b/{c,/foo.js}/e.f.g/}"})
			// api.scan.js line 642
			assertBoth(t, "/a/b/{c,d}/", [2]string{"/a/b", "{c,d}/"})
			// api.scan.js line 643
			assertBoth(t, "/a/b/{c,d}/*.js", [2]string{"/a/b", "{c,d}/*.js"})
			// api.scan.js line 644
			assertBoth(t, "/a/b/{c,d}/*.min.js", [2]string{"/a/b", "{c,d}/*.min.js"})
			// api.scan.js line 645
			assertBoth(t, "/a/b/{c,d}/e.f.g/", [2]string{"/a/b", "{c,d}/e.f.g/"})
			// api.scan.js line 646
			assertBoth(t, "{.,*}", [2]string{"", "{.,*}"})
		})

		t.Run("should support braces in filename", func(t *testing.T) {
			// api.scan.js line 650
			assertBoth(t, "a/b/.{c,.gitignore}", [2]string{"a/b", ".{c,.gitignore}"})
			// api.scan.js line 651
			assertBoth(t, "a/b/.{c,/.gitignore}", [2]string{"a/b", ".{c,/.gitignore}"})
			// api.scan.js line 652
			assertBoth(t, "a/b/.{foo,bar}", [2]string{"a/b", ".{foo,bar}"})
			// api.scan.js line 653
			assertBoth(t, "a/b/{c,.gitignore}", [2]string{"a/b", "{c,.gitignore}"})
			// api.scan.js line 654
			assertBoth(t, "a/b/{c,/.gitignore}", [2]string{"a/b", "{c,/.gitignore}"})
			// api.scan.js line 655
			assertBoth(t, "a/b/{c,/gitignore}", [2]string{"a/b", "{c,/gitignore}"})
			// api.scan.js line 656
			assertBoth(t, "a/b/{c,d}", [2]string{"a/b", "{c,d}"})
		})

		t.Run("should support braces in dirname", func(t *testing.T) {
			// api.scan.js line 660
			assertBoth(t, "a/b/{c,./d}/e/f.g", [2]string{"a/b", "{c,./d}/e/f.g"})
			// api.scan.js line 661
			assertBoth(t, "a/b/{c,./d}/e/f.min.g", [2]string{"a/b", "{c,./d}/e/f.min.g"})
			// api.scan.js line 662
			assertBoth(t, "a/b/{c,.gitignore,{a,./b}}/{a,b}/abc.foo.js", [2]string{"a/b", "{c,.gitignore,{a,./b}}/{a,b}/abc.foo.js"})
			// api.scan.js line 663
			assertBoth(t, "a/b/{c,.gitignore,{a,b}}/{a,b}/*.foo.js", [2]string{"a/b", "{c,.gitignore,{a,b}}/{a,b}/*.foo.js"})
			// api.scan.js line 664
			assertBoth(t, "a/b/{c,.gitignore,{a,b}}/{a,b}/abc.foo.js", [2]string{"a/b", "{c,.gitignore,{a,b}}/{a,b}/abc.foo.js"})
			// api.scan.js line 665
			assertBoth(t, "a/b/{c,/d}/e/f.g", [2]string{"a/b", "{c,/d}/e/f.g"})
			// api.scan.js line 666
			assertBoth(t, "a/b/{c,/d}/e/f.min.g", [2]string{"a/b", "{c,/d}/e/f.min.g"})
			// api.scan.js line 667
			assertBoth(t, "a/b/{c,d}/", [2]string{"a/b", "{c,d}/"})
			// api.scan.js line 668
			assertBoth(t, "a/b/{c,d}/*.js", [2]string{"a/b", "{c,d}/*.js"})
			// api.scan.js line 669
			assertBoth(t, "a/b/{c,d}/*.min.js", [2]string{"a/b", "{c,d}/*.min.js"})
			// api.scan.js line 670
			assertBoth(t, "a/b/{c,d}/e.f.g/", [2]string{"a/b", "{c,d}/e.f.g/"})
			// api.scan.js line 671
			assertBoth(t, "a/b/{c,d}/e/f.g", [2]string{"a/b", "{c,d}/e/f.g"})
			// api.scan.js line 672
			assertBoth(t, "a/b/{c,d}/e/f.min.g", [2]string{"a/b", "{c,d}/e/f.min.g"})
			// api.scan.js line 673
			assertBoth(t, "foo/{a,b}.min.js", [2]string{"foo", "{a,b}.min.js"})
		})
	})
}

// --- Scan test helpers ---

// assertEqual is a simple string equality assertion.
func assertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// assertBoth checks that scanBoth(input) == expected.
func assertBoth(t *testing.T, input string, expected [2]string) {
	t.Helper()
	got := scanBoth(input)
	if got != expected {
		t.Errorf("both(%q) = %v, want %v", input, got, expected)
	}
}
