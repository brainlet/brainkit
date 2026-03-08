package graymatter

import (
	"path/filepath"
	"testing"
)

func TestRead(t *testing.T) {
	fixture := func(name string) string {
		return filepath.Join("testdata", "fixtures", name)
	}

	t.Run("should extract YAML front matter from files with content", func(t *testing.T) {
		file, err := Read(fixture("basic.txt"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		// path property on file object
		if file.Path == "" {
			t.Error("expected path property")
		}
		// data.title property
		data := dataMap(t, file.Data)
		if data["title"] != "Basic" {
			t.Errorf("expected title 'Basic', got %v", data["title"])
		}
		if file.Content != "this is content." {
			t.Errorf("expected content 'this is content.', got %q", file.Content)
		}
	})

	t.Run("should parse complex YAML front matter", func(t *testing.T) {
		file, err := Read(fixture("complex.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		// data property exists
		if file.Data == nil {
			t.Error("expected data property")
		}
		data := dataMap(t, file.Data)
		if data["root"] != "_gh_pages" {
			t.Errorf("expected root '_gh_pages', got %v", data["root"])
		}

		// path property on file
		if file.Path == "" {
			t.Error("expected path property")
		}
		// content property - just check it exists (struct field always exists)
		_ = file.Content
		// orig property - just check it exists (struct field always exists)
		_ = file.Orig
	})

	t.Run("should return an object when a file is empty", func(t *testing.T) {
		file, err := Read(fixture("empty.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		if file.Path == "" {
			t.Error("expected path property")
		}
		// Data exists as a field (may be empty)
		_ = file.Data
		// Content exists as a field (may be empty for empty files)
		_ = file.Content
		// Orig exists as a field
		_ = file.Orig
	})

	t.Run("should return an object when no front matter exists", func(t *testing.T) {
		file, err := Read(fixture("hasnt-matter.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		if file.Path == "" {
			t.Error("expected path property")
		}
		// Data exists as a field
		_ = file.Data
		// Content exists as a field
		_ = file.Content
		// Orig exists as a field
		_ = file.Orig
	})

	t.Run("should parse YAML files directly", func(t *testing.T) {
		file, err := Read(fixture("all.yaml"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		if file.Path == "" {
			t.Error("expected path property")
		}
		// Data exists as a field
		_ = file.Data
		// Content exists as a field (may be empty for files with only front-matter)
		_ = file.Content
		// Orig exists as a field
		_ = file.Orig

		data := dataMap(t, file.Data)
		if data["one"] != "foo" || data["two"] != "bar" || data["three"] != "baz" {
			t.Errorf("expected data {one: foo, two: bar, three: baz}, got %v", file.Data)
		}
	})
}

func TestReadFile(t *testing.T) {
	// Test with absolute path
	t.Run("should read file with absolute path", func(t *testing.T) {
		file, err := Read("/Users/davidroman/Documents/code/brainlet/brainkit/gray-matter/testdata/fixtures/basic.txt")
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}
		data := dataMap(t, file.Data)
		if data["title"] != "Basic" {
			t.Errorf("expected title 'Basic', got %v", data["title"])
		}
	})
}
