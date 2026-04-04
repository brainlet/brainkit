// Package bus provides the bus domain test suite.
// All test functions take *suite.TestEnv and are registered via Run().
// The standalone bus_test.go creates a Full env for the memory fast path.
// Campaigns call Run() with transport-specific envs.
package bus

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run executes all bus domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("bus", func(t *testing.T) {
		// publish.go — JS bridge + deploy + bus.on flow
		t.Run("js_publish_returns_reply_to", func(t *testing.T) { testJSPublishReturnsReplyTo(t, env) })
		t.Run("js_emit_fire_and_forget", func(t *testing.T) { testJSEmitFireAndForget(t, env) })
		t.Run("js_reply_done_flag", func(t *testing.T) { testJSReplyDoneFlag(t, env) })
		t.Run("js_subscribe_receives_metadata", func(t *testing.T) { testJSSubscribeReceivesMetadata(t, env) })
		t.Run("go_to_js_round_trip", func(t *testing.T) { testGoToJSRoundTrip(t, env) })
		t.Run("deploy_with_bus_on", func(t *testing.T) { testDeployWithBusOn(t, env) })
		t.Run("streaming_chunks", func(t *testing.T) { testStreamingChunks(t, env) })
		t.Run("kit_register_agent_discovery", func(t *testing.T) { testKitRegisterAgentDiscovery(t, env) })

		// async.go — correlation, concurrency, cancellation
		t.Run("correlation_id_filtering", func(t *testing.T) { testCorrelationIDFiltering(t, env) })
		t.Run("multiple_in_flight", func(t *testing.T) { testMultipleInFlight(t, env) })
		t.Run("context_cancellation", func(t *testing.T) { testContextCancellation(t, env) })
		t.Run("subscribe_cancellation", func(t *testing.T) { testSubscribeCancellation(t, env) })

		// sdk_reply.go — sdk.Reply, sdk.SendChunk, sdk.SendToService
		t.Run("sdk_reply", func(t *testing.T) { testSDKReply(t, env) })
		t.Run("sdk_reply_go_to_go", func(t *testing.T) { testSDKReplyGoToGo(t, env) })
		t.Run("sdk_send_chunk", func(t *testing.T) { testSDKSendChunk(t, env) })
		t.Run("sdk_send_to_service", func(t *testing.T) { testSDKSendToService(t, env) })

		// failure.go — handler throw, retry, dead letter, exhausted event
		t.Run("sync_throw_error_response", func(t *testing.T) { testSyncThrowErrorResponse(t, env) })
		t.Run("async_rejection_error_response", func(t *testing.T) { testAsyncRejectionErrorResponse(t, env) })
		t.Run("handler_failed_event_emitted", func(t *testing.T) { testHandlerFailedEventEmitted(t, env) })
		t.Run("retry_policy_retries", func(t *testing.T) { testRetryPolicyRetries(t, env) })
		t.Run("retry_exhausted_dead_letter", func(t *testing.T) { testRetryExhaustedDeadLetter(t, env) })
		t.Run("exhausted_event_emitted", func(t *testing.T) { testExhaustedEventEmitted(t, env) })
		t.Run("retry_preserves_reply_to", func(t *testing.T) { testRetryPreservesReplyTo(t, env) })

		// ratelimit.go
		t.Run("bus_rate_limit_exceeds", func(t *testing.T) { testBusRateLimitExceeds(t, env) })

		// pump.go
		t.Run("pump_schedule_latency", func(t *testing.T) { testPumpScheduleLatency(t, env) })
		t.Run("pump_responsive_after_idle", func(t *testing.T) { testPumpResponsiveAfterIdle(t, env) })

		// log.go
		t.Run("log_handler_ts_compartment", func(t *testing.T) { testLogHandlerTSCompartment(t, env) })
		t.Run("log_handler_multiple_files", func(t *testing.T) { testLogHandlerMultipleFiles(t, env) })
		t.Run("log_handler_nil_default", func(t *testing.T) { testLogHandlerNilDefault(t, env) })

		// error_contract.go
		t.Run("bus_error_response_carries_code", func(t *testing.T) { testBusErrorResponseCarriesCode(t, env) })
		t.Run("result_meta_includes_code", func(t *testing.T) { testResultMetaIncludesCode(t, env) })

		// test_framework.go — JS built-in test framework
		t.Run("framework_passing_tests", func(t *testing.T) { testFrameworkPassingTests(t, env) })
		t.Run("framework_failing_test", func(t *testing.T) { testFrameworkFailingTest(t, env) })
		t.Run("framework_async_tests", func(t *testing.T) { testFrameworkAsyncTests(t, env) })
		t.Run("framework_deploy_and_test", func(t *testing.T) { testFrameworkDeployAndTest(t, env) })
		t.Run("framework_hooks", func(t *testing.T) { testFrameworkHooks(t, env) })
		t.Run("framework_not_assertions", func(t *testing.T) { testFrameworkNotAssertions(t, env) })

		// errors.go — bus error paths (from adversarial/bus_error_paths_test.go)
		t.Run("publish_to_command_topic", func(t *testing.T) { testPublishToCommandTopic(t, env) })
		t.Run("emit_to_command_topic", func(t *testing.T) { testEmitToCommandTopic(t, env) })
		t.Run("subscribe_receives_metadata_adv", func(t *testing.T) { testSubscribeReceivesMetadataAdv(t, env) })
		t.Run("reply_without_reply_to", func(t *testing.T) { testReplyWithoutReplyTo(t, env) })
		t.Run("send_to_nonexistent_service", func(t *testing.T) { testSendToNonexistentService(t, env) })
		t.Run("correlation_id_preserved", func(t *testing.T) { testCorrelationIDPreserved(t, env) })
		t.Run("multiple_replies", func(t *testing.T) { testMultipleReplies(t, env) })
		t.Run("subscribe_unsubscribe", func(t *testing.T) { testSubscribeUnsubscribe(t, env) })
		t.Run("deployment_namespace", func(t *testing.T) { testDeploymentNamespace(t, env) })
		t.Run("schedule_with_payload", func(t *testing.T) { testScheduleWithPayload(t, env) })

		// integration.go — multi-service chain
		t.Run("two_service_interaction", func(t *testing.T) { testTwoServiceInteraction(t, env) })

		// async_diag.go — async operation levels inside bus.on handlers
		t.Run("diag_await_promise_resolve", func(t *testing.T) { testDiagBusOnAwaitPromiseResolve(t, env) })
		t.Run("diag_await_set_timeout", func(t *testing.T) { testDiagBusOnAwaitSetTimeout(t, env) })
		t.Run("diag_await_tools_call", func(t *testing.T) { testDiagBusOnAwaitToolsCall(t, env) })
		t.Run("diag_await_fetch", func(t *testing.T) { testDiagBusOnAwaitFetch(t, env) })
		t.Run("diag_await_generate_text", func(t *testing.T) { testDiagBusOnAwaitGenerateText(t, env) })

		// log.go (continued) — concurrent logging
		t.Run("log_handler_concurrent", func(t *testing.T) { testLogHandlerConcurrent(t, env) })

		// surface.go — bus command matrix (valid input, empty input, garbage payload)
		t.Run("bus_matrix_valid_input", func(t *testing.T) { testBusMatrixValidInput(t, env) })
		t.Run("bus_matrix_empty_input", func(t *testing.T) { testBusMatrixEmptyInput(t, env) })
		t.Run("bus_matrix_garbage_payload", func(t *testing.T) { testBusMatrixGarbagePayload(t, env) })
	})
}
