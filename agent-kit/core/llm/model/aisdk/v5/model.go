// Ported from: packages/core/src/llm/model/aisdk/v5/model.ts
package v5

import (
	"regexp"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model/aisdk"
)

// ---------------------------------------------------------------------------
// AI SDK v5 types (LanguageModelV2)
//
// ai-kit only ported the latest AI SDK (V3/V6). V2/V5 types remain as
// local definitions. The V3 equivalent is in:
//   brainlink/experiments/ai-kit/provider/languagemodel
// See aisdk/v6/model.go for the V3 adapter that imports from ai-kit.
// ---------------------------------------------------------------------------

// LanguageModelV2CallOptions is a stub for AI SDK v5 call options.
// ai-kit does not have V2 types — only V3 is ported.
type LanguageModelV2CallOptions struct {
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`
}

// LanguageModelV2StreamResult is a stub for AI SDK v5 stream result.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type LanguageModelV2StreamResult struct {
	Request  any                    `json:"request,omitempty"`
	Response *StreamResultResponse  `json:"response,omitempty"`
	Stream   <-chan aisdk.StreamEvent `json:"stream,omitempty"`
}

// StreamResultResponse holds the response metadata from a stream result.
type StreamResultResponse struct {
	ID        string `json:"id,omitempty"`
	ModelID   string `json:"modelId,omitempty"`
	Timestamp any    `json:"timestamp,omitempty"`
}

// LanguageModelV2 is a stub for the AI SDK v5 LanguageModelV2 interface.
// ai-kit does not have V2 types — only V3 (lm.LanguageModel) is ported.
type LanguageModelV2 interface {
	// SpecificationVersion returns "v2".
	SpecificationVersion() string
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// SupportedURLs returns supported URL patterns by media type.
	SupportedURLs() map[string][]*regexp.Regexp
	// DoGenerate performs a non-streaming generation.
	DoGenerate(options LanguageModelV2CallOptions) (*DoGenerateResult, error)
	// DoStream performs a streaming generation.
	DoStream(options LanguageModelV2CallOptions) (*LanguageModelV2StreamResult, error)
}

// DoGenerateResult holds the result from LanguageModelV2.DoGenerate.
type DoGenerateResult struct {
	Warnings         []any                        `json:"warnings"`
	Request          any                          `json:"request,omitempty"`
	Response         *aisdk.GenerateResultResponse `json:"response,omitempty"`
	Content          []map[string]any             `json:"content"`
	FinishReason     any                          `json:"finishReason"`
	Usage            any                          `json:"usage"`
	ProviderMetadata any                          `json:"providerMetadata,omitempty"`
}

// ---------------------------------------------------------------------------
// MastraLanguageModelV2 — the Mastra wrapper interface
// ---------------------------------------------------------------------------

// MastraLanguageModelV2 is a wrapped V2 model with unified doGenerate/doStream
// that both return stream results. This is the Mastra-specific extension.
// See aisdk/v6 for the V3 equivalent that imports from ai-kit.
type MastraLanguageModelV2 interface {
	SpecificationVersion() string
	Provider() string
	ModelID() string
	SupportedURLs() map[string][]*regexp.Regexp
	DoGenerate(options LanguageModelV2CallOptions) (*LanguageModelV2StreamResult, error)
	DoStream(options LanguageModelV2CallOptions) (*LanguageModelV2StreamResult, error)
}

// ---------------------------------------------------------------------------
// AISDKV5LanguageModel
// ---------------------------------------------------------------------------

// AISDKV5LanguageModel wraps an AI SDK V5 (LanguageModelV2) to convert
// doGenerate to return a stream format for consistency with Mastra's
// streaming architecture.
type AISDKV5LanguageModel struct {
	specificationVersion string
	provider             string
	modelID              string
	supportedURLs        map[string][]*regexp.Regexp
	model                LanguageModelV2
}

// Compile-time check that AISDKV5LanguageModel implements MastraLanguageModelV2.
var _ MastraLanguageModelV2 = (*AISDKV5LanguageModel)(nil)

// NewAISDKV5LanguageModel creates a new AISDKV5LanguageModel wrapper.
func NewAISDKV5LanguageModel(model LanguageModelV2) *AISDKV5LanguageModel {
	return &AISDKV5LanguageModel{
		specificationVersion: "v2",
		provider:             model.Provider(),
		modelID:              model.ModelID(),
		supportedURLs:        model.SupportedURLs(),
		model:                model,
	}
}

// SpecificationVersion implements MastraLanguageModelV2.
func (m *AISDKV5LanguageModel) SpecificationVersion() string { return m.specificationVersion }

// Provider implements MastraLanguageModelV2.
func (m *AISDKV5LanguageModel) Provider() string { return m.provider }

// ModelID implements MastraLanguageModelV2.
func (m *AISDKV5LanguageModel) ModelID() string { return m.modelID }

// SupportedURLs implements MastraLanguageModelV2.
func (m *AISDKV5LanguageModel) SupportedURLs() map[string][]*regexp.Regexp {
	return m.supportedURLs
}

// DoGenerate wraps the underlying model's DoGenerate, converting the result
// to a stream format for consistency with Mastra's streaming architecture.
func (m *AISDKV5LanguageModel) DoGenerate(options LanguageModelV2CallOptions) (*LanguageModelV2StreamResult, error) {
	result, err := m.model.DoGenerate(options)
	if err != nil {
		return nil, err
	}

	genResult := &aisdk.GenerateResult{
		Warnings:         result.Warnings,
		Response:         result.Response,
		Content:          result.Content,
		FinishReason:     result.FinishReason,
		Usage:            result.Usage,
		ProviderMetadata: result.ProviderMetadata,
	}

	var resp *StreamResultResponse
	if result.Response != nil {
		resp = &StreamResultResponse{
			ID:        result.Response.ID,
			ModelID:   result.Response.ModelID,
			Timestamp: result.Response.Timestamp,
		}
	}

	return &LanguageModelV2StreamResult{
		Request:  result.Request,
		Response: resp,
		Stream:   aisdk.CreateStreamFromGenerateResult(genResult),
	}, nil
}

// DoStream delegates to the underlying model's DoStream.
func (m *AISDKV5LanguageModel) DoStream(options LanguageModelV2CallOptions) (*LanguageModelV2StreamResult, error) {
	return m.model.DoStream(options)
}
