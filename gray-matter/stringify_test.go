package graymatter

import (
	"testing"
)

func TestStringify(t *testing.T) {
	t.Run("should stringify front-matter from a file object", func(t *testing.T) {
		file := File{
			Content: "Name: {{name}}\n",
			Data:    map[string]any{"name": "gray-matter"},
		}

		result, _ := StringifyFile(file, nil)
		expected := "---\nname: gray-matter\n---\nName: {{name}}\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("should stringify from a string", func(t *testing.T) {
		result, _ := Stringify("Name: {{name}}\n", nil)
		expected := "Name: {{name}}\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("should use custom delimiters to stringify", func(t *testing.T) {
		result, _ := Stringify("Name: {{name}}", map[string]any{"name": "gray-matter"}, Options{Delimiters: "~~~"})
		expected := "~~~\nname: gray-matter\n~~~\nName: {{name}}\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("should stringify a file object", func(t *testing.T) {
		file := File{
			Content: "Name: {{name}}",
			Data:    map[string]any{"name": "gray-matter"},
		}
		result, _ := StringifyFile(file, nil)
		expected := "---\nname: gray-matter\n---\nName: {{name}}\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("should stringify an excerpt", func(t *testing.T) {
		file := File{
			Content: "Name: {{name}}",
			Data:    map[string]any{"name": "gray-matter"},
			Excerpt: "This is an excerpt.",
		}

		result, _ := StringifyFile(file, nil)
		expected := "---\nname: gray-matter\n---\nThis is an excerpt.\n---\nName: {{name}}\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("should not add an excerpt if it already exists", func(t *testing.T) {
		file := File{
			Content: "Name: {{name}}\n\nThis is an excerpt.",
			Data:    map[string]any{"name": "gray-matter"},
			Excerpt: "This is an excerpt.",
		}

		result, _ := StringifyFile(file, nil)
		expected := "---\nname: gray-matter\n---\nName: {{name}}\n\nThis is an excerpt.\n"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}
