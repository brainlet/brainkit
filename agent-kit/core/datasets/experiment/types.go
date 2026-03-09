// Ported from: packages/core/src/datasets/experiment/types.ts
package experiment

import (
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/mastra"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Local interface types
// ---------------------------------------------------------------------------

// MastraScorer is a narrow interface for scorers used by experiment execution.
// The real evals.MastraScorer has different method signatures
// (Run takes context.Context + *ScorerRun, Judge returns *ScorerJudgeConfig).
// This interface is intentionally simplified for the experiment use case.
type MastraScorer interface {
	// ID returns the scorer's unique identifier.
	ID() string
	// Name returns the scorer's display name.
	Name() string
	// Description returns the scorer's description.
	Description() string
	// HasJudge reports whether the scorer uses a judge LLM.
	HasJudge() bool
	// Run executes the scorer with the given input.
	Run(input any) (*ScorerRunResult, error)
}

// ScorerRunResult is the result returned by MastraScorer.Run().
// Simplified version for the experiment use case — the real evals.ScorerRunResult
// has additional fields (Extract/Analyze step results, prompts).
type ScorerRunResult struct {
	Score  *float64 `json:"score"`
	Reason *string  `json:"reason,omitempty"`
}

// Mastra is the narrow interface for the Mastra orchestrator used by experiments.
// Ported from: packages/core/src/datasets/experiment — uses mastra instance.
type Mastra interface {
	GetStorage() *storage.MastraCompositeStore
	GetScorerByID(id string) mastra.MastraScorer
	GetAgentByID(id string) (mastra.Agent, error)
	GetAgent(name string) (mastra.Agent, error)
	GetWorkflowByID(id string) (mastra.AnyWorkflow, error)
	GetWorkflow(name string) (mastra.AnyWorkflow, error)
}

// MastraCompositeStore is re-exported from the storage package.
type MastraCompositeStore = storage.MastraCompositeStore

// ScorerRunInputForAgent represents scorer input for agent targets.
// STUB REASON: The real type is evals.ScorerRunInputForAgent (struct with
// InputMessages, RememberedMessages, SystemMessages, TaggedSystemMessages fields).
// However, executeAgent extracts this from rawResult["scoringData"].(map[string]any)["input"]
// which yields any at runtime. A type assertion would be needed but the concrete
// value is often map[string]any (from JSON deserialization), not the struct. Kept as
// any to preserve runtime compatibility.
type ScorerRunInputForAgent = any

// ScorerRunOutputForAgent represents scorer output for agent targets.
// STUB REASON: The real type is evals.ScorerRunOutputForAgent (= []evals.MastraDBMessage).
// Same runtime issue as ScorerRunInputForAgent — extracted from map[string]any at runtime.
// Kept as any to preserve runtime compatibility.
type ScorerRunOutputForAgent = any

// ScoringData holds input/output pairs for scorer evaluation.
// The real model.ScoringData uses map[string]any for Input/Output fields;
// this version uses any (via ScorerRunInputForAgent/ScorerRunOutputForAgent aliases).
type ScoringData struct {
	Input  ScorerRunInputForAgent  `json:"input,omitempty"`
	Output ScorerRunOutputForAgent `json:"output,omitempty"`
}

// TargetType is the type of entity a dataset experiment targets.
// Re-exported from storage/domains — single source of truth.
type TargetType = domains.TargetType

const (
	TargetTypeAgent     = domains.TargetTypeAgent
	TargetTypeWorkflow  = domains.TargetTypeWorkflow
	TargetTypeScorer    = domains.TargetTypeScorer
	TargetTypeProcessor = domains.TargetTypeProcessor
)

// ExperimentStatus represents the status of an experiment.
// Re-exported from storage/domains — single source of truth.
type ExperimentStatus = domains.ExperimentStatus

const (
	ExperimentStatusPending   = domains.ExperimentStatusPending
	ExperimentStatusRunning   = domains.ExperimentStatusRunning
	ExperimentStatusCompleted = domains.ExperimentStatusCompleted
	ExperimentStatusFailed    = domains.ExperimentStatusFailed
)

// ============================================================================
// Data Item
// ============================================================================

// DataItem is a single data item for inline experiment data.
// Internal — not publicly exported from @mastra/core in TS.
type DataItem struct {
	// ID is a unique ID (auto-generated if omitted).
	ID string `json:"id,omitempty"`
	// Input is the input data passed to the task.
	Input any `json:"input"`
	// GroundTruth is the ground truth for scoring.
	GroundTruth any `json:"groundTruth,omitempty"`
	// Metadata is additional metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// Experiment Configuration
// ============================================================================

// TaskFunc is the signature for an inline task function.
type TaskFunc func(args TaskFuncArgs) (any, error)

// TaskFuncArgs holds the arguments passed to a TaskFunc.
type TaskFuncArgs struct {
	// Input is the input data for this item.
	Input any
	// Mastra is the Mastra instance.
	Mastra Mastra
	// GroundTruth is the expected output for scoring.
	GroundTruth any
	// Metadata is additional metadata from the dataset item.
	Metadata map[string]any
}

// ExperimentConfig is the internal configuration for running a dataset experiment.
// Not publicly exported — users interact via Dataset.StartExperiment().
// All new fields are optional — existing internal callers are unaffected.
type ExperimentConfig struct {
	// DatasetID is the ID of dataset in storage (injected by Dataset).
	DatasetID string `json:"datasetId,omitempty"`
	// Data is an override data source — inline array (bypasses storage load).
	Data []DataItem `json:"-"`
	// DataFactory is an async factory for inline data (bypasses storage load).
	DataFactory func() ([]DataItem, error) `json:"-"`

	// TargetType is the registry-based target type.
	TargetType TargetType `json:"targetType,omitempty"`
	// TargetID is the registry-based target ID.
	TargetID string `json:"targetId,omitempty"`
	// Task is an inline task function.
	Task TaskFunc `json:"-"`

	// Scorers is a list of MastraScorer instances or string IDs.
	Scorers []any `json:"-"`

	// Version pins to a specific dataset version (default: latest).
	// Only applies when DatasetID is used.
	Version *int `json:"version,omitempty"`
	// MaxConcurrency is the maximum concurrent executions (default: 5).
	MaxConcurrency int `json:"maxConcurrency,omitempty"`
	// ItemTimeout is the per-item execution timeout in milliseconds.
	ItemTimeout int `json:"itemTimeout,omitempty"`
	// MaxRetries is the maximum retries per item on failure (default: 0 = no retries).
	// Abort errors are never retried.
	MaxRetries int `json:"maxRetries,omitempty"`
	// ExperimentID is a pre-created experiment ID (for async trigger — skips experiment creation).
	ExperimentID string `json:"experimentId,omitempty"`
	// Name is the experiment name (used for display / grouping).
	Name string `json:"name,omitempty"`
	// Description is the experiment description.
	Description string `json:"description,omitempty"`
	// Metadata is arbitrary metadata for the experiment.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StartExperimentConfig is the configuration for starting an experiment on a dataset.
// The dataset is always the data source — no DatasetID/Data needed.
type StartExperimentConfig struct {
	// TargetType is the registry-based target type.
	TargetType TargetType `json:"targetType,omitempty"`
	// TargetID is the registry-based target ID.
	TargetID string `json:"targetId,omitempty"`
	// Task is an inline task function.
	Task TaskFunc `json:"-"`

	// Scorers is a list of MastraScorer instances or string IDs.
	Scorers []any `json:"-"`

	// Version pins to a specific dataset version (default: latest).
	Version *int `json:"version,omitempty"`
	// MaxConcurrency is the maximum concurrent executions (default: 5).
	MaxConcurrency int `json:"maxConcurrency,omitempty"`
	// ItemTimeout is the per-item execution timeout in milliseconds.
	ItemTimeout int `json:"itemTimeout,omitempty"`
	// MaxRetries is the maximum retries per item on failure (default: 0).
	MaxRetries int `json:"maxRetries,omitempty"`
	// Name is the experiment name.
	Name string `json:"name,omitempty"`
	// Description is the experiment description.
	Description string `json:"description,omitempty"`
	// Metadata is arbitrary metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// Item Result
// ============================================================================

// ItemResult is the result of executing a single dataset item.
type ItemResult struct {
	// ItemID is the ID of the dataset item.
	ItemID string `json:"itemId"`
	// ItemVersion is the dataset version of the item when executed.
	ItemVersion int `json:"itemVersion"`
	// Input is the input data that was passed to the target.
	Input any `json:"input"`
	// Output is the output from the target (nil if failed).
	Output any `json:"output"`
	// GroundTruth is the expected output from the dataset item.
	GroundTruth any `json:"groundTruth"`
	// Error is the structured error if execution failed.
	Error *ExecutionError `json:"error"`
	// StartedAt is when execution started.
	StartedAt time.Time `json:"startedAt"`
	// CompletedAt is when execution completed.
	CompletedAt time.Time `json:"completedAt"`
	// RetryCount is the number of retry attempts.
	RetryCount int `json:"retryCount"`
}

// ExecutionError is a structured error from target execution.
type ExecutionError struct {
	// Message is the error message.
	Message string `json:"message"`
	// Stack is the stack trace (optional).
	Stack string `json:"stack,omitempty"`
	// Code is an error code (optional).
	Code string `json:"code,omitempty"`
}

// ============================================================================
// Scorer Result
// ============================================================================

// ScorerResult is the result from a single scorer for an item.
type ScorerResult struct {
	// ScorerID is the ID of the scorer.
	ScorerID string `json:"scorerId"`
	// ScorerName is the display name of the scorer.
	ScorerName string `json:"scorerName"`
	// Score is the computed score (nil if scorer failed).
	Score *float64 `json:"score"`
	// Reason is the reason/explanation for the score.
	Reason *string `json:"reason"`
	// Error is the error message if scorer failed.
	Error *string `json:"error"`
}

// ============================================================================
// Item With Scores
// ============================================================================

// ItemWithScores is an item result with all scorer results attached.
type ItemWithScores struct {
	ItemResult
	// Scores holds the results from all scorers for this item.
	Scores []ScorerResult `json:"scores"`
}

// ============================================================================
// Experiment Summary
// ============================================================================

// ExperimentSummary is the summary of an entire dataset experiment.
type ExperimentSummary struct {
	// ExperimentID is the unique ID of this experiment.
	ExperimentID string `json:"experimentId"`
	// Status is the final status of the experiment.
	Status ExperimentStatus `json:"status"`
	// TotalItems is the total number of items in the dataset.
	TotalItems int `json:"totalItems"`
	// SucceededCount is the number of items that succeeded.
	SucceededCount int `json:"succeededCount"`
	// FailedCount is the number of items that failed.
	FailedCount int `json:"failedCount"`
	// SkippedCount is the number of items skipped (e.g. due to abort).
	SkippedCount int `json:"skippedCount"`
	// CompletedWithErrors is true if run completed but some items failed.
	CompletedWithErrors bool `json:"completedWithErrors"`
	// StartedAt is when the experiment started.
	StartedAt time.Time `json:"startedAt"`
	// CompletedAt is when the experiment completed.
	CompletedAt time.Time `json:"completedAt"`
	// Results holds all item results with their scores.
	Results []ItemWithScores `json:"results"`
}
