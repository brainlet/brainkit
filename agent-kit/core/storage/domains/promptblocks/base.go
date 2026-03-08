// Ported from: packages/core/src/storage/domains/prompt-blocks/base.ts
package promptblocks

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Prompt Block Version Types
// ---------------------------------------------------------------------------

// PromptBlockVersion represents a stored version of a prompt block's content.
// Config fields are top-level on the version row (no nested snapshot object).
type PromptBlockVersion struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the prompt block this version belongs to.
	BlockID string `json:"blockId"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`

	// TODO: Embed StoragePromptBlockSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// CreatePromptBlockVersionInput is the input for creating a new prompt block version.
type CreatePromptBlockVersionInput struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the prompt block this version belongs to.
	BlockID string `json:"blockId"`
	// Sequential version number.
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`

	// TODO: Embed StoragePromptBlockSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// PromptBlockVersionSortDirection is the sort direction for version listings.
type PromptBlockVersionSortDirection string

const (
	PromptBlockVersionSortASC  PromptBlockVersionSortDirection = "ASC"
	PromptBlockVersionSortDESC PromptBlockVersionSortDirection = "DESC"
)

// PromptBlockVersionOrderBy defines fields for ordering version listings.
type PromptBlockVersionOrderBy string

const (
	PromptBlockVersionOrderByVersionNumber PromptBlockVersionOrderBy = "versionNumber"
	PromptBlockVersionOrderByCreatedAt     PromptBlockVersionOrderBy = "createdAt"
)

// ListPromptBlockVersionsInput is the input for listing prompt block versions.
type ListPromptBlockVersionsInput struct {
	// ID of the prompt block to list versions for.
	BlockID string `json:"blockId"`
	// Zero-indexed page number.
	Page *int `json:"page,omitempty"`
	// Number of items per page. Use -1 to fetch all records without limit.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderByField     *PromptBlockVersionOrderBy      `json:"orderByField,omitempty"`
	OrderByDirection *PromptBlockVersionSortDirection `json:"orderByDirection,omitempty"`
}

// ListPromptBlockVersionsOutput is the output for listing prompt block versions.
type ListPromptBlockVersionsOutput struct {
	Versions []PromptBlockVersion `json:"versions"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PerPage  int                  `json:"perPage"`
	HasMore  bool                 `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// PromptBlocksStorage Interface
// ---------------------------------------------------------------------------

// PromptBlocksStorage is the storage interface for the prompt blocks domain.
// It extends VersionedStorageDomain with prompt block-specific types.
//
// TODO: Replace `any` parameter/return types with concrete types from
// storage/types.go once ported (StoragePromptBlockType,
// StorageCreatePromptBlockInput, StorageUpdatePromptBlockInput, etc.).
type PromptBlocksStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Entity CRUD ---

	// GetByID retrieves a prompt block by ID.
	GetByID(ctx context.Context, id string) (any, error)

	// Create creates a new prompt block.
	Create(ctx context.Context, input any) (any, error)

	// Update updates an existing prompt block.
	Update(ctx context.Context, input any) (any, error)

	// Delete removes a prompt block by ID.
	Delete(ctx context.Context, id string) error

	// List lists prompt blocks with optional filtering.
	List(ctx context.Context, args any) (any, error)

	// --- Version Methods ---

	// CreateVersion creates a new prompt block version.
	CreateVersion(ctx context.Context, input CreatePromptBlockVersionInput) (*PromptBlockVersion, error)

	// GetVersion retrieves a version by its ID.
	GetVersion(ctx context.Context, id string) (*PromptBlockVersion, error)

	// GetVersionByNumber retrieves a version by block ID and version number.
	GetVersionByNumber(ctx context.Context, blockID string, versionNumber int) (*PromptBlockVersion, error)

	// GetLatestVersion retrieves the latest version for a prompt block.
	GetLatestVersion(ctx context.Context, blockID string) (*PromptBlockVersion, error)

	// ListVersions lists versions for a prompt block with pagination and sorting.
	ListVersions(ctx context.Context, input ListPromptBlockVersionsInput) (*ListPromptBlockVersionsOutput, error)

	// DeleteVersion removes a version by ID.
	DeleteVersion(ctx context.Context, id string) error

	// DeleteVersionsByParentID removes all versions for a prompt block.
	DeleteVersionsByParentID(ctx context.Context, blockID string) error

	// CountVersions returns the number of versions for a prompt block.
	CountVersions(ctx context.Context, blockID string) (int, error)

	// --- Resolution Methods ---

	// GetByIDResolved resolves an entity by merging its thin record with
	// the active or latest version config.
	GetByIDResolved(ctx context.Context, id string, status string) (any, error)

	// ListResolved lists entities with version resolution.
	ListResolved(ctx context.Context, args any) (any, error)
}
