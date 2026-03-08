package graymatter

import (
	"testing"
)

func TestEmpty(t *testing.T) {
	t.Run("should work with empty front-matter", func(t *testing.T) {
		file1, _ := Parse("---\n---\nThis is content")
		if file1.Content != "This is content" {
			t.Errorf("expected content 'This is content', got %q", file1.Content)
		}
		if len(file1.Data) != 0 {
			t.Errorf("expected empty data, got %v", file1.Data)
		}

		file2, _ := Parse("---\n\n---\nThis is content")
		if file2.Content != "This is content" {
			t.Errorf("expected content 'This is content', got %q", file2.Content)
		}
		if len(file2.Data) != 0 {
			t.Errorf("expected empty data, got %v", file2.Data)
		}

		file3, _ := Parse("---\n\n\n\n\n\n---\nThis is content")
		if file3.Content != "This is content" {
			t.Errorf("expected content 'This is content', got %q", file3.Content)
		}
		if len(file3.Data) != 0 {
			t.Errorf("expected empty data, got %v", file3.Data)
		}
	})

	t.Run("should add content with empty front matter to file.empty", func(t *testing.T) {
		file, _ := Parse("---\n---")
		if file.Empty != "---\n---" {
			t.Errorf("expected empty to be '---\\n---', got %q", file.Empty)
		}
	})

	t.Run("should update file.isEmpty to true", func(t *testing.T) {
		file, _ := Parse("---\n---")
		if !file.IsEmpty {
			t.Errorf("expected isEmpty to be true")
		}
	})

	t.Run("should work when front-matter has comments", func(t *testing.T) {
		fixture := "---\n# this is a comment\n# another one\n---"
		file, _ := Parse(fixture)
		if file.Empty != fixture {
			t.Errorf("expected empty to be %q, got %q", fixture, file.Empty)
		}
	})
}
