package scheduling

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("scheduling", func(t *testing.T) {
		t.Run("every_fires_repeatedly", func(t *testing.T) { testEveryFiresRepeatedly(t, env) })
		t.Run("in_fires_once", func(t *testing.T) { testInFiresOnce(t, env) })
		t.Run("unschedule", func(t *testing.T) { testUnschedule(t, env) })
		t.Run("invalid_expression", func(t *testing.T) { testInvalidExpression(t, env) })
		t.Run("teardown_cancels", func(t *testing.T) { testTeardownCancelsSchedules(t, env) })
		t.Run("drain_skips_firing", func(t *testing.T) { testDrainSkipsFiring(t, env) })
	})
}
