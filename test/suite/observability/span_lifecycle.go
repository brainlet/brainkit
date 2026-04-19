package observability

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/test/fixtures"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// testSpanLifecycle drives the existing observability/custom-span
// fixture through the fixture runner — which DOES transpile ES
// imports, so `import { Observability } from "agent"` resolves. The
// fixture exercises the full span lifecycle; this suite test proves
// the runner + classifier + deploy path wire it up correctly when
// driven from a suite harness rather than the fixtures package
// directly.
func testSpanLifecycle(t *testing.T, _ *suite.TestEnv) {
	// The shared fixtures helpers assume CWD is test/fixtures/ (two
	// levels below the project root). Chdir there so LoadExpect /
	// FixturesRoot resolve correctly.
	projectRoot := findProjectRoot(t)
	prevCwd, _ := os.Getwd()
	require.NoError(t, os.Chdir(filepath.Join(projectRoot, "test", "fixtures")))
	t.Cleanup(func() { _ = os.Chdir(prevCwd) })

	runner := fixtures.NewRunner(filepath.Join(projectRoot, "fixtures"))
	runner.RunMatching(t, "observability/custom-span")
}

// findProjectRoot walks up from CWD until a go.mod is found; suite
// tests live 3 levels below the root (test/suite/<domain>/), so the
// shared fixtures.FixturesRoot helper can't be used verbatim.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	for cur := wd; cur != "/" && cur != ""; cur = filepath.Dir(cur) {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
	}
	t.Fatalf("go.mod not found walking up from %s", wd)
	return ""
}
