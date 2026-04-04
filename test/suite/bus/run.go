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
	})
}
