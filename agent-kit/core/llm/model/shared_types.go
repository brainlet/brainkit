// Ported from: packages/core/src/llm/model/shared.types.ts
package model

import (
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ---------------------------------------------------------------------------
// AI SDK type stubs
//
// The AI SDK V3 (LanguageModel) is fully ported in ai-kit:
//   brainlink/experiments/ai-kit/provider/languagemodel
//
// The V6 adapter (aisdk/v6) imports and wraps lm.LanguageModel directly.
// These Mastra-level types below have DIFFERENT signatures from the raw
// ai-kit interface (e.g., DoGenerate returns StreamResult instead of
// GenerateResult) because Mastra unifies both methods to return streams.
//
// AI SDK V2 (LanguageModelV2) is NOT ported in ai-kit — stubs remain.
// AI SDK V1 (LanguageModelV1) is legacy — stubs remain.
// ---------------------------------------------------------------------------

// LanguageModelV1 is a stub for the AI SDK v4 LanguageModelV1 interface.
// V1 is legacy and not available in ai-kit.
type LanguageModelV1 interface {
	// SpecificationVersion returns the version string (e.g., "v1").
	SpecificationVersion() string
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
}

// LanguageModelV2 is a stub for the AI SDK v5 LanguageModelV2 interface.
// V2 is not ported in ai-kit (only V3 is available).
type LanguageModelV2 interface {
	// SpecificationVersion returns "v2".
	SpecificationVersion() string
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoGenerate performs a non-streaming generation.
	DoGenerate(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error)
	// DoStream performs a streaming generation.
	DoStream(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error)
}

// LanguageModelV3 is the Mastra-level V3 interface with unified streaming.
// NOTE: This differs from ai-kit's lm.LanguageModel — Mastra's version
// makes DoGenerate return a StreamResult (not GenerateResult).
// The raw ai-kit interface is used internally by aisdk/v6.AISDKV6LanguageModel.
type LanguageModelV3 interface {
	// SpecificationVersion returns "v3".
	SpecificationVersion() string
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoGenerate performs a non-streaming generation (returns stream for Mastra compatibility).
	DoGenerate(options LanguageModelV3CallOptions) (LanguageModelV3StreamResult, error)
	// DoStream performs a streaming generation.
	DoStream(options LanguageModelV3CallOptions) (LanguageModelV3StreamResult, error)
}

// LanguageModelV2CallOptions is a stub for AI SDK v5 call options.
// V2 is not ported in ai-kit.
type LanguageModelV2CallOptions struct {
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`
}

// LanguageModelV3CallOptions is the Mastra-level V3 call options.
// The full ai-kit equivalent is languagemodel.CallOptions in
// brainlink/experiments/ai-kit/provider/languagemodel.
type LanguageModelV3CallOptions struct {
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`
}

// LanguageModelV2StreamResult is a stub for AI SDK v5 stream result.
// V2 is not ported in ai-kit.
type LanguageModelV2StreamResult struct {
	Stream any `json:"stream,omitempty"`
}

// LanguageModelV3StreamResult is the Mastra-level V3 stream result.
// The full ai-kit equivalent is languagemodel.StreamResult in
// brainlink/experiments/ai-kit/provider/languagemodel.
type LanguageModelV3StreamResult struct {
	Stream any `json:"stream,omitempty"`
}

// SharedV2ProviderOptions is a stub for AI SDK v5 shared provider options.
// V2 is not ported in ai-kit.
type SharedV2ProviderOptions = map[string]any

// SharedV3ProviderOptions maps to ai-kit's shared.ProviderOptions
// (map[string]map[string]any). Kept as map[string]any here for
// backward compatibility with existing Mastra code.
type SharedV3ProviderOptions = map[string]any

// ---------------------------------------------------------------------------
// Observability stubs
// ---------------------------------------------------------------------------

// TracingPolicy is re-exported from observability/types.
type TracingPolicy = obstypes.TracingPolicy

// ---------------------------------------------------------------------------
// ScoringData stub
// ---------------------------------------------------------------------------

// ScoringData holds the input/output of a scoring run.
// Corresponds to base.types.ts ScoringData.
type ScoringData struct {
	Input  map[string]any `json:"input,omitempty"`
	Output map[string]any `json:"output,omitempty"`
}

// ---------------------------------------------------------------------------
// TripwireProperties
// ---------------------------------------------------------------------------

// TripwireProperties holds tripwire data when processing was aborted.
type TripwireProperties struct {
	// Tripwire is set when processing was aborted by a tripwire.
	Tripwire *TripwireData `json:"tripwire,omitempty"`
}

// TripwireData describes why processing was aborted.
type TripwireData struct {
	Reason      string `json:"reason"`
	Retry       *bool  `json:"retry,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
	ProcessorID string `json:"processorId,omitempty"`
}

// ---------------------------------------------------------------------------
// ScoringProperties
// ---------------------------------------------------------------------------

// ScoringProperties contains optional scoring data for a generation result.
type ScoringProperties struct {
	ScoringData *ScoringData `json:"scoringData,omitempty"`
}

// ---------------------------------------------------------------------------
// OpenAICompatibleConfig
// ---------------------------------------------------------------------------

// OpenAICompatibleConfig represents configuration for an OpenAI-compatible model.
// It supports two forms:
//   - ID-based: { ID: "openai/gpt-4o", URL: "...", APIKey: "...", Headers: {...} }
//   - Provider/Model-based: { ProviderID: "openai", ModelID: "gpt-4o", ... }
type OpenAICompatibleConfig struct {
	// ID is the model ID like "openai/gpt-4o" or "custom-provider/my-model".
	// Mutually exclusive with ProviderID/ModelID form.
	ID string `json:"id,omitempty"`
	// ProviderID is the provider identifier (e.g., "openai").
	// Used with ModelID for the provider/model form.
	ProviderID string `json:"providerId,omitempty"`
	// ModelID is the model identifier (e.g., "gpt-4o").
	// Used with ProviderID for the provider/model form.
	ModelID string `json:"modelId,omitempty"`
	// URL is an optional custom URL endpoint.
	URL string `json:"url,omitempty"`
	// APIKey is an optional API key (falls back to env vars).
	APIKey string `json:"apiKey,omitempty"`
	// Headers contains additional HTTP headers.
	Headers map[string]string `json:"headers,omitempty"`
}

// HasID returns true if this config uses the ID form.
func (c OpenAICompatibleConfig) HasID() bool {
	return c.ID != ""
}

// HasProviderModel returns true if this config uses the ProviderID/ModelID form.
func (c OpenAICompatibleConfig) HasProviderModel() bool {
	return c.ProviderID != "" && c.ModelID != ""
}

// ---------------------------------------------------------------------------
// MastraLanguageModelV2
// ---------------------------------------------------------------------------

// MastraLanguageModelV2 is a wrapped V2 model with unified doGenerate/doStream
// that both return stream results. This is the Mastra-specific extension of
// LanguageModelV2. The V2 adapter is in aisdk/v5.
type MastraLanguageModelV2 interface {
	LanguageModelV2
}

// ---------------------------------------------------------------------------
// MastraLanguageModelV3
// ---------------------------------------------------------------------------

// MastraLanguageModelV3 is a wrapped V3 model with unified doGenerate/doStream
// that both return stream results. This is the Mastra-specific extension of
// LanguageModelV3. The V3 adapter is in aisdk/v6, which wraps ai-kit's
// lm.LanguageModel (brainlink/experiments/ai-kit/provider/languagemodel).
type MastraLanguageModelV3 interface {
	LanguageModelV3
}

// ---------------------------------------------------------------------------
// MastraLanguageModel
// ---------------------------------------------------------------------------

// MastraLanguageModel is a union of modern language models (V2/V3).
// In Go this is represented as an interface that both V2 and V3 implement.
// The underlying raw AI SDK interface for V3 is lm.LanguageModel from
// brainlink/experiments/ai-kit/provider/languagemodel.
type MastraLanguageModel interface {
	// SpecificationVersion returns the version string ("v2" or "v3").
	SpecificationVersion() string
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
}

// MastraLegacyLanguageModel is an alias for LanguageModelV1.
type MastraLegacyLanguageModel = LanguageModelV1

// ---------------------------------------------------------------------------
// SharedProviderOptions
// ---------------------------------------------------------------------------

// SharedProviderOptions is a union of V2 and V3 shared provider options.
// In Go, both are map[string]any so this is a type alias.
type SharedProviderOptions = map[string]any

// ---------------------------------------------------------------------------
// ModelRouterModelId (forward declaration)
// ---------------------------------------------------------------------------

// ModelRouterModelID is the type for model router model ID strings
// like "openai/gpt-4o". The exhaustive union is defined in the
// provider-registry generated types, but for the Go port we use string.
type ModelRouterModelID = string

// ---------------------------------------------------------------------------
// MastraModelConfig
// ---------------------------------------------------------------------------

// MastraModelConfig represents a model configuration that can be one of:
//   - A LanguageModelV1 instance
//   - A LanguageModelV2 instance
//   - A LanguageModelV3 instance
//   - A ModelRouterModelID string (like "openai/gpt-4o")
//   - An OpenAICompatibleConfig object
//   - A MastraLanguageModel instance
//
// In Go, we represent this as an interface{} / any since Go does not have
// discriminated unions. Consumers should use type switches or type guards.
type MastraModelConfig = any

// ---------------------------------------------------------------------------
// MastraModelOptions
// ---------------------------------------------------------------------------

// MastraModelOptions holds options for creating a Mastra model.
type MastraModelOptions struct {
	TracingPolicy *TracingPolicy `json:"tracingPolicy,omitempty"`
}
