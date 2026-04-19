package voicerealtime

import (
	"testing"

	"github.com/brainlet/brainkit/test/fixtures"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// testFixtureDiscovery anchors the domain. The realtime-specific
// path sits under voice/openai-realtime/* today; a future suite
// test here will drive those fixtures against a local mock.
func testFixtureDiscovery(t *testing.T, _ *suite.TestEnv) {
	needs := fixtures.ClassifyFixture("voice/openai-realtime/connect")
	require.True(t, needs.AI, "openai-realtime fixtures inherit voice/* AI classification")
}
