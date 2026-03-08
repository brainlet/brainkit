// Ported from: packages/core/src/storage/domains/agents/base.ts
package agents

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Agent Version Types
// ---------------------------------------------------------------------------

// AgentVersion represents a stored version of an agent configuration.
// The config fields are top-level on the version row (no nested snapshot object).
type AgentVersion struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the agent this version belongs to.
	AgentID string `json:"agentId"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`

	// TODO: Embed StorageAgentSnapshotType fields once storage/types.go is ported.
	// The snapshot fields are top-level on the version row.
	Snapshot any `json:"snapshot,omitempty"`
}

// CreateAgentVersionInput is the input for creating a new agent version.
// Config fields are top-level (no nested snapshot object).
type CreateAgentVersionInput struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the agent this version belongs to.
	AgentID string `json:"agentId"`
	// Sequential version number.
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`

	// TODO: Embed StorageAgentSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// VersionSortDirection is the sort direction for version listings.
type VersionSortDirection string

const (
	VersionSortASC  VersionSortDirection = "ASC"
	VersionSortDESC VersionSortDirection = "DESC"
)

// VersionOrderBy defines fields that can be used for ordering version listings.
type VersionOrderBy string

const (
	VersionOrderByVersionNumber VersionOrderBy = "versionNumber"
	VersionOrderByCreatedAt    VersionOrderBy = "createdAt"
)

// ListVersionsInput is the input for listing agent versions with pagination and sorting.
type ListVersionsInput struct {
	// ID of the agent to list versions for.
	AgentID string `json:"agentId"`
	// Zero-indexed page number.
	Page *int `json:"page,omitempty"`
	// Number of items per page. Use -1 to fetch all records without limit.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderByField     *VersionOrderBy      `json:"orderByField,omitempty"`
	OrderByDirection *VersionSortDirection `json:"orderByDirection,omitempty"`
}

// ListVersionsOutput is the output for listing agent versions with pagination info.
type ListVersionsOutput struct {
	Versions []AgentVersion `json:"versions"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PerPage  int            `json:"perPage"`
	HasMore  bool           `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// AgentsStorage Interface
// ---------------------------------------------------------------------------

// AgentsStorage is the storage interface for the agents domain.
// It extends VersionedStorageDomain with agent-specific types.
//
// TODO: Replace `any` parameter/return types with concrete types from
// storage/types.go once that file is ported (StorageAgentType,
// StorageCreateAgentInput, StorageUpdateAgentInput, StorageListAgentsInput,
// StorageListAgentsOutput, StorageListAgentsResolvedOutput,
// StorageResolvedAgentType).
type AgentsStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Entity CRUD ---

	// GetByID retrieves an agent by ID.
	GetByID(ctx context.Context, id string) (any, error)

	// Create creates a new agent.
	Create(ctx context.Context, input any) (any, error)

	// Update updates an existing agent.
	Update(ctx context.Context, input any) (any, error)

	// Delete removes an agent by ID.
	Delete(ctx context.Context, id string) error

	// List lists agents with optional filtering.
	List(ctx context.Context, args any) (any, error)

	// --- Version Methods ---

	// CreateVersion creates a new agent version.
	CreateVersion(ctx context.Context, input CreateAgentVersionInput) (*AgentVersion, error)

	// GetVersion retrieves a version by its ID.
	GetVersion(ctx context.Context, id string) (*AgentVersion, error)

	// GetVersionByNumber retrieves a version by agent ID and version number.
	GetVersionByNumber(ctx context.Context, agentID string, versionNumber int) (*AgentVersion, error)

	// GetLatestVersion retrieves the latest version for an agent.
	GetLatestVersion(ctx context.Context, agentID string) (*AgentVersion, error)

	// ListVersions lists versions for an agent with pagination and sorting.
	ListVersions(ctx context.Context, input ListVersionsInput) (*ListVersionsOutput, error)

	// DeleteVersion removes a version by ID.
	DeleteVersion(ctx context.Context, id string) error

	// DeleteVersionsByParentID removes all versions for an agent.
	DeleteVersionsByParentID(ctx context.Context, agentID string) error

	// CountVersions returns the number of versions for an agent.
	CountVersions(ctx context.Context, agentID string) (int, error)

	// --- Resolution Methods ---

	// GetByIDResolved resolves an entity by merging its thin record with
	// the active or latest version config.
	GetByIDResolved(ctx context.Context, id string, status string) (any, error)

	// ListResolved lists entities with version resolution.
	ListResolved(ctx context.Context, args any) (any, error)
}
