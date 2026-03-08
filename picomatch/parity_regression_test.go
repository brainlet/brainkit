package picomatch

import "testing"

func TestParityRegressions(t *testing.T) {
	t.Run("matchBase keeps upstream basename semantics on backslash paths", func(t *testing.T) {
		assertMatch(t, false, `C:\path\x`, "[[:alpha:]]", &Options{MatchBase: true})
		assertMatch(t, false, `C:\path\x`, "[[:alpha:]]", &Options{Basename: true})
	})

	t.Run("capture mode does not reuse literal equality", func(t *testing.T) {
		assertMatch(t, false, "a(b)", "a(b)", &Options{Capture: true})
	})

	t.Run("literalBrackets keeps regex-like POSIX classes active", func(t *testing.T) {
		assertMatch(t, true, "a", "[[:alpha:]]", &Options{LiteralBrackets: boolPtr(true)})
	})

	t.Run("posix punct class matches upstream runtime behavior", func(t *testing.T) {
		opts := &Options{Posix: true, Regex: boolPtr(true), StrictSlashes: true}
		assertMatch(t, false, `foo\bar.js`, "foo[[:punct:]]*", opts)
	})

	t.Run("makeRe windows source stays source-faithful", func(t *testing.T) {
		re := MakeRe("*", &Options{Windows: true})
		expected := `^(?:(?!\.)(?=.)[^\\/]*?[\\/]?)$`
		if got := re.re.String(); got != expected {
			t.Fatalf("MakeRe(%q, {Windows:true}).source: expected %q, got %q", "*", expected, got)
		}
	})

	t.Run("parse output preserves upstream POSIX class source", func(t *testing.T) {
		opts := &Options{Posix: true, Regex: boolPtr(true), StrictSlashes: true}
		parsed := Parse("[abc[:punct:][0-9]", opts)
		expected := "(?=.)[abc\\-!\"#$%&'()\\*+,./:;<=>?@[\\]^_`{|}~\\[0-9]"
		if parsed.Output != expected {
			t.Fatalf("Parse(%q).Output: expected %q, got %q", "[abc[:punct:][0-9]", expected, parsed.Output)
		}
	})

	t.Run("compile output helper mirrors upstream compileRe returnOutput mode", func(t *testing.T) {
		parsed := Parse("a*.txt", nil)
		compileExpected := `a[^/]*?\.txt`
		if got := CompileReOutput(parsed, nil); got != compileExpected {
			t.Fatalf("CompileReOutput(Parse(%q)): expected %q, got %q", "a*.txt", compileExpected, got)
		}
		makeReExpected := `^(?:a[^/]*?\.txt)$`
		if got := MakeReOutput("a*.txt", nil); got != makeReExpected {
			t.Fatalf("MakeReOutput(%q): expected %q, got %q", "a*.txt", makeReExpected, got)
		}
	})

	t.Run("compiled matcher supports parsed-state input and result objects", func(t *testing.T) {
		parsed := Parse("a*.txt", nil)
		matcher := CompileWithResult(parsed, nil)
		result := matcher("ab.txt", true)
		if result == nil {
			t.Fatal("expected MatchResult, got nil")
		}
		if !result.IsMatch {
			t.Fatalf("expected parsed-state matcher to match %q", "ab.txt")
		}
		if result.State != parsed {
			t.Fatal("expected MatchResult.State to reuse the parsed state input")
		}
		if result.Glob != "a*.txt" {
			t.Fatalf("expected MatchResult.Glob to be %q, got %q", "a*.txt", result.Glob)
		}
	})
}
