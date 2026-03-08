// Ported from: packages/core/src/storage/domains/workflows/base.ts
package workflows

import (
	"context"
	"time"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Workflow Run Types
// ---------------------------------------------------------------------------

// WorkflowRunState represents a workflow run's state snapshot.
// In TS this is imported from ../workflows. Stored as JSON or string.
type WorkflowRunState = map[string]any

// StepResult represents the result of a workflow step.
// In TS this is imported from ../workflows.
type StepResult = map[string]any

// StorageWorkflowRun is the internal storage representation of a workflow run.
type StorageWorkflowRun struct {
	WorkflowName string    `json:"workflow_name"`
	RunID        string    `json:"run_id"`
	ResourceID   string    `json:"resourceId,omitempty"`
	Snapshot     any       `json:"snapshot,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// WorkflowRun is the public representation of a workflow run.
type WorkflowRun struct {
	WorkflowName string    `json:"workflowName"`
	RunID        string    `json:"runId"`
	ResourceID   string    `json:"resourceId,omitempty"`
	Snapshot     any       `json:"snapshot,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// WorkflowRuns holds a list of workflow runs with total count.
type WorkflowRuns struct {
	Runs  []WorkflowRun `json:"runs"`
	Total int           `json:"total"`
}

// ListWorkflowRunsInput holds the arguments for listing workflow runs.
type ListWorkflowRunsInput struct {
	WorkflowName string     `json:"workflowName,omitempty"`
	FromDate     *time.Time `json:"fromDate,omitempty"`
	ToDate       *time.Time `json:"toDate,omitempty"`
	PerPage      *int       `json:"perPage,omitempty"`
	Page         *int       `json:"page,omitempty"`
	ResourceID   string     `json:"resourceId,omitempty"`
	Status       string     `json:"status,omitempty"`
}

// UpdateWorkflowResumeLabel describes a resume label in workflow state.
type UpdateWorkflowResumeLabel = domains.UpdateWorkflowResumeLabel

// UpdateWorkflowStateOptions holds the options for updating workflow state.
type UpdateWorkflowStateOptions = domains.UpdateWorkflowStateOptions

// UpdateWorkflowResultsArgs holds the arguments for updateWorkflowResults.
type UpdateWorkflowResultsArgs struct {
	WorkflowName   string         `json:"workflowName"`
	RunID          string         `json:"runId"`
	StepID         string         `json:"stepId"`
	Result         StepResult     `json:"result"`
	RequestContext map[string]any `json:"requestContext,omitempty"`
}

// UpdateWorkflowStateArgs holds the arguments for updateWorkflowState.
type UpdateWorkflowStateArgs struct {
	WorkflowName string                     `json:"workflowName"`
	RunID        string                     `json:"runId"`
	Opts         UpdateWorkflowStateOptions `json:"opts"`
}

// PersistWorkflowSnapshotArgs holds the arguments for persistWorkflowSnapshot.
type PersistWorkflowSnapshotArgs struct {
	WorkflowName string           `json:"workflowName"`
	RunID        string           `json:"runId"`
	ResourceID   string           `json:"resourceId,omitempty"`
	Snapshot     WorkflowRunState `json:"snapshot"`
	CreatedAt    *time.Time       `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time       `json:"updatedAt,omitempty"`
}

// LoadWorkflowSnapshotArgs holds the arguments for loadWorkflowSnapshot.
type LoadWorkflowSnapshotArgs struct {
	WorkflowName string `json:"workflowName"`
	RunID        string `json:"runId"`
}

// GetWorkflowRunByIDArgs holds the arguments for getWorkflowRunById.
type GetWorkflowRunByIDArgs struct {
	RunID        string `json:"runId"`
	WorkflowName string `json:"workflowName,omitempty"`
}

// DeleteWorkflowRunByIDArgs holds the arguments for deleteWorkflowRunById.
type DeleteWorkflowRunByIDArgs struct {
	RunID        string `json:"runId"`
	WorkflowName string `json:"workflowName"`
}

// ---------------------------------------------------------------------------
// WorkflowsStorage Interface
// ---------------------------------------------------------------------------

// WorkflowsStorage is the storage interface for the workflows domain.
type WorkflowsStorage interface {
	// Init initializes the storage domain.
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// SupportsConcurrentUpdates returns true if the storage backend supports
	// concurrent step result updates (e.g. for forEach with concurrency > 1).
	SupportsConcurrentUpdates() bool

	// UpdateWorkflowResults updates workflow step results.
	UpdateWorkflowResults(ctx context.Context, args UpdateWorkflowResultsArgs) (map[string]StepResult, error)

	// UpdateWorkflowState updates workflow state.
	UpdateWorkflowState(ctx context.Context, args UpdateWorkflowStateArgs) (WorkflowRunState, error)

	// PersistWorkflowSnapshot persists a workflow run snapshot.
	PersistWorkflowSnapshot(ctx context.Context, args PersistWorkflowSnapshotArgs) error

	// LoadWorkflowSnapshot loads a workflow run snapshot.
	LoadWorkflowSnapshot(ctx context.Context, args LoadWorkflowSnapshotArgs) (WorkflowRunState, error)

	// ListWorkflowRuns lists workflow runs with optional filtering.
	ListWorkflowRuns(ctx context.Context, args *ListWorkflowRunsInput) (*WorkflowRuns, error)

	// GetWorkflowRunByID retrieves a workflow run by ID.
	GetWorkflowRunByID(ctx context.Context, args GetWorkflowRunByIDArgs) (*WorkflowRun, error)

	// DeleteWorkflowRunByID deletes a workflow run by ID.
	DeleteWorkflowRunByID(ctx context.Context, args DeleteWorkflowRunByIDArgs) error
}
