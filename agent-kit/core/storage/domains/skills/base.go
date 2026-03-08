// Ported from: packages/core/src/storage/domains/skills/base.ts
package skills

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Skill Version Types
// ---------------------------------------------------------------------------

// SkillVersion represents a stored version of a skill's definition.
// Definition fields are top-level on the version row (no nested snapshot object).
type SkillVersion struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the skill this version belongs to.
	SkillID string `json:"skillId"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`

	// TODO: Embed StorageSkillSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// CreateSkillVersionInput is the input for creating a new skill version.
type CreateSkillVersionInput struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the skill this version belongs to.
	SkillID string `json:"skillId"`
	// Sequential version number.
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`

	// TODO: Embed StorageSkillSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// SkillVersionSortDirection is the sort direction for version listings.
type SkillVersionSortDirection string

const (
	SkillVersionSortASC  SkillVersionSortDirection = "ASC"
	SkillVersionSortDESC SkillVersionSortDirection = "DESC"
)

// SkillVersionOrderBy defines fields for ordering version listings.
type SkillVersionOrderBy string

const (
	SkillVersionOrderByVersionNumber SkillVersionOrderBy = "versionNumber"
	SkillVersionOrderByCreatedAt     SkillVersionOrderBy = "createdAt"
)

// ListSkillVersionsInput is the input for listing skill versions.
type ListSkillVersionsInput struct {
	// ID of the skill to list versions for.
	SkillID string `json:"skillId"`
	// Zero-indexed page number.
	Page *int `json:"page,omitempty"`
	// Number of items per page. Use -1 to fetch all records without limit.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderByField     *SkillVersionOrderBy      `json:"orderByField,omitempty"`
	OrderByDirection *SkillVersionSortDirection `json:"orderByDirection,omitempty"`
}

// ListSkillVersionsOutput is the output for listing skill versions.
type ListSkillVersionsOutput struct {
	Versions []SkillVersion `json:"versions"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PerPage  int            `json:"perPage"`
	HasMore  bool           `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// SkillsStorage Interface
// ---------------------------------------------------------------------------

// SkillsStorage is the storage interface for the skills domain.
// It extends VersionedStorageDomain with skill-specific types.
//
// TODO: Replace `any` parameter/return types with concrete types from
// storage/types.go once ported (StorageSkillType,
// StorageCreateSkillInput, StorageUpdateSkillInput, etc.).
type SkillsStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Entity CRUD ---

	// GetByID retrieves a skill by ID.
	GetByID(ctx context.Context, id string) (any, error)

	// Create creates a new skill.
	Create(ctx context.Context, input any) (any, error)

	// Update updates an existing skill.
	Update(ctx context.Context, input any) (any, error)

	// Delete removes a skill by ID.
	Delete(ctx context.Context, id string) error

	// List lists skills with optional filtering.
	List(ctx context.Context, args any) (any, error)

	// --- Version Methods ---

	// CreateVersion creates a new skill version.
	CreateVersion(ctx context.Context, input CreateSkillVersionInput) (*SkillVersion, error)

	// GetVersion retrieves a version by its ID.
	GetVersion(ctx context.Context, id string) (*SkillVersion, error)

	// GetVersionByNumber retrieves a version by skill ID and version number.
	GetVersionByNumber(ctx context.Context, skillID string, versionNumber int) (*SkillVersion, error)

	// GetLatestVersion retrieves the latest version for a skill.
	GetLatestVersion(ctx context.Context, skillID string) (*SkillVersion, error)

	// ListVersions lists versions for a skill with pagination and sorting.
	ListVersions(ctx context.Context, input ListSkillVersionsInput) (*ListSkillVersionsOutput, error)

	// DeleteVersion removes a version by ID.
	DeleteVersion(ctx context.Context, id string) error

	// DeleteVersionsByParentID removes all versions for a skill.
	DeleteVersionsByParentID(ctx context.Context, skillID string) error

	// CountVersions returns the number of versions for a skill.
	CountVersions(ctx context.Context, skillID string) (int, error)

	// --- Resolution Methods ---

	// GetByIDResolved resolves an entity by merging its thin record with
	// the active or latest version config.
	GetByIDResolved(ctx context.Context, id string, status string) (any, error)

	// ListResolved lists entities with version resolution.
	ListResolved(ctx context.Context, args any) (any, error)
}
