// Package voice hosts suite tests for the voice providers. Real
// tests drive OpenAIVoice against api.openai.com with a real key —
// no mocks — to prove the polyfill + fetch + FormData + stream
// plumbing reach the wire.
package voice

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run executes all voice-domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("voice", func(t *testing.T) {
		t.Run("tts_real", func(t *testing.T) { testTTSReal(t, env) })
	})
}
