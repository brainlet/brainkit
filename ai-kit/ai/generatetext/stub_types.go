// Ported from: various external packages (stub types for dependencies not yet ported)
// TODO: import from correct packages once they exist
package generatetext

import "time"

// --- From @ai-sdk/provider ---

// JSONSchema7 represents a JSON Schema Draft 7 object.
// TODO: import from brainlink/experiments/ai-kit/provider
type JSONSchema7 = map[string]interface{}

// LanguageModelV4ToolCall represents a tool call from the language model.
// TODO: import from brainlink/experiments/ai-kit/provider
type LanguageModelV4ToolCall struct {
	Type             string
	ToolCallID       string
	ToolName         string
	Input            string
	ProviderExecuted bool
	Dynamic          bool
	ProviderMetadata ProviderMetadata
}

// LanguageModelV4Content is a union type for model content parts.
// TODO: import from brainlink/experiments/ai-kit/provider
type LanguageModelV4Content struct {
	Type string

	// For text parts
	Text string

	// For file parts
	Data      interface{} // string | []byte
	MediaType string

	// For tool-call parts
	ToolCallID string
	ToolName   string
	Input      string

	// For tool-result parts
	Result  interface{}
	IsError bool

	// For tool-approval-request parts
	ApprovalID string

	// Common
	ProviderExecuted bool
	Dynamic          bool
	ProviderMetadata ProviderMetadata
}

// LanguageModelV4Reasoning represents a reasoning content part.
// TODO: import from brainlink/experiments/ai-kit/provider
type LanguageModelV4Reasoning struct {
	Type string
	Text string
}

// LanguageModelV4Text represents a text content part.
// TODO: import from brainlink/experiments/ai-kit/provider
type LanguageModelV4Text struct {
	Type string
	Text string
}

// LanguageModelV4ToolChoice represents tool choice configuration.
// TODO: import from brainlink/experiments/ai-kit/provider
type LanguageModelV4ToolChoice struct {
	Type     string
	ToolName string
}

// LanguageModelV4CallOptions represents call options.
// TODO: import from brainlink/experiments/ai-kit/provider
type LanguageModelV4CallOptions struct {
	ResponseFormat *ResponseFormat
}

// ResponseFormat describes the expected response format.
type ResponseFormat struct {
	Type        string
	Schema      JSONSchema7
	Name        string
	Description string
}

// SharedV4Warning represents a provider warning.
// TODO: import from brainlink/experiments/ai-kit/provider
type SharedV4Warning struct {
	Type    string
	Message string
}

// SharedV4ProviderMetadata is provider-specific metadata.
// TODO: import from brainlink/experiments/ai-kit/provider
type SharedV4ProviderMetadata = map[string]map[string]interface{}

// LanguageModelV4StreamPart represents a streaming chunk from the model.
// TODO: import from brainlink/experiments/ai-kit/provider
type LanguageModelV4StreamPart struct {
	Type string

	// For text/reasoning deltas
	ID    string
	Delta string
	Text  string

	// For tool-input parts
	ToolName string

	// For tool-call parts
	ToolCallID string
	Input      string

	// For tool-result parts
	Result  interface{}
	IsError bool

	// For file parts
	Data      interface{} // string | []byte
	MediaType string

	// For finish parts
	FinishReason struct {
		Unified FinishReason
		Raw     string
	}
	Usage struct {
		InputTokens  TokenCount
		OutputTokens TokenCount
	}

	// For source parts
	Source *Source

	// For stream-start parts
	Warnings []SharedV4Warning

	// For response-metadata parts
	Timestamp *time.Time
	ModelID   string

	// For tool-approval-request parts
	ApprovalID string

	// Common
	ProviderExecuted bool
	Dynamic          bool
	ProviderMetadata ProviderMetadata
	RawValue         interface{}
	Error            interface{}
}

// TokenCount represents a token count with total.
type TokenCount struct {
	Total int
}

// --- From @ai-sdk/provider-utils ---

// ModelMessage represents a message in the model conversation.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type ModelMessage struct {
	Role    string
	Content interface{} // string | []ModelMessageContent
}

// ModelMessageContent represents a content part in a model message.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type ModelMessageContent struct {
	Type string

	// Varies by type
	Text             string
	ToolCallID       string
	ToolName         string
	Input            interface{}
	Output           interface{}
	ApprovalID       string
	Approved         bool
	Reason           string
	Data             interface{}
	MediaType        string
	ProviderExecuted bool
	ProviderOptions  ProviderOptions
}

// SystemModelMessage represents a system message.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type SystemModelMessage struct {
	Role    string
	Content string
}

// ProviderOptions is a map of provider-specific options.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type ProviderOptions = map[string]map[string]interface{}

// ReasoningPart represents a reasoning segment from the model.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type ReasoningPart struct {
	Type             string
	Text             string
	ProviderMetadata ProviderMetadata
}

// ToolApprovalRequest represents a request for tool approval.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type ToolApprovalRequest struct {
	Type       string
	ApprovalID string
	ToolCallID string
}

// ToolApprovalResponse represents a response to a tool approval request.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type ToolApprovalResponse struct {
	Type             string
	ApprovalID       string
	Approved         bool
	Reason           string
	ProviderExecuted bool
}

// IdGenerator is a function that generates unique IDs.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type IdGenerator func() string

// AssistantModelMessage represents an assistant message.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type AssistantModelMessage = ModelMessage

// ToolModelMessage represents a tool message.
// TODO: import from brainlink/experiments/ai-kit/providerutils
type ToolModelMessage = ModelMessage

// --- From packages/ai/src/types ---

// ProviderMetadata holds provider-specific metadata.
// TODO: import from brainlink/experiments/ai-kit/types
type ProviderMetadata = map[string]map[string]interface{}

// FinishReason represents why a language model finished generating.
// TODO: import from brainlink/experiments/ai-kit/types
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content-filter"
	FinishReasonToolCalls     FinishReason = "tool-calls"
	FinishReasonError         FinishReason = "error"
	FinishReasonOther         FinishReason = "other"
)

// CallWarning represents a warning from a model provider.
// TODO: import from brainlink/experiments/ai-kit/types
type CallWarning struct {
	Type    string
	Message string
}

// LanguageModelUsage represents token usage from a model call.
// TODO: import from brainlink/experiments/ai-kit/types
type LanguageModelUsage struct {
	InputTokens       *int
	OutputTokens      *int
	TotalTokens       *int
	ReasoningTokens   *int
	CachedInputTokens *int
}

// LanguageModelRequestMetadata holds request metadata.
// TODO: import from brainlink/experiments/ai-kit/types
type LanguageModelRequestMetadata struct {
	Body interface{}
}

// LanguageModelResponseMetadata holds response metadata.
// TODO: import from brainlink/experiments/ai-kit/types
type LanguageModelResponseMetadata struct {
	ID        string
	Timestamp time.Time
	ModelID   string
	Headers   map[string]string
	Body      interface{}
}

// Source represents a reference source used during generation.
// TODO: import from brainlink/experiments/ai-kit/types
type Source struct {
	Type        string
	ID          string
	URL         string
	Title       string
	Description string
	Provenance  interface{}
}

// LanguageModel represents a language model instance.
// TODO: import from brainlink/experiments/ai-kit/types
type LanguageModel interface {
	Provider() string
	ModelID() string
}

// ToolChoice represents the tool choice strategy.
// TODO: import from brainlink/experiments/ai-kit/types
type ToolChoice struct {
	Type     string
	ToolName string
}

// --- From packages/ai/src/prompt ---

// TimeoutConfiguration holds timeout settings.
// TODO: import from brainlink/experiments/ai-kit/prompt
type TimeoutConfiguration struct {
	TotalMs *int
	StepMs  *int
	ChunkMs *int
}

// --- From packages/ai/src/util ---

// DeepPartial is a type alias for partial JSON values (Go has no equivalent generic).
// TODO: proper implementation
type DeepPartial = interface{}

// --- Utility types ---

// AddLanguageModelUsage combines two usage structs.
func AddLanguageModelUsage(a, b LanguageModelUsage) LanguageModelUsage {
	add := func(x, y *int) *int {
		if x == nil && y == nil {
			return nil
		}
		xv, yv := 0, 0
		if x != nil {
			xv = *x
		}
		if y != nil {
			yv = *y
		}
		result := xv + yv
		return &result
	}
	return LanguageModelUsage{
		InputTokens:       add(a.InputTokens, b.InputTokens),
		OutputTokens:      add(a.OutputTokens, b.OutputTokens),
		TotalTokens:       add(a.TotalTokens, b.TotalTokens),
		ReasoningTokens:   add(a.ReasoningTokens, b.ReasoningTokens),
		CachedInputTokens: add(a.CachedInputTokens, b.CachedInputTokens),
	}
}

// AsLanguageModelUsage converts raw token counts to LanguageModelUsage.
func AsLanguageModelUsage(usage struct {
	InputTokens  TokenCount
	OutputTokens TokenCount
}) LanguageModelUsage {
	input := usage.InputTokens.Total
	output := usage.OutputTokens.Total
	total := input + output
	return LanguageModelUsage{
		InputTokens:  &input,
		OutputTokens: &output,
		TotalTokens:  &total,
	}
}
