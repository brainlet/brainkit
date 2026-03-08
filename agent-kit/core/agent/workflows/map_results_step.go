// Ported from: packages/core/src/agent/workflows/prepare-stream/map-results-step.ts
package workflows

import "fmt"

// MapResultsStepOptions holds options for creating the map-results step.
type MapResultsStepOptions struct {
	Capabilities   AgentCapabilities
	Options        InnerAgentExecutionOptions
	ResourceID     string
	RunID          string
	RequestContext RequestContext
	Memory         MastraMemory
	MemoryConfig   MemoryConfig
	AgentSpan      Span
	AgentID        string
	MethodType     AgentMethodType
}

// ModelLoopStreamArgs holds the assembled arguments for the model loop.
// MISMATCH: real model.ModelLoopStreamArgs has different field types and structure:
//   - Tools: model uses ToolSet (map[string]ToolDefinition), this uses map[string]CoreTool
//   - Temperature: model has no Temperature field (it's in ModelSettings)
//   - Thread: model has no Thread/*StorageThreadType field
//   - OnFinish/OnStepFinish: model uses *ModelLoopStreamOptions, this has inline callbacks
//   - Options: model uses *ModelLoopStreamOptions, this uses map[string]any
//   - MaxSteps: model uses int, this uses *int
//   - ProcessorStates: model uses any, this uses map[string]ProcessorState
// This is an agent-workflow-specific aggregation type, not the same as the model loop type.
type ModelLoopStreamArgs struct {
	AgentID             string
	Tools               map[string]CoreTool
	RunID               string
	Temperature         *float64
	ToolChoice          any
	Thread              *StorageThreadType
	ThreadID            string
	ResourceID          string
	RequestContext      RequestContext
	MessageList         MessageList
	MethodType          any
	StopWhen            any
	MaxSteps            *int
	ProviderOptions     any
	IncludeRawChunks    bool
	ActiveTools         any
	StructuredOutput    any
	InputProcessors     any
	OutputProcessors    any
	ModelSettings       map[string]any
	MaxProcessorRetries *int
	IsTaskComplete      any
	OnIterationComplete any
	ProcessorStates     map[string]ProcessorState
	Tripwire            *TripwireData
	// Callbacks
	OnFinish     func(payload any)
	OnStepFinish func(props any) error
	OnChunk      any
	OnError      any
	OnAbort      any
	AbortSignal  any
	// Options sub-struct for prepareStep, onFinish, onStepFinish, etc.
	Options map[string]any
}

// CreateMapResultsStep creates the step that maps the parallel results from
// prepare-tools and prepare-memory into the ModelLoopStreamArgs format
// consumed by the stream step.
//
// This step also handles:
// - Tripwire detection and early bail (return early if input processor triggered abort)
// - Output/input processor resolution (overrides vs capabilities)
// - onStepFinish callback wiring (per-step memory persistence)
// - onFinish callback wiring (final memory persistence and scoring)
// - StructuredOutput processor creation
//
// Ported from TS: createMapResultsStep()
func CreateMapResultsStep(opts MapResultsStepOptions) func(toolsOutput *PrepareToolsStepOutput, memoryOutput *PrepareMemoryStepOutput) (*ModelLoopStreamArgs, error) {
	return func(toolsOutput *PrepareToolsStepOutput, memoryOutput *PrepareMemoryStepOutput) (*ModelLoopStreamArgs, error) {
		var threadCreatedByStep bool

		// Build initial result by spreading options + specific fields
		result := &ModelLoopStreamArgs{
			AgentID:    opts.AgentID,
			RunID:      opts.RunID,
			ResourceID: opts.ResourceID,
			ModelSettings: map[string]any{
				"temperature": float64(0),
			},
		}

		// Merge tools from prepare-tools step
		if toolsOutput != nil {
			tools := make(map[string]CoreTool)
			for k, v := range toolsOutput.ConvertedTools {
				tools[k] = v
			}
			result.Tools = tools
		}

		// Merge memory data from prepare-memory step
		if memoryOutput != nil {
			result.Thread = memoryOutput.Thread
			if memoryOutput.Thread != nil {
				result.ThreadID = memoryOutput.Thread.ID
			}
			result.MessageList = memoryOutput.MessageList
			result.ProcessorStates = memoryOutput.ProcessorStates

			// Propagate tripwire if present
			if memoryOutput.Tripwire != nil {
				result.Tripwire = memoryOutput.Tripwire
			}
		}

		// Merge options fields
		result.RequestContext = opts.RequestContext
		// Temperature from modelSettings
		if modelSettings, ok := opts.Options.ModelSettings.(map[string]any); ok {
			if temp, ok := modelSettings["temperature"]; ok {
				if t, ok := temp.(float64); ok {
					result.Temperature = &t
				}
			}
			// Merge all model settings
			for k, v := range modelSettings {
				result.ModelSettings[k] = v
			}
		}
		result.ToolChoice = opts.Options.ToolChoice
		result.StopWhen = opts.Options.StopWhen
		result.MaxSteps = opts.Options.MaxSteps
		result.ProviderOptions = opts.Options.ProviderOptions
		result.ActiveTools = opts.Options.ActiveTools
		result.StructuredOutput = opts.Options.StructuredOutput
		result.MaxProcessorRetries = opts.Options.MaxProcessorRetries
		result.IsTaskComplete = opts.Options.IsTaskComplete
		result.OnIterationComplete = opts.Options.OnIterationComplete
		result.IncludeRawChunks = opts.Options.IncludeRawChunks
		result.OnChunk = opts.Options.OnChunk
		result.OnError = opts.Options.OnError
		result.OnAbort = opts.Options.OnAbort
		result.AbortSignal = opts.Options.AbortSignal

		// Wire onStepFinish callback for per-step memory persistence
		result.OnStepFinish = func(props any) error {
			// Per-step memory save if savePerStep is enabled and memory is not read-only
			if opts.Options.SavePerStep {
				readOnly := false
				if mc, ok := opts.MemoryConfig.(map[string]any); ok {
					if ro, ok := mc["readOnly"].(bool); ok {
						readOnly = ro
					}
				}

				if !readOnly && memoryOutput != nil {
					if !memoryOutput.ThreadExists && !threadCreatedByStep && opts.Memory != nil && memoryOutput.Thread != nil {
						// TODO: create thread via memory.CreateThread({
						//   threadId: memoryOutput.Thread.ID,
						//   title: memoryOutput.Thread.Title,
						//   metadata: memoryOutput.Thread.Metadata,
						//   resourceId: memoryOutput.Thread.ResourceID,
						//   memoryConfig: opts.MemoryConfig,
						// })
						threadCreatedByStep = true
					}

					// Save step messages
					if opts.Capabilities.SaveStepMessages != nil {
						if err := opts.Capabilities.SaveStepMessages(map[string]any{
							"result":      props,
							"messageList": memoryOutput.MessageList,
							"runId":       opts.RunID,
						}); err != nil {
							// Log but don't fail on step save errors
							if logger, ok := opts.Capabilities.Logger.(interface {
								Error(msg string, fields ...any)
							}); ok {
								logger.Error("Error saving step messages", "error", err, "runId", opts.RunID)
							}
						}
					}
				}
			}

			// Call user-provided onStepFinish if set
			if opts.Options.OnStepFinish != nil {
				if fn, ok := opts.Options.OnStepFinish.(func(any) error); ok {
					return fn(props)
				}
			}
			return nil
		}

		// Check for tripwire and return early if triggered
		if result.Tripwire != nil {
			// In TS, this calls bail() with getModelOutputForTripwire().
			// bail() is a workflow primitive that short-circuits the workflow.
			// For the Go port, we still build the result but the caller (stream step)
			// should check for Tripwire and handle accordingly.
			//
			// TODO: implement full tripwire handling once model/tripwire packages are ported:
			//   agentModel, err := opts.Capabilities.GetModel(map[string]any{"requestContext": result.RequestContext})
			//   if !isSupportedLanguageModel(agentModel) {
			//       return nil, fmt.Errorf("MAP_RESULTS_STEP_UNSUPPORTED_MODEL: Tripwire handling requires a v2/v3 model")
			//   }
			//   modelOutput := getModelOutputForTripwire(tripwire, runId, observabilityContext, options, model, messageList)
			//   return bail(modelOutput)

			if logger, ok := opts.Capabilities.Logger.(interface {
				Debug(msg string, fields ...any)
			}); ok {
				logger.Debug(
					fmt.Sprintf("[Agent:%s] - Tripwire triggered, bailing early", opts.Capabilities.AgentName),
					"runId", opts.RunID,
					"reason", result.Tripwire.Reason,
				)
			}
			// Return with tripwire set - caller handles the bail
			return result, nil
		}

		// Resolve output processors - overrides replace user-configured but auto-derived (memory) are kept.
		// In TS:
		//   if capabilities.outputProcessors is a function, call it with requestContext + overrides
		//   else use options.outputProcessors || capabilities.outputProcessors
		var effectiveOutputProcessors any
		if opts.Capabilities.OutputProcessors != nil {
			if fn, ok := opts.Capabilities.OutputProcessors.(func(args any) (any, error)); ok {
				resolved, err := fn(map[string]any{
					"requestContext": result.RequestContext,
					"overrides":      opts.Options.OutputProcessors,
				})
				if err != nil {
					return nil, err
				}
				effectiveOutputProcessors = resolved
			} else {
				if opts.Options.OutputProcessors != nil {
					effectiveOutputProcessors = opts.Options.OutputProcessors
				} else {
					effectiveOutputProcessors = opts.Capabilities.OutputProcessors
				}
			}
		} else {
			effectiveOutputProcessors = opts.Options.OutputProcessors
		}

		// Handle structuredOutput option by creating a StructuredOutputProcessor.
		// Only create the processor if a model is explicitly provided.
		// TODO: implement once StructuredOutputProcessor is ported:
		//   if opts.Options.StructuredOutput != nil {
		//       if so, ok := opts.Options.StructuredOutput.(map[string]any); ok {
		//           if so["model"] != nil {
		//               structuredProcessor := NewStructuredOutputProcessor(so, opts.Capabilities.Logger)
		//               effectiveOutputProcessors = append(effectiveOutputProcessors, structuredProcessor)
		//           }
		//       }
		//   }

		// Resolve input processors - overrides replace user-configured but auto-derived (memory, skills) are kept.
		var effectiveInputProcessors any
		if opts.Capabilities.InputProcessors != nil {
			if fn, ok := opts.Capabilities.InputProcessors.(func(args any) (any, error)); ok {
				resolved, err := fn(map[string]any{
					"requestContext": result.RequestContext,
					"overrides":      opts.Options.InputProcessors,
				})
				if err != nil {
					return nil, err
				}
				effectiveInputProcessors = resolved
			} else {
				if opts.Options.InputProcessors != nil {
					effectiveInputProcessors = opts.Options.InputProcessors
				} else {
					effectiveInputProcessors = opts.Capabilities.InputProcessors
				}
			}
		} else {
			effectiveInputProcessors = opts.Options.InputProcessors
		}

		// Get model method type from agent method
		// NOT A TYPE STUB: getModelMethodFromAgentMethod is a function in llm/model that
		// maps AgentMethodType to model.ModelMethodType. The function is not exported or
		// doesn't exist yet in the Go port. The identity assignment works because both
		// types are string-based.
		var modelMethodType any = opts.MethodType

		// Build the final loop options
		result.MethodType = modelMethodType
		result.InputProcessors = effectiveInputProcessors
		result.OutputProcessors = effectiveOutputProcessors

		// Wire onFinish callback for final memory persistence and scoring
		result.OnFinish = func(payload any) {
			payloadMap, _ := payload.(map[string]any)

			// Check for error finish reason
			if payloadMap != nil {
				if finishReason, ok := payloadMap["finishReason"].(string); ok && finishReason == "error" {
					provider, _ := payloadMap["provider"].(string)
					modelID, _ := payloadMap["modelId"].(string)
					payloadErr, _ := payloadMap["error"]

					// TODO: check APICallError.isInstance for upstream vs generic error
					if logger, ok := opts.Capabilities.Logger.(interface {
						Error(msg string, fields ...any)
					}); ok {
						providerInfo := ""
						if provider != "" {
							providerInfo = fmt.Sprintf(" from %s", provider)
						}
						modelInfo := ""
						if modelID != "" {
							modelInfo = fmt.Sprintf(" (model: %s)", modelID)
						}
						logger.Error(
							fmt.Sprintf("Upstream LLM API error%s%s", providerInfo, modelInfo),
							"error", payloadErr,
							"runId", opts.RunID,
						)
					}
					return
				}
			}

			// Skip memory persistence when abort signal has fired
			// TODO: check opts.Options.AbortSignal for abort state
			aborted := false
			if opts.Options.AbortSignal != nil {
				// TODO: check actual abort state via context.Context
			}

			if !aborted {
				// Execute on-finish memory persistence and scoring
				if opts.Capabilities.ExecuteOnFinish != nil {
					// TODO: build outputText from messageList.get.all.core().map(m => m.content).join('\n')
					var outputText string

					if err := opts.Capabilities.ExecuteOnFinish(map[string]any{
						"result":           payload,
						"outputText":       outputText,
						"thread":           result.Thread,
						"threadId":         result.ThreadID,
						"readOnlyMemory":   false, // TODO: from memoryConfig.readOnly
						"resourceId":       opts.ResourceID,
						"memoryConfig":     opts.MemoryConfig,
						"requestContext":   opts.RequestContext,
						"agentSpan":        opts.AgentSpan,
						"runId":            opts.RunID,
						"messageList":      result.MessageList,
						"threadExists":     memoryOutput != nil && memoryOutput.ThreadExists,
						"structuredOutput": opts.Options.StructuredOutput != nil,
						"overrideScorers":  opts.Options.Scorers,
					}); err != nil {
						if logger, ok := opts.Capabilities.Logger.(interface {
							Error(msg string, fields ...any)
						}); ok {
							logger.Error("Error saving memory on finish", "error", err, "runId", opts.RunID)
						}
					}
				}
			}

			// Call user-provided onFinish if set
			if opts.Options.OnFinish != nil {
				if fn, ok := opts.Options.OnFinish.(func(any)); ok {
					fn(payload)
				}
			}
		}

		// Build Options sub-map for prepareStep
		optionsMap := make(map[string]any)
		if opts.Options.PrepareStep != nil {
			optionsMap["prepareStep"] = opts.Options.PrepareStep
		}
		optionsMap["onFinish"] = result.OnFinish
		optionsMap["onStepFinish"] = result.OnStepFinish
		optionsMap["onChunk"] = opts.Options.OnChunk
		optionsMap["onError"] = opts.Options.OnError
		optionsMap["onAbort"] = opts.Options.OnAbort
		optionsMap["abortSignal"] = opts.Options.AbortSignal
		result.Options = optionsMap

		return result, nil
	}
}
