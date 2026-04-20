package workflows

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// RunStableSubset runs the workflow checks that are most useful for transport
// and persistence promotion campaigns.
func RunStableSubset(t *testing.T, env *suite.TestEnv) {
	t.Run("workflows/stable_subset", func(t *testing.T) {
		t.Run("start_sequential", func(t *testing.T) { testStartSequential(t, env) })
		t.Run("suspend_resume", func(t *testing.T) { testSuspendResume(t, env) })
		t.Run("cancel", func(t *testing.T) { testCancel(t, env) })
		t.Run("runs_on_transport", func(t *testing.T) { testRunsOnTransport(t, env) })
		t.Run("crash_recovery_suspended_on_transport", func(t *testing.T) { testCrashRecoverySuspendedOnTransport(t, env) })
	})
}
