// Port of: assemblyscript/tests/parser.js
// Auto-discovers and runs all official AssemblyScript parser test fixtures.
//
// Each fixture is a .ts file in tests/parser/ with:
//   - <name>.ts             — AssemblyScript source to parse
//   - <name>.ts.fixture.ts  — expected serialized AST output
//
// The full fixture comparison requires ASTBuilder (src/extra/ast.ts) which
// is not yet ported. For now, tests verify parsing completes without panicking.
package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/parser"
)

func parserFixturesRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "parser")
}

// discoverParserFixtures finds all .ts test fixtures in tests/parser/.
// Mirrors: assemblyscript/tests/parser.js glob pattern
func discoverParserFixtures(t *testing.T) []string {
	t.Helper()
	root := parserFixturesRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("parser fixtures not found at %s: %v", root, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".ts") {
			continue
		}
		// Skip fixture output files
		if strings.HasSuffix(name, ".fixture.ts") {
			continue
		}
		// Skip files starting with _ (convention for helper files)
		if strings.HasPrefix(name, "_") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestParserFixtures_NoPanic verifies that each parser fixture parses without panicking.
// Ported from: assemblyscript/tests/parser.js
func TestParserFixtures_NoPanic(t *testing.T) {
	fixtures := discoverParserFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no parser fixtures found")
	}

	passed := 0
	failed := 0

	for _, name := range fixtures {
		name := name
		t.Run(strings.TrimSuffix(name, ".ts"), func(t *testing.T) {
			sourcePath := filepath.Join(parserFixturesRoot(), name)
			sourceText, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read source %s: %v", sourcePath, err)
			}

			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
						buf := make([]byte, 4096)
						n := runtime.Stack(buf, false)
						t.Logf("PANIC: %v\n%s", r, buf[:n])
					}
				}()

				p := parser.NewParser(nil)
				text := strings.ReplaceAll(string(sourceText), "\r\n", "\n")
				p.ParseFile(text, name, true)
			}()

			if panicked {
				failed++
				t.Errorf("fixture %s panicked during parsing", name)
			} else {
				passed++
			}
		})
	}

	t.Logf("Results: %d passed, %d panicked (of %d total)", passed, failed, len(fixtures))
}

// TODO: TestParserFixtures_FixtureMatch — requires ASTBuilder (port of src/extra/ast.ts)
// to serialize AST back to source text and compare against .fixture.ts files.
