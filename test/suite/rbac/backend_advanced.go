package rbac

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// testRBACEnforcementOnTransport — RBAC publish deny works on the transport.
// Ported from adversarial/rbac_backend_test.go:TestRBACBackend_EnforcementOnEveryBackend.
func testRBACEnforcementOnTransport(t *testing.T, env *suite.TestEnv) {
	k := newRBACKernel(t, "observer")

	result := bridgeDeployAndCheck(t, k, "observer", `
		var caught = "ALLOWED";
		try { bus.publish("forbidden.topic", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result, "RBAC should enforce on transport")
}

// testRBACToolCallOnTransport — service role can call tools on the transport.
// Ported from adversarial/rbac_backend_test.go:TestRBACBackend_ToolCallOnEveryBackend.
func testRBACToolCallOnTransport(t *testing.T, env *suite.TestEnv) {
	k := newRBACKernel(t, "service")

	result := bridgeDeployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { await tools.call("echo", {message: "transport-test"}); }
		catch(e) { caught = "DENIED:" + (e.message || ""); }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result, "service should call tools on transport")
}
