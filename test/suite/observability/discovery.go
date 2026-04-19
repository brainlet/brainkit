package observability

import (
	"testing"

	"github.com/brainlet/brainkit/test/fixtures"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// testFixtureDiscovery anchors the domain. A future test here will
// plug a capturing ObservabilityStorage into env.Kit, re-run the
// end-to-end TS fixture, and assert the captured span tree.
func testFixtureDiscovery(t *testing.T, _ *suite.TestEnv) {
	needs := fixtures.ClassifyFixture("observability/end-to-end")
	require.True(t, needs.AI, "observability/* fixtures inherit the category-wide AI classification")
}
