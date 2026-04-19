package voice

import (
	"testing"

	"github.com/brainlet/brainkit/test/fixtures"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// testFixtureDiscovery asserts the classifier marks the voice tree as
// needing AI. Anchors the domain so future mock-backed tests (TTS
// capture, STT round-trip) have a place to land.
func testFixtureDiscovery(t *testing.T, _ *suite.TestEnv) {
	needs := fixtures.ClassifyFixture("voice/basic")
	require.True(t, needs.AI, "voice/* fixtures must be flagged as needing AI")
}
