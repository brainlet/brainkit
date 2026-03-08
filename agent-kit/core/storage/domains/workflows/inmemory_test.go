// Ported from: packages/core/src/storage/domains/workflows/inmemory.ts (test coverage)
//
// There is no dedicated TS test file for the workflows storage domain in the
// upstream mastra repository. Tests are derived from the WorkflowsStorage
// interface contract and the InMemory implementation behavior observed in:
//   - packages/core/src/storage/domains/workflows/inmemory.ts
//   - packages/core/src/storage/domains/workflows/base.ts
//
// The TS InMemory implementation uses a shared InMemoryDB Map<string, StorageWorkflowRun>
// keyed by `${workflowName}-${runId}`. The Go port mirrors this structure.
package workflows

import (
	"context"
	"testing"
	"time"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createTestSnapshot creates a minimal valid workflow snapshot suitable for
// testing. The snapshot must include a "context" key to satisfy the
// ensureSnapshot guard in UpdateWorkflowResults / UpdateWorkflowState.
func createTestSnapshot() WorkflowRunState {
	return map[string]any{
		"context":             map[string]any{},
		"activePaths":         []any{},
		"activeStepsPath":     map[string]any{},
		"timestamp":           time.Now().UnixMilli(),
		"suspendedPaths":      map[string]any{},
		"resumeLabels":        map[string]any{},
		"serializedStepGraph": []any{},
		"value":               map[string]any{},
		"waitingPaths":        map[string]any{},
		"status":              "pending",
		"runId":               "run-1",
	}
}

// persistHelper is a convenience that calls PersistWorkflowSnapshot with
// defaults for workflowName, runID, and snapshot, allowing selective overrides.
func persistHelper(
	t *testing.T,
	s *InMemoryWorkflowsStorage,
	ctx context.Context,
	workflowName, runID string,
	snapshot WorkflowRunState,
) {
	t.Helper()
	if snapshot == nil {
		snapshot = createTestSnapshot()
		snapshot["runId"] = runID
	}
	err := s.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
		WorkflowName: workflowName,
		RunID:        runID,
		Snapshot:     snapshot,
	})
	if err != nil {
		t.Fatalf("PersistWorkflowSnapshot returned error: %v", err)
	}
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int { return &i }

// timePtr returns a pointer to a time.Time.
func timePtr(t time.Time) *time.Time { return &t }

// ---------------------------------------------------------------------------
// PersistWorkflowSnapshot
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_PersistWorkflowSnapshot(t *testing.T) {
	ctx := context.Background()

	// TS: persistWorkflowSnapshot stores a workflow run with snapshot, keyed by
	// workflowName + runId. If createdAt/updatedAt are nil, defaults to now.
	t.Run("should persist a workflow run with snapshot", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		snapshot := createTestSnapshot()

		err := storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "test-workflow",
			RunID:        "run-1",
			Snapshot:     snapshot,
		})
		if err != nil {
			t.Fatalf("PersistWorkflowSnapshot returned error: %v", err)
		}

		// Verify it can be loaded back.
		loaded, err := storage.LoadWorkflowSnapshot(ctx, LoadWorkflowSnapshotArgs{
			WorkflowName: "test-workflow",
			RunID:        "run-1",
		})
		if err != nil {
			t.Fatalf("LoadWorkflowSnapshot returned error: %v", err)
		}
		if loaded == nil {
			t.Fatal("expected snapshot to be loaded, got nil")
		}
		if loaded["status"] != "pending" {
			t.Errorf("expected status=pending, got %v", loaded["status"])
		}
	})

	// TS: createdAt/updatedAt default to now if not provided.
	t.Run("should use custom createdAt and updatedAt when provided", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		created := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC)

		err := storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			Snapshot:     createTestSnapshot(),
			CreatedAt:    timePtr(created),
			UpdatedAt:    timePtr(updated),
		})
		if err != nil {
			t.Fatalf("PersistWorkflowSnapshot returned error: %v", err)
		}

		// Verify via GetWorkflowRunByID which exposes createdAt/updatedAt.
		run, err := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID:        "run-1",
			WorkflowName: "wf",
		})
		if err != nil {
			t.Fatalf("GetWorkflowRunByID returned error: %v", err)
		}
		if run == nil {
			t.Fatal("expected run, got nil")
		}
		if !run.CreatedAt.Equal(created) {
			t.Errorf("expected createdAt=%v, got %v", created, run.CreatedAt)
		}
		if !run.UpdatedAt.Equal(updated) {
			t.Errorf("expected updatedAt=%v, got %v", updated, run.UpdatedAt)
		}
	})

	// TS: persisting with the same key overwrites the previous run.
	t.Run("should overwrite existing run with same key", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		snap1 := createTestSnapshot()
		snap1["status"] = "running"
		persistHelper(t, storage, ctx, "wf", "run-1", snap1)

		snap2 := createTestSnapshot()
		snap2["status"] = "completed"
		persistHelper(t, storage, ctx, "wf", "run-1", snap2)

		loaded, err := storage.LoadWorkflowSnapshot(ctx, LoadWorkflowSnapshotArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
		})
		if err != nil {
			t.Fatalf("LoadWorkflowSnapshot returned error: %v", err)
		}
		if loaded["status"] != "completed" {
			t.Errorf("expected status=completed after overwrite, got %v", loaded["status"])
		}
	})

	// TS: resourceId is optional and stored alongside the run.
	t.Run("should store resourceId when provided", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		err := storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			ResourceID:   "resource-abc",
			Snapshot:     createTestSnapshot(),
		})
		if err != nil {
			t.Fatalf("PersistWorkflowSnapshot returned error: %v", err)
		}

		run, err := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID:        "run-1",
			WorkflowName: "wf",
		})
		if err != nil {
			t.Fatalf("GetWorkflowRunByID returned error: %v", err)
		}
		if run == nil {
			t.Fatal("expected run, got nil")
		}
		if run.ResourceID != "resource-abc" {
			t.Errorf("expected resourceId=resource-abc, got %s", run.ResourceID)
		}
	})
}

// ---------------------------------------------------------------------------
// LoadWorkflowSnapshot
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_LoadWorkflowSnapshot(t *testing.T) {
	ctx := context.Background()

	// TS: loadWorkflowSnapshot returns null if the run does not exist.
	t.Run("should return nil for non-existent run", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		loaded, err := storage.LoadWorkflowSnapshot(ctx, LoadWorkflowSnapshotArgs{
			WorkflowName: "non-existent",
			RunID:        "non-existent",
		})
		if err != nil {
			t.Fatalf("LoadWorkflowSnapshot returned error: %v", err)
		}
		if loaded != nil {
			t.Errorf("expected nil for non-existent run, got %v", loaded)
		}
	})

	// TS: loadWorkflowSnapshot returns a deep copy to prevent mutation.
	t.Run("should return a deep copy preventing mutation of stored snapshot", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		loaded1, err := storage.LoadWorkflowSnapshot(ctx, LoadWorkflowSnapshotArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
		})
		if err != nil {
			t.Fatalf("LoadWorkflowSnapshot returned error: %v", err)
		}

		// Mutate the returned snapshot.
		loaded1["status"] = "mutated"

		// Load again — should still have the original value.
		loaded2, err := storage.LoadWorkflowSnapshot(ctx, LoadWorkflowSnapshotArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
		})
		if err != nil {
			t.Fatalf("LoadWorkflowSnapshot returned error: %v", err)
		}
		if loaded2["status"] == "mutated" {
			t.Error("loaded snapshot was mutated — deep copy is not working")
		}
		if loaded2["status"] != "pending" {
			t.Errorf("expected status=pending, got %v", loaded2["status"])
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateWorkflowResults
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_UpdateWorkflowResults(t *testing.T) {
	ctx := context.Background()

	// TS: updateWorkflowResults returns {} when the run does not exist.
	t.Run("should return empty map for non-existent run", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		result, err := storage.UpdateWorkflowResults(ctx, UpdateWorkflowResultsArgs{
			WorkflowName: "no-such-wf",
			RunID:        "no-such-run",
			StepID:       "step-1",
			Result:       StepResult{"output": "hello"},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowResults returned error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	// TS: updateWorkflowResults stores step results in snapshot.context[stepId].
	t.Run("should store step result in snapshot context", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		stepResult := StepResult{"output": "hello", "status": "success"}
		result, err := storage.UpdateWorkflowResults(ctx, UpdateWorkflowResultsArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			StepID:       "step-1",
			Result:       stepResult,
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowResults returned error: %v", err)
		}
		if result["step-1"] == nil {
			t.Fatal("expected step-1 in result context")
		}
		step1 := result["step-1"]
		if step1["output"] != "hello" {
			t.Errorf("expected output=hello, got %v", step1["output"])
		}
	})

	// TS: Multiple step results accumulate in snapshot.context.
	t.Run("should accumulate multiple step results", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		_, err := storage.UpdateWorkflowResults(ctx, UpdateWorkflowResultsArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			StepID:       "step-1",
			Result:       StepResult{"output": "first"},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowResults (step-1) returned error: %v", err)
		}

		result, err := storage.UpdateWorkflowResults(ctx, UpdateWorkflowResultsArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			StepID:       "step-2",
			Result:       StepResult{"output": "second"},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowResults (step-2) returned error: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 step results, got %d", len(result))
		}
		if result["step-1"]["output"] != "first" {
			t.Errorf("expected step-1 output=first, got %v", result["step-1"]["output"])
		}
		if result["step-2"]["output"] != "second" {
			t.Errorf("expected step-2 output=second, got %v", result["step-2"]["output"])
		}
	})

	// TS: requestContext is merged into snapshot.requestContext.
	t.Run("should merge requestContext into snapshot", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		_, err := storage.UpdateWorkflowResults(ctx, UpdateWorkflowResultsArgs{
			WorkflowName:   "wf",
			RunID:          "run-1",
			StepID:         "step-1",
			Result:         StepResult{"output": "ok"},
			RequestContext: map[string]any{"key1": "val1"},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowResults returned error: %v", err)
		}

		// Persist a second result with additional request context.
		_, err = storage.UpdateWorkflowResults(ctx, UpdateWorkflowResultsArgs{
			WorkflowName:   "wf",
			RunID:          "run-1",
			StepID:         "step-2",
			Result:         StepResult{"output": "ok2"},
			RequestContext: map[string]any{"key2": "val2"},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowResults returned error: %v", err)
		}

		// Verify request context was merged in the snapshot.
		loaded, err := storage.LoadWorkflowSnapshot(ctx, LoadWorkflowSnapshotArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
		})
		if err != nil {
			t.Fatalf("LoadWorkflowSnapshot returned error: %v", err)
		}
		rc, ok := loaded["requestContext"].(map[string]any)
		if !ok {
			t.Fatalf("expected requestContext to be map[string]any, got %T", loaded["requestContext"])
		}
		if rc["key1"] != "val1" {
			t.Errorf("expected requestContext.key1=val1, got %v", rc["key1"])
		}
		if rc["key2"] != "val2" {
			t.Errorf("expected requestContext.key2=val2, got %v", rc["key2"])
		}
	})

	// TS: updateWorkflowResults returns a deep copy of snapshot.context.
	t.Run("should return a deep copy of context", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		result, err := storage.UpdateWorkflowResults(ctx, UpdateWorkflowResultsArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			StepID:       "step-1",
			Result:       StepResult{"output": "original"},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowResults returned error: %v", err)
		}

		// Mutate the returned map.
		result["step-1"]["output"] = "mutated"

		// Load snapshot and verify internal state is unchanged.
		loaded, err := storage.LoadWorkflowSnapshot(ctx, LoadWorkflowSnapshotArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
		})
		if err != nil {
			t.Fatalf("LoadWorkflowSnapshot returned error: %v", err)
		}
		ctxMap, _ := loaded["context"].(map[string]any)
		step1, _ := ctxMap["step-1"].(map[string]any)
		if step1["output"] == "mutated" {
			t.Error("returned context was not a deep copy — mutation leaked to stored snapshot")
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateWorkflowState
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_UpdateWorkflowState(t *testing.T) {
	ctx := context.Background()

	// TS: updateWorkflowState returns undefined when the run does not exist.
	t.Run("should return nil for non-existent run", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		result, err := storage.UpdateWorkflowState(ctx, UpdateWorkflowStateArgs{
			WorkflowName: "no-such-wf",
			RunID:        "no-such-run",
			Opts: UpdateWorkflowStateOptions{
				Status: domains.WorkflowRunStatusRunning,
			},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowState returned error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existent run, got %v", result)
		}
	})

	// TS: updateWorkflowState merges opts into snapshot. Status is always set.
	t.Run("should update status in snapshot", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		result, err := storage.UpdateWorkflowState(ctx, UpdateWorkflowStateArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			Opts: UpdateWorkflowStateOptions{
				Status: domains.WorkflowRunStatusRunning,
			},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowState returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result["status"] != string(domains.WorkflowRunStatusRunning) {
			t.Errorf("expected status=running, got %v", result["status"])
		}
	})

	// TS: status transitions — pending → running → completed.
	t.Run("should handle state transitions pending → running → completed", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		// Transition to running.
		result, err := storage.UpdateWorkflowState(ctx, UpdateWorkflowStateArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			Opts: UpdateWorkflowStateOptions{
				Status: domains.WorkflowRunStatusRunning,
			},
		})
		if err != nil {
			t.Fatalf("transition to running: %v", err)
		}
		if result["status"] != string(domains.WorkflowRunStatusRunning) {
			t.Errorf("expected status=running, got %v", result["status"])
		}

		// Transition to completed with a result.
		finalResult := map[string]any{"answer": 42}
		result, err = storage.UpdateWorkflowState(ctx, UpdateWorkflowStateArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			Opts: UpdateWorkflowStateOptions{
				Status: domains.WorkflowRunStatusCompleted,
				Result: finalResult,
			},
		})
		if err != nil {
			t.Fatalf("transition to completed: %v", err)
		}
		if result["status"] != string(domains.WorkflowRunStatusCompleted) {
			t.Errorf("expected status=completed, got %v", result["status"])
		}
		if result["result"] == nil {
			t.Fatal("expected result to be set")
		}
	})

	// TS: error field is set when status transitions to failed.
	t.Run("should set error when transitioning to failed", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		errRef := &domains.SerializedErrorRef{Message: "something went wrong"}
		result, err := storage.UpdateWorkflowState(ctx, UpdateWorkflowStateArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			Opts: UpdateWorkflowStateOptions{
				Status: domains.WorkflowRunStatusFailed,
				Error:  errRef,
			},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowState returned error: %v", err)
		}
		if result["status"] != string(domains.WorkflowRunStatusFailed) {
			t.Errorf("expected status=failed, got %v", result["status"])
		}
		if result["error"] == nil {
			t.Fatal("expected error to be set in snapshot")
		}
	})

	// TS: suspendedPaths / waitingPaths / resumeLabels are optional and merged
	// into snapshot when provided.
	t.Run("should set suspendedPaths and resumeLabels", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		suspendedPaths := map[string][]int{
			"path.step-a": {0, 1},
		}
		resumeLabels := map[string]UpdateWorkflowResumeLabel{
			"label-1": {StepID: "step-a"},
		}

		result, err := storage.UpdateWorkflowState(ctx, UpdateWorkflowStateArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			Opts: UpdateWorkflowStateOptions{
				Status:         domains.WorkflowRunStatusSuspended,
				SuspendedPaths: suspendedPaths,
				ResumeLabels:   resumeLabels,
			},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowState returned error: %v", err)
		}
		if result["status"] != string(domains.WorkflowRunStatusSuspended) {
			t.Errorf("expected status=suspended, got %v", result["status"])
		}
		if result["suspendedPaths"] == nil {
			t.Error("expected suspendedPaths to be set")
		}
		if result["resumeLabels"] == nil {
			t.Error("expected resumeLabels to be set")
		}
	})

	t.Run("should set waitingPaths", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		waitingPaths := map[string][]int{
			"path.step-b": {2},
		}

		result, err := storage.UpdateWorkflowState(ctx, UpdateWorkflowStateArgs{
			WorkflowName: "wf",
			RunID:        "run-1",
			Opts: UpdateWorkflowStateOptions{
				Status:       domains.WorkflowRunStatusWaiting,
				WaitingPaths: waitingPaths,
			},
		})
		if err != nil {
			t.Fatalf("UpdateWorkflowState returned error: %v", err)
		}
		if result["status"] != string(domains.WorkflowRunStatusWaiting) {
			t.Errorf("expected status=waiting, got %v", result["status"])
		}
		if result["waitingPaths"] == nil {
			t.Error("expected waitingPaths to be set")
		}
	})
}

// ---------------------------------------------------------------------------
// GetWorkflowRunByID
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_GetWorkflowRunByID(t *testing.T) {
	ctx := context.Background()

	// TS: getWorkflowRunById returns null when the run does not exist.
	t.Run("should return nil for non-existent run", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		run, err := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID:        "non-existent",
			WorkflowName: "wf",
		})
		if err != nil {
			t.Fatalf("GetWorkflowRunByID returned error: %v", err)
		}
		if run != nil {
			t.Errorf("expected nil for non-existent run, got %+v", run)
		}
	})

	// TS: getWorkflowRunById filters by both runId and workflowName.
	t.Run("should return run when found by runId and workflowName", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf-alpha", "run-1", nil)

		run, err := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID:        "run-1",
			WorkflowName: "wf-alpha",
		})
		if err != nil {
			t.Fatalf("GetWorkflowRunByID returned error: %v", err)
		}
		if run == nil {
			t.Fatal("expected run, got nil")
		}
		if run.RunID != "run-1" {
			t.Errorf("expected runId=run-1, got %s", run.RunID)
		}
		if run.WorkflowName != "wf-alpha" {
			t.Errorf("expected workflowName=wf-alpha, got %s", run.WorkflowName)
		}
	})

	// TS: When workflowName is omitted, getWorkflowRunById matches by runId
	// alone across all workflow names.
	t.Run("should find run by runId alone when workflowName is empty", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf-beta", "run-1", nil)

		run, err := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID: "run-1",
			// WorkflowName intentionally omitted.
		})
		if err != nil {
			t.Fatalf("GetWorkflowRunByID returned error: %v", err)
		}
		if run == nil {
			t.Fatal("expected run to be found by runId alone, got nil")
		}
		if run.RunID != "run-1" {
			t.Errorf("expected runId=run-1, got %s", run.RunID)
		}
	})

	// TS: getWorkflowRunById returns a deep copy with snapshot.
	t.Run("should return a deep copy of the run", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		run1, err := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID:        "run-1",
			WorkflowName: "wf",
		})
		if err != nil {
			t.Fatalf("GetWorkflowRunByID returned error: %v", err)
		}

		// Mutate the returned snapshot.
		snap, ok := run1.Snapshot.(map[string]any)
		if ok {
			snap["status"] = "mutated"
		}

		// Fetch again to verify independence.
		run2, err := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID:        "run-1",
			WorkflowName: "wf",
		})
		if err != nil {
			t.Fatalf("GetWorkflowRunByID returned error: %v", err)
		}
		snap2, _ := run2.Snapshot.(map[string]any)
		if snap2["status"] == "mutated" {
			t.Error("returned run was not a deep copy — mutation leaked to stored data")
		}
	})
}

// ---------------------------------------------------------------------------
// ListWorkflowRuns
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_ListWorkflowRuns(t *testing.T) {
	ctx := context.Background()

	// TS: listWorkflowRuns returns all runs when no filters are applied.
	t.Run("should list all workflow runs", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf-A", "run-1", nil)
		persistHelper(t, storage, ctx, "wf-A", "run-2", nil)
		persistHelper(t, storage, ctx, "wf-B", "run-3", nil)

		result, err := storage.ListWorkflowRuns(ctx, nil)
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 3 {
			t.Errorf("expected total=3, got %d", result.Total)
		}
		if len(result.Runs) != 3 {
			t.Errorf("expected 3 runs, got %d", len(result.Runs))
		}
	})

	// TS: filter by workflowName.
	t.Run("should filter by workflowName", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf-A", "run-1", nil)
		persistHelper(t, storage, ctx, "wf-A", "run-2", nil)
		persistHelper(t, storage, ctx, "wf-B", "run-3", nil)

		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			WorkflowName: "wf-A",
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("expected total=2 for wf-A, got %d", result.Total)
		}
		for _, run := range result.Runs {
			if run.WorkflowName != "wf-A" {
				t.Errorf("expected workflowName=wf-A, got %s", run.WorkflowName)
			}
		}
	})

	// TS: filter by resourceId.
	t.Run("should filter by resourceId", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-1", ResourceID: "res-1", Snapshot: createTestSnapshot(),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-2", ResourceID: "res-2", Snapshot: createTestSnapshot(),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-3", ResourceID: "res-1", Snapshot: createTestSnapshot(),
		})

		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			ResourceID: "res-1",
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("expected total=2 for res-1, got %d", result.Total)
		}
	})

	// TS: filter by status — filters based on snapshot.status.
	t.Run("should filter by status", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		snap1 := createTestSnapshot()
		snap1["status"] = "completed"
		persistHelper(t, storage, ctx, "wf", "run-1", snap1)

		snap2 := createTestSnapshot()
		snap2["status"] = "running"
		persistHelper(t, storage, ctx, "wf", "run-2", snap2)

		snap3 := createTestSnapshot()
		snap3["status"] = "completed"
		persistHelper(t, storage, ctx, "wf", "run-3", snap3)

		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			Status: "completed",
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("expected total=2 completed runs, got %d", result.Total)
		}
	})

	// TS: filter by date range (fromDate + toDate).
	t.Run("should filter by date range", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
		t3 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)

		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-1", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t1),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-2", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t2),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-3", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t3),
		})

		fromDate := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
		toDate := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)

		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			FromDate: &fromDate,
			ToDate:   &toDate,
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected total=1 in date range, got %d", result.Total)
		}
		if len(result.Runs) > 0 && result.Runs[0].RunID != "run-2" {
			t.Errorf("expected run-2, got %s", result.Runs[0].RunID)
		}
	})

	// TS: filter by fromDate only.
	t.Run("should filter by fromDate only", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)

		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-1", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t1),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-2", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t2),
		})

		fromDate := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			FromDate: &fromDate,
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected total=1, got %d", result.Total)
		}
	})

	// TS: filter by toDate only.
	t.Run("should filter by toDate only", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)

		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-1", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t1),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-2", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t2),
		})

		toDate := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			ToDate: &toDate,
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected total=1, got %d", result.Total)
		}
	})

	// TS: runs are sorted by createdAt DESC.
	t.Run("should sort runs by createdAt descending", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
		t3 := time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC)

		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-1", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t1),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-2", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t2),
		})
		_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
			WorkflowName: "wf", RunID: "run-3", Snapshot: createTestSnapshot(), CreatedAt: timePtr(t3),
		})

		result, err := storage.ListWorkflowRuns(ctx, nil)
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if len(result.Runs) != 3 {
			t.Fatalf("expected 3 runs, got %d", len(result.Runs))
		}
		// Expect: run-2 (Jan 5), run-3 (Jan 3), run-1 (Jan 1)
		if result.Runs[0].RunID != "run-2" {
			t.Errorf("expected first run=run-2 (newest), got %s", result.Runs[0].RunID)
		}
		if result.Runs[1].RunID != "run-3" {
			t.Errorf("expected second run=run-3, got %s", result.Runs[1].RunID)
		}
		if result.Runs[2].RunID != "run-1" {
			t.Errorf("expected third run=run-1 (oldest), got %s", result.Runs[2].RunID)
		}
	})

	// TS: pagination with perPage and page.
	t.Run("should paginate results", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		for i := 0; i < 5; i++ {
			created := time.Date(2024, 1, i+1, 10, 0, 0, 0, time.UTC)
			_ = storage.PersistWorkflowSnapshot(ctx, PersistWorkflowSnapshotArgs{
				WorkflowName: "wf",
				RunID:        "run-" + string(rune('A'+i)),
				Snapshot:     createTestSnapshot(),
				CreatedAt:    timePtr(created),
			})
		}

		// Page 0, perPage 2.
		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			PerPage: intPtr(2),
			Page:    intPtr(0),
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns page 0: %v", err)
		}
		if result.Total != 5 {
			t.Errorf("expected total=5, got %d", result.Total)
		}
		if len(result.Runs) != 2 {
			t.Errorf("expected 2 runs on page 0, got %d", len(result.Runs))
		}

		// Page 1, perPage 2.
		result, err = storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			PerPage: intPtr(2),
			Page:    intPtr(1),
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns page 1: %v", err)
		}
		if len(result.Runs) != 2 {
			t.Errorf("expected 2 runs on page 1, got %d", len(result.Runs))
		}

		// Page 2, perPage 2.
		result, err = storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			PerPage: intPtr(2),
			Page:    intPtr(2),
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns page 2: %v", err)
		}
		if len(result.Runs) != 1 {
			t.Errorf("expected 1 run on page 2 (last page), got %d", len(result.Runs))
		}
	})

	// TS: page beyond available data returns empty list with correct total.
	t.Run("should return empty runs for page beyond data", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		result, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			PerPage: intPtr(10),
			Page:    intPtr(5),
		})
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if len(result.Runs) != 0 {
			t.Errorf("expected 0 runs for page beyond data, got %d", len(result.Runs))
		}
		if result.Total != 1 {
			t.Errorf("expected total=1, got %d", result.Total)
		}
	})

	// TS: page must be >= 0.
	t.Run("should error for negative page", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		_, err := storage.ListWorkflowRuns(ctx, &ListWorkflowRunsInput{
			Page: intPtr(-1),
		})
		if err == nil {
			t.Fatal("expected error for negative page")
		}
	})

	// TS: empty store returns zero total.
	t.Run("should return empty list for empty store", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		result, err := storage.ListWorkflowRuns(ctx, nil)
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected total=0, got %d", result.Total)
		}
		if len(result.Runs) != 0 {
			t.Errorf("expected 0 runs, got %d", len(result.Runs))
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteWorkflowRunByID
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_DeleteWorkflowRunByID(t *testing.T) {
	ctx := context.Background()

	// TS: deleteWorkflowRunById removes the run from storage.
	t.Run("should delete a workflow run", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)

		// Verify it exists.
		run, _ := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID: "run-1", WorkflowName: "wf",
		})
		if run == nil {
			t.Fatal("expected run to exist before deletion")
		}

		err := storage.DeleteWorkflowRunByID(ctx, DeleteWorkflowRunByIDArgs{
			RunID:        "run-1",
			WorkflowName: "wf",
		})
		if err != nil {
			t.Fatalf("DeleteWorkflowRunByID returned error: %v", err)
		}

		// Verify it is gone.
		run, _ = storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID: "run-1", WorkflowName: "wf",
		})
		if run != nil {
			t.Error("expected run to be deleted")
		}
	})

	// TS: deleting a non-existent run is a no-op (no error).
	t.Run("should not error when deleting non-existent run", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()

		err := storage.DeleteWorkflowRunByID(ctx, DeleteWorkflowRunByIDArgs{
			RunID:        "no-such-run",
			WorkflowName: "no-such-wf",
		})
		if err != nil {
			t.Fatalf("expected no error for non-existent delete, got %v", err)
		}
	})

	// TS: deleting one run does not affect other runs.
	t.Run("should not affect other runs when deleting one", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf", "run-1", nil)
		persistHelper(t, storage, ctx, "wf", "run-2", nil)

		err := storage.DeleteWorkflowRunByID(ctx, DeleteWorkflowRunByIDArgs{
			RunID: "run-1", WorkflowName: "wf",
		})
		if err != nil {
			t.Fatalf("DeleteWorkflowRunByID returned error: %v", err)
		}

		// run-2 should still exist.
		run, _ := storage.GetWorkflowRunByID(ctx, GetWorkflowRunByIDArgs{
			RunID: "run-2", WorkflowName: "wf",
		})
		if run == nil {
			t.Error("expected run-2 to still exist after deleting run-1")
		}
	})
}

// ---------------------------------------------------------------------------
// DangerouslyClearAll
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()

	// TS: dangerouslyClearAll removes all data from the store.
	t.Run("should clear all workflow runs", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		persistHelper(t, storage, ctx, "wf-A", "run-1", nil)
		persistHelper(t, storage, ctx, "wf-A", "run-2", nil)
		persistHelper(t, storage, ctx, "wf-B", "run-3", nil)

		err := storage.DangerouslyClearAll(ctx)
		if err != nil {
			t.Fatalf("DangerouslyClearAll returned error: %v", err)
		}

		result, err := storage.ListWorkflowRuns(ctx, nil)
		if err != nil {
			t.Fatalf("ListWorkflowRuns returned error: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected total=0 after clear, got %d", result.Total)
		}
	})
}

// ---------------------------------------------------------------------------
// SupportsConcurrentUpdates
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_SupportsConcurrentUpdates(t *testing.T) {
	// TS: InMemory supports concurrent updates (returns true).
	t.Run("should return true", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		if !storage.SupportsConcurrentUpdates() {
			t.Error("expected SupportsConcurrentUpdates=true for InMemory")
		}
	})
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestInMemoryWorkflowsStorage_Init(t *testing.T) {
	// TS: Init is a no-op for the in-memory store.
	t.Run("should be a no-op", func(t *testing.T) {
		storage := NewInMemoryWorkflowsStorage()
		err := storage.Init(context.Background())
		if err != nil {
			t.Fatalf("Init returned error: %v", err)
		}
	})
}
