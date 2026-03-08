// Ported from: packages/ai/src/generate-text/generate-text.ts
package generatetext

import (
	"context"
	"fmt"
)

// GenerateTextOnStartCallback is called when the generateText operation begins.
type GenerateTextOnStartCallback func(event OnStartEvent)

// GenerateTextOnStepStartCallback is called when a step begins.
type GenerateTextOnStepStartCallback func(event OnStepStartEvent)

// GenerateTextOnToolCallStartCallback is called before each tool execution.
type GenerateTextOnToolCallStartCallback func(event OnToolCallStartEvent)

// GenerateTextOnToolCallFinishCallback is called after each tool execution.
type GenerateTextOnToolCallFinishCallback func(event OnToolCallFinishEvent)

// GenerateTextOnStepFinishCallback is called when a step completes.
type GenerateTextOnStepFinishCallback func(event OnStepFinishEvent)

// GenerateTextOnFinishCallback is called when the entire generation completes.
type GenerateTextOnFinishCallback func(event OnFinishEvent)

// GenerateTextIncludeSettings controls what data is included in step results.
type GenerateTextIncludeSettings struct {
	RequestBody  *bool
	ResponseBody *bool
}

// GenerateTextOptions contains all options for the generateText function.
type GenerateTextOptions struct {
	// Ctx is the Go context for cancellation (replaces AbortSignal).
	Ctx context.Context

	// Model is the language model to use.
	Model LanguageModel

	// Tools are the tools that the model can call.
	Tools ToolSet

	// ToolChoice is the tool choice strategy. Default: "auto".
	ToolChoice *ToolChoice

	// System is a system message for the prompt.
	System interface{} // string | SystemModelMessage | []SystemModelMessage

	// Prompt is a simple text prompt (use either Prompt or Messages, not both).
	Prompt string

	// Messages is a list of messages (use either Prompt or Messages, not both).
	Messages []ModelMessage

	// MaxOutputTokens is the maximum number of tokens to generate.
	MaxOutputTokens *int

	// Temperature is the sampling temperature.
	Temperature *float64

	// TopP is the nucleus sampling parameter.
	TopP *float64

	// TopK is the top-K sampling parameter.
	TopK *int

	// PresencePenalty affects likelihood of repeating prompt information.
	PresencePenalty *float64

	// FrequencyPenalty affects likelihood of repeating words/phrases.
	FrequencyPenalty *float64

	// StopSequences causes generation to stop when one is generated.
	StopSequences []string

	// Seed for reproducible generation.
	Seed *int

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional HTTP headers for the request.
	Headers map[string]string

	// Timeout configuration.
	Timeout *TimeoutConfiguration

	// StopWhen is the condition(s) for stopping generation.
	// Default: StepCountIs(1).
	StopWhen []StopCondition

	// Output is the specification for structured outputs.
	Output Output

	// ProviderOptions are additional provider-specific options.
	ProviderOptions ProviderOptions

	// ActiveTools limits which tools are available for the model to call.
	ActiveTools []string

	// PrepareStep is an optional function to provide different settings for a step.
	PrepareStep PrepareStepFunction

	// RepairToolCall attempts to repair a tool call that failed to parse.
	RepairToolCall ToolCallRepairFunction

	// ExperimentalContext is a user-defined context object.
	ExperimentalContext interface{}

	// Include controls what data is included in step results.
	Include *GenerateTextIncludeSettings

	// GenerateID is an optional ID generator (for testing).
	GenerateID IdGenerator

	// Callbacks
	OnStart          GenerateTextOnStartCallback
	OnStepStart      GenerateTextOnStepStartCallback
	OnToolCallStart  GenerateTextOnToolCallStartCallback
	OnToolCallFinish GenerateTextOnToolCallFinishCallback
	OnStepFinish     GenerateTextOnStepFinishCallback
	OnFinish         GenerateTextOnFinishCallback
}

// GenerateText generates text and calls tools for a given prompt using a language model.
//
// This function does not stream the output. If you want to stream the output, use StreamText instead.
//
// TODO: Full implementation requires porting the model invocation infrastructure,
// telemetry, prompt conversion, and retry logic. This is a structural port of the
// function signature, types, and core flow.
func GenerateText(opts GenerateTextOptions) (*GenerateTextResult, error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	if opts.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	stopConditions := opts.StopWhen
	if stopConditions == nil {
		stopConditions = []StopCondition{StepCountIs(1)}
	}

	experimentalContext := opts.ExperimentalContext

	// Collect tool approvals from initial messages
	approvals, err := CollectToolApprovals(opts.Messages)
	if err != nil {
		return nil, fmt.Errorf("collecting tool approvals: %w", err)
	}

	// Filter non-provider-executed approved tool approvals
	var localApprovedToolApprovals []CollectedToolApproval
	for _, ta := range approvals.ApprovedToolApprovals {
		if !ta.ToolCall.ProviderExecuted {
			localApprovedToolApprovals = append(localApprovedToolApprovals, ta)
		}
	}

	var steps []StepResult
	responseMessages := make([]ResponseMessage, 0)

	// Execute pre-approved tool calls if any exist
	if len(approvals.DeniedToolApprovals) > 0 || len(localApprovedToolApprovals) > 0 {
		var toolCallsToExec []ToolCall
		for _, ta := range localApprovedToolApprovals {
			toolCallsToExec = append(toolCallsToExec, ta.ToolCall)
		}

		toolOutputs, err := executeTools(executeToolsOpts{
			toolCalls:           toolCallsToExec,
			tools:               opts.Tools,
			messages:            opts.Messages,
			abortSignal:         opts.Ctx.Done(),
			experimentalContext: experimentalContext,
		})
		if err != nil {
			return nil, err
		}

		var toolContent []ModelMessageContent
		for _, output := range toolOutputs {
			switch o := output.(type) {
			case *ToolResult:
				toolContent = append(toolContent, ModelMessageContent{
					Type:       "tool-result",
					ToolCallID: o.ToolCallID,
					ToolName:   o.ToolName,
					Output:     o.Output,
				})
			case *ToolError:
				toolContent = append(toolContent, ModelMessageContent{
					Type:       "tool-result",
					ToolCallID: o.ToolCallID,
					ToolName:   o.ToolName,
					Output:     o.Error,
				})
			}
		}

		// Add denied tool results
		for _, ta := range approvals.DeniedToolApprovals {
			toolContent = append(toolContent, ModelMessageContent{
				Type:       "tool-result",
				ToolCallID: ta.ToolCall.ToolCallID,
				ToolName:   ta.ToolCall.ToolName,
				Output: map[string]interface{}{
					"type":   "execution-denied",
					"reason": ta.ApprovalResponse.Reason,
				},
			})
		}

		if len(toolContent) > 0 {
			responseMessages = append(responseMessages, ModelMessage{
				Role:    "tool",
				Content: toolContent,
			})
		}
	}

	// TODO: The main generation loop requires model.doGenerate() which is not yet ported.
	// The loop structure would be:
	// 1. Prepare step (call PrepareStep if provided)
	// 2. Convert to language model prompt
	// 3. Call model.doGenerate()
	// 4. Parse tool calls
	// 5. Check approval needs
	// 6. Execute tools
	// 7. Build step result
	// 8. Check stop conditions
	// 9. Loop if needed

	// For now, return an empty result to allow compilation
	totalUsage := LanguageModelUsage{}
	for _, step := range steps {
		totalUsage = AddLanguageModelUsage(totalUsage, step.Usage)
	}

	result := NewDefaultGenerateTextResult(steps, totalUsage, nil)
	r := result.ToResult()
	return &r, nil
}

type executeToolsOpts struct {
	toolCalls           []ToolCall
	tools               ToolSet
	messages            []ModelMessage
	abortSignal         <-chan struct{}
	experimentalContext interface{}
}

func executeTools(opts executeToolsOpts) ([]ToolOutput, error) {
	var outputs []ToolOutput
	for _, tc := range opts.toolCalls {
		output, err := ExecuteToolCall(ExecuteToolCallOptions{
			ToolCall:            tc,
			Tools:               opts.tools,
			Messages:            opts.messages,
			AbortSignal:         opts.abortSignal,
			ExperimentalContext: opts.experimentalContext,
		})
		if err != nil {
			return nil, err
		}
		if output != nil {
			outputs = append(outputs, output)
		}
	}
	return outputs, nil
}

// AsContent converts model content, tool calls, and tool outputs into ContentParts.
func AsContent(
	content []LanguageModelV4Content,
	toolCalls []ToolCall,
	toolOutputs []ToolOutput,
	toolApprovalRequests []ToolApprovalRequestOutput,
	tools ToolSet,
) ([]ContentPart, error) {
	var contentParts []ContentPart

	for _, part := range content {
		switch part.Type {
		case "text":
			contentParts = append(contentParts, NewTextContentPart(part.Text, part.ProviderMetadata))
		case "reasoning":
			contentParts = append(contentParts, NewReasoningContentPart(part.Text, part.ProviderMetadata))
		case "source":
			contentParts = append(contentParts, NewSourceContentPart(Source{
				Type: part.Type,
			}))
		case "file":
			contentParts = append(contentParts, NewFileContentPart(
				NewDefaultGeneratedFile(part.Data, part.MediaType),
				part.ProviderMetadata,
			))
		case "tool-call":
			for i := range toolCalls {
				if toolCalls[i].ToolCallID == part.ToolCallID {
					contentParts = append(contentParts, NewToolCallContentPart(toolCalls[i]))
					break
				}
			}
		case "tool-result":
			found := false
			for i := range toolCalls {
				if toolCalls[i].ToolCallID == part.ToolCallID {
					found = true
					if part.IsError {
						contentParts = append(contentParts, NewToolErrorContentPart(ToolError{
							Type:             "tool-error",
							ToolCallID:       part.ToolCallID,
							ToolName:         part.ToolName,
							Input:            toolCalls[i].Input,
							Error:            part.Result,
							ProviderExecuted: true,
							Dynamic:          toolCalls[i].Dynamic,
						}))
					} else {
						contentParts = append(contentParts, NewToolResultContentPart(ToolResult{
							Type:             "tool-result",
							ToolCallID:       part.ToolCallID,
							ToolName:         part.ToolName,
							Input:            toolCalls[i].Input,
							Output:           part.Result,
							ProviderExecuted: true,
							Dynamic:          toolCalls[i].Dynamic,
						}))
					}
					break
				}
			}
			if !found {
				// Handle deferred results for provider-executed tools
				tool, toolExists := tools[part.ToolName]
				supportsDeferredResults := toolExists && tool.Type == "provider" && tool.SupportsDeferredResults
				if !supportsDeferredResults {
					return nil, fmt.Errorf("tool call %s not found", part.ToolCallID)
				}
				if part.IsError {
					contentParts = append(contentParts, NewToolErrorContentPart(ToolError{
						Type:             "tool-error",
						ToolCallID:       part.ToolCallID,
						ToolName:         part.ToolName,
						Error:            part.Result,
						ProviderExecuted: true,
						Dynamic:          part.Dynamic,
					}))
				} else {
					contentParts = append(contentParts, NewToolResultContentPart(ToolResult{
						Type:             "tool-result",
						ToolCallID:       part.ToolCallID,
						ToolName:         part.ToolName,
						Output:           part.Result,
						ProviderExecuted: true,
						Dynamic:          part.Dynamic,
					}))
				}
			}
		case "tool-approval-request":
			found := false
			for i := range toolCalls {
				if toolCalls[i].ToolCallID == part.ToolCallID {
					found = true
					contentParts = append(contentParts, NewToolApprovalRequestContentPart(ToolApprovalRequestOutput{
						Type:       "tool-approval-request",
						ApprovalID: part.ApprovalID,
						ToolCall:   toolCalls[i],
					}))
					break
				}
			}
			if !found {
				return nil, &ToolCallNotFoundForApprovalError{
					ToolCallID: part.ToolCallID,
					ApprovalID: part.ApprovalID,
				}
			}
		}
	}

	// Append tool outputs
	for _, output := range toolOutputs {
		switch o := output.(type) {
		case *ToolResult:
			contentParts = append(contentParts, NewToolResultContentPart(*o))
		case *ToolError:
			contentParts = append(contentParts, NewToolErrorContentPart(*o))
		}
	}

	// Append tool approval requests
	for _, tar := range toolApprovalRequests {
		contentParts = append(contentParts, NewToolApprovalRequestContentPart(tar))
	}

	return contentParts, nil
}

// AsToolCalls extracts tool calls from language model content.
func AsToolCalls(content []LanguageModelV4Content) []struct {
	ToolCallID string
	ToolName   string
	Input      string
} {
	var calls []struct {
		ToolCallID string
		ToolName   string
		Input      string
	}
	for _, part := range content {
		if part.Type == "tool-call" {
			calls = append(calls, struct {
				ToolCallID string
				ToolName   string
				Input      string
			}{
				ToolCallID: part.ToolCallID,
				ToolName:   part.ToolName,
				Input:      part.Input,
			})
		}
	}
	return calls
}
