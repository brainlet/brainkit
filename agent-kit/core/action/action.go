// Ported from: packages/core/src/action/index.ts
package action

import (
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/vector"
)

// ---------------------------------------------------------------------------
// Stub type references for unported packages
// ---------------------------------------------------------------------------

// AgentRef represents an agent in the Mastra system.
// Defined here (not imported from agent package) to break circular dependency:
// agent imports action, so action cannot import agent.
// This interface captures the minimal contract needed by MastraPrimitives.Agents.
// core.Agent interface (in core/mastra.go) and agent.Agent struct both satisfy this.
type AgentRef interface {
	// ID returns the agent's unique identifier.
	ID() string
	// Name returns the agent's display name.
	Name() string
	// SetLogger sets the logger on the agent.
	SetLogger(l logger.IMastraLogger)
}

// MastraRef represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports action (for MastraPrimitives/IAction), so action cannot import core.
// core.Mastra struct satisfies this interface.
type MastraRef interface {
	// GetLogger returns the configured logger instance.
	GetLogger() logger.IMastraLogger
	// GetStorage returns the composite storage provider.
	GetStorage() *storage.MastraCompositeStore
}

// MastraMemoryRef represents a memory instance in the Mastra system.
// Defined here (not imported from memory package) to break circular dependency:
// memory imports action indirectly through other packages.
// memory.MastraMemoryBase and any MastraMemory implementation satisfy this.
type MastraMemoryRef interface {
	// ID returns the memory instance's unique identifier.
	ID() string
}

// MastraTTSRef represents a text-to-speech provider in the Mastra system.
// Defined here (not imported from tts package) to break circular dependency:
// tts imports action indirectly through agent/core packages.
// tts.MastraTTS struct satisfies this via its embedded MastraBase.
type MastraTTSRef interface {
	// SetLogger sets the logger on the TTS provider.
	SetLogger(l logger.IMastraLogger)
}

// SchemaWithValidation is a stub for the SchemaWithValidation type from ../stream.
// TODO: import from stream package when ported.
type SchemaWithValidation interface{}

// ---------------------------------------------------------------------------
// MastraPrimitives
// ---------------------------------------------------------------------------

// MastraPrimitives holds references to core Mastra services.
// All fields are optional (pointer/nil-able) matching the TS partial type.
//
// TS: export type MastraPrimitives = {
//
//	logger?: IMastraLogger;
//	storage?: MastraCompositeStore;
//	agents?: Record<string, Agent>;
//	tts?: Record<string, MastraTTS>;
//	vectors?: Record<string, MastraVector>;
//	memory?: MastraMemory;
//	};
type MastraPrimitives struct {
	Logger  logger.IMastraLogger
	Storage *storage.MastraCompositeStore
	Agents  map[string]AgentRef
	TTS     map[string]MastraTTSRef
	Vectors map[string]vector.MastraVector
	Memory  MastraMemoryRef
}

// ---------------------------------------------------------------------------
// MastraUnion
// ---------------------------------------------------------------------------

// MastraUnion represents the union of all Mastra instance keys and MastraPrimitives.
// In TypeScript this is: { [K in keyof Mastra]: Mastra[K] } & MastraPrimitives
//
// Since the Mastra type is not yet ported, this embeds MastraPrimitives and
// adds a Mastra field for forward compatibility.
type MastraUnion struct {
	MastraPrimitives
	Mastra MastraRef
}

// ---------------------------------------------------------------------------
// IExecutionContext
// ---------------------------------------------------------------------------

// IExecutionContext represents the execution context passed to an action's
// execute function.
//
// TS: export interface IExecutionContext<TInput> {
//
//	context: TInput;
//	runId?: string;
//	threadId?: string;
//	resourceId?: string;
//	memory?: MastraMemory;
//	};
//
// The TInput generic is replaced with any; callers type-assert as needed.
type IExecutionContext struct {
	Context    any
	RunID      string
	ThreadID   string
	ResourceID string
	Memory     MastraMemoryRef
}

// ---------------------------------------------------------------------------
// ExecuteFunc
// ---------------------------------------------------------------------------

// ExecuteFunc is the function signature for an action's execute method.
//
// TS: execute?: (context: TContext, options?: TOptions) => Promise<TOutput>;
//
// In Go the generics are replaced with any; callers type-assert as needed.
// The error return replaces the Promise rejection semantics.
type ExecuteFunc func(ctx *IExecutionContext, options any) (any, error)

// ---------------------------------------------------------------------------
// IAction
// ---------------------------------------------------------------------------

// IAction represents an action specification with an optional execute function.
//
// TS: export interface IAction<TId, TInput, TOutput, TContext, TOptions> {
//
//	id: TId;
//	description?: string;
//	inputSchema?: SchemaWithValidation<TInput>;
//	outputSchema?: SchemaWithValidation<TOutput>;
//	execute?: (context: TContext, options?: TOptions) => Promise<TOutput>;
//	};
//
// The execute field is optional because ITools extends IAction and tools may
// need execute to be optional when forwarding tool calls to the client or to
// a queue instead of executing them in the same process.
type IAction struct {
	ID           string
	Description  string
	InputSchema  SchemaWithValidation
	OutputSchema SchemaWithValidation
	Execute      ExecuteFunc
}
