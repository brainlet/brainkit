// Ported from: packages/core/src/storage/domains/shared.ts
package domains

import "time"

// ---------------------------------------------------------------------------
// Pagination
// ---------------------------------------------------------------------------

// PaginationArgs holds pagination arguments for list queries.
// Page is zero-indexed. PerPage defaults to 10 if nil.
type PaginationArgs struct {
	// Zero-indexed page number.
	Page int `json:"page"`
	// Number of items per page (1–100).
	PerPage int `json:"perPage"`
}

// PaginationInfo is the pagination response envelope returned by paginated list endpoints.
// PerPage is -1 when pagination is disabled (equivalent to TS `false`).
type PaginationInfo struct {
	// Total number of items available.
	Total int `json:"total"`
	// Current page number.
	Page int `json:"page"`
	// Number of items per page, or -1 if pagination is disabled.
	PerPage int `json:"perPage"`
	// True if more pages are available.
	HasMore bool `json:"hasMore"`
}

// PerPageDisabled is the sentinel value for "no pagination limit",
// equivalent to `perPage: false` in the TypeScript source.
const PerPageDisabled = -1

// ---------------------------------------------------------------------------
// Date range filtering
// ---------------------------------------------------------------------------

// DateRange represents a date range for filtering by time.
type DateRange struct {
	// Start of date range (inclusive by default).
	Start *time.Time `json:"start,omitempty"`
	// End of date range (inclusive by default).
	End *time.Time `json:"end,omitempty"`
	// When true, excludes the start date from results (uses > instead of >=).
	StartExclusive bool `json:"startExclusive,omitempty"`
	// When true, excludes the end date from results (uses < instead of <=).
	EndExclusive bool `json:"endExclusive,omitempty"`
}

// ---------------------------------------------------------------------------
// Sort direction
// ---------------------------------------------------------------------------

// SortDirection represents the sort order for list queries.
type SortDirection string

const (
	SortASC  SortDirection = "ASC"
	SortDESC SortDirection = "DESC"
)

// ---------------------------------------------------------------------------
// Entity type (ported from observability/types/tracing.ts EntityType enum)
// ---------------------------------------------------------------------------

// EntityType identifies the kind of entity (agent, processor, tool, workflow).
// TODO: Replace with the canonical EntityType once the observability package is ported.
type EntityType string

const (
	EntityTypeAgent     EntityType = "agent"
	EntityTypeProcessor EntityType = "processor"
	EntityTypeTool      EntityType = "tool"
	EntityTypeWorkflow  EntityType = "workflow"
)

// ---------------------------------------------------------------------------
// Common field-value helpers (Go equivalents of the zod field descriptors)
// ---------------------------------------------------------------------------
// In TypeScript these are zod schemas used for validation. In Go they serve
// purely as documentation — the actual validation is done at the adapter layer.
//
// The TS source defines: createdAtField, updatedAtField, dbTimestamps,
// entityTypeField, entityIdField, entityNameField, userIdField,
// organizationIdField, resourceIdField, runIdField, sessionIdField,
// threadIdField, requestIdField, environmentField, sourceField,
// serviceNameField.
//
// In Go these are represented as struct fields on the domain-specific types
// rather than standalone schema objects. No additional Go code is needed.

// ---------------------------------------------------------------------------
// StoragePagination (alias for PaginationArgs)
// ---------------------------------------------------------------------------
// StoragePagination is an alias for PaginationArgs, used by dataset, experiment,
// and other domain types that reference the storage-layer pagination struct.
type StoragePagination = PaginationArgs

// ---------------------------------------------------------------------------
// EntityStatus (versioned entity publication status)
// ---------------------------------------------------------------------------

// EntityStatus represents the publication status of a versioned entity.
type EntityStatus string

const (
	EntityStatusDraft     EntityStatus = "draft"
	EntityStatusPublished EntityStatus = "published"
	EntityStatusArchived  EntityStatus = "archived"
)

// ---------------------------------------------------------------------------
// TargetType (experiment target type)
// ---------------------------------------------------------------------------

// TargetType is the type of entity a dataset experiment targets.
type TargetType string

const (
	TargetTypeAgent     TargetType = "agent"
	TargetTypeWorkflow  TargetType = "workflow"
	TargetTypeScorer    TargetType = "scorer"
	TargetTypeProcessor TargetType = "processor"
)

// ---------------------------------------------------------------------------
// ExperimentStatus (experiment lifecycle status)
// ---------------------------------------------------------------------------

// ExperimentStatus represents the status of an experiment.
type ExperimentStatus string

const (
	ExperimentStatusPending   ExperimentStatus = "pending"
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusFailed    ExperimentStatus = "failed"
)

// ---------------------------------------------------------------------------
// WorkflowRunStatus (workflow run lifecycle status)
// ---------------------------------------------------------------------------

// WorkflowRunStatus represents the status of a workflow run.
type WorkflowRunStatus string

const (
	WorkflowRunStatusPending   WorkflowRunStatus = "pending"
	WorkflowRunStatusRunning   WorkflowRunStatus = "running"
	WorkflowRunStatusCompleted WorkflowRunStatus = "completed"
	WorkflowRunStatusFailed    WorkflowRunStatus = "failed"
	WorkflowRunStatusSuspended WorkflowRunStatus = "suspended"
	WorkflowRunStatusWaiting   WorkflowRunStatus = "waiting"
)

// ---------------------------------------------------------------------------
// Index types (used by operations domain)
// ---------------------------------------------------------------------------

// IndexMethod enumerates supported index methods.
type IndexMethod string

const (
	IndexMethodBtree  IndexMethod = "btree"
	IndexMethodHash   IndexMethod = "hash"
	IndexMethodGIN    IndexMethod = "gin"
	IndexMethodGIST   IndexMethod = "gist"
	IndexMethodSPGIST IndexMethod = "spgist"
	IndexMethodBRIN   IndexMethod = "brin"
)

// CreateIndexOptions specifies options for creating a database index.
type CreateIndexOptions struct {
	Name       string         `json:"name"`
	Table      string         `json:"table"`
	Columns    []string       `json:"columns"`
	Unique     *bool          `json:"unique,omitempty"`
	Concurrent *bool          `json:"concurrent,omitempty"`
	Where      *string        `json:"where,omitempty"` // WARNING: not parameterized, must be pre-validated
	Method     *IndexMethod   `json:"method,omitempty"`
	Opclass    *string        `json:"opclass,omitempty"`
	Storage    map[string]any `json:"storage,omitempty"`
	Tablespace *string        `json:"tablespace,omitempty"`
}

// IndexInfo describes an existing database index.
type IndexInfo struct {
	Name       string   `json:"name"`
	Table      string   `json:"table"`
	Columns    []string `json:"columns"`
	Unique     bool     `json:"unique"`
	Size       string   `json:"size"`
	Definition string   `json:"definition"`
}

// StorageIndexStats extends IndexInfo with usage statistics.
type StorageIndexStats struct {
	IndexInfo
	Scans         int        `json:"scans"`
	TuplesRead    int        `json:"tuples_read"`
	TuplesFetched int        `json:"tuples_fetched"`
	LastUsed      *time.Time `json:"last_used,omitempty"`
	Method        *string    `json:"method,omitempty"`
}

// ---------------------------------------------------------------------------
// Serialized Error (lightweight reference for workflow state)
// ---------------------------------------------------------------------------

// SerializedErrorRef is a lightweight serialized error structure for use in
// workflow state options. This avoids importing the mastraerror package in
// domain sub-packages.
type SerializedErrorRef struct {
	Message string  `json:"message"`
	Stack   *string `json:"stack,omitempty"`
	Code    *string `json:"code,omitempty"`
	Name    *string `json:"name,omitempty"`
}

// ---------------------------------------------------------------------------
// UpdateWorkflowStateOptions
// ---------------------------------------------------------------------------

// UpdateWorkflowResumeLabel describes a resume label in workflow state.
type UpdateWorkflowResumeLabel struct {
	StepID       string `json:"stepId"`
	ForeachIndex *int   `json:"foreachIndex,omitempty"`
}

// UpdateWorkflowStateOptions specifies options for updating workflow state.
type UpdateWorkflowStateOptions struct {
	Status         WorkflowRunStatus                    `json:"status"`
	Result         any                                  `json:"result,omitempty"`         // StepResult
	Error          *SerializedErrorRef                  `json:"error,omitempty"`
	SuspendedPaths map[string][]int                     `json:"suspendedPaths,omitempty"`
	WaitingPaths   map[string][]int                     `json:"waitingPaths,omitempty"`
	ResumeLabels   map[string]UpdateWorkflowResumeLabel `json:"resumeLabels,omitempty"`
}

// ---------------------------------------------------------------------------
// ExperimentResultError
// ---------------------------------------------------------------------------

// ExperimentResultError is the error structure in an experiment result.
type ExperimentResultError struct {
	Message string  `json:"message"`
	Stack   *string `json:"stack,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// ---------------------------------------------------------------------------
// BatchInsertItemInput (used by datasets domain)
// ---------------------------------------------------------------------------

// BatchInsertItemInput is a single item in a batch insert operation.
type BatchInsertItemInput struct {
	Input       any            `json:"input"`
	GroundTruth any            `json:"groundTruth,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
