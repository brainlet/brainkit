package graymatter

import (
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCustomParser(t *testing.T) {
	called := false

	file, err := Read(filepath.Join("testdata", "fixtures", "lang-yaml.md"), Options{
		Parser: func(str string, opts Options) (any, error) {
			called = true
			var out map[string]any
			if err := yaml.Unmarshal([]byte(str), &out); err != nil {
				return nil, err
			}
			return out, nil
		},
	})
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if !called {
		t.Fatal("expected custom parser to be called")
	}

	data := dataMap(t, file.Data)
	if data["title"] != "YAML" {
		t.Fatalf("expected title YAML, got %v", data["title"])
	}
}

func TestCustomEngineAliases(t *testing.T) {
	engine := EngineFunc(func(input string) (any, error) {
		return map[string]any{
			"title": "aliased-engine",
			"raw":   input,
		}, nil
	})

	t.Run("should resolve cson to coffee engine", func(t *testing.T) {
		file, err := Read(filepath.Join("testdata", "fixtures", "autodetect-cson.md"), Options{
			Engines: map[string]Engine{
				"coffee": engine,
			},
		})
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}

		data := dataMap(t, file.Data)
		if data["title"] != "aliased-engine" {
			t.Fatalf("expected aliased engine result, got %v", data["title"])
		}
	})

	t.Run("should resolve yml to yaml engine", func(t *testing.T) {
		file, err := Parse("---yml\nname: test\n---")
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		data := dataMap(t, file.Data)
		if data["name"] != "test" {
			t.Fatalf("expected yaml alias to parse, got %v", data["name"])
		}
	})
}
