// Ported from: packages/core/src/loop/workflows/schema.ts
package workflows

import (
	"time"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ReasoningPart is a stub for @ai-sdk/provider-utils-v5.ReasoningPart.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider-utils remain local stubs.
type ReasoningPart struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Details []any          `json:"details,omitempty"`
	Meta    map[string]any `json:"providerMetadata,omitempty"`
}

// LanguageModelV2FinishReason is a stub for @ai-sdk/provider-v5.LanguageModelV2FinishReason.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2FinishReason = string

// LanguageModelV2CallWarning is a stub for @ai-sdk/provider-v5.LanguageModelV2CallWarning.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2CallWarning = any

// SharedV2ProviderMetadata is a stub for @ai-sdk/provider-v5.SharedV2ProviderMetadata.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type SharedV2ProviderMetadata = map[string]any

// LanguageModelV2Source is a stub for @ai-sdk/provider-v5.LanguageModelV2Source.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2Source = any

// LanguageModelRequestMetadata is a stub for @internal/ai-sdk-v4.LanguageModelRequestMetadata.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V4 internal types remain local stubs.
type LanguageModelRequestMetadata = map[string]any

// LanguageModelV1LogProbs is a stub for @internal/ai-sdk-v4.LogProbs.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V4 internal types remain local stubs.
type LanguageModelV1LogProbs = any

// StepResult is a stub for @internal/ai-sdk-v5.StepResult.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type StepResult = any

// ModelMessage is a stub for @internal/ai-sdk-v5.ModelMessage.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type ModelMessage = any

// LanguageModelUsage is a stub for @internal/ai-sdk-v5.LanguageModelUsage.
// Stub: stream.LanguageModelUsage exists but uses int fields (not *int pointers) and
// has additional Raw field. V5 version uses optional pointer fields. Shape mismatch.
type LanguageModelUsage struct {
	InputTokens      *int `json:"inputTokens,omitempty"`
	OutputTokens     *int `json:"outputTokens,omitempty"`
	TotalTokens      *int `json:"totalTokens,omitempty"`
	ReasoningTokens  *int `json:"reasoningTokens,omitempty"`
	CachedInputTokens *int `json:"cachedInputTokens,omitempty"`
}

// TypedToolCall is a stub for @internal/ai-sdk-v5.TypedToolCall.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type TypedToolCall = any

// TypedToolResult is a stub for @internal/ai-sdk-v5.TypedToolResult.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type TypedToolResult = any

// StaticToolCall is a stub for @internal/ai-sdk-v5.StaticToolCall.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type StaticToolCall = any

// StaticToolResult is a stub for @internal/ai-sdk-v5.StaticToolResult.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type StaticToolResult = any

// DynamicToolCall is a stub for @internal/ai-sdk-v5.DynamicToolCall.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type DynamicToolCall = any

// DynamicToolResult is a stub for @internal/ai-sdk-v5.DynamicToolResult.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type DynamicToolResult = any

// GeneratedFile is a stub for @internal/ai-sdk-v5.GeneratedFile.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type GeneratedFile = any

// ToolSet is a stub for @internal/ai-sdk-v5.ToolSet.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 ToolSet remains local.
// model.ToolSet = map[string]Tool where Tool = any — same shape but different V5 context.
type ToolSet = map[string]any

// ---------------------------------------------------------------------------
// LLMIterationStepResult
// ---------------------------------------------------------------------------

// LLMIterationStepResult holds per-step result metadata from an LLM
// iteration. The Reason field can include 'tripwire' and 'retry' for
// processor scenarios.
type LLMIterationStepResult struct {
	Reason      string                        `json:"reason"`
	Warnings    []LanguageModelV2CallWarning  `json:"warnings"`
	IsContinued bool                          `json:"isContinued"`
	LogProbs    LanguageModelV1LogProbs       `json:"logprobs,omitempty"`
	TotalUsage  *LanguageModelUsage           `json:"totalUsage,omitempty"`
	Headers     map[string]string             `json:"headers,omitempty"`
	MessageID   string                        `json:"messageId,omitempty"`
	Request     LanguageModelRequestMetadata  `json:"request,omitempty"`
}

// ---------------------------------------------------------------------------
// LLMIterationOutput
// ---------------------------------------------------------------------------

// LLMIterationOutput holds the output from one LLM iteration.
type LLMIterationOutput struct {
	Text               string               `json:"text,omitempty"`
	Reasoning          []ReasoningPart       `json:"reasoning,omitempty"`
	ReasoningText      string               `json:"reasoningText,omitempty"`
	Files              []GeneratedFile       `json:"files,omitempty"`
	ToolCalls          []TypedToolCall       `json:"toolCalls,omitempty"`
	ToolResults        []TypedToolResult     `json:"toolResults,omitempty"`
	Sources            []LanguageModelV2Source `json:"sources,omitempty"`
	StaticToolCalls    []StaticToolCall      `json:"staticToolCalls,omitempty"`
	DynamicToolCalls   []DynamicToolCall     `json:"dynamicToolCalls,omitempty"`
	StaticToolResults  []StaticToolResult    `json:"staticToolResults,omitempty"`
	DynamicToolResults []DynamicToolResult   `json:"dynamicToolResults,omitempty"`
	Usage              LanguageModelUsage    `json:"usage"`
	Steps              []StepResult          `json:"steps"`
	Object             any                   `json:"object,omitempty"`
}

// ---------------------------------------------------------------------------
// LLMIterationMetadata
// ---------------------------------------------------------------------------

// LLMIterationMetadata holds metadata about the model and provider for one
// LLM iteration.
type LLMIterationMetadata struct {
	ID               string                  `json:"id,omitempty"`
	Model            string                  `json:"model,omitempty"`
	ModelID          string                  `json:"modelId,omitempty"`
	ModelMetadata    *SchemaModelMetadata    `json:"modelMetadata,omitempty"`
	Timestamp        *time.Time              `json:"timestamp,omitempty"`
	ProviderMetadata SharedV2ProviderMetadata `json:"providerMetadata,omitempty"`
	Headers          map[string]string       `json:"headers,omitempty"`
	Request          LanguageModelRequestMetadata `json:"request,omitempty"`
}

// SchemaModelMetadata is an inline sub-struct for LLMIterationMetadata.
type SchemaModelMetadata struct {
	ModelID       string `json:"modelId"`
	ModelVersion  string `json:"modelVersion"`
	ModelProvider string `json:"modelProvider"`
}

// ---------------------------------------------------------------------------
// LLMIterationData
// ---------------------------------------------------------------------------

// LLMIterationMessages groups all, user, and non-user model messages.
type LLMIterationMessages struct {
	All     []ModelMessage `json:"all"`
	User    []ModelMessage `json:"user"`
	NonUser []ModelMessage `json:"nonUser"`
}

// LLMIterationData is the primary data structure flowing through the
// agentic workflow. It corresponds to the TS LLMIterationData<Tools, OUTPUT>.
type LLMIterationData struct {
	MessageID string               `json:"messageId"`
	Messages  LLMIterationMessages `json:"messages"`
	Output    LLMIterationOutput   `json:"output"`
	Metadata  LLMIterationMetadata `json:"metadata"`
	StepResult LLMIterationStepResult `json:"stepResult"`
	// ProcessorRetryCount tracks the number of times processors have triggered
	// retry for this generation. Used to enforce MaxProcessorRetries limit.
	ProcessorRetryCount int `json:"processorRetryCount,omitempty"`
	// ProcessorRetryFeedback is the feedback message from the processor to be
	// added as a system message on retry.
	ProcessorRetryFeedback string `json:"processorRetryFeedback,omitempty"`
	// IsTaskCompleteCheckFailed is true if the isTaskComplete check failed
	// and the LLM has to run again.
	IsTaskCompleteCheckFailed bool `json:"isTaskCompleteCheckFailed,omitempty"`
}

// ---------------------------------------------------------------------------
// ToolCallInput / ToolCallOutput
// ---------------------------------------------------------------------------

// ToolCallInput represents the input for a single tool call step. It
// corresponds to the TS toolCallInputSchema.
type ToolCallInput struct {
	ToolCallID       string         `json:"toolCallId"`
	ToolName         string         `json:"toolName"`
	Args             map[string]any `json:"args"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
	ProviderExecuted *bool          `json:"providerExecuted,omitempty"`
	Output           any            `json:"output,omitempty"`
}

// ToolCallOutput extends ToolCallInput with execution results. It
// corresponds to the TS toolCallOutputSchema.
type ToolCallOutput struct {
	ToolCallInput
	Result any `json:"result,omitempty"`
	Error  any `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

// ValidateLLMIterationStepResult performs basic runtime validation on an
// LLMIterationStepResult, returning an error if required fields are missing.
// This mirrors the Zod llmIterationStepResultSchema.
func ValidateLLMIterationStepResult(sr *LLMIterationStepResult) error {
	if sr == nil {
		return &validationError{Field: "stepResult", Msg: "cannot be nil"}
	}
	if sr.Reason == "" {
		return &validationError{Field: "stepResult.reason", Msg: "cannot be empty"}
	}
	return nil
}

// ValidateLLMIterationData performs basic runtime validation on an
// LLMIterationData, returning an error if required fields are missing.
// This mirrors the Zod llmIterationOutputSchema.
func ValidateLLMIterationData(d *LLMIterationData) error {
	if d == nil {
		return &validationError{Field: "data", Msg: "cannot be nil"}
	}
	if d.MessageID == "" {
		return &validationError{Field: "messageId", Msg: "cannot be empty"}
	}
	return ValidateLLMIterationStepResult(&d.StepResult)
}

// ValidateToolCallInput performs basic runtime validation on a ToolCallInput.
// This mirrors the Zod toolCallInputSchema.
func ValidateToolCallInput(tc *ToolCallInput) error {
	if tc == nil {
		return &validationError{Field: "toolCallInput", Msg: "cannot be nil"}
	}
	if tc.ToolCallID == "" {
		return &validationError{Field: "toolCallId", Msg: "cannot be empty"}
	}
	if tc.ToolName == "" {
		return &validationError{Field: "toolName", Msg: "cannot be empty"}
	}
	return nil
}

// validationError is a simple validation error type.
type validationError struct {
	Field string
	Msg   string
}

func (e *validationError) Error() string {
	return e.Field + ": " + e.Msg
}
