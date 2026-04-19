package voice

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// testScaffoldPlaceholder pins the domain without pretending to do
// work. Real tests (httptest TTS mock + audio-sink capture) land here
// per plans-03/fixture-coverage-plan.md §14.
func testScaffoldPlaceholder(t *testing.T, _ *suite.TestEnv) {
	t.Skip("TODO: httptest TTS mock + audio sink capture — plans-03 §14")
}
