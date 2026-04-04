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
	})
}
