package fixtures_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	typescript "github.com/brainlet/brainkit/vendor_typescript"
)

// fixturesRoot returns the absolute path to the fixtures/ directory.
func fixturesRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// test/fixtures/ → brainkit root → fixtures/
	return filepath.Join(wd, "..", "..", "fixtures")
}

// loadTSFixture reads and transpiles a TS fixture to JS.
func loadTSFixture(t *testing.T, name string) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(fixturesRoot(t), "ts", name, "index.ts"))
	if err != nil {
		t.Fatalf("read fixture ts/%s: %v", name, err)
	}
	js, err := typescript.Transpile(string(source), typescript.TranspileOptions{
		FileName: name + ".ts",
	})
	if err != nil {
		t.Fatalf("transpile ts/%s: %v", name, err)
	}
	return js
}

// loadASFixture reads an AS fixture source.
func loadASFixture(t *testing.T, name string) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(fixturesRoot(t), "as", name, "index.ts"))
	if err != nil {
		t.Fatalf("read fixture as/%s: %v", name, err)
	}
	return string(source)
}

// loadExpect reads the expect.json sidecar for a fixture.
// Returns nil if no expect.json exists.
func loadExpect(t *testing.T, category, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixturesRoot(t), category, name, "expect.json"))
	if err != nil {
		return nil
	}
	var expect map[string]any
	if err := json.Unmarshal(data, &expect); err != nil {
		t.Fatalf("parse expect.json for %s/%s: %v", category, name, err)
	}
	return expect
}
