// Ported from: packages/xai/src/responses/xai-responses-options.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// XaiResponsesModelId is the model ID type for xAI responses models.
type XaiResponsesModelId = string

// XaiLanguageModelResponsesOptions contains options specific to xAI responses API.
type XaiLanguageModelResponsesOptions struct {
	// ReasoningEffort constrains how hard a reasoning model thinks before responding.
	// Possible values are "low", "medium", and "high".
	ReasoningEffort *string `json:"reasoningEffort,omitempty"`
	// Logprobs enables log probabilities in the response.
	Logprobs *bool `json:"logprobs,omitempty"`
	// TopLogprobs is the number of top log probabilities to return (0-8).
	TopLogprobs *int `json:"topLogprobs,omitempty"`
	// Store indicates whether to store the input and response for later retrieval.
	// Default is true.
	Store *bool `json:"store,omitempty"`
	// PreviousResponseId is the ID of the previous response from the model.
	PreviousResponseId *string `json:"previousResponseId,omitempty"`
	// Include specifies additional output data to include in the model response.
	// Example values: "file_search_call.results".
	Include []string `json:"include,omitempty"`
}

// xaiLanguageModelResponsesOptionsSchema is the schema for responses options.
var xaiLanguageModelResponsesOptionsSchema = &providerutils.Schema[XaiLanguageModelResponsesOptions]{}

// XaiResponsesIncludeOptions is the type for the include parameter.
type XaiResponsesIncludeOptions = []string
