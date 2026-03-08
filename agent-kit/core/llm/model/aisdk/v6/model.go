// Ported from: packages/core/src/llm/model/aisdk/v6/model.ts
package v6

import (
	"regexp"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model/aisdk"
	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// ---------------------------------------------------------------------------
// Local Mastra types
// ---------------------------------------------------------------------------

// LanguageModelV3CallOptions are the Mastra-level call options passed through
// the V6 adapter. These are a simplified subset of ai-kit's CallOptions
// (brainlink/experiments/ai-kit/provider/languagemodel.CallOptions).
// TODO: Consider using lm.CallOptions directly as the codebase matures.
type LanguageModelV3CallOptions struct {
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`
}

// LanguageModelV3StreamResult is the Mastra-level stream result.
// Mastra unifies doGenerate and doStream to both return stream results.
type LanguageModelV3StreamResult struct {
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

// ---------------------------------------------------------------------------
// MastraLanguageModelV3 — the Mastra wrapper interface
// ---------------------------------------------------------------------------

// MastraLanguageModelV3 is a wrapped V3 model with unified doGenerate/doStream
// that both return stream results. This is the Mastra-specific extension.
//
// The raw AI SDK V3 interface is lm.LanguageModel from
// brainlink/experiments/ai-kit/provider/languagemodel.
type MastraLanguageModelV3 interface {
	SpecificationVersion() string
	Provider() string
	ModelID() string
	SupportedURLs() map[string][]*regexp.Regexp
	DoGenerate(options LanguageModelV3CallOptions) (*LanguageModelV3StreamResult, error)
	DoStream(options LanguageModelV3CallOptions) (*LanguageModelV3StreamResult, error)
}

// ---------------------------------------------------------------------------
// AISDKV6LanguageModel
// ---------------------------------------------------------------------------

// AISDKV6LanguageModel wraps an AI SDK V6 LanguageModel (from ai-kit) to
// convert doGenerate to return a stream format for consistency with Mastra's
// streaming architecture. This is the bridge between ai-kit's typed interface
// and Mastra's internal streaming model.
type AISDKV6LanguageModel struct {
	specificationVersion string
	provider             string
	modelID              string
	supportedURLs        map[string][]*regexp.Regexp
	model                lm.LanguageModel
}

// Compile-time check that AISDKV6LanguageModel implements MastraLanguageModelV3.
var _ MastraLanguageModelV3 = (*AISDKV6LanguageModel)(nil)

// NewAISDKV6LanguageModel creates a new AISDKV6LanguageModel wrapper around
// an ai-kit LanguageModel (V3 specification).
func NewAISDKV6LanguageModel(model lm.LanguageModel) *AISDKV6LanguageModel {
	urls, _ := model.SupportedUrls()
	return &AISDKV6LanguageModel{
		specificationVersion: "v3",
		provider:             model.Provider(),
		modelID:              model.ModelID(),
		supportedURLs:        urls,
		model:                model,
	}
}

// SpecificationVersion implements MastraLanguageModelV3.
func (m *AISDKV6LanguageModel) SpecificationVersion() string { return m.specificationVersion }

// Provider implements MastraLanguageModelV3.
func (m *AISDKV6LanguageModel) Provider() string { return m.provider }

// ModelID implements MastraLanguageModelV3.
func (m *AISDKV6LanguageModel) ModelID() string { return m.modelID }

// SupportedURLs implements MastraLanguageModelV3.
func (m *AISDKV6LanguageModel) SupportedURLs() map[string][]*regexp.Regexp {
	return m.supportedURLs
}

// DoGenerate wraps the underlying ai-kit model's DoGenerate, converting the
// result to a stream format for consistency with Mastra's streaming architecture.
func (m *AISDKV6LanguageModel) DoGenerate(options LanguageModelV3CallOptions) (*LanguageModelV3StreamResult, error) {
	callOpts := toAIKitCallOptions(options)

	result, err := m.model.DoGenerate(callOpts)
	if err != nil {
		return nil, err
	}

	genResult := aiKitGenerateResultToAisdk(result)
	resp := aiKitGenerateResponseToStreamResponse(result.Response)

	var request any
	if result.Request != nil {
		request = result.Request.Body
	}

	return &LanguageModelV3StreamResult{
		Request:  request,
		Response: resp,
		Stream:   aisdk.CreateStreamFromGenerateResult(genResult),
	}, nil
}

// DoStream delegates to the underlying ai-kit model's DoStream, converting
// the ai-kit StreamResult to Mastra's LanguageModelV3StreamResult.
func (m *AISDKV6LanguageModel) DoStream(options LanguageModelV3CallOptions) (*LanguageModelV3StreamResult, error) {
	callOpts := toAIKitCallOptions(options)

	result, err := m.model.DoStream(callOpts)
	if err != nil {
		return nil, err
	}

	var request any
	if result.Request != nil {
		request = result.Request.Body
	}

	return &LanguageModelV3StreamResult{
		Request:  request,
		Response: nil, // Response metadata arrives via stream events
		Stream:   convertStreamPartsToEvents(result.Stream),
	}, nil
}
