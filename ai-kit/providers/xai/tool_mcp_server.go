// Ported from: packages/xai/src/tool/mcp-server.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// McpServerInput is the input schema for the MCP server tool (empty, args are passed via ProviderTool.Args).
type McpServerInput struct{}

// McpServerOutput is the output of the MCP server tool.
type McpServerOutput struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    any    `json:"result"`
}

// McpServerArgs are the arguments for the MCP server tool.
type McpServerArgs struct {
	// ServerUrl is the URL of the MCP server.
	ServerUrl string `json:"serverUrl"`
	// ServerLabel is a label for the MCP server.
	ServerLabel *string `json:"serverLabel,omitempty"`
	// ServerDescription is a description of the MCP server.
	ServerDescription *string `json:"serverDescription,omitempty"`
	// AllowedTools is a list of allowed tool names.
	AllowedTools []string `json:"allowedTools,omitempty"`
	// Headers are custom headers to send.
	Headers map[string]string `json:"headers,omitempty"`
	// Authorization is the authorization header value.
	Authorization *string `json:"authorization,omitempty"`
}

// mcpServerToolFactory is the factory for the MCP server tool.
var mcpServerToolFactory = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[McpServerInput, McpServerOutput]{
		ID:           "xai.mcp",
		InputSchema:  &providerutils.Schema[McpServerInput]{},
		OutputSchema: &providerutils.Schema[McpServerOutput]{},
	},
)

// McpServer creates an MCP server provider tool.
func McpServer(opts providerutils.ProviderToolOptions[McpServerInput, McpServerOutput]) providerutils.ProviderTool[McpServerInput, McpServerOutput] {
	return mcpServerToolFactory(opts)
}
