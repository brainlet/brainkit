// Ported from: packages/openai/src/tool/mcp.ts
package openai

// MCPAllowedToolsFilter represents a filter object for MCP allowed tools.
type MCPAllowedToolsFilter struct {
	// ReadOnly indicates whether to allow only read-only tools.
	ReadOnly *bool `json:"readOnly,omitempty"`

	// ToolNames is an optional list of allowed tool names.
	ToolNames []string `json:"toolNames,omitempty"`
}

// MCPRequireApprovalNever represents the "never" approval configuration with exceptions.
type MCPRequireApprovalNever struct {
	// Never specifies tools that never require approval.
	Never *MCPRequireApprovalNeverConfig `json:"never,omitempty"`
}

// MCPRequireApprovalNeverConfig contains the tool names that never need approval.
type MCPRequireApprovalNeverConfig struct {
	// ToolNames are tools that don't require approval.
	ToolNames []string `json:"toolNames,omitempty"`
}

// MCPOutput is the output schema for the MCP tool.
type MCPOutput struct {
	// Type is always "call".
	Type string `json:"type"`

	// ServerLabel is the label of the MCP server.
	ServerLabel string `json:"serverLabel"`

	// Name is the name of the tool called.
	Name string `json:"name"`

	// Arguments is the JSON-encoded arguments string.
	Arguments string `json:"arguments"`

	// Output is the optional output string.
	Output *string `json:"output,omitempty"`

	// Error is the optional error value.
	Error interface{} `json:"error,omitempty"`
}

// MCPArgs contains configuration options for the MCP tool.
type MCPArgs struct {
	// ServerLabel is a label for this MCP server, used to identify it in tool calls.
	ServerLabel string `json:"serverLabel"`

	// AllowedTools is a list of allowed tool names or a filter object.
	// Can be []string or *MCPAllowedToolsFilter.
	AllowedTools interface{} `json:"allowedTools,omitempty"`

	// Authorization is an OAuth access token usable with the remote MCP server.
	Authorization string `json:"authorization,omitempty"`

	// ConnectorID is an identifier for a service connector.
	ConnectorID string `json:"connectorId,omitempty"`

	// Headers are optional HTTP headers to send to the MCP server.
	Headers map[string]string `json:"headers,omitempty"`

	// RequireApproval specifies which tools require approval before execution.
	// Can be "always", "never" (strings), or *MCPRequireApprovalNever.
	RequireApproval interface{} `json:"requireApproval,omitempty"`

	// ServerDescription is an optional description of the MCP server.
	ServerDescription string `json:"serverDescription,omitempty"`

	// ServerURL is the URL for the MCP server.
	// One of ServerURL or ConnectorID must be provided.
	ServerURL string `json:"serverUrl,omitempty"`
}

// MCPToolID is the provider tool ID for MCP.
const MCPToolID = "openai.mcp"

// NewMCPTool creates a provider tool configuration for the MCP tool.
func NewMCPTool(args MCPArgs) map[string]interface{} {
	result := map[string]interface{}{
		"type":        "provider",
		"id":          MCPToolID,
		"serverLabel": args.ServerLabel,
	}
	if args.AllowedTools != nil {
		result["allowedTools"] = args.AllowedTools
	}
	if args.Authorization != "" {
		result["authorization"] = args.Authorization
	}
	if args.ConnectorID != "" {
		result["connectorId"] = args.ConnectorID
	}
	if args.Headers != nil {
		result["headers"] = args.Headers
	}
	if args.RequireApproval != nil {
		result["requireApproval"] = args.RequireApproval
	}
	if args.ServerDescription != "" {
		result["serverDescription"] = args.ServerDescription
	}
	if args.ServerURL != "" {
		result["serverUrl"] = args.ServerURL
	}
	return result
}
