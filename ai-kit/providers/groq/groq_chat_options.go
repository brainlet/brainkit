// Ported from: packages/groq/src/groq-chat-options.ts
package groq

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GroqChatModelId is a type alias for Groq chat model identifiers.
// Known models include:
//   - gemma2-9b-it
//   - llama-3.1-8b-instant
//   - llama-3.3-70b-versatile
//   - meta-llama/llama-guard-4-12b
//   - openai/gpt-oss-120b
//   - openai/gpt-oss-20b
//   - deepseek-r1-distill-llama-70b
//   - meta-llama/llama-4-maverick-17b-128e-instruct
//   - meta-llama/llama-4-scout-17b-16e-instruct
//   - meta-llama/llama-prompt-guard-2-22m
//   - meta-llama/llama-prompt-guard-2-86m
//   - moonshotai/kimi-k2-instruct-0905
//   - qwen/qwen3-32b
//   - llama-guard-3-8b
//   - llama3-70b-8192
//   - llama3-8b-8192
//   - mixtral-8x7b-32768
//   - qwen-qwq-32b
//   - qwen-2.5-32b
//   - deepseek-r1-distill-qwen-32b
//
// Any string is accepted.
type GroqChatModelId = string

// GroqLanguageModelOptions contains Groq-specific language model options.
type GroqLanguageModelOptions struct {
	// ReasoningFormat specifies the reasoning format.
	// Valid values: "parsed", "raw", "hidden".
	ReasoningFormat *string `json:"reasoningFormat,omitempty"`

	// ReasoningEffort specifies the reasoning effort level for model inference.
	// Valid values: "none", "default", "low", "medium", "high".
	ReasoningEffort *string `json:"reasoningEffort,omitempty"`

	// ParallelToolCalls controls whether to enable parallel function calling during tool use.
	// Default is true.
	ParallelToolCalls *bool `json:"parallelToolCalls,omitempty"`

	// User is a unique identifier representing the end-user.
	User *string `json:"user,omitempty"`

	// StructuredOutputs controls whether to use structured outputs.
	// Default is true.
	StructuredOutputs *bool `json:"structuredOutputs,omitempty"`

	// StrictJSONSchema controls whether to use strict JSON schema validation.
	// Default is true.
	StrictJSONSchema *bool `json:"strictJsonSchema,omitempty"`

	// ServiceTier for the request.
	// Valid values: "on_demand", "flex", "auto".
	// Default is "on_demand".
	ServiceTier *string `json:"serviceTier,omitempty"`
}

// GroqLanguageModelOptionsSchema is the schema for validating GroqLanguageModelOptions.
var GroqLanguageModelOptionsSchema = &providerutils.Schema[GroqLanguageModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GroqLanguageModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[GroqLanguageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts GroqLanguageModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[GroqLanguageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[GroqLanguageModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
