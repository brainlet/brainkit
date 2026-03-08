// Ported from: packages/provider-utils/src/types/tool.ts
package providerutils

import "context"

// ToolExecutionOptions contains additional options sent into each tool call.
type ToolExecutionOptions struct {
	// ToolCallID is the ID of the tool call.
	ToolCallID string
	// Messages are the messages that were sent to the language model to initiate
	// the response that contained the tool call.
	Messages []ModelMessage
	// Ctx is an optional context for cancellation (maps to AbortSignal in TypeScript).
	Ctx context.Context
	// ExperimentalContext is user-defined context. Treat as immutable inside tools.
	ExperimentalContext interface{}
}

// Tool represents a tool that contains the description and the schema of the input
// that the tool expects. This enables the language model to generate the input.
type Tool[INPUT any, OUTPUT any] struct {
	// Type is the tool type: "function" (default), "dynamic", or "provider".
	Type string
	// ID is the tool ID (for provider tools). Must follow "provider.toolName" format.
	ID string
	// Description is an optional description of what the tool does.
	Description string
	// Title is an optional title of the tool.
	Title string
	// ProviderOptions contains additional provider-specific metadata.
	ToolProviderOptions ProviderOptions
	// InputSchema is the schema of the input that the tool expects.
	InputSchema *Schema[INPUT]
	// OutputSchema is the optional schema for validating tool output.
	OutputSchema *Schema[OUTPUT]
	// InputExamples is an optional list of input examples.
	InputExamples []INPUT
	// Execute is the function that runs the tool.
	Execute func(input INPUT, opts ToolExecutionOptions) (OUTPUT, error)
	// NeedsApproval indicates whether the tool needs approval before execution.
	// Can be a bool or a function.
	NeedsApproval interface{}
	// Strict mode setting for the tool.
	Strict bool
	// Args are additional provider-specific arguments (for provider tools).
	Args map[string]interface{}
	// SupportsDeferredResults indicates whether the tool supports deferred results.
	SupportsDeferredResults bool
}

// ExecuteTool executes a tool and yields results through a channel.
// It mirrors the TypeScript async generator, yielding preliminary results
// for async iterable executions and a final result at the end.
type ExecuteToolResult[OUTPUT any] struct {
	// Type is "preliminary" or "final".
	Type   string
	Output OUTPUT
}

// ExecuteToolFunc executes a tool and sends results to the returned channel.
func ExecuteToolFunc[INPUT any, OUTPUT any](
	execute func(INPUT, ToolExecutionOptions) (OUTPUT, error),
	input INPUT,
	opts ToolExecutionOptions,
) (<-chan ExecuteToolResult[OUTPUT], <-chan error) {
	results := make(chan ExecuteToolResult[OUTPUT])
	errs := make(chan error, 1)

	go func() {
		defer close(results)
		defer close(errs)

		output, err := execute(input, opts)
		if err != nil {
			errs <- err
			return
		}

		results <- ExecuteToolResult[OUTPUT]{
			Type:   "final",
			Output: output,
		}
	}()

	return results, errs
}

// DynamicTool creates a tool with type "dynamic".
type DynamicToolConfig struct {
	Description         string
	Title               string
	ToolProviderOptions ProviderOptions
	InputSchema         *Schema[interface{}]
	Execute             func(input interface{}, opts ToolExecutionOptions) (interface{}, error)
	NeedsApproval       interface{}
}

// NewDynamicTool creates a new dynamic tool.
func NewDynamicTool(config DynamicToolConfig) Tool[interface{}, interface{}] {
	return Tool[interface{}, interface{}]{
		Type:                "dynamic",
		Description:         config.Description,
		Title:               config.Title,
		ToolProviderOptions: config.ToolProviderOptions,
		InputSchema:         config.InputSchema,
		Execute:             config.Execute,
		NeedsApproval:       config.NeedsApproval,
	}
}
