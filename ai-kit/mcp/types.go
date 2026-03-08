// Ported from: packages/mcp/src/tool/types.ts
package mcp

import "encoding/json"

const LatestProtocolVersion = "2025-06-18"

var SupportedProtocolVersions = []string{
	LatestProtocolVersion,
	"2025-03-26",
	"2024-11-05",
}

// ToolMeta represents optional MCP tool metadata.
// Keys should follow MCP _meta key format specification.
type ToolMeta map[string]interface{}

// Configuration represents client or server implementation info.
type Configuration struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// BaseParams is the base parameter schema with optional _meta.
type BaseParams struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// Result is the base result schema (same structure as BaseParams).
type Result = BaseParams

// Request represents an MCP request with method and optional params.
type Request struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// RequestOptions provides options for a request.
type RequestOptions struct {
	Signal <-chan struct{} // context cancellation channel
}

// Notification is the same structure as Request in the MCP spec.
type Notification = Request

// ElicitationCapability represents the elicitation capability.
type ElicitationCapability struct {
	ApplyDefaults *bool `json:"applyDefaults,omitempty"`
}

// ServerCapabilities represents the capabilities advertised by the MCP server.
type ServerCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Logging      map[string]interface{} `json:"logging,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Tools        *ToolsCapability       `json:"tools,omitempty"`
	Elicitation  *ElicitationCapability `json:"elicitation,omitempty"`
}

// PromptsCapability represents prompts support in server capabilities.
type PromptsCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

// ResourcesCapability represents resources support in server capabilities.
type ResourcesCapability struct {
	Subscribe   *bool `json:"subscribe,omitempty"`
	ListChanged *bool `json:"listChanged,omitempty"`
}

// ToolsCapability represents tools support in server capabilities.
type ToolsCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

// ClientCapabilities represents the capabilities advertised by the MCP client.
type ClientCapabilities struct {
	Elicitation *ElicitationCapability `json:"elicitation,omitempty"`
}

// InitializeResult is the result of the initialize request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Configuration      `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// PaginatedRequestParams extends BaseParams with an optional cursor.
type PaginatedRequestParams struct {
	Cursor string                 `json:"cursor,omitempty"`
	Meta   map[string]interface{} `json:"_meta,omitempty"`
}

// MCPToolInputSchema represents the JSON Schema for tool input.
type MCPToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	// Additional schema fields are captured here.
	Extra map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON marshaling for MCPToolInputSchema
// to include Extra fields at the top level.
func (s MCPToolInputSchema) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["type"] = s.Type
	if s.Properties != nil {
		m["properties"] = s.Properties
	}
	for k, v := range s.Extra {
		if k != "type" && k != "properties" {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for MCPToolInputSchema.
func (s *MCPToolInputSchema) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if t, ok := m["type"].(string); ok {
		s.Type = t
	}
	if p, ok := m["properties"].(map[string]interface{}); ok {
		s.Properties = p
	}
	s.Extra = make(map[string]interface{})
	for k, v := range m {
		if k != "type" && k != "properties" {
			s.Extra[k] = v
		}
	}
	return nil
}

// MCPToolAnnotations represents optional annotations on a tool.
type MCPToolAnnotations struct {
	Title string                 `json:"title,omitempty"`
	Extra map[string]interface{} `json:"-"`
}

// MCPTool represents a tool definition from the MCP server.
type MCPTool struct {
	Name         string              `json:"name"`
	Title        string              `json:"title,omitempty"`
	Description  string              `json:"description,omitempty"`
	InputSchema  MCPToolInputSchema  `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
	Annotations  *MCPToolAnnotations `json:"annotations,omitempty"`
	Meta         ToolMeta            `json:"_meta,omitempty"`
}

// ListToolsResult is the result of tools/list.
type ListToolsResult struct {
	Tools      []MCPTool `json:"tools"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

// TextContent represents text content in tool results.
type TextContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// ImageContent represents image content in tool results.
type ImageContent struct {
	Type     string `json:"type"` // "image"
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

// ResourceContents represents a resource's contents.
type ResourceContents struct {
	URI      string `json:"uri"`
	Name     string `json:"name,omitempty"`
	Title    string `json:"title,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// EmbeddedResource represents an embedded resource in content.
type EmbeddedResource struct {
	Type     string           `json:"type"` // "resource"
	Resource ResourceContents `json:"resource"`
}

// ContentPart is a union type for text, image, or embedded resource content.
// Use the Type field to determine which fields are populated.
type ContentPart struct {
	Type string `json:"type"`
	// For type == "text"
	Text string `json:"text,omitempty"`
	// For type == "image"
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	// For type == "resource"
	Resource *ResourceContents `json:"resource,omitempty"`
}

// CallToolResult is the result of tools/call.
type CallToolResult struct {
	Content           []ContentPart  `json:"content,omitempty"`
	StructuredContent interface{}    `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
	ToolResult        interface{}    `json:"toolResult,omitempty"`
	Meta              map[string]interface{} `json:"_meta,omitempty"`
}

// MCPResource represents a resource exposed by the MCP server.
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Size        *int64 `json:"size,omitempty"`
}

// ListResourcesResult is the result of resources/list.
type ListResourcesResult struct {
	Resources  []MCPResource `json:"resources"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

// ReadResourceResult is the result of resources/read.
type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
	Meta     map[string]interface{} `json:"_meta,omitempty"`
}

// ResourceTemplate represents a resource template from the MCP server.
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourceTemplatesResult is the result of resources/templates/list.
type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	Meta              map[string]interface{} `json:"_meta,omitempty"`
}

// PromptArgument represents an argument for a prompt.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    *bool  `json:"required,omitempty"`
}

// MCPPrompt represents a prompt from the MCP server.
type MCPPrompt struct {
	Name        string           `json:"name"`
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// ListPromptsResult is the result of prompts/list.
type ListPromptsResult struct {
	Prompts    []MCPPrompt `json:"prompts"`
	NextCursor string      `json:"nextCursor,omitempty"`
}

// PromptMessage represents a message in a prompt result.
type PromptMessage struct {
	Role    string      `json:"role"` // "user" or "assistant"
	Content ContentPart `json:"content"`
}

// GetPromptResult is the result of prompts/get.
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
}

// ElicitationRequestParams represents the params for an elicitation/create request.
type ElicitationRequestParams struct {
	Message         string      `json:"message"`
	RequestedSchema interface{} `json:"requestedSchema"`
	Meta            map[string]interface{} `json:"_meta,omitempty"`
}

// ElicitationRequest represents an elicitation/create request from the server.
type ElicitationRequest struct {
	Method string                   `json:"method"` // "elicitation/create"
	Params ElicitationRequestParams `json:"params"`
}

// ElicitResult is the result of an elicitation request.
type ElicitResult struct {
	Action  string                 `json:"action"` // "accept", "decline", or "cancel"
	Content map[string]interface{} `json:"content,omitempty"`
	Meta    map[string]interface{} `json:"_meta,omitempty"`
}
