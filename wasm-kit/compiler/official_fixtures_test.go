package compiler

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/parser"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

type officialCompilerFixtureConfig struct {
	AscFlags []string `json:"asc_flags"`
}

func TestOfficialCompilerFixtures(t *testing.T) {
	for _, name := range []string{"builtins", "simd"} {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Run("debug", func(t *testing.T) {
				runOfficialCompilerFixtureMode(t, name, false)
			})
			t.Run("release", func(t *testing.T) {
				runOfficialCompilerFixtureMode(t, name, true)
			})
		})
	}
}

func runOfficialCompilerFixtureMode(t *testing.T, name string, release bool) {
	t.Helper()

	mode := "debug"
	if release {
		mode = "release"
	}

	expectedPath := filepath.Join(officialCompilerTestsRoot(), name+"."+mode+".wat")
	expected := normalizeOfficialFixtureText(readOfficialFixtureFile(t, expectedPath))
	actual := normalizeOfficialFixtureText(compileOfficialCompilerFixture(t, name, release))
	if actual != expected {
		t.Fatalf("official fixture mismatch for %s.%s.wat", name, mode)
	}
}

func compileOfficialCompilerFixture(t *testing.T, name string, release bool) string {
	t.Helper()

	opts := program.NewOptions()
	if release {
		opts.OptimizeLevelHint = 3
		opts.DebugInfo = false
	} else {
		opts.DebugInfo = true
	}
	applyOfficialFixtureConfig(t, opts, name)

	sources, parserDiagnostics := loadOfficialCompilerFixtureSources(t, name)

	prog := program.NewProgram(opts, nil)
	prog.Sources = sources
	compiler := NewCompiler(prog)
	mod := compiler.CompileProgram()
	if release {
		mod.Optimize(
			int(opts.OptimizeLevelHint),
			int(opts.ShrinkLevelHint),
			opts.DebugInfo,
			opts.ZeroFilledMemory,
		)
	}

	var diags []*diagnostics.DiagnosticMessage
	diags = append(diags, parserDiagnostics...)
	diags = append(diags, prog.Diagnostics...)
	diags = append(diags, compiler.Diagnostics...)
	if len(diags) != 0 {
		t.Fatalf("official fixture %s emitted diagnostics:\n%s", name, formatOfficialDiagnostics(diags))
	}

	if !mod.Validate() {
		t.Fatalf("official fixture %s produced an invalid module", name)
	}

	return mod.ToText(false)
}

func loadOfficialCompilerFixtureSources(t *testing.T, name string) ([]*ast.Source, []*diagnostics.DiagnosticMessage) {
	t.Helper()

	p := parser.NewParser(nil)
	p.ParseFile(
		readOfficialFixtureFile(t, filepath.Join(officialCompilerTestsRoot(), name+".ts")),
		name+".ts",
		true,
	)

	stdSources := collectOfficialStdSources(t)
	for _, filename := range stdSources {
		relativePath, err := filepath.Rel(officialStdAssemblyRoot(), filename)
		if err != nil {
			t.Fatalf("rel %s: %v", filename, err)
		}
		logicalPath := common.LIBRARY_PREFIX + filepath.ToSlash(relativePath)
		p.ParseFile(readOfficialFixtureFile(t, filename), logicalPath, false)
	}

	for next := p.NextFile(); next != ""; next = p.NextFile() {
		// The upstream fixture parser queues imports as it discovers them.
		// Parsing the copied stdlib eagerly above makes these backlog entries redundant.
	}

	diagnosticsCopy := append([]*diagnostics.DiagnosticMessage(nil), p.Diagnostics...)
	sources := append([]*ast.Source(nil), p.Sources()...)
	p.Finish()
	return sources, diagnosticsCopy
}

func applyOfficialFixtureConfig(t *testing.T, opts *program.Options, name string) {
	t.Helper()

	configPath := filepath.Join(officialCompilerTestsRoot(), name+".json")
	raw := readOfficialFixtureFile(t, configPath)

	var config officialCompilerFixtureConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatalf("unmarshal %s: %v", configPath, err)
	}

	var tokens []string
	for _, flag := range config.AscFlags {
		tokens = append(tokens, strings.Fields(flag)...)
	}
	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "--enable":
			if i+1 >= len(tokens) {
				t.Fatalf("missing value for --enable in %s", configPath)
			}
			for _, featureName := range strings.Split(tokens[i+1], ",") {
				feature, ok := officialFeatureByName(strings.TrimSpace(featureName))
				if !ok {
					t.Fatalf("unsupported --enable feature %q in %s", featureName, configPath)
				}
				opts.SetFeature(feature, true)
			}
			i++
		default:
			t.Fatalf("unsupported asc flag %q in %s", tokens[i], configPath)
		}
	}
}

func collectOfficialStdSources(t *testing.T) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(officialStdAssemblyRoot(), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".ts" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk official std sources: %v", err)
	}
	sort.Strings(files)
	return files
}

func officialCompilerTestsRoot() string {
	return filepath.Join("testdata", "assemblyscript", "tests", "compiler")
}

func officialStdAssemblyRoot() string {
	return filepath.Join("testdata", "assemblyscript", "std", "assembly")
}

func readOfficialFixtureFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func normalizeOfficialFixtureText(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

func formatOfficialDiagnostics(diags []*diagnostics.DiagnosticMessage) string {
	var builder strings.Builder
	for i, diag := range diags {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(diag.String())
	}
	return builder.String()
}

func officialFeatureByName(name string) (common.Feature, bool) {
	switch name {
	case "sign-extension":
		return common.FeatureSignExtension, true
	case "mutable-globals":
		return common.FeatureMutableGlobals, true
	case "nontrapping-f2i":
		return common.FeatureNontrappingF2I, true
	case "bulk-memory":
		return common.FeatureBulkMemory, true
	case "simd":
		return common.FeatureSimd, true
	case "threads":
		return common.FeatureThreads, true
	case "exception-handling":
		return common.FeatureExceptionHandling, true
	case "tail-calls":
		return common.FeatureTailCalls, true
	case "reference-types":
		return common.FeatureReferenceTypes, true
	case "multivalue":
		return common.FeatureMultiValue, true
	case "gc":
		return common.FeatureGC, true
	case "memory64":
		return common.FeatureMemory64, true
	case "relaxed-simd":
		return common.FeatureRelaxedSimd, true
	case "extended-const":
		return common.FeatureExtendedConst, true
	case "stringref":
		return common.FeatureStringref, true
	default:
		return 0, false
	}
}
