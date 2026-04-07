package engine

import (
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/types"
)

// deserializeProviderConfig converts JSON config + type string into a concrete provider config struct.
// The concrete type is needed because extractProviderCredentials does type switches.
func deserializeProviderConfig(typ string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	switch typ {
	case "openai":
		var c types.OpenAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "anthropic":
		var c types.AnthropicProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "google":
		var c types.GoogleProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "mistral":
		var c types.MistralProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "cohere":
		var c types.CohereProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "groq":
		var c types.GroqProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "perplexity":
		var c types.PerplexityProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "deepseek":
		var c types.DeepSeekProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "fireworks":
		var c types.FireworksProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "togetherai":
		var c types.TogetherAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "xai":
		var c types.XAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "azure":
		var c types.AzureProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "bedrock":
		var c types.BedrockProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "vertex":
		var c types.VertexProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "huggingface":
		var c types.HuggingFaceProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "cerebras":
		var c types.CerebrasProviderConfig
		return c, json.Unmarshal(raw, &c)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", typ)
	}
}

// deserializeStorageConfig converts JSON config + type string into a StorageConfig.
func deserializeStorageConfig(typ string, raw json.RawMessage) (StorageConfig, error) {
	var base struct {
		Path             string `json:"path"`
		ConnectionString string `json:"connectionString"`
		URI              string `json:"uri"`
		DBName           string `json:"dbName"`
		URL              string `json:"url"`
		Token            string `json:"token"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &base); err != nil {
			return StorageConfig{}, fmt.Errorf("invalid storage config: %w", err)
		}
	}
	return StorageConfig{
		Type:             typ,
		Path:             base.Path,
		ConnectionString: base.ConnectionString,
		URI:              base.URI,
		DBName:           base.DBName,
		URL:              base.URL,
		Token:            base.Token,
	}, nil
}
