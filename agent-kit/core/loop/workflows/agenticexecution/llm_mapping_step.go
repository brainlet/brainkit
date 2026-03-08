// Ported from: packages/core/src/loop/workflows/agentic-execution/llm-mapping-step.ts
package agenticexecution

import (
	"time"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported (llm-mapping-specific)
// ---------------------------------------------------------------------------

// ProcessorRunner is a stub for ../../../processors/runner.ProcessorRunner.
// Stub: real ProcessorRunner has unexported fields (logger, agentName, processorStates sync.Map),
// constructor NewProcessorRunner(ProcessorRunnerConfig), and different ProcessPart signature
// (7 typed params including logger.IMastraLogger, observability context types). Stub uses exported
// fields and simplified ProcessPart with all-any params. Shape mismatch in fields and methods.
type ProcessorRunner struct {
	InputProcessors  []any
	OutputProcessors []any
	Logger           IMastraLogger
	AgentName        string
	ProcessorStates  map[string]any
}

// ProcessPart runs a chunk through output processors.
func (pr *ProcessorRunner) ProcessPart(
	chunk any,
	processorStates any,
	observabilityContext any,
	requestContext any,
	messageList any,
	stepNumber int,
	writer any,
) *ProcessPartResult {
	return &ProcessPartResult{Part: chunk}
}

// ProcessPartResult holds the result of processing a single part.
type ProcessPartResult struct {
	Part            any
	Blocked         bool
	Reason          string
	TripwireOptions *TripWireOptions
	ProcessorID     string
}

// ProcessorStreamWriter is a stub for ../../../processors.ProcessorStreamWriter.
// Stub: real ProcessorStreamWriter is an interface{ Custom(DataChunkType) error } where
// DataChunkType is a typed struct. Stub is a struct with Custom func(any) error.
// Interface vs struct mismatch; DataChunkType vs any param mismatch.
type ProcessorStreamWriter struct {
	Custom func(data any) error
}

// SanitizeToolName is a stub for ../../../agent/messagelist/utils/tool_name.SanitizeToolName.
// Stub: real function has same signature (string) → string but applies regex-based sanitization
// (replaces non-alphanumeric chars). Stub is a no-op passthrough. No import cycle exists
// but importing agent/messagelist/utils would add a deep dependency for a single utility function.
func SanitizeToolName(name string) string {
	return name
}

// ---------------------------------------------------------------------------
// ToolCallOutput
// ---------------------------------------------------------------------------

// ToolCallOutput represents a tool call result with input and output.
type ToolCallOutput struct {
	ToolCallID       string         `json:"toolCallId"`
	ToolName         string         `json:"toolName"`
	Args             map[string]any `json:"args"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
	ProviderExecuted *bool          `json:"providerExecuted,omitempty"`
	Result           any            `json:"result,omitempty"`
	Error            any            `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// processAndEnqueueChunk helper
// ---------------------------------------------------------------------------

// processAndEnqueueChunk processes a chunk through output processors and
// enqueues it to the controller. Returns the processed chunk or nil if
// blocked by a processor.
func processAndEnqueueChunk(
	chunk map[string]any,
	processorRunner *ProcessorRunner,
	processorStates map[string]any,
	observabilityContext any,
	requestContext any,
	messageList any,
	controller any,
	writer *ProcessorStreamWriter,
) map[string]any {
	if processorRunner != nil && processorStates != nil {
		result := processorRunner.ProcessPart(
			chunk,
			processorStates,
			observabilityContext,
			requestContext,
			messageList,
			0,
			writer,
		)

		if result.Blocked {
			// Emit tripwire chunk.
			tripwireChunk := map[string]any{
				"type": "tripwire",
				"payload": map[string]any{
					"reason": result.Reason,
				},
			}
			if result.TripwireOptions != nil {
				payload := tripwireChunk["payload"].(map[string]any)
				payload["retry"] = result.TripwireOptions.Retry
				payload["metadata"] = result.TripwireOptions.Metadata
			}
			if result.ProcessorID != "" {
				payload := tripwireChunk["payload"].(map[string]any)
				payload["processorId"] = result.ProcessorID
			}
			SafeEnqueue(controller, tripwireChunk)
			return nil
		}

		if result.Part != nil {
			if processed, ok := result.Part.(map[string]any); ok {
				SafeEnqueue(controller, processed)
				return processed
			}
		}
		return nil
	}

	// No processor runner — enqueue directly.
	SafeEnqueue(controller, chunk)
	return chunk
}

// getProviderMetadataWithModelOutput computes toModelOutput for a tool call
// and returns providerMetadata with the result stored at mastra.modelOutput.
func getProviderMetadataWithModelOutput(
	tools ToolSet,
	toolCall ToolCallOutput,
) map[string]any {
	if tools == nil {
		return toolCall.ProviderMetadata
	}
	tool := tools[toolCall.ToolName]
	if tool == nil {
		return toolCall.ProviderMetadata
	}

	toolMap, ok := tool.(map[string]any)
	if !ok {
		return toolCall.ProviderMetadata
	}

	var modelOutput any
	if toModelOutputFn, ok := toolMap["toModelOutput"].(func(output any) any); ok && toolCall.Result != nil {
		modelOutput = toModelOutputFn(toolCall.Result)
	}

	result := make(map[string]any)
	for k, v := range toolCall.ProviderMetadata {
		result[k] = v
	}
	if modelOutput != nil {
		mastra, _ := result["mastra"].(map[string]any)
		if mastra == nil {
			mastra = make(map[string]any)
		}
		mastra["modelOutput"] = modelOutput
		result["mastra"] = mastra
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ---------------------------------------------------------------------------
// CreateLLMMappingStep
// ---------------------------------------------------------------------------

// CreateLLMMappingStep creates the workflow step that maps tool call results
// back into the iteration data and emits the appropriate chunks.
//
// This step:
//  1. Creates a ProcessorRunner for output processors (if configured) that
//     shares the same processorStates as the inner MastraModelOutput. This
//     ensures output processors see tool-result chunks too.
//  2. Retrieves the initial result from the llmExecutionStep.
//  3. Handles error results:
//     - Emits tool-error chunks through output processors.
//     - Adds error messages to the message list.
//     - Handles mixed turns (errors + HITL tools).
//     - Sets isContinued = true so the model can self-correct.
//  4. Handles pending HITL tool calls (bail if no result and no error).
//  5. Handles successful tool results:
//     - Emits tool-result chunks through output processors.
//     - Adds tool invocation messages to the message list (split by
//       client-executed vs provider-executed).
//  6. Checks for delegation bail flag.
//  7. Returns updated iteration data with refreshed messages.
//
// Output processor integration:
//   - LLM-generated chunks (text-delta, tool-call, etc.) are processed in
//     the inner MastraModelOutput (llm-execution-step).
//   - Tool-result and tool-error chunks are processed HERE since they're
//     created after tool execution, outside the MastraModelOutput pipeline.
//   - Both share the same processorStates map for state consistency.
func CreateLLMMappingStep(params OuterLLMRun, llmExecutionStep *Step) *Step {
	// Build ProcessorRunner for tool result chunks if output processors exist.
	var processorRunner *ProcessorRunner
	if len(params.OutputProcessors) > 0 && params.Logger != nil {
		processorRunner = &ProcessorRunner{
			InputProcessors:  nil,
			OutputProcessors: params.OutputProcessors,
			Logger:           params.Logger.(IMastraLogger),
			AgentName:        "LLMMappingStep",
			ProcessorStates:  params.ProcessorStates,
		}
	}

	// Build observability context from modelSpanTracker if available.
	observabilityContext := CreateObservabilityContext(params.ModelSpanTracker)

	// Create a ProcessorStreamWriter from outputWriter.
	var streamWriter *ProcessorStreamWriter
	if params.OutputWriter != nil {
		streamWriter = &ProcessorStreamWriter{
			Custom: func(data any) error {
				if ow, ok := params.OutputWriter.(func(data any) error); ok {
					return ow(data)
				}
				return nil
			},
		}
	}

	return &Step{
		ID: "llmExecutionMappingStep",
		Execute: func(args StepExecuteArgs) (any, error) {
			inputData, ok := args.InputData.([]ToolCallOutput)
			if !ok {
				// Try converting from []any.
				if arr, ok := args.InputData.([]any); ok {
					inputData = make([]ToolCallOutput, 0, len(arr))
					for _, item := range arr {
						if tc, ok := item.(ToolCallOutput); ok {
							inputData = append(inputData, tc)
						}
					}
				}
				if len(inputData) == 0 {
					return args.InputData, nil
				}
			}

			// Retrieve initial result from llmExecutionStep.
			// In TS this uses getStepResult(llmExecutionStep) to get the prior step's output.
			// Since the workflow engine chains steps, the last llmExecutionStep result should
			// have been stored. We retrieve it from the workflow's step result cache.
			// For now, we construct it from available data and the message list.
			initialResult := map[string]any{
				"stepResult": map[string]any{
					"reason":      "stop",
					"isContinued": false,
				},
				"messages": getMessagesFromList(params.MessageList),
				"output":   map[string]any{},
				"metadata": map[string]any{},
			}

			// Check for undefined results (errors or HITL pending).
			hasUndefined := false
			for _, tc := range inputData {
				if tc.Result == nil && tc.Error == nil {
					isProviderExecuted := tc.ProviderExecuted != nil && *tc.ProviderExecuted
					if !isProviderExecuted {
						hasUndefined = true
						break
					}
				}
			}

			if hasUndefined {
				// Collect error results.
				var errorResults []ToolCallOutput
				for _, tc := range inputData {
					if tc.Error != nil {
						errorResults = append(errorResults, tc)
					}
				}

				// Generate a tool result message ID.
				toolResultMessageID := ""
				if params.ExperimentalGenerateMessageID != nil {
					toolResultMessageID = params.ExperimentalGenerateMessageID()
				}
				if toolResultMessageID == "" {
					if params.Internal != nil {
						if internal, ok := params.Internal.(map[string]any); ok {
							if genID, ok := internal["generateId"].(func() string); ok {
								toolResultMessageID = genID()
							}
						}
					}
				}

				if len(errorResults) > 0 {
					// Emit tool-error chunks.
					for _, toolCall := range errorResults {
						chunk := map[string]any{
							"type":  "tool-error",
							"runId": params.RunID,
							"from":  ChunkFromAGENT,
							"payload": map[string]any{
								"error":            toolCall.Error,
								"args":             toolCall.Args,
								"toolCallId":       toolCall.ToolCallID,
								"toolName":         toolCall.ToolName,
								"providerMetadata": toolCall.ProviderMetadata,
							},
						}
						processed := processAndEnqueueChunk(
							chunk, processorRunner, params.ProcessorStates,
							observabilityContext, params.RequestContext, params.MessageList,
							params.Controller, streamWriter,
						)
						if processed != nil {
							if params.Options != nil {
								if opts, ok := params.Options.(map[string]any); ok {
									if onChunk, ok := opts["onChunk"].(func(chunk map[string]any) error); ok {
										_ = onChunk(processed)
									}
								}
							}
						}
					}

					// Add error messages to the message list.
					errorParts := make([]map[string]any, 0, len(errorResults))
					for _, tc := range errorResults {
						errMsg := ""
						if e, ok := tc.Error.(error); ok {
							errMsg = e.Error()
						} else if s, ok := tc.Error.(string); ok {
							errMsg = s
						}
						errorParts = append(errorParts, map[string]any{
							"type": "tool-invocation",
							"toolInvocation": map[string]any{
								"state":      "result",
								"toolCallId": tc.ToolCallID,
								"toolName":   SanitizeToolName(tc.ToolName),
								"args":       tc.Args,
								"result":     errMsg,
							},
						})
					}
					errorMsg := map[string]any{
						"id":   toolResultMessageID,
						"role": "assistant",
						"content": map[string]any{
							"format": 2,
							"parts":  errorParts,
						},
						"createdAt": time.Now(),
					}
					if ml, ok := params.MessageList.(MessageListFull); ok {
						ml.Add(errorMsg, "response")
					}

					// Check for pending HITL tool calls.
					hasPendingHITL := false
					for _, tc := range inputData {
						isProviderExecuted := tc.ProviderExecuted != nil && *tc.ProviderExecuted
						if tc.Result == nil && tc.Error == nil && !isProviderExecuted {
							hasPendingHITL = true
							break
						}
					}

					if len(errorResults) > 0 && !hasPendingHITL {
						// Process any successful tool results from this turn.
						var successfulResults []ToolCallOutput
						for _, tc := range inputData {
							if tc.Result != nil {
								successfulResults = append(successfulResults, tc)
							}
						}
						if len(successfulResults) > 0 {
							for _, toolCall := range successfulResults {
								chunk := map[string]any{
									"type":  "tool-result",
									"runId": params.RunID,
									"from":  ChunkFromAGENT,
									"payload": map[string]any{
										"args":             toolCall.Args,
										"toolCallId":       toolCall.ToolCallID,
										"toolName":         toolCall.ToolName,
										"result":           toolCall.Result,
										"providerMetadata": toolCall.ProviderMetadata,
										"providerExecuted": toolCall.ProviderExecuted,
									},
								}
								processed := processAndEnqueueChunk(
									chunk, processorRunner, params.ProcessorStates,
									observabilityContext, params.RequestContext, params.MessageList,
									params.Controller, streamWriter,
								)
								if processed != nil {
									if params.Options != nil {
										if opts, ok := params.Options.(map[string]any); ok {
											if onChunk, ok := opts["onChunk"].(func(chunk map[string]any) error); ok {
												_ = onChunk(processed)
											}
										}
									}
								}
							}

							// Split client vs provider executed.
							var clientResults []ToolCallOutput
							var providerResults []ToolCallOutput
							for _, tc := range successfulResults {
								isProvider := tc.ProviderExecuted != nil && *tc.ProviderExecuted
								if isProvider {
									providerResults = append(providerResults, tc)
								} else {
									clientResults = append(clientResults, tc)
								}
							}

							if len(clientResults) > 0 {
								parts := make([]map[string]any, 0, len(clientResults))
								for _, tc := range clientResults {
									pm := getProviderMetadataWithModelOutput(params.Tools, tc)
									part := map[string]any{
										"type": "tool-invocation",
										"toolInvocation": map[string]any{
											"state":      "result",
											"toolCallId": tc.ToolCallID,
											"toolName":   SanitizeToolName(tc.ToolName),
											"args":       tc.Args,
											"result":     tc.Result,
										},
									}
									if pm != nil {
										part["providerMetadata"] = pm
									}
									parts = append(parts, part)
								}
								successMsg := map[string]any{
									"id":   toolResultMessageID,
									"role": "assistant",
									"content": map[string]any{
										"format": 2,
										"parts":  parts,
									},
									"createdAt": time.Now(),
								}
								if ml, ok := params.MessageList.(MessageListFull); ok {
									ml.Add(successMsg, "response")
								}
							}

							if len(providerResults) > 0 {
								parts := make([]map[string]any, 0, len(providerResults))
								for _, tc := range providerResults {
									part := map[string]any{
										"type": "tool-invocation",
										"toolInvocation": map[string]any{
											"state":      "result",
											"toolCallId": tc.ToolCallID,
											"toolName":   SanitizeToolName(tc.ToolName),
											"args":       tc.Args,
											"result":     tc.Result,
										},
										"providerExecuted": true,
									}
									if tc.ProviderMetadata != nil {
										part["providerMetadata"] = tc.ProviderMetadata
									}
									parts = append(parts, part)
								}
								providerMsg := map[string]any{
									"id":   toolResultMessageID,
									"role": "assistant",
									"content": map[string]any{
										"format": 2,
										"parts":  parts,
									},
									"createdAt": time.Now(),
								}
								if ml, ok := params.MessageList.(MessageListFull); ok {
									ml.Add(providerMsg, "response")
								}
							}
						}

						// Continue loop — model sees error and can self-correct.
						stepResult := initialResult["stepResult"].(map[string]any)
						stepResult["isContinued"] = true
						stepResult["reason"] = "tool-calls"
						// Refresh messages from the messageList.
						initialResult["messages"] = getMessagesFromList(params.MessageList)
						return initialResult, nil
					}

					// Pending HITL or no errors but undefined results — bail.
					stepResult := initialResult["stepResult"].(map[string]any)
					if stepResult["reason"] != "retry" {
						stepResult["isContinued"] = false
					}
					// Refresh messages from the messageList.
					initialResult["messages"] = getMessagesFromList(params.MessageList)
					return initialResult, nil
				}

				// No errors but undefined results — bail.
				stepResult := initialResult["stepResult"].(map[string]any)
				if stepResult["reason"] != "retry" {
					stepResult["isContinued"] = false
				}
				// Refresh messages from the messageList.
				initialResult["messages"] = getMessagesFromList(params.MessageList)
				return initialResult, nil
			}

			// --- Process successful tool results ---
			if len(inputData) > 0 {
				// Emit tool-result chunks.
				for _, toolCall := range inputData {
					chunk := map[string]any{
						"type":  "tool-result",
						"runId": params.RunID,
						"from":  ChunkFromAGENT,
						"payload": map[string]any{
							"args":             toolCall.Args,
							"toolCallId":       toolCall.ToolCallID,
							"toolName":         toolCall.ToolName,
							"result":           toolCall.Result,
							"providerMetadata": toolCall.ProviderMetadata,
							"providerExecuted": toolCall.ProviderExecuted,
						},
					}
					processed := processAndEnqueueChunk(
						chunk, processorRunner, params.ProcessorStates,
						observabilityContext, params.RequestContext, params.MessageList,
						params.Controller, streamWriter,
					)
					if processed != nil {
						if params.Options != nil {
							if opts, ok := params.Options.(map[string]any); ok {
								if onChunk, ok := opts["onChunk"].(func(chunk map[string]any) error); ok {
									_ = onChunk(processed)
								}
							}
						}
					}
				}

				// Split client-executed and provider-executed tools.
				var clientExecuted []ToolCallOutput
				var providerExecuted []ToolCallOutput
				for _, tc := range inputData {
					isProvider := tc.ProviderExecuted != nil && *tc.ProviderExecuted
					if isProvider {
						providerExecuted = append(providerExecuted, tc)
					} else {
						clientExecuted = append(clientExecuted, tc)
					}
				}

				// Add client-executed tool results to message list.
				if len(clientExecuted) > 0 {
					toolResultMessageID := ""
					if params.ExperimentalGenerateMessageID != nil {
						toolResultMessageID = params.ExperimentalGenerateMessageID()
					}
					if toolResultMessageID == "" {
						if params.Internal != nil {
							if internal, ok := params.Internal.(map[string]any); ok {
								if genID, ok := internal["generateId"].(func() string); ok {
									toolResultMessageID = genID()
								}
							}
						}
					}

					parts := make([]map[string]any, 0, len(clientExecuted))
					for _, tc := range clientExecuted {
						pm := getProviderMetadataWithModelOutput(params.Tools, tc)
						part := map[string]any{
							"type": "tool-invocation",
							"toolInvocation": map[string]any{
								"state":      "result",
								"toolCallId": tc.ToolCallID,
								"toolName":   SanitizeToolName(tc.ToolName),
								"args":       tc.Args,
								"result":     tc.Result,
							},
						}
						if pm != nil {
							part["providerMetadata"] = pm
						}
						parts = append(parts, part)
					}
					msg := map[string]any{
						"id":   toolResultMessageID,
						"role": "assistant",
						"content": map[string]any{
							"format": 2,
							"parts":  parts,
						},
						"createdAt": time.Now(),
					}
				if ml, ok := params.MessageList.(MessageListFull); ok {
						ml.Add(msg, "response")
					}
				}

				// Add provider-executed tool results.
				if len(providerExecuted) > 0 {
					providerMessageID := ""
					if params.ExperimentalGenerateMessageID != nil {
						providerMessageID = params.ExperimentalGenerateMessageID()
					}
					if providerMessageID == "" {
						if params.Internal != nil {
							if internal, ok := params.Internal.(map[string]any); ok {
								if genID, ok := internal["generateId"].(func() string); ok {
									providerMessageID = genID()
								}
							}
						}
					}

					parts := make([]map[string]any, 0, len(providerExecuted))
					for _, tc := range providerExecuted {
						part := map[string]any{
							"type": "tool-invocation",
							"toolInvocation": map[string]any{
								"state":      "result",
								"toolCallId": tc.ToolCallID,
								"toolName":   SanitizeToolName(tc.ToolName),
								"args":       tc.Args,
								"result":     tc.Result,
							},
							"providerExecuted": true,
						}
						if tc.ProviderMetadata != nil {
							part["providerMetadata"] = tc.ProviderMetadata
						}
						parts = append(parts, part)
					}
					msg := map[string]any{
						"id":   providerMessageID,
						"role": "assistant",
						"content": map[string]any{
							"format": 2,
							"parts":  parts,
						},
						"createdAt": time.Now(),
					}
					if ml, ok := params.MessageList.(MessageListFull); ok {
						ml.Add(msg, "response")
					}
				}

				// Check for delegation bail flag.
				if params.RequestContext != nil {
					if rc, ok := params.RequestContext.(map[string]any); ok {
						if bailed, ok := rc["__mastra_delegationBailed"].(bool); ok && bailed {
							if params.Internal != nil {
								if internal, ok := params.Internal.(map[string]any); ok {
									internal["_delegationBailed"] = true
								}
							}
							rc["__mastra_delegationBailed"] = false
						}
					}
				}

				// Refresh messages from the messageList.
				initialResult["messages"] = getMessagesFromList(params.MessageList)
				return initialResult, nil
			}

			// Fallback: empty input data.
			return initialResult, nil
		},
	}
}
