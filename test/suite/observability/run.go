// Package observability hosts suite tests that capture exported spans
// from the .ts fixtures via a test-harness ObservabilityStorage. Today
// the domain is a stub; it anchors the contract so the captured-span
// graph assertions can land here.
package observability

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("observability", func(t *testing.T) {
		t.Run("fixture_discovery", func(t *testing.T) { testFixtureDiscovery(t, env) })
	})
}
