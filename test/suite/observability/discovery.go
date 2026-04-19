package observability

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// testScaffoldPlaceholder pins the domain. A future test here will
// plug a capturing ObservabilityStorage into env.Kit, re-run the
// end-to-end TS fixture, and assert the captured span tree.
func testScaffoldPlaceholder(t *testing.T, _ *suite.TestEnv) {
	t.Skip("TODO: capturing ObservabilityStorage + span-graph assertions — plans-03 §14")
}
