package graymatter

import (
	"testing"
)

func TestParseJSON(t *testing.T) {
	t.Run("should parse JSON front matter", func(t *testing.T) {
		actual, err := Read(fixturePath("lang-json.md"), Options{Language: "json"})
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		if actual.Data["title"] != "JSON" {
			t.Errorf("expected title 'JSON', got %v", actual.Data["title"])
		}
	})

	t.Run("should auto-detect JSON as the language", func(t *testing.T) {
		actual, err := Read(fixturePath("autodetect-json.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		if actual.Data["title"] != "autodetect-JSON" {
			t.Errorf("expected title 'autodetect-JSON', got %v", actual.Data["title"])
		}
	})
}
