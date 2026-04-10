// Package stress provides the stress/concurrency domain test suite.
// All test functions take *suite.TestEnv and are registered via Run().
// The standalone stress_test.go creates a Full env for the memory fast path.
// Campaigns call Run() with transport-specific envs.
package stress

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
)

// Run executes all stress domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("stress", func(t *testing.T) {
		// gc.go -- GC and memory pressure tests
		t.Run("gc_single_kernel_clean_close", func(t *testing.T) { testGCSingleKernelCleanClose(t, env) })
		t.Run("gc_multiple_kernel_clean_close", func(t *testing.T) { testGCMultipleKernelCleanClose(t, env) })
		t.Run("gc_ten_kernel_clean_close", func(t *testing.T) { testGCTenKernelCleanClose(t, env) })
		t.Run("gc_zero_leak_quickjs_memory", func(t *testing.T) { testGCZeroLeakQuickJSMemory(t, env) })
		t.Run("gc_zero_leak_ses_runtime", func(t *testing.T) { testGCZeroLeakSESRuntime(t, env) })

		// scaling.go -- Pool scaling and strategy tests
		t.Run("pool_spawn_and_kill", func(t *testing.T) { testPoolSpawnAndKill(t, env) })
		t.Run("pool_scale_up_down", func(t *testing.T) { testPoolScaleUpDown(t, env) })
		t.Run("pool_duplicate_and_not_found", func(t *testing.T) { testPoolDuplicateAndNotFound(t, env) })
		t.Run("pool_shared_tools", func(t *testing.T) { testPoolSharedTools(t, env) })
		t.Run("strategy_static", func(t *testing.T) { testStrategyStatic(t, env) })
		t.Run("pool_evaluate_and_scale", func(t *testing.T) { testPoolEvaluateAndScale(t, env) })
		t.Run("pool_instances_process_messages", func(t *testing.T) { testPoolInstancesProcessMessages(t, env) })

		// concurrent.go -- Concurrent operation tests
		t.Run("parallel_deploy", func(t *testing.T) { testParallelDeploy(t, env) })
		t.Run("parallel_publish", func(t *testing.T) { testParallelPublish(t, env) })
		t.Run("parallel_eval_ts", func(t *testing.T) { testParallelEvalTS(t, env) })
		t.Run("deploy_during_handler", func(t *testing.T) { testDeployDuringHandler(t, env) })
		t.Run("teardown_during_handler", func(t *testing.T) { testTeardownDuringHandler(t, env) })
		t.Run("deploy_teardown_race_same_source", func(t *testing.T) { testDeployTeardownRaceOnSameSource(t, env) })
		t.Run("stress_deploy_teardown_cycles", func(t *testing.T) { testStressDeployTeardownCycles(t, env) })
		t.Run("redeploy_race", func(t *testing.T) { testRedeployRace(t, env) })
		t.Run("deploy_during_drain", func(t *testing.T) { testDeployDuringDrain(t, env) })

		// concurrency.go -- Adversarial concurrency tests
		t.Run("concurrency_deploy_teardown_race", func(t *testing.T) { testConcurrencyDeployTeardownRace(t, env) })
		t.Run("concurrency_publish_unsubscribe_race", func(t *testing.T) { testConcurrencyPublishUnsubscribeRace(t, env) })
		t.Run("concurrency_secret_set_get_race", func(t *testing.T) { testConcurrencySecretSetGetRace(t, env) })
		t.Run("concurrency_mass_deploy_teardown", func(t *testing.T) { testConcurrencyMassDeployTeardown(t, env) })
		t.Run("concurrency_schedule_unschedule_race", func(t *testing.T) { testConcurrencyScheduleUnscheduleRace(t, env) })
		t.Run("concurrency_close_during_handlers", func(t *testing.T) { testConcurrencyCloseDuringHandlers(t, env) })
		t.Run("concurrency_parallel_eval_ts", func(t *testing.T) { testConcurrencyParallelEvalTS(t, env) })
		t.Run("concurrency_storage_add_remove_race", func(t *testing.T) { testConcurrencyStorageAddRemoveRace(t, env) })
		t.Run("concurrency_metrics_during_churn", func(t *testing.T) { testConcurrencyMetricsDuringChurn(t, env) })
		t.Run("concurrency_shared_sqlite_store", func(t *testing.T) { testConcurrencySharedSQLiteStore(t, env) })
		t.Run("concurrency_deploy_during_restore", func(t *testing.T) { testConcurrencyDeployDuringRestore(t, env) })
		t.Run("concurrency_rbac_assign_check_race", func(t *testing.T) { testConcurrencyRBACAssignCheckRace(t, env) })

		// concurrency_stress.go -- Concurrency stress tests
		t.Run("100_deploys_simultaneously", func(t *testing.T) { test100DeploysSimultaneously(t, env) })
		t.Run("1000_bus_publishes", func(t *testing.T) { test1000BusPublishes(t, env) })
		t.Run("secret_rotation_during_reads", func(t *testing.T) { testSecretRotationDuringReads(t, env) })
		t.Run("deploy_while_eval_ts", func(t *testing.T) { testDeployWhileEvalTS(t, env) })
		t.Run("tool_calls_under_load", func(t *testing.T) { testToolCallsUnderLoad(t, env) })
		t.Run("schedule_storm", func(t *testing.T) { testScheduleStorm(t, env) })
		t.Run("multi_surface_simultaneous", func(t *testing.T) { testMultiSurfaceSimultaneous(t, env) })

		// exhaustion.go -- Resource exhaustion tests
		t.Run("exhaustion_memory_bomb", func(t *testing.T) { testExhaustionMemoryBomb(t, env) })
		t.Run("exhaustion_stack_overflow", func(t *testing.T) { testExhaustionStackOverflow(t, env) })
		t.Run("exhaustion_promise_flood", func(t *testing.T) { testExhaustionPromiseFlood(t, env) })
		t.Run("exhaustion_deploy_bomb", func(t *testing.T) { testExhaustionDeployBomb(t, env) })
		t.Run("exhaustion_fetch_bomb", func(t *testing.T) { testExhaustionFetchBomb(t, env) })
		t.Run("exhaustion_lifecycle_churn", func(t *testing.T) { testExhaustionLifecycleChurn(t, env) })
		t.Run("exhaustion_output_bomb", func(t *testing.T) { testExhaustionOutputBomb(t, env) })
		t.Run("exhaustion_concurrent_eval_ts", func(t *testing.T) { testExhaustionConcurrentEvalTS(t, env) })
		t.Run("exhaustion_large_payload_via_js", func(t *testing.T) { testExhaustionLargePayloadViaJS(t, env) })
		t.Run("exhaustion_timer_bomb", func(t *testing.T) { testExhaustionTimerBomb(t, env) })
		t.Run("exhaustion_secret_value_bomb", func(t *testing.T) { testExhaustionSecretValueBomb(t, env) })
		t.Run("exhaustion_json_stringify_hijack", func(t *testing.T) { testExhaustionJSONStringifyHijack(t, env) })
		t.Run("exhaustion_filesystem_fill", func(t *testing.T) { testExhaustionFilesystemFill(t, env) })
		t.Run("exhaustion_pump_starvation", func(t *testing.T) { testExhaustionPumpStarvation(t, env) })
		t.Run("exhaustion_persistence_bomb", func(t *testing.T) { testExhaustionPersistenceBomb(t, env) })
		t.Run("eval_ts_infinite_loop", func(t *testing.T) { testEvalTSInfiniteLoop(t, env) })

		// e2e_stress.go — E2E stress scenarios (from adversarial/e2e_scenarios_test.go + e2e/scenarios_test.go)
		t.Run("e2e_multiple_kernels", func(t *testing.T) { testE2EMultipleKernels(t, env) })
		t.Run("e2e_concurrent_operations", func(t *testing.T) { testE2EConcurrentOperations(t, env) })
	})
}

// --- helpers ---

// sendAndReceive publishes a typed message and waits for the raw response.
func sendAndReceive(t *testing.T, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, msg)
	if err != nil {
		t.Logf("publish failed: %v", err)
		return nil, false
	}

	ch := make(chan json.RawMessage, 1)
	unsub, err := rt.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	if err != nil {
		t.Logf("subscribe failed: %v", err)
		return nil, false
	}
	defer unsub()

	select {
	case payload := <-ch:
		return payload, true
	case <-ctx.Done():
		return nil, false
	}
}

// responseHasError checks if a bus response contains an error field.
func responseHasError(payload json.RawMessage) bool {
	var resp struct {
		Error string `json:"error"`
	}
	json.Unmarshal(payload, &resp)
	return resp.Error != ""
}
