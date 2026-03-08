// Ported from: packages/ai/src/generate-text/generate-text-result.ts
package generatetext

// GenerateTextResult is the result of a generateText call.
// It contains the generated text, tool calls, tool results, and additional information.
type GenerateTextResult struct {
	// Content is the content generated in the last step.
	Content []ContentPart

	// Text is the text generated in the last step.
	Text string

	// Reasoning is the full reasoning generated in the last step.
	Reasoning []ReasoningOutput

	// ReasoningText is the reasoning text generated in the last step.
	ReasoningText string

	// Files are the files generated in the last step.
	Files []GeneratedFile

	// Sources are sources used as references in the last step.
	Sources []Source

	// ToolCalls are the tool calls made in the last step.
	ToolCalls []ToolCall

	// StaticToolCalls are the static tool calls made in the last step.
	StaticToolCalls []ToolCall

	// DynamicToolCalls are the dynamic tool calls made in the last step.
	DynamicToolCalls []ToolCall

	// ToolResults are the results of the tool calls from the last step.
	ToolResults []ToolResult

	// StaticToolResults are the static tool results from the last step.
	StaticToolResults []ToolResult

	// DynamicToolResults are the dynamic tool results from the last step.
	DynamicToolResults []ToolResult

	// FinishReason is the unified reason why the generation finished.
	FinishReason FinishReason

	// RawFinishReason is the raw reason from the provider.
	RawFinishReason string

	// Usage is the token usage of the last step.
	Usage LanguageModelUsage

	// TotalUsage is the total token usage across all steps.
	TotalUsage LanguageModelUsage

	// Warnings are warnings from the model provider.
	Warnings []CallWarning

	// Request contains additional request information.
	Request LanguageModelRequestMetadata

	// Response contains additional response information.
	Response GenerateTextResponseMetadata

	// ProviderMetadata contains additional provider-specific metadata.
	ProviderMetadata ProviderMetadata

	// Steps contains details for all steps.
	Steps []StepResult

	// Output is the generated structured output using the output specification.
	Output interface{}
}

// GenerateTextResponseMetadata extends response metadata with messages and body.
type GenerateTextResponseMetadata struct {
	LanguageModelResponseMetadata

	// Messages are the response messages generated during the call.
	Messages []ResponseMessage

	// Body is the response body (available only for HTTP-based providers).
	Body interface{}
}

// DefaultGenerateTextResult builds a GenerateTextResult from steps and computed values.
type DefaultGenerateTextResult struct {
	steps      []StepResult
	totalUsage LanguageModelUsage
	output     interface{}
}

// NewDefaultGenerateTextResult creates a new DefaultGenerateTextResult.
func NewDefaultGenerateTextResult(steps []StepResult, totalUsage LanguageModelUsage, output interface{}) *DefaultGenerateTextResult {
	return &DefaultGenerateTextResult{
		steps:      steps,
		totalUsage: totalUsage,
		output:     output,
	}
}

// ToResult converts to a GenerateTextResult, pulling most fields from the final step.
func (d *DefaultGenerateTextResult) ToResult() GenerateTextResult {
	if len(d.steps) == 0 {
		return GenerateTextResult{
			TotalUsage: d.totalUsage,
			Steps:      d.steps,
			Output:     d.output,
		}
	}

	finalStep := d.steps[len(d.steps)-1]

	return GenerateTextResult{
		Content:            finalStep.Content,
		Text:               finalStep.Text(),
		Reasoning:          toReasoningOutputs(finalStep.Reasoning()),
		ReasoningText:      finalStep.ReasoningText(),
		Files:              finalStep.Files(),
		Sources:            finalStep.Sources(),
		ToolCalls:          finalStep.ToolCalls(),
		StaticToolCalls:    finalStep.StaticToolCalls(),
		DynamicToolCalls:   finalStep.DynamicToolCalls(),
		ToolResults:        finalStep.ToolResults(),
		StaticToolResults:  finalStep.StaticToolResults(),
		DynamicToolResults: finalStep.DynamicToolResults(),
		FinishReason:       finalStep.FinishReason,
		RawFinishReason:    finalStep.RawFinishReason,
		Usage:              finalStep.Usage,
		TotalUsage:         d.totalUsage,
		Warnings:           finalStep.Warnings,
		Request:            finalStep.Request,
		Response: GenerateTextResponseMetadata{
			LanguageModelResponseMetadata: finalStep.Response.LanguageModelResponseMetadata,
			Messages:                      finalStep.Response.Messages,
			Body:                          finalStep.Response.Body,
		},
		ProviderMetadata: finalStep.ProviderMetadata,
		Steps:            d.steps,
		Output:           d.output,
	}
}

func toReasoningOutputs(parts []ReasoningPart) []ReasoningOutput {
	outputs := make([]ReasoningOutput, len(parts))
	for i, p := range parts {
		outputs[i] = ReasoningOutput{
			Type:             "reasoning",
			Text:             p.Text,
			ProviderMetadata: p.ProviderMetadata,
		}
	}
	return outputs
}
