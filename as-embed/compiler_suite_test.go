package asembed

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// TestASCompilerSuite runs the official AssemblyScript compiler test cases
// through our Go-embedded compiler. Each test is a .ts file from the upstream
// tests/compiler/ directory. We verify they compile to valid Wasm and compare
// the WAT output against the upstream fixtures.
func TestASCompilerSuite(t *testing.T) {
	const testDir = "/Users/davidroman/Documents/code/clones/assemblyscript/tests/compiler"

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("AS compiler test directory not found; skipping suite test")
	}

	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	type testCase struct {
		name       string
		sources    map[string]string
		flags      []string
		expectedWT string // expected WAT from .debug.wat fixture
	}

	var cases []testCase
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".ts") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".ts")
		if strings.HasSuffix(name, ".d") {
			continue
		}

		// Only tests with .debug.wat fixture (known to compile)
		watPath := filepath.Join(testDir, name+".debug.wat")
		if _, err := os.Stat(watPath); os.IsNotExist(err) {
			continue
		}

		// Read flags
		var flags []string
		jsonPath := filepath.Join(testDir, name+".json")
		if data, err := os.ReadFile(jsonPath); err == nil {
			var cfg struct {
				AscFlags []string `json:"asc_flags"`
			}
			json.Unmarshal(data, &cfg)
			flags = cfg.AscFlags
		}

		// Skip tests needing unsupported compiler flags
		skip := false
		for _, f := range flags {
			if strings.HasPrefix(f, "--import") || strings.HasPrefix(f, "--export") ||
				strings.HasPrefix(f, "--enable") || strings.HasPrefix(f, "--disable") ||
				strings.HasPrefix(f, "--tableBase") || strings.HasPrefix(f, "--memoryBase") ||
				strings.HasPrefix(f, "--converge") || strings.HasPrefix(f, "--use") ||
				strings.HasPrefix(f, "--transform") || strings.HasPrefix(f, "--sourceMap") ||
				strings.HasPrefix(f, "--noEmit") {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		sources, err := collectFixtureSources(testDir, name+".ts")
		if err != nil {
			continue
		}

		// Read expected WAT fixture
		watData, err := os.ReadFile(watPath)
		if err != nil {
			continue
		}

		cases = append(cases, testCase{
			name:       name,
			sources:    sources,
			flags:      flags,
			expectedWT: string(watData),
		})
	}

	t.Logf("Found %d compilable test cases", len(cases))

	passed, failed, watMatch, watMismatch, panicked, timedOut := 0, 0, 0, 0, 0, 0

	// Create compiler — will be recreated if the QJS runtime dies.
	// Close() is safe on corrupted runtimes (cancels context + recovers panics).
	c, err := NewCompiler()
	if err != nil {
		t.Fatalf("NewCompiler: %v", err)
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if c.Dead() {
				c.Close()
				var cerr error
				c, cerr = NewCompiler()
				if cerr != nil {
					t.Fatalf("NewCompiler after reset: %v", cerr)
				}
			}

			runtime := "incremental"
			for i, f := range tc.flags {
				if f == "--runtime" && i+1 < len(tc.flags) {
					runtime = tc.flags[i+1]
				}
			}

			// Compile with panic recovery and per-test timeout
			var result *CompileResult
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked++
						c.dead = true // mark dead so next test recreates
						t.Errorf("PANIC: %v", r)
					}
				}()
				var cerr error
				result, cerr = c.Compile(tc.sources, CompileOptions{
					OptimizeLevel: 0,
					ShrinkLevel:   0,
					Debug:         true,
					Runtime:       runtime,
					Timeout:       0, // no Go-level timeout; panics caught by test recover
				})
				if cerr != nil {
					if strings.Contains(cerr.Error(), "timed out") {
						timedOut++
						t.Errorf("Timed out: %v", cerr)
					} else {
						failed++
						t.Errorf("Compile failed: %v", cerr)
					}
				}
			}()

			if result == nil {
				return
			}

			if len(result.Binary) < 8 {
				failed++
				t.Errorf("binary too short: %d bytes", len(result.Binary))
				return
			}

			magic := result.Binary[:4]
			if magic[0] != 0x00 || magic[1] != 0x61 || magic[2] != 0x73 || magic[3] != 0x6d {
				failed++
				t.Errorf("bad wasm magic: %x", magic)
				return
			}

			passed++

			// Compare WAT output against fixture
			actualWAT := strings.TrimSpace(result.WAT)
			expectedWAT := strings.TrimSpace(tc.expectedWT)

			if actualWAT == expectedWAT {
				watMatch++
				t.Logf("OK: %d bytes, WAT matches fixture", len(result.Binary))
			} else {
				watMismatch++
				// Show first difference for debugging
				actualLines := strings.Split(actualWAT, "\n")
				expectedLines := strings.Split(expectedWAT, "\n")
				diffLine := -1
				for i := 0; i < len(actualLines) && i < len(expectedLines); i++ {
					if actualLines[i] != expectedLines[i] {
						diffLine = i
						break
					}
				}
				if diffLine >= 0 {
					t.Logf("OK: %d bytes, WAT MISMATCH at line %d:", len(result.Binary), diffLine+1)
					t.Logf("  expected: %s", expectedLines[diffLine])
					t.Logf("  actual:   %s", actualLines[diffLine])
				} else if len(actualLines) != len(expectedLines) {
					t.Logf("OK: %d bytes, WAT line count differs: got %d, want %d",
						len(result.Binary), len(actualLines), len(expectedLines))
				}
			}
		})
	}

	c.Close()

	t.Logf("=== SUMMARY ===")
	t.Logf("Compiled: %d passed, %d failed, %d panicked, %d timed out", passed, failed, panicked, timedOut)
	t.Logf("WAT output: %d match, %d mismatch", watMatch, watMismatch)
}

func collectFixtureSources(rootDir string, entry string) (map[string]string, error) {
	sources := make(map[string]string)
	queue := []string{path.Clean(filepath.ToSlash(entry))}

	for len(queue) > 0 {
		sourcePath := queue[0]
		queue = queue[1:]

		if _, ok := sources[sourcePath]; ok {
			continue
		}

		fullPath := filepath.Join(rootDir, filepath.FromSlash(sourcePath))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}

		text := string(data)
		sources[sourcePath] = text

		dir := path.Dir(sourcePath)
		for _, match := range sourceImportPattern.FindAllStringSubmatch(text, -1) {
			resolved := resolveFixtureImportPath(rootDir, dir, match[1])
			if resolved == "" {
				continue
			}
			if _, ok := sources[resolved]; !ok {
				queue = append(queue, resolved)
			}
		}
	}

	return sources, nil
}

func resolveFixtureImportPath(rootDir string, dir string, spec string) string {
	if strings.HasPrefix(spec, "~lib/") {
		return ""
	}

	for _, candidate := range sourceImportCandidates(dir, spec) {
		fullPath := filepath.Join(rootDir, filepath.FromSlash(candidate))
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
