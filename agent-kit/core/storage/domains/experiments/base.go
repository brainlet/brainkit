// Ported from: packages/core/src/storage/domains/experiments/base.ts
package experiments

import (
	"context"
	"time"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Experiment Types
// ---------------------------------------------------------------------------

// Experiment represents an experiment record.
type Experiment struct {
	ID             string                  `json:"id"`
	Name           *string                 `json:"name,omitempty"`
	Description    *string                 `json:"description,omitempty"`
	Metadata       map[string]any          `json:"metadata,omitempty"`
	DatasetID      *string                 `json:"datasetId"`      // null allowed
	DatasetVersion *int                    `json:"datasetVersion"` // null allowed
	TargetType     domains.TargetType      `json:"targetType"`
	TargetID       string                  `json:"targetId"`
	Status         domains.ExperimentStatus `json:"status"`
	TotalItems     int                     `json:"totalItems"`
	SucceededCount int                     `json:"succeededCount"`
	FailedCount    int                     `json:"failedCount"`
	SkippedCount   int                     `json:"skippedCount"`
	StartedAt      *time.Time              `json:"startedAt"`   // null allowed
	CompletedAt    *time.Time              `json:"completedAt"` // null allowed
	CreatedAt      time.Time               `json:"createdAt"`
	UpdatedAt      time.Time               `json:"updatedAt"`
}

// ExperimentResult is a single result within an experiment.
type ExperimentResult struct {
	ID                 string                          `json:"id"`
	ExperimentID       string                          `json:"experimentId"`
	ItemID             string                          `json:"itemId"`
	ItemDatasetVersion *int                            `json:"itemDatasetVersion"` // null allowed
	Input              any                             `json:"input"`
	Output             any                             `json:"output"`             // null allowed
	GroundTruth        any                             `json:"groundTruth"`        // null allowed
	Error              *domains.ExperimentResultError  `json:"error"`              // null allowed
	StartedAt          time.Time                       `json:"startedAt"`
	CompletedAt        time.Time                       `json:"completedAt"`
	RetryCount         int                             `json:"retryCount"`
	TraceID            *string                         `json:"traceId"` // null allowed
	CreatedAt          time.Time                       `json:"createdAt"`
}

// ---------------------------------------------------------------------------
// Experiment Input/Output Types
// ---------------------------------------------------------------------------

// CreateExperimentInput is the input for creating a new experiment.
type CreateExperimentInput struct {
	ID             *string                  `json:"id,omitempty"`
	Name           *string                  `json:"name,omitempty"`
	Description    *string                  `json:"description,omitempty"`
	Metadata       map[string]any           `json:"metadata,omitempty"`
	DatasetID      *string                  `json:"datasetId"`      // null allowed
	DatasetVersion *int                     `json:"datasetVersion"` // null allowed
	TargetType     domains.TargetType       `json:"targetType"`
	TargetID       string                   `json:"targetId"`
	TotalItems     int                      `json:"totalItems"`
}

// UpdateExperimentInput is the input for updating an experiment.
type UpdateExperimentInput struct {
	ID             string                    `json:"id"`
	Name           *string                   `json:"name,omitempty"`
	Description    *string                   `json:"description,omitempty"`
	Metadata       map[string]any            `json:"metadata,omitempty"`
	Status         *domains.ExperimentStatus `json:"status,omitempty"`
	TotalItems     *int                      `json:"totalItems,omitempty"`
	SucceededCount *int                      `json:"succeededCount,omitempty"`
	FailedCount    *int                      `json:"failedCount,omitempty"`
	SkippedCount   *int                      `json:"skippedCount,omitempty"`
	StartedAt      *time.Time               `json:"startedAt,omitempty"`
	CompletedAt    *time.Time               `json:"completedAt,omitempty"`
}

// AddExperimentResultInput is the input for adding an experiment result.
type AddExperimentResultInput struct {
	ID                 *string                        `json:"id,omitempty"`
	ExperimentID       string                         `json:"experimentId"`
	ItemID             string                         `json:"itemId"`
	ItemDatasetVersion *int                           `json:"itemDatasetVersion"` // null allowed
	Input              any                            `json:"input"`
	Output             any                            `json:"output"`             // null allowed
	GroundTruth        any                            `json:"groundTruth"`        // null allowed
	Error              *domains.ExperimentResultError  `json:"error"`              // null allowed
	StartedAt          time.Time                      `json:"startedAt"`
	CompletedAt        time.Time                      `json:"completedAt"`
	RetryCount         int                            `json:"retryCount"`
	TraceID            *string                        `json:"traceId,omitempty"` // null allowed
}

// ListExperimentsInput is the input for listing experiments.
type ListExperimentsInput struct {
	DatasetID  *string                   `json:"datasetId,omitempty"`
	Pagination domains.StoragePagination `json:"pagination"`
}

// ListExperimentsOutput is the paginated output for listing experiments.
type ListExperimentsOutput struct {
	Experiments []Experiment           `json:"experiments"`
	Pagination  domains.PaginationInfo `json:"pagination"`
}

// ListExperimentResultsInput is the input for listing experiment results.
type ListExperimentResultsInput struct {
	ExperimentID string                    `json:"experimentId"`
	Pagination   domains.StoragePagination `json:"pagination"`
}

// ListExperimentResultsOutput is the paginated output for listing experiment results.
type ListExperimentResultsOutput struct {
	Results    []ExperimentResult     `json:"results"`
	Pagination domains.PaginationInfo `json:"pagination"`
}

// ---------------------------------------------------------------------------
// ExperimentsStorage Interface
// ---------------------------------------------------------------------------

// ExperimentsStorage is the storage interface for dataset experiments.
// Provides the contract for experiment lifecycle and result tracking.
type ExperimentsStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Experiment Lifecycle ---

	// CreateExperiment creates a new experiment.
	CreateExperiment(ctx context.Context, input CreateExperimentInput) (Experiment, error)

	// UpdateExperiment updates an existing experiment.
	UpdateExperiment(ctx context.Context, input UpdateExperimentInput) (Experiment, error)

	// GetExperimentByID retrieves an experiment by ID.
	GetExperimentByID(ctx context.Context, id string) (Experiment, error)

	// ListExperiments lists experiments with optional filtering.
	ListExperiments(ctx context.Context, args ListExperimentsInput) (ListExperimentsOutput, error)

	// DeleteExperiment removes an experiment by ID.
	DeleteExperiment(ctx context.Context, id string) error

	// --- Results (per-item) ---

	// AddExperimentResult adds a result for a specific item.
	AddExperimentResult(ctx context.Context, input AddExperimentResultInput) (ExperimentResult, error)

	// GetExperimentResultByID retrieves an experiment result by ID.
	GetExperimentResultByID(ctx context.Context, id string) (ExperimentResult, error)

	// ListExperimentResults lists experiment results with optional filtering.
	ListExperimentResults(ctx context.Context, args ListExperimentResultsInput) (ListExperimentResultsOutput, error)

	// DeleteExperimentResults removes all results for an experiment.
	DeleteExperimentResults(ctx context.Context, experimentID string) error
}
