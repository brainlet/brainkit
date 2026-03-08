// Ported from: packages/core/src/agent/workflows/prepare-stream/schema.ts
package workflows

import (
	"github.com/brainlet/brainkit/agent-kit/core/memory"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraBase is a stub for ../../../base.MastraBase.
// CIRCULAR: cannot import agent-kit base from agent/workflows because
// agent imports agent/workflows. Kept as local interface stub.
type MastraBase interface {
	Logger() any
}

// MastraLLMVNext is a stub for ../../../llm/model/model.loop.MastraLLMVNext.
// MISMATCH: real model.MastraLLMVNext is a struct (not interface). This local
// interface captures the Stream method signature needed by the workflow steps.
type MastraLLMVNext interface {
	Stream(args any) any
}

// Mastra is the narrow interface for the Mastra orchestrator used by agent workflows.
// core.Mastra satisfies this interface.
//
// Ported from: packages/core/src/agent/workflows — uses mastra.generateId()
type Mastra interface {
	GenerateID(ctx *IdGeneratorContext) string
}

// IdGeneratorContext is re-exported from the types package.
// Ported from: packages/core/src/types/dynamic-argument.ts — IdGeneratorContext
type IdGeneratorContext = aktypes.IdGeneratorContext

// InputProcessorOrWorkflow is a stub for ../../../processors.InputProcessorOrWorkflow.
// NOT DEFINED in processors package; real ProcessorRunner uses []any.
type InputProcessorOrWorkflow = any

// OutputProcessorOrWorkflow is a stub for ../../../processors.OutputProcessorOrWorkflow.
// NOT DEFINED in processors package; real ProcessorRunner uses []any.
type OutputProcessorOrWorkflow = any

// ProcessorState is a stub for ../../../processors.ProcessorState.
// MISMATCH: real processors.ProcessorState is a struct with sync.Mutex and
// private fields. This stub uses map[string]any as a simplified representation.
type ProcessorState = map[string]any

// RequestContext is re-exported from requestcontext (pointer to match agent package usage).
type RequestContext = *requestcontext.RequestContext

// Agent is a stub for ../../agent.Agent.
// CIRCULAR: cannot import parent agent package from agent/workflows sub-package.
// The agent/workflows sub-package does not call methods on Agent directly; it accesses
// agent capabilities via the AgentCapabilities struct. These methods provide minimum
// useful identification for pass-through scenarios.
type Agent interface {
	// GetID returns the agent's unique identifier.
	GetID() string
	// GetName returns the agent's display name.
	GetName() string
}

// MessageList is a stub for ../../message-list.MessageList.
// CIRCULAR: cannot import parent agent package from agent/workflows sub-package.
type MessageList = any

// StorageThreadType is re-exported from memory.
type StorageThreadType = memory.StorageThreadType

// AgentExecuteOnFinishOptions is a stub for ../../types.AgentExecuteOnFinishOptions.
// CIRCULAR: cannot import parent agent package from agent/workflows sub-package.
type AgentExecuteOnFinishOptions = any

// AgentCapabilities holds the resolved capabilities of an agent needed
// by the prepare-stream workflow steps.
type AgentCapabilities struct {
	AgentName           string
	Logger              any // IMastraLogger
	GetMemory           func() any
	GetModel            func(opts any) (any, error)
	GenerateMessageID   func() string
	AgentNetworkAppend  bool
	SaveStepMessages    func(args any) error
	ConvertTools        func(args any) (map[string]any, error)
	RunInputProcessors  func(args any) (InputProcessorsResult, error)
	ExecuteOnFinish     func(args AgentExecuteOnFinishOptions) error
	OutputProcessors    any // OutputProcessorOrWorkflow[] or function
	InputProcessors     any // InputProcessorOrWorkflow[] or function
	LLM                 MastraLLMVNext
}

// InputProcessorsResult holds the result from running input processors.
type InputProcessorsResult struct {
	Tripwire *TripwireData
}

// TripwireData holds data when an input processor triggers an abort.
type TripwireData struct {
	Reason      string `json:"reason"`
	Retry       bool   `json:"retry,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
	ProcessorID string `json:"processorId,omitempty"`
}

// CoreTool represents a converted tool in the schema format.
type CoreTool struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
	Execute     any    `json:"execute,omitempty"`
}

// PrepareToolsStepOutput is the output of the prepare-tools step.
type PrepareToolsStepOutput struct {
	ConvertedTools map[string]CoreTool
}

// PrepareMemoryStepOutput is the output of the prepare-memory step.
type PrepareMemoryStepOutput struct {
	ThreadExists    bool
	Thread          *StorageThreadType
	MessageList     MessageList
	ProcessorStates map[string]ProcessorState
	Tripwire        *TripwireData
}
