// Ported from: packages/openai/src/chat/openai-chat-options.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAIChatModelId is the model ID type for OpenAI chat models.
// In Go we use a plain string since TypeScript's branded string types
// don't have a direct equivalent.
type OpenAIChatModelId = string

// OpenAILanguageModelChatOptions holds the provider-specific options for OpenAI chat models.
type OpenAILanguageModelChatOptions struct {
	// LogitBias modifies the likelihood of specified tokens appearing in the completion.
	LogitBias map[string]float64 `json:"logitBias,omitempty"`

	// Logprobs controls log probability output. Can be a bool or a number (top n).
	// In Go, we use *interface{} to support both.
	Logprobs interface{} `json:"logprobs,omitempty"`

	// ParallelToolCalls controls whether to enable parallel function calling during tool use.
	ParallelToolCalls *bool `json:"parallelToolCalls,omitempty"`

	// User is a unique identifier representing your end-user.
	User *string `json:"user,omitempty"`

	// ReasoningEffort is the reasoning effort for reasoning models.
	ReasoningEffort *string `json:"reasoningEffort,omitempty"`

	// MaxCompletionTokens is the maximum number of completion tokens to generate.
	MaxCompletionTokens *int `json:"maxCompletionTokens,omitempty"`

	// Store controls whether to enable persistence in responses API.
	Store *bool `json:"store,omitempty"`

	// Metadata to associate with the request.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Prediction parameters for prediction mode.
	Prediction map[string]any `json:"prediction,omitempty"`

	// ServiceTier for the request.
	ServiceTier *string `json:"serviceTier,omitempty"`

	// StrictJSONSchema controls whether to use strict JSON schema validation.
	StrictJSONSchema *bool `json:"strictJsonSchema,omitempty"`

	// TextVerbosity controls the verbosity of the model's responses.
	TextVerbosity *string `json:"textVerbosity,omitempty"`

	// PromptCacheKey is a cache key for prompt caching.
	PromptCacheKey *string `json:"promptCacheKey,omitempty"`

	// PromptCacheRetention is the retention policy for the prompt cache.
	PromptCacheRetention *string `json:"promptCacheRetention,omitempty"`

	// SafetyIdentifier is a stable identifier used to help detect users violating policies.
	SafetyIdentifier *string `json:"safetyIdentifier,omitempty"`

	// SystemMessageMode overrides the system message mode for this model.
	SystemMessageMode *string `json:"systemMessageMode,omitempty"`

	// ForceReasoning forces treating this model as a reasoning model.
	ForceReasoning *bool `json:"forceReasoning,omitempty"`
}

// openaiLanguageModelChatOptionsSchema is the schema for parsing OpenAI chat options.
var openaiLanguageModelChatOptionsSchema = &providerutils.Schema[OpenAILanguageModelChatOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAILanguageModelChatOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAILanguageModelChatOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts OpenAILanguageModelChatOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[OpenAILanguageModelChatOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAILanguageModelChatOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
