// Ported from: packages/ai/src/generate-text/run-tools-transformation.ts
package generatetext

import (
	"fmt"
	"sync"
)

// SingleRequestTextStreamPart represents a chunk in a text generation stream.
// This is a discriminated union in TypeScript; in Go we use a struct with a Type discriminator.
type SingleRequestTextStreamPart struct {
	// Type discriminates the part kind.
	Type string

	// For text-start, text-delta, text-end, reasoning-start, reasoning-delta, reasoning-end
	ID               string
	Text             string // for deltas
	Delta            string // for tool-input-delta
	ProviderMetadata ProviderMetadata

	// For tool-input-start
	ToolName string
	Dynamic  bool
	Title    string

	// For tool-call, tool-result, tool-error
	ToolCall   *ToolCall
	ToolResult *ToolResult
	ToolError  *ToolError

	// For tool-approval-request
	ToolApprovalRequest *ToolApprovalRequestOutput

	// For source
	Source *Source

	// For file
	File GeneratedFile

	// For stream-start
	Warnings []SharedV4Warning

	// For response-metadata
	Timestamp interface{}
	ModelID   string

	// For finish
	FinishReason    FinishReason
	RawFinishReason string
	Usage           LanguageModelUsage

	// For error
	Error interface{}

	// For raw
	RawValue interface{}

	// Provider executed flag
	ProviderExecuted bool
}

// RunToolsTransformationOptions contains the parameters for the tools transformation.
type RunToolsTransformationOptions struct {
	Tools              ToolSet
	GeneratorStream    <-chan LanguageModelV4StreamPart
	System             interface{} // string | SystemModelMessage | []SystemModelMessage | nil
	Messages           []ModelMessage
	AbortSignal        <-chan struct{}
	RepairToolCall     ToolCallRepairFunction
	ExperimentalContext interface{}
	GenerateID         IdGenerator
	StepNumber         *int
	Model              *ModelInfo
	OnToolCallStart    []func(event OnToolCallStartEvent)
	OnToolCallFinish   []func(event OnToolCallFinishEvent)
}

// RunToolsTransformation processes a generator stream, parsing tool calls,
// executing tools, and combining the results into a single output stream.
func RunToolsTransformation(opts RunToolsTransformationOptions) <-chan SingleRequestTextStreamPart {
	output := make(chan SingleRequestTextStreamPart, 100)

	// Tool results channel for async tool executions
	toolResults := make(chan SingleRequestTextStreamPart, 100)

	// Track outstanding tool results
	var outstandingMu sync.Mutex
	outstandingToolResults := map[string]bool{}

	// Track tool inputs for provider-side tool results
	toolInputs := map[string]interface{}{}

	// Track parsed tool calls for provider-emitted approval requests
	toolCallsByToolCallID := map[string]ToolCall{}

	canClose := false
	var finishChunk *SingleRequestTextStreamPart

	attemptClose := func() {
		outstandingMu.Lock()
		outstanding := len(outstandingToolResults)
		outstandingMu.Unlock()

		if canClose && outstanding == 0 {
			if finishChunk != nil {
				toolResults <- *finishChunk
			}
			close(toolResults)
		}
	}

	go func() {
		// Process generator stream (forward stream)
		for chunk := range opts.GeneratorStream {
			switch chunk.Type {
			// Forward passthrough types
			case "stream-start", "text-start", "text-delta", "text-end",
				"reasoning-start", "reasoning-delta", "reasoning-end",
				"tool-input-start", "tool-input-delta", "tool-input-end",
				"source", "response-metadata", "error", "raw":
				output <- convertStreamPart(chunk)

			case "file":
				output <- SingleRequestTextStreamPart{
					Type:             "file",
					File:             NewDefaultGeneratedFileWithType(chunk.Data, chunk.MediaType),
					ProviderMetadata: chunk.ProviderMetadata,
				}

			case "finish":
				finishChunk = &SingleRequestTextStreamPart{
					Type:            "finish",
					FinishReason:    chunk.FinishReason.Unified,
					RawFinishReason: chunk.FinishReason.Raw,
					Usage: AsLanguageModelUsage(struct {
						InputTokens  TokenCount
						OutputTokens TokenCount
					}{
						InputTokens:  chunk.Usage.InputTokens,
						OutputTokens: chunk.Usage.OutputTokens,
					}),
					ProviderMetadata: chunk.ProviderMetadata,
				}

			case "tool-approval-request":
				tc, ok := toolCallsByToolCallID[chunk.ToolCallID]
				if !ok {
					toolResults <- SingleRequestTextStreamPart{
						Type:  "error",
						Error: &ToolCallNotFoundForApprovalError{ToolCallID: chunk.ToolCallID, ApprovalID: chunk.ApprovalID},
					}
					continue
				}
				output <- SingleRequestTextStreamPart{
					Type: "tool-approval-request",
					ToolApprovalRequest: &ToolApprovalRequestOutput{
						Type:       "tool-approval-request",
						ApprovalID: chunk.ApprovalID,
						ToolCall:   tc,
					},
				}

			case "tool-call":
				func() {
					toolCallV4 := LanguageModelV4ToolCall{
						Type:             "tool-call",
						ToolCallID:       chunk.ToolCallID,
						ToolName:         chunk.ToolName,
						Input:            chunk.Input,
						ProviderExecuted: chunk.ProviderExecuted,
						Dynamic:          chunk.Dynamic,
						ProviderMetadata: chunk.ProviderMetadata,
					}

					toolCall, err := ParseToolCall(ParseToolCallOptions{
						ToolCall:       toolCallV4,
						Tools:          opts.Tools,
						RepairToolCall: opts.RepairToolCall,
						System:         opts.System,
						Messages:       opts.Messages,
					})
					if err != nil {
						toolResults <- SingleRequestTextStreamPart{Type: "error", Error: err}
						return
					}

					toolCallsByToolCallID[toolCall.ToolCallID] = toolCall
					output <- SingleRequestTextStreamPart{
						Type:     "tool-call",
						ToolCall: &toolCall,
					}

					if toolCall.Invalid {
						toolResults <- SingleRequestTextStreamPart{
							Type: "tool-error",
							ToolError: &ToolError{
								Type:       "tool-error",
								ToolCallID: toolCall.ToolCallID,
								ToolName:   toolCall.ToolName,
								Input:      toolCall.Input,
								Error:      fmt.Sprintf("%v", toolCall.Error),
								Dynamic:    true,
								Title:      toolCall.Title,
							},
						}
						return
					}

					tool, ok := opts.Tools[toolCall.ToolName]
					if !ok {
						return // Ignore tool calls for tools that are not available
					}

					if tool.OnInputAvailable != nil {
						_ = tool.OnInputAvailable(ToolInputAvailableOptions{
							Input:               toolCall.Input,
							ToolCallID:          toolCall.ToolCallID,
							Messages:            opts.Messages,
							AbortSignal:         opts.AbortSignal,
							ExperimentalContext: opts.ExperimentalContext,
						})
					}

					needsApproval, _ := IsApprovalNeeded(tool, toolCall, opts.Messages, opts.ExperimentalContext)
					if needsApproval {
						toolResults <- SingleRequestTextStreamPart{
							Type: "tool-approval-request",
							ToolApprovalRequest: &ToolApprovalRequestOutput{
								Type:       "tool-approval-request",
								ApprovalID: opts.GenerateID(),
								ToolCall:   toolCall,
							},
						}
						return
					}

					toolInputs[toolCall.ToolCallID] = toolCall.Input

					// Execute tool if it has an execute function and is not provider-executed
					if tool.Execute != nil && !toolCall.ProviderExecuted {
						toolExecID := opts.GenerateID()
						outstandingMu.Lock()
						outstandingToolResults[toolExecID] = true
						outstandingMu.Unlock()

						go func() {
							defer func() {
								outstandingMu.Lock()
								delete(outstandingToolResults, toolExecID)
								outstandingMu.Unlock()
								attemptClose()
							}()

							result, err := ExecuteToolCall(ExecuteToolCallOptions{
								ToolCall:            toolCall,
								Tools:               opts.Tools,
								Messages:            opts.Messages,
								AbortSignal:         opts.AbortSignal,
								ExperimentalContext: opts.ExperimentalContext,
								StepNumber:          opts.StepNumber,
								Model:               opts.Model,
								OnToolCallStart:     opts.OnToolCallStart,
								OnToolCallFinish:    opts.OnToolCallFinish,
							})
							if err != nil {
								toolResults <- SingleRequestTextStreamPart{Type: "error", Error: err}
								return
							}
							if result == nil {
								return
							}
							switch r := result.(type) {
							case *ToolResult:
								toolResults <- SingleRequestTextStreamPart{
									Type:       "tool-result",
									ToolResult: r,
								}
							case *ToolError:
								toolResults <- SingleRequestTextStreamPart{
									Type:      "tool-error",
									ToolError: r,
								}
							}
						}()
					}
				}()

			case "tool-result":
				toolName := chunk.ToolName
				if chunk.IsError {
					toolResults <- SingleRequestTextStreamPart{
						Type: "tool-error",
						ToolError: &ToolError{
							Type:             "tool-error",
							ToolCallID:       chunk.ToolCallID,
							ToolName:         toolName,
							Input:            toolInputs[chunk.ToolCallID],
							ProviderExecuted: true,
							Error:            chunk.Result,
							Dynamic:          chunk.Dynamic,
						},
					}
				} else {
					output <- SingleRequestTextStreamPart{
						Type: "tool-result",
						ToolResult: &ToolResult{
							Type:             "tool-result",
							ToolCallID:       chunk.ToolCallID,
							ToolName:         toolName,
							Input:            toolInputs[chunk.ToolCallID],
							Output:           chunk.Result,
							ProviderExecuted: true,
							Dynamic:          chunk.Dynamic,
						},
					}
				}
			}
		}

		canClose = true
		attemptClose()
	}()

	// Merge output and toolResults into a single channel
	merged := make(chan SingleRequestTextStreamPart, 100)
	go func() {
		defer close(merged)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for part := range output {
				merged <- part
			}
		}()

		go func() {
			defer wg.Done()
			for part := range toolResults {
				merged <- part
			}
		}()

		wg.Wait()
	}()

	return merged
}

func convertStreamPart(chunk LanguageModelV4StreamPart) SingleRequestTextStreamPart {
	part := SingleRequestTextStreamPart{
		Type:             chunk.Type,
		ID:               chunk.ID,
		Text:             chunk.Text,
		Delta:            chunk.Delta,
		ProviderMetadata: chunk.ProviderMetadata,
		ToolName:         chunk.ToolName,
		Dynamic:          chunk.Dynamic,
		Error:            chunk.Error,
		RawValue:         chunk.RawValue,
	}
	if chunk.Source != nil {
		part.Source = chunk.Source
	}
	if chunk.Warnings != nil {
		part.Warnings = chunk.Warnings
	}
	if chunk.Timestamp != nil {
		part.Timestamp = chunk.Timestamp
	}
	if chunk.ModelID != "" {
		part.ModelID = chunk.ModelID
	}
	return part
}
