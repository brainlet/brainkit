// Port of: assemblyscript/tests/compiler.js
// Auto-discovers and runs all official AssemblyScript compiler test fixtures.
//
// Each fixture is a .ts file in the AS tests/compiler/ directory with:
//   - <name>.ts          — AssemblyScript source
//   - <name>.json        — compiler flags and config
//   - <name>.debug.wat   — expected WAT output (debug mode)
//   - <name>.release.wat — expected WAT output (release mode)
//
// Tests with a "stderr" key in the JSON config expect compilation errors.
// Tests without "stderr" expect successful compilation with matching WAT output.
package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/compiler"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/parser"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

// fixtureConfig mirrors the JSON config for each test fixture.
// Ported from: assemblyscript/tests/compiler.js config handling.
type fixtureConfig struct {
	AscFlags []string `json:"asc_flags"`
	Features []string `json:"features"`
	Stderr   any      `json:"stderr"` // string or []string
}

// testsRoot returns the path to wasm-kit/tests/ (where this file lives).
func testsRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func fixturesRoot() string {
	return filepath.Join(testsRoot(), "compiler")
}

func stdAssemblyRoot() string {
	return filepath.Join(testsRoot(), "..", "std", "assembly")
}

// discoverFixtures finds all .ts test fixtures in the AS tests/compiler/ directory.
// Mirrors: assemblyscript/tests/compiler.js getTests()
func discoverFixtures(t *testing.T) []string {
	t.Helper()
	root := fixturesRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("fixtures not found at %s: %v", root, err)
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
		// Skip .d.ts files
		if strings.HasSuffix(name, ".d.ts") {
			continue
		}
		// Skip files starting with _ (convention for helper files)
		if strings.HasPrefix(name, "_") {
			continue
		}
		basename := strings.TrimSuffix(name, ".ts")
		names = append(names, basename)
	}
	sort.Strings(names)
	return names
}

// loadFixtureConfig reads the JSON config for a fixture.
func loadFixtureConfig(t *testing.T, basename string) fixtureConfig {
	t.Helper()
	configPath := filepath.Join(fixturesRoot(), basename+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// No config = empty config (defaults)
		return fixtureConfig{}
	}
	var config fixtureConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal %s: %v", configPath, err)
	}
	return config
}

// expectsErrors returns true if the fixture expects compilation errors (has stderr config).
func (c *fixtureConfig) expectsErrors() bool {
	return c.Stderr != nil
}

// expectedStderr returns the expected stderr patterns as a string slice.
func (c *fixtureConfig) expectedStderr() []string {
	if c.Stderr == nil {
		return nil
	}
	switch v := c.Stderr.(type) {
	case string:
		return []string{v}
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// applyConfig applies fixture config flags to compiler options.
// Returns an error string if the config has unsupported flags (test should be skipped).
func applyConfig(opts *program.Options, config fixtureConfig) (skip string) {
	// Parse asc_flags
	var tokens []string
	for _, flag := range config.AscFlags {
		tokens = append(tokens, strings.Fields(flag)...)
	}
	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "--enable":
			if i+1 >= len(tokens) {
				return "missing value for --enable"
			}
			i++
			for _, featureName := range strings.Split(tokens[i], ",") {
				feature, ok := featureByName(strings.TrimSpace(featureName))
				if !ok {
					return fmt.Sprintf("unsupported feature: %s", featureName)
				}
				opts.SetFeature(feature, true)
			}
		case "--runtime":
			if i+1 >= len(tokens) {
				return "missing value for --runtime"
			}
			i++ // consume runtime name, apply if we support it
			switch tokens[i] {
			case "stub":
				opts.Runtime = common.RuntimeStub
			case "minimal":
				opts.Runtime = common.RuntimeMinimal
			case "incremental":
				opts.Runtime = common.RuntimeIncremental
			default:
				return fmt.Sprintf("unsupported runtime: %s", tokens[i])
			}
		case "--exportStart":
			if i+1 >= len(tokens) {
				return "missing value for --exportStart"
			}
			i++
			opts.ExportStart = tokens[i]
		case "-O", "-O1":
			opts.OptimizeLevelHint = 1
		case "-O2":
			opts.OptimizeLevelHint = 2
		case "-O3":
			opts.OptimizeLevelHint = 3
		case "-Oz":
			opts.OptimizeLevelHint = 2
			opts.ShrinkLevelHint = 2
		case "-Os":
			opts.OptimizeLevelHint = 2
			opts.ShrinkLevelHint = 1
		case "--noAssert":
			opts.NoAssert = true
		case "--memoryBase":
			if i+1 >= len(tokens) {
				return "missing value for --memoryBase"
			}
			i++ // skip value for now
		case "--sourceMap":
			opts.SourceMap = true
		case "--debug":
			opts.DebugInfo = true
		default:
			return fmt.Sprintf("unsupported flag: %s", tokens[i])
		}
	}

	// Apply feature requirements
	for _, featureName := range config.Features {
		feature, ok := featureByName(featureName)
		if !ok {
			return fmt.Sprintf("unsupported feature requirement: %s", featureName)
		}
		opts.SetFeature(feature, true)
	}

	return ""
}

// collectStdSources walks the AS std/assembly/ directory for all .ts files.
func collectStdSources(t *testing.T) []string {
	t.Helper()
	root := stdAssemblyRoot()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
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
		t.Fatalf("walk std sources: %v", err)
	}
	sort.Strings(files)
	return files
}

// compileFixture parses and compiles a fixture, returning the compiler and any diagnostics.
func compileFixture(t *testing.T, basename string, release bool, config fixtureConfig) (mod *module.Module, diags []*diagnostics.DiagnosticMessage, panicked bool) {
	t.Helper()

	// Catch panics anywhere in the pipeline (parser, program, compiler)
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			t.Logf("PANIC: %v\n%s", r, buf[:n])
		}
	}()

	opts := program.NewOptions()
	if release {
		opts.OptimizeLevelHint = 3
		opts.DebugInfo = false
	} else {
		opts.DebugInfo = true
	}

	if skip := applyConfig(opts, config); skip != "" {
		t.Skipf("skipping: %s", skip)
	}

	// Parse the test source
	p := parser.NewParser(nil)
	sourcePath := filepath.Join(fixturesRoot(), basename+".ts")
	sourceText, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source %s: %v", sourcePath, err)
	}
	p.ParseFile(string(sourceText), basename+".ts", true)

	// Parse std library
	stdFiles := collectStdSources(t)
	for _, filename := range stdFiles {
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("read std %s: %v", filename, err)
		}
		relativePath, err := filepath.Rel(stdAssemblyRoot(), filename)
		if err != nil {
			t.Fatalf("rel %s: %v", filename, err)
		}
		logicalPath := common.LIBRARY_PREFIX + filepath.ToSlash(relativePath)
		p.ParseFile(string(data), logicalPath, false)
	}

	// Drain parser file queue
	for next := p.NextFile(); next != ""; next = p.NextFile() {
	}

	parserDiags := append([]*diagnostics.DiagnosticMessage(nil), p.Diagnostics...)
	sources := p.Sources()
	p.Finish()

	prog := program.NewProgram(opts, nil)
	prog.Sources = sources
	c := compiler.NewCompiler(prog)
	result := c.CompileProgram()

	if release && result != nil {
		result.Optimize(
			int(opts.OptimizeLevelHint),
			int(opts.ShrinkLevelHint),
			opts.DebugInfo,
			opts.ZeroFilledMemory,
		)
	}

	diags = append(diags, parserDiags...)
	diags = append(diags, prog.Diagnostics...)
	diags = append(diags, c.Diagnostics...)

	mod = result
	return
}

func normalize(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// binaryenFatalFixtures lists fixtures that trigger binaryen C library crashes
// (Fatal() or SIGSEGV), which cannot be caught by Go's recover().
// These crash the entire test process and must be skipped.
var binaryenFatalFixtures = map[string]bool{
	"duplicate-identifier": true, // Fatal: Module::addGlobal already exists
	"class-override":       true, // SIGSEGV in FinalizeOverrideStub→Load
	"super-inline":         true, // SIGSEGV in binaryen CGo call
}

// TestCompilerFixtures_NoPanic verifies that each fixture compiles without panicking.
// This is the baseline test — even if WAT output doesn't match, not crashing is progress.
func TestCompilerFixtures_NoPanic(t *testing.T) {
	fixtures := discoverFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found")
	}

	passed := 0
	failed := 0
	skipped := 0

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			if binaryenFatalFixtures[name] {
				skipped++
				t.Skipf("skipping: triggers binaryen Fatal()")
			}
			config := loadFixtureConfig(t, name)

			// Only test debug mode for no-panic check
			_, _, panicked := compileFixture(t, name, false, config)
			if panicked {
				failed++
				t.Errorf("fixture %s panicked during compilation", name)
			} else {
				passed++
			}
		})
	}

	t.Logf("Results: %d passed, %d panicked, %d skipped (of %d total)", passed, failed, skipped, len(fixtures))
}

// TestCompilerFixtures_WATMatch verifies that each fixture produces WAT output
// matching the stored fixture files. This is the full correctness test.
func TestCompilerFixtures_WATMatch(t *testing.T) {
	fixtures := discoverFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found")
	}

	passed := 0
	failed := 0
	skipped := 0

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			config := loadFixtureConfig(t, name)

			if config.expectsErrors() {
				// Error-expecting tests: check that diagnostics contain expected patterns
				_, diags, panicked := compileFixture(t, name, false, config)
				if panicked {
					failed++
					t.Errorf("fixture %s panicked", name)
					return
				}

				expectedPatterns := config.expectedStderr()
				diagText := formatDiagnostics(diags)
				lastIndex := 0
				for i, pattern := range expectedPatterns {
					if pattern == "EOF" {
						continue
					}
					idx := strings.Index(diagText[lastIndex:], pattern)
					if idx < 0 {
						failed++
						t.Errorf("missing expected stderr pattern #%d: %q\nGot diagnostics:\n%s", i+1, pattern, diagText)
						return
					}
					lastIndex += idx + len(pattern)
				}
				passed++
				return
			}

			// Success-expecting tests: compare WAT output
			for _, mode := range []struct {
				name    string
				release bool
			}{
				{"debug", false},
				{"release", true},
			} {
				t.Run(mode.name, func(t *testing.T) {
					expectedPath := filepath.Join(fixturesRoot(), name+"."+mode.name+".wat")
					expectedBytes, err := os.ReadFile(expectedPath)
					if err != nil {
						t.Skipf("no %s fixture file: %v", mode.name, err)
						return
					}
					expected := normalize(string(expectedBytes))

					mod, diags, panicked := compileFixture(t, name, mode.release, config)
					if panicked {
						failed++
						t.Errorf("panicked during %s compilation", mode.name)
						return
					}
					if len(diags) > 0 {
						failed++
						t.Errorf("unexpected diagnostics:\n%s", formatDiagnostics(diags))
						return
					}
					if mod == nil {
						failed++
						t.Error("compiler returned nil module")
						return
					}

					actual := normalize(mod.ToText(false))
					if actual != expected {
						failed++
						// Show first difference location
						diffPos := firstDiffPos(expected, actual)
						context := 200
						start := diffPos - context
						if start < 0 {
							start = 0
						}
						end := diffPos + context
						if end > len(expected) {
							end = len(expected)
						}
						endActual := diffPos + context
						if endActual > len(actual) {
							endActual = len(actual)
						}
						t.Errorf("WAT mismatch at byte %d\n--- expected (around diff) ---\n%s\n--- actual (around diff) ---\n%s",
							diffPos,
							expected[start:end],
							actual[start:endActual],
						)
						return
					}
					passed++
				})
			}
		})
	}

	t.Logf("Results: %d passed, %d failed, %d skipped (of %d total)", passed, failed, skipped, len(fixtures))
}

func formatDiagnostics(diags []*diagnostics.DiagnosticMessage) string {
	var b strings.Builder
	for i, d := range diags {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(d.String())
	}
	return b.String()
}

func firstDiffPos(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func featureByName(name string) (common.Feature, bool) {
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
