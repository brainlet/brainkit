// Package envelope tests the wire envelope contract: every typed
// BrainkitError round-trips through the bus as an ok=false envelope,
// every success path emits ok=true, unknown codes fall back to BusError.
package envelope

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run executes the envelope domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("envelope", func(t *testing.T) {
		// typed_errors.go — every typed error round-trips
		t.Run("not_found_round_trip", func(t *testing.T) { testNotFoundRoundTrip(t, env) })
		t.Run("validation_error_round_trip", func(t *testing.T) { testValidationErrorRoundTrip(t, env) })
		t.Run("unknown_code_becomes_bus_error", func(t *testing.T) { testUnknownCodeBecomesBusError(t, env) })

		// shape.go — wire shape contract
		t.Run("success_reply_is_envelope", func(t *testing.T) { testSuccessReplyIsEnvelope(t, env) })
		t.Run("error_reply_is_envelope", func(t *testing.T) { testErrorReplyIsEnvelope(t, env) })
		t.Run("envelope_metadata_flag_present", func(t *testing.T) { testEnvelopeMetadataFlagPresent(t, env) })

		// call.go — brainkit.Call surfaces typed errors
		t.Run("call_returns_typed_not_found", func(t *testing.T) { testCallReturnsTypedNotFound(t, env) })
		t.Run("call_returns_typed_validation", func(t *testing.T) { testCallReturnsTypedValidation(t, env) })
		t.Run("call_returns_bus_error_on_unknown_code", func(t *testing.T) { testCallReturnsBusErrorOnUnknownCode(t, env) })
	})
}
