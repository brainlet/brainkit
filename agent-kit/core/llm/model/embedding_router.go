// Ported from: packages/core/src/llm/model/embedding-router.ts
package model

import (
	"fmt"
	"os"
	"strings"
)

// ---------------------------------------------------------------------------
// EmbeddingModel stub
// ---------------------------------------------------------------------------

// EmbeddingModelV2 is a stub for the AI SDK v5 EmbeddingModel interface.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type EmbeddingModelV2 interface {
	SpecificationVersion() string
	ModelID() string
	Provider() string
	MaxEmbeddingsPerCall() int
	SupportsParallelCalls() bool
	DoEmbed(args EmbedArgs) (*EmbedResult, error)
}

// EmbedArgs holds the arguments for an embedding call.
type EmbedArgs struct {
	Values []string `json:"values"`
}

// EmbedResult holds the result of an embedding call.
type EmbedResult struct {
	Embeddings [][]float64 `json:"embeddings"`
	Warnings   []any       `json:"warnings,omitempty"`
}

// ---------------------------------------------------------------------------
// EmbeddingModelInfo
// ---------------------------------------------------------------------------

// EmbeddingModelInfo describes a known embedding model.
type EmbeddingModelInfo struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"`
	Dimensions     int    `json:"dimensions"`
	MaxInputTokens int    `json:"maxInputTokens"`
	Description    string `json:"description,omitempty"`
}

// EMBEDDING_MODELS is a curated list of known embedding models.
var EMBEDDING_MODELS = []EmbeddingModelInfo{
	// OpenAI
	{
		ID:             "text-embedding-3-small",
		Provider:       "openai",
		Dimensions:     1536,
		MaxInputTokens: 8191,
		Description:    "OpenAI text-embedding-3-small model",
	},
	{
		ID:             "text-embedding-3-large",
		Provider:       "openai",
		Dimensions:     3072,
		MaxInputTokens: 8191,
		Description:    "OpenAI text-embedding-3-large model",
	},
	{
		ID:             "text-embedding-ada-002",
		Provider:       "openai",
		Dimensions:     1536,
		MaxInputTokens: 8191,
		Description:    "OpenAI text-embedding-ada-002 model",
	},
	// Google
	{
		ID:             "gemini-embedding-001",
		Provider:       "google",
		Dimensions:     768,
		MaxInputTokens: 2048,
		Description:    "Google gemini-embedding-001 model",
	},
}

// EmbeddingModelID represents known embedding model IDs.
type EmbeddingModelID = string

// Well-known embedding model ID constants.
const (
	EmbeddingModelOpenAISmall   EmbeddingModelID = "openai/text-embedding-3-small"
	EmbeddingModelOpenAILarge   EmbeddingModelID = "openai/text-embedding-3-large"
	EmbeddingModelOpenAIAda002  EmbeddingModelID = "openai/text-embedding-ada-002"
	EmbeddingModelGoogleGemini  EmbeddingModelID = "google/gemini-embedding-001"
)

// IsKnownEmbeddingModel checks if a model ID is a known embedding model.
func IsKnownEmbeddingModel(modelID string) bool {
	for _, m := range EMBEDDING_MODELS {
		if m.ID == modelID {
			return true
		}
	}
	return false
}

// GetEmbeddingModelInfo returns information about a known embedding model.
func GetEmbeddingModelInfo(modelID string) *EmbeddingModelInfo {
	for i := range EMBEDDING_MODELS {
		if EMBEDDING_MODELS[i].ID == modelID {
			return &EMBEDDING_MODELS[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// ModelRouterEmbeddingModel
// ---------------------------------------------------------------------------

// ModelRouterEmbeddingModel routes embedding model requests using the
// provider/model string format. It automatically resolves the correct
// AI SDK provider and initializes the embedding model.
//
// TS: export class ModelRouterEmbeddingModel<VALUE extends string = string> implements EmbeddingModelV2<VALUE>
type ModelRouterEmbeddingModel struct {
	specificationVersion  string
	modelID               string
	provider              string
	maxEmbeddingsPerCall  int
	supportsParallelCalls bool

	// providerModel is the underlying AI SDK embedding model.
	// This is initialized in the constructor when the AI SDK provider packages are available.
	// TODO: use EmbeddingModelV2 interface once AI SDK provider packages are ported.
	providerModel EmbeddingModelV2
}

// ModelRouterEmbeddingModelConfig holds the configuration for creating
// a ModelRouterEmbeddingModel. Config can be a string ("provider/model")
// or an OpenAICompatibleConfig.
type ModelRouterEmbeddingModelConfig struct {
	// StringConfig is used when config is a simple string.
	StringConfig string
	// ObjectConfig is used when config is an OpenAICompatibleConfig.
	ObjectConfig *OpenAICompatibleConfig
}

// NewModelRouterEmbeddingModel creates a new ModelRouterEmbeddingModel.
func NewModelRouterEmbeddingModel(config any) (*ModelRouterEmbeddingModel, error) {
	var providerID, modelID, cfgURL, cfgAPIKey string
	var cfgHeaders map[string]string

	switch c := config.(type) {
	case string:
		parts := strings.SplitN(c, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid model string format: %q. Expected format: \"provider/model\"", c)
		}
		providerID = parts[0]
		modelID = parts[1]
	case OpenAICompatibleConfig:
		if c.HasProviderModel() {
			providerID = c.ProviderID
			modelID = c.ModelID
			cfgURL = c.URL
			cfgAPIKey = c.APIKey
			cfgHeaders = c.Headers
		} else if c.HasID() {
			parts := strings.SplitN(c.ID, "/", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid model string format: %q. Expected format: \"provider/model\"", c.ID)
			}
			providerID = parts[0]
			modelID = parts[1]
			cfgURL = c.URL
			cfgAPIKey = c.APIKey
			cfgHeaders = c.Headers
		} else {
			return nil, fmt.Errorf("OpenAICompatibleConfig must have either ID or ProviderID+ModelID")
		}
	default:
		return nil, fmt.Errorf("invalid embedding model config type: %T", config)
	}

	m := &ModelRouterEmbeddingModel{
		specificationVersion:  "v2",
		modelID:               modelID,
		provider:              providerID,
		maxEmbeddingsPerCall:  2048,
		supportsParallelCalls: true,
	}

	// Validate config
	if cfgURL != "" {
		// Custom URL -- skip provider registry validation
		_ = cfgAPIKey
		_ = cfgHeaders
		// TODO: initialize provider model via createOpenAICompatible when ported
	} else {
		registry := GetGatewayRegistry()
		providerConfig, ok := registry.GetProviderConfig(providerID)
		if !ok {
			return nil, fmt.Errorf("unknown provider: %s", providerID)
		}

		// Get API key from config or environment
		apiKey := cfgAPIKey
		if apiKey == "" {
			envVars := providerConfig.APIKeyEnvVarStrings()
			for _, envVar := range envVars {
				apiKey = os.Getenv(envVar)
				if apiKey != "" {
					break
				}
			}
		}

		if apiKey == "" {
			envVarDisplay := strings.Join(providerConfig.APIKeyEnvVarStrings(), " or ")
			return nil, fmt.Errorf("API key not found for provider %s. Set %s", providerID, envVarDisplay)
		}

		// TODO: initialize provider model via createOpenAI/createGoogleGenerativeAI/
		// createOpenAICompatible when AI SDK packages are ported
		_ = apiKey
	}

	return m, nil
}

// SpecificationVersion returns the specification version.
func (m *ModelRouterEmbeddingModel) SpecificationVersion() string { return m.specificationVersion }

// ModelIDValue returns the model ID.
func (m *ModelRouterEmbeddingModel) ModelIDValue() string { return m.modelID }

// ProviderValue returns the provider name.
func (m *ModelRouterEmbeddingModel) ProviderValue() string { return m.provider }

// MaxEmbeddingsPerCall returns the max embeddings per call.
func (m *ModelRouterEmbeddingModel) MaxEmbeddingsPerCall() int { return m.maxEmbeddingsPerCall }

// SupportsParallelCalls returns whether parallel calls are supported.
func (m *ModelRouterEmbeddingModel) SupportsParallelCalls() bool { return m.supportsParallelCalls }

// DoEmbed performs an embedding call.
//
// TS:
//
//	async doEmbed(args): Promise<...> {
//	  const result = await this.providerModel.doEmbed(args);
//	  const warnings = (result as { warnings?: unknown[] }).warnings ?? [];
//	  return { ...result, warnings };
//	}
//
// Delegates to the underlying provider model's DoEmbed method.
// Ensures warnings is always a non-nil slice (AI SDK v6's embedMany spreads
// result.warnings and crashes if it's undefined).
func (m *ModelRouterEmbeddingModel) DoEmbed(args EmbedArgs) (*EmbedResult, error) {
	if m.providerModel == nil {
		return nil, fmt.Errorf("embedding model %s/%s: provider model not initialized (AI SDK provider packages not yet ported to Go)", m.provider, m.modelID)
	}

	result, err := m.providerModel.DoEmbed(args)
	if err != nil {
		return nil, err
	}

	// Ensure warnings is always a non-nil slice
	// TS: const warnings = (result as { warnings?: unknown[] }).warnings ?? [];
	if result != nil && result.Warnings == nil {
		result.Warnings = []any{}
	}

	return result, nil
}
