// Package voicerealtime hosts suite tests for the WebSocket-backed
// realtime voice path. Boots a local WebSocket echo server and drives
// OpenAIRealtimeVoice.connect() at it to validate the full TCP →
// WebSocket upgrade path from .ts through jsbridge.
package voicerealtime

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("voice-realtime", func(t *testing.T) {
		t.Run("ws_echo_connect", func(t *testing.T) { testWSEchoConnect(t, env) })
	})
}
