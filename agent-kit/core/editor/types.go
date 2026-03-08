// Package editor provides the editor interface for managing agents, prompts,
// scorers, and MCP configs from stored data.
//
// Ported from: packages/core/src/editor/types.ts
package editor

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent"
	"github.com/brainlet/brainkit/agent-kit/core/evals"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/mastra"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/toolprovider"
)

// MastraEditorConfig holds configuration for the Mastra editor.
//
// Ported from: packages/core/src/editor/types.ts — MastraEditorConfig
type MastraEditorConfig struct {
	// Logger is the optional logger instance.
	Logger logger.IMastraLogger

	// ToolProviders holds tool providers for integration tools (e.g., Composio).
	ToolProviders map[string]toolprovider.ToolProvider
}

// GetByIDOptions specifies version retrieval options.
//
// Ported from: packages/core/src/editor/types.ts — GetByIdOptions
type GetByIDOptions struct {
	// VersionID retrieves a specific version by ID.
	VersionID string

	// VersionNumber retrieves a specific version by number.
	VersionNumber int
}

// ============================================================================
// Agent Namespace Interface
// ============================================================================

// IEditorAgentNamespace manages agent CRUD operations from stored data.
//
// Ported from: packages/core/src/editor/types.ts — IEditorAgentNamespace
type IEditorAgentNamespace interface {
	// Create creates a new agent from storage input.
	Create(input storage.StorageCreateAgentInput) (*agent.Agent, error)

	// GetByID retrieves an agent by ID, optionally at a specific version.
	GetByID(id string, options *GetByIDOptions) (*agent.Agent, error)

	// Update updates an existing agent.
	Update(input storage.StorageUpdateAgentInput) (*agent.Agent, error)

	// Delete deletes an agent by ID.
	Delete(id string) error

	// List lists thin agent records with optional filtering.
	List(args *storage.StorageListAgentsInput) (*storage.StorageListAgentsOutput, error)

	// ListResolved lists resolved agents with version snapshots.
	ListResolved(args *storage.StorageListAgentsInput) (*storage.StorageListAgentsResolvedOutput, error)

	// ClearCache clears the agent cache, optionally for a specific agent ID.
	ClearCache(agentID string)

	// Clone clones an existing agent with new identity and optional metadata.
	Clone(a *agent.Agent, options CloneAgentOptions) (*storage.StorageResolvedAgentType, error)
}

// CloneAgentOptions holds options for cloning an agent.
//
// Ported from: packages/core/src/editor/types.ts — clone() options parameter
type CloneAgentOptions struct {
	NewID          string
	NewName        string
	Metadata       map[string]any
	AuthorID       string
	RequestContext *requestcontext.RequestContext
}

// ============================================================================
// Prompt Namespace Interface
// ============================================================================

// IEditorPromptNamespace manages prompt block CRUD operations from stored data.
//
// Ported from: packages/core/src/editor/types.ts — IEditorPromptNamespace
type IEditorPromptNamespace interface {
	// Create creates a new prompt block.
	Create(input storage.StorageCreatePromptBlockInput) (*storage.StorageResolvedPromptBlockType, error)

	// GetByID retrieves a prompt block by ID, optionally at a specific version.
	GetByID(id string, options *GetByIDOptions) (*storage.StorageResolvedPromptBlockType, error)

	// Update updates an existing prompt block.
	Update(input storage.StorageUpdatePromptBlockInput) (*storage.StorageResolvedPromptBlockType, error)

	// Delete deletes a prompt block by ID.
	Delete(id string) error

	// List lists thin prompt block records with optional filtering.
	List(args *storage.StorageListPromptBlocksInput) (*storage.StorageListPromptBlocksOutput, error)

	// ListResolved lists resolved prompt blocks with version snapshots.
	ListResolved(args *storage.StorageListPromptBlocksInput) (*storage.StorageListPromptBlocksResolvedOutput, error)

	// ClearCache clears the prompt block cache, optionally for a specific ID.
	ClearCache(id string)

	// Preview renders a set of instruction blocks with the given context variables.
	Preview(blocks []storage.AgentInstructionBlock, context map[string]any) (string, error)
}

// ============================================================================
// Scorer Namespace Interface
// ============================================================================

// IEditorScorerNamespace manages scorer definition CRUD operations from stored data.
//
// Ported from: packages/core/src/editor/types.ts — IEditorScorerNamespace
type IEditorScorerNamespace interface {
	// Create creates a new scorer definition.
	Create(input storage.StorageCreateScorerDefinitionInput) (*storage.StorageResolvedScorerDefinitionType, error)

	// GetByID retrieves a scorer definition by ID, optionally at a specific version.
	GetByID(id string, options *GetByIDOptions) (*storage.StorageResolvedScorerDefinitionType, error)

	// Update updates an existing scorer definition.
	Update(input storage.StorageUpdateScorerDefinitionInput) (*storage.StorageResolvedScorerDefinitionType, error)

	// Delete deletes a scorer definition by ID.
	Delete(id string) error

	// List lists thin scorer records with optional filtering.
	List(args *storage.StorageListScorerDefinitionsInput) (*storage.StorageListScorerDefinitionsOutput, error)

	// ListResolved lists resolved scorers with version snapshots.
	ListResolved(args *storage.StorageListScorerDefinitionsInput) (*storage.StorageListScorerDefinitionsResolvedOutput, error)

	// ClearCache clears the scorer cache, optionally for a specific ID.
	ClearCache(id string)

	// Resolve resolves a stored scorer definition into a runnable MastraScorer.
	// Returns nil if the scorer cannot be resolved.
	Resolve(storedScorer *storage.StorageResolvedScorerDefinitionType) *evals.MastraScorer
}

// ============================================================================
// MCP Config Namespace Interface
// ============================================================================

// IEditorMCPNamespace manages MCP client config CRUD operations from stored data.
//
// Ported from: packages/core/src/editor/types.ts — IEditorMCPNamespace
type IEditorMCPNamespace interface {
	// Create creates a new MCP client config.
	Create(input storage.StorageCreateMCPClientInput) (*storage.StorageResolvedMCPClientType, error)

	// GetByID retrieves an MCP client config by ID, optionally at a specific version.
	GetByID(id string, options *GetByIDOptions) (*storage.StorageResolvedMCPClientType, error)

	// Update updates an existing MCP client config.
	Update(input storage.StorageUpdateMCPClientInput) (*storage.StorageResolvedMCPClientType, error)

	// Delete deletes an MCP client config by ID.
	Delete(id string) error

	// List lists thin MCP client records with optional filtering.
	List(args *storage.StorageListMCPClientsInput) (*storage.StorageListMCPClientsOutput, error)

	// ListResolved lists resolved MCP clients with version snapshots.
	ListResolved(args *storage.StorageListMCPClientsInput) (*storage.StorageListMCPClientsResolvedOutput, error)

	// ClearCache clears the MCP client cache, optionally for a specific ID.
	ClearCache(id string)
}

// ============================================================================
// Main Editor Interface
// ============================================================================

// IMastraEditor is the interface for the Mastra Editor, which handles agent,
// prompt, scorer, and MCP config management from stored data.
//
// Ported from: packages/core/src/editor/types.ts — IMastraEditor
type IMastraEditor interface {
	// RegisterWithMastra registers this editor with a Mastra instance.
	// This gives the editor access to Mastra's storage, tools, workflows, etc.
	RegisterWithMastra(m *mastra.Mastra)

	// Agent returns the agent management namespace.
	Agent() IEditorAgentNamespace

	// MCP returns the MCP config management namespace.
	MCP() IEditorMCPNamespace

	// Prompt returns the prompt block management namespace.
	Prompt() IEditorPromptNamespace

	// Scorer returns the scorer definition management namespace.
	Scorer() IEditorScorerNamespace

	// GetToolProvider retrieves a registered tool provider by ID.
	GetToolProvider(id string) (toolprovider.ToolProvider, bool)

	// GetToolProviders returns all registered tool providers.
	GetToolProviders() map[string]toolprovider.ToolProvider
}
