package graymatter

import (
	"path/filepath"
	"testing"
)

func TestParseYAML(t *testing.T) {
	t.Run("should parse YAML", func(t *testing.T) {
		file, err := Read(filepath.Join("testdata", "fixtures", "all.yaml"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}
		if file.Data["one"] != "foo" || file.Data["two"] != "bar" || file.Data["three"] != "baz" {
			t.Errorf("expected {one: foo, two: bar, three: baz}, got %v", file.Data)
		}
	})

	t.Run("should parse YAML with closing ...", func(t *testing.T) {
		file, err := Read(filepath.Join("testdata", "fixtures", "all-dots.yaml"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}
		if file.Data["one"] != "foo" || file.Data["two"] != "bar" || file.Data["three"] != "baz" {
			t.Errorf("expected {one: foo, two: bar, three: baz}, got %v", file.Data)
		}
	})

	t.Run("should parse YAML front matter", func(t *testing.T) {
		actual, err := Read(filepath.Join("testdata", "fixtures", "lang-yaml.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}
		if actual.Data["title"] != "YAML" {
			t.Errorf("expected title 'YAML', got %v", actual.Data["title"])
		}
		// Check file object has data, content, orig properties
		if actual.Data == nil {
			t.Error("expected data property on file")
		}
		_ = actual.Content // exists as struct field
		_ = actual.Orig    // exists as struct field
	})

	t.Run("should detect YAML as the language with no language defined after the first fence", func(t *testing.T) {
		actual, err := Read(filepath.Join("testdata", "fixtures", "autodetect-no-lang.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}
		if actual.Data["title"] != "autodetect-no-lang" {
			t.Errorf("expected title 'autodetect-no-lang', got %v", actual.Data["title"])
		}
		// Check file object has data, content, orig properties
		if actual.Data == nil {
			t.Error("expected data property on file")
		}
		_ = actual.Content // exists as struct field
		_ = actual.Orig    // exists as struct field
	})

	t.Run("should detect YAML as the language", func(t *testing.T) {
		actual, err := Read(filepath.Join("testdata", "fixtures", "autodetect-yaml.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}
		if actual.Data["title"] != "autodetect-yaml" {
			t.Errorf("expected title 'autodetect-yaml', got %v", actual.Data["title"])
		}
		// Check file object has data, content, orig properties
		if actual.Data == nil {
			t.Error("expected data property on file")
		}
		_ = actual.Content // exists as struct field
		_ = actual.Orig    // exists as struct field
	})
}
