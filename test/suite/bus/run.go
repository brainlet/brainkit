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
		// call.go — brainkit.Call shared-inbox request/response
		t.Run("call_happy_path", func(t *testing.T) { testCallHappyPath(t, env) })
		t.Run("call_requires_deadline", func(t *testing.T) { testCallRequiresDeadline(t, env) })
		t.Run("call_with_call_timeout", func(t *testing.T) { testCallWithCallTimeout(t, env) })
		t.Run("call_timeout_error", func(t *testing.T) { testCallTimeoutError(t, env) })
		t.Run("call_cancelled_error", func(t *testing.T) { testCallCancelledError(t, env) })
		t.Run("call_concurrent_demux", func(t *testing.T) { testCallConcurrentDemux(t, env) })
		t.Run("call_raw_payload", func(t *testing.T) { testCallRawPayload(t, env) })

		// ts_call.go — .ts bus.call / bus.callTo wire envelope
		t.Run("ts_bus_call_happy_path", func(t *testing.T) { testTSBusCallHappyPath(t, env) })
		t.Run("ts_bus_call_requires_timeout", func(t *testing.T) { testTSBusCallRequiresTimeout(t, env) })
		t.Run("ts_bus_call_propagates_brainkit_error", func(t *testing.T) { testTSBusCallPropagatesBrainkitError(t, env) })
		t.Run("ts_bus_call_times_out", func(t *testing.T) { testTSBusCallTimesOut(t, env) })
		t.Run("go_bus_call_to_ts", func(t *testing.T) { testGoBusCallToTS(t, env) })
		t.Run("go_bus_call_ts_handler_throws_typed_error", func(t *testing.T) { testGoBusCallTSHandlerThrowsTypedError(t, env) })

		// call_cancel_failfast.go — Bundle C: cancel signal + exhausted metadata + metrics
		t.Run("call_emits_cancel_on_ctx_cancel", func(t *testing.T) { testCallEmitsCancelOnCtxCancel(t, env) })
		t.Run("call_no_cancel_signal_suppresses", func(t *testing.T) { testCallNoCancelSignalSuppresses(t, env) })
		t.Run("exhausted_event_carries_correlation_id", func(t *testing.T) { testExhaustedEventCarriesCorrelationId(t, env) })
		t.Run("caller_metrics_snapshot", func(t *testing.T) { testCallerMetricsSnapshot(t, env) })

		// call_stream.go — brainkit.CallStream + backpressure policies
		t.Run("call_stream_all_delivered", func(t *testing.T) { testCallStreamAllDelivered(t, env) })
		t.Run("call_stream_requires_callback", func(t *testing.T) { testCallStreamRequiresCallback(t, env) })
		t.Run("call_stream_buffer_error_policy", func(t *testing.T) { testCallStreamBufferErrorPolicy(t, env) })
		t.Run("call_stream_drop_newest", func(t *testing.T) { testCallStreamDropNewest(t, env) })
		t.Run("call_stream_handler_error_aborts", func(t *testing.T) { testCallStreamHandlerErrorAborts(t, env) })

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

		// pump.go
		t.Run("pump_schedule_latency", func(t *testing.T) { testPumpScheduleLatency(t, env) })
		t.Run("pump_responsive_after_idle", func(t *testing.T) { testPumpResponsiveAfterIdle(t, env) })

		// log.go
		t.Run("log_handler_ts_compartment", func(t *testing.T) { testLogHandlerTSCompartment(t, env) })
		t.Run("log_handler_multiple_files", func(t *testing.T) { testLogHandlerMultipleFiles(t, env) })
		t.Run("log_handler_nil_default", func(t *testing.T) { testLogHandlerNilDefault(t, env) })

		// error_contract.go
		t.Run("bus_error_response_carries_code", func(t *testing.T) { testBusErrorResponseCarriesCode(t, env) })

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

		// cross_feature.go — cross-feature adversarial (from adversarial/cross_feature_test.go)
		t.Run("cross_deploy_calls_go_tool", func(t *testing.T) { testCrossDeployCallsGoTool(t, env) })
		t.Run("cross_ts_tool_calls_another_ts_tool", func(t *testing.T) { testCrossTSToolCallsAnotherTSTool(t, env) })
		t.Run("cross_handler_calls_tool", func(t *testing.T) { testCrossHandlerCallsTool(t, env) })
		t.Run("cross_handler_reads_secret", func(t *testing.T) { testCrossHandlerReadsSecret(t, env) })
		t.Run("cross_handler_writes_fs", func(t *testing.T) { testCrossHandlerWritesFS(t, env) })
		t.Run("cross_go_tool_emits_bus_event", func(t *testing.T) { testCrossGoToolEmitsBusEvent(t, env) })
		t.Run("cross_traced_tool_call", func(t *testing.T) { testCrossTracedToolCall(t, env) })
		t.Run("cross_health_during_deploy_churn", func(t *testing.T) { testCrossHealthDuringDeployChurn(t, env) })
		t.Run("cross_metrics_track_schedules", func(t *testing.T) { testCrossMetricsTrackSchedules(t, env) })
		t.Run("cross_deploy_with_persistence_and_restart", func(t *testing.T) { testCrossDeployWithPersistenceAndRestart(t, env) })

		// error_contract_adv.go — bus error contract adversarial (from adversarial/error_contract_test.go)
		t.Run("error_contract_bus_not_found", func(t *testing.T) { testErrorContractBusNotFound(t, env) })
		t.Run("error_contract_bus_validation_error", func(t *testing.T) { testErrorContractBusValidationError(t, env) })
		t.Run("error_contract_bus_idempotent_deploy", func(t *testing.T) { testErrorContractBusIdempotentDeploy(t, env) })
		t.Run("error_contract_bus_deploy_error_bad_syntax", func(t *testing.T) { testErrorContractBusDeployErrorBadSyntax(t, env) })
		t.Run("error_contract_errors_as_all_types", func(t *testing.T) { testErrorContractErrorsAsAllTypes(t, env) })
		t.Run("error_contract_jsbridge_validation_error_missing_args", func(t *testing.T) { testErrorContractJSBridgeValidationErrorMissingArgs(t, env) })
		t.Run("error_contract_jsbridge_not_configured_secrets", func(t *testing.T) { testErrorContractJSBridgeNotConfiguredSecrets(t, env) })
		t.Run("error_contract_error_handler_persistence_error", func(t *testing.T) { testErrorContractErrorHandlerPersistenceError(t, env) })
		t.Run("error_contract_error_handler_deploy_error", func(t *testing.T) { testErrorContractErrorHandlerDeployError(t, env) })

		// input_abuse.go — bus input abuse adversarial (from adversarial/input_abuse_test.go)
		t.Run("input_abuse_bus_empty_topic", func(t *testing.T) { testInputAbuseBusEmptyTopic(t, env) })
		t.Run("input_abuse_bus_large_payload", func(t *testing.T) { testInputAbuseBusLargePayload(t, env) })
		t.Run("input_abuse_bus_deeply_nested_json", func(t *testing.T) { testInputAbuseBusDeeplyNestedJSON(t, env) })
		t.Run("input_abuse_bus_subscribe_empty_topic", func(t *testing.T) { testInputAbuseBusSubscribeEmptyTopic(t, env) })

		// e2e.go — multi-service chain E2E (from adversarial/e2e_scenarios_test.go + e2e/scenarios_test.go)
		t.Run("e2e_multi_service_chain", func(t *testing.T) { testE2EMultiServiceChain(t, env) })
		t.Run("e2e_streaming_response", func(t *testing.T) { testE2EStreamingResponse(t, env) })
		t.Run("e2e_multi_domain", func(t *testing.T) { testE2EMultiDomain(t, env) })

		// failure_cascade.go — failure cascade tests (from adversarial/failure_cascade_test.go)
		t.Run("cascade_deploy_with_broken_store", func(t *testing.T) { testCascadeDeployWithBrokenStore(t, env) })
		t.Run("cascade_corrupted_store", func(t *testing.T) { testCascadeCorruptedStore(t, env) })
		t.Run("cascade_publish_during_drain", func(t *testing.T) { testCascadePublishDuringDrain(t, env) })
		t.Run("cascade_eval_ts_during_close", func(t *testing.T) { testCascadeEvalTSDuringClose(t, env) })
		t.Run("cascade_secret_rotate_plugin_fails", func(t *testing.T) { testCascadeSecretRotatePluginFails(t, env) })
		t.Run("cascade_retry_exhausted", func(t *testing.T) { testCascadeRetryExhausted(t, env) })
		t.Run("cascade_handler_throw_no_reply_to", func(t *testing.T) { testCascadeHandlerThrowNoReplyTo(t, env) })
		t.Run("cascade_teardown_cleans_subscriptions", func(t *testing.T) { testCascadeTeardownCleansSubscriptions(t, env) })
		t.Run("cascade_schedule_no_handler", func(t *testing.T) { testCascadeScheduleNoHandler(t, env) })
		t.Run("cascade_concurrent_error_handler", func(t *testing.T) { testCascadeConcurrentErrorHandler(t, env) })

		// backend_advanced.go — ported from adversarial/backend_advanced_test.go + backend_matrix_test.go
		t.Run("concurrent_publish_50", func(t *testing.T) { testConcurrentPublish50(t, env) })
		t.Run("large_payload_100kb", func(t *testing.T) { testLargePayload100KB(t, env) })
		t.Run("dotted_topic_names", func(t *testing.T) { testDottedTopicNames(t, env) })
		t.Run("deploy_handler_call", func(t *testing.T) { testDeployHandlerCall(t, env) })
		t.Run("publish_reply", func(t *testing.T) { testPublishReply(t, env) })
		t.Run("error_code_on_bus", func(t *testing.T) { testErrorCodeOnBus(t, env) })

		// transport compliance — ported from transport/compliance_test.go
		t.Run("transport_compliance_publish_subscribe", func(t *testing.T) { testTransportCompliancePublishSubscribe(t, env) })
		t.Run("transport_compliance_correlation_id", func(t *testing.T) { testTransportComplianceCorrelationID(t, env) })
		t.Run("transport_compliance_dotted_topics", func(t *testing.T) { testTransportComplianceDottedTopics(t, env) })

		// surface_matrix.go — ported from adversarial/surface_matrix_test.go
		t.Run("surface_go_sdk", func(t *testing.T) { testSurfaceGoSDK(t, env) })
		t.Run("surface_ts_deployed", func(t *testing.T) { testSurfaceTSDeployed(t, env) })
		t.Run("surface_evalts", func(t *testing.T) { testSurfaceEvalTS(t, env) })
		t.Run("surface_error_consistency", func(t *testing.T) { testSurfaceErrorConsistency(t, env) })

		// transport_matrix.go — ported from transport/matrix_test.go
		t.Run("transport_matrix_tools_call", func(t *testing.T) { testTransportMatrixToolsCall(t, env) })
		t.Run("transport_matrix_tools_list", func(t *testing.T) { testTransportMatrixToolsList(t, env) })
		t.Run("transport_matrix_tools_resolve", func(t *testing.T) { testTransportMatrixToolsResolve(t, env) })
		t.Run("transport_matrix_fs_write_read", func(t *testing.T) { testTransportMatrixFSWriteRead(t, env) })
		t.Run("transport_matrix_fs_mkdir_list_stat_delete", func(t *testing.T) { testTransportMatrixFSMkdirListStatDelete(t, env) })
		t.Run("transport_matrix_agents_list_empty", func(t *testing.T) { testTransportMatrixAgentsListEmpty(t, env) })
		t.Run("transport_matrix_kit_deploy_teardown", func(t *testing.T) { testTransportMatrixKitDeployTeardown(t, env) })
		t.Run("transport_matrix_kit_redeploy", func(t *testing.T) { testTransportMatrixKitRedeploy(t, env) })
		t.Run("transport_matrix_registry_has_list", func(t *testing.T) { testTransportMatrixRegistryHasList(t, env) })
		t.Run("transport_matrix_async_correlation", func(t *testing.T) { testTransportMatrixAsyncCorrelation(t, env) })

		// audit.go — centralized audit log bus commands
		t.Run("audit_query_after_deploy", func(t *testing.T) { testAuditQueryAfterDeploy(t, env) })
		t.Run("audit_stats_response", func(t *testing.T) { testAuditStatsResponse(t, env) })
		t.Run("audit_prune_works", func(t *testing.T) { testAuditPruneWorks(t, env) })
		t.Run("audit_tool_call_recorded", func(t *testing.T) { testAuditToolCallRecorded(t, env) })
		t.Run("audit_metrics_get_includes_bus", func(t *testing.T) { testAuditMetricsGetIncludesBus(t, env) })
		t.Run("audit_no_module_commands_absent", func(t *testing.T) { testAuditNoModuleCommandsAbsent(t, env) })
	})
}
