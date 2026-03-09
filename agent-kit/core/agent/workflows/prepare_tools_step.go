// Ported from: packages/core/src/agent/workflows/prepare-stream/prepare-tools-step.ts
package workflows

import (
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/interfaces"
	"github.com/brainlet/brainkit/agent-kit/core/memory"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ---------------------------------------------------------------------------
// Re-exported types from ported packages
// ---------------------------------------------------------------------------

// Span is re-exported from observability/types.
type Span = obstypes.Span

// MastraMemory is re-exported from memory.
type MastraMemory = memory.MastraMemory

// InnerAgentExecutionOptions is a stub for ../../agent.types.InnerAgentExecutionOptions.
// CIRCULAR: cannot import parent agent package from agent/workflows sub-package.
type InnerAgentExecutionOptions struct {
	Toolsets                 map[string]any
	ClientTools              map[string]any
	Memory                   *MemoryWrapper
	OutputWriter             any
	AutoResumeSuspendedTools bool
	Delegation               any
	InputProcessors          any
	OutputProcessors         any
	Context                  []any
	System                   any
	Messages                 any
	ModelSettings            any
	ToolChoice               any
	StopWhen                 any
	MaxSteps                 *int
	ProviderOptions          any
	IncludeRawChunks         bool
	ActiveTools              any
	StructuredOutput         any
	MaxProcessorRetries      *int
	IsTaskComplete           any
	OnIterationComplete      any
	SavePerStep              bool
	OnStepFinish             any
	OnFinish                 any
	OnChunk                  any
	OnError                  any
	OnAbort                  any
	AbortSignal              any
	Scorers                  any
	PrepareStep              any
}

// MemoryWrapper wraps memory options.
// CIRCULAR: cannot import parent agent package. This type mirrors agent.AgentMemoryOption.
type MemoryWrapper struct {
	Options any
}

// AgentMethodType is the shared agent method type, defined in core/interfaces
// to break the circular dependency between agent and llm/model packages.
type AgentMethodType = interfaces.AgentMethodType

// PrepareToolsStepOptions holds options for creating a prepare-tools step.
type PrepareToolsStepOptions struct {
	Capabilities   AgentCapabilities
	Options        InnerAgentExecutionOptions
	ThreadFromArgs *StorageThreadType
	ResourceID     string
	RunID          string
	RequestContext RequestContext
	AgentSpan      Span
	MethodType     AgentMethodType
	Memory         MastraMemory
}

// CreatePrepareToolsStep creates the tool preparation step for the agent workflow.
// Ported from TS: createPrepareToolsStep({ capabilities, options, ... })
//
// This step resolves and converts all tools (agent tools, toolsets, client tools)
// into the format needed by the model loop.
func CreatePrepareToolsStep(opts PrepareToolsStepOptions) func() (*PrepareToolsStepOutput, error) {
	return func() (*PrepareToolsStepOutput, error) {
		// Log tool enhancements.
		var toolEnhancements []string
		if len(opts.Options.Toolsets) > 0 {
			toolEnhancements = append(toolEnhancements,
				fmt.Sprintf("toolsets present (%d tools)", len(opts.Options.Toolsets)))
		}
		if opts.Memory != nil && opts.ResourceID != "" {
			toolEnhancements = append(toolEnhancements, "memory and resourceId available")
		}

		if logger, ok := opts.Capabilities.Logger.(interface {
			Debug(msg string, fields ...any)
		}); ok {
			enhStr := ""
			for i, e := range toolEnhancements {
				if i > 0 {
					enhStr += ", "
				}
				enhStr += e
			}
			logger.Debug(fmt.Sprintf("[Agent:%s] - Enhancing tools: %s", opts.Capabilities.AgentName, enhStr),
				"runId", opts.RunID,
			)
		}

		var threadID string
		if opts.ThreadFromArgs != nil {
			threadID = opts.ThreadFromArgs.ID
		}

		// Call capabilities.ConvertTools with all resolved options.
		convertedTools := make(map[string]CoreTool)

		if opts.Capabilities.ConvertTools != nil {
			var memoryConfigVal any
			if opts.Options.Memory != nil {
				memoryConfigVal = opts.Options.Memory.Options
			}

			result, err := opts.Capabilities.ConvertTools(map[string]any{
				"toolsets":                opts.Options.Toolsets,
				"clientTools":             opts.Options.ClientTools,
				"threadId":               threadID,
				"resourceId":             opts.ResourceID,
				"runId":                  opts.RunID,
				"requestContext":         opts.RequestContext,
				"methodType":            opts.MethodType,
				"outputWriter":          opts.Options.OutputWriter,
				"memoryConfig":          memoryConfigVal,
				"autoResumeSuspendedTools": opts.Options.AutoResumeSuspendedTools,
				"delegation":            opts.Options.Delegation,
			})
			if err != nil {
				return nil, err
			}
			for k, v := range result {
				if ct, ok := v.(CoreTool); ok {
					convertedTools[k] = ct
				}
			}
		}

		return &PrepareToolsStepOutput{
			ConvertedTools: convertedTools,
		}, nil
	}
}
