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
	})
}
