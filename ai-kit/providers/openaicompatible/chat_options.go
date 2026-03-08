// Ported from: packages/openai-compatible/src/chat/openai-compatible-chat-options.ts
package openaicompatible

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ChatModelID is the identifier for an OpenAI-compatible chat model.
type ChatModelID = string

// ChatOptions contains provider-specific options for OpenAI-compatible chat models.
type ChatOptions struct {
	// User is a unique identifier representing the end-user, which can help
	// the provider monitor and detect abuse.
	User *string `json:"user,omitempty"`

	// ReasoningEffort controls reasoning effort for reasoning models.
	// Defaults to "medium".
	ReasoningEffort *string `json:"reasoningEffort,omitempty"`

	// TextVerbosity controls the verbosity of the generated text.
	// Defaults to "medium".
	TextVerbosity *string `json:"textVerbosity,omitempty"`

	// StrictJSONSchema controls whether to use strict JSON schema validation.
	// When true, the model uses constrained decoding to guarantee schema compliance.
	// Only used when the provider supports structured outputs and a schema is provided.
	// Defaults to true.
	StrictJSONSchema *bool `json:"strictJsonSchema,omitempty"`
}

// ChatOptionsSchema is the providerutils.Schema used to validate and parse
// ChatOptions from provider options maps.
var ChatOptionsSchema = &providerutils.Schema[ChatOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[ChatOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[ChatOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts ChatOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[ChatOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[ChatOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
