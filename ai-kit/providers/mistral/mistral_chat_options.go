// Ported from: packages/mistral/src/mistral-chat-options.ts
package mistral

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// MistralChatModelId is the identifier for Mistral chat models.
// Known models include: ministral-3b-latest, ministral-8b-latest,
// mistral-large-latest, mistral-medium-latest, mistral-medium-2508,
// mistral-medium-2505, mistral-small-latest, pixtral-large-latest,
// magistral-small-2507, magistral-medium-2507, magistral-small-2506,
// magistral-medium-2506, pixtral-12b-2409, open-mistral-7b,
// open-mixtral-8x7b, open-mixtral-8x22b, or any custom string.
type MistralChatModelId = string

// MistralLanguageModelOptions contains Mistral-specific provider options.
type MistralLanguageModelOptions struct {
	// SafePrompt controls whether to inject a safety prompt before all conversations.
	// Defaults to false.
	SafePrompt *bool `json:"safePrompt,omitempty"`

	// DocumentImageLimit limits the number of document images.
	DocumentImageLimit *int `json:"documentImageLimit,omitempty"`

	// DocumentPageLimit limits the number of document pages.
	DocumentPageLimit *int `json:"documentPageLimit,omitempty"`

	// StructuredOutputs controls whether to use structured outputs.
	// Defaults to true.
	StructuredOutputs *bool `json:"structuredOutputs,omitempty"`

	// StrictJSONSchema controls whether to use strict JSON schema validation.
	// Defaults to false.
	StrictJSONSchema *bool `json:"strictJsonSchema,omitempty"`

	// ParallelToolCalls controls whether to enable parallel function calling during tool use.
	// When set to false, the model will use at most one tool per response.
	// Defaults to true.
	ParallelToolCalls *bool `json:"parallelToolCalls,omitempty"`
}

// MistralLanguageModelOptionsSchema is the schema for validating MistralLanguageModelOptions.
var MistralLanguageModelOptionsSchema = &providerutils.Schema[MistralLanguageModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[MistralLanguageModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[MistralLanguageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts MistralLanguageModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[MistralLanguageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[MistralLanguageModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
