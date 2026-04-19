// Package voicerealtime hosts suite tests for the WebSocket-backed
// realtime voice path. Today the domain is a stub; future tests will
// boot an httptest WebSocket echo server and drive the OpenAIRealtime
// client fixture against it.
package voicerealtime

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("voice-realtime", func(t *testing.T) {
		t.Run("scaffold_placeholder", func(t *testing.T) { testScaffoldPlaceholder(t, env) })
	})
}
