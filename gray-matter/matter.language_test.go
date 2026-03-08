package graymatter

import (
	"testing"
)

func TestLanguage(t *testing.T) {
	t.Run("should detect the name of the language to parse", func(t *testing.T) {
		result := Language("---\nfoo: bar\n---")
		expected := LanguageResult{Raw: "", Name: ""}
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}

		result = Language("---js\nfoo: bar\n---")
		expected = LanguageResult{Raw: "js", Name: "js"}
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}

		result = Language("---coffee\nfoo: bar\n---")
		expected = LanguageResult{Raw: "coffee", Name: "coffee"}
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("should work around whitespace", func(t *testing.T) {
		result := Language("--- \nfoo: bar\n---")
		expected := LanguageResult{Raw: " ", Name: ""}
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}

		result = Language("--- js \nfoo: bar\n---")
		expected = LanguageResult{Raw: " js ", Name: "js"}
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}

		result = Language("---  coffee \nfoo: bar\n---")
		expected = LanguageResult{Raw: "  coffee ", Name: "coffee"}
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})
}
