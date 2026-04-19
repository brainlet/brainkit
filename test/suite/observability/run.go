// Package observability hosts suite tests that exercise the Mastra
// Observability surface end-to-end from .ts: start a span, add
// attributes, end it, then retrieve the recorded trace through the
// public Observability API.
package observability

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("observability", func(t *testing.T) {
		t.Run("span_lifecycle", func(t *testing.T) { testSpanLifecycle(t, env) })
	})
}
