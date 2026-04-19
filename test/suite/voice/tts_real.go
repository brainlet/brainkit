package voice

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/test/fixtures"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// findProjectRoot walks up from CWD until a go.mod is found.
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

// testTTSReal runs the real voice/openai/speak fixture against the
// real OpenAI TTS endpoint using OPENAI_API_KEY from the environment.
// Proves the OpenAIVoice polyfill + fetch + FormData + stream
// plumbing deliver non-empty audio bytes end-to-end. No mocks: the
// fixture hits api.openai.com directly through jsbridge fetch and
// drains the returned audio stream inside QuickJS.
func testTTSReal(t *testing.T, _ *suite.TestEnv) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("real OpenAI TTS test requires OPENAI_API_KEY")
	}

	// OPENAI_BASE_URL leaks from earlier sibling tests would redirect
	// this suite test away from the real endpoint; clear it so we
	// always hit api.openai.com.
	if prev, ok := os.LookupEnv("OPENAI_BASE_URL"); ok {
		require.NoError(t, os.Unsetenv("OPENAI_BASE_URL"))
		t.Cleanup(func() { _ = os.Setenv("OPENAI_BASE_URL", prev) })
	}

	// The shared fixtures helpers assume CWD is test/fixtures/.
	projectRoot := findProjectRoot(t)
	prevCwd, _ := os.Getwd()
	require.NoError(t, os.Chdir(filepath.Join(projectRoot, "test", "fixtures")))
	t.Cleanup(func() { _ = os.Chdir(prevCwd) })

	runner := fixtures.NewRunner(filepath.Join(projectRoot, "fixtures"))
	runner.RunMatching(t, "voice/openai/speak")
}
