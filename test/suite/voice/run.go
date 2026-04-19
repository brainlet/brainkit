// Package voice hosts suite tests for the voice providers. Real
// tests boot Go-side infra (httptest TTS mock, audio-sink capture)
// that pure .ts fixtures can't express, then drive an OpenAIVoice
// through the mock and assert the round-tripped bytes.
package voice

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run executes all voice-domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("voice", func(t *testing.T) {
		t.Run("tts_mock", func(t *testing.T) { testTTSMock(t, env) })
	})
}
