package health

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("health", func(t *testing.T) {
		// checks.go
		t.Run("alive_when_running", func(t *testing.T) { testAliveWhenRunning(t, env) })
		t.Run("ready_when_running", func(t *testing.T) { testReadyWhenRunning(t, env) })
		t.Run("ready_false_when_draining", func(t *testing.T) { testReadyFalseWhenDraining(t, env) })
		t.Run("status_running", func(t *testing.T) { testStatusRunning(t, env) })
		t.Run("transport_probe", func(t *testing.T) { testTransportProbe(t, env) })
		t.Run("storage_bridge_check", func(t *testing.T) { testStorageBridgeCheck(t, env) })
		t.Run("status_draining", func(t *testing.T) { testStatusDraining(t, env) })
		t.Run("deployments_count", func(t *testing.T) { testDeploymentsCount(t, env) })

		// shutdown.go (all use fresh kernels since they drain/close)
		t.Run("drains_before_close", func(t *testing.T) { testDrainsBeforeClose(t, env) })
		t.Run("drain_timeout_forces_close", func(t *testing.T) { testDrainTimeoutForcesClose(t, env) })
		t.Run("close_still_works", func(t *testing.T) { testCloseStillWorks(t, env) })
		t.Run("messages_dropped_during_drain", func(t *testing.T) { testMessagesDroppedDuringDrain(t, env) })
		t.Run("evalts_works_during_drain", func(t *testing.T) { testEvalTSWorksDuringDrain(t, env) })

		// metrics.go
		t.Run("metrics_reflects_state", func(t *testing.T) { testMetricsReflectsState(t, env) })

		// probes.go (health probes — most use fresh kernels with specific configs)
		t.Run("probe_ai_provider_real_openai", func(t *testing.T) { testProbeAIProviderRealOpenAI(t, env) })
		t.Run("probe_ai_provider_bad_key", func(t *testing.T) { testProbeAIProviderBadKey(t, env) })
		t.Run("probe_ai_provider_not_registered", func(t *testing.T) { testProbeAIProviderNotRegistered(t, env) })
		t.Run("probe_storage_inmemory", func(t *testing.T) { testProbeStorageInMemory(t, env) })
		t.Run("probe_vector_store_pgvector", func(t *testing.T) { testProbeVectorStoreRealPgVector(t, env) })
		t.Run("probe_all", func(t *testing.T) { testProbeAll(t, env) })
		t.Run("probe_periodic_ticker", func(t *testing.T) { testProbePeriodicTicker(t, env) })

		// degraded.go (adversarial degraded health scenarios)
		t.Run("alive_after_heavy_load-degraded", func(t *testing.T) { testAliveAfterHeavyLoad(t, env) })
		t.Run("ready_toggle_during_drain-degraded", func(t *testing.T) { testReadyToggleDuringDrain(t, env) })
		t.Run("full_health_check_categories-degraded", func(t *testing.T) { testFullHealthCheckCategories(t, env) })
		t.Run("health_with_tracing_store-degraded", func(t *testing.T) { testHealthWithTracingStore(t, env) })
		t.Run("health_with_storage_bridges-degraded", func(t *testing.T) { testHealthWithStorageBridges(t, env) })
		t.Run("metrics_reflect_deployments-degraded", func(t *testing.T) { testMetricsReflectDeployments(t, env) })
		t.Run("uptime_increases-degraded", func(t *testing.T) { testUptimeIncreases(t, env) })
		t.Run("health_after_close-degraded", func(t *testing.T) { testHealthAfterClose(t, env) })
		t.Run("persistence_store_health-degraded", func(t *testing.T) { testPersistenceStoreHealth(t, env) })

		// shutdown_adv.go (adversarial shutdown tests)
		t.Run("shutdown_graceful_with_active_deployments-adv", func(t *testing.T) { testShutdownGracefulWithActiveDeployments(t, env) })
		t.Run("shutdown_with_active_schedules-adv", func(t *testing.T) { testShutdownWithActiveSchedules(t, env) })
		t.Run("shutdown_with_active_subscriptions-adv", func(t *testing.T) { testShutdownWithActiveSubscriptions(t, env) })
		t.Run("shutdown_drain_timeout-adv", func(t *testing.T) { testShutdownDrainTimeoutAdv(t, env) })
		t.Run("shutdown_concurrent_close-adv", func(t *testing.T) { testShutdownConcurrentClose(t, env) })
		t.Run("shutdown_storage_access_before_close-adv", func(t *testing.T) { testShutdownStorageAccessBeforeClose(t, env) })
	})
}
