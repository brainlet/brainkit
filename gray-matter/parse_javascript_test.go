package graymatter

import (
	"path/filepath"
	"testing"
)

func TestParseJavaScript(t *testing.T) {
	fixture := func(name string) string {
		return filepath.Join("testdata", "fixtures", name)
	}

	t.Run("should parse front matter when options.lang is javascript", func(t *testing.T) {
		file, err := Read(fixture("lang-javascript-object-fn.md"), Options{Lang: "javascript"})
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		data := dataMap(t, file.Data)
		if data["title"] != "javascript front matter" {
			t.Fatalf("expected title, got %v", data["title"])
		}

		fnGroup := dataMap(t, data["fn"])
		reverse := jsFunction(t, fnGroup["reverse"])
		result, err := reverse.Call("brainlet")
		if err != nil {
			t.Fatalf("reverse.Call returned error: %v", err)
		}
		if result != "telniarb" {
			t.Fatalf("expected reversed string, got %v", result)
		}
	})

	t.Run("should parse front matter when options.language is js", func(t *testing.T) {
		file, err := Read(fixture("lang-javascript-object-fn.md"), Options{Language: "js"})
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		data := dataMap(t, file.Data)
		if data["title"] != "javascript front matter" {
			t.Fatalf("expected title, got %v", data["title"])
		}
	})

	t.Run("should eval functions", func(t *testing.T) {
		file, err := Read(fixture("lang-javascript-fn.md"), Options{Language: "js"})
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		reverse := jsFunction(t, file.Data)
		result, err := reverse.Call("abc")
		if err != nil {
			t.Fatalf("reverse.Call returned error: %v", err)
		}
		if result != "cba" {
			t.Fatalf("expected cba, got %v", result)
		}
	})

	t.Run("should detect javascript after the first delimiter", func(t *testing.T) {
		file, err := Read(fixture("autodetect-javascript.md"))
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		data := dataMap(t, file.Data)
		if data["title"] != "autodetect-javascript" {
			t.Fatalf("expected title, got %v", data["title"])
		}
	})
}
