package deploy

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// Run executes all deploy domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("deploy", func(t *testing.T) {
		// lifecycle.go
		t.Run("list_empty", func(t *testing.T) { testListEmpty(t, env) })
		t.Run("deploy_teardown", func(t *testing.T) { testDeployTeardown(t, env) })
		t.Run("redeploy", func(t *testing.T) { testRedeploy(t, env) })
		t.Run("deploy_invalid_code", func(t *testing.T) { testDeployInvalidCode(t, env) })
		t.Run("deploy_duplicate", func(t *testing.T) { testDeployDuplicate(t, env) })
		t.Run("concurrent_deploy_same_source", func(t *testing.T) { testConcurrentDeploySameSource(t, env) })

		// edge_cases.go — deploy adversarial tests
		t.Run("ts_imports_stripped", func(t *testing.T) { testTSImportsStripped(t, env) })
		t.Run("multiple_deployments_coexist", func(t *testing.T) { testMultipleDeploymentsCoexist(t, env) })
		t.Run("redeploy_preserves_other", func(t *testing.T) { testRedeployPreservesOtherDeployments(t, env) })
		t.Run("long_source_name", func(t *testing.T) { testLongSourceName(t, env) })
		t.Run("unicode_source_name", func(t *testing.T) { testUnicodeSourceName(t, env) })
		t.Run("js_not_ts", func(t *testing.T) { testJSNotTS(t, env) })
		t.Run("empty_code", func(t *testing.T) { testEmptyCode(t, env) })
		t.Run("code_with_only_comments", func(t *testing.T) { testCodeWithOnlyComments(t, env) })
		t.Run("async_init", func(t *testing.T) { testAsyncInit(t, env) })
		t.Run("tool_with_complex_schema", func(t *testing.T) { testToolWithComplexSchema(t, env) })
		t.Run("multiple_tools_one_deployment", func(t *testing.T) { testMultipleToolsOneDeployment(t, env) })
		t.Run("agent_registration", func(t *testing.T) { testAgentRegistration(t, env) })
		t.Run("workflow_registration", func(t *testing.T) { testWorkflowRegistration(t, env) })
		t.Run("memory_registration", func(t *testing.T) { testMemoryRegistration(t, env) })

		// input_abuse.go — deploy input abuse tests
		t.Run("deploy_empty_source", func(t *testing.T) { testDeployEmptySource(t, env) })
		t.Run("deploy_empty_code_adv", func(t *testing.T) { testDeployEmptyCode(t, env) })
		t.Run("deploy_huge_code", func(t *testing.T) { testDeployHugeCode(t, env) })
		t.Run("deploy_source_path_traversal", func(t *testing.T) { testDeploySourcePathTraversal(t, env) })
		t.Run("deploy_then_immediate_teardown", func(t *testing.T) { testDeployThenImmediateTeardown(t, env) })
		t.Run("deploy_duplicate_source", func(t *testing.T) { testDeployDuplicateSource(t, env) })
		t.Run("deploy_invalid_ts_syntax", func(t *testing.T) { testDeployInvalidTSSyntax(t, env) })
		t.Run("deploy_null_bytes_in_source", func(t *testing.T) { testDeployNullBytesInSourceName(t, env) })
		t.Run("deploy_throws_during_init", func(t *testing.T) { testDeployThrowsDuringInit(t, env) })
		t.Run("deploy_partial_cleanup", func(t *testing.T) { testDeployPartialCleanup(t, env) })
		t.Run("deploy_redeploy_different_tools", func(t *testing.T) { testDeployRedeployDifferentTools(t, env) })

		// state_corruption.go — deploy state corruption tests
		t.Run("teardown_during_handler_execution", func(t *testing.T) { testTeardownDuringHandlerExecution(t, env) })
		t.Run("redeploy_while_handlers_active", func(t *testing.T) { testRedeployWhileHandlersActive(t, env) })
		t.Run("deploy_removes_other_deployment", func(t *testing.T) { testDeployRemovesOtherDeployment(t, env) })

		// e2e.go — deploy lifecycle e2e
		t.Run("deploy_lifecycle", func(t *testing.T) { testDeployLifecycle(t, env) })

		// surface.go — TS surface deploy tests
		t.Run("ts_namespace_isolation", func(t *testing.T) { testTSNamespaceIsolation(t, env) })
		t.Run("ts_module_imports", func(t *testing.T) { testTSModuleImports(t, env) })
		t.Run("ts_file_extension_handling", func(t *testing.T) { testTSFileExtensionHandling(t, env) })
	})
}
