package graymatter

import (
	"testing"
)

func TestExcerpt(t *testing.T) {
	t.Run("should get an excerpt after front matter", func(t *testing.T) {
		file, _ := Parse("---\nabc: xyz\n---\nfoo\nbar\nbaz\n---\ncontent", Options{Excerpt: true})

		if file.Matter != "\nabc: xyz" {
			t.Errorf("expected matter to be '\\nabc: xyz', got %q", file.Matter)
		}
		if file.Content != "foo\nbar\nbaz\n---\ncontent" {
			t.Errorf("expected content to be 'foo\\nbar\\nbaz\\n---\\ncontent', got %q", file.Content)
		}
		if file.Excerpt != "foo\nbar\nbaz\n" {
			t.Errorf("expected excerpt to be 'foo\\nbar\\nbaz\\n', got %q", file.Excerpt)
		}
		if file.Data["abc"] != "xyz" {
			t.Errorf("expected data.abc to be 'xyz', got %v", file.Data["abc"])
		}
	})

	t.Run("should not get excerpt when disabled", func(t *testing.T) {
		file, _ := Parse("---\nabc: xyz\n---\nfoo\nbar\nbaz\n---\ncontent", Options{})

		if file.Matter != "\nabc: xyz" {
			t.Errorf("expected matter to be '\\nabc: xyz', got %q", file.Matter)
		}
		if file.Content != "foo\nbar\nbaz\n---\ncontent" {
			t.Errorf("expected content to be 'foo\\nbar\\nbaz\\n---\\ncontent', got %q", file.Content)
		}
		if file.Excerpt != "" {
			t.Errorf("expected excerpt to be empty, got %q", file.Excerpt)
		}
		if file.Data["abc"] != "xyz" {
			t.Errorf("expected data.abc to be 'xyz', got %v", file.Data["abc"])
		}
	})

	t.Run("should use a custom separator", func(t *testing.T) {
		file, _ := Parse("---\nabc: xyz\n---\nfoo\nbar\nbaz\n<!-- endexcerpt -->\ncontent", Options{
			ExcerptSeparator: "<!-- endexcerpt -->",
		})

		if file.Matter != "\nabc: xyz" {
			t.Errorf("expected matter to be '\\nabc: xyz', got %q", file.Matter)
		}
		if file.Content != "foo\nbar\nbaz\n<!-- endexcerpt -->\ncontent" {
			t.Errorf("expected content to be 'foo\\nbar\\nbaz\\n<!-- endexcerpt -->\\ncontent', got %q", file.Content)
		}
		if file.Excerpt != "foo\nbar\nbaz\n" {
			t.Errorf("expected excerpt to be 'foo\\nbar\\nbaz\\n', got %q", file.Excerpt)
		}
		if file.Data["abc"] != "xyz" {
			t.Errorf("expected data.abc to be 'xyz', got %v", file.Data["abc"])
		}
	})

	t.Run("should use a custom separator when no front-matter exists", func(t *testing.T) {
		file, _ := Parse("foo\nbar\nbaz\n<!-- endexcerpt -->\ncontent", Options{
			ExcerptSeparator: "<!-- endexcerpt -->",
		})

		if file.Matter != "" {
			t.Errorf("expected matter to be empty, got %q", file.Matter)
		}
		if file.Content != "foo\nbar\nbaz\n<!-- endexcerpt -->\ncontent" {
			t.Errorf("expected content to be 'foo\\nbar\\nbaz\\n<!-- endexcerpt -->\\ncontent', got %q", file.Content)
		}
		if file.Excerpt != "foo\nbar\nbaz\n" {
			t.Errorf("expected excerpt to be 'foo\\nbar\\nbaz\\n', got %q", file.Excerpt)
		}
		if len(file.Data) != 0 {
			t.Errorf("expected empty data, got %v", file.Data)
		}
	})
}
