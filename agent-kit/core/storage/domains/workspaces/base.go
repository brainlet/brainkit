// Ported from: packages/core/src/storage/domains/workspaces/base.ts
package workspaces

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Workspace Version Types
// ---------------------------------------------------------------------------

// WorkspaceVersion represents a stored version of a workspace's configuration.
// Config fields are top-level on the version row (no nested snapshot object).
type WorkspaceVersion struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the workspace this version belongs to.
	WorkspaceID string `json:"workspaceId"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`

	// TODO: Embed StorageWorkspaceSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// CreateWorkspaceVersionInput is the input for creating a new workspace version.
type CreateWorkspaceVersionInput struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the workspace this version belongs to.
	WorkspaceID string `json:"workspaceId"`
	// Sequential version number.
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`

	// TODO: Embed StorageWorkspaceSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// WorkspaceVersionSortDirection is the sort direction for version listings.
type WorkspaceVersionSortDirection string

const (
	WorkspaceVersionSortASC  WorkspaceVersionSortDirection = "ASC"
	WorkspaceVersionSortDESC WorkspaceVersionSortDirection = "DESC"
)

// WorkspaceVersionOrderBy defines fields for ordering version listings.
type WorkspaceVersionOrderBy string

const (
	WorkspaceVersionOrderByVersionNumber WorkspaceVersionOrderBy = "versionNumber"
	WorkspaceVersionOrderByCreatedAt     WorkspaceVersionOrderBy = "createdAt"
)

// ListWorkspaceVersionsInput is the input for listing workspace versions.
type ListWorkspaceVersionsInput struct {
	// ID of the workspace to list versions for.
	WorkspaceID string `json:"workspaceId"`
	// Zero-indexed page number.
	Page *int `json:"page,omitempty"`
	// Number of items per page. Use -1 to fetch all records without limit.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderByField     *WorkspaceVersionOrderBy      `json:"orderByField,omitempty"`
	OrderByDirection *WorkspaceVersionSortDirection `json:"orderByDirection,omitempty"`
}

// ListWorkspaceVersionsOutput is the output for listing workspace versions.
type ListWorkspaceVersionsOutput struct {
	Versions []WorkspaceVersion `json:"versions"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PerPage  int                `json:"perPage"`
	HasMore  bool               `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// WorkspacesStorage Interface
// ---------------------------------------------------------------------------

// WorkspacesStorage is the storage interface for the workspaces domain.
// It extends VersionedStorageDomain with workspace-specific types.
//
// TODO: Replace `any` parameter/return types with concrete types from
// storage/types.go once ported (StorageWorkspaceType,
// StorageCreateWorkspaceInput, StorageUpdateWorkspaceInput, etc.).
type WorkspacesStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Entity CRUD ---

	// GetByID retrieves a workspace by ID.
	GetByID(ctx context.Context, id string) (any, error)

	// Create creates a new workspace.
	Create(ctx context.Context, input any) (any, error)

	// Update updates an existing workspace.
	Update(ctx context.Context, input any) (any, error)

	// Delete removes a workspace by ID.
	Delete(ctx context.Context, id string) error

	// List lists workspaces with optional filtering.
	List(ctx context.Context, args any) (any, error)

	// --- Version Methods ---

	// CreateVersion creates a new workspace version.
	CreateVersion(ctx context.Context, input CreateWorkspaceVersionInput) (*WorkspaceVersion, error)

	// GetVersion retrieves a version by its ID.
	GetVersion(ctx context.Context, id string) (*WorkspaceVersion, error)

	// GetVersionByNumber retrieves a version by workspace ID and version number.
	GetVersionByNumber(ctx context.Context, workspaceID string, versionNumber int) (*WorkspaceVersion, error)

	// GetLatestVersion retrieves the latest version for a workspace.
	GetLatestVersion(ctx context.Context, workspaceID string) (*WorkspaceVersion, error)

	// ListVersions lists versions for a workspace with pagination and sorting.
	ListVersions(ctx context.Context, input ListWorkspaceVersionsInput) (*ListWorkspaceVersionsOutput, error)

	// DeleteVersion removes a version by ID.
	DeleteVersion(ctx context.Context, id string) error

	// DeleteVersionsByParentID removes all versions for a workspace.
	DeleteVersionsByParentID(ctx context.Context, workspaceID string) error

	// CountVersions returns the number of versions for a workspace.
	CountVersions(ctx context.Context, workspaceID string) (int, error)

	// --- Resolution Methods ---

	// GetByIDResolved resolves an entity by merging its thin record with
	// the active or latest version config.
	GetByIDResolved(ctx context.Context, id string, status string) (any, error)

	// ListResolved lists entities with version resolution.
	ListResolved(ctx context.Context, args any) (any, error)
}
