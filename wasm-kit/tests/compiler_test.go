// Port of: assemblyscript/tests/compiler.js
// Auto-discovers and runs all official AssemblyScript compiler test fixtures.
//
// Test levels:
//   - TestCompiler_NoPanic          — Level 0: compile without crashing (172 fixtures)
//   - TestCompiler_ErrorDiagnostics — Level 1: error fixtures produce expected stderr (42 fixtures)
//   - TestCompiler_WATMatch         — Level 2: WAT output matches reference files (130 × 2 modes)
//   - TestCompiler_Runtime          — Level 3+4: compiled Wasm runs in wazero without abort (130 fixtures)
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

// ---------------------------------------------------------------------------
// Fixture config (mirrors the JSON config for each test fixture)
// ---------------------------------------------------------------------------

type fixtureConfig struct {
	AscFlags        []string `json:"asc_flags"`
	Features        []string `json:"features"`
	Stderr          any      `json:"stderr"`          // string or []string
	AscRtrace       bool     `json:"asc_rtrace"`      // enable rtrace
	SkipInstantiate bool     `json:"skipInstantiate"` // skip runtime instantiation
}

func (c *fixtureConfig) expectsErrors() bool {
	return c.Stderr != nil
}

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

// ---------------------------------------------------------------------------
// Fixture discovery and config loading
// ---------------------------------------------------------------------------

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
		if !strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".d.ts") || strings.HasPrefix(name, "_") {
			continue
		}
		names = append(names, strings.TrimSuffix(name, ".ts"))
	}
	sort.Strings(names)
	return names
}

func loadFixtureConfig(t *testing.T, basename string) fixtureConfig {
	t.Helper()
	configPath := filepath.Join(fixturesRoot(), basename+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fixtureConfig{} // No config = defaults
	}
	var config fixtureConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal %s: %v", configPath, err)
	}
	return config
}

func applyConfig(opts *program.Options, config fixtureConfig) (skip string) {
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
			i++
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
	for _, featureName := range config.Features {
		feature, ok := featureByName(featureName)
		if !ok {
			return fmt.Sprintf("unsupported feature requirement: %s", featureName)
		}
		opts.SetFeature(feature, true)
	}
	return ""
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

// ---------------------------------------------------------------------------
// Skip lists
// ---------------------------------------------------------------------------

// binaryenFatalFixtures trigger binaryen C library crashes (Fatal() or SIGSEGV)
// which cannot be caught by Go's recover(). These crash the entire test process.
var binaryenFatalFixtures = map[string]bool{
	"class-override":       true, // SIGSEGV in FinalizeOverrideStub
	"duplicate-identifier": true, // Fatal: Module::addGlobal already exists
	"exports-lazy":         true, // Assertion: local.set index too large (binaryen ToBinary)
	"super-inline":         true, // SIGSEGV in binaryen CGo call
}

// binaryenReleaseFatalFixtures crash binaryen in release/optimized mode.
var binaryenReleaseFatalFixtures = map[string]bool{
	"class-overloading": true, // SIGABRT in optimizer
	"class":             true, // SIGABRT in optimizer
}

// glueFixtures need custom host imports (ported in glue_test.go).
// Runtime tests skip these unless glue is registered.
var glueFixtures = map[string]bool{
	"bigint-integration": true,
	"declare":            true,
	"exportimport-table": true,
	"external":           true,
	"mutable-globals":    true,
}

// ---------------------------------------------------------------------------
// Level 0: TestCompiler_NoPanic
// ---------------------------------------------------------------------------

// TestCompiler_NoPanic verifies that each fixture compiles without panicking.
// This is the broadest, cheapest test — 172 fixtures, debug mode only.
func TestCompiler_NoPanic(t *testing.T) {
	fixtures := discoverFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found")
	}

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			if binaryenFatalFixtures[name] {
				t.Skipf("triggers binaryen Fatal()")
			}
			config := loadFixtureConfig(t, name)
			result := compileFixture(t, name, false, config)
			if result.Panicked {
				t.Errorf("panicked: %v", result.PanicValue)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Level 1: TestCompiler_ErrorDiagnostics
// ---------------------------------------------------------------------------

// TestCompiler_ErrorDiagnostics verifies that error-expecting fixtures produce
// the expected diagnostic patterns in the correct order.
func TestCompiler_ErrorDiagnostics(t *testing.T) {
	fixtures := discoverFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found")
	}

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			config := loadFixtureConfig(t, name)
			if !config.expectsErrors() {
				t.Skip("not an error fixture")
			}
			if binaryenFatalFixtures[name] {
				t.Skipf("triggers binaryen Fatal()")
			}

			result := compileFixture(t, name, false, config)
			if result.Panicked {
				t.Fatalf("panicked: %v", result.PanicValue)
			}

			expectedPatterns := config.expectedStderr()
			diagText := formatDiagnostics(result.Diagnostics)
			lastIndex := 0
			for i, pattern := range expectedPatterns {
				if pattern == "EOF" {
					continue
				}
				idx := strings.Index(diagText[lastIndex:], pattern)
				if idx < 0 {
					t.Errorf("missing expected stderr pattern #%d: %q\nGot diagnostics:\n%s", i+1, pattern, diagText)
					return
				}
				lastIndex += idx + len(pattern)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Level 2: TestCompiler_WATMatch
// ---------------------------------------------------------------------------

// TestCompiler_WATMatch verifies that success fixtures produce WAT output
// matching the stored reference files, in both debug and release modes.
func TestCompiler_WATMatch(t *testing.T) {
	fixtures := discoverFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found")
	}

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			config := loadFixtureConfig(t, name)
			if config.expectsErrors() {
				t.Skip("error fixture — tested by TestCompiler_ErrorDiagnostics")
			}
			if binaryenFatalFixtures[name] {
				t.Skipf("triggers binaryen Fatal()")
			}

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
						t.Skipf("no %s fixture file", mode.name)
					}
					expected := normalize(string(expectedBytes))

					result := compileFixture(t, name, mode.release, config)
					if result.Panicked {
						t.Fatalf("panicked: %v", result.PanicValue)
					}
					if result.Module == nil {
						t.Fatal("compiler returned nil module")
					}

					// Filter to only error-level diagnostics for success fixtures
					var errors []*diagnostics.DiagnosticMessage
					for _, d := range result.Diagnostics {
					if d.Category == diagnostics.DiagnosticCategoryError {
							errors = append(errors, d)
						}
					}
					if len(errors) > 0 {
						t.Errorf("unexpected error diagnostics:\n%s", formatDiagnostics(errors))
						return
					}

					actual := normalize(result.Module.ToText(false))
					if actual != expected {
						diffPos := firstDiffPos(expected, actual)
						ctx := 200
						start := max(0, diffPos-ctx)
						end := min(len(expected), diffPos+ctx)
						endActual := min(len(actual), diffPos+ctx)
						t.Errorf("WAT mismatch at byte %d\n--- expected ---\n%s\n--- actual ---\n%s",
							diffPos, expected[start:end], actual[start:endActual])
					}
				})
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Level 3+4: TestCompiler_Runtime
// ---------------------------------------------------------------------------

// TestCompiler_Runtime compiles each success fixture to Wasm binary in both debug
// and release modes, instantiates via wazero with standard AS host imports
// (env.abort, env.trace, env.seed), calls _start/_initialize, and verifies no
// abort or trap occurs.
//
// This is the most valuable test — it proves the full pipeline works end-to-end.
func TestCompiler_Runtime(t *testing.T) {
	fixtures := discoverFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found")
	}

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			config := loadFixtureConfig(t, name)
			if config.expectsErrors() {
				t.Skip("error fixture")
			}
			if binaryenFatalFixtures[name] {
				t.Skip("triggers binaryen Fatal()")
			}
			if config.SkipInstantiate {
				t.Skip("fixture config says skipInstantiate")
			}

			// Check for glue — use registered glue or skip
			var glue *Glue
			if glueFixtures[name] {
				glue = getGlue(name)
				if glue == nil {
					t.Skipf("needs glue imports (not yet ported)")
				}
			}

			// Test both debug and release modes
			for _, mode := range []struct {
				name    string
				release bool
			}{
				{"debug", false},
				{"release", true},
			} {
				t.Run(mode.name, func(t *testing.T) {
					if mode.release && binaryenReleaseFatalFixtures[name] {
						t.Skip("crashes binaryen optimizer")
					}

					ctx := context.Background()
					wasmBytes, result := compileFixtureToBinary(t, name, mode.release, config)
					if result.Panicked {
						t.Fatalf("panicked during compilation: %v", result.PanicValue)
					}
					if wasmBytes == nil {
						t.Fatal("compilation produced nil binary")
					}
					if len(wasmBytes) == 0 {
						t.Fatal("compilation produced empty binary")
					}

					// Instantiate and run
					rr := instantiateAndRun(t, ctx, wasmBytes, glue)
					if rr.Runtime != nil {
						defer rr.Runtime.Close(ctx)
					}

					if rr.InstErr != nil {
						t.Fatalf("instantiation error: %v", rr.InstErr)
					}
					if rr.Aborted {
						t.Errorf("abort called: %s in %s(%d:%d)", rr.AbortMsg, rr.AbortFile, rr.AbortLine, rr.AbortCol)
					}
					if rr.StartErr != nil && !rr.Aborted {
						t.Errorf("runtime error: %v", rr.StartErr)
					}
				})
			}
		})
	}
}
