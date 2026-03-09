package interfaces

// Agent is the shared interface for agent instances, used by the mastra
// package to interact with agents without importing the agent package.
//
// This breaks the circular dependency: agent imports mastra (via
// RegisterMastra), and mastra needs to reference agents. By having both
// packages depend on this interface, neither needs to import the other
// directly for this type.
//
// Method signatures use `any` for parameters that would otherwise require
// importing agent-kit internal packages (logger, storage, workflows,
// workspace), keeping this package dependency-free. Consumers that need
// typed access should type-assert at the call site.
type Agent interface {
	// ID returns the agent's unique identifier.
	ID() string
	// Name returns the agent's display name.
	Name() string
	// SetLogger sets the logger instance on the agent.
	// The concrete value is logger.IMastraLogger.
	SetLogger(l any)
	// RegisterMastra registers the Mastra instance with the agent.
	// The concrete value is *mastra.Mastra.
	RegisterMastra(m any)
	// RegisterPrimitives registers framework primitives with the agent.
	// The concrete value is an AgentPrimitives-shaped struct.
	RegisterPrimitives(p any)
	// HasOwnWorkspace returns whether the agent has its own workspace.
	HasOwnWorkspace() bool
	// GetWorkspace returns the agent's workspace, or nil.
	GetWorkspace() any
	// Source returns the agent's source ("code" or "stored").
	Source() string
	// SetSource sets the agent's source.
	SetSource(s string)
	// ListScorers returns the scorers registered on this agent.
	// Returns map[string]*ScorerEntry.
	ListScorers() map[string]*ScorerEntry
	// GetConfiguredProcessorWorkflows returns processor workflows
	// configured on this agent.
	GetConfiguredProcessorWorkflows() []any
}

// AgentMethodType represents agent method types, shared between the agent
// and llm/model packages to avoid circular imports.
//
// This breaks the circular dependency: agent/types.go defines AgentMethodType
// and imports llm/model, while llm/model/model_method_from_agent.go needs
// AgentMethodType but cannot import agent. With this shared definition, both
// packages reference the same type from a common dependency.
type AgentMethodType string

const (
	AgentMethodGenerate       AgentMethodType = "generate"
	AgentMethodGenerateLegacy AgentMethodType = "generateLegacy"
	AgentMethodStream         AgentMethodType = "stream"
	AgentMethodStreamLegacy   AgentMethodType = "streamLegacy"
)
