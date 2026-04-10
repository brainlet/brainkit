// Package rbac previously provided the RBAC domain test suite.
// RBAC has been removed from brainkit.
package rbac

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run is a no-op — RBAC has been removed.
func Run(t *testing.T, _ *suite.TestEnv) {
	t.Run("rbac", func(t *testing.T) {
		t.Skip("RBAC has been removed from brainkit")
	})
}
