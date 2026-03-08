// Ported from: packages/core/src/llm/model/base.types.ts
package model

import (
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ---------------------------------------------------------------------------
// Stub types for unported packages
// ---------------------------------------------------------------------------

// UIMessage is a stub for AI SDK UIMessage.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type UIMessage struct {
	ID      string `json:"id,omitempty"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CoreMessage is a stub for AI SDK CoreMessage.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type CoreMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// Tool is a stub for the AI SDK Tool type.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type Tool = any

// ToolSet is a map of tool names to Tool instances.
// TS: Record<string, Tool>
type ToolSet = map[string]Tool

// DeepPartial is a stub for the AI SDK DeepPartial utility type.
// In Go, we just use any since we can't express recursive partial types.
type DeepPartial = any

// MessageList is a stub for the agent MessageList type.
// STUB REASON: Cannot import agent due to circular dependency: agent imports llm/model.
// The real agent.MessageList is a complex struct. Using `= any` as placeholder.
type MessageList = any

// TracingProperties is re-exported from observability/types.
type TracingProperties = obstypes.TracingProperties

// ObservabilityContext is re-exported from observability/types.
type ObservabilityContext = obstypes.ObservabilityContext

// OutputProcessorOrWorkflow is a stub for the processors type.
// STUB REASON: The real processors.Processor interface has 10+ methods and depends on
// types from agent, stream, and observability. Using `= any` as a union placeholder
// for Processor | ProcessorWorkflow.
type OutputProcessorOrWorkflow = any

// ---------------------------------------------------------------------------
// GenerateText types
// ---------------------------------------------------------------------------

// GenerateTextResult represents the result of a text generation call.
type GenerateTextResult struct {
	// Text is the generated text content.
	Text string `json:"text,omitempty"`
	// Object is the structured output when experimental_output is used.
	Object any `json:"object,omitempty"`
	// FinishReason indicates why the generation stopped.
	FinishReason string `json:"finishReason,omitempty"`
	// Usage contains token usage information.
	Usage *TokenUsage `json:"usage,omitempty"`
	// Response contains response metadata.
	Response *ResponseMeta `json:"response,omitempty"`
	// Reasoning details (if available).
	ReasoningDetails any `json:"reasoningDetails,omitempty"`
	// Reasoning text (if available).
	Reasoning string `json:"reasoning,omitempty"`
	// Files returned by the model.
	Files any `json:"files,omitempty"`
	// Sources referenced by the model.
	Sources any `json:"sources,omitempty"`
	// Warnings from the model.
	Warnings []any `json:"warnings,omitempty"`
	// ToolCalls contains any tool calls made.
	ToolCalls any `json:"toolCalls,omitempty"`
	// ToolResults contains results from tool calls.
	ToolResults any `json:"toolResults,omitempty"`
	// MessageList holds the agent message list when applicable.
	MessageList MessageList `json:"messageList,omitempty"`

	TripwireProperties
	ScoringProperties
	TracingProperties
}

// GenerateObjectResult represents the result of a structured object generation call.
type GenerateObjectResult struct {
	// Object is the generated structured object.
	Object any `json:"object,omitempty"`
	// FinishReason indicates why the generation stopped.
	FinishReason string `json:"finishReason,omitempty"`
	// Usage contains token usage information.
	Usage *TokenUsage `json:"usage,omitempty"`
	// Response contains response metadata.
	Response *ResponseMeta `json:"response,omitempty"`
	// Warnings from the model.
	Warnings []any `json:"warnings,omitempty"`

	TripwireProperties
	ScoringProperties
	TracingProperties
}

// ---------------------------------------------------------------------------
// StreamText types
// ---------------------------------------------------------------------------

// StreamTextResult represents the result of a streaming text generation call.
type StreamTextResult struct {
	// Object is the structured output when experimental_output is used.
	Object any `json:"object,omitempty"`

	// The underlying stream and result accessor fields would be here
	// in a full port. For the type definition we capture the shape.

	TripwireProperties
	TracingProperties
}

// StreamObjectResult represents the result of a streaming object generation call.
type StreamObjectResult struct {
	TripwireProperties
}

// ---------------------------------------------------------------------------
// GenerateReturn / StreamReturn union types
// ---------------------------------------------------------------------------

// GenerateReturn is a union return type: either GenerateTextResult or GenerateObjectResult.
// In Go, callers should use type assertion.
type GenerateReturn = any

// StreamReturn is a union return type: either StreamTextResult or StreamObjectResult.
// In Go, callers should use type assertion.
type StreamReturn = any

// ---------------------------------------------------------------------------
// TokenUsage
// ---------------------------------------------------------------------------

// TokenUsage holds token usage statistics from a model call.
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// ---------------------------------------------------------------------------
// ResponseMeta
// ---------------------------------------------------------------------------

// ResponseMeta holds metadata about a model response.
type ResponseMeta struct {
	ID      string            `json:"id,omitempty"`
	ModelID string            `json:"modelId,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ---------------------------------------------------------------------------
// Callback types
// ---------------------------------------------------------------------------

// StreamTextOnFinishCallback is called when a text stream finishes.
// TS: (event: { ...props, runId: string }) => Promise<void> | void
type StreamTextOnFinishCallback func(event StreamTextFinishEvent) error

// StreamObjectOnFinishCallback is called when an object stream finishes.
type StreamObjectOnFinishCallback func(event StreamObjectFinishEvent) error

// GenerateTextOnStepFinishCallback is called when a generate text step finishes.
type GenerateTextOnStepFinishCallback func(event StepFinishEvent) error

// StreamTextOnStepFinishCallback is called when a stream text step finishes.
type StreamTextOnStepFinishCallback func(event StepFinishEvent) error

// StreamTextFinishEvent is the event passed to StreamTextOnFinishCallback.
type StreamTextFinishEvent struct {
	RunID        string     `json:"runId"`
	Text         string     `json:"text,omitempty"`
	FinishReason string     `json:"finishReason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
	ToolCalls    any        `json:"toolCalls,omitempty"`
	ToolResults  any        `json:"toolResults,omitempty"`
}

// StreamObjectFinishEvent is the event passed to StreamObjectOnFinishCallback.
type StreamObjectFinishEvent struct {
	RunID        string     `json:"runId"`
	Object       any        `json:"object,omitempty"`
	FinishReason string     `json:"finishReason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
}

// StepFinishEvent is the event passed to step finish callbacks.
type StepFinishEvent struct {
	RunID        string     `json:"runId"`
	Text         string     `json:"text,omitempty"`
	FinishReason string     `json:"finishReason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
	ToolCalls    any        `json:"toolCalls,omitempty"`
	ToolResults  any        `json:"toolResults,omitempty"`
	Response     *ResponseMeta `json:"response,omitempty"`
}
