// Ported from: packages/core/src/agent/workflows/prepare-stream/stream-step.ts
package workflows

import "fmt"

// ---------------------------------------------------------------------------
// Additional stub types
// ---------------------------------------------------------------------------

// SaveQueueManager is a stub for ../../save-queue.SaveQueueManager.
// Real savequeue.SaveQueueManager is a struct with private fields (mu, logger, debounceMs, etc.).
// Kept as = any because this type is only passed opaquely in a map[string]any to LLM.Stream().
// Wiring to *savequeue.SaveQueueManager would constrain callers without benefit.
type SaveQueueManager = any

// Workspace is a stub for ../../../workspace/workspace.Workspace.
// Real workspace.Workspace is a struct with 10+ fields and methods.
// Kept as = any because this type is only passed opaquely in a map[string]any to LLM.Stream().
// Wiring to *workspace.Workspace would constrain callers without benefit.
type Workspace = any

// MastraModelOutput is a stub for ../../../stream/base.MastraModelOutput.
// Real stream/base.MastraModelOutput is a struct with private fields (mu, channels, processors).
// Kept as = any because LLM.Stream() returns any, and this is used as the return type.
// Wiring would require LLM.Stream() to return *base.MastraModelOutput, breaking the interface.
type MastraModelOutput = any

// StreamStepOptions holds options for creating the stream step.
type StreamStepOptions struct {
	Capabilities            AgentCapabilities
	RunID                   string
	ReturnScorerData        bool
	RequireToolApproval     bool
	ToolCallConcurrency     int
	ResumeContext           *ResumeContext
	AgentID                 string
	AgentName               string
	ToolCallID              string
	MethodType              AgentMethodType
	SaveQueueManager        SaveQueueManager
	MemoryConfig            MemoryConfig
	Memory                  MastraMemory
	ResourceID              string
	AutoResumeSuspendedTools bool
	Workspace               Workspace
}

// ResumeContext holds context for resuming a suspended tool execution.
type ResumeContext struct {
	ResumeData any
	Snapshot   any
}

// CreateStreamStep creates the streaming step for the agent workflow.
// This step takes the prepared tools and memory context, resolves output processors,
// and initiates the LLM streaming loop.
//
// Ported from TS: createStreamStep()
func CreateStreamStep(opts StreamStepOptions) func(inputData any) (MastraModelOutput, error) {
	return func(inputData any) (MastraModelOutput, error) {
		if logger, ok := opts.Capabilities.Logger.(interface {
			Debug(msg string, fields ...any)
		}); ok {
			logger.Debug(
				fmt.Sprintf("Starting agent %s llm stream call", opts.Capabilities.AgentName),
				"runId", opts.RunID,
			)
		}

		// Cast inputData to ModelLoopStreamArgs
		validatedInputData, ok := inputData.(*ModelLoopStreamArgs)
		if !ok {
			return nil, fmt.Errorf("stream step: inputData is not *ModelLoopStreamArgs")
		}

		// Resolve output processors.
		// Use processors from input data if available, otherwise fall back to capabilities.
		var processors any
		if validatedInputData.OutputProcessors != nil {
			processors = validatedInputData.OutputProcessors
		} else if opts.Capabilities.OutputProcessors != nil {
			if fn, ok := opts.Capabilities.OutputProcessors.(func(args any) (any, error)); ok {
				resolved, err := fn(map[string]any{
					"requestContext": validatedInputData.RequestContext,
				})
				if err != nil {
					return nil, fmt.Errorf("stream step: failed to resolve output processors: %w", err)
				}
				processors = resolved
			} else {
				processors = opts.Capabilities.OutputProcessors
			}
		}

		// Get model method type from agent method type
		// NOT A TYPE STUB: getModelMethodFromAgentMethod is a function in llm/model that
		// maps AgentMethodType to model.ModelMethodType. The function is not exported or
		// doesn't exist yet in the Go port. The identity assignment works because both
		// types are string-based.
		var modelMethodType any = opts.MethodType

		// Call capabilities.LLM.Stream with all resolved options.
		// In TS: capabilities.llm.stream({ ...validatedInputData, outputProcessors, ... })
		if opts.Capabilities.LLM == nil {
			return nil, fmt.Errorf("stream step: capabilities.LLM is nil")
		}

		streamResult := opts.Capabilities.LLM.Stream(map[string]any{
			// Spread validated input data fields
			"agentId":             validatedInputData.AgentID,
			"tools":               validatedInputData.Tools,
			"runId":               validatedInputData.RunID,
			"temperature":         validatedInputData.Temperature,
			"toolChoice":          validatedInputData.ToolChoice,
			"thread":              validatedInputData.Thread,
			"threadId":            validatedInputData.ThreadID,
			"resourceId":          validatedInputData.ResourceID,
			"requestContext":      validatedInputData.RequestContext,
			"messageList":         validatedInputData.MessageList,
			"methodType":          modelMethodType,
			"stopWhen":            validatedInputData.StopWhen,
			"maxSteps":            validatedInputData.MaxSteps,
			"providerOptions":     validatedInputData.ProviderOptions,
			"includeRawChunks":    validatedInputData.IncludeRawChunks,
			"activeTools":         validatedInputData.ActiveTools,
			"structuredOutput":    validatedInputData.StructuredOutput,
			"inputProcessors":     validatedInputData.InputProcessors,
			"modelSettings":       validatedInputData.ModelSettings,
			"maxProcessorRetries": validatedInputData.MaxProcessorRetries,
			"isTaskComplete":      validatedInputData.IsTaskComplete,
			"onIterationComplete": validatedInputData.OnIterationComplete,
			"processorStates":     validatedInputData.ProcessorStates,
			"options":             validatedInputData.Options,
			// Stream step specific overrides
			"outputProcessors":        processors,
			"returnScorerData":        opts.ReturnScorerData,
			"requireToolApproval":     opts.RequireToolApproval,
			"toolCallConcurrency":     opts.ToolCallConcurrency,
			"resumeContext":           opts.ResumeContext,
			"autoResumeSuspendedTools": opts.AutoResumeSuspendedTools,
			"workspace":              opts.Workspace,
			"agentName":              opts.AgentName,
			"toolCallId":             opts.ToolCallID,
			// Internal options for memory persistence
			"_internal": map[string]any{
				"generateId":       opts.Capabilities.GenerateMessageID,
				"saveQueueManager": opts.SaveQueueManager,
				"memoryConfig":     opts.MemoryConfig,
				"threadId":         validatedInputData.ThreadID,
				"resourceId":       opts.ResourceID,
				"memory":           opts.Memory,
			},
		})

		return streamResult, nil
	}
}
