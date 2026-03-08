// Ported from: packages/core/src/llm/model/ (complex files not yet fully ported)
//
// This file contains placeholder types/interfaces exported by complex model files
// that are not yet fully ported. Each section references the original TS source file.
//
// Fully ported files (in their own .go files):
//   - gateway_resolver.go        (gateway-resolver.ts)
//   - model_loop.go              (model.loop.ts)
//   - model_loop_types.go        (model.loop.types.ts)
//   - model_method_from_agent.go (model-method-from-agent.ts)
//   - openai_websocket_fetch.go   (openai-websocket-fetch.ts)
//   - registry_generator.go      (registry-generator.ts)
package model

// ===========================================================================
// aisdk/generate-to-stream.ts
// Ported from: packages/core/src/llm/model/aisdk/generate-to-stream.ts
// ===========================================================================

// GenerateResultContent represents a content item from a generate result.
// TODO: implement CreateStreamFromGenerateResult when stream package is ported.
type GenerateResultContent struct {
	Type string `json:"type"`
	// Additional fields vary by content type.
	Extra map[string]any `json:"-"`
}

// ===========================================================================
// aisdk/v5/model.ts
// Ported from: packages/core/src/llm/model/aisdk/v5/model.ts
// ===========================================================================

// AISDKV5LanguageModel wraps an AI SDK V5 (LanguageModelV2) to convert
// doGenerate to return a stream format for consistency with Mastra's
// streaming architecture.
// TODO: implement fully when ai-sdk v5 types are ported.
type AISDKV5LanguageModel struct {
	specificationVersion string
	provider             string
	modelID              string
	model                LanguageModelV2
}

// NewAISDKV5LanguageModel creates a new AISDKV5LanguageModel wrapper.
func NewAISDKV5LanguageModel(model LanguageModelV2) *AISDKV5LanguageModel {
	return &AISDKV5LanguageModel{
		specificationVersion: "v2",
		provider:             model.Provider(),
		modelID:              model.ModelID(),
		model:                model,
	}
}

// SpecificationVersion implements MastraLanguageModel.
func (m *AISDKV5LanguageModel) SpecificationVersion() string { return m.specificationVersion }

// Provider implements MastraLanguageModel.
func (m *AISDKV5LanguageModel) Provider() string { return m.provider }

// ModelID implements MastraLanguageModel.
func (m *AISDKV5LanguageModel) ModelID() string { return m.modelID }

// DoGenerate wraps the underlying model's DoGenerate.
// TODO: convert result to stream format when stream package is ported.
func (m *AISDKV5LanguageModel) DoGenerate(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error) {
	return m.model.DoGenerate(options)
}

// DoStream delegates to the underlying model's DoStream.
func (m *AISDKV5LanguageModel) DoStream(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error) {
	return m.model.DoStream(options)
}

// ===========================================================================
// aisdk/v6/model.ts — FULLY PORTED with ai-kit integration
// Ported from: packages/core/src/llm/model/aisdk/v6/model.ts
//
// The real V6 adapter is in aisdk/v6/model.go and imports
// lm.LanguageModel from brainlink/experiments/ai-kit/provider/languagemodel.
// This stub remains for use within the model package; it wraps the
// Mastra-level LanguageModelV3 interface (not the raw ai-kit interface).
// ===========================================================================

// AISDKV6LanguageModelStub wraps a Mastra-level LanguageModelV3.
// For the full ai-kit-backed implementation, see aisdk/v6.AISDKV6LanguageModel.
type AISDKV6LanguageModelStub struct {
	specificationVersion string
	provider             string
	modelID              string
	model                LanguageModelV3
}

// NewAISDKV6LanguageModelStub creates a stub V6 wrapper using Mastra-level types.
// For the full ai-kit-backed version, use aisdk/v6.NewAISDKV6LanguageModel.
func NewAISDKV6LanguageModelStub(model LanguageModelV3) *AISDKV6LanguageModelStub {
	return &AISDKV6LanguageModelStub{
		specificationVersion: "v3",
		provider:             model.Provider(),
		modelID:              model.ModelID(),
		model:                model,
	}
}

// SpecificationVersion implements MastraLanguageModel.
func (m *AISDKV6LanguageModelStub) SpecificationVersion() string { return m.specificationVersion }

// Provider implements MastraLanguageModel.
func (m *AISDKV6LanguageModelStub) Provider() string { return m.provider }

// ModelID implements MastraLanguageModel.
func (m *AISDKV6LanguageModelStub) ModelID() string { return m.modelID }

// DoGenerate wraps the underlying model's DoGenerate.
func (m *AISDKV6LanguageModelStub) DoGenerate(options LanguageModelV3CallOptions) (LanguageModelV3StreamResult, error) {
	return m.model.DoGenerate(options)
}

// DoStream delegates to the underlying model's DoStream.
func (m *AISDKV6LanguageModelStub) DoStream(options LanguageModelV3CallOptions) (LanguageModelV3StreamResult, error) {
	return m.model.DoStream(options)
}

// ===========================================================================
// gateways/azure.ts, gateways/models-dev.ts, gateways/netlify.ts
// Ported from: packages/core/src/llm/model/gateways/
// ===========================================================================

// AzureOpenAIGatewayConfig holds configuration for the Azure OpenAI gateway.
// TODO: implement AzureOpenAIGateway when needed.
type AzureOpenAIGatewayConfig struct {
	ResourceName string `json:"resourceName"`
	APIVersion   string `json:"apiVersion,omitempty"`
}

// ModelsDevGatewayStub is a placeholder for the ModelsDevGateway implementation.
// TODO: implement fully as a MastraModelGateway.
type ModelsDevGatewayStub struct {
	id   string
	name string
}

// NetlifyGatewayStub is a placeholder for the NetlifyGateway implementation.
// TODO: implement fully as a MastraModelGateway.
type NetlifyGatewayStub struct {
	id   string
	name string
}
