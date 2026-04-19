// Package voice provides the voice domain test suite.
//
// These tests wrap the .ts voice fixtures with Go-side infrastructure:
// HTTP mock TTS endpoints, audio sink capture, and byte-count asserts
// that a pure .ts fixture can't express.
package voice

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run executes all voice-domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("voice", func(t *testing.T) {
		t.Run("fixture_discovery", func(t *testing.T) { testFixtureDiscovery(t, env) })
	})
}
