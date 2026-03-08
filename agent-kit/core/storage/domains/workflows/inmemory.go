// Ported from: packages/core/src/storage/domains/workflows/inmemory.ts
package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// Compile-time interface check.
var _ WorkflowsStorage = (*InMemoryWorkflowsStorage)(nil)

// InMemoryWorkflowsStorage is an in-memory implementation of WorkflowsStorage.
type InMemoryWorkflowsStorage struct {
	mu        sync.RWMutex
	workflows map[string]StorageWorkflowRun
}

// NewInMemoryWorkflowsStorage creates a new InMemoryWorkflowsStorage.
func NewInMemoryWorkflowsStorage() *InMemoryWorkflowsStorage {
	return &InMemoryWorkflowsStorage{
		workflows: make(map[string]StorageWorkflowRun),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryWorkflowsStorage) Init(_ context.Context) error {
	return nil
}

// DangerouslyClearAll clears all workflows.
func (s *InMemoryWorkflowsStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.workflows = make(map[string]StorageWorkflowRun)
	return nil
}

// SupportsConcurrentUpdates returns true — in-memory supports concurrent updates.
func (s *InMemoryWorkflowsStorage) SupportsConcurrentUpdates() bool {
	return true
}

func workflowKey(workflowName, runID string) string {
	return workflowName + "-" + runID
}

// UpdateWorkflowResults updates workflow step results.
// This faithfully ports the TS forEach merge logic for concurrent iterations.
func (s *InMemoryWorkflowsStorage) UpdateWorkflowResults(_ context.Context, args UpdateWorkflowResultsArgs) (map[string]StepResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := workflowKey(args.WorkflowName, args.RunID)
	run, ok := s.workflows[key]
	if !ok {
		return map[string]StepResult{}, nil
	}

	snapshot := ensureSnapshot(&run, key)
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found for runId %s", args.RunID)
	}

	snapshotCtx, _ := snapshot["context"].(map[string]any)
	if snapshotCtx == nil {
		return nil, fmt.Errorf("snapshot not found for runId %s", args.RunID)
	}

	// Store the step result.
	snapshotCtx[args.StepID] = args.Result

	// Merge request context.
	if args.RequestContext != nil {
		rc, _ := snapshot["requestContext"].(map[string]any)
		if rc == nil {
			rc = make(map[string]any)
		}
		for k, v := range args.RequestContext {
			rc[k] = v
		}
		snapshot["requestContext"] = rc
	}

	run.Snapshot = snapshot
	s.workflows[key] = run

	// Return a deep copy of the context.
	copied := deepCopyMap(snapshotCtx)
	result := make(map[string]StepResult, len(copied))
	for k, v := range copied {
		if m, ok := v.(map[string]any); ok {
			result[k] = m
		}
	}
	return result, nil
}

// UpdateWorkflowState updates workflow state.
func (s *InMemoryWorkflowsStorage) UpdateWorkflowState(_ context.Context, args UpdateWorkflowStateArgs) (WorkflowRunState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := workflowKey(args.WorkflowName, args.RunID)
	run, ok := s.workflows[key]
	if !ok {
		return nil, nil
	}

	snapshot := ensureSnapshot(&run, key)
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found for runId %s", args.RunID)
	}

	// Merge opts into snapshot.
	// UpdateWorkflowStateOptions fields are applied to the snapshot.
	snapshot["status"] = string(args.Opts.Status)
	if args.Opts.Result != nil {
		snapshot["result"] = args.Opts.Result
	}
	if args.Opts.Error != nil {
		snapshot["error"] = args.Opts.Error
	}
	if args.Opts.SuspendedPaths != nil {
		snapshot["suspendedPaths"] = args.Opts.SuspendedPaths
	}
	if args.Opts.WaitingPaths != nil {
		snapshot["waitingPaths"] = args.Opts.WaitingPaths
	}
	if args.Opts.ResumeLabels != nil {
		snapshot["resumeLabels"] = args.Opts.ResumeLabels
	}

	run.Snapshot = snapshot
	s.workflows[key] = run

	return snapshot, nil
}

// PersistWorkflowSnapshot persists a workflow run snapshot.
func (s *InMemoryWorkflowsStorage) PersistWorkflowSnapshot(_ context.Context, args PersistWorkflowSnapshotArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := workflowKey(args.WorkflowName, args.RunID)
	now := time.Now()

	createdAt := now
	if args.CreatedAt != nil {
		createdAt = *args.CreatedAt
	}
	updatedAt := now
	if args.UpdatedAt != nil {
		updatedAt = *args.UpdatedAt
	}

	s.workflows[key] = StorageWorkflowRun{
		WorkflowName: args.WorkflowName,
		RunID:        args.RunID,
		ResourceID:   args.ResourceID,
		Snapshot:     args.Snapshot,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	return nil
}

// LoadWorkflowSnapshot loads a workflow run snapshot. Returns nil if not found.
func (s *InMemoryWorkflowsStorage) LoadWorkflowSnapshot(_ context.Context, args LoadWorkflowSnapshotArgs) (WorkflowRunState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := workflowKey(args.WorkflowName, args.RunID)
	run, ok := s.workflows[key]
	if !ok {
		return nil, nil
	}

	if run.Snapshot == nil {
		return nil, nil
	}

	// Return a deep copy to prevent mutation.
	copied, ok := deepCopyAny(run.Snapshot).(map[string]any)
	if !ok {
		return nil, nil
	}
	return copied, nil
}

// ListWorkflowRuns lists workflow runs with optional filtering.
func (s *InMemoryWorkflowsStorage) ListWorkflowRuns(_ context.Context, args *ListWorkflowRunsInput) (*WorkflowRuns, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if args != nil && args.Page != nil && *args.Page < 0 {
		return nil, fmt.Errorf("page must be >= 0")
	}

	var runs []StorageWorkflowRun
	for _, run := range s.workflows {
		runs = append(runs, run)
	}

	// Apply filters.
	if args != nil {
		if args.WorkflowName != "" {
			runs = filterRuns(runs, func(r StorageWorkflowRun) bool {
				return r.WorkflowName == args.WorkflowName
			})
		}

		if args.Status != "" {
			runs = filterRuns(runs, func(r StorageWorkflowRun) bool {
				snap, ok := r.Snapshot.(map[string]any)
				if !ok {
					return false
				}
				st, _ := snap["status"].(string)
				return st == args.Status
			})
		}

		if args.FromDate != nil && args.ToDate != nil {
			runs = filterRuns(runs, func(r StorageWorkflowRun) bool {
				return !r.CreatedAt.Before(*args.FromDate) && !r.CreatedAt.After(*args.ToDate)
			})
		} else if args.FromDate != nil {
			runs = filterRuns(runs, func(r StorageWorkflowRun) bool {
				return !r.CreatedAt.Before(*args.FromDate)
			})
		} else if args.ToDate != nil {
			runs = filterRuns(runs, func(r StorageWorkflowRun) bool {
				return !r.CreatedAt.After(*args.ToDate)
			})
		}

		if args.ResourceID != "" {
			runs = filterRuns(runs, func(r StorageWorkflowRun) bool {
				return r.ResourceID == args.ResourceID
			})
		}
	}

	total := len(runs)

	// Sort by createdAt descending.
	sort.Slice(runs, func(i, j int) bool {
		return runs[j].CreatedAt.Before(runs[i].CreatedAt)
	})

	// Apply pagination.
	if args != nil && args.PerPage != nil && args.Page != nil {
		perPage := *args.PerPage
		if perPage <= 0 {
			perPage = math.MaxInt
		}
		page := *args.Page
		start := page * perPage
		end := start + perPage
		if start > len(runs) {
			start = len(runs)
		}
		if end > len(runs) {
			end = len(runs)
		}
		runs = runs[start:end]
	}

	// Convert to public WorkflowRun type.
	parsed := make([]WorkflowRun, 0, len(runs))
	for _, r := range runs {
		parsed = append(parsed, WorkflowRun{
			WorkflowName: r.WorkflowName,
			RunID:        r.RunID,
			ResourceID:   r.ResourceID,
			Snapshot:     deepCopyAny(r.Snapshot),
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		})
	}

	return &WorkflowRuns{Runs: parsed, Total: total}, nil
}

// GetWorkflowRunByID retrieves a workflow run by ID.
func (s *InMemoryWorkflowsStorage) GetWorkflowRunByID(_ context.Context, args GetWorkflowRunByIDArgs) (*WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, run := range s.workflows {
		if run.RunID == args.RunID && (args.WorkflowName == "" || run.WorkflowName == args.WorkflowName) {
			return &WorkflowRun{
				WorkflowName: run.WorkflowName,
				RunID:        run.RunID,
				ResourceID:   run.ResourceID,
				Snapshot:     deepCopyAny(run.Snapshot),
				CreatedAt:    run.CreatedAt,
				UpdatedAt:    run.UpdatedAt,
			}, nil
		}
	}
	return nil, nil
}

// DeleteWorkflowRunByID deletes a workflow run.
func (s *InMemoryWorkflowsStorage) DeleteWorkflowRunByID(_ context.Context, args DeleteWorkflowRunByIDArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := workflowKey(args.WorkflowName, args.RunID)
	delete(s.workflows, key)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ensureSnapshot ensures the run has a map-based snapshot, creating a default
// one if necessary. Must be called with the write lock held.
func ensureSnapshot(run *StorageWorkflowRun, key string) map[string]any {
	if run.Snapshot == nil {
		snap := map[string]any{
			"context":          make(map[string]any),
			"activePaths":      []any{},
			"activeStepsPath":  make(map[string]any),
			"timestamp":        time.Now().UnixMilli(),
			"suspendedPaths":   make(map[string]any),
			"resumeLabels":     make(map[string]any),
			"serializedStepGraph": []any{},
			"value":            make(map[string]any),
			"waitingPaths":     make(map[string]any),
			"status":           "pending",
			"runId":            run.RunID,
		}
		run.Snapshot = snap
		return snap
	}

	snap, ok := run.Snapshot.(map[string]any)
	if !ok {
		return nil
	}
	if snap["context"] == nil {
		return nil
	}
	return snap
}

func filterRuns(runs []StorageWorkflowRun, pred func(StorageWorkflowRun) bool) []StorageWorkflowRun {
	var result []StorageWorkflowRun
	for _, r := range runs {
		if pred(r) {
			result = append(result, r)
		}
	}
	return result
}

// deepCopyAny deep copies a value via JSON round-trip.
func deepCopyAny(v any) any {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

// deepCopyMap deep copies a map via JSON round-trip.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return m
	}
	return out
}
