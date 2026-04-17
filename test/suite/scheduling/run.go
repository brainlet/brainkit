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

		// E2E and input abuse (from adversarial e2e_scenarios_test.go + input_abuse_test.go)
		t.Run("e2e_schedule_fires", func(t *testing.T) { testE2EScheduleFires(t, env) })
		t.Run("input_abuse_invalid_expression", func(t *testing.T) { testInputAbuseScheduleInvalidExpression(t, env) })
		t.Run("input_abuse_empty_topic", func(t *testing.T) { testInputAbuseScheduleEmptyTopic(t, env) })

		// backend_advanced.go — ported from adversarial/backend_advanced_test.go
		t.Run("schedule_fire_on_transport", func(t *testing.T) { testScheduleFireOnTransport(t, env) })

		// bus_commands.go — schedule management via typed bus commands
		t.Run("bus/create", func(t *testing.T) { testScheduleCreateViaBus(t, env) })
		t.Run("bus/create_invalid", func(t *testing.T) { testScheduleCreateInvalidExpression(t, env) })
		t.Run("bus/create_blocks_command", func(t *testing.T) { testScheduleCreateBlocksCommandTopic(t, env) })
		t.Run("bus/list", func(t *testing.T) { testScheduleListViaBus(t, env) })
		t.Run("bus/cancel", func(t *testing.T) { testScheduleCancelViaBus(t, env) })
		t.Run("bus/fires_on_topic", func(t *testing.T) { testScheduleCreateFiresOnTopic(t, env) })
		t.Run("bus/one_time_fires", func(t *testing.T) { testScheduleCreateOneTimeFires(t, env) })
		t.Run("bus/fires_with_payload", func(t *testing.T) { testScheduleCreateWithPayload(t, env) })
		t.Run("bus/cancel_stops_firing", func(t *testing.T) { testScheduleCancelStopsFiring(t, env) })

		// Module-absent path
		t.Run("no_module_throws_not_configured", func(t *testing.T) { testNoModuleThrowsNotConfigured(t, env) })
	})
}
