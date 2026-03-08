// Ported from: packages/ai/src/generate-text/tool-set.ts
package generatetext

// Tool represents a tool that can be called by the model.
// TODO: import from brainlink/experiments/ai-kit/providerutils once ported
type Tool struct {
	// Type discriminates between static ("function"), dynamic ("dynamic"), and provider ("provider") tools.
	Type string

	// Title is an optional human-readable title for the tool.
	Title string

	// InputSchema describes the JSON schema for the tool's input.
	// TODO: use proper schema type from providerutils
	InputSchema interface{}

	// Execute is the function to run when the tool is called.
	// It receives the parsed input and execution options, and returns an output or error.
	Execute func(input interface{}, options ToolExecuteOptions) (interface{}, error)

	// OnInputAvailable is called when tool input becomes available (before execution).
	OnInputAvailable func(options ToolInputAvailableOptions) error

	// OnInputStart is called when tool input streaming starts.
	OnInputStart func(options ToolInputStartOptions)

	// OnInputDelta is called for each incremental chunk of tool input.
	OnInputDelta func(options ToolInputDeltaOptions)

	// NeedsApproval indicates whether the tool requires approval before execution.
	// Can be a static bool or a function that evaluates dynamically.
	NeedsApproval interface{} // bool | func(input interface{}, options NeedsApprovalOptions) (bool, error)

	// SupportsDeferredResults indicates provider tools that may return results in a later turn.
	SupportsDeferredResults bool
}

// ToolExecuteOptions are options passed to a tool's Execute function.
type ToolExecuteOptions struct {
	ToolCallID          string
	Messages            []ModelMessage
	AbortSignal         <-chan struct{}
	ExperimentalContext interface{}
}

// ToolInputAvailableOptions are passed when tool input becomes available.
type ToolInputAvailableOptions struct {
	Input               interface{}
	ToolCallID          string
	Messages            []ModelMessage
	AbortSignal         <-chan struct{}
	ExperimentalContext interface{}
}

// ToolInputStartOptions are passed when tool input streaming starts.
type ToolInputStartOptions struct {
	ToolCallID string
}

// ToolInputDeltaOptions are passed for incremental tool input chunks.
type ToolInputDeltaOptions struct {
	ToolCallID string
	Delta      string
}

// NeedsApprovalOptions are passed to a dynamic NeedsApproval function.
type NeedsApprovalOptions struct {
	ToolCallID          string
	Messages            []ModelMessage
	ExperimentalContext interface{}
}

// ToolSet is a map of tool names to Tool definitions.
type ToolSet map[string]Tool
