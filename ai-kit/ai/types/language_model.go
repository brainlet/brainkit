// Ported from: packages/ai/src/types/language-model.ts
package aitypes

// LanguageModel is the language model used by the AI SDK.
//
// In TypeScript this is a union: GlobalProviderModelId | LanguageModelV4 | LanguageModelV3 | LanguageModelV2.
// In Go, we represent this as an interface that can hold either a string model ID
// or a model interface implementation.
type LanguageModel = any

// FinishReason indicates why a language model finished generating a response.
//
// Can be one of the following:
//   - "stop": model generated stop sequence
//   - "length": model generated maximum number of tokens
//   - "content-filter": content filter violation stopped the model
//   - "tool-calls": model triggered tool calls
//   - "error": model stopped because of an error
//   - "other": model stopped for other reasons
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content-filter"
	FinishReasonToolCalls     FinishReason = "tool-calls"
	FinishReasonError         FinishReason = "error"
	FinishReasonOther         FinishReason = "other"
)

// CallWarning is a warning from the model provider for this call.
// The call will proceed, but e.g. some settings might not be supported,
// which can lead to suboptimal results.
//
// Corresponds to SharedV4Warning from @ai-sdk/provider.
type CallWarning = Warning

// Source is a source that has been used as input to generate the response.
//
// Corresponds to LanguageModelV4Source from @ai-sdk/provider.
type Source struct {
	// Type is always "source".
	Type string `json:"type"`

	// SourceType is the type of source: "url" or "document".
	SourceType string `json:"sourceType"`

	// ID is the identifier of the source.
	ID string `json:"id"`

	// URL is the URL of the source. Applicable when SourceType is "url".
	URL string `json:"url,omitempty"`

	// Title is the title of the source.
	Title string `json:"title,omitempty"`

	// MediaType is the IANA media type of the document (e.g., 'application/pdf').
	// Applicable when SourceType is "document".
	MediaType string `json:"mediaType,omitempty"`

	// Filename is the optional filename of the document.
	// Applicable when SourceType is "document".
	Filename string `json:"filename,omitempty"`

	// ProviderMetadata is additional provider metadata for the source.
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// ToolChoice specifies the tool choice for the generation.
//
// It supports the following settings:
//   - "auto" (default): the model can choose whether and which tools to call.
//   - "required": the model must call a tool. It can choose which tool to call.
//   - "none": the model must not call tools.
//   - ToolChoiceSpecific{ToolName: "..."}: the model must call the specified tool.
type ToolChoice struct {
	// Type is "auto", "none", "required", or "tool".
	Type string `json:"type"`

	// ToolName is the name of the tool when Type is "tool".
	ToolName string `json:"toolName,omitempty"`
}

// ToolChoiceAuto returns a ToolChoice with type "auto".
func ToolChoiceAuto() ToolChoice {
	return ToolChoice{Type: "auto"}
}

// ToolChoiceNone returns a ToolChoice with type "none".
func ToolChoiceNone() ToolChoice {
	return ToolChoice{Type: "none"}
}

// ToolChoiceRequired returns a ToolChoice with type "required".
func ToolChoiceRequired() ToolChoice {
	return ToolChoice{Type: "required"}
}

// ToolChoiceTool returns a ToolChoice that forces the model to call a specific tool.
func ToolChoiceTool(toolName string) ToolChoice {
	return ToolChoice{Type: "tool", ToolName: toolName}
}
