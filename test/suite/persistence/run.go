package persistence

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("persistence", func(t *testing.T) {
		// Store operations — deploy/teardown/order persistence across kernel restarts
		t.Run("deploy_survives_restart", func(t *testing.T) { testDeploySurvivesRestart(t, env) })
		t.Run("teardown_removes_from_store", func(t *testing.T) { testTeardownRemovesFromStore(t, env) })
		t.Run("order_preserved", func(t *testing.T) { testOrderPreserved(t, env) })
		t.Run("failed_redeploy_does_not_block", func(t *testing.T) { testFailedRedeployDoesNotBlock(t, env) })
		t.Run("package_name_survives_restart", func(t *testing.T) { testPackageNameSurvivesRestart(t, env) })
		t.Run("redeploy_preserves_metadata", func(t *testing.T) { testRedeployPreservesMetadata(t, env) })
		t.Run("with_restoring_skips_persist", func(t *testing.T) { testWithRestoringSkipsPersist(t, env) })
		t.Run("role_preserved_across_restart", func(t *testing.T) { testRolePreservedAcrossRestart(t, env) })
		t.Run("schedule_catchup_on_restart", func(t *testing.T) { testScheduleCatchUpOnRestart(t, env) })
		t.Run("recurring_schedule_restarts_correctly", func(t *testing.T) { testRecurringScheduleRestartsCorrectly(t, env) })
		t.Run("deploy_order_preserved_exactly", func(t *testing.T) { testDeployOrderPreservedExactly(t, env) })

		// Schedule persistence — schedule-specific restart behavior
		t.Run("schedule_survives_restart", func(t *testing.T) { testScheduleSurvivesRestart(t, env) })
		t.Run("missed_recurring_catchup", func(t *testing.T) { testMissedRecurringCatchUp(t, env) })
		t.Run("expired_one_time_fires", func(t *testing.T) { testExpiredOneTimeFires(t, env) })

		// Edge cases — corrupt store recovery
		t.Run("corrupt_deployment_table", func(t *testing.T) { testCorruptDeploymentTable(t, env) })
		t.Run("corrupt_schedule_table", func(t *testing.T) { testCorruptScheduleTable(t, env) })

		// backend_matrix.go — ported from adversarial/backend_matrix_test.go + persistence_matrix_test.go
		t.Run("deploy_persist_restart", func(t *testing.T) { testDeployPersistRestart(t, env) })
		t.Run("secrets_survive_restart", func(t *testing.T) { testSecretsSurviveRestart(t, env) })
		t.Run("multi_deploy_order_and_metadata", func(t *testing.T) { testMultiDeployOrderAndMetadata(t, env) })
		t.Run("multiple_schedules_survive", func(t *testing.T) { testMultipleSchedulesSurvive(t, env) })
		t.Run("deploy_with_bus_handler_survives_restart", func(t *testing.T) { testDeployWithBusHandlerSurvivesRestart(t, env) })
	})
}
