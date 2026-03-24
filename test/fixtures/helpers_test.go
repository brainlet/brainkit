package fixtures_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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

// importLineRe matches ES import lines:
//   import { ... } from "kit";
//   import type { ... } from "kit";
//   import { ... } from "ai";
// These are stripped because kit.Deploy runs code in a SES Compartment
// where all symbols are injected as endowments (globals), not ES modules.
var importLineRe = regexp.MustCompile(`(?m)^import\s+(type\s+)?(\{[^}]*\}|[^\s]+)\s+from\s+"[^"]+";\s*\n?`)

// stripImports removes ES import lines from transpiled JS.
// The SES Compartment injects all module symbols as global endowments,
// so import statements would cause a SyntaxError.
func stripImports(js string) string {
	return importLineRe.ReplaceAllString(js, "")
}

// loadTSFixture reads, transpiles, and strips imports from a TS fixture.
// Returns JS ready to pass to kit.Deploy.
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
	return stripImports(js)
}

// loadASFixture reads an AS fixture source (no transpilation — AS compiler handles it).
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
