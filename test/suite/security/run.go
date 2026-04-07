// Package security provides the security domain test suite.
// Cross-domain security probe tests migrated from adversarial sources.
// All test functions take (t *testing.T, env *suite.TestEnv) — unexported.
// The standalone security_test.go creates a Full env for the memory fast path.
// Campaigns call Run() with different envs for backend combinations.
package security

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// Run executes all security domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("security", func(t *testing.T) {
		// sandbox.go — sandbox escape attacks (10 tests)
		t.Run("sandbox_direct_bridge_access", func(t *testing.T) { testSandboxDirectBridgeAccess(t, env) })
		t.Run("sandbox_hijack_compartment", func(t *testing.T) { testSandboxHijackCompartment(t, env) })
		t.Run("sandbox_registry_manipulation", func(t *testing.T) { testSandboxRegistryManipulation(t, env) })
		t.Run("sandbox_bus_subs_hijack", func(t *testing.T) { testSandboxBusSubsHijack(t, env) })
		t.Run("sandbox_prototype_pollution", func(t *testing.T) { testSandboxPrototypePollution(t, env) })
		t.Run("sandbox_endowment_overwrite", func(t *testing.T) { testSandboxEndowmentOverwrite(t, env) })
		t.Run("sandbox_global_this_access", func(t *testing.T) { testSandboxGlobalThisAccess(t, env) })
		t.Run("sandbox_fs_path_traversal", func(t *testing.T) { testSandboxFSPathTraversal(t, env) })
		t.Run("sandbox_fs_write_escape", func(t *testing.T) { testSandboxFSWriteEscape(t, env) })
		t.Run("sandbox_runtime_modification", func(t *testing.T) { testSandboxRuntimeModification(t, env) })

		// data_leakage.go — data leakage tests (8 tests)
		t.Run("leakage_error_message_content", func(t *testing.T) { testLeakageErrorMessageContent(t, env) })
		t.Run("leakage_shared_global_state", func(t *testing.T) { testLeakageSharedGlobalState(t, env) })
		t.Run("leakage_tool_state_leak", func(t *testing.T) { testLeakageToolStateLeak(t, env) })
		t.Run("leakage_metadata_leak", func(t *testing.T) { testLeakageMetadataLeak(t, env) })
		t.Run("leakage_secret_timing_side_channel", func(t *testing.T) { testLeakageSecretTimingSideChannel(t, env) })
		t.Run("leakage_deployment_reconnaissance", func(t *testing.T) { testLeakageDeploymentReconnaissance(t, env) })
		t.Run("leakage_filesystem_reconnaissance", func(t *testing.T) { testLeakageFilesystemReconnaissance(t, env) })
		t.Run("leakage_provider_reconnaissance", func(t *testing.T) { testLeakageProviderReconnaissance(t, env) })

		// bus_forgery.go — bus forgery tests (12 tests)
		t.Run("forgery_steal_reply_to", func(t *testing.T) { testForgeryStealReplyTo(t, env) })
		t.Run("forgery_inject_fake_reply", func(t *testing.T) { testForgeryInjectFakeReply(t, env) })
		t.Run("forgery_correlation_id_collision", func(t *testing.T) { testForgeryCorrelationIdCollision(t, env) })
		t.Run("forgery_recursive_bus_loop", func(t *testing.T) { testForgeryRecursiveBusLoop(t, env) })
		t.Run("forgery_flood_bus", func(t *testing.T) { testForgeryFloodBus(t, env) })
		t.Run("forgery_subscription_bomb", func(t *testing.T) { testForgerySubscriptionBomb(t, env) })
		t.Run("forgery_schedule_bomb", func(t *testing.T) { testForgeryScheduleBomb(t, env) })
		t.Run("forgery_command_topic_bypass", func(t *testing.T) { testForgeryCommandTopicBypass(t, env) })
		t.Run("forgery_tool_name_collision", func(t *testing.T) { testForgeryToolNameCollision(t, env) })
		t.Run("forgery_metadata_injection", func(t *testing.T) { testForgeryMetadataInjection(t, env) })
		t.Run("forgery_cross_deployment_result", func(t *testing.T) { testForgeryCrossDeploymentResult(t, env) })
		t.Run("forgery_malicious_go_tool", func(t *testing.T) { testForgeryMaliciousGoTool(t, env) })

		// cross_deploy.go — cross-deployment attack tests (10 tests)
		t.Run("xdeploy_teardown_another", func(t *testing.T) { testXDeployTeardownAnother(t, env) })
		t.Run("xdeploy_reply_impersonation", func(t *testing.T) { testXDeployReplyImpersonation(t, env) })
		t.Run("xdeploy_unregister_alien_tool", func(t *testing.T) { testXDeployUnregisterAlienTool(t, env) })
		t.Run("xdeploy_steal_output", func(t *testing.T) { testXDeployStealOutput(t, env) })
		t.Run("xdeploy_mailbox_eavesdrop", func(t *testing.T) { testXDeployMailboxEavesdrop(t, env) })
		t.Run("xdeploy_agent_registration_race", func(t *testing.T) { testXDeployAgentRegistrationRace(t, env) })
		t.Run("xdeploy_create_tool_monkey_patch", func(t *testing.T) { testXDeployCreateToolMonkeyPatch(t, env) })
		t.Run("xdeploy_send_to_crafted", func(t *testing.T) { testXDeploySendToCrafted(t, env) })
		t.Run("xdeploy_self_redeploy", func(t *testing.T) { testXDeploySelfRedeploy(t, env) })
		t.Run("xdeploy_workflow_escalation", func(t *testing.T) { testXDeployWorkflowEscalation(t, env) })

		// internal_exploit.go — internal exploit tests (13 tests)
		t.Run("exploit_current_source_poisoning", func(t *testing.T) { testExploitCurrentSourcePoisoning(t, env) })
		t.Run("exploit_reply_to_redirect", func(t *testing.T) { testExploitReplyToRedirect(t, env) })
		t.Run("exploit_send_to_namespace_confusion", func(t *testing.T) { testExploitSendToNamespaceConfusion(t, env) })
		t.Run("exploit_schedule_fires_command_topic", func(t *testing.T) { testExploitScheduleFiresCommandTopic(t, env) })
		t.Run("exploit_api_key_js_injection", func(t *testing.T) { testExploitAPIKeyJSInjection(t, env) })
		t.Run("exploit_deploy_file_escape", func(t *testing.T) { testExploitDeployFileEscape(t, env) })
		t.Run("exploit_harden_bypass", func(t *testing.T) { testExploitHardenBypass(t, env) })
		t.Run("exploit_deploy_ordering_attack", func(t *testing.T) { testExploitDeployOrderingAttack(t, env) })
		t.Run("exploit_reentrant_source_tracking", func(t *testing.T) { testExploitReentrantSourceTracking(t, env) })
		t.Run("exploit_plugin_state_key_collision", func(t *testing.T) { testExploitPluginStateKeyCollision(t, env) })
		t.Run("exploit_libsql_cache_exhaustion", func(t *testing.T) { testExploitLibSQLCacheExhaustion(t, env) })
		t.Run("exploit_registry_resolve_leak", func(t *testing.T) { testExploitRegistryResolveLeak(t, env) })
		t.Run("exploit_provider_global_leak", func(t *testing.T) { testExploitProviderGlobalLeak(t, env) })

		// rbac_escape.go — RBAC escape tests (9 tests)
		t.Run("rbac_observer_registers_tool_then_calls", func(t *testing.T) { testRBACObserverRegistersToolThenCalls(t, env) })
		t.Run("rbac_gateway_exfiltrates_secrets", func(t *testing.T) { testRBACGatewayExfiltratesSecrets(t, env) })
		t.Run("rbac_observer_hijacks_service_handler", func(t *testing.T) { testRBACObserverHijacksServiceHandler(t, env) })
		t.Run("rbac_service_tries_admin", func(t *testing.T) { testRBACServiceTriesAdmin(t, env) })
		t.Run("rbac_broken_reply_pattern_exploit", func(t *testing.T) { testRBACBrokenReplyPatternExploit(t, env) })
		t.Run("rbac_role_swap_tool_persistence", func(t *testing.T) { testRBACRoleSwapToolPersistence(t, env) })
		t.Run("rbac_schedule_to_command_topic", func(t *testing.T) { testRBACScheduleToCommandTopic(t, env) })
		t.Run("rbac_deploy_inception", func(t *testing.T) { testRBACDeployInception(t, env) })
		t.Run("rbac_caller_id_forgery", func(t *testing.T) { testRBACCallerIDForgery(t, env) })

		// reply_token.go — reply token security tests (7 tests)
		t.Run("token_own_mailbox_gets_token", func(t *testing.T) { testTokenOwnMailboxGetsToken(t, env) })
		t.Run("token_legit_handler_can_reply", func(t *testing.T) { testTokenLegitHandlerCanReply(t, env) })
		t.Run("token_observer_cannot_reply", func(t *testing.T) { testTokenObserverCannotReply(t, env) })
		t.Run("token_streaming_with_token", func(t *testing.T) { testTokenStreamingWithToken(t, env) })
		t.Run("token_no_rbac_no_tokens", func(t *testing.T) { testTokenNoRBACNoTokens(t, env) })
		t.Run("token_audit_event_emitted", func(t *testing.T) { testTokenAuditEventEmitted(t, env) })
		t.Run("token_cross_deployment_scoped", func(t *testing.T) { testTokenCrossDeploymentScoped(t, env) })

		// timing.go — timing attack tests (10 tests)
		t.Run("timing_preemptive_reply_subscribe", func(t *testing.T) { testTimingPreemptiveReplySubscribe(t, env) })
		t.Run("timing_deploy_teardown_race", func(t *testing.T) { testTimingDeployTeardownRace(t, env) })
		t.Run("timing_message_during_restore", func(t *testing.T) { testTimingMessageDuringRestore(t, env) })
		t.Run("timing_concurrent_redeploy", func(t *testing.T) { testTimingConcurrentRedeploy(t, env) })
		t.Run("timing_tool_call_during_deploy", func(t *testing.T) { testTimingToolCallDuringDeploy(t, env) })
		t.Run("timing_schedule_fires_before_handler_ready", func(t *testing.T) { testTimingScheduleFiresBeforeHandlerReady(t, env) })
		t.Run("timing_close_while_tool_call_in_progress", func(t *testing.T) { testTimingCloseWhileToolCallInProgress(t, env) })
		t.Run("timing_role_change_while_handler_running", func(t *testing.T) { testTimingRoleChangeWhileHandlerRunning(t, env) })
		t.Run("timing_schedule_unschedule_race", func(t *testing.T) { testTimingScheduleUnscheduleRace(t, env) })
		t.Run("timing_storage_race_with_deploy", func(t *testing.T) { testTimingStorageRaceWithDeploy(t, env) })

		// secrets.go — secret exfiltration tests (7 tests)
		t.Run("secret_publish_to_bus", func(t *testing.T) { testSecretPublishToBus(t, env) })
		t.Run("secret_observer_reads_secret", func(t *testing.T) { testSecretObserverReadsSecret(t, env) })
		t.Run("secret_env_var_dump", func(t *testing.T) { testSecretEnvVarDump(t, env) })
		t.Run("secret_enumeration", func(t *testing.T) { testSecretEnumeration(t, env) })
		t.Run("secret_audit_event_snooping", func(t *testing.T) { testSecretAuditEventSnooping(t, env) })
		t.Run("secret_rotate_dos", func(t *testing.T) { testSecretRotateDOS(t, env) })
		t.Run("secret_decryption_oracle", func(t *testing.T) { testSecretDecryptionOracle(t, env) })

		// gateway.go — gateway security tests (4 tests)
		t.Run("gateway_header_injection", func(t *testing.T) { testGatewayHeaderInjection(t, env) })
		t.Run("gateway_proto_pollution_via_http", func(t *testing.T) { testGatewayProtoPollutionViaHTTP(t, env) })
		t.Run("gateway_path_traversal_params", func(t *testing.T) { testGatewayPathTraversalParams(t, env) })
		t.Run("gateway_websocket_injection", func(t *testing.T) { testGatewayWebSocketInjection(t, env) })

		// state.go — state corruption security tests (2 tests)
		t.Run("state_nonexistent_role_on_deploy", func(t *testing.T) { testStateNonexistentRoleOnDeploy(t, env) })
		t.Run("state_store_wiped_midlife", func(t *testing.T) { testStateStoreWipedMidlife(t, env) })

		// persistence.go — persistence attack security tests (4 tests)
		t.Run("persist_sql_injection_in_source", func(t *testing.T) { testPersistSQLInjectionInSource(t, env) })
		t.Run("persist_code_mutates_store_during_restore", func(t *testing.T) { testPersistCodeMutatesStoreDuringRestore(t, env) })
		t.Run("persist_evil_plugin_paths", func(t *testing.T) { testPersistEvilPluginPaths(t, env) })
		t.Run("persist_concurrent_store_writes", func(t *testing.T) { testPersistConcurrentStoreWrites(t, env) })

		// libsql_validation.go — LibSQL file: URL blocking (from surface/ts_test.go)
		t.Run("libsql_file_url_blocked", func(t *testing.T) { testLibSQLFileURLBlocked(t, env) })
		t.Run("libsql_http_url_not_blocked", func(t *testing.T) { testLibSQLHttpURLNotBlocked(t, env) })
	})
}

// --- Shared helpers ---

// secKernel creates a kernel with all 4 standard RBAC roles for security escape tests.
// Mirrors rbacAttackKernel from adversarial/rbac_escape_test.go.
func secRBACKernel(t *testing.T) *brainkit.Kernel {
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
		DefaultRole: "observer",
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

// secReplyTokenKernel creates a kernel for reply token tests.
// Mirrors replyTokenKernel from adversarial/reply_token_test.go.
func secReplyTokenKernel(t *testing.T) *brainkit.Kernel {
	t.Helper()
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":    rbac.RoleAdmin,
			"service":  rbac.RoleService,
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "service",
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

// secSendAndReceive publishes a typed message via SDK and waits for the reply.
func secSendAndReceive(t *testing.T, k *brainkit.Kernel, msg messages.BrainkitMessage, timeout time.Duration) (json.RawMessage, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pr, err := sdk.Publish(k, ctx, msg)
	if err != nil {
		t.Logf("publish failed: %v", err)
		return nil, false
	}

	ch := make(chan json.RawMessage, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
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

// secResponseHasError checks if a bus response contains an error field.
func secResponseHasError(payload json.RawMessage) bool {
	var resp struct {
		Error string `json:"error"`
	}
	json.Unmarshal(payload, &resp)
	return resp.Error != ""
}

// secContainsSubstring checks if s contains sub.
func secContainsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// secMin returns the minimum of two ints.
func secMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
