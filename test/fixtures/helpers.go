package fixtures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// FixturesRoot returns the absolute path to the fixtures/ directory.
// It walks up from the working directory to find the project root (go.mod),
// then returns <root>/fixtures.
func FixturesRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// test/fixtures/ is two levels below the project root
	root := filepath.Join(wd, "..", "..")
	// Verify go.mod exists at the expected root
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("project root not found at %s: %v", root, err)
	}
	return filepath.Join(root, "fixtures")
}

// LoadTSFixtureRaw reads the raw .ts source for a fixture given its relative path
// from the ts/ directory (e.g. "agent/generate/basic").
func LoadTSFixtureRaw(t *testing.T, relPath string) string {
	t.Helper()
	path := filepath.Join(FixturesRoot(t), "ts", relPath, "index.ts")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture ts/%s: %v", relPath, err)
	}
	return string(source)
}

// LoadExpect reads the expect.json sidecar for a fixture given its relative path
// from the ts/ directory (e.g. "agent/generate/basic").
// Returns nil if no expect.json exists.
func LoadExpect(t *testing.T, relPath string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(FixturesRoot(t), "ts", relPath, "expect.json"))
	if err != nil {
		return nil
	}
	var expect map[string]any
	if err := json.Unmarshal(data, &expect); err != nil {
		t.Fatalf("parse expect.json for %s: %v", relPath, err)
	}
	return expect
}

// AssertExpect compares actual output against an expect.json map using the
// fixture assertion conventions:
//   - "*"           → key must exist (any value)
//   - "~prefix"     → actual must contain the substring after ~
//   - bool          → exact bool match
//   - float64       → numeric delta of 0.01
//   - anything else → exact equality
func AssertExpect(t *testing.T, fixtureName string, actual, expect map[string]any) {
	t.Helper()
	for key, expected := range expect {
		actualVal, exists := actual[key]
		if !exists {
			t.Errorf("[%s] missing key %q in output", fixtureName, key)
			continue
		}
		switch ev := expected.(type) {
		case bool:
			assert.Equal(t, ev, actualVal, "[%s] key %s", fixtureName, key)
		case float64:
			assert.InDelta(t, ev, actualVal, 0.01, "[%s] key %s", fixtureName, key)
		case string:
			if ev == "*" {
				assert.NotNil(t, actualVal, "[%s] key %s should exist", fixtureName, key)
			} else if strings.HasPrefix(ev, "~") {
				assert.Contains(t, actualVal, ev[1:], "[%s] key %s", fixtureName, key)
			} else {
				assert.Equal(t, ev, actualVal, "[%s] key %s", fixtureName, key)
			}
		default:
			assert.Equal(t, expected, actualVal, "[%s] key %s", fixtureName, key)
		}
	}
}

// Truncate returns the first n bytes of s, appending "..." if truncated.
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
