// Ported from: packages/core/src/tool-provider/types.ts
package toolprovider

import (
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/tools"
)

// ---------------------------------------------------------------------------
// Provider metadata
// ---------------------------------------------------------------------------

// ToolProviderInfo holds metadata about a tool provider.
type ToolProviderInfo struct {
	// ID is the unique identifier for this provider (e.g., "composio").
	ID string `json:"id"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// Description is a short description of the provider.
	Description string `json:"description,omitempty"`
}

// ---------------------------------------------------------------------------
// Toolkit (group of related tools)
// ---------------------------------------------------------------------------

// ToolProviderToolkit represents a toolkit (group of related tools) from a tool provider.
type ToolProviderToolkit struct {
	// Slug is the unique slug for this toolkit (e.g., "GITHUB", "SLACK").
	Slug string `json:"slug"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// Description of the toolkit.
	Description string `json:"description,omitempty"`
	// Icon is the icon URL or identifier.
	Icon string `json:"icon,omitempty"`
}

// ---------------------------------------------------------------------------
// Tool listing entry
// ---------------------------------------------------------------------------

// ToolProviderToolInfo is a tool listing entry from a tool provider.
// Used for UI discovery — does not include the full executable tool.
type ToolProviderToolInfo struct {
	// Slug is the unique slug for this tool (e.g., "GITHUB_CREATE_ISSUE").
	Slug string `json:"slug"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// Description of what this tool does.
	Description string `json:"description,omitempty"`
	// Toolkit is the toolkit this tool belongs to.
	Toolkit string `json:"toolkit,omitempty"`
}

// ---------------------------------------------------------------------------
// List options
// ---------------------------------------------------------------------------

// ListToolProviderToolsOptions specifies options for listing tools from a provider.
type ListToolProviderToolsOptions struct {
	// Toolkit filters by toolkit slug.
	Toolkit string `json:"toolkit,omitempty"`
	// Search is a query for filtering tools.
	Search string `json:"search,omitempty"`
	// Page is the pagination cursor or page number.
	Page *int `json:"page,omitempty"`
	// PerPage is the number of tools per page.
	PerPage *int `json:"perPage,omitempty"`
}

// ---------------------------------------------------------------------------
// Paginated results
// ---------------------------------------------------------------------------

// Pagination holds pagination metadata for list operations.
type Pagination struct {
	Total   *int `json:"total,omitempty"`
	Page    *int `json:"page,omitempty"`
	PerPage *int `json:"perPage,omitempty"`
	HasMore bool `json:"hasMore"`
}

// ToolProviderListResult is a paginated result from tool provider list operations.
// In TypeScript this is generic (ToolProviderListResult<T>); in Go we use concrete
// type aliases for each element type.
type ToolProviderListResult[T any] struct {
	Data       []T         `json:"data"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Convenience type aliases for common list results.
type (
	ToolkitListResult  = ToolProviderListResult[ToolProviderToolkit]
	ToolInfoListResult = ToolProviderListResult[ToolProviderToolInfo]
)

// ---------------------------------------------------------------------------
// Resolve options
// ---------------------------------------------------------------------------

// ResolveToolProviderToolsOptions specifies options for resolving executable tools
// at agent runtime.
type ResolveToolProviderToolsOptions struct {
	// UserID is the user ID for user-scoped tool execution (e.g., Composio).
	UserID string `json:"userId,omitempty"`
	// RequestContext holds per-request context (e.g., user-specific API keys, tenant IDs).
	RequestContext map[string]any `json:"requestContext,omitempty"`
	// Extra holds additional provider-specific options.
	// In TypeScript this was an index signature: [key: string]: unknown.
	Extra map[string]any `json:"extra,omitempty"`
}

// ---------------------------------------------------------------------------
// ToolProvider interface
// ---------------------------------------------------------------------------

// ToolProvider is the interface for tool providers (e.g., Composio) that supply
// tools to agents.
//
// Tool providers serve two purposes:
//  1. Discovery — UI uses ListToolkits(), ListTools(), GetToolSchema() to browse
//     available tools.
//  2. Runtime — Agent hydration uses ResolveTools() to get executable tools for
//     selected tool slugs.
type ToolProvider interface {
	// Info returns the provider metadata.
	Info() ToolProviderInfo

	// ListToolkits lists available toolkits from this provider.
	// Used by UI for browsing.
	// Implementations may return (ToolkitListResult{}, ErrNotSupported) if not applicable.
	ListToolkits() (ToolkitListResult, error)

	// ListTools lists available tools, optionally filtered by toolkit or search query.
	// Used by UI for browsing/selecting tools.
	ListTools(options *ListToolProviderToolsOptions) (ToolInfoListResult, error)

	// GetToolSchema returns the JSON schema for a specific tool's input.
	// Used by UI to display tool details.
	// Returns nil map when the tool is not found.
	GetToolSchema(toolSlug string) (map[string]any, error)

	// ResolveTools resolves executable tools for the given slugs.
	// Called during agent hydration to resolve integrationTools references.
	//
	// Parameters:
	//   - toolSlugs: slice of tool slugs to resolve.
	//   - toolConfigs: per-tool configuration (description overrides); may be nil.
	//   - options: provider-specific options (userId, requestContext, etc.); may be nil.
	//
	// Returns a map of tool ID to executable ToolAction.
	ResolveTools(
		toolSlugs []string,
		toolConfigs map[string]storage.StorageToolConfig,
		options *ResolveToolProviderToolsOptions,
	) (map[string]tools.ToolAction, error)
}
