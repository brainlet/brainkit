// Ported from: packages/core/src/storage/domains/mcp-clients/base.ts
package mcpclients

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// MCP Client Version Types
// ---------------------------------------------------------------------------

// MCPClientVersion represents a stored version of an MCP client's content.
// Client fields are top-level on the version row (no nested snapshot object).
type MCPClientVersion struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the MCP client this version belongs to.
	MCPClientID string `json:"mcpClientId"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`

	// TODO: Embed StorageMCPClientSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// CreateMCPClientVersionInput is the input for creating a new MCP client version.
type CreateMCPClientVersionInput struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the MCP client this version belongs to.
	MCPClientID string `json:"mcpClientId"`
	// Sequential version number.
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`

	// TODO: Embed StorageMCPClientSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// MCPClientVersionSortDirection is the sort direction for version listings.
type MCPClientVersionSortDirection string

const (
	MCPClientVersionSortASC  MCPClientVersionSortDirection = "ASC"
	MCPClientVersionSortDESC MCPClientVersionSortDirection = "DESC"
)

// MCPClientVersionOrderBy defines fields for ordering version listings.
type MCPClientVersionOrderBy string

const (
	MCPClientVersionOrderByVersionNumber MCPClientVersionOrderBy = "versionNumber"
	MCPClientVersionOrderByCreatedAt     MCPClientVersionOrderBy = "createdAt"
)

// ListMCPClientVersionsInput is the input for listing MCP client versions.
type ListMCPClientVersionsInput struct {
	// ID of the MCP client to list versions for.
	MCPClientID string `json:"mcpClientId"`
	// Zero-indexed page number.
	Page *int `json:"page,omitempty"`
	// Number of items per page. Use -1 to fetch all records without limit.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderByField     *MCPClientVersionOrderBy      `json:"orderByField,omitempty"`
	OrderByDirection *MCPClientVersionSortDirection `json:"orderByDirection,omitempty"`
}

// ListMCPClientVersionsOutput is the output for listing MCP client versions.
type ListMCPClientVersionsOutput struct {
	Versions []MCPClientVersion `json:"versions"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PerPage  int                `json:"perPage"`
	HasMore  bool               `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// MCPClientsStorage Interface
// ---------------------------------------------------------------------------

// MCPClientsStorage is the storage interface for the MCP clients domain.
// It extends VersionedStorageDomain with MCP client-specific types.
//
// TODO: Replace `any` parameter/return types with concrete types from
// storage/types.go once ported (StorageMCPClientType,
// StorageCreateMCPClientInput, StorageUpdateMCPClientInput, etc.).
type MCPClientsStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Entity CRUD ---

	// GetByID retrieves an MCP client by ID.
	GetByID(ctx context.Context, id string) (any, error)

	// Create creates a new MCP client.
	Create(ctx context.Context, input any) (any, error)

	// Update updates an existing MCP client.
	Update(ctx context.Context, input any) (any, error)

	// Delete removes an MCP client by ID.
	Delete(ctx context.Context, id string) error

	// List lists MCP clients with optional filtering.
	List(ctx context.Context, args any) (any, error)

	// --- Version Methods ---

	// CreateVersion creates a new MCP client version.
	CreateVersion(ctx context.Context, input CreateMCPClientVersionInput) (*MCPClientVersion, error)

	// GetVersion retrieves a version by its ID.
	GetVersion(ctx context.Context, id string) (*MCPClientVersion, error)

	// GetVersionByNumber retrieves a version by MCP client ID and version number.
	GetVersionByNumber(ctx context.Context, mcpClientID string, versionNumber int) (*MCPClientVersion, error)

	// GetLatestVersion retrieves the latest version for an MCP client.
	GetLatestVersion(ctx context.Context, mcpClientID string) (*MCPClientVersion, error)

	// ListVersions lists versions for an MCP client with pagination and sorting.
	ListVersions(ctx context.Context, input ListMCPClientVersionsInput) (*ListMCPClientVersionsOutput, error)

	// DeleteVersion removes a version by ID.
	DeleteVersion(ctx context.Context, id string) error

	// DeleteVersionsByParentID removes all versions for an MCP client.
	DeleteVersionsByParentID(ctx context.Context, mcpClientID string) error

	// CountVersions returns the number of versions for an MCP client.
	CountVersions(ctx context.Context, mcpClientID string) (int, error)

	// --- Resolution Methods ---

	// GetByIDResolved resolves an entity by merging its thin record with
	// the active or latest version config.
	GetByIDResolved(ctx context.Context, id string, status string) (any, error)

	// ListResolved lists entities with version resolution.
	ListResolved(ctx context.Context, args any) (any, error)
}
