// Package workflows provides the workflow domain test suite.
// All test functions take *suite.TestEnv and are registered via Run().
// The standalone workflows_test.go creates a Full env for the memory fast path.
package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// Run executes all workflow domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("workflows", func(t *testing.T) {
		t.Run("no_module_commands_absent", func(t *testing.T) { testNoModuleCommandsAbsent(t, env) })

		// commands.go — happy path + error paths (from infra/workflow_bus_test.go)
		t.Run("list_empty", func(t *testing.T) { testListEmpty(t, env) })
		t.Run("start_sequential", func(t *testing.T) { testStartSequential(t, env) })
		t.Run("start_parallel", func(t *testing.T) { testStartParallel(t, env) })
		t.Run("list", func(t *testing.T) { testList(t, env) })
		t.Run("suspend_resume", func(t *testing.T) { testSuspendResume(t, env) })
		t.Run("cancel", func(t *testing.T) { testCancel(t, env) })
		t.Run("with_tool_call", func(t *testing.T) { testWithToolCall(t, env) })
		t.Run("not_found", func(t *testing.T) { testNotFound(t, env) })
		t.Run("resume_nonexistent_run", func(t *testing.T) { testResumeNonexistentRun(t, env) })
		t.Run("status_nonexistent_run", func(t *testing.T) { testStatusNonexistentRun(t, env) })
		t.Run("cancel_nonexistent_run", func(t *testing.T) { testCancelNonexistentRun(t, env) })
		t.Run("resume_completed_run", func(t *testing.T) { testResumeCompletedRun(t, env) })
		t.Run("cancel_completed_run", func(t *testing.T) { testCancelCompletedRun(t, env) })
		t.Run("resume_wrong_step", func(t *testing.T) { testResumeWrongStep(t, env) })
		t.Run("restart_nonexistent_run", func(t *testing.T) { testRestartNonexistentRun(t, env) })
		t.Run("restart_completed_run", func(t *testing.T) { testRestartCompletedRun(t, env) })
		t.Run("step_with_error", func(t *testing.T) { testStepWithError(t, env) })

		// storage.go — persistence + storage tests
		t.Run("storage_upgrade", func(t *testing.T) { testStorageUpgrade(t, env) })
		t.Run("status_from_storage", func(t *testing.T) { testStatusFromStorage(t, env) })
		t.Run("runs", func(t *testing.T) { testRuns(t, env) })
		t.Run("start_async_event_shape", func(t *testing.T) { testStartAsyncEventShape(t, env) })
		t.Run("crash_recovery_suspended", func(t *testing.T) { testCrashRecoverySuspended(t, env) })
		t.Run("resume_after_restart", func(t *testing.T) { testResumeAfterRestart(t, env) })
		t.Run("cancel_after_restart", func(t *testing.T) { testCancelAfterRestart(t, env) })
		t.Run("restart_after_restart", func(t *testing.T) { testRestartAfterRestart(t, env) })
		t.Run("runs_after_restart", func(t *testing.T) { testRunsAfterRestart(t, env) })
		t.Run("corrupt_snapshot_fails_cleanly", func(t *testing.T) { testCorruptSnapshotFailsCleanly(t, env) })

		// concurrent.go — concurrency + stress
		t.Run("concurrent_starts", func(t *testing.T) { testConcurrentStarts(t, env) })
		t.Run("multi_workflow_stress", func(t *testing.T) { testMultiWorkflowStress(t, env) })
		t.Run("long_running_integration", func(t *testing.T) { testLongRunningIntegration(t, env) })

		// developer.go — developer scenario tests
		t.Run("tool_call_inside_step", func(t *testing.T) { testToolCallInsideStep(t, env) })
		t.Run("tool_failure_inside_step", func(t *testing.T) { testToolFailureInsideStep(t, env) })
		t.Run("bus_emit_from_step", func(t *testing.T) { testBusEmitFromStep(t, env) })
		t.Run("conditional_branch", func(t *testing.T) { testConditionalBranch(t, env) })
		t.Run("branch_fallback_path", func(t *testing.T) { testBranchFallbackPath(t, env) })
		t.Run("step_state", func(t *testing.T) { testStepState(t, env) })
		t.Run("suspend_with_context_data", func(t *testing.T) { testSuspendWithContextData(t, env) })
	})
}

// wfPublishAndWait publishes a workflow command and waits for the typed response.
// Generic helper replicating publishAndWait from infra/workflow_bus_test.go.
// Returns both the typed response and the raw sdk.Message so callers can
// extract envelope-shaped error messages via suite.ResponseErrorMessage.
func wfPublishAndWait[Req sdk.BrainkitMessage, Resp any](
	t *testing.T, k *brainkit.Kit, msg Req, timeout time.Duration,
) (Resp, sdk.Message) {
	t.Helper()
	result, err := sdk.Publish(k, context.Background(), msg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var resp Resp
	var respMsg sdk.Message
	unsub, err := sdk.SubscribeTo[Resp](k, ctx, result.ReplyTo, func(r Resp, m sdk.Message) {
		resp = r
		respMsg = m
		cancel()
	})
	require.NoError(t, err)
	defer unsub()
	<-ctx.Done()
	return resp, respMsg
}

// wfDeploy deploys a .ts file that registers a workflow.
func wfDeploy(t *testing.T, k *brainkit.Kit, source, code string) {
	t.Helper()
	testutil.Deploy(t, k, source, code)
}
