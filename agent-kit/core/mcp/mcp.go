// Ported from: packages/core/src/mcp/index.ts
package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// InternalCoreTool is a stub for the tools.InternalCoreTool type.
// TODO: import from tools package once ported.
type InternalCoreTool = any

// ToolAction is a stub for the tools.ToolAction type.
// TODO: import from tools package once ported.
type ToolAction = any

// ToolExecutionContext is a stub for the tools.ToolExecutionContext type.
// TODO: import from tools package once ported.
type ToolExecutionContext = any

// Mastra is the narrow interface for the mastra.Mastra type, defining only
// the methods used by the MCP package. core.Mastra satisfies this interface.
//
// Method signatures match the real *mastra.Mastra:
//   - AddTool/AddAgent/AddWorkflow do NOT return errors; they panic on nil
//     and silently skip duplicates (matching TS behavior where duplicates
//     are logged at debug level and skipped).
//   - GenerateID takes an optional context pointer (nil = random UUID).
//
// Ported from: packages/core/src/mcp/index.ts — uses mastra.addTool(),
// mastra.addAgent(), mastra.addWorkflow(), mastra.generateId()
type Mastra interface {
	GenerateID(ctx *IdGeneratorContext) string
	AddTool(tool ToolAction, key string)
	AddAgent(agent Agent, key string, options *AddPrimitiveOptions)
	AddWorkflow(workflow AnyWorkflow, key string)
}

// IdGeneratorContext provides optional context for deterministic ID generation.
// Matches core/mastra.IdGeneratorContext.
//
// Ported from: packages/core/src/types — IdGeneratorContext
type IdGeneratorContext struct {
	IDType   string
	Source   string
	ThreadID string
}

// Agent is the narrow interface for agent types passed to Mastra.AddAgent.
// Matches core/mastra.Agent.
type Agent interface {
	ID() string
}

// AnyWorkflow is the narrow interface for workflow types passed to Mastra.AddWorkflow.
// Matches core/mastra.AnyWorkflow.
type AnyWorkflow interface {
	ID() string
}

// AddPrimitiveOptions holds options for adding primitives (agents, scorers, etc.).
// Matches core/mastra.AddPrimitiveOptions.
type AddPrimitiveOptions struct {
	Source string // "code" | "stored"
}

// MastraError is a stub for the error.MastraError type.
// TODO: import from error package once ported.
type MastraError struct {
	ID      string
	Message string
}

func (e *MastraError) Error() string {
	return fmt.Sprintf("%s: %s", e.ID, e.Message)
}

// ---------------------------------------------------------------------------
// ToolInfo
// ---------------------------------------------------------------------------

// ToolInfo describes a single tool exposed by an MCP server.
type ToolInfo struct {
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	InputSchema  any         `json:"inputSchema"`
	OutputSchema any         `json:"outputSchema,omitempty"`
	ToolType     MCPToolType `json:"toolType,omitempty"`
}

// ToolListInfo holds a list of tools provided by an MCP server.
type ToolListInfo struct {
	Tools []ToolInfo `json:"tools"`
}

// ---------------------------------------------------------------------------
// MCPServerBase
// ---------------------------------------------------------------------------

// MCPServerBase is the abstract base for MCP server implementations.
// This provides a common interface and shared functionality for all MCP servers
// that can be registered with Mastra, including handling of server metadata.
//
// In Go, concrete implementations embed this struct and implement the
// MCPServer interface methods.
type MCPServerBase struct {
	// idWasSet tracks if the server ID has been definitively set.
	idWasSet bool

	// Name is the display name of the MCP server.
	Name string

	// Version is the semantic version of the MCP server.
	Version string

	// id is internal storage for the server's unique ID.
	id string

	// Description is a description of what the MCP server does.
	Description string

	// Instructions describes how to use the server and its features.
	Instructions string

	// Repository is repository information for the server's source code.
	Repository *Repository

	// ReleaseDate is the release date of this server version (ISO 8601 string).
	ReleaseDate string

	// IsLatest indicates if this version is the latest available.
	IsLatest bool

	// PackageCanonical is the canonical packaging format (e.g., "npm", "docker").
	PackageCanonical string

	// Packages contains information about installable packages for this server.
	Packages []PackageInfo

	// Remotes contains information about remote access points for this server.
	Remotes []RemoteInfo

	// ConvertedTools are the tools registered with and converted by this MCP server.
	ConvertedTools map[string]InternalCoreTool

	// MastraInstance is a reference to the Mastra instance if registered.
	MastraInstance Mastra

	// Agents are agent instances to be exposed as tools.
	Agents map[string]any

	// Workflows are workflow instances to be exposed as tools.
	Workflows map[string]any

	// OriginalTools is the original tools configuration for re-conversion.
	OriginalTools map[string]any
}

// NewMCPServerBase creates a new MCPServerBase with the given configuration.
func NewMCPServerBase(config MCPServerConfig) *MCPServerBase {
	s := &MCPServerBase{
		Name:             config.Name,
		Version:          config.Version,
		Description:      config.Description,
		Instructions:     config.Instructions,
		Repository:       config.Repository,
		PackageCanonical: config.PackageCanonical,
		Packages:         config.Packages,
		Remotes:          config.Remotes,
		Agents:           config.Agents,
		Workflows:        config.Workflows,
		OriginalTools:    config.Tools,
		ConvertedTools:   make(map[string]InternalCoreTool),
	}

	// Set ID
	if config.ID != "" {
		s.id = slugify(config.ID)
		s.idWasSet = true
	} else {
		s.id = uuid.New().String()
	}

	// Set release date
	if config.ReleaseDate != "" {
		s.ReleaseDate = config.ReleaseDate
	} else {
		s.ReleaseDate = time.Now().UTC().Format(time.RFC3339)
	}

	// Set isLatest (defaults to true)
	if config.IsLatest != nil {
		s.IsLatest = *config.IsLatest
	} else {
		s.IsLatest = true
	}

	return s
}

// ID returns the server's unique ID.
func (s *MCPServerBase) ID() string {
	return s.id
}

// SetID sets the server's unique ID. This method is typically called by Mastra
// when registering the server. It ensures the ID is set only once.
func (s *MCPServerBase) SetID(id string) {
	if s.idWasSet {
		return
	}
	s.id = id
	s.idWasSet = true
}

// Tools returns a read-only view of the registered tools.
func (s *MCPServerBase) Tools() map[string]InternalCoreTool {
	// Return a copy to prevent mutation
	result := make(map[string]InternalCoreTool, len(s.ConvertedTools))
	for k, v := range s.ConvertedTools {
		result[k] = v
	}
	return result
}

// RegisterMastra is the internal method used by Mastra to register itself
// with the server. It re-converts tools with the Mastra instance available,
// then auto-registers tools, agents, and workflows.
func (s *MCPServerBase) RegisterMastra(mastra Mastra, convertTools func(tools map[string]any, agents map[string]any, workflows map[string]any) map[string]InternalCoreTool) {
	s.MastraInstance = mastra

	// Re-convert tools now that we have the Mastra instance
	s.ConvertedTools = convertTools(s.OriginalTools, s.Agents, s.Workflows)

	// Auto-register tools with the Mastra instance
	if s.OriginalTools != nil {
		for key, tool := range s.OriginalTools {
			if tool == nil {
				continue
			}
			// Use tool's intrinsic ID to avoid collisions across MCP servers.
			// AddTool silently skips duplicates (matching TS behavior).
			toolKey := key
			if toolMap, ok := tool.(map[string]any); ok {
				if id, exists := toolMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						toolKey = idStr
					}
				}
			}
			mastra.AddTool(tool, toolKey)
		}
	}

	// Auto-register agents with the Mastra instance.
	// AddAgent silently skips duplicates (matching TS behavior).
	if s.Agents != nil {
		for key, agent := range s.Agents {
			if a, ok := agent.(Agent); ok {
				mastra.AddAgent(a, key, nil)
			}
		}
	}

	// Auto-register workflows with the Mastra instance.
	// AddWorkflow silently skips duplicates (matching TS behavior).
	if s.Workflows != nil {
		for key, workflow := range s.Workflows {
			if wf, ok := workflow.(AnyWorkflow); ok {
				mastra.AddWorkflow(wf, key)
			}
		}
	}
}

// MCPServer defines the interface that concrete MCP server implementations
// must satisfy. This corresponds to the abstract methods on MCPServerBase
// in TypeScript.
type MCPServer interface {
	// StartStdio starts the MCP server using stdio transport.
	StartStdio() error

	// StartSSE starts the MCP server using SSE transport.
	StartSSE(options MCPServerSSEOptions) error

	// StartHonoSSE starts the MCP server using Hono SSE transport.
	StartHonoSSE(options MCPServerHonoSSEOptions) (any, error)

	// StartHTTP starts the MCP server using HTTP transport.
	StartHTTP(options MCPServerHTTPOptions) error

	// Close closes the MCP server and all its connections.
	Close() error

	// GetServerInfo gets basic information about the server.
	GetServerInfo() ServerInfo

	// GetServerDetail gets detailed information about the server.
	GetServerDetail() ServerDetailInfo

	// GetToolListInfo gets a list of tools provided by this MCP server.
	GetToolListInfo() ToolListInfo

	// GetToolInfo gets information for a specific tool.
	GetToolInfo(toolID string) *ToolInfo

	// ExecuteTool executes a specific tool.
	ExecuteTool(toolID string, args any, executionContext *ToolExecutionCtx) (any, error)

	// ConvertTools converts and validates tool definitions.
	ConvertTools(tools map[string]any, agents map[string]any, workflows map[string]any) map[string]InternalCoreTool
}

// ToolExecutionCtx holds optional context for tool execution.
type ToolExecutionCtx struct {
	Messages   []any  `json:"messages,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// slugify is a simplified slugification that converts a string to a
// URL-friendly slug. This is a minimal port of @sindresorhus/slugify.
func slugify(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))

	// Replace non-alphanumeric characters with hyphens
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else if r == ' ' || r == '_' {
			result.WriteRune('-')
		}
	}

	// Collapse multiple hyphens
	slug := result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	return strings.Trim(slug, "-")
}
