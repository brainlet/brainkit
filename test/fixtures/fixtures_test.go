package fixtures_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/fixtures"
)

func TestFixtures(t *testing.T) {
	runner := fixtures.NewRunner(fixtures.FixturesRoot(t))
	runner.RunAll(t)
}
