// Ported from: packages/core/src/mcp/types.ts
package mcp

import (
	"net/http"
)

// ---------------------------------------------------------------------------
// SSE / HTTP transport options
// ---------------------------------------------------------------------------

// MCPServerSSEOptionsBase holds common fields for SSE transport options.
type MCPServerSSEOptionsBase struct {
	// URL is the parsed URL of the incoming request.
	URL string

	// SSEPath is the path for establishing the SSE connection (e.g. "/sse").
	SSEPath string

	// MessagePath is the path for POSTing client messages (e.g. "/message").
	MessagePath string
}

// MCPServerSSEOptions holds options for starting an MCP server with SSE transport.
type MCPServerSSEOptions struct {
	MCPServerSSEOptionsBase

	// Req is the incoming HTTP request.
	Req *http.Request

	// Res is the HTTP response writer.
	Res http.ResponseWriter
}

// MCPServerHonoSSEOptions holds options for starting an MCP server with
// Hono SSE transport. In Go, Hono-specific context is not applicable;
// this is provided for 1:1 parity.
type MCPServerHonoSSEOptions struct {
	MCPServerSSEOptionsBase

	// Context is an opaque Hono context placeholder.
	// In Go, use the standard http.Request / http.ResponseWriter instead.
	Context any
}

// MCPServerHTTPOptions holds options for starting an MCP server with HTTP transport.
type MCPServerHTTPOptions struct {
	// URL is the parsed URL of the incoming request.
	URL string

	// HTTPPath is the path for establishing the HTTP connection (e.g. "/mcp").
	HTTPPath string

	// Req is the incoming HTTP request.
	Req *http.Request

	// Res is the HTTP response writer.
	Res http.ResponseWriter

	// Options holds optional transport-level options (e.g. sessionIdGenerator).
	Options any
}

// ---------------------------------------------------------------------------
// MCP Registry API Spec Types
// ---------------------------------------------------------------------------

// Repository describes a source code repository.
type Repository struct {
	// URL is the URL of the repository (e.g., a GitHub URL).
	URL string `json:"url"`

	// Source is the source control platform (e.g., "github", "gitlab").
	Source string `json:"source"`

	// ID is a unique identifier for the repository at the source.
	ID string `json:"id"`
}

// VersionDetail provides details about a specific version of an MCP server.
type VersionDetail struct {
	// Version is the semantic version string (e.g., "1.0.2").
	Version string `json:"version"`

	// ReleaseDate is the ISO 8601 date-time string when this version was released.
	ReleaseDate string `json:"release_date"`

	// IsLatest indicates if this version is the latest available.
	IsLatest bool `json:"is_latest"`
}

// ArgumentInfo is a base type for command-line arguments.
type ArgumentInfo struct {
	// Name is the name of the argument.
	Name string `json:"name"`

	// Description describes what the argument is for.
	Description string `json:"description"`

	// IsRequired indicates whether the argument is required.
	IsRequired bool `json:"is_required"`

	// IsRepeatable indicates whether the argument can be specified multiple times.
	IsRepeatable bool `json:"is_repeatable,omitempty"`

	// IsEditable indicates whether the argument's value can be edited by the user.
	IsEditable bool `json:"is_editable,omitempty"`

	// Choices is a list of predefined choices for the argument's value.
	Choices []string `json:"choices,omitempty"`

	// DefaultValue is the default value for the argument if not specified.
	DefaultValue any `json:"default_value,omitempty"`
}

// PositionalArgumentInfo describes a positional argument for a command.
type PositionalArgumentInfo struct {
	ArgumentInfo

	// Position is the 0-indexed position of the argument.
	Position int `json:"position"`
}

// NamedArgumentInfo describes a named argument (flag) for a command.
type NamedArgumentInfo struct {
	ArgumentInfo

	// ShortFlag is the short flag for the argument (e.g., "-y").
	ShortFlag string `json:"short_flag,omitempty"`

	// LongFlag is the long flag for the argument (e.g., "--yes").
	LongFlag string `json:"long_flag,omitempty"`

	// RequiresValue indicates whether the flag requires a value.
	RequiresValue bool `json:"requires_value,omitempty"`
}

// SubcommandInfo describes a subcommand for a command-line tool.
type SubcommandInfo struct {
	// Name is the name of the subcommand (e.g., "run", "list").
	Name string `json:"name"`

	// Description describes what the subcommand does.
	Description string `json:"description"`

	// IsRequired indicates whether this subcommand is required if its parent is used.
	IsRequired bool `json:"is_required,omitempty"`

	// Subcommands are nested subcommands.
	Subcommands []SubcommandInfo `json:"subcommands,omitempty"`

	// PositionalArguments are positional arguments for this subcommand.
	PositionalArguments []PositionalArgumentInfo `json:"positional_arguments,omitempty"`

	// NamedArguments are named arguments (flags) for this subcommand.
	NamedArguments []NamedArgumentInfo `json:"named_arguments,omitempty"`
}

// CommandInfo describes a command to run an MCP server package.
type CommandInfo struct {
	// Name is the primary command executable (e.g., "npx", "docker").
	Name string `json:"name"`

	// Subcommands to append to the primary command.
	Subcommands []SubcommandInfo `json:"subcommands,omitempty"`

	// PositionalArguments are positional arguments for the command.
	PositionalArguments []PositionalArgumentInfo `json:"positional_arguments,omitempty"`

	// NamedArguments are named arguments (flags) for the command.
	NamedArguments []NamedArgumentInfo `json:"named_arguments,omitempty"`
}

// EnvironmentVariableInfo describes an env var required or used by an MCP server package.
type EnvironmentVariableInfo struct {
	// Name is the name of the environment variable (e.g., "API_KEY").
	Name string `json:"name"`

	// Description describes what the environment variable is for.
	Description string `json:"description"`

	// Required indicates whether the environment variable is required.
	Required bool `json:"required,omitempty"`

	// DefaultValue is the default value for the env var if not set.
	DefaultValue string `json:"default_value,omitempty"`
}

// PackageInfo describes an installable package for an MCP server.
type PackageInfo struct {
	// RegistryName is the name of the package registry (e.g., "npm", "docker").
	RegistryName string `json:"registry_name"`

	// Name is the name of the package.
	Name string `json:"name"`

	// Version is the version of the package.
	Version string `json:"version"`

	// Command is the command structure to run this package as an MCP server.
	Command *CommandInfo `json:"command,omitempty"`

	// EnvironmentVariables are env vars relevant to this package.
	EnvironmentVariables []EnvironmentVariableInfo `json:"environment_variables,omitempty"`
}

// RemoteInfo describes a remote endpoint for accessing an MCP server.
type RemoteInfo struct {
	// TransportType is the transport type (e.g., "sse", "streamable").
	TransportType string `json:"transport_type"`

	// URL is the URL of the remote endpoint.
	URL string `json:"url"`
}

// ---------------------------------------------------------------------------
// MCPServerConfig
// ---------------------------------------------------------------------------

// MCPServerConfig holds configuration options for creating an MCPServer instance.
type MCPServerConfig struct {
	// Name is the display name of the MCP server.
	Name string `json:"name"`

	// Version is the semantic version of the MCP server.
	Version string `json:"version"`

	// Tools are the tools that this MCP server will expose.
	// TODO: use ToolsInput from the agent package once ported.
	Tools map[string]any `json:"tools,omitempty"`

	// Agents are optional Agent instances to be exposed as tools.
	// TODO: use Agent type from the agent package once ported.
	Agents map[string]any `json:"agents,omitempty"`

	// Workflows are optional Workflow instances to be exposed as tools.
	// TODO: use Workflow type from the workflows package once ported.
	Workflows map[string]any `json:"workflows,omitempty"`

	// ID is an optional unique identifier for the server.
	// If not provided, a UUID will be generated.
	ID string `json:"id,omitempty"`

	// Description is an optional description of the MCP server.
	Description string `json:"description,omitempty"`

	// Instructions describes how to use the server and its features.
	Instructions string `json:"instructions,omitempty"`

	// Repository is optional repository information for the server's source code.
	Repository *Repository `json:"repository,omitempty"`

	// ReleaseDate is the optional release date (ISO 8601 string).
	// Defaults to the time of instantiation if not provided.
	ReleaseDate string `json:"releaseDate,omitempty"`

	// IsLatest indicates if this is the latest version. Defaults to true.
	IsLatest *bool `json:"isLatest,omitempty"`

	// PackageCanonical is the optional canonical packaging format
	// (e.g., "npm", "docker", "pypi", "crates").
	PackageCanonical string `json:"packageCanonical,omitempty"`

	// Packages is an optional list of installable packages for this server.
	Packages []PackageInfo `json:"packages,omitempty"`

	// Remotes is an optional list of remote access points for this server.
	Remotes []RemoteInfo `json:"remotes,omitempty"`
}

// ---------------------------------------------------------------------------
// Server Information Structures
// ---------------------------------------------------------------------------

// ServerInfo contains basic information about an MCP server, conforming to
// the MCP Registry 'Server' schema.
type ServerInfo struct {
	// ID is the unique ID of the server.
	ID string `json:"id"`

	// Name is the name of the server.
	Name string `json:"name"`

	// Description is an optional description of the server.
	Description string `json:"description,omitempty"`

	// Repository is optional repository information.
	Repository *Repository `json:"repository,omitempty"`

	// VersionDetail contains detailed version information.
	VersionDetail VersionDetail `json:"version_detail"`
}

// ServerDetailInfo contains detailed information about an MCP server,
// conforming to the MCP Registry 'ServerDetail' schema.
type ServerDetailInfo struct {
	ServerInfo

	// PackageCanonical is the canonical packaging format, if applicable.
	PackageCanonical string `json:"package_canonical,omitempty"`

	// Packages contains information about installable packages.
	Packages []PackageInfo `json:"packages,omitempty"`

	// Remotes contains information about remote access points.
	Remotes []RemoteInfo `json:"remotes,omitempty"`
}

// ---------------------------------------------------------------------------
// MCPToolType
// ---------------------------------------------------------------------------

// MCPToolType represents the type of an MCP tool.
// Re-exported from the tools package in TypeScript.
type MCPToolType string

const (
	MCPToolTypeDefault MCPToolType = ""
	// Additional MCPToolType values would be added here as needed.
)
