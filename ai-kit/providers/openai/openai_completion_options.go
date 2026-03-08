// Ported from: packages/openai/src/completion/openai-completion-options.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAICompletionModelId is the model ID type for OpenAI completion models.
type OpenAICompletionModelId = string

// OpenAILanguageModelCompletionOptions holds the provider-specific options for OpenAI completion models.
type OpenAILanguageModelCompletionOptions struct {
	// Echo controls whether to echo back the prompt in addition to the completion.
	Echo *bool `json:"echo,omitempty"`

	// LogitBias modifies the likelihood of specified tokens appearing in the completion.
	LogitBias map[string]float64 `json:"logitBias,omitempty"`

	// Suffix is the suffix that comes after a completion of inserted text.
	Suffix *string `json:"suffix,omitempty"`

	// User is a unique identifier representing your end-user.
	User *string `json:"user,omitempty"`

	// Logprobs controls log probability output. Can be a bool or a number.
	Logprobs interface{} `json:"logprobs,omitempty"`
}

// openaiLanguageModelCompletionOptionsSchema is the schema for parsing OpenAI completion options.
var openaiLanguageModelCompletionOptionsSchema = &providerutils.Schema[OpenAILanguageModelCompletionOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAILanguageModelCompletionOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAILanguageModelCompletionOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts OpenAILanguageModelCompletionOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[OpenAILanguageModelCompletionOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAILanguageModelCompletionOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
