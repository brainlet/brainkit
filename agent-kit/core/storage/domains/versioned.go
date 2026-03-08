// Ported from: packages/core/src/storage/domains/versioned.ts
package domains

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Generic Version Types
// ---------------------------------------------------------------------------

// VersionBase holds the metadata fields that exist on every version row.
type VersionBase struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`
}

// CreateVersionInputBase holds version creation input — same as VersionBase
// but without the server-assigned CreatedAt timestamp.
type CreateVersionInputBase struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
}

// VersionSortDirection is the sort direction for version listings.
// Alias for SortDirection (mirrors TS: VersionSortDirectionGeneric = ThreadSortDirection).
type VersionSortDirection = SortDirection

// VersionOrderBy defines the fields that can be used for ordering version listings.
type VersionOrderBy string

const (
	VersionOrderByVersionNumber VersionOrderBy = "versionNumber"
	VersionOrderByCreatedAt    VersionOrderBy = "createdAt"
)

// ListVersionsInputBase holds input for listing versions with pagination and sorting.
type ListVersionsInputBase struct {
	// Page number (0-indexed).
	Page *int `json:"page,omitempty"`
	// Number of items per page, or PerPageDisabled (-1) to fetch all records.
	// Defaults to 20 if nil.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderBy *VersionOrderByClause `json:"orderBy,omitempty"`
}

// VersionOrderByClause holds the sorting options for version listings.
type VersionOrderByClause struct {
	Field     VersionOrderBy     `json:"field,omitempty"`
	Direction VersionSortDirection `json:"direction,omitempty"`
}

// ListVersionsOutputBase holds output for listing versions with pagination info.
type ListVersionsOutputBase struct {
	// Array of versions for the current page (typed as any — callers cast).
	Versions []any `json:"versions"`
	// Total number of versions.
	Total int `json:"total"`
	// Current page number.
	Page int `json:"page"`
	// Items per page (-1 means disabled).
	PerPage int `json:"perPage"`
	// Whether there are more pages.
	HasMore bool `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// Entity base — the "thin record" must have these fields
// ---------------------------------------------------------------------------

// VersionedEntityBase is the minimum shape for a versioned entity.
type VersionedEntityBase struct {
	ID              string `json:"id"`
	ActiveVersionID string `json:"activeVersionId,omitempty"`
}

// ---------------------------------------------------------------------------
// Order-by / sort-direction types (from storage/types.ts)
// ---------------------------------------------------------------------------

// ThreadOrderBy defines the fields that can be used for ordering entity listings.
// TODO: Move to a shared storage types package once storage/types.ts is ported.
type ThreadOrderBy string

const (
	ThreadOrderByCreatedAt ThreadOrderBy = "createdAt"
	ThreadOrderByUpdatedAt ThreadOrderBy = "updatedAt"
)

// StorageOrderBy holds the sorting options for entity list queries.
type StorageOrderBy struct {
	Field     ThreadOrderBy `json:"field,omitempty"`
	Direction SortDirection `json:"direction,omitempty"`
}

// ---------------------------------------------------------------------------
// Validation sets (constants shared across all versioned domains)
// ---------------------------------------------------------------------------

var entityOrderBySet = map[ThreadOrderBy]bool{
	ThreadOrderByCreatedAt: true,
	ThreadOrderByUpdatedAt: true,
}

var sortDirectionSet = map[SortDirection]bool{
	SortASC:  true,
	SortDESC: true,
}

var versionOrderBySet = map[VersionOrderBy]bool{
	VersionOrderByVersionNumber: true,
	VersionOrderByCreatedAt:     true,
}

// ---------------------------------------------------------------------------
// VersionedStorageDomain — interface
// ---------------------------------------------------------------------------

// ResolveStatus controls which version is used when resolving an entity.
type ResolveStatus string

const (
	ResolveStatusDraft     ResolveStatus = "draft"
	ResolveStatusPublished ResolveStatus = "published"
	ResolveStatusArchived  ResolveStatus = "archived"
)

// ResolveOptions holds options for entity resolution.
type ResolveOptions struct {
	Status ResolveStatus
}

// VersionedStorageDomain is the interface for versioned storage domains
// (agents, prompt blocks, scorer definitions, etc.).
//
// In TypeScript this is a generic abstract class with 12 type parameters.
// Go does not support that level of generic complexity, so we use `any` for
// the domain-specific payload types. Concrete implementations should provide
// typed wrapper methods on top of this interface.
type VersionedStorageDomain interface {
	StorageDomain

	// --- Entity CRUD ---

	GetByID(ctx context.Context, id string) (any, error)
	Create(ctx context.Context, input any) (any, error)
	Update(ctx context.Context, input any) (any, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, args any) (any, error)

	// --- Version methods ---

	CreateVersion(ctx context.Context, input any) (any, error)
	GetVersion(ctx context.Context, id string) (any, error)
	GetVersionByNumber(ctx context.Context, entityID string, versionNumber int) (any, error)
	GetLatestVersion(ctx context.Context, entityID string) (any, error)
	ListVersions(ctx context.Context, input any) (any, error)
	DeleteVersion(ctx context.Context, id string) error
	DeleteVersionsByParentID(ctx context.Context, entityID string) error
	CountVersions(ctx context.Context, entityID string) (int, error)

	// --- Resolution methods ---

	GetByIDResolved(ctx context.Context, id string, opts *ResolveOptions) (any, error)
	ListResolved(ctx context.Context, args any) (any, error)
}

// ---------------------------------------------------------------------------
// VersionedStorageDomainBase — embeddable base with concrete helper methods
// ---------------------------------------------------------------------------

// VersionedStorageDomainBase provides the concrete resolution and helper
// methods from the TypeScript VersionedStorageDomain abstract class.
//
// Concrete domain implementations embed this struct and supply:
//   - ListKey: the key name used in list outputs (e.g. "agents").
//   - VersionMetadataFields: version metadata field names to strip when
//     extracting snapshot config.
//   - An implementation of VersionedStorageDomain (the abstract methods).
type VersionedStorageDomainBase struct {
	StorageDomainBase

	// ListKey is the key name used in list outputs (e.g. "agents", "promptBlocks").
	ListKey string

	// VersionMetadataFields is the set of version metadata field names
	// (including the FK field) to strip when extracting snapshot config.
	// e.g. ["id", "agentId", "versionNumber", "changedFields", "changeMessage", "createdAt"]
	VersionMetadataFields []string
}

// ExtractSnapshotConfig strips version metadata fields from a version row
// (represented as map[string]any), leaving only snapshot config fields.
func (v *VersionedStorageDomainBase) ExtractSnapshotConfig(version map[string]any) map[string]any {
	metadataSet := make(map[string]bool, len(v.VersionMetadataFields))
	for _, f := range v.VersionMetadataFields {
		metadataSet[f] = true
	}

	result := make(map[string]any)
	for key, val := range version {
		if !metadataSet[key] {
			result[key] = val
		}
	}
	return result
}

// ParseOrderBy validates and returns sanitized entity ordering parameters.
// Invalid values are replaced with defaults (field="createdAt", direction=defaultDirection).
func (v *VersionedStorageDomainBase) ParseOrderBy(orderBy *StorageOrderBy, defaultDirection SortDirection) ParsedOrderBy {
	if defaultDirection == "" {
		defaultDirection = SortDESC
	}

	field := ThreadOrderByCreatedAt
	direction := defaultDirection

	if orderBy != nil {
		if _, ok := entityOrderBySet[orderBy.Field]; ok {
			field = orderBy.Field
		}
		if _, ok := sortDirectionSet[orderBy.Direction]; ok {
			direction = orderBy.Direction
		}
	}

	return ParsedOrderBy{Field: field, Direction: direction}
}

// ParsedOrderBy holds the validated entity ordering parameters.
type ParsedOrderBy struct {
	Field     ThreadOrderBy `json:"field"`
	Direction SortDirection `json:"direction"`
}

// ParseVersionOrderBy validates and returns sanitized version ordering parameters.
// Invalid values are replaced with defaults (field="versionNumber", direction=defaultDirection).
func (v *VersionedStorageDomainBase) ParseVersionOrderBy(orderBy *VersionOrderByClause, defaultDirection VersionSortDirection) ParsedVersionOrderBy {
	if defaultDirection == "" {
		defaultDirection = SortDESC
	}

	field := VersionOrderByVersionNumber
	direction := defaultDirection

	if orderBy != nil {
		if _, ok := versionOrderBySet[orderBy.Field]; ok {
			field = orderBy.Field
		}
		if _, ok := sortDirectionSet[orderBy.Direction]; ok {
			direction = orderBy.Direction
		}
	}

	return ParsedVersionOrderBy{Field: field, Direction: direction}
}

// ParsedVersionOrderBy holds the validated version ordering parameters.
type ParsedVersionOrderBy struct {
	Field     VersionOrderBy       `json:"field"`
	Direction VersionSortDirection `json:"direction"`
}
