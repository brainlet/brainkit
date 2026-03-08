// Ported from: packages/core/src/llm/index.ts
package llm

import (
	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
)

// ---------------------------------------------------------------------------
// Re-exported types from model package
// ---------------------------------------------------------------------------

// LanguageModel is the primary language model type (MastraLanguageModel).
type LanguageModel = model.MastraLanguageModel

// CoreMessage is the core message type for LLM interactions.
type CoreMessage = model.CoreMessage

// UIMessage is the UI message type for LLM interactions.
type UIMessage = model.UIMessage

// ---------------------------------------------------------------------------
// Structured output types
// ---------------------------------------------------------------------------

// BaseStructuredOutputType enumerates the base types for structured output fields.
type BaseStructuredOutputType string

const (
	BaseStructuredOutputString  BaseStructuredOutputType = "string"
	BaseStructuredOutputNumber  BaseStructuredOutputType = "number"
	BaseStructuredOutputBoolean BaseStructuredOutputType = "boolean"
	BaseStructuredOutputDate    BaseStructuredOutputType = "date"
)

// StructuredOutputType enumerates all types for structured output fields.
type StructuredOutputType string

const (
	StructuredOutputArray   StructuredOutputType = "array"
	StructuredOutputString  StructuredOutputType = "string"
	StructuredOutputNumber  StructuredOutputType = "number"
	StructuredOutputObject  StructuredOutputType = "object"
	StructuredOutputBoolean StructuredOutputType = "boolean"
	StructuredOutputDate    StructuredOutputType = "date"
)

// StructuredOutputArrayItem represents an item in a structured output array.
// It can be either a base type or a nested object.
type StructuredOutputArrayItem struct {
	Type  StructuredOutputType `json:"type"`
	Items StructuredOutput     `json:"items,omitempty"` // Only for type == "object"
}

// StructuredOutput is a map of field names to their type definitions.
// Each value specifies the field type and optional nested structure.
type StructuredOutput map[string]StructuredOutputField

// StructuredOutputField represents a field in structured output.
type StructuredOutputField struct {
	Type  StructuredOutputType       `json:"type"`
	Items any                        `json:"items,omitempty"` // StructuredOutput for "object", StructuredOutputArrayItem for "array"
}

// OutputType represents an output type definition -- structured output,
// JSON schema, or nil (for plain text).
type OutputType = any

// ---------------------------------------------------------------------------
// System message types
// ---------------------------------------------------------------------------

// SystemMessage represents a system message that can be:
//   - A single string
//   - A slice of strings
//   - A CoreSystemMessage
//   - A SystemModelMessage
//   - A slice of CoreSystemMessage
//   - A slice of SystemModelMessage
//
// In Go we use any since we cannot express this union directly.
type SystemMessage = any

// CoreSystemMessage is the core system message type.
type CoreSystemMessage = model.CoreMessage

// SystemModelMessage is a stub for the AI SDK v5 SystemModelMessage.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type SystemModelMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ---------------------------------------------------------------------------
// Re-exported result types from model package
// ---------------------------------------------------------------------------

// GenerateReturn is the return type for generate calls.
type GenerateReturn = model.GenerateReturn

// StreamReturn is the return type for stream calls.
type StreamReturn = model.StreamReturn

// GenerateObjectResult is the result of a structured object generation.
type GenerateObjectResult = model.GenerateObjectResult

// GenerateTextResult is the result of a text generation.
type GenerateTextResult = model.GenerateTextResult

// StreamObjectResult is the result of a streaming object generation.
type StreamObjectResult = model.StreamObjectResult

// StreamTextResult is the result of a streaming text generation.
type StreamTextResult = model.StreamTextResult

// ---------------------------------------------------------------------------
// Re-exported configuration types
// ---------------------------------------------------------------------------

// TripwireProperties holds tripwire data when processing was aborted.
type TripwireProperties = model.TripwireProperties

// MastraModelConfig represents all supported model configuration forms.
type MastraModelConfig = model.MastraModelConfig

// OpenAICompatibleConfig represents an OpenAI-compatible model config.
type OpenAICompatibleConfig = model.OpenAICompatibleConfig

// ---------------------------------------------------------------------------
// Re-exported from model sub-packages
// ---------------------------------------------------------------------------

// ModelRouterLanguageModel is the model router that resolves model IDs.
type ModelRouterLanguageModel = model.ModelRouterLanguageModel

// ProviderConfig holds configuration for a model provider.
type ProviderConfig = model.ProviderConfig

// MastraModelGateway is the interface for model gateways.
type MastraModelGateway = model.MastraModelGateway

// GatewayLanguageModel is the union type for gateway-resolved models.
type GatewayLanguageModel = model.GatewayLanguageModel

// EmbeddingModelInfo describes a known embedding model.
type EmbeddingModelInfo = model.EmbeddingModelInfo

// ModelRouterEmbeddingModel routes embedding model requests.
type ModelRouterEmbeddingModel = model.ModelRouterEmbeddingModel

// ---------------------------------------------------------------------------
// Re-exported functions
// ---------------------------------------------------------------------------

// ParseModelString parses a model string to extract provider and model ID.
var ParseModelString = model.ParseModelString

// GetProviderConfigByID retrieves a provider config from the registry.
var GetProviderConfigByID = model.GetProviderConfigByID

// ResolveModelConfig resolves a model configuration to a language model instance.
var ResolveModelConfig = model.ResolveModelConfig

// EMBEDDING_MODELS is the curated list of known embedding models.
var EMBEDDING_MODELS = model.EMBEDDING_MODELS

// ---------------------------------------------------------------------------
// LLM option types (from the root index.ts)
// ---------------------------------------------------------------------------

// LLMTextOptions holds options for text generation calls.
type LLMTextOptions struct {
	Messages           []any              `json:"messages"`
	Tools              map[string]any     `json:"tools,omitempty"`
	OnStepFinish       func(step any) error `json:"-"`
	ExperimentalOutput any                `json:"experimental_output,omitempty"`
	ThreadID           string             `json:"threadId,omitempty"`
	ResourceID         string             `json:"resourceId,omitempty"`
	// RequestContext is a stub; use requestcontext.RequestContext once finalized.
	RequestContext any `json:"-"`
	RunID          string `json:"runId,omitempty"`
}

// LLMStreamOptions holds options for streaming text generation calls.
type LLMStreamOptions struct {
	Output             OutputType         `json:"output,omitempty"`
	OnFinish           func(event any) error `json:"-"`
	Tools              map[string]any     `json:"tools,omitempty"`
	OnStepFinish       func(step any) error `json:"-"`
	ExperimentalOutput any                `json:"experimental_output,omitempty"`
	ThreadID           string             `json:"threadId,omitempty"`
	ResourceID         string             `json:"resourceId,omitempty"`
	RequestContext     any                `json:"-"`
	RunID              string             `json:"runId,omitempty"`
}

// LLMTextObjectOptions extends LLMTextOptions with a structured output requirement.
type LLMTextObjectOptions struct {
	LLMTextOptions
	StructuredOutput any `json:"structuredOutput"`
}

// LLMStreamObjectOptions extends LLMStreamOptions with a structured output requirement.
type LLMStreamObjectOptions struct {
	Messages         []any              `json:"messages"`
	StructuredOutput any                `json:"structuredOutput"`
	OnFinish         func(event any) error `json:"-"`
	Tools            map[string]any     `json:"tools,omitempty"`
	ThreadID         string             `json:"threadId,omitempty"`
	ResourceID       string             `json:"resourceId,omitempty"`
	RequestContext   any                `json:"-"`
	RunID            string             `json:"runId,omitempty"`
}
