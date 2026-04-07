// Package rbac provides the RBAC domain test suite.
// All test functions take *suite.TestEnv and are registered via Run().
// The standalone rbac_test.go creates an RBAC-enabled env for the memory fast path.
// Campaigns call Run() with transport-specific envs.
package rbac

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// Run executes all RBAC domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("rbac", func(t *testing.T) {
		// enforcement.go — kernel-level RBAC enforcement (from infra/rbac_test.go)
		t.Run("restricted_cannot_publish_forbidden", func(t *testing.T) { testRestrictedCannotPublishForbidden(t, env) })
		t.Run("restricted_cannot_register_tools", func(t *testing.T) { testRestrictedCannotRegisterTools(t, env) })
		t.Run("own_mailbox_always_allowed", func(t *testing.T) { testOwnMailboxAlwaysAllowed(t, env) })
		t.Run("admin_can_do_everything", func(t *testing.T) { testAdminCanDoEverything(t, env) })
		t.Run("assign_revoke_via_bus", func(t *testing.T) { testAssignRevokeViaBus(t, env) })
		t.Run("permission_denied_event_emitted", func(t *testing.T) { testPermissionDeniedEventEmitted(t, env) })
		t.Run("with_role_on_deploy", func(t *testing.T) { testWithRoleOnDeploy(t, env) })
		t.Run("role_persistence_across_restart", func(t *testing.T) { testRolePersistenceAcrossRestart(t, env) })
		t.Run("secret_bridge_enforcement", func(t *testing.T) { testSecretBridgeEnforcement(t, env) })
		t.Run("gateway_route_enforcement", func(t *testing.T) { testGatewayRouteEnforcement(t, env) })
		t.Run("command_matrix", func(t *testing.T) { testCommandMatrix(t, env) })
		t.Run("multi_deployment_isolation", func(t *testing.T) { testMultiDeploymentIsolation(t, env) })

		// bridge.go — bridge-level enforcement via deploy+eval (from adversarial/rbac_enforcement_test.go)
		t.Run("bridge_service_can_publish_incoming", func(t *testing.T) { testBridgeServiceCanPublishIncoming(t, env) })
		t.Run("bridge_service_cannot_publish_random", func(t *testing.T) { testBridgeServiceCannotPublishRandom(t, env) })
		t.Run("bridge_service_can_emit_events", func(t *testing.T) { testBridgeServiceCanEmitEvents(t, env) })
		t.Run("bridge_service_cannot_emit_gateway", func(t *testing.T) { testBridgeServiceCannotEmitGateway(t, env) })
		t.Run("bridge_service_can_register_tool", func(t *testing.T) { testBridgeServiceCanRegisterTool(t, env) })
		t.Run("bridge_service_cannot_register_agent", func(t *testing.T) { testBridgeServiceCannotRegisterAgent(t, env) })
		t.Run("bridge_gateway_can_publish_gateway", func(t *testing.T) { testBridgeGatewayCanPublishGateway(t, env) })
		t.Run("bridge_gateway_cannot_publish_events", func(t *testing.T) { testBridgeGatewayCannotPublishEvents(t, env) })
		t.Run("bridge_gateway_can_emit_gateway", func(t *testing.T) { testBridgeGatewayCanEmitGateway(t, env) })
		t.Run("bridge_observer_cannot_publish", func(t *testing.T) { testBridgeObserverCannotPublish(t, env) })
		t.Run("bridge_observer_cannot_emit", func(t *testing.T) { testBridgeObserverCannotEmit(t, env) })
		t.Run("bridge_observer_can_subscribe", func(t *testing.T) { testBridgeObserverCanSubscribe(t, env) })
		t.Run("bridge_admin_can_do_everything", func(t *testing.T) { testBridgeAdminCanDoEverything(t, env) })
		t.Run("bridge_own_mailbox_always_allowed", func(t *testing.T) { testBridgeOwnMailboxAlwaysAllowed(t, env) })

		// matrix.go — RBAC permission matrix (from adversarial/rbac_matrix_test.go)
		t.Run("matrix_command_permissions", func(t *testing.T) { testMatrixCommandPermissions(t, env) })
		t.Run("matrix_bus_publish", func(t *testing.T) { testMatrixBusPublish(t, env) })
		t.Run("matrix_bus_subscribe", func(t *testing.T) { testMatrixBusSubscribe(t, env) })
		t.Run("matrix_bus_emit", func(t *testing.T) { testMatrixBusEmit(t, env) })
		t.Run("matrix_registration", func(t *testing.T) { testMatrixRegistration(t, env) })
		t.Run("matrix_own_mailbox", func(t *testing.T) { testMatrixOwnMailbox(t, env) })
		t.Run("matrix_integration_observer_denied_publish", func(t *testing.T) { testMatrixIntegrationObserverDeniedPublish(t, env) })
		t.Run("matrix_integration_service_allowed_tool_call", func(t *testing.T) { testMatrixIntegrationServiceAllowedToolCall(t, env) })
		t.Run("matrix_integration_gateway_denied_everything", func(t *testing.T) { testMatrixIntegrationGatewayDeniedEverything(t, env) })

		// enforcement.go — RBAC denied from TS (from adversarial/bus_command_matrix_test.go)
		t.Run("rbac_denied_from_ts", func(t *testing.T) { testRBACDeniedFromTS(t, env) })

		// input_abuse — RBAC input abuse (from adversarial/input_abuse_test.go)
		t.Run("input_abuse_empty_source", func(t *testing.T) { testInputAbuseRBACEmptySource(t, env) })
		t.Run("input_abuse_nonexistent_role", func(t *testing.T) { testInputAbuseRBACNonexistentRole(t, env) })

		// backend_advanced.go — ported from adversarial/rbac_backend_test.go
		t.Run("rbac_enforcement_on_transport", func(t *testing.T) { testRBACEnforcementOnTransport(t, env) })
		t.Run("rbac_tool_call_on_transport", func(t *testing.T) { testRBACToolCallOnTransport(t, env) })
	})
}

// newRestrictedKernel creates a kernel with a custom "restricted" role for RBAC enforcement tests.
// Replicates startKernelWithRBAC from infra/rbac_test.go.
func newRestrictedKernel(t *testing.T) *brainkit.Kernel {
	t.Helper()
	storePath := t.TempDir() + "/rbac-test.db"
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Store: store,
		Roles: map[string]rbac.Role{
			"restricted": {
				Bus: rbac.BusPermissions{
					Publish:   rbac.TopicFilter{Allow: []string{"events.*"}},
					Subscribe: rbac.TopicFilter{Allow: []string{"*.reply.*"}},
					Emit:      rbac.TopicFilter{Allow: []string{"events.*"}},
				},
				Commands:     rbac.CommandPermissions{Allow: []string{"tools.list", "tools.call"}},
				Registration: rbac.RegistrationPermissions{Tools: false, Agents: false},
			},
		},
		DefaultRole: "restricted",
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return k
}

// newRBACKernel creates a kernel with all 4 standard RBAC roles and a specified default role.
// Replicates rbacKernel from adversarial/rbac_enforcement_test.go.
func newRBACKernel(t *testing.T, defaultRole string) *brainkit.Kernel {
	t.Helper()
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":    rbac.RoleAdmin,
			"service":  rbac.RoleService,
			"gateway":  rbac.RoleGateway,
			"observer": rbac.RoleObserver,
		},
		DefaultRole: defaultRole,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
		},
	})
	require.NoError(t, err)

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", tools.TypedTool[echoIn]{
		Description: "echoes",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})

	t.Cleanup(func() { k.Close() })
	return k
}

// bridgeDeployAndCheck deploys TS code with a role and reads the output() result.
// Replicates deployAndCheck from adversarial/rbac_enforcement_test.go.
func bridgeDeployAndCheck(t *testing.T, k *brainkit.Kernel, role, tsCode string) string {
	t.Helper()
	ctx := context.Background()
	src := "rbac-enforce-" + role + ".ts"

	_, err := k.Deploy(ctx, src, tsCode, brainkit.WithRole(role))
	if err != nil {
		return "DEPLOY_FAILED:" + err.Error()
	}
	defer k.Teardown(ctx, src)

	result, err := k.EvalTS(ctx, "__rbac_enforce_result.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || "");
	`)
	if err != nil {
		return "EVAL_FAILED"
	}
	return result
}

