// Package voice hosts suite tests for the voice providers. Today
// the domain is a scaffold — it claims the path so future tests
// (httptest TTS mock, audio-sink capture, STT round-trip) have a
// place to land without reshaping the suite tree.
package voice

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run executes all voice-domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("voice", func(t *testing.T) {
		t.Run("scaffold_placeholder", func(t *testing.T) { testScaffoldPlaceholder(t, env) })
	})
}
