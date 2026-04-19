package voice

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/brainlet/brainkit/test/fixtures"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// testTTSMock boots an httptest OpenAI-TTS mock, points
// OPENAI_BASE_URL at it, then drives the existing voice/openai/speak
// fixture through it. Confirms the fixture reaches OpenAI's TTS
// endpoint with multipart/binary response shape — no real OpenAI
// network call, deterministic in CI.
func testTTSMock(t *testing.T, _ *suite.TestEnv) {
	mockBytes := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/audio/speech" {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", "12")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockBytes)
	}))
	t.Cleanup(srv.Close)

	prevBase := os.Getenv("OPENAI_BASE_URL")
	prevKey := os.Getenv("OPENAI_API_KEY")
	require.NoError(t, os.Setenv("OPENAI_BASE_URL", srv.URL+"/v1"))
	if prevKey == "" {
		require.NoError(t, os.Setenv("OPENAI_API_KEY", "sk-test-voice-mock"))
	}
	t.Cleanup(func() {
		if prevBase == "" {
			_ = os.Unsetenv("OPENAI_BASE_URL")
		} else {
			_ = os.Setenv("OPENAI_BASE_URL", prevBase)
		}
		if prevKey == "" {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
	})

	// The shared fixtures helpers assume CWD is test/fixtures/ (two
	// levels below the project root). Chdir there for the test run
	// so LoadExpect / FixturesRoot resolve correctly.
	projectRoot := findProjectRoot(t)
	prevCwd, _ := os.Getwd()
	require.NoError(t, os.Chdir(filepath.Join(projectRoot, "test", "fixtures")))
	t.Cleanup(func() { _ = os.Chdir(prevCwd) })

	runner := fixtures.NewRunner(filepath.Join(projectRoot, "fixtures"))
	runner.RunMatching(t, "voice/openai/speak")

	assert.GreaterOrEqual(t, atomic.LoadInt32(&hits), int32(1),
		"mock /v1/audio/speech should have been hit at least once")
}
