package graymatter

import "testing"

func TestBehaviorParity(t *testing.T) {
	t.Run("detected language should override options language", func(t *testing.T) {
		file, err := Parse("---json\n{\"a\":1}\n---", Options{Language: "yaml"})
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if file.Language != "json" {
			t.Fatalf("expected detected language to win, got %q", file.Language)
		}

		data := dataMap(t, file.Data)
		if data["a"] != float64(1) && data["a"] != 1 {
			t.Fatalf("expected parsed json data, got %v", data["a"])
		}
	})

	t.Run("language should work without a leading fence", func(t *testing.T) {
		result := Language("foo\nbar")
		if result.Raw != "foo" || result.Name != "foo" {
			t.Fatalf("unexpected language result: %+v", result)
		}
	})

	t.Run("custom excerpt callback should mutate the file", func(t *testing.T) {
		file, err := Parse("---\na: 1\n---\nhello", Options{
			Excerpt: func(file *File) {
				file.Excerpt = "custom"
			},
		})
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}
		if file.Excerpt != "custom" {
			t.Fatalf("expected custom excerpt, got %q", file.Excerpt)
		}
	})

	t.Run("empty input should keep orig as the empty string", func(t *testing.T) {
		file, err := Parse("")
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		orig, ok := file.Orig.(string)
		if !ok {
			t.Fatalf("expected orig string, got %T", file.Orig)
		}
		if orig != "" {
			t.Fatalf("expected empty orig, got %q", orig)
		}
	})
}
