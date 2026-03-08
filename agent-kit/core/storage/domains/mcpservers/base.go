// Ported from: packages/core/src/storage/domains/mcp-servers/base.ts
package mcpservers

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// MCP Server Version Types
// ---------------------------------------------------------------------------

// MCPServerVersion represents a stored version of an MCP server's content.
// Server fields are top-level on the version row (no nested snapshot object).
type MCPServerVersion struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the MCP server this version belongs to.
	MCPServerID string `json:"mcpServerId"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`

	// TODO: Embed StorageMCPServerSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// CreateMCPServerVersionInput is the input for creating a new MCP server version.
type CreateMCPServerVersionInput struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the MCP server this version belongs to.
	MCPServerID string `json:"mcpServerId"`
	// Sequential version number.
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`

	// TODO: Embed StorageMCPServerSnapshotType fields once storage/types.go is ported.
	Snapshot any `json:"snapshot,omitempty"`
}

// MCPServerVersionSortDirection is the sort direction for version listings.
type MCPServerVersionSortDirection string

const (
	MCPServerVersionSortASC  MCPServerVersionSortDirection = "ASC"
	MCPServerVersionSortDESC MCPServerVersionSortDirection = "DESC"
)

// MCPServerVersionOrderBy defines fields for ordering version listings.
type MCPServerVersionOrderBy string

const (
	MCPServerVersionOrderByVersionNumber MCPServerVersionOrderBy = "versionNumber"
	MCPServerVersionOrderByCreatedAt     MCPServerVersionOrderBy = "createdAt"
)

// ListMCPServerVersionsInput is the input for listing MCP server versions.
type ListMCPServerVersionsInput struct {
	// ID of the MCP server to list versions for.
	MCPServerID string `json:"mcpServerId"`
	// Zero-indexed page number.
	Page *int `json:"page,omitempty"`
	// Number of items per page. Use -1 to fetch all records without limit.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderByField     *MCPServerVersionOrderBy      `json:"orderByField,omitempty"`
	OrderByDirection *MCPServerVersionSortDirection `json:"orderByDirection,omitempty"`
}

// ListMCPServerVersionsOutput is the output for listing MCP server versions.
type ListMCPServerVersionsOutput struct {
	Versions []MCPServerVersion `json:"versions"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PerPage  int                `json:"perPage"`
	HasMore  bool               `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// MCPServersStorage Interface
// ---------------------------------------------------------------------------

// MCPServersStorage is the storage interface for the MCP servers domain.
// It extends VersionedStorageDomain with MCP server-specific types.
//
// TODO: Replace `any` parameter/return types with concrete types from
// storage/types.go once ported (StorageMCPServerType,
// StorageCreateMCPServerInput, StorageUpdateMCPServerInput, etc.).
type MCPServersStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Entity CRUD ---

	// GetByID retrieves an MCP server by ID.
	GetByID(ctx context.Context, id string) (any, error)

	// Create creates a new MCP server.
	Create(ctx context.Context, input any) (any, error)

	// Update updates an existing MCP server.
	Update(ctx context.Context, input any) (any, error)

	// Delete removes an MCP server by ID.
	Delete(ctx context.Context, id string) error

	// List lists MCP servers with optional filtering.
	List(ctx context.Context, args any) (any, error)

	// --- Version Methods ---

	// CreateVersion creates a new MCP server version.
	CreateVersion(ctx context.Context, input CreateMCPServerVersionInput) (*MCPServerVersion, error)

	// GetVersion retrieves a version by its ID.
	GetVersion(ctx context.Context, id string) (*MCPServerVersion, error)

	// GetVersionByNumber retrieves a version by MCP server ID and version number.
	GetVersionByNumber(ctx context.Context, mcpServerID string, versionNumber int) (*MCPServerVersion, error)

	// GetLatestVersion retrieves the latest version for an MCP server.
	GetLatestVersion(ctx context.Context, mcpServerID string) (*MCPServerVersion, error)

	// ListVersions lists versions for an MCP server with pagination and sorting.
	ListVersions(ctx context.Context, input ListMCPServerVersionsInput) (*ListMCPServerVersionsOutput, error)

	// DeleteVersion removes a version by ID.
	DeleteVersion(ctx context.Context, id string) error

	// DeleteVersionsByParentID removes all versions for an MCP server.
	DeleteVersionsByParentID(ctx context.Context, mcpServerID string) error

	// CountVersions returns the number of versions for an MCP server.
	CountVersions(ctx context.Context, mcpServerID string) (int, error)

	// --- Resolution Methods ---

	// GetByIDResolved resolves an entity by merging its thin record with
	// the active or latest version config.
	GetByIDResolved(ctx context.Context, id string, status string) (any, error)

	// ListResolved lists entities with version resolution.
	ListResolved(ctx context.Context, args any) (any, error)
}
